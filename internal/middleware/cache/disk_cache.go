package cache

import (
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func init() {
	// Register types with gob for serialization
	gob.Register(&http.Response{})
	gob.Register(&CachedResponse{})
}

// DiskCache implements the Cache interface using the filesystem for storage
type DiskCache struct {
	basePath    string
	mutex       sync.RWMutex
	indexMutex  sync.RWMutex
	index       map[string]string // Maps cache keys to filenames
	maxDiskSize int64             // Maximum size in bytes
	currentSize int64             // Current size in bytes
}

// NewDiskCache creates a new disk-based cache at the specified directory
func NewDiskCache(basePath string, maxSizeMB int) (*DiskCache, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &DiskCache{
		basePath:    basePath,
		index:       make(map[string]string),
		maxDiskSize: int64(maxSizeMB) * 1024 * 1024, // Convert MB to bytes
	}

	// Initialize the cache by loading the index
	if err := cache.loadIndex(); err != nil {
		return nil, err
	}

	return cache, nil
}

// Get retrieves a cached response if available
func (c *DiskCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
	c.indexMutex.RLock()
	filename, exists := c.index[key]
	c.indexMutex.RUnlock()

	if !exists {
		return nil, false
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	filePath := filepath.Join(c.basePath, filename)
	file, err := os.Open(filePath)
	if err != nil {
		// File doesn't exist or can't be opened, remove from index
		c.indexMutex.Lock()
		delete(c.index, key)
		c.indexMutex.Unlock()
		return nil, false
	}
	defer file.Close()

	var cachedResp CachedResponse
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&cachedResp); err != nil {
		// Corrupted file, remove it
		c.indexMutex.Lock()
		delete(c.index, key)
		c.indexMutex.Unlock()
		os.Remove(filePath)
		return nil, false
	}

	// Check if expired
	if time.Now().After(cachedResp.ExpiresAt) {
		// Remove expired entry
		c.indexMutex.Lock()
		delete(c.index, key)
		c.indexMutex.Unlock()
		os.Remove(filePath)
		return nil, false
	}

	// Update last accessed time
	cachedResp.LastAccessed = time.Now()
	return &cachedResp, true
}

// Set stores a response in the cache
func (c *DiskCache) Set(ctx context.Context, key string, response *CachedResponse) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Generate filename from key
	filename := c.keyToFilename(key)
	filePath := filepath.Join(c.basePath, filename)

	// Ensure we have enough space
	if err := c.ensureSpace(response); err != nil {
		return err
	}

	// Create a temporary file for atomic write
	tempFile, err := os.CreateTemp(c.basePath, "temp_cache_")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // Remove temp file in case of error
	}()

	// Encode and write the cached response
	encoder := gob.NewEncoder(tempFile)
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	// Close the file before renaming
	tempFile.Close()

	// Move temp file to final location (atomic replace)
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to save cache file: %w", err)
	}

	// Update index
	c.indexMutex.Lock()
	c.index[key] = filename
	c.indexMutex.Unlock()

	// Update cache size
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		c.currentSize += fileInfo.Size()
	}

	return nil
}

// Delete removes a cached response
func (c *DiskCache) Delete(ctx context.Context, key string) error {
	c.indexMutex.Lock()
	filename, exists := c.index[key]
	if exists {
		delete(c.index, key)
	}
	c.indexMutex.Unlock()

	if !exists {
		return nil
	}

	filePath := filepath.Join(c.basePath, filename)

	// Get file size before removing
	info, err := os.Stat(filePath)
	if err == nil {
		c.currentSize -= info.Size()
	}

	return os.Remove(filePath)
}

// Clear removes all cached responses
func (c *DiskCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Reset the index
	c.indexMutex.Lock()
	c.index = make(map[string]string)
	c.indexMutex.Unlock()

	// Remove all cache files
	entries, err := os.ReadDir(c.basePath)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(c.basePath, entry.Name())
		if err := os.Remove(filePath); err != nil {
			// Log error but continue
			fmt.Printf("Error removing cache file %s: %v\n", filePath, err)
		}
	}

	c.currentSize = 0

	return nil
}

// Close performs any cleanup needed
func (c *DiskCache) Close() error {
	// Nothing special needed for disk cache cleanup
	return nil
}

// keyToFilename converts a cache key to a valid filename
func (c *DiskCache) keyToFilename(key string) string {
	// Use a hash of the key as the filename to avoid invalid characters
	hasher := hashKey(key)
	return hasher + ".cache"
}

// ensureSpace ensures there is enough space for a new cache entry
func (c *DiskCache) ensureSpace(response *CachedResponse) error {
	if c.maxDiskSize <= 0 {
		return nil // No size limit
	}

	// Estimate size of new entry
	estimatedSize := int64(len(response.Body))
	for k, v := range response.Response.Header {
		estimatedSize += int64(len(k))
		for _, val := range v {
			estimatedSize += int64(len(val))
		}
	}
	estimatedSize += 1024 // Additional overhead for gob encoding

	// Check if we're near the limit
	if c.currentSize+estimatedSize > c.maxDiskSize {
		// Need to free up space
		return c.evictOldEntries(estimatedSize)
	}

	return nil
}

// evictOldEntries removes the oldest entries to free up disk space
func (c *DiskCache) evictOldEntries(neededSpace int64) error {
	type entryInfo struct {
		key      string
		filename string
		lastUsed time.Time
		size     int64
	}

	// Collect information about all entries
	entries := make([]entryInfo, 0, len(c.index))

	c.indexMutex.RLock()
	for key, filename := range c.index {
		filePath := filepath.Join(c.basePath, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// Try to get last accessed time from file
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}

		var cachedResp CachedResponse
		decoder := gob.NewDecoder(file)
		if err := decoder.Decode(&cachedResp); err != nil {
			file.Close()
			continue
		}
		file.Close()

		entries = append(entries, entryInfo{
			key:      key,
			filename: filename,
			lastUsed: cachedResp.LastAccessed,
			size:     info.Size(),
		})
	}
	c.indexMutex.RUnlock()

	// Sort by last accessed time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lastUsed.Before(entries[j].lastUsed)
	})

	// Remove entries until we have enough space
	spaceFreed := int64(0)
	for _, entry := range entries {
		if c.currentSize-spaceFreed+neededSpace <= c.maxDiskSize {
			break
		}

		// Delete the entry
		filePath := filepath.Join(c.basePath, entry.filename)
		if err := os.Remove(filePath); err == nil {
			spaceFreed += entry.size
			c.indexMutex.Lock()
			delete(c.index, entry.key)
			c.indexMutex.Unlock()
		}
	}

	c.currentSize -= spaceFreed

	if c.currentSize+neededSpace > c.maxDiskSize {
		return errors.New("not enough space available in cache after eviction")
	}

	return nil
}

// loadIndex rebuilds the index from the cache directory
func (c *DiskCache) loadIndex() error {
	entries, err := os.ReadDir(c.basePath)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	c.indexMutex.Lock()
	defer c.indexMutex.Unlock()

	c.index = make(map[string]string)
	c.currentSize = 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cache") {
			continue
		}

		filePath := filepath.Join(c.basePath, entry.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// Try to load the entry to get its key
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}

		var cachedResp CachedResponse
		decoder := gob.NewDecoder(file)
		if err := decoder.Decode(&cachedResp); err != nil {
			file.Close()
			continue
		}
		file.Close()

		// Check if expired
		if time.Now().After(cachedResp.ExpiresAt) {
			os.Remove(filePath)
			continue
		}

		// Generate key from URL
		url, err := url.Parse(cachedResp.RequestURL)
		if err != nil {
			continue
		}

		key := url.String()
		c.index[key] = entry.Name()
		c.currentSize += info.Size()
	}

	return nil
}

// StartCleanupTask starts a goroutine that periodically removes expired entries
func (c *DiskCache) StartCleanupTask(interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Hour
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.cleanup()
		}
	}()
}

// cleanup removes expired entries from the cache
func (c *DiskCache) cleanup() {
	c.indexMutex.Lock()
	keysToCheck := make([]string, 0, len(c.index))
	filesToCheck := make([]string, 0, len(c.index))
	for key, filename := range c.index {
		keysToCheck = append(keysToCheck, key)
		filesToCheck = append(filesToCheck, filename)
	}
	c.indexMutex.Unlock()

	now := time.Now()
	removedSize := int64(0)

	for i, key := range keysToCheck {
		filename := filesToCheck[i]
		filePath := filepath.Join(c.basePath, filename)

		// Read the file to check expiration
		file, err := os.Open(filePath)
		if err != nil {
			c.indexMutex.Lock()
			delete(c.index, key)
			c.indexMutex.Unlock()
			continue
		}

		var cachedResp CachedResponse
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&cachedResp)
		file.Close()

		if err != nil || now.After(cachedResp.ExpiresAt) {
			// Get file size before removing
			info, err := os.Stat(filePath)
			if err == nil {
				removedSize += info.Size()
			}

			// Remove expired entry
			os.Remove(filePath)
			c.indexMutex.Lock()
			delete(c.index, key)
			c.indexMutex.Unlock()
		}
	}

	// Update current size
	if removedSize > 0 {
		c.mutex.Lock()
		c.currentSize -= removedSize
		if c.currentSize < 0 {
			c.currentSize = 0
		}
		c.mutex.Unlock()
	}
}

// hashKey creates a hash of the provided string
func hashKey(key string) string {
	hasher := md5.New()
	io.WriteString(hasher, key)
	return hex.EncodeToString(hasher.Sum(nil))
}

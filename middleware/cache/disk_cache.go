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
	gob.Register(&http.Response{})
	gob.Register(&CachedResponse{})
}

// DiskCache implements the Cache interface using the filesystem for storage
type DiskCache struct {
	// basePath is the directory where cache files are stored
	basePath string
	// mutex protects file operations
	mutex sync.RWMutex
	// indexMutex protects the index map
	indexMutex sync.RWMutex
	// index maps cache keys to filenames
	index map[string]string
	// maxDiskSize is the maximum size in bytes
	maxDiskSize int64
	// currentSize is the current size in bytes
	currentSize int64
}

func NewDiskCache(basePath string, maxSizeMB int) (*DiskCache, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &DiskCache{
		basePath:    basePath,
		index:       make(map[string]string),
		maxDiskSize: int64(maxSizeMB) * 1024 * 1024,
	}

	if err := cache.loadIndex(); err != nil {
		return nil, err
	}

	return cache, nil
}

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
		c.indexMutex.Lock()
		delete(c.index, key)
		c.indexMutex.Unlock()
		return nil, false
	}
	defer file.Close()

	var cachedResp CachedResponse
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&cachedResp); err != nil {
		c.indexMutex.Lock()
		delete(c.index, key)
		c.indexMutex.Unlock()
		os.Remove(filePath)
		return nil, false
	}

	if time.Now().After(cachedResp.ExpiresAt) {
		c.indexMutex.Lock()
		delete(c.index, key)
		c.indexMutex.Unlock()
		os.Remove(filePath)
		return nil, false
	}

	cachedResp.LastAccessed = time.Now()
	return &cachedResp, true
}

func (c *DiskCache) Set(ctx context.Context, key string, response *CachedResponse) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	filename := c.keyToFilename(key)
	filePath := filepath.Join(c.basePath, filename)

	if err := c.ensureSpace(response); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(c.basePath, "temp_cache_")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	encoder := gob.NewEncoder(tempFile)
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	tempFile.Close()

	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to save cache file: %w", err)
	}

	c.indexMutex.Lock()
	c.index[key] = filename
	c.indexMutex.Unlock()

	fileInfo, err := os.Stat(filePath)
	if err == nil {
		c.currentSize += fileInfo.Size()
	}

	return nil
}

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

	info, err := os.Stat(filePath)
	if err == nil {
		c.currentSize -= info.Size()
	}

	return os.Remove(filePath)
}

func (c *DiskCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.indexMutex.Lock()
	c.index = make(map[string]string)
	c.indexMutex.Unlock()

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
	return nil
}

func (c *DiskCache) keyToFilename(key string) string {
	hasher := hashKey(key)
	return hasher + ".cache"
}

func (c *DiskCache) ensureSpace(response *CachedResponse) error {
	if c.maxDiskSize <= 0 {
		return nil
	}

	estimatedSize := int64(len(response.Body))
	for k, v := range response.Response.Header {
		estimatedSize += int64(len(k))
		for _, val := range v {
			estimatedSize += int64(len(val))
		}
	}
	estimatedSize += 1024

	if c.currentSize+estimatedSize > c.maxDiskSize {
		return c.evictOldEntries(estimatedSize)
	}

	return nil
}

func (c *DiskCache) evictOldEntries(neededSpace int64) error {
	type entryInfo struct {
		key      string
		filename string
		lastUsed time.Time
		size     int64
	}

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

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lastUsed.Before(entries[j].lastUsed)
	})

	spaceFreed := int64(0)
	for _, entry := range entries {
		if c.currentSize-spaceFreed+neededSpace <= c.maxDiskSize {
			break
		}

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

		if time.Now().After(cachedResp.ExpiresAt) {
			os.Remove(filePath)
			continue
		}

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
			info, err := os.Stat(filePath)
			if err == nil {
				removedSize += info.Size()
			}

			os.Remove(filePath)
			c.indexMutex.Lock()
			delete(c.index, key)
			c.indexMutex.Unlock()
		}
	}

	if removedSize > 0 {
		c.mutex.Lock()
		c.currentSize -= removedSize
		if c.currentSize < 0 {
			c.currentSize = 0
		}
		c.mutex.Unlock()
	}
}

func hashKey(key string) string {
	hasher := md5.New()
	io.WriteString(hasher, key)
	return hex.EncodeToString(hasher.Sum(nil))
}

package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// MemoryCache implements the Cache interface using in-memory storage
// with an LRU eviction policy.
type MemoryCache struct {
	data     map[string]*list.Element
	lruList  *list.List
	capacity int
	mutex    sync.RWMutex
}

// cacheEntry represents an entry in the LRU list
type cacheEntry struct {
	key      string
	response *CachedResponse
}

// NewMemoryCache creates a new in-memory cache with the specified capacity
func NewMemoryCache(capacity int) *MemoryCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	return &MemoryCache{
		data:     make(map[string]*list.Element),
		lruList:  list.New(),
		capacity: capacity,
	}
}

// Get retrieves a cached response if available
func (c *MemoryCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
	c.mutex.RLock()
	element, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		return nil, false
	}

	// Get the response from the list element
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Move to front of LRU list (most recently used)
	c.lruList.MoveToFront(element)
	entry := element.Value.(*cacheEntry)

	// Check if the response has expired
	if time.Now().After(entry.response.ExpiresAt) {
		// Remove expired entry
		c.lruList.Remove(element)
		delete(c.data, key)
		return nil, false
	}

	// Update last accessed time
	entry.response.LastAccessed = time.Now()
	return entry.response, true
}

// Set stores a response in the cache
func (c *MemoryCache) Set(ctx context.Context, key string, response *CachedResponse) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if the key already exists
	if element, exists := c.data[key]; exists {
		// Update existing entry and move to front
		c.lruList.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		entry.response = response
		return nil
	}

	// Check if we're at capacity
	if c.lruList.Len() >= c.capacity {
		// Remove least recently used item
		oldest := c.lruList.Back()
		if oldest != nil {
			entry := oldest.Value.(*cacheEntry)
			delete(c.data, entry.key)
			c.lruList.Remove(oldest)
		}
	}

	// Add new entry to front of list
	entry := &cacheEntry{
		key:      key,
		response: response,
	}
	element := c.lruList.PushFront(entry)
	c.data[key] = element

	return nil
}

// Delete removes a cached response
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.data[key]; exists {
		c.lruList.Remove(element)
		delete(c.data, key)
	}

	return nil
}

// Clear removes all cached responses
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*list.Element)
	c.lruList.Init()

	return nil
}

// Close performs any cleanup needed
func (c *MemoryCache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = nil
	c.lruList = nil

	return nil
}

// Size returns the current number of items in the cache
func (c *MemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.lruList.Len()
}

// StartCleanupTask starts a goroutine that periodically removes expired entries
func (c *MemoryCache) StartCleanupTask(interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Minute
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
func (c *MemoryCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Iterate through all entries and remove expired ones
	for element := c.lruList.Front(); element != nil; {
		entry := element.Value.(*cacheEntry)
		nextElement := element.Next() // Store next before potentially removing

		if now.After(entry.response.ExpiresAt) {
			delete(c.data, entry.key)
			c.lruList.Remove(element)
		}

		element = nextElement
	}
}

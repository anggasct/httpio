package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// MemoryCache implements the Cache interface using in-memory storage with an LRU eviction policy
type MemoryCache struct {
	// data maps cache keys to list elements
	data map[string]*list.Element
	// lruList maintains the LRU order
	lruList *list.List
	// capacity is the maximum number of entries
	capacity int
	// mutex protects concurrent access
	mutex sync.RWMutex
}

// cacheEntry represents an entry in the LRU list
type cacheEntry struct {
	// key is the cache key
	key string
	// response is the cached response
	response *CachedResponse
}

func NewMemoryCache(capacity int) *MemoryCache {
	if capacity <= 0 {
		capacity = 100
	}

	return &MemoryCache{
		data:     make(map[string]*list.Element),
		lruList:  list.New(),
		capacity: capacity,
	}
}

func (c *MemoryCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
	c.mutex.RLock()
	element, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		return nil, false
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.lruList.MoveToFront(element)
	entry := element.Value.(*cacheEntry)

	if time.Now().After(entry.response.ExpiresAt) {
		c.lruList.Remove(element)
		delete(c.data, key)
		return nil, false
	}

	entry.response.LastAccessed = time.Now()
	return entry.response, true
}

func (c *MemoryCache) Set(ctx context.Context, key string, response *CachedResponse) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.data[key]; exists {
		c.lruList.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		entry.response = response
		return nil
	}

	if c.lruList.Len() >= c.capacity {
		oldest := c.lruList.Back()
		if oldest != nil {
			entry := oldest.Value.(*cacheEntry)
			delete(c.data, entry.key)
			c.lruList.Remove(oldest)
		}
	}

	entry := &cacheEntry{
		key:      key,
		response: response,
	}
	element := c.lruList.PushFront(entry)
	c.data[key] = element

	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.data[key]; exists {
		c.lruList.Remove(element)
		delete(c.data, key)
	}

	return nil
}

func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*list.Element)
	c.lruList.Init()

	return nil
}

func (c *MemoryCache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = nil
	c.lruList = nil

	return nil
}

func (c *MemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.lruList.Len()
}

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

func (c *MemoryCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	for element := c.lruList.Front(); element != nil; {
		entry := element.Value.(*cacheEntry)
		nextElement := element.Next()

		if now.After(entry.response.ExpiresAt) {
			delete(c.data, entry.key)
			c.lruList.Remove(element)
		}

		element = nextElement
	}
}

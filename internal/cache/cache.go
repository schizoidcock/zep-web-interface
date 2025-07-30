package cache

import (
	"sync"
	"time"
)

type CacheItem struct {
	Data      interface{}
	ExpiresAt time.Time
}

type Cache struct {
	items map[string]CacheItem
	mutex sync.RWMutex
}

func NewCache() *Cache {
	cache := &Cache{
		items: make(map[string]CacheItem),
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	return cache
}

func (c *Cache) Set(key string, data interface{}, duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.items[key] = CacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(duration),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	
	return item.Data, true
}

func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.items, key)
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpiresAt) {
				delete(c.items, key)
			}
		}
		c.mutex.Unlock()
	}
}
package cache

import (
	"errors"
	"maps"
	"sync"
)

var (
	ErrNotFound = errors.New("key not found")
)

type Cache struct {
	storage map[string]any
	mu      sync.RWMutex
}

func Init() *Cache {
	return &Cache{
		storage: make(map[string]any),
	}
}

func (c *Cache) Set(key string, value any) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.storage[key] = value

}

func (c *Cache) Get(key string) (any, error) {

	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.storage[key]
	if !ok {
		return nil, ErrNotFound
	}

	return value, nil

}

func (c *Cache) Snapshot() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]any, len(c.storage))
	maps.Copy(result, c.storage)
	return result
}

package core_storage

import (
	"errors"
	"sync"
)

var (
	ErrNotFound = errors.New("Key not found")
)

// temporary storage
type Cache struct {
	storage map[string]string
	mu      sync.RWMutex
}

func Init() *Cache {
	return &Cache{
		storage: make(map[string]string),
	}
}

func (c *Cache) Set(key, value string) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.storage[key] = value

}

func (c *Cache) Get(key string) (string, error) {

	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.storage[key]
	if !ok {
		return "", ErrNotFound
	}

	return value, nil

}

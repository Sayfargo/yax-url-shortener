// Модуль cache предоставляет потокобезопасное хранилище типа «key-value»
// в оперативной памяти (in-memory cache) с поддержкой создания снапшотов данных.
package cache

import (
	"errors"
	"maps"
	"sync"
)

var (
	// ErrNotFound возвращается методом [Cache.Get], если запрашиваемый ключ
	// отсутствует в хранилище.
	ErrNotFound = errors.New("key not found")
)

// Cache представляет собой потокобезопасное хранилище данных в оперативной памяти.
// Для синхронизации конкурентного доступа используется [sync.RWMutex].
type Cache struct {
	storage map[string]any
	mu      sync.RWMutex
}

// Init создает, инициализирует и возвращает новый пустой экземпляр [Cache],
// готовый к использованию.
func Init() *Cache {
	return &Cache{
		storage: make(map[string]any),
	}
}

// Set сохраняет или обновляет значение [value] по указанному ключу [key].
// Операция является потокобезопасной и блокирует пишущих конкурентов.
func (c *Cache) Set(key string, value any) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.storage[key] = value

}

// Get возвращает сохраненное значение по ключу [key].
// Если ключ не найден, метод возвращает nil и ошибку [ErrNotFound].
//
// Операция безопасна для параллельного чтения множеством горутин.
func (c *Cache) Get(key string) (any, error) {

	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.storage[key]
	if !ok {
		return nil, ErrNotFound
	}

	return value, nil

}

// Snapshot создает и возвращает изолированную, независимую копию (слепок)
// всех текущих данных, находящихся в кэше.
//
// Изменения в возвращенной карте не влияют на состояние самого [Cache].
// Метод безопасно делать параллельно с другими операциями чтения, однако
// на время копирования он блокирует пишущие операции (вызовы Set).
func (c *Cache) Snapshot() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]any, len(c.storage))
	maps.Copy(result, c.storage)
	return result
}

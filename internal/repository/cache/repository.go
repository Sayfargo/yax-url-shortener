package repository_cache

import (
	"context"
	"errors"
	"fmt"

	core_storage_cache "github.com/Sayfargo/yax-url-shortener/internal/core/storage/cache"
)

// temporary repository
type CacheRepository struct {
	cache *core_storage_cache.Cache
}

func New(cache *core_storage_cache.Cache) *CacheRepository {
	return &CacheRepository{
		cache: cache,
	}
}

var (
	ErrNotExists     = errors.New("row not found in db")
	ErrAlreadyExists = errors.New("row already exists")
)

func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	url, err := r.cache.Get(shortCode)
	if err != nil {
		if errors.Is(err, core_storage_cache.ErrNotFound) {
			return "", ErrNotExists
		}
		return "", fmt.Errorf("Cache storage Get err: %w", err)
	}

	return url, nil
}

func (r *CacheRepository) Create(ctx context.Context, url, shortCode string) error {

	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.cache.Set(shortCode, url)

	return nil

}

package repository

import (
	"context"
	"errors"
	"fmt"

	core_storage "github.com/Sayfargo/yax-url-shortener/internal/core/storage"
)

// temporary repository
type URLRepository struct {
	cache *core_storage.Cache
}

func New(cache *core_storage.Cache) *URLRepository {
	return &URLRepository{
		cache: cache,
	}
}

var (
	ErrNotExists = errors.New("row not found in db")
)

func (r *URLRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	url, err := r.cache.Get(shortCode)
	if err != nil {
		if errors.Is(err, core_storage.ErrNotFound) {
			return "", ErrNotExists
		}
		return "", fmt.Errorf("Cache storage Get err: %w", err)
	}

	return url, nil
}

func (r *URLRepository) Create(ctx context.Context, url, shortCode string) error {

	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.cache.Set(shortCode, url)

	return nil

}

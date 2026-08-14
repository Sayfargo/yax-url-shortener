package repository_cache

import (
	"context"
	"errors"
	"fmt"

	core_storage_cache "github.com/Sayfargo/yax-url-shortener/internal/core/storage/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type CacheRepository struct {
	cache *core_storage_cache.Cache
}

func New(cache *core_storage_cache.Cache) *CacheRepository {
	return &CacheRepository{
		cache: cache,
	}
}

var (
	ErrNotExists      = errors.New("row not found in db")
	ErrAlreadyExists  = errors.New("row already exists")
	ErrUnexpectedType = errors.New("unexpected data type in cache")
)

func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	value, err := r.cache.Get(shortCode)
	if err != nil {
		if errors.Is(err, core_storage_cache.ErrNotFound) {
			return "", ErrNotExists
		}
		return "", fmt.Errorf("cache storage get err: %w", err)
	}

	shortenedUrl, ok := value.(model.ShortenedUrl)
	if !ok {
		return "", ErrUnexpectedType
	}

	return shortenedUrl.OriginalUrl, nil
}

func (r *CacheRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := r.cache.Get(shortenedUrl.ShortCode)
	// Если новые ошибки появятся
	switch {
	case err == nil:
		return ErrAlreadyExists
	case !errors.Is(err, core_storage_cache.ErrNotFound):
		return err
	}

	r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)

	return nil

}

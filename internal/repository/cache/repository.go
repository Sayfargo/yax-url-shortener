package repository_cache

import (
	"context"
	"errors"
	"fmt"
	"sync"

	core_storage_cache "github.com/Sayfargo/yax-url-shortener/internal/core/storage/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	repository_errors "github.com/Sayfargo/yax-url-shortener/internal/repository/errors"
)

type CacheRepository struct {
	cache *core_storage_cache.Cache
	mu    sync.Mutex
}

func New(cache *core_storage_cache.Cache) *CacheRepository {
	return &CacheRepository{
		cache: cache,
	}
}

func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	value, err := r.cache.Get(shortCode)
	if err != nil {
		if errors.Is(err, core_storage_cache.ErrNotFound) {
			return "", repository_errors.ErrNotExists
		}
		return "", fmt.Errorf("cache storage get err: %w", err)
	}

	shortenedUrl, ok := value.(model.ShortenedUrl)
	if !ok {
		return "", repository_errors.ErrUnexpectedType
	}

	return shortenedUrl.OriginalUrl, nil
}

func (r *CacheRepository) CreateBatch(ctx context.Context, shortenedUrls []model.ShortenedUrl) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for _, shortenedUrl := range shortenedUrls {
		_, err := r.cache.Get(shortenedUrl.ShortCode)

		switch {
		case err == nil:
			return repository_errors.ErrAlreadyExists
		case !errors.Is(err, core_storage_cache.ErrNotFound):
			return err
		}
	}

	for _, shortenedUrl := range shortenedUrls {
		r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)
	}

	return nil
}

func (r *CacheRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := r.cache.Get(shortenedUrl.ShortCode)
	// Если новые ошибки появятся
	switch {
	case err == nil:
		return repository_errors.ErrAlreadyExists
	case !errors.Is(err, core_storage_cache.ErrNotFound):
		return err
	}

	r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)

	return nil

}

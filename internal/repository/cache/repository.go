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
	cache       *core_storage_cache.Cache
	originalMap map[string]string
	mu          sync.RWMutex
}

func New(cache *core_storage_cache.Cache) *CacheRepository {
	return &CacheRepository{
		cache:       cache,
		originalMap: make(map[string]string),
	}
}

func (r *CacheRepository) GetByOriginalURL(ctx context.Context, url string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	shortCode, ok := r.originalMap[url]
	if !ok {
		return "", repository_errors.ErrNotExists
	}

	return shortCode, nil
}

func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

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
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, shortenedUrl := range shortenedUrls {

		if _, ok := r.originalMap[shortenedUrl.OriginalUrl]; ok {
			return &repository_errors.OriginalUrlConflictError{
				URL: shortenedUrl.OriginalUrl,
			}
		}

		_, err := r.cache.Get(shortenedUrl.ShortCode)

		switch {
		case err == nil:
			return repository_errors.ErrConflictShortCode
		case !errors.Is(err, core_storage_cache.ErrNotFound):
			return err
		}
	}

	for _, shortenedUrl := range shortenedUrls {
		r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)
		r.originalMap[shortenedUrl.OriginalUrl] = shortenedUrl.ShortCode
	}

	return nil
}

func (r *CacheRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.originalMap[shortenedUrl.OriginalUrl]; ok {
		return &repository_errors.OriginalUrlConflictError{
			URL: shortenedUrl.OriginalUrl,
		}
	}

	_, err := r.cache.Get(shortenedUrl.ShortCode)

	switch {
	case err == nil:
		return repository_errors.ErrConflictShortCode
	case !errors.Is(err, core_storage_cache.ErrNotFound):
		return err
	}

	r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)
	r.originalMap[shortenedUrl.OriginalUrl] = shortenedUrl.ShortCode

	return nil

}

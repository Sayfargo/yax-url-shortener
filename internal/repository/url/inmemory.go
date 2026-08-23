package url

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cache "github.com/Sayfargo/yax-url-shortener/internal/core/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type CacheRepository struct {
	c           *cache.Cache
	originalMap map[string]string
	mu          sync.RWMutex
}

func NewInMemoryRepo(cache *cache.Cache) *CacheRepository {
	return &CacheRepository{
		c:           cache,
		originalMap: make(map[string]string),
	}
}

func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	value, err := r.c.Get(shortCode)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
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

func (r *CacheRepository) CreateBatch(ctx context.Context, shortenedUrls []model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	seenShort := make(map[string]struct{})

	for i, shortenedUrl := range shortenedUrls {

		if _, ok := seenShort[shortenedUrl.ShortCode]; ok {
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		}

		_, err := r.c.Get(shortenedUrl.ShortCode)

		switch {
		case err == nil:
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		case !errors.Is(err, cache.ErrNotFound):
			return err
		}

		seenShort[shortenedUrl.ShortCode] = struct{}{}
	}

	for _, shortenedUrl := range shortenedUrls {
		r.c.Set(shortenedUrl.ShortCode, shortenedUrl)
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

	if v, ok := r.originalMap[shortenedUrl.OriginalUrl]; ok {
		return &OriginalUrlConflictError{
			ShortCode: v,
		}
	}

	_, err := r.c.Get(shortenedUrl.ShortCode)

	switch {
	case err == nil:
		return ErrConflictShortCode
	case !errors.Is(err, cache.ErrNotFound):
		return err
	}

	r.c.Set(shortenedUrl.ShortCode, shortenedUrl)
	r.originalMap[shortenedUrl.OriginalUrl] = shortenedUrl.ShortCode

	return nil

}

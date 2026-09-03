package url

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cache "github.com/Sayfargo/yax-url-shortener/internal/core/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
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

func (r *CacheRepository) GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error) {

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	urls := make([]model.ShortenedURL, 0, 32)

	for _, val := range r.c.Snapshot() {
		if rec, ok := val.(model.ShortenedURL); ok {
			if rec.UserID == uid {
				urls = append(urls, rec)
			}
		} else {
			return nil, ErrUnexpectedType
		}
	}

	return urls, nil

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

	ShortenedURL, ok := value.(model.ShortenedURL)
	if !ok {
		return "", ErrUnexpectedType
	}

	return ShortenedURL.OriginalURL, nil
}

func (r *CacheRepository) CreateBatch(ctx context.Context, ShortenedURLs []model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	seenShort := make(map[string]struct{})

	for i, ShortenedURL := range ShortenedURLs {

		if _, ok := seenShort[ShortenedURL.ShortCode]; ok {
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		}

		_, err := r.c.Get(ShortenedURL.ShortCode)

		switch {
		case err == nil:
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		case !errors.Is(err, cache.ErrNotFound):
			return err
		}

		seenShort[ShortenedURL.ShortCode] = struct{}{}
	}

	for _, ShortenedURL := range ShortenedURLs {
		r.c.Set(ShortenedURL.ShortCode, ShortenedURL)
		r.originalMap[ShortenedURL.OriginalURL] = ShortenedURL.ShortCode
	}

	return nil
}

func (r *CacheRepository) Create(ctx context.Context, ShortenedURL model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.originalMap[ShortenedURL.OriginalURL]; ok {
		return &OriginalURLConflictError{
			ShortCode: v,
		}
	}

	_, err := r.c.Get(ShortenedURL.ShortCode)

	switch {
	case err == nil:
		return ErrConflictShortCode
	case !errors.Is(err, cache.ErrNotFound):
		return err
	}

	r.c.Set(ShortenedURL.ShortCode, ShortenedURL)
	r.originalMap[ShortenedURL.OriginalURL] = ShortenedURL.ShortCode

	return nil

}

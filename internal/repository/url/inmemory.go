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

func (r *CacheRepository) SoftDeleteURLs(ctx context.Context, uid uuid.UUID, shortCodes ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, shortCode := range shortCodes {
		value, err := r.c.Get(shortCode)
		if err != nil {
			if errors.Is(err, cache.ErrNotFound) {
				return ErrNotExists
			}
			return fmt.Errorf("cache storage get err: %w", err)
		}
		if shortenedURL, ok := value.(model.ShortenedURL); ok {
			shortenedURL.IsDeleted = true
			r.c.Set(shortCode, shortenedURL)
		}
	}
	return nil

}

func (r *CacheRepository) GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error) {

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	urls := make([]model.ShortenedURL, 0, 32)

	for _, val := range r.c.Snapshot() {
		if rec, ok := val.(model.ShortenedURL); ok {
			if rec.UserID == uid && !rec.IsDeleted {
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

	shortenedURL, ok := value.(model.ShortenedURL)
	if !ok {
		return "", ErrUnexpectedType
	}

	if shortenedURL.IsDeleted {
		return "", ErrRowGone
	}

	return shortenedURL.OriginalURL, nil
}

func (r *CacheRepository) CreateBatch(ctx context.Context, shortenedURLs []model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	seenShort := make(map[string]struct{})

	for i, u := range shortenedURLs {

		if _, ok := seenShort[u.ShortCode]; ok {
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		}

		_, err := r.c.Get(u.ShortCode)

		switch {
		case err == nil:
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		case !errors.Is(err, cache.ErrNotFound):
			return err
		}

		seenShort[u.ShortCode] = struct{}{}
	}

	for _, u := range shortenedURLs {
		r.c.Set(u.ShortCode, u)
		r.originalMap[u.OriginalURL] = u.ShortCode
	}

	return nil
}

func (r *CacheRepository) Create(ctx context.Context, shortenedURL model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.originalMap[shortenedURL.OriginalURL]; ok {
		return &OriginalURLConflictError{
			ShortCode: v,
		}
	}

	_, err := r.c.Get(shortenedURL.ShortCode)

	switch {
	case err == nil:
		return ErrConflictShortCode
	case !errors.Is(err, cache.ErrNotFound):
		return err
	}

	r.c.Set(shortenedURL.ShortCode, shortenedURL)
	r.originalMap[shortenedURL.OriginalURL] = shortenedURL.ShortCode

	return nil

}

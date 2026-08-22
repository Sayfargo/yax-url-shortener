package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cache "github.com/Sayfargo/yax-url-shortener/internal/core/storage/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	repoerrors "github.com/Sayfargo/yax-url-shortener/internal/repository/errors"
)

type CacheRepository struct {
	cache       *cache.Cache
	originalMap map[string]string
	mu          sync.RWMutex
}

func New(cache *cache.Cache) *CacheRepository {
	return &CacheRepository{
		cache:       cache,
		originalMap: make(map[string]string),
	}
}

func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	value, err := r.cache.Get(shortCode)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return "", repoerrors.ErrNotExists
		}
		return "", fmt.Errorf("cache storage get err: %w", err)
	}

	shortenedUrl, ok := value.(model.ShortenedUrl)
	if !ok {
		return "", repoerrors.ErrUnexpectedType
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
			return &repoerrors.BatchConflictError{
				Index: i,
				Err:   repoerrors.ErrConflictShortCode,
			}
		}

		_, err := r.cache.Get(shortenedUrl.ShortCode)

		switch {
		case err == nil:
			return &repoerrors.BatchConflictError{
				Index: i,
				Err:   repoerrors.ErrConflictShortCode,
			}
		case !errors.Is(err, cache.ErrNotFound):
			return err
		}

		seenShort[shortenedUrl.ShortCode] = struct{}{}
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

	if v, ok := r.originalMap[shortenedUrl.OriginalUrl]; ok {
		return &repoerrors.OriginalUrlConflictError{
			ShortCode: v,
		}
	}

	_, err := r.cache.Get(shortenedUrl.ShortCode)

	switch {
	case err == nil:
		return repoerrors.ErrConflictShortCode
	case !errors.Is(err, cache.ErrNotFound):
		return err
	}

	r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)
	r.originalMap[shortenedUrl.OriginalUrl] = shortenedUrl.ShortCode

	return nil

}

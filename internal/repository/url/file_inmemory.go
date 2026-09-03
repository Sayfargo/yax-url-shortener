package url

import (
	"context"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type FileCacheRepository struct {
	*CacheRepository

	fs *filestorage.FileStorage
}

func NewFileInMemoryRepository(cacheRepository *CacheRepository, fileStorage *filestorage.FileStorage) (*FileCacheRepository, error) {

	repo := &FileCacheRepository{
		CacheRepository: cacheRepository,
		fs:              fileStorage,
	}

	if err := repo.restoreCache(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *FileCacheRepository) CreateBatch(ctx context.Context, ShortenedURLs []model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := r.CacheRepository.CreateBatch(ctx, ShortenedURLs); err != nil {
		return err
	}

	return r.fs.WriteURLs(ShortenedURLs)
}

func (r *FileCacheRepository) Create(ctx context.Context, ShortenedURL model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := r.CacheRepository.Create(ctx, ShortenedURL); err != nil {
		return err
	}

	return r.fs.WriteURL(ShortenedURL)
}

func (r *FileCacheRepository) restoreCache() error {

	ShortenedURLs, err := r.fs.ReadURLs()
	if err != nil {
		return fmt.Errorf("failed to read urls: %w", err)
	}

	for _, ShortenedURL := range ShortenedURLs {
		r.c.Set(ShortenedURL.ShortCode, ShortenedURL)
	}

	return nil
}

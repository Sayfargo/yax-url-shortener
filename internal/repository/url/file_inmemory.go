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

func (r *FileCacheRepository) CreateBatch(ctx context.Context, shortenedUrls []model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := r.CacheRepository.CreateBatch(ctx, shortenedUrls); err != nil {
		return err
	}

	return r.fs.WriteURLs(shortenedUrls)
}

func (r *FileCacheRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := r.CacheRepository.Create(ctx, shortenedUrl); err != nil {
		return err
	}

	return r.fs.WriteURL(shortenedUrl)
}

func (r *FileCacheRepository) restoreCache() error {

	shortenedUrls, err := r.fs.ReadURLs()
	if err != nil {
		return fmt.Errorf("failed to read urls: %w", err)
	}

	for _, shortenedUrl := range shortenedUrls {
		r.c.Set(shortenedUrl.ShortCode, shortenedUrl)
	}

	return nil
}

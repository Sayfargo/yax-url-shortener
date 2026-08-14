package repository_cache

import (
	"context"
	"fmt"

	core_storage_file "github.com/Sayfargo/yax-url-shortener/internal/core/storage/file"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type FileCacheRepository struct {
	*CacheRepository

	fs *core_storage_file.FileStorage
}

func Wrap(cacheRepository *CacheRepository, fileStorage *core_storage_file.FileStorage) *FileCacheRepository {
	return &FileCacheRepository{
		CacheRepository: cacheRepository,
		fs:              fileStorage,
	}
}

func (r *FileCacheRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {
	if err := r.CacheRepository.Create(ctx, shortenedUrl); err != nil {
		return err
	}

	return r.fs.WriteURL(shortenedUrl)
}

func (r *FileCacheRepository) Restore() error {

	shortenedUrls, err := r.fs.ReadURLs()
	if err != nil {
		return fmt.Errorf("failed to read urls: %w", err)
	}

	for _, shortenedUrl := range shortenedUrls {
		r.cache.Set(shortenedUrl.ShortCode, shortenedUrl)
	}

	return nil
}

package url

import (
	"context"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
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

func (r *FileCacheRepository) SoftDeleteURLs(ctx context.Context, uid uuid.UUID, shortCodes ...string) error {

	if err := r.CacheRepository.SoftDeleteURLs(
		ctx,
		uid,
		shortCodes...,
	); err != nil {
		return err
	}

	result := make([]model.ShortenedURL, 0)

	for _, val := range r.c.Snapshot() {
		shortenedURL, ok := val.(model.ShortenedURL)
		if !ok {
			return ErrUnexpectedType
		}

		result = append(result, shortenedURL)
	}

	return r.fs.RewriteURLs(result)

}

func (r *FileCacheRepository) CreateBatch(ctx context.Context, shortenedURLs []model.ShortenedURL) error {

	if err := r.CacheRepository.CreateBatch(ctx, shortenedURLs); err != nil {
		return err
	}

	return r.fs.WriteURLs(shortenedURLs)
}

func (r *FileCacheRepository) Create(ctx context.Context, shortenedURL model.ShortenedURL) error {

	if err := r.CacheRepository.Create(ctx, shortenedURL); err != nil {
		return err
	}

	return r.fs.WriteURL(shortenedURL)
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

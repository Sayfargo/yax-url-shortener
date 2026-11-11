package url

import (
	"context"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
)

// FileCacheRepository предоставляет гибридное хранилище, комбинирующее быстрое
// чтение из оперативной памяти [CacheRepository] с персистентной записью на диск
// через [filestorage.FileStorage] (паттерн Write-Through Cache).
type FileCacheRepository struct {
	*CacheRepository

	fs *filestorage.FileStorage
}

// NewFileInMemoryRepository создает новый экземпляр [FileCacheRepository], связывает
// кэш с файловым хранилищем и автоматически восстанавливает состояние памяти
// из файла при старте.
//
// Возвращает ошибку, если не удалось прочитать существующие записи с диска.
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

// SoftDeleteURLs выполняет мягкое удаление ссылок в оперативной памяти, после чего
// делает полный снимок (snapshot) кэша и перезаписывает файл на диске актуальным состоянием.
//
// Возвращает [ErrUnexpectedType], если типы данных в кэше повреждены, или ошибку
// записи на диск.
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

// CreateBatch атомарно добавляет пачку сокращенных ссылок в оперативную память
// и, в случае успеха, дублирует их в конец лог-файла на диске.
func (r *FileCacheRepository) CreateBatch(ctx context.Context, shortenedURLs []model.ShortenedURL) error {

	if err := r.CacheRepository.CreateBatch(ctx, shortenedURLs); err != nil {
		return err
	}

	return r.fs.WriteURLs(shortenedURLs)
}

// Create сохраняет одну сокращенную ссылку в оперативную память и, в случае успеха,
// дописывает ее в конец лог-файла на диске.
func (r *FileCacheRepository) Create(ctx context.Context, shortenedURL model.ShortenedURL) error {

	if err := r.CacheRepository.Create(ctx, shortenedURL); err != nil {
		return err
	}

	return r.fs.WriteURL(shortenedURL)
}

func (r *FileCacheRepository) restoreCache() error {

	shortenedURLs, err := r.fs.ReadURLs()
	if err != nil {
		return fmt.Errorf("failed to read urls: %w", err)
	}

	for _, u := range shortenedURLs {
		r.c.Set(u.ShortCode, u)
	}

	return nil
}

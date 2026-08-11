package repository_file

import (
	"context"
	"fmt"

	core_storage_file "github.com/Sayfargo/yax-url-shortener/internal/core/storage/file"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type FileRepository struct {
	storage *core_storage_file.FileStorage
}

func New(storage *core_storage_file.FileStorage) *FileRepository {
	return &FileRepository{
		storage: storage,
	}
}

func (fr *FileRepository) Save(ctx context.Context, shortenedUrl model.ShortenedUrl) error {

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := fr.storage.WriteURL(shortenedUrl); err != nil {
		return fmt.Errorf("failed to write rec to file storage %w", err)
	}

	return nil
}

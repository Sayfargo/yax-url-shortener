package core_storage_file

import (
	"encoding/json"
	"os"
	"sync"

	config_storage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type FileStorage struct {
	file    *os.File
	encoder *json.Encoder

	// Для потокобезопасной запси json.Encode
	mu sync.Mutex
}

func New(cfg *config_storage.Config) (*FileStorage, error) {
	file, err := os.OpenFile(cfg.FilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		file: file,

		encoder: json.NewEncoder(file),
	}, nil
}

func (fs *FileStorage) WriteURL(shortenedUrl model.ShortenedUrl) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.encoder.Encode(shortenedUrl)
}

func (fs *FileStorage) Close() error {
	return fs.file.Close()
}

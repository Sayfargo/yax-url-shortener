package core_storage_file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	config_storage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type FileStorage struct {
	file    *os.File
	encoder *json.Encoder

	mu sync.Mutex
}

func Init(cfg *config_storage.Config) (*FileStorage, error) {
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

func (fs *FileStorage) ReadURLs() ([]model.ShortenedUrl, error) {

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	scanner := bufio.NewScanner(fs.file)
	shortenedUrls := make([]model.ShortenedUrl, 0)

	for scanner.Scan() {
		var shortenedUrl model.ShortenedUrl

		if err := json.Unmarshal(scanner.Bytes(), &shortenedUrl); err != nil {
			return nil, fmt.Errorf("unmarshal json: %w", err)
		}

		shortenedUrls = append(shortenedUrls, shortenedUrl)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return shortenedUrls, nil

}

func (fs *FileStorage) Close() error {
	return fs.file.Close()
}

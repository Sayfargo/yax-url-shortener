package filestorage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

type FileStorage struct {
	file    *os.File
	encoder *json.Encoder

	mu sync.Mutex
}

func Init(cfg *Config) (*FileStorage, error) {
	file, err := os.OpenFile(cfg.FilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		file: file,

		encoder: json.NewEncoder(file),
	}, nil
}

func (fs *FileStorage) WriteURL(ShortenedURL model.ShortenedURL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.encoder.Encode(ShortenedURL)
}

func (fs *FileStorage) WriteURLs(ShortenedURLs []model.ShortenedURL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, ShortenedURL := range ShortenedURLs {
		if err := fs.encoder.Encode(ShortenedURL); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileStorage) ReadURLs() ([]model.ShortenedURL, error) {

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	scanner := bufio.NewScanner(fs.file)
	ShortenedURLs := make([]model.ShortenedURL, 0)

	for scanner.Scan() {
		var ShortenedURL model.ShortenedURL

		if err := json.Unmarshal(scanner.Bytes(), &ShortenedURL); err != nil {
			return nil, fmt.Errorf("unmarshal json: %w", err)
		}

		ShortenedURLs = append(ShortenedURLs, ShortenedURL)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return ShortenedURLs, nil

}

func (fs *FileStorage) Close() error {
	return fs.file.Close()
}

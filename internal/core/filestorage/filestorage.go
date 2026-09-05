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

func (fs *FileStorage) RewriteURLs(shortenedURLs []model.ShortenedURL) error {
	if err := fs.file.Truncate(0); err != nil {
		return fmt.Errorf("truncate file storage: %w", err)
	}

	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("io seek: %w", err)
	}

	encoder := json.NewEncoder(fs.file)

	for _, u := range shortenedURLs {
		if err := encoder.Encode(u); err != nil {
			return fmt.Errorf("encode shortened url: %w", err)
		}
	}

	return nil
}

func (fs *FileStorage) WriteURL(shortenedURL model.ShortenedURL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.encoder.Encode(shortenedURL)
}

func (fs *FileStorage) WriteURLs(shortenedURLs []model.ShortenedURL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, u := range shortenedURLs {
		if err := fs.encoder.Encode(u); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileStorage) ReadURLs() ([]model.ShortenedURL, error) {

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("io seek: %w", err)
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

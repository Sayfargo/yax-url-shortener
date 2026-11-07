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

// FileStorage представляет собой потокобезопасное хранилище данных на диске.
// Запись производится построчно, где каждая строка является валидным JSON-объектом.
// Конкурентный доступ синхронизируется с помощью [sync.Mutex].
type FileStorage struct {
	file    *os.File
	encoder *json.Encoder

	mu sync.Mutex
}

// Init открывает или создает файл по пути, указанному в [Config], в режиме
// чтения/записи и добавления данных (append-only), после чего возвращает [FileStorage].
//
// Файл инициализируется с правами доступа 0666. Возвращает ошибку, если не удалось
// получить доступ к файлу.
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

// RewriteURLs полностью очищает текущее содержимое файла и перезаписывает его
// переданным слайсом [model.ShortenedURL].
//
// Метод сбрасывает указатель файла в начало, выполняет очистку через Truncate
// и последовательно кодирует элементы. Возвращает обернутую ошибку при сбое ввода-вывода.
func (fs *FileStorage) RewriteURLs(shortenedURLs []model.ShortenedURL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

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

// WriteURL сериализует одну структуру [model.ShortenedURL] в JSON и дописывает
// ее в конец файла с символом новой строки.
func (fs *FileStorage) WriteURL(shortenedURL model.ShortenedURL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.encoder.Encode(shortenedURL)
}

// WriteURLs последовательно сериализует и дописывает в конец файла
// массив структур [model.ShortenedURL].
//
// Возвращает ошибку при первом же сбое записи.
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

// ReadURLs считывает все сохраненные записи [model.ShortenedURL] из файла.
//
// Перед началом чтения метод сбрасывает позицию указателя файла в начало (SeekStart).
// Чтение выполняется построчно с использованием [bufio.Scanner].
// Возвращает слайс прочитанных URL или ошибку декодирования/чтения.
func (fs *FileStorage) ReadURLs() ([]model.ShortenedURL, error) {

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("io seek: %w", err)
	}

	scanner := bufio.NewScanner(fs.file)
	shortenedURLs := make([]model.ShortenedURL, 0)

	for scanner.Scan() {
		var shortenedURL model.ShortenedURL

		if err := json.Unmarshal(scanner.Bytes(), &shortenedURL); err != nil {
			return nil, fmt.Errorf("unmarshal json: %w", err)
		}

		shortenedURLs = append(shortenedURLs, shortenedURL)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return shortenedURLs, nil

}

// Close закрывает нижележащий дескриптор файла [os.File].
// После вызова этого метода любые операции чтения или записи будут завершаться ошибкой.
func (fs *FileStorage) Close() error {
	return fs.file.Close()
}

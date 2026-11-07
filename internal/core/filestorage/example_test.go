package filestorage

import (
	"fmt"
	"os"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
)

// Example демонстрирует базовый жизненный цикл FileStorage:
// инициализацию хранилища через временный файл, одиночную и массовую
// запись данных, а также чтение всей истории обратно в память.
func Example() {
	// Создаем временный файл для демонстрации работы хранилища
	tmpFile, err := os.CreateTemp("", "audit_storage_example_*.jsonl")
	if err != nil {
		fmt.Printf("failed to create temp file: %v\n", err)
		return
	}
	// Гарантируем очистку диска после выполнения примера
	defer os.Remove(tmpFile.Name())
	_ = tmpFile.Close()

	// 1. Инициализируем конфигурацию и файловое хранилище
	cfg := &Config{
		FilePath: tmpFile.Name(),
	}

	storage, err := Init(cfg)
	if err != nil {
		fmt.Printf("failed to init storage: %v\n", err)
		return
	}
	defer storage.Close()

	// Подготавливаем тестовые данные
	url1 := model.ShortenedURL{ShortCode: "googl", OriginalURL: "https://google.com"}
	url2 := model.ShortenedURL{ShortCode: "yandx", OriginalURL: "https://yandex.ru"}
	url3 := model.ShortenedURL{ShortCode: "githb", OriginalURL: "https://github.com"}

	// 2. Записываем одну запись в конец файла
	if err := storage.WriteURL(url1); err != nil {
		fmt.Printf("failed to write single url: %v\n", err)
		return
	}

	// 3. Записываем пачку записей массовым методом
	if err := storage.WriteURLs([]model.ShortenedURL{url2, url3}); err != nil {
		fmt.Printf("failed to write multiple urls: %v\n", err)
		return
	}

	// 4. Считываем все сохраненные данные с диска
	// Метод автоматически сбросит курсор файла в начало перед чтением
	records, err := storage.ReadURLs()
	if err != nil {
		fmt.Printf("failed to read urls: %v\n", err)
		return
	}

	// Выводим результат для проверки утилитой go test
	for _, rec := range records {
		fmt.Printf("Loaded: ShortCode=%s, Target=%s\n", rec.ShortCode, rec.OriginalURL)
	}

	// Output:
	// Loaded: ShortCode=googl, Target=https://google.com
	// Loaded: ShortCode=yandx, Target=https://yandex.ru
	// Loaded: ShortCode=githb, Target=https://github.com
}

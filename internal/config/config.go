// Моудль config объединяет конфигурационные структуры всех подсистем приложения
// (сервера, базы данных, файлового хранилища и аудита) и предоставляет
// единую точку инициализации параметров.
package config

import (
	"flag"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/audit"
	"github.com/Sayfargo/yax-url-shortener/internal/core/db/postgres"
	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/core/httpserver"
)

// StorageType определяет стратегию (тип) используемого хранилища данных.
type StorageType string

const (
	// StorageTypeDB указывает на использование реляционной базы данных PostgreSQL.
	StorageTypeDB StorageType = "database"
	// StorageTypeFile указывает на сохранение данных в локальном файле.
	StorageTypeFile StorageType = "file"
	// StorageTypeMemory указывает на работу в оперативной памяти (in-memory) без сохранения состояния.
	StorageTypeMemory StorageType = "memory"
)

// Config представляет глобальную конфигурацию всего приложения,
// агрегирующую настройки отдельных независимых модулей.
type Config struct {
	// Server содержит параметры конфигурации HTTP-сервера.
	Server *httpserver.Config
	// DB содержит параметры подключения к PostgreSQL.
	DB *postgres.Config
	// FileStorage содержит настройки путей для файлового сохранения.
	FileStorage *filestorage.Config
	// Audit содержит настройки адресов и файлов для системы логирования аудита.
	Audit *audit.Config
}

type parser interface {
	ParseEnv() error
}

// Load выполняет сборку конфигурации всего приложения на основе переданных аргументов командной строки [args].
//
// Процесс инициализации параметров происходит в два этапа:
// 1. Из [args] извлекаются и регистрируются флаги для всех подсистем.
// 2. Поля структур дополняются или перетираются значениями из соответствующих переменных окружения.
//
// Таким образом, переменные окружения имеют более высокий приоритет, чем флаги.
// Возвращает ошибку, если разбор флагов или переменных окружения завершился со сбоем.
func Load(args []string) (*Config, error) {

	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	serverConfig := httpserver.RegisterFlags(fs)
	dbConfig := postgres.RegisterFlags(fs)
	fileStorageConfig := filestorage.RegisterFlags(fs)
	auditConfig := audit.RegisterFlags(fs)

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	for _, config := range []parser{serverConfig, dbConfig, fileStorageConfig, auditConfig} {
		if err := config.ParseEnv(); err != nil {
			return nil, fmt.Errorf("failed to parse env: %w", err)
		}
	}

	return &Config{
		Server:      serverConfig,
		DB:          dbConfig,
		FileStorage: fileStorageConfig,
		Audit:       auditConfig,
	}, nil

}

// StorageType динамически вычисляет и возвращает целевой [StorageType]
// на основе заполненных параметров конфигурации.
func (c *Config) StorageType() StorageType {

	if c.DB != nil && c.DB.DBDsn != "" {
		return StorageTypeDB
	}

	if c.FileStorage != nil && c.FileStorage.FilePath != "" {
		return StorageTypeFile
	}

	return StorageTypeMemory

}

package config

import (
	"flag"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/audit"
	"github.com/Sayfargo/yax-url-shortener/internal/core/db/postgres"
	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/core/httpserver"
)

type StorageType string

const (
	StorageTypeDB     StorageType = "database"
	StorageTypeFile   StorageType = "file"
	StorageTypeMemory StorageType = "memory"
)

type Config struct {
	Server      *httpserver.Config
	DB          *postgres.Config
	FileStorage *filestorage.Config
	Audit       *audit.Config
}

type parser interface {
	ParseEnv() error
}

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

func (c *Config) StorageType() StorageType {

	if c.DB != nil && c.DB.DBDsn != "" {
		return StorageTypeDB
	}

	if c.FileStorage != nil && c.FileStorage.FilePath != "" {
		return StorageTypeFile
	}

	return StorageTypeMemory

}

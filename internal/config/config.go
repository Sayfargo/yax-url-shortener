package config

import (
	"flag"
	"fmt"

	postgres "github.com/Sayfargo/yax-url-shortener/internal/config/db/postgres"
	server "github.com/Sayfargo/yax-url-shortener/internal/config/server"
	filestorage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
)

type StorageType string

const (
	StorageTypeDB     StorageType = "database"
	StorageTypeFile   StorageType = "file"
	StorageTypeMemory StorageType = "memory"
)

type Config struct {
	Server      *server.Config
	DB          *postgres.Config
	FileStorage *filestorage.Config
}

func Load(args []string) (*Config, error) {

	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	serverConfig := server.RegisterFlags(fs)
	dbConfig := postgres.RegisterFlags(fs)
	fileStorageConfig := filestorage.RegisterFlags(fs)

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if err := serverConfig.ParseEnv(); err != nil {
		return nil, fmt.Errorf("server config: %w", err)
	}

	if err := dbConfig.ParseEnv(); err != nil {
		return nil, fmt.Errorf("db config: %w", err)
	}

	if err := fileStorageConfig.ParseEnv(); err != nil {
		return nil, fmt.Errorf("file storage config: %w", err)
	}

	return &Config{
		Server:      serverConfig,
		DB:          dbConfig,
		FileStorage: fileStorageConfig,
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

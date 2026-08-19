package config

import (
	"flag"
	"fmt"

	config_db_postgres "github.com/Sayfargo/yax-url-shortener/internal/config/db/postgres"
	config_server "github.com/Sayfargo/yax-url-shortener/internal/config/server"
	config_storage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
)

type Config struct {
	Server      *config_server.Config
	DB          *config_db_postgres.Config
	FileStorage *config_storage.Config
}

func Load(args []string) (*Config, error) {

	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	serverConfig := config_server.RegisterFlags(fs)
	dbConfig := config_db_postgres.RegisterFlags(fs)
	fileStorageConfig := config_storage.RegisterFlags(fs)

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

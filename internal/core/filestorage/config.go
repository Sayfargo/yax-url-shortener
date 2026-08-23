package filestorage

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	FilePath string `env:"FILE_STORAGE_PATH"`
}

func RegisterFlags(fs *flag.FlagSet) *Config {

	fileStorageConfig := new(Config)

	fs.StringVar(
		&fileStorageConfig.FilePath,
		"f",
		"",
		"URL file storage path",
	)

	return fileStorageConfig

}

func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}

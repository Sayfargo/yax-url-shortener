package config_storage

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	FilePath string `env:"FILE_STORAGE_PATH"`
}

func RegisterFlags(fs *flag.FlagSet) *Config {

	fileStorageConfig := new(Config)

	fs.StringVar(
		&fileStorageConfig.FilePath,
		"f",
		"./urls.json",
		"URL file storage path",
	)

	return fileStorageConfig

}

func (c *Config) ParseEnv() {
	if err := env.Parse(c); err != nil {
		panic(err) // TODO: Обработать ошибку более корректно + логирование
	}
}

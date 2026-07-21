package config

import (
	"flag"

	config_server "github.com/Sayfargo/yax-url-shortener/internal/config/server"
)

type Config struct {
	Server *config_server.Config
}

func Load() *Config {

	serverConfig := config_server.Load()

	flag.Parse()

	return &Config{
		Server: serverConfig,
	}

}

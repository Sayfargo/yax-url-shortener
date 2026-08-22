package server

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Addr    string `env:"SERVER_ADDRESS"`
	BaseURL string `env:"BASE_URL"`
}

func RegisterFlags(fs *flag.FlagSet) *Config {

	serverCfg := new(Config)

	fs.StringVar(
		&serverCfg.Addr,
		"a",
		"localhost:8080",
		"Server address",
	)

	fs.StringVar(
		&serverCfg.BaseURL,
		"b",
		"http://localhost:8080",
		"Base URL",
	)

	return serverCfg

}

func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("failed to parse env: %w", err)
	}
	return nil
}

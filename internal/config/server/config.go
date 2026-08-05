package config_server

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	Addr    string `env:"SERVER_ADDRESS"`
	BaseURL string `env:"BASIC_URL"`
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

func (c *Config) ParseEnv() {
	if err := env.Parse(c); err != nil {
		panic(err) // TODO: Обработать ошибку более корректно + логирование
	}
}

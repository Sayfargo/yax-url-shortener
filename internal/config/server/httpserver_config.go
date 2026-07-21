package config_server

import "flag"

type Config struct {
	Addr    string
	BaseURL string
}

func Load() *Config {

	serverCfg := new(Config)

	flag.StringVar(
		&serverCfg.Addr,
		"a",
		"localhost:8080",
		"HTTP server addr",
	)

	flag.StringVar(
		&serverCfg.BaseURL,
		"b",
		"http://localhost:8080",
		"Base URL",
	)

	return serverCfg

}

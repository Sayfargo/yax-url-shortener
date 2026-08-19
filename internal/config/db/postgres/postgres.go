package config_db_postgres

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	DBDsn string `env:"DATABASE_DSN"`
}

func RegisterFlags(fs *flag.FlagSet) *Config {

	postgresConfig := new(Config)

	fs.StringVar(
		&postgresConfig.DBDsn,
		"d",
		"",
		"database url connection",
	)

	return postgresConfig

}

func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}

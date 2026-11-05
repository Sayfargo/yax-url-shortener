package audit

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	File string `env:"AUDIT_FILE"`
	URL  string `env:"AUDIT_URL"`
}

func RegisterFlags(fs *flag.FlagSet) *Config {

	auditConfig := new(Config)

	fs.StringVar(
		&auditConfig.File,
		"audit-file",
		"",
		"Audit file path",
	)
	fs.StringVar(
		&auditConfig.URL,
		"audit-url",
		"",
		"Audit URL",
	)
	return auditConfig
}

func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}

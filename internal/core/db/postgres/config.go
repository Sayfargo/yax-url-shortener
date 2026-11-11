package postgres

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config содержит параметры конфигурации для подключения к PostgreSQL.
type Config struct {
	// DBDsn — строка подключения к базе данных (Data Source Name / Connection String).
	// Настраивается через флаг -d или переменную окружения DATABASE_DSN.
	DBDsn string `env:"DATABASE_DSN"`
}

// RegisterFlags связывает поле DBDsn структуры [Config] с флагом командной строки "-d"
// в переданном наборе [flag.FlagSet] и возвращает указатель на созданную конфигурацию.
//
// Метод только регистрирует флаг, его фактический разбор (fs.Parse) должен производиться
// на более высоком уровне приложения (например, в пакете config).
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

// ParseEnv дополняет или перезаписывает текущее значение строки подключения [Config.DBDsn]
// данными из переменной окружения DATABASE_DSN, используя библиотеку [env.Parse].
//
// Возвращает обернутую ошибку, если разбор переменных окружения завершился неудачно.
func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}

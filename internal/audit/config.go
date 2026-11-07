package audit

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config содержит параметры конфигурации для системы аудита.
// Значения могут быть загружены как из флагов командной строки,
// так и из переменных окружения.
type Config struct {
	// File — путь к файлу для записи логов аудита.
	// Настраивается через флаг -audit-file или переменную окружения AUDIT_FILE.
	File string `env:"AUDIT_FILE"`

	// URL — адрес внешнего сервиса (эндпоинта) для отправки событий аудита.
	// Настраивается через флаг -audit-url или переменную окружения AUDIT_URL.
	URL string `env:"AUDIT_URL"`
}

// RegisterFlags связывает поля структуры [Config] с флагами командной строки
// в переданном наборе [flag.FlagSet] и возвращает указатель на созданную конфигурацию.
//
// Метод не вызывает fs.Parse() автоматически, это необходимо сделать на уровне вызывающего кода.
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

// ParseEnv дополняет или перезаписывает текущие значения полей [Config]
// данными из переменных окружения, используя библиотеку [env.Parse].
//
// Возвращает обернутую ошибку, если разбор переменных окружения завершился неудачно.
func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}

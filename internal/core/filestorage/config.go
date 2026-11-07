package filestorage

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config содержит параметры конфигурации для файлового хранилища сокращенных URL.
type Config struct {
	// FilePath — путь к локальному файлу, в который будут записываться данные.
	// Настраивается через короткий флаг -f или переменную окружения FILE_STORAGE_PATH.
	FilePath string `env:"FILE_STORAGE_PATH"`
}

// RegisterFlags связывает поле FilePath структуры [Config] с флагом командной строки "-f"
// в переданном наборе [flag.FlagSet] и возвращает указатель на созданную конфигурацию.
//
// Метод осуществляет только привязку флага. Его фактический разбор (fs.Parse) должен
// выполняться на более высоком уровне приложения (например, в пакете config).
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

// ParseEnv дополняет или перезаписывает текущее значение пути к файлу [Config.FilePath]
// данными из переменной окружения FILE_STORAGE_PATH, используя библиотеку [env.Parse].
//
// Возвращает обернутую ошибку, если разбор переменных окружения завершился неудачно.
func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}

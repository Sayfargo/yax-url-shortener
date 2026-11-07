package httpserver

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config содержит параметры конфигурации для запуска HTTP-сервера
// и формирования сокращенных ссылок.
type Config struct {
	// Addr — сетевой адрес и порт, на которых сервер будет слушать входящие HTTP-запросы.
	// Настраивается через короткий флаг -a или переменную окружения SERVER_ADDRESS.
	Addr string `env:"SERVER_ADDRESS"`

	// BaseURL — базовый адрес (префикс), используемый для формирования итоговых сокращенных ссылок.
	// Настраивается через короткий флаг -b или переменную окружения BASE_URL.
	BaseURL string `env:"BASE_URL"`
}

// RegisterFlags связывает поля структуры [Config] с соответствующими флагами командной строки
// в переданном наборе [flag.FlagSet] и возвращает указатель на созданную конфигурацию.
//
// Для флагов установлены дефолтные значения:
//   - Флаг "-a": "localhost:8080"
//   - Флаг "-b": "http://localhost:8080"
//
// Метод осуществляет только привязку флагов. Их фактический разбор (fs.Parse) должен
// выполняться на более высоком уровне приложения (например, в пакете config).
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

// ParseEnv дополняет или перезаписывает текущие значения полей [Config]
// данными из переменных окружения SERVER_ADDRESS и BASE_URL, используя библиотеку [env.Parse].
//
// Возвращает обернутую ошибку, если разбор переменных окружения завершился неудачно.
func (c *Config) ParseEnv() error {
	if err := env.Parse(c); err != nil {
		return fmt.Errorf("failed to parse env: %w", err)
	}
	return nil
}

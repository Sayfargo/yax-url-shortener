package config

import (
	"flag"
	"fmt"

	config_server "github.com/Sayfargo/yax-url-shortener/internal/config/server"
	config_storage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
)

type Config struct {
	Server      *config_server.Config
	FileStorage *config_storage.Config
}

func Load(args []string) *Config {

	// Используем отдельный flagset чтобы не конфликтовать с глобальным
	// Плюс удобно для unit тестов
	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	serverConfig := config_server.RegisterFlags(fs)
	fileStorageConfig := config_storage.RegisterFlags(fs)

	// Парсим флаги. Если ошибка - обрабатываем и сможем продолжить работу приложения
	if err := fs.Parse(args); err != nil {
		fmt.Printf("Failed to parse flags: %v\n", err) // TODO: Добавить логирование
	}

	// Если переменные окружения заданы, то они перезапишут флаги
	serverConfig.ParseEnv()
	fileStorageConfig.ParseEnv()

	return &Config{
		Server:      serverConfig,
		FileStorage: fileStorageConfig,
	}

}

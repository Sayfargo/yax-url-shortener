package config_slogger

import (
	"io"
	"log/slog"
)

// TODO: Сделать конфигурацию настраиваемой через yaml/env/flags

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Config struct {
	// Директория с log файлами
	Directory string

	// Структура с конфигурацией для вывода в stdout (консоль)
	Stdout StdoutConfig
	// Слайс структур с конфигурациями для файлов логов
	Files []FileConfig
}

type StdoutConfig struct {
	// Enabled даёт возможность выключить вывод логов через конфигурацию
	Enabled bool
	// Format задаёт формат вывода в виде json или text
	Format Format

	Level slog.Level
	// Можно использовать любой Writer + удобно для unit тестов
	Writer io.Writer
}

type FileConfig struct {
	// Наименования лог файла для читабельности
	Name string

	Enabled bool
	Format  Format
	Level   slog.Level
}

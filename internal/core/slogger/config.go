package slogger

import (
	"io"
	"log/slog"
)

// Format определяет формат кодирования и визуального представления логов.
type Format string

const (
	// FormatText указывает на вывод логов в простом текстовом виде (через [slog.TextHandler]).
	FormatText Format = "text"
	// FormatJSON указывает на вывод логов в структурированном виде JSON (через [slog.JSONHandler]).
	FormatJSON Format = "json"
)

// Config агрегирует параметры для инициализации всей подсистемы логирования приложения.
type Config struct {
	// Directory — путь к директории, в которой slogger будет создавать лог-файлы.
	Directory string

	// Stdout содержит настройки для вывода логов в консоль или другой поток вывода.
	Stdout StdoutConfig

	// Files — список конфигураций для создания независимых файлов логирования.
	Files []FileConfig
}

// StdoutConfig описывает параметры вывода логов в стандартный поток вывода ([io.Writer]).
type StdoutConfig struct {
	// Enabled активирует или деактивирует вывод логов в поток командной строки.
	Enabled bool

	// Format задает формат логирования: [FormatText] или [FormatJSON].
	Format Format

	// Level определяет минимальный уровень логирования для этого потока (например, [slog.LevelDebug]).
	Level slog.Level

	// Writer задает целевой поток для записи данных. В продакшене обычно используется [os.Stdout],
	// а в unit-тестах — кастомный буфер (например, [bytes.Buffer]).
	Writer io.Writer
}

// FileConfig описывает параметры для создания и наполнения конкретного файла логов.
type FileConfig struct {
	// Name — человекочитаемый суффикс, добавляемый в имя создаваемого файла логов.
	Name string

	// Enabled активирует или деактивирует запись логов в данный файл.
	Enabled bool

	// Format задает формат кодирования записей в файле: [FormatText] или [FormatJSON].
	Format Format

	// Level определяет минимальный уровень логирования, который будет записываться в этот файл.
	Level slog.Level
}

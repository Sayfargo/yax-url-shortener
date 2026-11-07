package slogger

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Example демонстрирует полный цикл работы с пакетом slogger:
// конфигурацию одновременного вывода в кастомный io.Writer (вместо stdout)
// и в изолированный файл логов, запись сообщений разных уровней
// и безопасное закрытие ресурсов с помощью Closer.
func Example() {
	// 1. Подготавливаем временную директорию для лог-файлов
	tmpDir, err := os.MkdirTemp("", "slogger_example_*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Буфер для перехвата вывода консоли, чтобы go test мог его проверить
	var stdoutBuf bytes.Buffer
	// 2. Формируем конфигурацию для логера
	cfg := Config{
		Directory: tmpDir,
		Stdout: StdoutConfig{
			Enabled: true,
			Format:  FormatText,
			Level:   slog.LevelInfo,
			Writer:  &stdoutBuf,
		},
		Files: []FileConfig{
			{
				Name:    "debug-log",
				Enabled: true,
				Format:  FormatJSON,
				Level:   slog.LevelDebug,
			},
		},
	}
	// 3. Инициализируем slogger
	slog, closer, err := New(cfg)
	if err != nil {
		fmt.Printf("failed to initialize slogger: %v\n", err)
		return
	}
	// Гарантируем закрытие файловых дескрипторов при выходе из функции
	defer func() {
		if err := closer.Close(); err != nil {
			fmt.Printf("falied to close log files: %v\n", err)
		}
	}()
	// 4. Записываем логи разных уровней
	slog.Debug("Debug msg (only file)")
	slog.Info("Info msg (file and stdout)")

	_ = closer.Close()
	// 5. Проверяем вывод в консоль (stdoutBuf)
	// Убираем динамический таймстемп из начала строки ("time=... ") для стабильности теста
	stdoutOutput := stdoutBuf.String()
	if idx := strings.Index(stdoutOutput, "level="); idx != -1 {
		fmt.Print(stdoutOutput[idx:])
	}
	// 6. Проверяем, что файл с логами действительно создался в указанной директории
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		fmt.Printf("failed to read log dir: %v\n", err)
		return
	}

	for _, file := range files {
		if strings.Contains(file.Name(), "debug-log") {
			fmt.Println("Successfully created log file for: debug-log")
		}
	}

	// Output:
	// level=INFO msg="Info msg (file and stdout)"
	// Successfully created log file for: debug-log
}

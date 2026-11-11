package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Example демонстрирует полный жизненный цикл HTTPServer:
// инициализацию с кастомным обработчиком, конкурентный запуск в отдельной горутине
// и последующую плавную остановку (graceful shutdown) через отмену контекста.
func Example() {
	// Настраиваем простой логер, выводящий сообщения в stdout
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		// Отключаем вывод времени и уровня для стабильности теста GoDoc
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	// 1. Создаем конфигурацию сервера на случайном свободном порту
	cfg := &Config{
		Addr:    "127.0.0.1:0", // Порт 0 заставит ОС выделить свободный порт
		BaseURL: "http://127.0.0.1",
	}

	// Простейший обработчик запросов для демонстрации
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 2. Инициализируем сервер
	server := New(handler, cfg, log)

	// Создаем контекст с возможностью отмены для управления жизненным циклом
	ctx, cancel := context.WithCancel(context.Background())

	// Канал для отслеживания завершения работы метода Run
	chErr := make(chan error, 1)

	// 3. Запускаем сервер асинхронно
	go func() {
		chErr <- server.Run(ctx)
	}()

	// Имитируем некоторую работу приложения
	time.Sleep(100 * time.Millisecond)

	// 4. Инициируем плавный останов (Graceful Shutdown) путем отмены контекста
	log.Info("canceling context to trigger shutdown")
	cancel()

	// Ожидаем завершения работы сервера
	if err := <-chErr; err != nil {
		fmt.Printf("server exited with error: %v\n", err)
	}

	// Unordered output:
	// msg="starting http server" Addr=127.0.0.1:0
	// msg="canceling context to trigger shutdown"
	// msg="server starting shutdown"
	// msg="server successfully closed" Addr=127.0.0.1:0
}

package audit

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Example демонстрирует базовый процесс настройки системы аудита:
// создание издателя, регистрацию файлового подписчика и отправку
// событий через высокоуровневый AuditService.
func Example() {

	// Инициализируем издателя (Publisher)
	publisher := NewPublisher()

	// Временный файл для демонстрации вывода на лету
	logFile, err := os.CreateTemp("", "audit_example_*.log")
	if err != nil {
		fmt.Printf("failed to create temp file: %v\n", err)
		return
	}
	defer os.Remove(logFile.Name())

	// Создаем файлового подписчика и регистрируем его у издателя
	fileSub, err := NewFileSubscriber(logFile.Name())
	if err != nil {
		fmt.Printf("failed to create subscriber: %v\n", err)
		return
	}
	defer fileSub.Close()

	publisher.Subscribe(fileSub)

	// Создаем сервис-обертку поверх издателя для удобного вызова в коде бизнес-логики
	auditService := NewAuditService(publisher, slog.New(slog.DiscardHandler), 100, 6, 1*time.Second)

	// Фиксируем события в аудит (внутри используется time.Now(), в лог пойдет реальный TS)
	auditService.Shorten("user_abc", "https://google.com")
	auditService.Follow("user_xyz", "https://short.url")

	// Закрываем файл перед чтением, чтобы сбросить буферы на диск
	_ = fileSub.Close()

	// Считываем записанный результат и выводим его в stdout для проверки GoDoc
	data, err := os.ReadFile(logFile.Name())
	if err != nil {
		fmt.Printf("failed to read log file: %v\n", err)
		return
	}

	fmt.Print(string(data))
}

// ExamplePublisher_Publish демонстрирует работу низкоуровневой публикации событий
// напрямую через объект Publisher.
func ExamplePublisher_Publish() {
	// Создаем кастомного демонстрационного подписчика, который просто печатает события
	stdoutSub := &logSubscriber{}

	p := NewPublisher()
	p.Subscribe(stdoutSub)

	event := &Event{
		TS:     1717171717,
		Action: "custom_action",
		UserID: "user_42",
		URL:    "https://example.com",
	}

	// Публикуем событие напрямую
	err := p.Publish(context.Background(), event)
	if err != nil {
		fmt.Printf("publish error: %v\n", err)
	}

	// Output:
	// Received event: action=custom_action, user=user_42, url=https://example.com
}

type logSubscriber struct{}

func (s *logSubscriber) Notify(ctx context.Context, e *Event) error {
	fmt.Printf("Received event: action=%s, user=%s, url=%s\n", e.Action, e.UserID, e.URL)
	return nil
}

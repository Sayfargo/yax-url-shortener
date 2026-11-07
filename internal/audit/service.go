package audit

import (
	"context"
	"time"
)

// AuditService предоставляет высокоуровневый API поверх [Publisher] для удобной
// отправки специфичных бизнес-событий в систему аудита.
//
// Структура абстрагирует создание объектов [Event], автоматически заполняя
// временные метки и типы действий.
type AuditService struct {
	publisher *Publisher
}

// NewAuditService создает и возвращает новый экземпляр [AuditService],
// использующий переданный [Publisher] для отправки событий.
func NewAuditService(publisher *Publisher) *AuditService {
	return &AuditService{
		publisher: publisher,
	}
}

// Shorten регистрирует в аудите событие сокращения ссылки (действие "shorten").
// Метод автоматически фиксирует текущее время в формате Unix Timestamp и
// передает событие в нижележащий [Publisher].
func (a *AuditService) Shorten(ctx context.Context, userID, url string) {
	event := &Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    url,
	}

	_ = a.publisher.Publish(ctx, event)
}

// Follow регистрирует в аудите событие перехода по сокращенной ссылке (действие "follow").
// Метод автоматически фиксирует текущее время в формате Unix Timestamp и
// передает событие в нижележащий [Publisher].
func (a *AuditService) Follow(ctx context.Context, userID, url string) {
	event := &Event{
		TS:     time.Now().Unix(),
		Action: "follow",
		UserID: userID,
		URL:    url,
	}

	_ = a.publisher.Publish(ctx, event)
}

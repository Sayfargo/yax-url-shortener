package audit

import (
	"context"
	"time"
)

// AuditService - структура поверх Publisher, которая реализует интерфейс Subscriber и позволяет публиковать события в аудит
// для более удобного использования в сервисах
type AuditService struct {
	publisher *Publisher
}

func NewAuditService(publisher *Publisher) *AuditService {
	return &AuditService{
		publisher: publisher,
	}
}

// Shorten - метод, который публикует событие "shorten" в аудит
func (a *AuditService) Shorten(ctx context.Context, userID, url string) {
	event := &Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    url,
	}

	a.publisher.Publish(ctx, event)
}

// Follow - метод, который публикует событие "follow" в аудит
func (a *AuditService) Follow(ctx context.Context, userID, url string) {
	event := &Event{
		TS:     time.Now().Unix(),
		Action: "follow",
		UserID: userID,
		URL:    url,
	}

	a.publisher.Publish(ctx, event)
}

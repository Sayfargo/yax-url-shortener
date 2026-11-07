package audit

import (
	"context"
	"errors"
)

// Event представляет запись события аудита для URL Shortener Service.
// Содержит информацию о времени и действии произошедшего события, а так же информацию о пользователи.
type Event struct {
	// TS — временная метка события в формате Unix Timestamp.
	TS int64 `json:"ts"`
	// Action — описание совершенного действия (например, "shorten").
	Action string `json:"action"`
	// UserID — уникальный идентификатор пользователя, совершившего действие.
	// Может быть пустым, если действие выполнено неавторизованным пользователем.
	UserID string `json:"user_id,omitempty"`
	// URL — адрес оригинального URL, который был сокращен с помощью сервиса.
	URL string `json:"url"`
}

// Subscriber определяет интерфейс для компонентов(подписчиков), обрабатывающие событие аудита.
// Любая структура, реализующая этот интерфейс, может быть подписана на уведомления Publisher.
type Subscriber interface {
	// Notify обрабатывает входящее событие аудита.
	// Возвращает ошибку, если обработка события завершилась неудачно либо отменилась по контексту.
	Notify(ctx context.Context, event *Event) error
}

// Publisher управляет списком подписчиков и отвечает за публикацию
// событий аудита всем зарегистрированным подписчикам.
type Publisher struct {
	subs []Subscriber
}

// NewPublisher конструктор для инициализации нового экземпляра Publisher ready-to-use.
func NewPublisher() *Publisher {
	return &Publisher{
		subs: []Subscriber{},
	}
}

// Subscribe регистрирует нового подписчика в системе.
// Подписчик будет получать все последующие события, отправленные через метод Publish.
func (p *Publisher) Subscribe(sub Subscriber) {
	p.subs = append(p.subs, sub)
}

// Publish отправляет переданное событие всем зарегистрированным подписчикам поочередно.
// Если один или несколько подписчиков возвращают ошибку, Publish собирает их все
// с помощью [errors.Join] и возвращает общую ошибку после завершения обхода.
func (p *Publisher) Publish(ctx context.Context, e *Event) error {
	var errs error

	for _, s := range p.subs {
		err := s.Notify(ctx, e)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

package audit

import (
	"context"
	"errors"
	"sync"
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
	mu   sync.RWMutex
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
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subs = append(p.subs, sub)
}

// Publish отправляет переданное событие всем зарегистрированным подписчикам поочередно.
// Если один или несколько подписчиков возвращают ошибку, Publish собирает их все
// с помощью [errors.Join] и возвращает общую ошибку после завершения обхода.
func (p *Publisher) Publish(ctx context.Context, e *Event) error {
	p.mu.RLock()
	subs := make([]Subscriber, len(p.subs))
	copy(subs, p.subs)
	p.mu.RUnlock()

	var (
		errs  error
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 16)
		errCh = make(chan error, len(subs))
	)

	for _, s := range subs {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			continue
		case sem <- struct{}{}:
		}
		wg.Go(func() {
			if err := s.Notify(ctx, e); err != nil {
				errCh <- err
			}
			<-sem
		})
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		errs = errors.Join(errs, err)
	}

	return errs
}

package audit

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// EventPublisher определяет интерфейс для отправки и регистрации обработчиков
//
// событий аудита.
//
// Реализация интерфейса отвечает за фактическую доставку событий
// (например, в файл, HTTP endpoint или другой внешний источник).
type EventPublisher interface {
	Publish(ctx context.Context, e *Event) error
	Subscribe(sub Subscriber)
}

// AuditService предоставляет высокоуровневый API для регистрации бизнес-событий
// в системе аудита.
//
// Сервис автоматически формирует объекты Event, добавляет временные метки,
// определяет тип действия и выполняет асинхронную отправку событий через
// переданный EventPublisher.
//
// Для повышения надёжности используется внутренняя очередь задач и временное
// хранилище переполнения, позволяющее не терять события при кратковременной
// недоступности обработчиков.
type AuditService struct {
	publisher EventPublisher
	l         *slog.Logger
	// worker opts
	events          chan *Event
	mu              sync.Mutex
	overflow        []*Event
	workers         int
	pollingInterval time.Duration

	wg   sync.WaitGroup
	once sync.Once
}

// NewAuditService создает новый экземпляр AuditService.
//
// publisher используется для фактической отправки событий аудита.
// queueSize задаёт максимальный размер основной очереди событий.
// workers определяет количество фоновых обработчиков очереди.
// pollingInterval задаёт период проверки временного хранилища переполнения.
func NewAuditService(
	publisher EventPublisher,
	logger *slog.Logger,
	queueSize int,
	workers int,
	pollingInterval time.Duration,
) *AuditService {
	return &AuditService{
		publisher:       publisher,
		l:               logger,
		events:          make(chan *Event, queueSize),
		overflow:        make([]*Event, 0),
		workers:         workers,
		pollingInterval: pollingInterval,
	}
}

// StartAuditWorkerPool запускает фоновые обработчики событий аудита.
//
// Метод запускает worker pool, который получает события из внутренней очереди
// и передает их в EventPublisher.
//
// Дополнительно запускается периодический обработчик overflow-очереди,
// который пытается повторно отправить события, не попавшие в основную очередь.
//
// Метод безопасен для повторного вызова: фактический запуск выполняется только
// один раз.
func (a *AuditService) StartAuditWorkerPool(ctx context.Context) {
	a.once.Do(func() {
		for range a.workers {
			a.wg.Go(func() {
				a.work(ctx)
			})
		}

		a.wg.Go(func() {
			a.polling(ctx)
		})
	})
}

func (a *AuditService) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-a.events:
			a.publish(ctx, event)
		}
	}
}

func (a *AuditService) polling(ctx context.Context) {
	ticker := time.NewTicker(a.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.flushOverflow()
		}
	}
}

func (a *AuditService) flushOverflow() {
	a.mu.Lock()
	if len(a.overflow) == 0 {
		a.mu.Unlock()
		return
	}
	a.l.Debug(
		"audit worker start polling process",
		"overflow_size", len(a.overflow),
	)

	events := a.overflow
	a.overflow = make([]*Event, 0)
	a.mu.Unlock()

	var remainingEvents []*Event

	for _, event := range events {
		select {
		case a.events <- event:
		default:
			remainingEvents = append(remainingEvents, event)
		}
	}

	if len(remainingEvents) > 0 {
		a.mu.Lock()
		a.overflow = append(remainingEvents, a.overflow...)
		a.l.Debug(
			"some events still blocked due overflow queue",
			"overflow_size", len(a.overflow),
		)
		a.mu.Unlock()
	}
}

func (a *AuditService) enqueue(e *Event) {
	select {
	case a.events <- e:
	default:
		a.mu.Lock()
		a.overflow = append(a.overflow, e)
		a.l.Debug(
			"audit worker event queue is full: event moved to temp overflow cache",
			"action", e.Action,
		)
		a.mu.Unlock()
	}
}

func (a *AuditService) shutdown(ctx context.Context) {
	a.l.Info(
		"audit service shutting down",
		"pending_events", len(a.events),
	)
	a.wg.Wait()

	for {
		select {
		case event := <-a.events:
			if err := a.publisher.Publish(ctx, event); err != nil {
				a.l.Error(
					"failed to publish event",
					"err", err,
				)
			}
		default:
			goto overflow
		}

	}

overflow:
	a.mu.Lock()
	events := a.overflow
	a.overflow = nil
	a.mu.Unlock()

	for _, event := range events {
		if err := a.publisher.Publish(ctx, event); err != nil {
			a.l.Error(
				"failed to publish event",
				"err", err,
			)
		}
	}

	a.l.Info(
		"audit shutdown finished",
	)
}

// Shutdown запускает процесс плавного завершения работы воркер пула.
func (a *AuditService) Shutdown(ctx context.Context) {
	a.shutdown(ctx)
}

// Shorten регистрирует событие сокращения ссылки.
//
// Создает событие аудита с действием "shorten", текущей временной меткой,
// идентификатором пользователя и исходным URL, после чего помещает его
// во внутреннюю очередь для асинхронной отправки.
func (a *AuditService) Shorten(userID, url string) {
	event := &Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    url,
	}

	a.enqueue(event)
}

// Follow регистрирует событие перехода по сокращенной ссылке.
//
// Создает событие аудита с действием "follow", текущей временной меткой,
// идентификатором пользователя и URL, после чего помещает его во внутреннюю
// очередь для асинхронной отправки.
func (a *AuditService) Follow(userID, url string) {
	event := &Event{
		TS:     time.Now().Unix(),
		Action: "follow",
		UserID: userID,
		URL:    url,
	}

	a.enqueue(event)
}

func (a *AuditService) publish(ctx context.Context, e *Event) {
	// игнорируем отмену главного контекста
	wrapped, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	// при отмене главного контекста даём время на то, чтобы Publish доработал
	// если нет, то принудительное завершение работы через отмену прокинутого контекста
	stop := context.AfterFunc(ctx, func() {
		time.AfterFunc(3*time.Second, cancel)
	})
	defer stop()

	if err := a.publisher.Publish(wrapped, e); err != nil {
		a.l.Error(
			"failed to publish event",
			"err", err,
		)
	}
}

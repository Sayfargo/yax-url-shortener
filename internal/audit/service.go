package audit

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// IPublisher определяет интерфейс для отправки и регистрации обработчиков
// событий аудита.
//
// Реализация интерфейса отвечает за фактическую доставку событий
// (например, в файл, HTTP endpoint или другой внешний источник).
type IPublisher interface {
	Publish(ctx context.Context, e *Event) error
	Subscribe(sub Subscriber)
}

// AuditService предоставляет высокоуровневый API для регистрации бизнес-событий
// в системе аудита.
//
// Сервис автоматически формирует объекты Event, добавляет временные метки,
// определяет тип действия и выполняет асинхронную отправку событий через
// переданный IPublisher.
//
// Для повышения надёжности используется внутренняя очередь задач и временное
// хранилище переполнения, позволяющее не терять события при кратковременной
// недоступности обработчиков.
type AuditService struct {
	publisher IPublisher
	l         *slog.Logger
	// worker opts
	jobs            chan *Event
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
	publisher IPublisher,
	logger *slog.Logger,
	queueSize int,
	workers int,
	pollingInterval time.Duration,
) *AuditService {
	return &AuditService{
		publisher:       publisher,
		l:               logger,
		jobs:            make(chan *Event, queueSize),
		overflow:        make([]*Event, 0),
		workers:         workers,
		pollingInterval: pollingInterval,
	}
}

// StartAuditWorkerPool запускает фоновые обработчики событий аудита.
//
// Метод запускает worker pool, который получает события из внутренней очереди
// и передает их в IPublisher.
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
			a.pooling(ctx)
		})

		go func() {
			<-ctx.Done()
			a.shutdown()
		}()
	})
}

func (a *AuditService) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-a.jobs:
			if !ok {
				return
			}
			a.publish(ctx, job)
		}
	}
}

func (a *AuditService) pooling(ctx context.Context) {
	ticker := time.NewTicker(a.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.flushOverflow(ctx)
		}
	}
}

func (a *AuditService) flushOverflow(ctx context.Context) {
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
		case <-ctx.Done():
			return
		case a.jobs <- event:
		default:
			remainingEvents = append(remainingEvents, event)
		}
	}

	if len(remainingEvents) > 0 {
		a.mu.Lock()
		// держим строгую очередь, добавляя новые пришедшие события в конец, чтобы все были довольны
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
	case a.jobs <- e:
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

func (a *AuditService) shutdown() {
	a.l.Info(
		"audit service shutting down",
		"pending_jobs", len(a.jobs),
	)

	// дожидаемся всех воркеров в том числе пулинг воркера
	a.wg.Wait()
	// даём 5 секунд времени на то чтобы разгребсти оставшиеся джобы
	shutDownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case event := <-a.jobs:
			a.publish(shutDownCtx, event)
		default:
			goto overflow
		}
	}
	// если джобов больше не осталось отправляем разребать накопившийся overflow
overflow:
	a.mu.Lock()
	events := a.overflow
	a.overflow = nil
	a.mu.Unlock()

	for _, event := range events {
		a.publish(shutDownCtx, event)
	}

	a.l.Info(
		"audit shutdown finished",
	)
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
	if err := a.publisher.Publish(ctx, e); err != nil {
		a.l.Error(
			"failed to publish event",
			"err", err,
		)

	}
}

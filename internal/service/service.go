package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	urlrepo "github.com/Sayfargo/yax-url-shortener/internal/repository/url"
	"github.com/google/uuid"
)

// URLRepository определяет контракт взаимодействия сервиса со слоем хранения данных.
type URLRepository interface {
	Create(ctx context.Context, shortenedURL model.ShortenedURL) error
	CreateBatch(ctx context.Context, shortenedURLs []model.ShortenedURL) error
	Get(ctx context.Context, shortCode string) (string, error)
	GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error)
	SoftDeleteURLs(ctx context.Context, uid uuid.UUID, shortCodes ...string) error
}

// Generator определяет интерфейс для генерации уникальных буквенно-цифровых кодов.
type Generator interface {
	Generate() (string, error)
}

// DeletedTask описывает единичную задачу на удаление, содержащую идентификатор
// пользователя и список принадлежащих ему коротких кодов для удаления.
type DeletedTask struct {
	UID        uuid.UUID
	ShortCodes []string
}

// Option определяет сигнатуру функций для конфигурации параметров [URLShortenerService].
type Option func(*URLShortenerService)

// URLShortenerService предоставляет методы бизнес-логики для управления жизненным
// циклом сокращенных ссылок.
type URLShortenerService struct {
	repo      URLRepository
	generator Generator
	log       *slog.Logger
	baseURL   string
	// delete worker opts
	deleteQueue      chan DeletedTask
	deleteWorkerOnce sync.Once
	deleteMaxSize    int
	deleteMaxWait    time.Duration
}

// New инициализирует и возвращает новый экземпляр [URLShortenerService].
// По умолчанию воркер удаления настраивается на буфер из 100 задач, максимальный
// размер пачки — 10 элементов, и время ожидания сброса неполного батча — 100 мс.
func New(
	repo URLRepository,
	generator Generator,
	baseURL string,
	log *slog.Logger,
	opts ...Option,
) *URLShortenerService {
	return &URLShortenerService{
		repo:      repo,
		generator: generator,
		baseURL:   baseURL,
		log:       log,

		deleteQueue:   make(chan DeletedTask, 100),
		deleteMaxSize: 10,
		deleteMaxWait: 100 * time.Millisecond,
	}
}

// WithDeleteQueue настраивает фоновый воркер удаления.
// Без этой опции используются дефолтные значения (очередь: 100, батч: 10, таймер: 100мс).
//
// Параметры:
//   - chanQueueSize: размер канала очереди задач
//   - buffMaxSize: максимальный размер заполнения буфера для отправки пачки в БД
//   - timerMaxWait: максимальное время ожидания для отправки неполной пачки на диск/БД
func WithDeleteQueue(chanQueueSize, buffMaxSize int, timerMaxWait time.Duration) Option {
	return func(s *URLShortenerService) {
		if chanQueueSize > 0 {
			s.deleteQueue = make(chan DeletedTask, chanQueueSize)
		}
		if buffMaxSize > 0 {
			s.deleteMaxSize = buffMaxSize
		}
		if timerMaxWait > 0 {
			s.deleteMaxWait = timerMaxWait
		}
	}
}

// StartDeleteWorker безопасно и строго однократно запускает горутину фоновой
// обработки пакетов удаления ссылок.
func (s *URLShortenerService) StartDeleteWorker(ctx context.Context) {
	s.deleteWorkerOnce.Do(func() {
		go s.deleteWorker(ctx)
	})
}

var (
	// ErrURLDoesNotExists возвращается, если запрашиваемый короткий код отсутствует в системе.
	ErrURLDoesNotExists = errors.New("requested URL code does not exists")
	// ErrShortCodeCollisionLimitExceeded возвращается, если после 3 попыток генерации не удалось получить уникальный код.
	ErrShortCodeCollisionLimitExceeded = errors.New("failed to generate short code after all attempts")
	// ErrIncorrectURL возвращается при передаче невалидной структуры URL-адреса.
	ErrIncorrectURL = errors.New("incorrect url")
	// ErrCorruptedData возвращается, если данные в кэше повреждены.
	ErrCorruptedData = errors.New("cached data is corrupted or invalid")
	// ErrEmptyBatch возвращается при отправке пустого массива в пакетных операциях.
	ErrEmptyBatch = errors.New("empty batch")
	// ErrOriginalURLConflict возвращается, если оригинальный URL уже сокращен в системе.
	ErrOriginalURLConflict = errors.New("original url conflict")
	// ErrURLsNotFound возвращается, если у пользователя нет созданных ссылок.
	ErrURLsNotFound = errors.New("no URLs found for user")
	// ErrURLIsDeleted возвращается при попытке доступа к ссылке, которая была удалена.
	ErrURLIsDeleted = errors.New("requested URL has been deleted")
)

// CreateURLBatchRequest описывает элемент входящего массива для пакетного сокращения ссылок.
type CreateURLBatchRequest struct {
	CorrelationID string
	OriginalURL   string
}

// CreateURLBatchResponse описывает элемент исходящего массива с готовой ссылкой.
type CreateURLBatchResponse struct {
	CorrelationID string
	ShortURL      string
}

// GetURLsResponse описывает маппинг короткой и оригинальной ссылки.
type GetURLsResponse struct {
	ShortURL    string
	OriginalURL string
}

// DeleteURLs помещает задачу на удаление списка кодов в неблокирующую очередь [deleteQueue].
// Фактическое удаление происходит асинхронно фоновым воркером.
//
// Возвращает ошибку, если контекст отменен или парсинг UID завершился неудачно.
func (s *URLShortenerService) DeleteURLs(ctx context.Context, uid string, shortCodes ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return fmt.Errorf("uuid parse: %w", err)
	}

	task := DeletedTask{
		UID:        uidUUID,
		ShortCodes: shortCodes,
	}

	select {
	case s.deleteQueue <- task:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetUserURLs извлекает все активные ссылки пользователя и преобразует
// короткие коды в полноценные абсолютные URL-адреса с префиксом [baseURL].
func (s *URLShortenerService) GetUserURLs(ctx context.Context, uid string) ([]model.UserURL, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return nil, fmt.Errorf("uuid parse: %w", err)
	}

	result, err := s.repo.GetURLs(ctx, uidUUID)
	if err != nil {
		if errors.Is(err, urlrepo.ErrNoRows) {
			return nil, ErrURLsNotFound
		}
		return nil, fmt.Errorf("repository get URLs: %w", err)
	}

	response := make([]model.UserURL, len(result))

	for i, u := range result {
		response[i].ShortURL = s.buildShortedURL(u.ShortCode)
		response[i].OriginalURL = u.OriginalURL
	}

	return response, nil
}

// CreateURLBatch выполняет пакетное создание укороченных ссылок.
//
// Метод поддерживает механизм автоматического разрешения коллизий: если репозиторий
// возвращает ошибку нарушения уникальности кода, сервис заново перегенерирует код
// для сбойной строки и повторяет попытку (до 3 итераций на весь батч).
func (s *URLShortenerService) CreateURLBatch(ctx context.Context, req []CreateURLBatchRequest, uid string) ([]CreateURLBatchResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if len(req) == 0 {
		return nil, ErrEmptyBatch
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return nil, fmt.Errorf("uuid from bytes: %w", err)
	}

	shortenedURLs := make([]model.ShortenedURL, len(req))
	resp := make([]CreateURLBatchResponse, len(req))

	for i, item := range req {
		if err := validateURL(item.OriginalURL); err != nil {
			return nil, fmt.Errorf(
				"got incorrect url on row #%d with id %s: %w",
				i+1, item.CorrelationID,
				ErrIncorrectURL,
			)
		}

		shortCode, err := s.generator.Generate()
		if err != nil {
			return nil, fmt.Errorf("generator generate: %w", err)

		}

		shortenedURLs[i] = model.ShortenedURL{
			UUID:        uuid.New(),
			ShortCode:   shortCode,
			OriginalURL: item.OriginalURL,
			UserID:      uidUUID,
			IsDeleted:   false,
		}

		resp[i] = CreateURLBatchResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.buildShortedURL(shortCode),
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		err := s.repo.CreateBatch(ctx, shortenedURLs)

		var batchErr *urlrepo.BatchConflictError

		switch {
		case err == nil:
			return resp, nil
		case errors.As(err, &batchErr):

			if attempt == 2 {
				break
			}

			s.log.Info(
				"short code collision occurred, retrying",
				"url", shortenedURLs[batchErr.Index].OriginalURL,
				"code", shortenedURLs[batchErr.Index].ShortCode,
				"attempt", attempt+1,
			)

			shortCode, err := s.generator.Generate()
			if err != nil {
				return nil, fmt.Errorf("generator generate: %w", err)
			}

			shortenedURLs[batchErr.Index].ShortCode = shortCode
			resp[batchErr.Index].ShortURL = s.buildShortedURL(shortCode)

			continue
		default:
			return nil, fmt.Errorf("repository create: %w", err)
		}
	}

	return nil, ErrShortCodeCollisionLimitExceeded
}

// GetOriginalURL запрашивает репозиторий и возвращает исходную ссылку по коду.
// Мапит внутренние ошибки хранилища в понятные для доменного слоя бизнес-ошибки пакета.
func (s *URLShortenerService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	originalURL, err := s.repo.Get(ctx, shortCode)
	if err != nil {
		if errors.Is(err, urlrepo.ErrNotExists) {
			return "", ErrURLDoesNotExists
		} else if errors.Is(err, urlrepo.ErrUnexpectedType) {
			return "", ErrCorruptedData
		} else if errors.Is(err, urlrepo.ErrRowGone) {
			return "", ErrURLIsDeleted
		}
		return "", fmt.Errorf("repository get err: %w", err)
	}

	return originalURL, nil

}

// CreateShortURL генерирует короткий код, проверяет валидность ссылки и сохраняет запись.
//
// Обладает встроенным циклом разрешения коллизий кодов на 3 попытки.
// Если в процессе вставки выясняется, что данный длинный URL уже регистрировался
// (ошибка [urlrepo.OriginalURLConflictError]), метод возвращает ранее созданный адрес
// вместе со статусом-ошибкой [ErrOriginalURLConflict].
func (s *URLShortenerService) CreateShortURL(ctx context.Context, url, uid string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return "", fmt.Errorf("uuid from bytes: %w", err)
	}

	if err := validateURL(url); err != nil {
		return "", ErrIncorrectURL
	}

	for attempt := 0; attempt < 3; attempt++ {

		shortCode, err := s.generator.Generate()
		if err != nil {
			return "", fmt.Errorf("generator generate: %w", err)
		}

		shortenedURL := model.ShortenedURL{
			UUID:        uuid.New(),
			ShortCode:   shortCode,
			OriginalURL: url,
			UserID:      uidUUID,
			IsDeleted:   false,
		}

		err = s.repo.Create(ctx, shortenedURL)

		var origURLConflictErr *urlrepo.OriginalURLConflictError

		switch {
		case err == nil:

			return s.buildShortedURL(shortCode), nil

		case errors.Is(err, urlrepo.ErrConflictShortCode):

			s.log.Info(
				"short code collision occurred, retrying",
				"url", url,
				"code", shortCode,
				"attempt", attempt+1,
			)
			continue
		case errors.As(err, &origURLConflictErr):

			return s.buildShortedURL(origURLConflictErr.ShortCode), ErrOriginalURLConflict

		default:

			return "", fmt.Errorf("repository create: %w", err)
		}
	}

	return "", ErrShortCodeCollisionLimitExceeded
}

func (s *URLShortenerService) buildShortedURL(shortCode string) string {
	return s.baseURL + "/" + shortCode // Example: http://localhost:8080/EiXk21Dz
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("parse request URI: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("incorrect scheme")
	}
	return nil
}

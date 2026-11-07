package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/ctxkeys"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URLShortener описывает контракт взаимодействия с сервисом бизнес-логики сокращения ссылок.
type URLShortener interface {
	CreateShortURL(ctx context.Context, url, uid string) (string, error)
	CreateURLBatch(ctx context.Context, req []service.CreateURLBatchRequest, uid string) ([]service.CreateURLBatchResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	GetUserURLs(ctx context.Context, uid string) ([]model.UserURL, error)
	DeleteURLs(ctx context.Context, uid string, shortCodes ...string) error
}

// AuditService определяет методы логирования действий пользователей для системы аудита.
type AuditService interface {
	Shorten(ctx context.Context, userID, url string)
	Follow(ctx context.Context, userID, url string)
}

// Handler группирует HTTP-эндпоинты и инкапсулирует зависимости: логер, базу данных
// и сервисы бизнес-логики/аудита.
type Handler struct {
	auditService AuditService
	service      URLShortener
	log          *slog.Logger
	pool         *pgxpool.Pool
}

const maxBodySize = 1024 * 1024

// New инициализирует и возвращает новый экземпляр HTTP-обработчика [Handler].
func New(service URLShortener, log *slog.Logger, pool *pgxpool.Pool, auditService AuditService) *Handler {
	return &Handler{
		auditService: auditService,
		service:      service,
		log:          log,
		pool:         pool,
	}
}

// RegisterPublicAPI регистрирует эндпоинты, доступные без прохождения авторизации
// и проверки сессионных кук (например, редиректы и проверка здоровья).
func (h *Handler) RegisterPublicAPI(r chi.Router) {
	r.Get("/{id}", h.Redirect)
	r.Get("/ping", h.Ping)
}

// RegisterPrivateAPI регистрирует эндпоинты, защищенные авторизационным middleware.
// Данные маршруты требуют обязательного наличия идентификатора пользователя в контексте.
func (h *Handler) RegisterPrivateAPI(r chi.Router) {
	r.Post("/", h.Create)
	r.Post("/api/shorten", h.Shorten)
	r.Post("/api/shorten/batch", h.ShortenBatch)
	r.Get("/api/user/urls", h.GetURLs)
	r.Delete("/api/user/urls", h.DeleteURLs)
}

// DeleteURLs принимает массив коротких кодов для их асинхронного удаления.
// Запрос ограничивается размером в 1 МБ ([maxBodySize]).
//
// Возвращаемые HTTP-статусы:
//   - 202 Accepted: Запрос на удаление успешно принят в обработку.
//   - 400 Bad Request: Передан пустой массив или тело запроса повреждено.
//   - 413 Payload Too Large: Тело запроса превышает допустимый лимит 1 МБ.
//   - 499 Client Closed Request: Клиент разорвал соединение до завершения операции.
//   - 500 Internal Server Error: Ошибка извлечения UID или неожиданный сбой сервиса.
func (h *Handler) DeleteURLs(w http.ResponseWriter, r *http.Request) {

	uid, ok := h.getUserID(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	var shortCodes []string

	body := http.MaxBytesReader(w, r.Body, maxBodySize)

	if err := json.NewDecoder(body).Decode(&shortCodes); err != nil {

		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			h.log.Warn("request body exceeded max allowed size",
				slog.Int64("limit_bytes", maxBytesErr.Limit),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			w.WriteHeader(http.StatusRequestEntityTooLarge)
		} else {
			h.log.Info(
				"failed to decode json body",
				"err", err,
			)
			http.Error(w, "failed to read request body", http.StatusBadRequest)
		}
		return
	}

	if len(shortCodes) == 0 {

		h.log.Info(
			"no short codes provided for deletion",
		)

		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	err := h.service.DeleteURLs(r.Context(), uid, shortCodes...)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.log.Debug("delete URLs canceled by client")
			w.WriteHeader(499)
		} else {
			h.log.Error(
				"unexpected error during url deletion",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetURLs возвращает JSON-массив всех сокращенных ссылок, принадлежащих текущему авторизованному пользователю.
//
// Возвращаемые HTTP-статусы:
//   - 200 OK: Успешный возврат списка ссылок в формате JSON-массива [GetURLsResponse].
//   - 240 NoContent: У пользователя пока нет созданных сокращенных ссылок.
//   - 499 Client Closed Request: Клиент закрыл соединение.
//   - 500 Internal Server Error: Внутренний сбой сервера или ошибка сериализации JSON.
func (h *Handler) GetURLs(w http.ResponseWriter, r *http.Request) {

	uid, ok := h.getUserID(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	result, err := h.service.GetUserURLs(r.Context(), uid)
	if err != nil {
		if errors.Is(err, service.ErrURLsNotFound) {
			w.WriteHeader(http.StatusNoContent)
		} else if errors.Is(err, context.Canceled) {
			h.log.Debug("get URLs canceled by client")
			w.WriteHeader(499)
		} else {
			h.log.Error(
				"unexpected error",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	data, err := json.Marshal(result)
	if err != nil {
		h.log.Error(
			"failed to marshal response json",
			"err", err,
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ShortenBatch выполняет массовое (пакетное) сокращение переданного массива ссылок.
// Метод принимает массив элементов [CreateURLBatchRequest] и возвращает массив [CreateURLBatchResponse].
//
// Возвращаемые HTTP-статусы:
//   - 201 Created: Пакет успешно обработан, ссылки сгенерированы.
//   - 400 Bad Request: Пустой пакет, некорректный формат ссылок или ошибка JSON-декодера.
//   - 413 Payload Too Large: Размер JSON-пакета превышает лимит в 1 МБ.
//   - 499 Client Closed Request: Клиент разорвал соединение.
//   - 500 Internal Server Error: Системная ошибка коллизии кодов или сбой базы данных.
func (h *Handler) ShortenBatch(w http.ResponseWriter, r *http.Request) {

	var request []CreateURLBatchRequest

	uid, ok := h.getUserID(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	body := http.MaxBytesReader(
		w,
		r.Body,
		maxBodySize,
	)

	if err := json.NewDecoder(body).Decode(&request); err != nil {

		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {

			h.log.Warn("request body exceeded max allowed size",
				slog.Int64("limit_bytes", maxBytesErr.Limit),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			w.WriteHeader(http.StatusRequestEntityTooLarge)
		} else {
			h.log.Info(
				"failed to decode json body",
				"err", err,
			)

			http.Error(w, "failed to read request body", http.StatusBadRequest)
		}
		return
	}

	serviceReq := make([]service.CreateURLBatchRequest, len(request))

	for i, item := range request {
		serviceReq[i] = service.CreateURLBatchRequest{
			CorrelationID: item.CorrelationID,
			OriginalURL:   item.OriginalURL,
		}
	}

	result, err := h.service.CreateURLBatch(r.Context(), serviceReq, uid)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.log.Debug("create short url canceled by client")
			w.WriteHeader(499)
		} else if errors.Is(err, service.ErrIncorrectURL) {

			h.log.Info(
				"incorrect url request",
				"err", err,
			)
			// ошибка содержит информацию на какой именно строке и с каким id возникла ошибка для удобства
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if errors.Is(err, service.ErrShortCodeCollisionLimitExceeded) {
			h.log.Warn(
				"short code collision limit exceeded",
			)
			http.Error(w, "failed to process request, please try again", http.StatusInternalServerError)
		} else if errors.Is(err, service.ErrEmptyBatch) {
			http.Error(w, err.Error(), http.StatusBadRequest)

		} else {
			h.log.Error(
				"unexpected error during url shortening",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	response := make([]CreateURLBatchResponse, len(result))

	for i, u := range result {
		response[i].CorrelationID = u.CorrelationID
		response[i].ShortURL = u.ShortURL
	}

	data, err := json.Marshal(response)
	if err != nil {
		h.log.Error(
			"failed to marshal response json",
			"err", err,
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)

}

// Ping проверяет доступность подключения к базе данных PostgreSQL.
// Используется балансировщиками нагрузки и системами мониторинга в качестве liveness/readiness пробы.
//
// Возвращаемые HTTP-статусы:
//   - 200 OK: Подключение активно, база данных отвечает на запросы.
//   - 500 Internal Server Error: Пул соединений равен nil или база данных не ответила на Ping.
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {

	if h.pool == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.pool.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Redirect извлекает короткий код URL из параметров пути роутера, ищет оригинальный
// веб-адрес и осуществляет HTTP-редирект.
//
// Возвращаемые HTTP-статусы:
//   - 307 Temporary Redirect: Успешный редирект на найденный оригинальный URL-адрес.
//   - 400 Bad Request: Идентификатор ссылки (параметр пути) оказался пустым.
//   - 404 Not Found: Переданный короткий код отсутствует в системе.
//   - 410 Gone: Запрашиваемая ссылка была удалена пользователем ранее.
//   - 500 Internal Server Error: Системная ошибка кэша или неожиданный сбой при поиске.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {

	shortCode := chi.URLParam(r, "id")
	if shortCode == "" {
		http.Error(w, "URL Param is empty", http.StatusBadRequest)
		return
	}

	OriginalURL, err := h.service.GetOriginalURL(r.Context(), shortCode)
	if err != nil {
		if errors.Is(err, service.ErrURLDoesNotExists) {

			h.log.Info(
				"url does not exists",
				"code", shortCode,
			)

			http.NotFound(w, r)
		} else if errors.Is(err, service.ErrCorruptedData) {

			h.log.Error(
				"internal cache error",
				"err", err,
				"code", shortCode,
			)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else if errors.Is(err, service.ErrURLIsDeleted) {
			h.log.Info(
				"requested URL has been deleted",
				"code", shortCode,
			)
			w.WriteHeader(http.StatusGone)
		} else {
			h.log.Error(
				"unexpected error during redirect",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	h.auditService.Follow(r.Context(), "", OriginalURL)
	http.Redirect(w, r, OriginalURL, http.StatusTemporaryRedirect)
}

// Shorten принимает JSON-запрос [ShortURLRequest] с длинным URL-адресом,
// генерирует короткую ссылку и возвращает JSON-ответ [ShortURLResponse].
//
// После успешного создания ссылки метод отправляет событие сокращения в [AuditService].
//
// Возвращаемые HTTP-статусы:
//   - 201 Created: Ссылка успешно сокращена и сохранена.
//   - 409 Conflict: Переданный URL уже был сокращен ранее. Метод возвращает существующий код [service.ErrOriginalURLConflict].
//   - 400 Bad Request: Некорректный формат JSON или передан невалидный веб-адрес [service.ErrIncorrectURL].
//   - 499 Client Closed Request: Клиент прервал соединение (контекст отменен).
//   - 500 Internal Server Error: Превышен лимит коллизий [service.ErrShortCodeCollisionLimitExceeded] или произошел сбой БД.
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	var (
		request  ShortURLRequest
		response ShortURLResponse
	)

	uid, ok := h.getUserID(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	body := http.MaxBytesReader(
		w,
		r.Body,
		maxBodySize,
	)

	if err := json.NewDecoder(body).Decode(&request); err != nil {
		h.log.Info(
			"failed to decode json body",
			"err", err,
		)

		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	ShortedURL, err := h.service.CreateShortURL(r.Context(), request.URL, uid)

	status := http.StatusCreated

	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			h.log.Debug("create short url canceled by client")
			w.WriteHeader(499)
			return
		case errors.Is(err, service.ErrShortCodeCollisionLimitExceeded):
			h.log.Warn(
				"short code collision limit exceeded",
				"url", request.URL,
			)

			http.Error(w, "failed to process request, please try again", http.StatusInternalServerError)
			return
		case errors.Is(err, service.ErrIncorrectURL):
			h.log.Info(
				"incorrect url request",
				"err", err,
				"url", request.URL,
			)

			http.Error(w, "incorrect URL", http.StatusBadRequest)
			return
		case errors.Is(err, service.ErrOriginalURLConflict):
			status = http.StatusConflict
		default:
			h.log.Error(
				"unexpected error during url shortening",
				"err", err,
				"url", request.URL,
			)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	response.Result = ShortedURL

	data, err := json.Marshal(response)
	if err != nil {
		h.log.Error(
			"failed to marshal response json",
			"err", err,
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.auditService.Shorten(r.Context(), uid, request.URL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// Create принимает длинный URL-адрес в виде сырой строки (plain text) в теле запроса,
// генерирует короткую ссылку и возвращает результат текстовой строкой с типом "text/plain".
//
// Чтение тела запроса принудительно ограничивается размером в 1 МБ с помощью [io.LimitReader].
// После успешного создания ссылки метод отправляет событие сокращения в [AuditService].
//
// Возвращаемые HTTP-статусы:
//   - 201 Created: Ссылка успешно сокращена и сохранена.
//   - 409 Conflict: Переданный URL уже существует в системе. Метод возвращает существующий код [service.ErrOriginalURLConflict].
//   - 400 Bad Request: Ошибка чтения тела или передан невалидный веб-адрес [service.ErrIncorrectURL].
//   - 499 Client Closed Request: Клиент разорвал соединение до завершения генерации.
//   - 500 Internal Server Error: Внутренний сбой генератора или репозитория.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {

	uid, ok := h.getUserID(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		h.log.Info(
			"failed to decode json body",
			"err", err,
		)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	url := string(data)

	ShortedURL, err := h.service.CreateShortURL(r.Context(), url, uid)

	status := http.StatusCreated

	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			h.log.Debug("create short url canceled by client")
			w.WriteHeader(499)
			return
		case errors.Is(err, service.ErrShortCodeCollisionLimitExceeded):
			h.log.Warn(
				"short code collision limit exceeded",
				"url", url,
			)

			http.Error(w, "failed to process request, please try again", http.StatusInternalServerError)
			return
		case errors.Is(err, service.ErrIncorrectURL):
			h.log.Info(
				"incorrect url request",
				"err", err,
				"url", url,
			)

			http.Error(w, "incorrect URL", http.StatusBadRequest)
			return
		case errors.Is(err, service.ErrOriginalURLConflict):
			status = http.StatusConflict
		default:
			h.log.Error(
				"unexpected error during url shortening",
				"err", err,
				"url", url,
			)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	h.auditService.Shorten(r.Context(), uid, url)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(ShortedURL))
}

func (h *Handler) getUserID(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(ctxkeys.UserIDKey).(string)
	if !ok {
		h.log.Error(
			"failed to get user ID from context",
			"actual_type", fmt.Sprintf("%T", ctx.Value(ctxkeys.UserIDKey)),
		)
	}
	return uid, ok
}

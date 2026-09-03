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
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLShortener interface {
	CreateShortURL(ctx context.Context, url, uid string) (string, error)
	CreateURLBatch(ctx context.Context, req []service.CreateURLBatchRequest, uid string) ([]service.CreateURLBatchResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	GetUserURLs(ctx context.Context, uid string) ([]service.GetURLsResponse, error)
}

type Handler struct {
	service URLShortener
	log     *slog.Logger
	pool    *pgxpool.Pool
}

const maxBodySize = 1024 * 1024

func New(service URLShortener, log *slog.Logger, pool *pgxpool.Pool) *Handler {
	return &Handler{
		service: service,
		log:     log,
		pool:    pool,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/", h.Create)
	r.Post("/api/shorten", h.Shorten)
	r.Post("/api/shorten/batch", h.ShortenBatch)
	r.Get("/api/user/urls", h.GetURLs)
	r.Get("/{id}", h.Redirect)
	r.Get("/ping", h.Ping)
}

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

	response := make([]GetURLsResponse, len(result))

	for i, u := range result {
		response[i].ShortURL = u.ShortURL
		response[i].OriginalURL = u.OriginalURL
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
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

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
		} else if errors.Is(err, service.ErrIncorrectUrl) {

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

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {

	shortCode := chi.URLParam(r, "id")
	if shortCode == "" {
		http.Error(w, "URL Param is empty", http.StatusBadRequest)
		return
	}

	OriginalURL, err := h.service.GetOriginalURL(r.Context(), shortCode)
	if err != nil {
		if errors.Is(err, service.ErrUrlDoesNotExists) {

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
		} else {
			h.log.Error(
				"unexpected error during redirect",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, OriginalURL, http.StatusTemporaryRedirect)

}

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
		case errors.Is(err, service.ErrIncorrectUrl):
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

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
		case errors.Is(err, service.ErrIncorrectUrl):
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

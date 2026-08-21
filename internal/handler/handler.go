package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

type UrlShortener interface {
	CreateShortUrl(ctx context.Context, url string) (string, error)
	CreateUrlBatch(ctx context.Context, req []service.CreateUrlBatchRequest) ([]service.CreateUrlBatchResponse, error)
	GetOriginalUrl(ctx context.Context, shortCode string) (string, error)
}

type Handler struct {
	service UrlShortener
	log     *slog.Logger
	db      DBHealthChecker
}

type DBHealthChecker interface {
	Ping(ctx context.Context) error
}

const maxBodySize = 1024 * 1024

func New(service UrlShortener, log *slog.Logger, db DBHealthChecker) *Handler {
	return &Handler{
		service: service,
		log:     log,
		db:      db,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/", h.Create)
	r.Post("/api/shorten", h.Shorten)
	r.Post("/api/shorten/batch", h.ShortenBatch)
	r.Get("/{id}", h.Redirect)
	r.Get("/ping", h.Ping)
}

func (h *Handler) ShortenBatch(w http.ResponseWriter, r *http.Request) {

	var request []CreateUrlBatchRequest

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

	serviceReq := make([]service.CreateUrlBatchRequest, len(request))

	for i, item := range request {
		serviceReq[i] = service.CreateUrlBatchRequest{
			CorrelationID: item.CorrelationID,
			OriginalURL:   item.OriginalURL,
		}
	}

	resp, err := h.service.CreateUrlBatch(r.Context(), serviceReq)
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

	data, err := json.Marshal(resp)
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

	if h.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.db.Ping(r.Context()); err != nil {
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

	originalUrl, err := h.service.GetOriginalUrl(r.Context(), shortCode)
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

	http.Redirect(w, r, originalUrl, http.StatusTemporaryRedirect)

}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	var (
		request  ShortUrlRequest
		response ShortUrlResponse
	)

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

	shortedUrl, err := h.service.CreateShortUrl(r.Context(), request.URL)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.log.Debug("create short url canceled by client")
			w.WriteHeader(499)
		} else if errors.Is(err, service.ErrShortCodeCollisionLimitExceeded) {

			h.log.Warn(
				"short code collision limit exceeded",
				"url", request.URL,
			)

			http.Error(w, "failed to process request, please try again", http.StatusInternalServerError)
		} else if errors.Is(err, service.ErrIncorrectUrl) {

			h.log.Info(
				"incorrect url request",
				"err", err,
				"url", request.URL,
			)

			http.Error(w, "incorrect URL", http.StatusBadRequest)
		} else {

			h.log.Error(
				"unexpected error during url shortening",
				"err", err,
				"url", request.URL,
			)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	response.Result = shortedUrl

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {

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

	shortedUrl, err := h.service.CreateShortUrl(r.Context(), url)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.log.Debug("create short url canceled by client")
		} else if errors.Is(err, service.ErrShortCodeCollisionLimitExceeded) {
			h.log.Warn(
				"short code collision limit exceeded",
				"url", url,
			)
			http.Error(w, "failed to process request, please try again", http.StatusInternalServerError)
		} else if errors.Is(err, service.ErrIncorrectUrl) {
			http.Error(w, "incorrect URL", http.StatusBadRequest)
		} else {
			h.log.Error(
				"unexpected error during url shortening",
				"err", err,
				"url", url,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortedUrl))
}

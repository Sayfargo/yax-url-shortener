package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

type UrlShortener interface {
	CreateShortUrl(ctx context.Context, url string) (string, error)
	GetOriginalUrl(ctx context.Context, shortCode string) (string, error)
}

type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Handler struct {
	service UrlShortener

	log Logger
}

func New(service UrlShortener, log Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/", h.Create)
	r.Post("/api/shorten", h.Shorten)
	r.Get("/{id}", h.Redirect)
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

			http.Error(w, "internal server error", http.StatusInternalServerError)
		} else {
			h.log.Error(
				"unexpected error during redirect",
				"err", err,
			)
			http.Error(w, "internal server error", http.StatusInternalServerError)
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

	if err := json.NewDecoder(io.LimitReader(r.Body, 1024*1024)).Decode(&request); err != nil {

		h.log.Info(
			"failed to decode json body",
			"err", err,
		)

		http.Error(w, "internal server error", http.StatusBadRequest)
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
				"url", request.URL,
			)

			http.Error(w, "incorrect URL", http.StatusBadRequest)
		} else {

			h.log.Error(
				"unexpected error during url shortening",
				"err", err,
				"url", request.URL,
			)

			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	response.Result = shortedUrl

	body, err := json.Marshal(response)
	if err != nil {

		h.log.Error(
			"failed to marshal response json",
			"err", err,
		)

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(body)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortedUrl))
}

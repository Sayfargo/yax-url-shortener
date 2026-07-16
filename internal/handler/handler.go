package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type UrlShortener interface {
	CreateShortUrl(ctx context.Context, url string) (string, error)
	GetOriginalUrl(ctx context.Context, shortCode string) (string, error)
}

type Handler struct {
	service UrlShortener
}

func New(service UrlShortener) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/", h.Create)
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
			http.NotFound(w, r)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, originalUrl, http.StatusTemporaryRedirect)

}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {

	validate := validator.New()

	data, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	url := string(data)
	url = strings.TrimSpace(url)
	if url == "" {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	if err := validate.Var(url, "http_url"); err != nil {
		http.Error(w, "Incorrect URL", http.StatusBadRequest)
		return
	}

	shortedUrl, err := h.service.CreateShortUrl(r.Context(), url)
	if err != nil {
		// todo
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortedUrl))

}

package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

/*
	TODO:
		В будущем заменить http.Error ответ на свой кастомный тип ошибок
		И возвращать ошибки в Json формате

*/

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

	data, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	url := string(data)

	shortedUrl, err := h.service.CreateShortUrl(r.Context(), url)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Printf("request canceled: %v", err)
		} else if errors.Is(err, service.ErrShortCodeCollisionLimitExceeded) {
			http.Error(w, "Failed to process request, please try again", http.StatusInternalServerError)
		} else if errors.Is(err, service.ErrIncorrectUrl) {
			http.Error(w, "Incorrect URL", http.StatusBadRequest)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortedUrl))
}

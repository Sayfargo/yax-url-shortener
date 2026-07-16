package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/repository"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type URLRepository interface {
	Create(ctx context.Context, url, shortCode string) error
	Get(ctx context.Context, shortCode string) (string, error)
}

type UrlShortenerService struct {
	repo URLRepository
}

func New(repo URLRepository) *UrlShortenerService {
	return &UrlShortenerService{
		repo: repo,
	}
}

var (
	ErrUrlDoesNotExists = errors.New("requested URL code does not exists")
)

// TODO: Move to env
const (
	domain = "localhost"
	port   = "8080"
)

func (s *UrlShortenerService) GetOriginalUrl(ctx context.Context, shortCode string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	originalUrl, err := s.repo.Get(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotExists) {
			return "", ErrUrlDoesNotExists
		}
		return "", fmt.Errorf("repository get err: %w", err)
	}

	return originalUrl, nil

}

func (s *UrlShortenerService) CreateShortUrl(ctx context.Context, url string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	const (
		alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	)

	var (
		shortCode string
		err       error
	)
	// if conflict in DB
	for attempt := 0; attempt < 3; attempt++ {

		shortCode, err = gonanoid.Generate(alphabet, 8)
		if err != nil {
			return "", fmt.Errorf("gonanoid generate err: %w", err)
		}

		if err := s.repo.Create(ctx, url, shortCode); err != nil {
			/*
				TODO:
					If error is conflict do retry or return error

			*/
			continue
		} else {
			break
		}
	}

	shortedUrl := s.buildShortedUrl(shortCode)

	return shortedUrl, nil

}

func (s *UrlShortenerService) buildShortedUrl(shorCode string) string {
	return fmt.Sprintf("http://%s:%s/%s", domain, port, shorCode) // Example: http://localhost:8080/EiXk21Dz
}

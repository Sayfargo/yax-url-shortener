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

// Возможно лишнее, стоит подумать
type Generator interface {
	Generate(alphabet string, size int) (string, error)
}

type GoNanoIDGenerator struct{}

type UrlShortenerService struct {
	repo      URLRepository
	generator Generator
}

func New(repo URLRepository, generator Generator) *UrlShortenerService {
	return &UrlShortenerService{
		repo:      repo,
		generator: generator,
	}
}

var (
	ErrUrlDoesNotExists = errors.New("requested URL code does not exists")
)

// TODO: Убрать в .env
const (
	domain = "localhost"
	port   = "8080"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	size     = 8
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

	var (
		shortCode string
		err       error
	)

	for attempt := 0; attempt < 3; attempt++ {

		shortCode, err = s.generator.Generate(alphabet, size)
		if err != nil {
			return "", fmt.Errorf("gonanoid generate err: %w", err)
		}

		if err := s.repo.Create(ctx, url, shortCode); err != nil {
			/*
				TODO:
					Если реально будет конфликт с шорт кодом, попробуем ретрайнуть
					Здес будет обработка ошибки от БД, что шорт код уже существует
					<ErrAlreadyExists>
			*/
			continue
		} else {
			break
		}
	}

	shortedUrl := s.buildShortedUrl(shortCode)

	return shortedUrl, nil

}

func (n *GoNanoIDGenerator) Generate(alphabet string, size int) (string, error) {
	return gonanoid.Generate(alphabet, size)
}

func (s *UrlShortenerService) buildShortedUrl(shorCode string) string {
	return fmt.Sprintf("http://%s:%s/%s", domain, port, shorCode) // Example: http://localhost:8080/EiXk21Dz
}

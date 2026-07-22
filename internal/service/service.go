package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/repository"
	"github.com/go-playground/validator/v10"
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
	repo URLRepository

	generator Generator
	validate  *validator.Validate

	// Base URL для генерации шорт юрла
	baseURL string
}

func New(
	repo URLRepository,
	generator Generator,
	baseURL string,
	validate *validator.Validate,
) *UrlShortenerService {
	return &UrlShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validate,
		baseURL:   baseURL,
	}
}

var (
	ErrUrlDoesNotExists                = errors.New("requested URL code does not exists")
	ErrShortCodeCollisionLimitExceeded = errors.New("failed to generate short code after all attempts")
	ErrIncorrectUrl                    = errors.New("incorrect url")
)

// TODO: Убрать в .env

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
		return "", fmt.Errorf("Repository.Get err: %w", err)
	}

	return originalUrl, nil

}

func (s *UrlShortenerService) CreateShortUrl(ctx context.Context, url string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if err := s.validate.Var(url, "http_url"); err != nil {
		return "", ErrIncorrectUrl
	}

	for attempt := 0; attempt < 3; attempt++ {

		shortCode, err := s.generator.Generate(alphabet, size)
		if err != nil {
			return "", fmt.Errorf("Generator.Generate err: %w", err)
		}

		err = s.repo.Create(ctx, url, shortCode)
		if err == nil {
			// Успешно положили в БД юрл и вернули юзеру короткую ссылку
			return s.buildShortedUrl(shortCode), nil
		}

		// Если ошибка и она означает конфликт, пробуем ретрай
		if errors.Is(err, repository.ErrAlreadyExists) {
			continue
		}

		// Если ошибка и она не означает конфликт или что-то другое, обарачиваем и возварщаем в handler
		return "", fmt.Errorf("Repository.Create err: %w", err)
	}

	// На очень невероятный кейс если на каждую попытку пришёлся конфликт
	return "", ErrShortCodeCollisionLimitExceeded
}

func (n *GoNanoIDGenerator) Generate(alphabet string, size int) (string, error) {
	return gonanoid.Generate(alphabet, size)
}

func (s *UrlShortenerService) buildShortedUrl(shorCode string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shorCode) // Example: http://localhost:8080/EiXk21Dz
}

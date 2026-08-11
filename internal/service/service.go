package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	repository_cache "github.com/Sayfargo/yax-url-shortener/internal/repository/cache"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type CacheRepository interface {
	Create(ctx context.Context, url, shortCode string) error
	Get(ctx context.Context, shortCode string) (string, error)
}

type FileRepository interface {
	Save(ctx context.Context, shortenedUrl model.ShortenedUrl) error
}

type Generator interface {
	Generate(alphabet string, size int) (string, error)
}

type GoNanoIDGenerator struct{}

type UrlShortenerService struct {
	// Хранит в IMC
	cacheRepo CacheRepository
	// Хранит в файле
	fileRepo FileRepository

	generator Generator
	validate  *validator.Validate

	// Base URL для генерации шорт юрла
	baseURL string
}

func New(
	cacheRepo CacheRepository,
	fileRepo FileRepository,
	generator Generator,
	baseURL string,
	validate *validator.Validate,
) *UrlShortenerService {
	return &UrlShortenerService{
		cacheRepo: cacheRepo,
		fileRepo:  fileRepo,
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

	originalUrl, err := s.cacheRepo.Get(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repository_cache.ErrNotExists) {
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
			return "", fmt.Errorf("Generator.Generate: %w", err)
		}

		err = s.cacheRepo.Create(ctx, url, shortCode)

		switch {
		case err == nil:
			if err := s.saveShortenedURL(ctx, url, shortCode); err != nil {
				return "", fmt.Errorf("FileStorage.Save: %w", err)
			}

			return s.buildShortedUrl(shortCode), nil

		case errors.Is(err, repository_cache.ErrAlreadyExists):
			continue

		default:
			return "", fmt.Errorf("Repository.Create: %w", err)
		}
	}

	return "", ErrShortCodeCollisionLimitExceeded
}

func (s *UrlShortenerService) saveShortenedURL(ctx context.Context, originalUrl, shortCode string) error {
	uuid, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("failed to generate uuid: %w", err)
	}

	shortenedUrl := model.ShortenedUrl{
		UUID:        uuid.String(),
		ShortCode:   shortCode,
		OriginalUrl: originalUrl,
	}

	return s.fileRepo.Save(ctx, shortenedUrl)

}

func (n *GoNanoIDGenerator) Generate(alphabet string, size int) (string, error) {
	return gonanoid.Generate(alphabet, size)
}

func (s *UrlShortenerService) buildShortedUrl(shorCode string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shorCode) // Example: http://localhost:8080/EiXk21Dz
}

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

type Repository interface {
	Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error
	Get(ctx context.Context, shortCode string) (string, error)
}

type Generator interface {
	Generate(alphabet string, size int) (string, error)
}

type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type GoNanoIDGenerator struct{}

type UrlShortenerService struct {
	// Хранит в IMC
	cacheRepo Repository

	generator Generator
	log       Logger
	validate  *validator.Validate

	// Base URL для генерации шорт юрла
	baseURL string
}

func New(
	cacheRepo Repository,
	generator Generator,
	baseURL string,
	validate *validator.Validate,
	log Logger,
) *UrlShortenerService {
	return &UrlShortenerService{
		cacheRepo: cacheRepo,
		generator: generator,
		validate:  validate,
		baseURL:   baseURL,
		log:       log,
	}
}

var (
	ErrUrlDoesNotExists                = errors.New("requested URL code does not exists")
	ErrShortCodeCollisionLimitExceeded = errors.New("failed to generate short code after all attempts")
	ErrIncorrectUrl                    = errors.New("incorrect url")
	ErrCorruptedData                   = errors.New("cached data is corrupted or invalid")
)

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
		} else if errors.Is(err, repository_cache.ErrUnexpectedType) {
			return "", ErrCorruptedData
		}
		return "", fmt.Errorf("repository get err: %w", err)
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
			return "", fmt.Errorf("generator generate: %w", err)
		}

		shortenedUrl := model.ShortenedUrl{
			UUID:        uuid.New().String(),
			ShortCode:   shortCode,
			OriginalUrl: url,
		}

		err = s.cacheRepo.Create(ctx, shortenedUrl)

		switch {
		case err == nil:

			return s.buildShortedUrl(shortCode), nil

		case errors.Is(err, repository_cache.ErrAlreadyExists):

			s.log.Info(
				"short code collision occurred, retrying",
				"url", url,
				"code", shortCode,
				"attempt", attempt+1,
			)

			continue

		default:

			return "", fmt.Errorf("repository create: %w", err)
		}
	}

	return "", ErrShortCodeCollisionLimitExceeded
}

func (n *GoNanoIDGenerator) Generate(alphabet string, size int) (string, error) {
	return gonanoid.Generate(alphabet, size)
}

func (s *UrlShortenerService) buildShortedUrl(shorCode string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shorCode) // Example: http://localhost:8080/EiXk21Dz
}

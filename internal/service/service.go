package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	urlrepo "github.com/Sayfargo/yax-url-shortener/internal/repository/url"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type URLRepository interface {
	Create(ctx context.Context, ShortenedURL model.ShortenedURL) error
	CreateBatch(ctx context.Context, ShortenedURLs []model.ShortenedURL) error
	Get(ctx context.Context, shortCode string) (string, error)
	GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error)
}

type Generator interface {
	Generate(alphabet string, size int) (string, error)
}

type GoNanoIDGenerator struct{}

type URLShortenerService struct {
	// Хранит в IMC
	repo URLRepository

	generator Generator
	log       *slog.Logger
	validate  *validator.Validate

	// Base URL для генерации шорт юрла
	baseURL string
}

func New(
	repo URLRepository,
	generator Generator,
	baseURL string,
	validate *validator.Validate,
	log *slog.Logger,
) *URLShortenerService {
	return &URLShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validate,
		baseURL:   baseURL,
		log:       log,
	}
}

var (
	ErrURLDoesNotExists                = errors.New("requested URL code does not exists")
	ErrShortCodeCollisionLimitExceeded = errors.New("failed to generate short code after all attempts")
	ErrIncorrectURL                    = errors.New("incorrect url")
	ErrCorruptedData                   = errors.New("cached data is corrupted or invalid")
	ErrEmptyBatch                      = errors.New("empty batch")
	ErrOriginalURLConflict             = errors.New("original url conflict")
	ErrURLsNotFound                    = errors.New("no URLs found for user")
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	size     = 8
)

type CreateURLBatchRequest struct {
	CorrelationID string
	OriginalURL   string
}

type CreateURLBatchResponse struct {
	CorrelationID string
	ShortURL      string
}

type GetURLsResponse struct {
	ShortURL    string
	OriginalURL string
}

func (s *URLShortenerService) GetUserURLs(ctx context.Context, uid string) ([]GetURLsResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return nil, fmt.Errorf("uuid from bytes: %w", err)
	}

	result, err := s.repo.GetURLs(ctx, uidUUID)
	if err != nil {
		if errors.Is(err, urlrepo.ErrNoRows) {
			return nil, ErrURLsNotFound
		}
		return nil, fmt.Errorf("repository get URLs: %w", err)
	}

	response := make([]GetURLsResponse, len(result))

	for i, u := range result {
		response[i].ShortURL = s.buildShortedURL(u.ShortCode)
		response[i].OriginalURL = u.OriginalURL
	}

	return response, nil
}

func (s *URLShortenerService) CreateURLBatch(ctx context.Context, req []CreateURLBatchRequest, uid string) ([]CreateURLBatchResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if len(req) == 0 {
		return nil, ErrEmptyBatch
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return nil, fmt.Errorf("uuid from bytes: %w", err)
	}

	ShortenedURLs := make([]model.ShortenedURL, len(req))
	resp := make([]CreateURLBatchResponse, len(req))

	for i, item := range req {
		if err := s.validate.Var(item.OriginalURL, "http_url"); err != nil {
			return nil, fmt.Errorf(
				"got incorrect url on row #%d with id %s: %w",
				i+1, item.CorrelationID,
				ErrIncorrectURL,
			)
		}

		shortCode, err := s.generator.Generate(alphabet, size)
		if err != nil {
			return nil, fmt.Errorf("generator generate: %w", err)

		}

		ShortenedURLs[i] = model.ShortenedURL{
			UUID:        uuid.New(),
			ShortCode:   shortCode,
			OriginalURL: item.OriginalURL,
			UserID:      uidUUID,
		}

		resp[i] = CreateURLBatchResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.buildShortedURL(shortCode),
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		err := s.repo.CreateBatch(ctx, ShortenedURLs)

		var batchErr *urlrepo.BatchConflictError

		switch {
		case err == nil:
			return resp, nil
		case errors.As(err, &batchErr):

			if attempt == 2 {
				break
			}

			s.log.Info(
				"short code collision occurred, retrying",
				"url", ShortenedURLs[batchErr.Index].OriginalURL,
				"code", ShortenedURLs[batchErr.Index].ShortCode,
				"attempt", attempt+1,
			)

			shortCode, err := s.generator.Generate(alphabet, size)
			if err != nil {
				return nil, fmt.Errorf("generator generate: %w", err)
			}

			ShortenedURLs[batchErr.Index].ShortCode = shortCode
			resp[batchErr.Index].ShortURL = s.buildShortedURL(shortCode)

			continue
		default:
			return nil, fmt.Errorf("repository create: %w", err)
		}
	}

	return nil, ErrShortCodeCollisionLimitExceeded
}

func (s *URLShortenerService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	OriginalURL, err := s.repo.Get(ctx, shortCode)
	if err != nil {
		if errors.Is(err, urlrepo.ErrNotExists) {
			return "", ErrURLDoesNotExists
		} else if errors.Is(err, urlrepo.ErrUnexpectedType) {
			return "", ErrCorruptedData
		}
		return "", fmt.Errorf("repository get err: %w", err)
	}

	return OriginalURL, nil

}

func (s *URLShortenerService) CreateShortURL(ctx context.Context, url, uid string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	uidUUID, err := uuid.Parse(uid)
	if err != nil {
		return "", fmt.Errorf("uuid from bytes: %w", err)
	}

	if err := s.validate.Var(url, "http_url"); err != nil {
		return "", ErrIncorrectURL
	}

	for attempt := 0; attempt < 3; attempt++ {

		shortCode, err := s.generator.Generate(alphabet, size)
		if err != nil {
			return "", fmt.Errorf("generator generate: %w", err)
		}

		ShortenedURL := model.ShortenedURL{
			UUID:        uuid.New(),
			ShortCode:   shortCode,
			OriginalURL: url,
			UserID:      uidUUID,
		}

		err = s.repo.Create(ctx, ShortenedURL)

		var origURLConflictErr *urlrepo.OriginalURLConflictError

		switch {
		case err == nil:

			return s.buildShortedURL(shortCode), nil

		case errors.Is(err, urlrepo.ErrConflictShortCode):

			s.log.Info(
				"short code collision occurred, retrying",
				"url", url,
				"code", shortCode,
				"attempt", attempt+1,
			)
			continue
		case errors.As(err, &origURLConflictErr):

			return s.buildShortedURL(origURLConflictErr.ShortCode), ErrOriginalURLConflict

		default:

			return "", fmt.Errorf("repository create: %w", err)
		}
	}

	return "", ErrShortCodeCollisionLimitExceeded
}

func (n *GoNanoIDGenerator) Generate(alphabet string, size int) (string, error) {
	return gonanoid.Generate(alphabet, size)
}

func (s *URLShortenerService) buildShortedURL(shortCode string) string {
	return s.baseURL + "/" + shortCode // Example: http://localhost:8080/EiXk21Dz
}

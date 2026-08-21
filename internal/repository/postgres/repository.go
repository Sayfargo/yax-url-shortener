package repository_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	repository_errors "github.com/Sayfargo/yax-url-shortener/internal/repository/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

func (pr *PostgresRepository) GetByOriginalURL(ctx context.Context, url string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	query := `SELECT short_code FROM shortened_urls WHERE original_url = $1`

	var shortCode string

	if err := pr.pool.QueryRow(ctx, query, url).Scan(&shortCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", repository_errors.ErrNotExists
		}
		return "", fmt.Errorf("query row: %w", err)
	}

	return shortCode, nil
}

func (pr *PostgresRepository) Get(ctx context.Context, shortCode string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	query := `SELECT original_url FROM shortened_urls WHERE short_code = $1`

	var origUrl string

	err := pr.pool.QueryRow(ctx, query, shortCode).Scan(&origUrl)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", repository_errors.ErrNotExists
		}
		return "", fmt.Errorf("query row: %w", err)
	}

	return origUrl, nil

}

func (pr *PostgresRepository) CreateBatch(ctx context.Context, shortenedUrls []model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	batch := new(pgx.Batch)

	query := `
		INSERT INTO shortened_urls (uuid, short_code, original_url)
		VALUES ($1, $2, $3)
	`
	for _, u := range shortenedUrls {
		batch.Queue(query, u.UUID, u.ShortCode, u.OriginalUrl)
	}

	br := pr.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := range len(shortenedUrls) {
		_, err := br.Exec()
		if err != nil {

			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {

				switch pgErr.ConstraintName {
				case "shortened_urls_short_code_key":
					return &repository_errors.BatchConflictError{
						Index: i,
						Err:   repository_errors.ErrConflictShortCode,
					}
				}
			}
			return fmt.Errorf("batch exec item: %d: %w", i, err)
		}
	}

	return nil
}

func (pr *PostgresRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	query := `INSERT INTO shortened_urls (uuid, short_code, original_url) VALUES ($1, $2, $3)`

	_, err := pr.pool.Exec(ctx, query, shortenedUrl.UUID, shortenedUrl.ShortCode, shortenedUrl.OriginalUrl)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			switch pgErr.ConstraintName {

			case "shortened_urls_short_code_key":
				return repository_errors.ErrConflictShortCode

			case "idx_shortened_urls_original_url":
				return &repository_errors.OriginalUrlConflictError{
					URL: shortenedUrl.OriginalUrl,
				}
			}
		}
		return fmt.Errorf("pool exec: %w", err)
	}

	return nil
}

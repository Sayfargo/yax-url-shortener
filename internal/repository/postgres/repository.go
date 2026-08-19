package repository_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	repository_errors "github.com/Sayfargo/yax-url-shortener/internal/repository/errors"
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

const pgxUniqueErr = "23505"

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

func (pr *PostgresRepository) Create(ctx context.Context, shortenedUrl model.ShortenedUrl) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	query := `INSERT INTO shortened_urls (uuid, short_code, original_url) VALUES ($1, $2, $3)`

	_, err := pr.pool.Exec(ctx, query, shortenedUrl.UUID, shortenedUrl.ShortCode, shortenedUrl.OriginalUrl)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgxUniqueErr {
				return repository_errors.ErrAlreadyExists
			}
		}
		return fmt.Errorf("pool exec: %w", err)
	}

	return nil
}

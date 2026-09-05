package url

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

func (pr *PostgresRepository) GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	query := `SELECT uuid, short_code, original_url, user_id FROM shortened_urls WHERE user_id = $1`

	urls := make([]model.ShortenedURL, 0, 32)

	rows, err := pr.pool.Query(ctx, query, uid)
	if err != nil {
		return nil, fmt.Errorf("pgx query: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var u model.ShortenedURL

		if err := rows.Scan(&u.UUID, &u.ShortCode, &u.OriginalURL, &u.UserID); err != nil {
			return nil, fmt.Errorf("pgx rows scan: %w", err)
		}

		urls = append(urls, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgx rows err: %w", err)
	}

	if len(urls) == 0 {
		return nil, ErrNoRows
	}

	return urls, nil
}

func (pr *PostgresRepository) Get(ctx context.Context, shortCode string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	query := `SELECT original_url FROM shortened_urls WHERE short_code = $1`

	var origURL string

	err := pr.pool.QueryRow(ctx, query, shortCode).Scan(&origURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotExists
		}
		return "", fmt.Errorf("query row: %w", err)
	}

	return origURL, nil

}

func (pr *PostgresRepository) CreateBatch(ctx context.Context, ShortenedURLs []model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	tx, err := pr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pool begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batch := new(pgx.Batch)
	query := `
		INSERT INTO shortened_urls (uuid, short_code, original_url, user_id)
		VALUES ($1, $2, $3, $4)
	`
	for _, u := range ShortenedURLs {
		batch.Queue(query, u.UUID, u.ShortCode, u.OriginalURL, u.UserID)
	}

	br := tx.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()

	for i := range len(ShortenedURLs) {
		_, err := br.Exec()
		if err != nil {

			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				if pgErr.ConstraintName == "shortened_urls_short_code_key" {
					return &BatchConflictError{
						Index: i,
						Err:   ErrConflictShortCode,
					}
				}
			}
			return fmt.Errorf("batch exec item: %d: %w", i, err)
		}
	}

	if err := br.Close(); err != nil {
		return fmt.Errorf("close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (pr *PostgresRepository) Create(ctx context.Context, ShortenedURL model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	query := `
		INSERT INTO shortened_urls (uuid, short_code, original_url, user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (original_url) DO UPDATE 
		SET original_url = EXCLUDED.original_url
		RETURNING short_code, xmax = 0;
	`

	var (
		shortCode string
		inserted  bool
	)

	err := pr.pool.QueryRow(
		ctx,
		query,
		ShortenedURL.UUID,
		ShortenedURL.ShortCode,
		ShortenedURL.OriginalURL,
		ShortenedURL.UserID,
	).Scan(&shortCode, &inserted)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == pgerrcode.UniqueViolation &&
			pgErr.ConstraintName == "shortened_urls_short_code_key" {

			return ErrConflictShortCode
		}

		return err
	}

	if !inserted {
		return &OriginalURLConflictError{
			ShortCode: shortCode,
		}
	}

	return nil
}

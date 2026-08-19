package core_db_postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	config_db_postgres "github.com/Sayfargo/yax-url-shortener/internal/config/db/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(cfg config_db_postgres.Config, log *slog.Logger) (*pgxpool.Pool, error) {

	log.Info(
		"connecting to database",
	)

	poolCfg, err := pgxpool.ParseConfig(cfg.DBDsn)
	if err != nil {

		log.Error(
			"failed to parse conn string",
			"err", err,
		)

		return nil, fmt.Errorf("parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Error(
			"failed to initialize conntection pool",
			"err", err,
		)

		return nil, fmt.Errorf("new with config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		log.Error(
			"failed to ping database",
			"err", err,
		)
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info(
		"database initialized successfully",
	)

	return pool, nil
}

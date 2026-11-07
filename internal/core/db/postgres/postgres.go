package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/Sayfargo/yax-url-shortener/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// New создает пул соединений [pgxpool.Pool] к PostgreSQL и накатывает миграции.
//
// Процесс инициализации:
// 1. Парсится переданная строка подключения DSN из структуры [Config].
// 2. Создается пул соединений pgxpool для основного пула приложения.
// 3. Открывается временное стандартное соединение [sql.DB] через драйвер pgx/stdlib
// для выполнения миграций базы данных.
// 4. Используя библиотеку [goose], накатываются встроенные миграции из пакета migrations.
//
// В случае сбоя на этапе миграций функция гарантирует закрытие уже созданного пула.
// Возвращает готовый пул или обернутую ошибку инициализации.
func New(cfg Config, log *slog.Logger) (*pgxpool.Pool, error) {

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

	log.Info(
		"running db migrations",
	)

	db, err := sql.Open("pgx", cfg.DBDsn)
	if err != nil {
		log.Error(
			"faild to open sql db",
			"err", err,
		)
		pool.Close()
		return nil, fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Error(
			"failed to set dialect",
			"err", err,
		)
		return nil, fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		pool.Close()
		return nil, fmt.Errorf("goose up: %w", err)
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// if err := pool.Ping(ctx); err != nil {
	// 	log.Error(
	// 		"failed to ping database",
	// 		"err", err,
	// 	)
	// 	pool.Close()
	// 	return nil, fmt.Errorf("ping database: %w", err)
	// }

	log.Info(
		"database initialized successfully",
	)

	return pool, nil
}

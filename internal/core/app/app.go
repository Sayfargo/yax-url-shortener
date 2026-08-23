package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Sayfargo/yax-url-shortener/internal/config"
	"github.com/Sayfargo/yax-url-shortener/internal/core/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/core/db/postgres"
	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/core/httpserver"
	"github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/middleware"
	"github.com/Sayfargo/yax-url-shortener/internal/handler"
	"github.com/Sayfargo/yax-url-shortener/internal/repository/url"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Server  *httpserver.HTTPServer
	DB      *pgxpool.Pool
	Storage *filestorage.FileStorage
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {

	// chi router/middlewares
	rootRouter := chi.NewRouter()
	rootRouter.Use(middleware.Logging(log))
	rootRouter.Use(middleware.GzipCompress())

	var (
		db          *pgxpool.Pool
		fileStorage *filestorage.FileStorage
		activeRepo  service.URLRepository
		err         error
	)
	/*
		Выбираем хранилище в зависимости от конфигурации или примененных флагов
		Если флаг или env для DB были заданы, то используем базу данных
		Если флаг или env для DB не были заданы, но были заданы для File storage - используем File storage
		Если флаг или env не были заданы ни для DB ни для File storage - используем in memory
	*/
	switch cfg.StorageType() {
	case config.StorageTypeDB:
		db, err = postgres.New(*cfg.DB, log)
		if err != nil {
			log.Error("failed to create db connection pool", "err", err)
			return nil, fmt.Errorf("database initialize: %w", err)
		}
		activeRepo = url.NewPgRepo(db)

	case config.StorageTypeFile:
		fileStorage, err = filestorage.Init(cfg.FileStorage)
		if err != nil {
			log.Error("failed to initialize file storage", "err", err)
			return nil, fmt.Errorf("file storage init: %w", err)
		}

		cacheStorage := cache.Init()
		cr := url.NewInMemoryRepo(cacheStorage)

		fcr, err := url.NewFileInMemoryRepository(cr, fileStorage)
		if err != nil {
			log.Error("failed to initialize file cache repository", "err", err)
			_ = fileStorage.Close()
			return nil, fmt.Errorf("restore cache: %w", err)
		}

		activeRepo = fcr

	case config.StorageTypeMemory:
		cacheStorage := cache.Init()
		activeRepo = url.NewInMemoryRepo(cacheStorage)
	}

	svc := service.New(activeRepo, new(service.GoNanoIDGenerator), cfg.Server.BaseURL, validator.New(), log)
	h := handler.New(svc, log, db)
	h.Register(rootRouter)

	httpServer := httpserver.New(rootRouter, cfg.Server, log)

	return &App{
		Server:  httpServer,
		DB:      db,
		Storage: fileStorage,
	}, nil

}

func (a *App) Run(ctx context.Context) (errs error) {

	defer func() {

		if a.Storage != nil {
			if err := a.Storage.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("file storage close: %w", err))
			}
		}

		if a.DB != nil {
			a.DB.Close()
		}
	}()

	if err := a.Server.Run(ctx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("server run: %w", err))
	}

	if errs != nil {
		return errs
	}

	return nil
}

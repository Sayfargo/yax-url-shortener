package core_app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Sayfargo/yax-url-shortener/internal/config"
	core_server "github.com/Sayfargo/yax-url-shortener/internal/core/server"
	core_storage_cache "github.com/Sayfargo/yax-url-shortener/internal/core/storage/cache"
	core_storage_file "github.com/Sayfargo/yax-url-shortener/internal/core/storage/file"
	core_transport_http_middleware "github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/middleware"
	"github.com/Sayfargo/yax-url-shortener/internal/handler"
	repository_cache "github.com/Sayfargo/yax-url-shortener/internal/repository/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type App struct {
	Server  *core_server.HTTPServer
	Storage *core_storage_file.FileStorage
}

func New(cfg *config.Config, slog *slog.Logger) (*App, error) {

	// chi router/middlewares
	rootRouter := chi.NewRouter()
	rootRouter.Use(core_transport_http_middleware.Logging(slog))
	rootRouter.Use(core_transport_http_middleware.GzipCompress())

	// storages
	cacheStorage := core_storage_cache.Init()
	fileStorage, err := core_storage_file.Init(cfg.FileStorage)
	if err != nil {

		slog.Error(
			"failed to initialize file storage",
			"err", err,
		)

		return nil, fmt.Errorf("file storage init: %w", err)
	}

	// repositories
	cr := repository_cache.New(cacheStorage)
	fcr, err := repository_cache.NewFileCacheRepository(cr, fileStorage)

	if err != nil {
		slog.Error(
			"failed to initialize file cache repository",
			"err", err,
		)

		_ = fileStorage.Close()

		return nil, fmt.Errorf("restore cache: %w", err)
	}

	svc := service.New(fcr, new(service.GoNanoIDGenerator), cfg.Server.BaseURL, validator.New(), slog)
	handler := handler.New(svc, slog)
	handler.Register(rootRouter)

	httpServer := core_server.New(rootRouter, cfg.Server, slog)

	return &App{
		Server:  httpServer,
		Storage: fileStorage,
	}, nil

}

func (a *App) Run(ctx context.Context) (errs error) {

	defer func() {

		if a.Storage == nil {
			return
		}

		if err := a.Storage.Close(); err != nil {
			errs = errors.Join(
				errs,
				fmt.Errorf("file storage close: %w", err),
			)
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

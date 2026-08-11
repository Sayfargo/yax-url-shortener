package core_app

import (
	"context"
	"os"

	"github.com/Sayfargo/yax-url-shortener/internal/config"
	config_slogger "github.com/Sayfargo/yax-url-shortener/internal/config/slogger"
	core_server "github.com/Sayfargo/yax-url-shortener/internal/core/server"
	core_slogger "github.com/Sayfargo/yax-url-shortener/internal/core/slogger"
	core_storage_cache "github.com/Sayfargo/yax-url-shortener/internal/core/storage/cache"
	core_storage_file "github.com/Sayfargo/yax-url-shortener/internal/core/storage/file"
	core_transport_http_middleware "github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/middleware"
	"github.com/Sayfargo/yax-url-shortener/internal/handler"
	repository_cache "github.com/Sayfargo/yax-url-shortener/internal/repository/cache"
	repository_file "github.com/Sayfargo/yax-url-shortener/internal/repository/file"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type App struct {
	// HTTP server
	Server  *core_server.HTTPServer
	Storage *core_storage_file.FileStorage
}

func New() *App {

	cfg := config.Load(os.Args[1:])
	rootRouter := chi.NewRouter()
	// Пока slogger временно без env
	slog, _ := core_slogger.MustNew(config_slogger.Config{
		Directory: "./log",
		Stdout:    config_slogger.StdoutConfig{Enabled: true, Format: config_slogger.FormatText, Writer: os.Stdout},
	})

	rootRouter.Use(core_transport_http_middleware.Logging(slog))
	rootRouter.Use(core_transport_http_middleware.GzipCompress())
	validate := validator.New()

	cacheStorage := core_storage_cache.Init()
	// Подумать как и где закрывать storage. Может быть написать closer для di
	fileStorage, err := core_storage_file.New(cfg.FileStorage)

	if err != nil {
		slog.Error(
			"Failed to init file storage",
			"error:", err,
		)
	}

	cacheRepo := repository_cache.New(cacheStorage)
	fileRepo := repository_file.New(fileStorage)

	svc := service.New(cacheRepo, fileRepo, new(service.GoNanoIDGenerator), cfg.Server.BaseURL, validate)
	handler := handler.New(svc)
	handler.Register(rootRouter)

	httpServer := core_server.New(rootRouter, cfg.Server)

	return &App{
		Server:  httpServer,
		Storage: fileStorage,
	}

}

func (a *App) Run(ctx context.Context) error {

	defer a.Storage.Close()

	if err := a.Server.Run(ctx); err != nil {
		return err
	}
	return nil
}

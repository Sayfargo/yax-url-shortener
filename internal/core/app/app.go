package core_app

import (
	"context"
	"os"

	"github.com/Sayfargo/yax-url-shortener/internal/config"
	core_server "github.com/Sayfargo/yax-url-shortener/internal/core/server"
	core_storage "github.com/Sayfargo/yax-url-shortener/internal/core/storage"
	"github.com/Sayfargo/yax-url-shortener/internal/handler"
	"github.com/Sayfargo/yax-url-shortener/internal/repository"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type App struct {
	// HTTP server
	Server *core_server.HTTPServer
}

func New() *App {

	/*
		TODO:
			Swagger
			Config (viper/envconfig) DB cfg, Server cfg
			PostgreSQL/Redis
			Versioned API router (api v1, api v2...)
			Middlewares (Logger, Recovery, Trace, RateLimit...)
			Tests
	*/

	cfg := config.Load(os.Args[1:])
	rootRouter := chi.NewRouter()
	validate := validator.New() // потом можно свои правила добавить

	cache := core_storage.Init()
	repo := repository.New(cache)

	svc := service.New(repo, new(service.GoNanoIDGenerator), cfg.Server.BaseURL, validate)
	handler := handler.New(svc)
	handler.Register(rootRouter)

	httpServer := core_server.New(rootRouter, cfg.Server)

	return &App{
		Server: httpServer,
	}

}

func (a *App) Run(ctx context.Context) error {
	if err := a.Server.Run(ctx); err != nil {
		return err
	}
	return nil
}

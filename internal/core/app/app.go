package core_app

import (
	"context"

	core_server "github.com/Sayfargo/yax-url-shortener/internal/core/server"
	core_storage "github.com/Sayfargo/yax-url-shortener/internal/core/storage"
	"github.com/Sayfargo/yax-url-shortener/internal/handler"
	"github.com/Sayfargo/yax-url-shortener/internal/repository"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
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

	rootRouter := chi.NewRouter()

	cache := core_storage.Init()
	repo := repository.New(cache)
	svc := service.New(repo)
	handler := handler.New(svc)
	handler.Register(rootRouter)

	httpServer := core_server.New(rootRouter)

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

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sayfargo/yax-url-shortener/internal/config"
	cfgslogger "github.com/Sayfargo/yax-url-shortener/internal/config/slogger"
	app "github.com/Sayfargo/yax-url-shortener/internal/core/app"
	slogger "github.com/Sayfargo/yax-url-shortener/internal/core/slogger"
)

func main() {
	sigCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// config
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Printf("config load: %v\n", err)
		return
	}

	// logger config
	logConfig := cfgslogger.Config{
		Directory: "./log",
		Stdout: cfgslogger.StdoutConfig{
			Enabled: true,
			Format:  cfgslogger.FormatText,
			Writer:  os.Stdout,
		},
	}

	// initialize logger
	slog, closer, err := slogger.New(logConfig)
	if err != nil {
		log.Printf("slogger: %v\n", err)
		return
	}

	defer func() {
		if err := closer.Close(); err != nil {
			log.Printf("logger close: %v\n", err)
		}
	}()

	// application
	app, err := app.New(cfg, slog)
	if err != nil {
		slog.Error(
			"failed to initialize application",
			"err", err,
		)
		return
	}

	if err := app.Run(sigCtx); err != nil {
		slog.Error(
			"failed to run server",
			"err", err,
		)
		return
	}
}

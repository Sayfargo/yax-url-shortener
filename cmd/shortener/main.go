package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sayfargo/yax-url-shortener/internal/config"
	"github.com/Sayfargo/yax-url-shortener/internal/core/app"
	"github.com/Sayfargo/yax-url-shortener/internal/core/slogger"
)

var (
	buildVersion string = notAvailable
	buildDate    string = notAvailable
	buildCommit  string = notAvailable
)

const (
	notAvailable = "N/A"
	msg          = "Build version: %s\nBuild date: %s\nBuild commit: %s\n"
)

func printBuildInfo() {
	fmt.Fprintf(os.Stdout, msg, buildVersion, buildDate, buildCommit)
}

func main() {
	sigCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	printBuildInfo()

	// config
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Printf("config load: %v\n", err)
		return
	}

	// logger config
	logConfig := slogger.Config{
		Directory: "./log",
		Stdout: slogger.StdoutConfig{
			Enabled: true,
			Format:  slogger.FormatText,
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

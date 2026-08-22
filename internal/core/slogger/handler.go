package slogger

import (
	"log/slog"
	"os"

	cfgslogger "github.com/Sayfargo/yax-url-shortener/internal/config/slogger"
)

func newFileHandler(cfg cfgslogger.FileConfig, file *os.File) slog.Handler {
	switch cfg.Format {
	case cfgslogger.FormatJSON:
		return slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	case cfgslogger.FormatText:
		return slog.NewTextHandler(file, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	return slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: cfg.Level,
	})
}

func newStdoutHandler(cfg cfgslogger.StdoutConfig) slog.Handler {

	writer := cfg.Writer

	if writer == nil {
		writer = os.Stdout
	}

	switch cfg.Format {
	case cfgslogger.FormatJSON:
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	case cfgslogger.FormatText:
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: cfg.Level,
	})
}

package core_slogger

import (
	"log/slog"
	"os"

	config_slogger "github.com/Sayfargo/yax-url-shortener/internal/config/slogger"
)

func newFileHandler(cfg config_slogger.FileConfig, file *os.File) slog.Handler {
	switch cfg.Format {
	case config_slogger.FormatJSON:
		return slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	case config_slogger.FormatText:
		return slog.NewTextHandler(file, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	return slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: cfg.Level,
	})
}

func newStdoutHandler(cfg config_slogger.StdoutConfig) slog.Handler {

	writer := cfg.Writer

	if writer == nil {
		writer = os.Stdout
	}

	switch cfg.Format {
	case config_slogger.FormatJSON:
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	case config_slogger.FormatText:
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: cfg.Level,
	})
}

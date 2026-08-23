package slogger

import (
	"log/slog"
	"os"
)

func newFileHandler(cfg FileConfig, file *os.File) slog.Handler {
	switch cfg.Format {
	case FormatJSON:
		return slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	case FormatText:
		return slog.NewTextHandler(file, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	return slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: cfg.Level,
	})
}

func newStdoutHandler(cfg StdoutConfig) slog.Handler {

	writer := cfg.Writer

	if writer == nil {
		writer = os.Stdout
	}

	switch cfg.Format {
	case FormatJSON:
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	case FormatText:
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: cfg.Level,
	})
}

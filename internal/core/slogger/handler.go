package core_slogger

import (
	"context"
	"log/slog"
	"os"

	config_slogger "github.com/Sayfargo/yax-url-shortener/internal/config/slogger"
)

type ContextHandler struct {
	slog.Handler
}

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

// TODO: На потом для удобства извлечения request id из контекста. Пока в коде не используется

func NewContextHandler(handler slog.Handler) slog.Handler {
	return &ContextHandler{
		Handler: handler,
	}
}

func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	requestID, ok := ctx.Value(0).(string)

	if ok {
		record.AddAttrs(slog.String("Request_ID", requestID))
	}

	return h.Handler.Handle(ctx, record)
}

func (h *ContextHandler) WithGroup(
	name string,
) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithGroup(name),
	}
}

func (h *ContextHandler) WithAttrs(
	attrs []slog.Attr,
) slog.Handler {

	return &ContextHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

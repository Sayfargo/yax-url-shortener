package core_server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type HTTPServer struct {
	server *http.Server
}

// Добавить конфигурацию для более глубокой настройки http сервера
func New(handler http.Handler) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:    ":8080",
			Handler: handler,
		},
	}
}

func (s *HTTPServer) Run(ctx context.Context) error {

	chErr := make(chan error, 1)

	l := slog.With(
		slog.String("Addr", s.server.Addr),
	)

	go func() {

		l.Info("starting http server")

		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			chErr <- err
		}

		close(chErr)
	}()

	select {
	case err := <-chErr:
		l.Error(
			"failed to start server",
			"err",
			err,
		)
		return fmt.Errorf("listen and server: %w", err)
	case <-ctx.Done():
		l.Info(
			"server starting shutdown",
		)

		if err := s.shutdown(); err != nil {
			l.Error(
				"failed to shutdown",
				"err",
				err,
			)
			_ = s.server.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
	}

	l.Info("server successfully closed")

	return nil
}

func (s *HTTPServer) shutdown() error {

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return nil

}

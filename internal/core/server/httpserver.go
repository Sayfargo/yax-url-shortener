package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	server "github.com/Sayfargo/yax-url-shortener/internal/config/server"
)

type HTTPServer struct {
	server *http.Server

	log *slog.Logger
}

func New(handler http.Handler, cfg *server.Config, log *slog.Logger) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:    cfg.Addr,
			Handler: handler,
		},
		log: log,
	}
}

func (s *HTTPServer) Run(ctx context.Context) error {

	chErr := make(chan error, 1)

	go func() {

		s.log.Info(
			"starting http server",
			slog.String("Addr", s.server.Addr),
		)

		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			chErr <- err
		}

		close(chErr)
	}()

	select {
	case err := <-chErr:
		s.log.Error(
			"failed to start server",
			"err",
			err,
			slog.String("Addr", s.server.Addr),
		)
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		s.log.Info(
			"server starting shutdown",
		)

		if err := s.shutdown(); err != nil {
			s.log.Error(
				"failed to shutdown",
				"err",
				err,
				slog.String("Addr", s.server.Addr),
			)
			_ = s.server.Close()
			return fmt.Errorf("failed to shutdown: %w", err)
		}
	}

	s.log.Info(
		"server successfully closed",
		slog.String("Addr", s.server.Addr),
	)

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

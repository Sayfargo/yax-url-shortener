package core_transport_http_middleware

import (
	"log/slog"
	"net/http"
	"time"

	core_transport_http_reponse "github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/response"
)

type Middleware func(http.Handler) http.Handler

type Logger interface {
	Info(msg string, args ...any)
}

func Logging(log Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			rw := core_transport_http_reponse.New(w)

			start := time.Now()

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			log.Info(
				"HTTP Request done",
				slog.String("URI", r.RequestURI),
				slog.String("Method", r.Method),
				slog.Int("status", rw.GetStatusCode()),
				slog.Duration("latency", duration),
				slog.Int("size", rw.GetBodySize()),
			)
		})
	}
}

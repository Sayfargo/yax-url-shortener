package core_transport_http_middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	core_transport_http_request "github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/request"
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

func GzipCompress() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			originalWriter := w

			acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

			if acceptsGzip {
				gzipWriter := core_transport_http_reponse.NewGzipWriter(w)

				originalWriter = gzipWriter

				defer gzipWriter.Close()
			}

			sendsGzip := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

			if sendsGzip {
				gzipReader, err := core_transport_http_request.NewGzipReader(r.Body)
				if err != nil {
					http.Error(
						w,
						"invalid gzip body",
						http.StatusBadRequest,
					)
					return
				}

				r.Body = gzipReader
				defer gzipReader.Close()
			}

			next.ServeHTTP(originalWriter, r)
		})
	}
}

package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/ctxkeys"
	httprequest "github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/request"
	httpreponse "github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/response"
	mycrypto "github.com/Sayfargo/yax-url-shortener/pkg/crypto"
	"github.com/google/uuid"
)

type Middleware func(http.Handler) http.Handler

type Logger interface {
	Info(msg string, args ...any)
}

func Auth(secretKey string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var userIDStr string

			cookie, err := r.Cookie("uid")

			switch {
			case err == nil:
				userIDStr, err = mycrypto.DecryptAESGCM(cookie.Value, secretKey)
				if err != nil {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}
			case errors.Is(err, http.ErrNoCookie):

				userID, err := uuid.NewUUID()
				if err != nil {
					http.Error(
						w,
						http.StatusText(http.StatusInternalServerError),
						http.StatusInternalServerError,
					)
					return
				}

				userIDStr = userID.String()

				encryptedVal, err := mycrypto.EncryptAESGCM(userIDStr, secretKey)
				if err != nil {
					http.Error(
						w,
						http.StatusText(http.StatusInternalServerError),
						http.StatusInternalServerError,
					)
					return
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "uid",
					Value:    encryptedVal,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					Expires:  time.Now().Add(time.Hour * 730),
				})

			default:
				http.Error(
					w,
					http.StatusText(http.StatusBadRequest),
					http.StatusBadRequest,
				)
				return
			}

			ctx := context.WithValue(r.Context(), ctxkeys.UserIDKey, userIDStr)
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func Logging(log Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			rw := httpreponse.New(w)

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
				gzipWriter := httpreponse.NewGzipWriter(w)

				originalWriter = gzipWriter

				defer gzipWriter.Close()
			}

			sendsGzip := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

			if sendsGzip {
				gzipReader, err := httprequest.NewGzipReader(r.Body)
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

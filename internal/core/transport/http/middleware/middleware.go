// Модуль middleware предоставляет набор слоев промежуточного программного обеспечения (HTTP middleware)
// для обеспечения аутентификации пользователей, логирования HTTP-запросов и прозрачного Gzip-сжатия.
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

// Middleware определяет сигнатуру функции для обертывания стандартных HTTP-обработчиков [http.Handler].
type Middleware func(http.Handler) http.Handler

// Logger описывает интерфейс логера, необходимого для работы middleware логирования.
type Logger interface {
	// Info записывает информационное сообщение с произвольным набором атрибутов структурированного лога.
	Info(msg string, args ...any)
}

// Auth возвращает middleware для аутентификации пользователей на основе зашифрованной куки сессии ("uid").
//
// Алгоритм работы:
//  1. Проверяется наличие куки "uid". Если она есть, значение расшифровывается с помощью [mycrypto.DecryptAESGCM].
//     При ошибке расшифровки возвращается статус 401 Unauthorized.
//  2. Если кука отсутствует, генерируется новый уникальный идентификатор пользователя [uuid.NewUUID] (UUID v1).
//     Идентификатор шифруется через [mycrypto.EncryptAESGCM] и выставляется клиенту в куку "uid"
//     со сроком действия 730 часов (примерно 1 месяц), флагами HttpOnly и SameSite=Lax.
//  3. Успешно полученный или сгенерированный строковый ID пользователя помещается в контекст запроса
//     по ключу [ctxkeys.UserIDKey].
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

// Logging возвращает middleware для профилирования и логирования входящих HTTP-запросов.
//
// По окончании обработки запроса метод выводит структурированную запись со следующими метриками:
//   - URI: путь запроса
//   - Method: HTTP-метод
//   - status: HTTP-статус ответа
//   - latency: общее время выполнения запроса
//   - size: размер тела ответа в байтах
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

// GzipCompress возвращает middleware для сквозного двухстороннего сжатия данных (Gzip).
//
// Поддерживаемые сценарии:
//  1. Если в заголовке запроса 'Accept-Encoding' содержится "gzip", поток ответа оборачивается
//     в [httpreponse.NewGzipWriter] для сжатия исходящих данных перед отправкой клиенту.
//  2. Если в заголовке запроса 'Content-Encoding' содержится "gzip", входящее тело запроса оборачивается
//     в [httprequest.NewGzipReader] для автоматической декомпрессии входящих данных, поступающих от клиента.
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

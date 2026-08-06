package core_transport_http_middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockLogger struct {
	Message string
	Attrs   map[string]any
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.Message = msg
	m.Attrs = make(map[string]any)

	for _, arg := range args {
		if attr, ok := arg.(slog.Attr); ok {
			m.Attrs[attr.Key] = attr.Value.Any()
		}
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Инициализируем мок логгера
	mockLogger := &MockLogger{}

	// Создаем тестовый обработчик (next), который имитирует успешный ответ
	expectedStatus := http.StatusOK
	expectedBody := "Hello, World"
	expectedSize := int64((len(expectedBody)))
	expectedUri := "/api/v1"
	expectedMethod := http.MethodGet
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Имитация работы
		time.Sleep(time.Millisecond * 5)

		w.WriteHeader(expectedStatus)
		_, _ = w.Write([]byte(expectedBody))
	})

	middleware := Logging(mockLogger)
	testHandler := middleware(nextHandler)

	req := httptest.NewRequest(expectedMethod, expectedUri, nil)
	rec := httptest.NewRecorder()

	testHandler.ServeHTTP(rec, req)

	assert.Equal(t, expectedStatus, rec.Code)
	assert.Equal(t, expectedBody, rec.Body.String())

	assert.Equal(t, "HTTP Request done", mockLogger.Message)
	assert.Equal(t, expectedUri, mockLogger.Attrs["URI"].(string))
	assert.Equal(t, expectedMethod, mockLogger.Attrs["Method"].(string))
	assert.Equal(t, int64(expectedStatus), mockLogger.Attrs["status"].(int64))
	assert.Equal(t, expectedSize, mockLogger.Attrs["size"].(int64))

	latency, ok := mockLogger.Attrs["latency"].(time.Duration)
	assert.True(t, ok)
	assert.NotZero(t, latency)
}

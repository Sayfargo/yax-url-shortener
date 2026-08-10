package core_transport_http_middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGzipCompress_RequestDecompression(t *testing.T) {
	handler := GzipCompress()(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			body, err := io.ReadAll(r.Body)

			require.NoError(t, err)

			assert.JSONEq(
				t,
				`{"message":"Hello, World!"}`,
				string(body),
			)
		},
	))

	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)

	zw.Write([]byte(`{"message":"Hello, World!"}`))

	zw.Close()

	req := httptest.NewRequest(http.MethodGet, "/", &buf)

	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestGzipCompress_NoAcceptEncoding(t *testing.T) {

	handler := GzipCompress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		json.NewEncoder(w).Encode(map[string]string{"message": "Hello, World!"})
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", nil)

	req.Header.Set("Accept-Encoding", "")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()

	assert.Empty(t, resp.Header.Get("Content-Encoding"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{"message":"Hello, World!"}`, string(body))
}

func TestGzipCompress_UnsupportedContentType(t *testing.T) {
	handler := GzipCompress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		w.Write([]byte("Hello, World!"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()

	assert.Empty(t, resp.Header.Get("Content-Encoding"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, "Hello, World!", string(body))
}

func TestGzipCompress_ResponseCompressedHTML(t *testing.T) {
	handler := GzipCompress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Type",
			"text/html",
		)

		w.Write([]byte(
			`<!DOCTYPE html>
				<title>Hello, World!</title>
				<p>Hello!`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()

	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	zr, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)

	body, err := io.ReadAll(zr)
	require.NoError(t, err)

	assert.Equal(
		t,
		`<!DOCTYPE html>
				<title>Hello, World!</title>
				<p>Hello!`,
		string(body))
}

func TestGzipCompress_InvalidGzip(t *testing.T) {

	handler := GzipCompress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Is nothing here"))
	}))

	var buf bytes.Buffer

	_, err := buf.Write([]byte("Hello, World!"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", &buf)

	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusCreated, rec.Code)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	responseBody := rec.Body.String()
	assert.Contains(t, responseBody, "invalid gzip body")
}

func TestGzipComrpess_ResponseCompressedJSON(t *testing.T) {

	handler := GzipCompress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		json.NewEncoder(w).Encode(map[string]string{"message": "Hello, World!"})
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()

	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	// Распаковывем и проверяем
	zr, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)

	body, err := io.ReadAll(zr)
	require.NoError(t, err)

	assert.JSONEq(t, `{"message":"Hello, World!"}`, string(body))
}

func TestGzipMiddleware_Response(t *testing.T) {
	router := chi.NewRouter()

	router.Use(GzipCompress())

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		json.NewEncoder(w).Encode(
			map[string]string{
				"message": "hello",
			},
		)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(
		http.MethodGet,
		server.URL+"/",
		nil,
	)
	require.NoError(t, err)

	req.Header.Set(
		"Accept-Encoding",
		"gzip",
	)

	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(
		t,
		"gzip",
		resp.Header.Get("Content-Encoding"),
	)

	zr, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)

	defer zr.Close()

	body, err := io.ReadAll(zr)
	require.NoError(t, err)

	assert.JSONEq(
		t,
		`{"message":"hello"}`,
		string(body),
	)
}
func TestGzipMiddleware_Request(t *testing.T) {
	router := chi.NewRouter()

	router.Use(GzipCompress())

	router.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		assert.Equal(
			t,
			`{"url":"https://google.com"}`,
			string(body),
		)

		w.WriteHeader(http.StatusCreated)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/api/shorten",
		gzipData(
			t,
			[]byte(`{"url":"https://google.com"}`),
		),
	)
	require.NoError(t, err)

	req.Header.Set(
		"Content-Encoding",
		"gzip",
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(
		t,
		http.StatusCreated,
		resp.StatusCode,
	)
}

func gzipData(t *testing.T, data []byte) io.Reader {
	t.Helper()

	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(data)
	require.NoError(t, err)

	err = zw.Close()
	require.NoError(t, err)

	return &buf
}

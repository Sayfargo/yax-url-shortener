package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/ctxkeys"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

// ExampleHandler_Create демонстрирует тестирование и использование эндпоинта Create.
// Метод принимает сырой URL в текстовом формате и возвращает сокращенную ссылку text/plain.
func ExampleHandler_Create() {
	// 1. Инициализируем моки для зависимостей хэндлера
	mockSvc := &mockURLShortener{
		createFn: func(ctx context.Context, url, uid string) (string, error) {
			return "http://localhost:8080/abcd1234", nil
		},
	}
	mockAudit := &mockAuditService{}
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 2. Создаем экземпляр хэндлера (пул БД передаем как nil, так как для Create он не нужен)
	h := New(mockSvc, discardLog, nil, mockAudit)

	// 3. Формируем тестовый HTTP-запрос с сырым URL в теле
	targetURL := "https://google.com"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(targetURL))

	// Обязательно добавляем ID пользователя в контекст, так как хэндлер проверяет авторизацию
	ctx := context.WithValue(req.Context(), ctxkeys.UserIDKey, "user-id-123")
	req = req.WithContext(ctx)

	// 4. Создаем httptest.ResponseRecorder для перехвата ответа сервера
	w := httptest.NewRecorder()

	// 5. Вызываем тестируемый хэндлер напрямую
	h.Create(w, req)

	// 6. Считываем и проверяем результат
	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Short URL: %s\n", string(body))

	// Output:
	// Status Code: 201
	// Content-Type: text/plain; charset=utf-8
	// Short URL: http://localhost:8080/abcd1234
}

// ExampleHandler_Shorten демонстрирует работу эндпоинта Shorten,
// который принимает JSON-запрос и возвращает сокращенный URL также в формате JSON.
func ExampleHandler_Shorten() {
	// 1. Инициализируем моки для зависимостей хэндлера
	mockSvc := &mockURLShortener{
		createFn: func(ctx context.Context, url, uid string) (string, error) {
			return "http://localhost:8080/abcd2345", nil
		},
	}

	mockAudit := &mockAuditService{}
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	// 2. Создаем экземпляр хэндлера (пул БД передаем как nil, так как для Create он не нужен)
	h := New(mockSvc, discardLog, nil, mockAudit)

	// 3. Формируем тестовый HTTP-запрос с JSON телом
	bodyReq := `{"url":"https://google.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(bodyReq))

	ctx := context.WithValue(req.Context(), ctxkeys.UserIDKey, "user-id-123")
	req = req.WithContext(ctx)

	// 4. Создаем httptest.ResponseRecorder для перехвата ответа сервера
	w := httptest.NewRecorder()
	// 5. Вызываем тестируемый хэндлер напрямую
	h.Shorten(w, req)
	// 6. Считываем и проверяем результат
	resp := w.Result()
	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Short URL: %s\n", string(bodyResp))

	// Output:
	// Status Code: 201
	// Content-Type: application/json
	// Short URL: {"result":"http://localhost:8080/abcd2345"}
}

// ExampleHandler_Redirect демонстрирует работу эндпоинта Redirect.
// Метод извлекает короткий код из параметров пути роутера, запрашивает
// оригинальный URL и возвращает HTTP-статус 307 (Temporary Redirect).
func ExampleHandler_Redirect() {
	mockSvc := &mockURLShortener{
		getOriginalFn: func(ctx context.Context, shortCode string) (string, error) {
			if shortCode == "abcdefg" {
				return "https://google.com", nil
			}
			return "", service.ErrURLDoesNotExists
		},
	}

	mockAudit := &mockAuditService{}
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := New(mockSvc, discardLog, nil, mockAudit)

	// 1. Создаем базовый GET запрос
	req := httptest.NewRequest(http.MethodGet, "/abcdefg", nil)

	// 2. Имитируем работу chi роутера: вручную добавляем параметр "id" в контекст запроса
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "abcdefg")

	reqCtx := context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx)
	req = req.WithContext(reqCtx)

	w := httptest.NewRecorder()

	// 3. Вызываем хэндлер
	h.Redirect(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Location Header: %s\n", resp.Header.Get("Location"))

	// Output:
	// Status Code: 307
	// Location Header: https://google.com
}

// ExampleHandler_ShortenBatch демонстрирует работу эндпоинта ShortenBatch,
// выполняющего массовое (пакетное) сокращение переданного массива ссылок.
func ExampleHandler_ShortenBatch() {
	// 1. Инициализируем мок сервиса для обработки пакета данных
	mockSvc := &mockURLShortener{
		createBatchFn: func(ctx context.Context, req []service.CreateURLBatchRequest, uid string) ([]service.CreateURLBatchResponse, error) {
			// мок возвращает массив ответов, сопоставляя их по CorrelationID
			res := make([]service.CreateURLBatchResponse, len(req))
			for i, item := range req {
				res[i] = service.CreateURLBatchResponse{
					CorrelationID: item.CorrelationID,
					ShortURL:      fmt.Sprintf("http://localhost:8080/short-%d", i+1),
				}
			}
			return res, nil
		},
	}

	mockAudit := &mockAuditService{}
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 2. Создаем экземпляр хэндлера
	h := New(mockSvc, discardLog, nil, mockAudit)

	// 3. Формируем массив ссылок для пакетного запроса в JSON формате
	bodyReq := `[
		{"correlation_id": "req-1", "original_url": "https://google.com"},
		{"correlation_id": "req-2", "original_url": "https://github.com"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(bodyReq))

	// 4. Эмулируем прохождение авторизационного слоя через добавление UserID в контекст
	ctx := context.WithValue(req.Context(), ctxkeys.UserIDKey, "user-id-777")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// 5. Вызываем обрабатывающий хэндлер
	h.ShortenBatch(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("JSON Response:\n%s\n", string(bodyResp))

	// Output:
	// Status Code: 201
	// Content-Type: application/json
	// JSON Response:
	// [{"correlation_id":"req-1","short_url":"http://localhost:8080/short-1"},{"correlation_id":"req-2","short_url":"http://localhost:8080/short-2"}]
}

// ExampleHandler_GetURLs демонстрирует работу эндпоинта GetURLs,
// возвращающего JSON-список всех сокращенных ссылок текущего пользователя.
func ExampleHandler_GetURLs() {
	// 1. Инициализируем мок сервиса, который возвращает список URL для конкретного пользователя
	mockSvc := &mockURLShortener{
		getUserURLsFn: func(ctx context.Context, uid string) ([]model.UserURL, error) {
			if uid == "user-id-999" {
				return []model.UserURL{
					{ShortURL: "http://localhost:8080/go", OriginalURL: "https://google.com"},
					{ShortURL: "http://localhost:8080/git", OriginalURL: "https://github.com"},
				}, nil
			}
			return nil, nil
		},
	}

	mockAudit := &mockAuditService{}
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 2. Создаем экземпляр хэндлера
	h := New(mockSvc, discardLog, nil, mockAudit)

	// 3. Формируем GET-запрос на получение списка ссылок
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	// 4. Эмулируем прохождение авторизации через добавление UserID в контекст
	ctx := context.WithValue(req.Context(), ctxkeys.UserIDKey, "user-id-999")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// 5. Вызываем обрабатывающий хэндлер
	h.GetURLs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("JSON Response:\n%s\n", string(bodyResp))

	// Output:
	// Status Code: 200
	// Content-Type: application/json
	// JSON Response:
	// [{"short_url":"http://localhost:8080/go","original_url":"https://google.com"},{"short_url":"http://localhost:8080/git","original_url":"https://github.com"}]
}

type mockURLShortener struct {
	createFn      func(ctx context.Context, url, uid string) (string, error)
	createBatchFn func(ctx context.Context, req []service.CreateURLBatchRequest, uid string) ([]service.CreateURLBatchResponse, error)
	getOriginalFn func(ctx context.Context, shortCode string) (string, error)
	getUserURLsFn func(ctx context.Context, uid string) ([]model.UserURL, error)
	deleteFn      func(ctx context.Context, uid string, shortCodes ...string) error
}

func (m *mockURLShortener) CreateShortURL(ctx context.Context, url, uid string) (string, error) {
	return m.createFn(ctx, url, uid)
}

func (m *mockURLShortener) CreateURLBatch(ctx context.Context, req []service.CreateURLBatchRequest, uid string) ([]service.CreateURLBatchResponse, error) {
	return m.createBatchFn(ctx, req, uid)
}

func (m *mockURLShortener) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	return m.getOriginalFn(ctx, shortCode)
}

func (m *mockURLShortener) GetUserURLs(ctx context.Context, uid string) ([]model.UserURL, error) {
	return m.getUserURLsFn(ctx, uid)
}

func (m *mockURLShortener) DeleteURLs(ctx context.Context, uid string, shortCodes ...string) error {
	return m.deleteFn(ctx, uid, shortCodes...)
}

type mockAuditService struct{}

func (m *mockAuditService) Shorten(ctx context.Context, userID, url string) {}
func (m *mockAuditService) Follow(ctx context.Context, userID, url string)  {}

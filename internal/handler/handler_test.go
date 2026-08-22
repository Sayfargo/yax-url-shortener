package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreate_OriginalURLConflict(t *testing.T) {
	mockSvc := NewMockUrlShortener(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	existingURL := "http://localhost:8080/HeZ1klEf"

	mockSvc.EXPECT().
		CreateShortUrl(mock.Anything, "https://google.com").
		Return(existingURL, service.ErrOriginalURLConflict).
		Once()

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("https://google.com"),
	)

	handler := New(mockSvc, log, nil)

	rw := httptest.NewRecorder()

	handler.Create(rw, req)

	resp := rw.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, existingURL, string(body))
}

func TestShorten_OriginalURLConflict(t *testing.T) {
	var (
		method = http.MethodPost
		target = "/api/shorten"
		header = "application/json"
	)

	existingURL := "http://localhost:8080/HeZ1klEf"

	mockSvc := NewMockUrlShortener(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSvc.EXPECT().
		CreateShortUrl(mock.Anything, "https://google.com").
		Return(existingURL, service.ErrOriginalURLConflict).
		Once()

	request := ShortUrlRequest{
		URL: "https://google.com",
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest(
		method,
		target,
		bytes.NewReader(data),
	)

	req.Header.Set("Content-Type", header)

	handler := New(mockSvc, log, nil)

	rw := httptest.NewRecorder()

	handler.Shorten(rw, req)

	resp := rw.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, header, resp.Header.Get("Content-Type"))

	var actual ShortUrlResponse

	err = json.NewDecoder(resp.Body).Decode(&actual)
	require.NoError(t, err)

	expected := ShortUrlResponse{
		Result: existingURL,
	}

	assert.Equal(t, expected, actual)
}

func TestShortenBatch_EmptyBatch(t *testing.T) {
	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	request := []CreateUrlBatchRequest{}

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSvc.EXPECT().CreateUrlBatch(mock.Anything, mock.Anything).Return(nil, service.ErrEmptyBatch)

	data, err := json.Marshal(request)
	require.NoError(t, err)

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(data))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	resp := rw.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "empty batch\n", string(body))
}

func TestShortenBatch_UnexpectedError(t *testing.T) {
	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	request := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://github.com",
		},
	}

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSvc.EXPECT().CreateUrlBatch(mock.Anything, mock.Anything).Return(nil, errors.New("unexpected error"))

	data, err := json.Marshal(request)
	require.NoError(t, err)

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(data))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	resp := rw.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

}

func TestShortenBatch_CollisionLimitExceeded(t *testing.T) {

	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	request := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://github.com",
		},
	}

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSvc.EXPECT().CreateUrlBatch(mock.Anything, mock.Anything).Return(nil, service.ErrShortCodeCollisionLimitExceeded)

	data, err := json.Marshal(request)
	require.NoError(t, err)

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(data))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	resp := rw.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "failed to process request, please try again\n", string(body))
}

func TestShortenBatch_IncorrectURL(t *testing.T) {

	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	request := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "biliberda!...",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://github.com",
		},
	}

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSvc.EXPECT().CreateUrlBatch(mock.Anything, mock.Anything).Return(nil, service.ErrIncorrectUrl)

	data, err := json.Marshal(request)
	require.NoError(t, err)

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(data))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	resp := rw.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

}

func TestShortenBatch_TooLargeBodyRequest(t *testing.T) {

	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	payload := []byte(
		`[{"correlation_id":"1","original_url":"` +
			strings.Repeat("a", maxBodySize+100) +
			`"}]`,
	)

	require.Greater(t, len(payload), maxBodySize)

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(payload))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	response := rw.Result()
	defer response.Body.Close()
	assert.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)

}

func TestShortenBatch_InvalidJSON(t *testing.T) {

	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	body := strings.NewReader(".![]>:)")

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	userRequest := httptest.NewRequest(method, target, body)
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	resp := rw.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, "failed to read request body\n", string(data))
}

func TestShortenBatch_Success(t *testing.T) {

	var (
		method = http.MethodPost
		target = "/api/shorten/batch"
		header = "application/json"
	)

	expectedResp := []service.CreateUrlBatchResponse{
		{
			CorrelationID: "1",
			ShortURL:      "http://localhost:8080/HeZ1klEf",
		},
		{
			CorrelationID: "2",
			ShortURL:      "http://localhost:8080/FhJ41liz",
		},
	}

	expectedBody := []CreateUrlBatchResponse{
		{
			CorrelationID: "1",
			ShortURL:      "http://localhost:8080/HeZ1klEf",
		},
		{
			CorrelationID: "2",
			ShortURL:      "http://localhost:8080/FhJ41liz",
		},
	}

	mockSvc := NewMockUrlShortener(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSvc.EXPECT().CreateUrlBatch(mock.Anything, mock.Anything).Return(expectedResp, nil)

	request := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://github.com",
		},
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(data))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, log, nil)
	rw := httptest.NewRecorder()
	handler.ShortenBatch(rw, userRequest)

	resp := rw.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var actual []CreateUrlBatchResponse

	err = json.NewDecoder(resp.Body).Decode(&actual)
	require.NoError(t, err)

	assert.Equal(t, header, resp.Header.Get("Content-Type"))
	assert.Equal(t, expectedBody, actual)

}

func TestCreate_ErrorResponses(t *testing.T) {
	testcases := []struct {
		name         string
		expectedCode int
		serviceError error
	}{
		{name: "Collision limit exceeded", expectedCode: http.StatusInternalServerError, serviceError: service.ErrShortCodeCollisionLimitExceeded},
		{name: "Incorrect URL", expectedCode: http.StatusBadRequest, serviceError: service.ErrIncorrectUrl},
		{name: "Unexpected error", expectedCode: http.StatusInternalServerError, serviceError: errors.New("unexpected error")},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			mockSvc := NewMockUrlShortener(t)
			logger := slog.New(
				slog.NewTextHandler(
					io.Discard,
					nil,
				),
			)
			mockSvc.EXPECT().CreateShortUrl(mock.Anything, mock.Anything).Return(mock.Anything, test.serviceError)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("bred..k"))

			handler := New(mockSvc, logger, nil)
			rw := httptest.NewRecorder()
			handler.Create(rw, req)

			response := rw.Result()
			defer response.Body.Close()

			assert.Equal(t, test.expectedCode, response.StatusCode)
		})
	}
}

func TestRedirect_ErrorResponses(t *testing.T) {
	testcases := []struct {
		name         string
		shortCode    string
		expectedCode int
		serviceError error
	}{
		{name: "URL does not exists", shortCode: "DnmPeRZF", expectedCode: http.StatusNotFound, serviceError: service.ErrUrlDoesNotExists},
		{name: "Unexpected error", shortCode: "DnmPeRZF", expectedCode: http.StatusInternalServerError, serviceError: errors.New("unexpected error")},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			mockSvc := NewMockUrlShortener(t)
			logger := slog.New(
				slog.NewTextHandler(
					io.Discard,
					nil,
				),
			)
			mockSvc.EXPECT().GetOriginalUrl(mock.Anything, test.shortCode).Return(mock.Anything, test.serviceError)

			target := fmt.Sprintf("/%s", test.shortCode)

			req := httptest.NewRequest(http.MethodGet, target, nil)

			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("id", test.shortCode)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

			handler := New(mockSvc, logger, nil)
			rw := httptest.NewRecorder()
			handler.Redirect(rw, req)

			response := rw.Result()
			defer response.Body.Close()

			assert.Equal(t, test.expectedCode, response.StatusCode)

		})
	}
}

func TestRedirect_InvalidShortCode(t *testing.T) {
	method := http.MethodGet
	shortCode := ""
	target := fmt.Sprintf("/%s", shortCode)

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	req := httptest.NewRequest(method, target, nil)

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.Redirect(rw, req)

	response := rw.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

// Весёлый кейс для обработчика Redirect с использованием корректных данных
func TestRedirect_Success(t *testing.T) {
	method := http.MethodGet
	shortCode := "Dkjf789E"
	target := fmt.Sprintf("/%s", shortCode) // /{id}
	// Ожидаемый url который должен вернуть сервер в своём ответе (в header)
	expectedLocation := "https://practicum.yandex.ru/go-developer-basic/?utm_source=ya&utm_medium=cpc&utm_campaign=Yan_Sch_RF_Prog_goDeba_b2c_Gener_Regular_Double_460&utm_content=sty_search:s_none:cid_711044074:gid_5766687058:kw_---autotargeting:pid_205766687058:aid_1913818178131297445:crid_0:rid_205766687058:p_1:pty_premium:mty_:mkw_:dty_desktop:cgcid_26898027:rn_%D0%A0%D0%BE%D1%81%D1%82%D0%BE%D0%B2-%D0%BD%D0%B0-%D0%94%D0%BE%D0%BD%D1%83:rid_39&etext=2202.KjoxDN06I-3sj5_fsSRWOB0TSMATnrks0_yjlXz4VIt9yG7I4nH0y2lfhULhTcKlw9ebyjxB_nUVfFb_9yTKt2Z6cGluc2h2Z2RqbWF2b3c.9a6e93b16369f6b76e7982ecada5deb67b2bdcce&yclid=11936302318399520767"

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockSvc.EXPECT().GetOriginalUrl(mock.Anything, shortCode).Return(expectedLocation, nil)

	req := httptest.NewRequest(method, target, nil)
	// Так как используется chi, передаем в роутер для chi ctx значение с shortCode
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.Redirect(rw, req)

	response := rw.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, response.StatusCode)
	assert.Equal(t, expectedLocation, response.Header.Get("Location"))
}

// Весёлый кейс для обработчика Create с использование корректных данных
func TestCreate_Success(t *testing.T) {
	method := http.MethodPost
	target := "/"
	header := "text/plain; charset=utf-8"
	expected := "https://anything.com/Dkjf789E"
	url := "https://practicum.yandex.ru/go-developer-basic/?utm_source=ya&utm_medium=cpc&utm_campaign=Yan_Sch_RF_Prog_goDeba_b2c_Gener_Regular_Double_460&utm_content=sty_search:s_none:cid_711044074:gid_5766687058:kw_---autotargeting:pid_205766687058:aid_1913818178131297445:crid_0:rid_205766687058:p_1:pty_premium:mty_:mkw_:dty_desktop:cgcid_26898027:rn_%D0%A0%D0%BE%D1%81%D1%82%D0%BE%D0%B2-%D0%BD%D0%B0-%D0%94%D0%BE%D0%BD%D1%83:rid_39&etext=2202.KjoxDN06I-3sj5_fsSRWOB0TSMATnrks0_yjlXz4VIt9yG7I4nH0y2lfhULhTcKlw9ebyjxB_nUVfFb_9yTKt2Z6cGluc2h2Z2RqbWF2b3c.9a6e93b16369f6b76e7982ecada5deb67b2bdcce&yclid=11936302318399520767"
	// Мок сервисного слоя
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockSvc := NewMockUrlShortener(t)
	mockSvc.EXPECT().CreateShortUrl(mock.Anything, url).Return(expected, nil)

	req := httptest.NewRequest(method, target, strings.NewReader(url))
	req.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.Create(rw, req)

	response := rw.Result()
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	assert.NotEmpty(t, body)
	assert.Equal(t, header, response.Header.Get("Content-Type"))
	assert.Equal(t, expected, string(body))

}

func TestShorten_ErrorResponses(t *testing.T) {
	testcases := []struct {
		name         string
		expectedCode int
		serviceError error
	}{
		{name: "Collision limit exceeded", expectedCode: http.StatusInternalServerError, serviceError: service.ErrShortCodeCollisionLimitExceeded},
		{name: "Incorrect URL", expectedCode: http.StatusBadRequest, serviceError: service.ErrIncorrectUrl},
		{name: "Unexpected error", expectedCode: http.StatusInternalServerError, serviceError: errors.New("unexpected error")},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			mockSvc := NewMockUrlShortener(t)
			logger := slog.New(
				slog.NewTextHandler(
					io.Discard,
					nil,
				),
			)
			mockSvc.EXPECT().CreateShortUrl(mock.Anything, mock.Anything).Return(mock.Anything, test.serviceError)

			request := ShortUrlRequest{
				URL: "bred..k",
			}

			data, err := json.Marshal(request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(data))

			handler := New(mockSvc, logger, nil)
			rw := httptest.NewRecorder()
			handler.Shorten(rw, req)

			response := rw.Result()
			defer response.Body.Close()

			assert.Equal(t, test.expectedCode, response.StatusCode)
		})
	}
}

func TestShorten_CreateShortUrl(t *testing.T) {
	var (
		method       = http.MethodPost
		target       = "/api/shorten"
		header       = "application/json"
		expected     = "https://anything.com/Dkjf789E"
		expectedJSON = `{"result":"https://anything.com/Dkjf789E"}`
		url          = "https://google.com"
	)

	mockSvc := NewMockUrlShortener(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockSvc.EXPECT().CreateShortUrl(mock.Anything, url).Return(expected, nil)

	request := ShortUrlRequest{
		URL: url,
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	userRequest := httptest.NewRequest(method, target, bytes.NewReader(data))
	userRequest.Header.Set("Content-Type", header)

	handler := New(mockSvc, logger, nil)
	rw := httptest.NewRecorder()
	handler.Shorten(rw, userRequest)

	response := rw.Result()
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	assert.NotEmpty(t, body)
	assert.Equal(t, header, response.Header.Get("Content-Type"))
	assert.Equal(t, expectedJSON, string(body))
}

package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler_mock "github.com/Sayfargo/yax-url-shortener/internal/handler/mock"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
			mockSvc := new(handler_mock.MockURLService)
			mockSvc.On("CreateShortUrl", mock.Anything, "bred..k").Return("", test.serviceError)

			t.Cleanup(func() {
				mockSvc.AssertExpectations(t)
			})

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bred..k"))

			handler := New(mockSvc)
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
			mockSvc := new(handler_mock.MockURLService)
			mockSvc.On("GetOriginalUrl", mock.Anything, test.shortCode).Return("", test.serviceError)

			target := fmt.Sprintf("/%s", test.shortCode)

			t.Cleanup(func() {
				mockSvc.AssertExpectations(t)
			})

			req := httptest.NewRequest(http.MethodGet, target, nil)

			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("id", test.shortCode)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

			handler := New(mockSvc)
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

	mockSvc := new(handler_mock.MockURLService)

	t.Cleanup(func() {
		mockSvc.AssertNotCalled(t, "GetOriginalUrl", mock.Anything, shortCode)
	})

	req := httptest.NewRequest(method, target, nil)

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler := New(mockSvc)
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

	mockSvc := new(handler_mock.MockURLService)
	mockSvc.On("GetOriginalUrl", mock.Anything, shortCode).Return(expectedLocation, nil)

	t.Cleanup(func() {
		mockSvc.AssertExpectations(t)
	})

	req := httptest.NewRequest(method, target, nil)
	// Так как используется chi, передаем в роутер для chi ctx значение с shortCode
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler := New(mockSvc)
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
	mockSvc := new(handler_mock.MockURLService)
	mockSvc.On("CreateShortUrl", mock.Anything, url).Return(expected, nil)

	t.Cleanup(func() {
		mockSvc.AssertExpectations(t)
	})

	req := httptest.NewRequest(method, target, strings.NewReader(url))
	req.Header.Set("Content-Type", header)

	handler := New(mockSvc)
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

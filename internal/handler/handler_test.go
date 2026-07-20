package handler

import (
	"context"
	"fmt"
	"io"
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

// Мок для сервисного слоя, так как без него не получится создать экземпляр handler
type MockURLService struct {
	mock.Mock
}

// Все методы, которые реализует сервисный слой для того, чтобы мок удовлетворял интерфейсу
func (m *MockURLService) CreateShortUrl(ctx context.Context, url string) (string, error) {
	args := m.Called(ctx, url)
	return args.String(0), args.Error(1)
}

func (m *MockURLService) GetOriginalUrl(ctx context.Context, shortCode string) (string, error) {
	args := m.Called(ctx, shortCode)
	return args.String(0), args.Error(1)
}

// Табличный тест с кейсами для обработчика Create, который принимает ссылку и отвечает короткой ссылкой
// Здесь только плохиие кейсы, которые явно должны вернуть ошибку
func TestCreate_Table(t *testing.T) {
	testcases := []struct {
		name         string
		url          string
		expectedCode int
		expectedMsg  string
	}{
		{name: "Empty URL", url: "", expectedCode: http.StatusBadRequest, expectedMsg: "Empty request body\n"},
		{name: "Incorrect URL #1", url: "ht:.ru", expectedCode: http.StatusBadRequest, expectedMsg: "Incorrect URL\n"},
		{name: "Incorrect URL #2", url: ";pk!ru", expectedCode: http.StatusBadRequest, expectedMsg: "Incorrect URL\n"},
		{name: "Incorrect URL #3", url: "lll//warket.com", expectedCode: http.StatusBadRequest, expectedMsg: "Incorrect URL\n"},
		{name: "Incorrect URL #4", url: "htps://goodgame.ogr", expectedCode: http.StatusBadRequest, expectedMsg: "Incorrect URL\n"},
		{name: "Incorrect URL #5", url: "https://goog le", expectedCode: http.StatusBadRequest, expectedMsg: "Incorrect URL\n"},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			mockSvc := new(MockURLService)
			// Заглушка для заглушки, просто чтоб была
			mockSvc.On("CreateShortUrl", mock.Anything, mock.Anything).Return("", nil)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.url))
			req.Header.Set("Content-Type", "text/plain; charset=utf-8")

			handler := New(mockSvc)
			rw := httptest.NewRecorder()
			handler.Create(rw, req)

			response := rw.Result()
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			assert.Equal(t, test.expectedCode, response.StatusCode)
			assert.Equal(t, test.expectedMsg, string(body))
		})
	}
}

// Табличный тест с кейсами для обработчика Redirect. Проверка на bad кейсы
func TestRedirect_Table(t *testing.T) {
	testcases := []struct {
		name         string
		shortCode    string
		expectedCode int
		expectedMsg  string
	}{
		{name: "Empty short code", shortCode: "", expectedCode: http.StatusBadRequest, expectedMsg: "URL Param is empty\n"},
		{name: "URL doesn't exists", shortCode: "Dks9dnbZ", expectedCode: http.StatusNotFound, expectedMsg: "404 page not found\n"},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			// Mock
			mockSvc := new(MockURLService)
			// Имитируем ответ от сервисного слоя
			if test.shortCode != "" {
				mockSvc.On("GetOriginalUrl", mock.Anything, test.shortCode).Return("", service.ErrUrlDoesNotExists)
			} else {
				mockSvc.On("GetOriginalUrl", mock.Anything, mock.Anything).Return("", nil)
			}

			target := fmt.Sprintf("/%s", test.shortCode)

			req := httptest.NewRequest(http.MethodGet, target, nil)
			// chi ctx
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("id", test.shortCode)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

			handler := New(mockSvc)
			rw := httptest.NewRecorder()
			handler.Redirect(rw, req)

			response := rw.Result()
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			assert.Equal(t, test.expectedCode, response.StatusCode)
			assert.Equal(t, test.expectedMsg, string(body))
		})
	}
}

// Весёлый кейс для обработчика Redirect с использованием корректных данных
func TestRedirect_HappyCase(t *testing.T) {
	method := http.MethodGet
	shortCode := "Dkjf789E"
	target := fmt.Sprintf("/%s", shortCode) // /{id}
	// Ожидаемый url который должен вернуть сервер в своём ответе (в header)
	expectedLocation := "https://practicum.yandex.ru/go-developer-basic/?utm_source=ya&utm_medium=cpc&utm_campaign=Yan_Sch_RF_Prog_goDeba_b2c_Gener_Regular_Double_460&utm_content=sty_search:s_none:cid_711044074:gid_5766687058:kw_---autotargeting:pid_205766687058:aid_1913818178131297445:crid_0:rid_205766687058:p_1:pty_premium:mty_:mkw_:dty_desktop:cgcid_26898027:rn_%D0%A0%D0%BE%D1%81%D1%82%D0%BE%D0%B2-%D0%BD%D0%B0-%D0%94%D0%BE%D0%BD%D1%83:rid_39&etext=2202.KjoxDN06I-3sj5_fsSRWOB0TSMATnrks0_yjlXz4VIt9yG7I4nH0y2lfhULhTcKlw9ebyjxB_nUVfFb_9yTKt2Z6cGluc2h2Z2RqbWF2b3c.9a6e93b16369f6b76e7982ecada5deb67b2bdcce&yclid=11936302318399520767"

	mockSvc := new(MockURLService)
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
func TestCreate_HappyCase(t *testing.T) {
	method := http.MethodPost
	target := "/"
	header := "text/plain; charset=utf-8"
	expected := "https://anything.com/Dkjf789E"
	url := "https://practicum.yandex.ru/go-developer-basic/?utm_source=ya&utm_medium=cpc&utm_campaign=Yan_Sch_RF_Prog_goDeba_b2c_Gener_Regular_Double_460&utm_content=sty_search:s_none:cid_711044074:gid_5766687058:kw_---autotargeting:pid_205766687058:aid_1913818178131297445:crid_0:rid_205766687058:p_1:pty_premium:mty_:mkw_:dty_desktop:cgcid_26898027:rn_%D0%A0%D0%BE%D1%81%D1%82%D0%BE%D0%B2-%D0%BD%D0%B0-%D0%94%D0%BE%D0%BD%D1%83:rid_39&etext=2202.KjoxDN06I-3sj5_fsSRWOB0TSMATnrks0_yjlXz4VIt9yG7I4nH0y2lfhULhTcKlw9ebyjxB_nUVfFb_9yTKt2Z6cGluc2h2Z2RqbWF2b3c.9a6e93b16369f6b76e7982ecada5deb67b2bdcce&yclid=11936302318399520767"
	// Мок сервисного слоя
	mockSvc := new(MockURLService)
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

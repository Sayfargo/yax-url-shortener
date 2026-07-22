package handler_mock

import (
	"context"

	"github.com/stretchr/testify/mock"
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

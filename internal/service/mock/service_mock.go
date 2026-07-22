package service_mock

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Мок для репозитория, чтобы создать экземпляр сервиса
type MockURLRepository struct {
	mock.Mock
}

type MockGoNanoIDGenerator struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url, shortCode string) error {
	args := m.Called(ctx, url, shortCode)
	return args.Error(0)
}

func (m *MockURLRepository) Get(ctx context.Context, shortCode string) (string, error) {
	args := m.Called(ctx, shortCode)
	return args.String(0), args.Error(1)
}

func (m *MockGoNanoIDGenerator) Generate(alphabet string, size int) (string, error) {
	args := m.Called(alphabet, size)
	return args.String(0), args.Error(1)
}

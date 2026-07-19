package service

import (
	"context"
	"testing"

	"net/url"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockURLRepository struct {
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

func TestGetOriginalUrl_HappyCase(t *testing.T) {
	// Params
	expectedUrl := "https://google.com"
	// Repository mock
	mockRep := new(MockURLRepository)
	mockRep.On("Get", mock.Anything, mock.Anything).Return(expectedUrl, nil)

	t.Cleanup(func() {
		mockRep.AssertExpectations(t)
	})

	svc := New(mockRep)

	originalUrl, err := svc.GetOriginalUrl(context.Background(), "FLeq19fl")
	require.NoError(t, err)

	assert.Equal(t, expectedUrl, originalUrl)
	_, err = url.ParseRequestURI(originalUrl)
	assert.NoError(t, err)
}

func TestCreateShortUrl_HappyCase(t *testing.T) {
	// Params
	testUrl := "https://google.com"
	// Repository mock
	mockRep := new(MockURLRepository)
	mockRep.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	t.Cleanup(func() {
		mockRep.AssertExpectations(t)
	})

	svc := New(mockRep)

	shortedUrl, err := svc.CreateShortUrl(context.Background(), testUrl)
	require.NoError(t, err)

	assert.NotEmpty(t, shortedUrl)
	_, err = url.ParseRequestURI(shortedUrl)
	assert.NoError(t, err)
}

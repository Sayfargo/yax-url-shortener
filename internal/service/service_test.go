package service

import (
	"context"
	"testing"

	"net/url"

	"github.com/Sayfargo/yax-url-shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestGetOriginalUrl_UrlNotExists(t *testing.T) {
	mockRepo := new(MockURLRepository)
	mockRepo.On("Get", mock.Anything, mock.Anything).Return("", repository.ErrNotExists)

	t.Cleanup(func() {
		mockRepo.AssertExpectations(t)
	})

	svc := New(mockRepo, new(GoNanoIDGenerator), "https://choto.com")

	result, err := svc.GetOriginalUrl(context.Background(), "fKM29FzE")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrUrlDoesNotExists)
}

func TestCreateShortUrl_ConflictRetry(t *testing.T) {

	mockRepo := new(MockURLRepository)
	mockGenerator := new(MockGoNanoIDGenerator)

	mockGenerator.On("Generate", alphabet, size).Return("ZEFIRMOY", nil).Once()
	mockGenerator.On("Generate", alphabet, size).Return("BULOCHKA", nil).Once()

	mockRepo.On("Create", mock.Anything, "random.com", "ZEFIRMOY").Return(repository.ErrAlreadyExists).Once()
	mockRepo.On("Create", mock.Anything, "random.com", "BULOCHKA").Return(nil).Once()

	t.Cleanup(func() {
		mockGenerator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	svc := New(mockRepo, mockGenerator, "https://choto.com")

	expectedUrl := svc.buildShortedUrl("BULOCHKA")
	require.NotEmpty(t, expectedUrl)

	result, err := svc.CreateShortUrl(context.Background(), "random.com")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, result)
}

func TestGetOriginalUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := new(MockURLRepository)
	mockRepo.On("Get", mock.Anything, mock.Anything).Return("", nil)

	t.Cleanup(func() {
		mockRepo.AssertNotCalled(t, "Get")
	})

	svc := New(mockRepo, new(GoNanoIDGenerator), "https://choto.com")

	result, err := svc.GetOriginalUrl(ctx, "shortCode")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}
func TestCreateShortUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := new(MockURLRepository)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	t.Cleanup(func() {
		mockRepo.AssertNotCalled(t, "Create")
	})

	svc := New(mockRepo, new(GoNanoIDGenerator), "https://choto.com")

	result, err := svc.CreateShortUrl(ctx, "anything")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
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

	svc := New(mockRep, new(GoNanoIDGenerator), "https://choto.com")

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

	svc := New(mockRep, new(GoNanoIDGenerator), "https://choto.com")

	shortedUrl, err := svc.CreateShortUrl(context.Background(), testUrl)
	require.NoError(t, err)

	assert.NotEmpty(t, shortedUrl)
	_, err = url.ParseRequestURI(shortedUrl)
	assert.NoError(t, err)
}

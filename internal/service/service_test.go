package service

import (
	"context"
	"testing"

	"net/url"

	repository_cache "github.com/Sayfargo/yax-url-shortener/internal/repository/cache"
	service_mock "github.com/Sayfargo/yax-url-shortener/internal/service/mock"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateShortUrl_IncorrectUrl(t *testing.T) {
	testcases := []struct {
		name        string
		url         string
		expectedErr error
	}{
		{name: "Incorrect URL #0", url: "      .ru", expectedErr: ErrIncorrectUrl},
		{name: "Incorrect URL #1", url: "ht:.ru", expectedErr: ErrIncorrectUrl},
		{name: "Incorrect URL #2", url: ";pk!ru", expectedErr: ErrIncorrectUrl},
		{name: "Incorrect URL #3", url: "lll//warket.com", expectedErr: ErrIncorrectUrl},
		{name: "Incorrect URL #4", url: "htps://goodgame.ogr", expectedErr: ErrIncorrectUrl},
		{name: "Incorrect URL #5", url: "https://goog le", expectedErr: ErrIncorrectUrl},
	}

	mockRepo := new(service_mock.MockCacheRepository)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	baseURL := "https://choto.com"

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			svc := New(mockRepo, new(service_mock.MockFileRepository), new(GoNanoIDGenerator), baseURL, validator.New())

			result, err := svc.CreateShortUrl(context.Background(), test.url)
			assert.Empty(t, result)
			assert.ErrorIs(t, err, test.expectedErr)
		})
	}
}

func TestGetOriginalUrl_UrlNotExists(t *testing.T) {
	mockRepo := new(service_mock.MockCacheRepository)
	mockRepo.On("Get", mock.Anything, mock.Anything).Return("", repository_cache.ErrNotExists)

	t.Cleanup(func() {
		mockRepo.AssertExpectations(t)
	})

	svc := New(mockRepo, new(service_mock.MockFileRepository), new(GoNanoIDGenerator), "https://choto.com", validator.New())

	result, err := svc.GetOriginalUrl(context.Background(), "fKM29FzE")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrUrlDoesNotExists)
}

func TestCreateShortUrl_ConflictRetry(t *testing.T) {

	mockRepo := new(service_mock.MockCacheRepository)
	mockFileRepo := new(service_mock.MockFileRepository)
	mockGenerator := new(service_mock.MockGoNanoIDGenerator)

	mockGenerator.On("Generate", alphabet, size).Return("ZEFIRMOY", nil).Once()
	mockGenerator.On("Generate", alphabet, size).Return("BULOCHKA", nil).Once()

	mockRepo.On("Create", mock.Anything, "https://random.com", "ZEFIRMOY").Return(repository_cache.ErrAlreadyExists).Once()
	mockRepo.On("Create", mock.Anything, "https://random.com", "BULOCHKA").Return(nil).Once()
	mockFileRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	t.Cleanup(func() {
		mockGenerator.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	svc := New(mockRepo, mockFileRepo, mockGenerator, "https://choto.com", validator.New())

	expectedUrl := svc.buildShortedUrl("BULOCHKA")
	require.NotEmpty(t, expectedUrl)

	result, err := svc.CreateShortUrl(context.Background(), "https://random.com")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, result)
}

func TestGetOriginalUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := new(service_mock.MockCacheRepository)
	mockRepo.On("Get", mock.Anything, mock.Anything).Return("", nil)

	t.Cleanup(func() {
		mockRepo.AssertNotCalled(t, "Get")
	})

	svc := New(mockRepo, new(service_mock.MockFileRepository), new(GoNanoIDGenerator), "https://choto.com", validator.New())

	result, err := svc.GetOriginalUrl(ctx, "shortCode")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}
func TestCreateShortUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := new(service_mock.MockCacheRepository)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	t.Cleanup(func() {
		mockRepo.AssertNotCalled(t, "Create")
	})

	svc := New(mockRepo, new(service_mock.MockFileRepository), new(GoNanoIDGenerator), "https://choto.com", validator.New())

	result, err := svc.CreateShortUrl(ctx, "anything")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGetOriginalUrl_Success(t *testing.T) {
	// Params
	expectedUrl := "https://google.com"
	// Repository mock
	mockRep := new(service_mock.MockCacheRepository)
	mockRep.On("Get", mock.Anything, mock.Anything).Return(expectedUrl, nil)

	t.Cleanup(func() {
		mockRep.AssertExpectations(t)
	})

	svc := New(mockRep, new(service_mock.MockFileRepository), new(GoNanoIDGenerator), "https://choto.com", validator.New())

	originalUrl, err := svc.GetOriginalUrl(context.Background(), "FLeq19fl")
	require.NoError(t, err)

	assert.Equal(t, expectedUrl, originalUrl)
	_, err = url.ParseRequestURI(originalUrl)
	assert.NoError(t, err)
}

func TestCreateShortUrl_Success(t *testing.T) {
	// Params
	testUrl := "https://google.com"
	// Repository mock
	mockRep := new(service_mock.MockCacheRepository)
	mockFileRepo := new(service_mock.MockFileRepository)
	mockRep.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockFileRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	t.Cleanup(func() {
		mockRep.AssertExpectations(t)
	})

	svc := New(mockRep, mockFileRepo, new(GoNanoIDGenerator), "https://choto.com", validator.New())

	shortedUrl, err := svc.CreateShortUrl(context.Background(), testUrl)
	require.NoError(t, err)

	assert.NotEmpty(t, shortedUrl)
	_, err = url.ParseRequestURI(shortedUrl)
	assert.NoError(t, err)
}

package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"net/url"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	repository_cache "github.com/Sayfargo/yax-url-shortener/internal/repository/cache"
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

	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	mockRepo := NewMockRepository(t)
	mockGen := NewMockGenerator(t)
	baseURL := "https://base.com"

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			svc := New(mockRepo, mockGen, baseURL, validator.New(), logger)

			result, err := svc.CreateShortUrl(context.Background(), test.url)
			assert.Empty(t, result)
			assert.ErrorIs(t, err, test.expectedErr)
		})
	}
}

func TestGetOriginalUrl_UrlNotExists(t *testing.T) {
	mockRepo := NewMockRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(mock.Anything, repository_cache.ErrNotExists)

	svc := New(mockRepo, mockGen, "https://base.com", validator.New(), logger)

	result, err := svc.GetOriginalUrl(context.Background(), "fKM29FzE")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrUrlDoesNotExists)
}

func TestCreateShortUrl_ConflictRetry(t *testing.T) {

	mockRepo := NewMockRepository(t)
	mockGenerator := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	mockGenerator.EXPECT().Generate(alphabet, size).Return("ZEFIRMOY", nil).Once()
	mockGenerator.EXPECT().Generate(alphabet, size).Return("BULOCHKA", nil).Once()

	expectedUrl := "https://base.com/BULOCHKA"

	mockRepo.EXPECT().
		Create(
			mock.Anything,
			mock.MatchedBy(func(u model.ShortenedUrl) bool {
				return u.ShortCode == "ZEFIRMOY" &&
					u.OriginalUrl == "https://original.url"
			}),
		).
		Return(repository_cache.ErrAlreadyExists).
		Once()
	mockRepo.EXPECT().
		Create(
			mock.Anything,
			mock.MatchedBy(func(u model.ShortenedUrl) bool {
				return u.ShortCode == "BULOCHKA" &&
					u.OriginalUrl == "https://original.url"
			}),
		).
		Return(nil).
		Once()

	svc := New(mockRepo, mockGenerator, "https://base.com", validator.New(), logger)

	require.NotEmpty(t, expectedUrl)

	result, err := svc.CreateShortUrl(context.Background(), "https://original.url")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, result)
}

func TestGetOriginalUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := NewMockRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	svc := New(mockRepo, mockGen, "https://base.com", validator.New(), logger)

	result, err := svc.GetOriginalUrl(ctx, "shortCode")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}
func TestCreateShortUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := NewMockRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	t.Cleanup(func() {
		mockRepo.AssertNotCalled(t, "Create")
	})

	svc := New(mockRepo, mockGen, "https://base.com", validator.New(), logger)

	result, err := svc.CreateShortUrl(ctx, "anything")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGetOriginalUrl_Success(t *testing.T) {
	// Params
	expectedUrl := "https://google.com"

	mockRep := NewMockRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockRep.EXPECT().Get(mock.Anything, mock.Anything).Return(expectedUrl, nil)

	svc := New(mockRep, mockGen, "https://base.com", validator.New(), logger)

	originalUrl, err := svc.GetOriginalUrl(context.Background(), "FLeq19fl")
	require.NoError(t, err)

	assert.Equal(t, expectedUrl, originalUrl)
	_, err = url.ParseRequestURI(originalUrl)
	assert.NoError(t, err)
}

func TestCreateShortUrl_Success(t *testing.T) {
	// Params
	testUrl := "https://google.com"

	mockRep := NewMockRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockGen.EXPECT().Generate(mock.Anything, mock.Anything).Return(mock.Anything, nil)
	mockRep.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	svc := New(mockRep, mockGen, "https://base.com", validator.New(), logger)

	shortedUrl, err := svc.CreateShortUrl(context.Background(), testUrl)
	require.NoError(t, err)

	assert.NotEmpty(t, shortedUrl)
	_, err = url.ParseRequestURI(shortedUrl)
	assert.NoError(t, err)
}

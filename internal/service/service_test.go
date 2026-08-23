package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"net/url"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	urlrepo "github.com/Sayfargo/yax-url-shortener/internal/repository/url"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const baseUrl = "https://base.com"

func TestCreateShortUrl_OriginalURLConflict(t *testing.T) {
	repo := NewMockURLRepository(t)
	generator := NewMockGenerator(t)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	validate := validator.New()

	service := New(
		repo,
		generator,
		baseUrl,
		validate,
		log,
	)

	originalURL := "https://google.com"

	generator.EXPECT().
		Generate(alphabet, size).
		Return("abc12345", nil)

	repo.EXPECT().
		Create(
			mock.Anything,
			mock.MatchedBy(func(u model.ShortenedUrl) bool {
				return u.OriginalUrl == originalURL &&
					u.ShortCode == "abc12345"
			}),
		).
		Return(&urlrepo.OriginalUrlConflictError{
			ShortCode: "existingCode",
		})

	result, err := service.CreateShortUrl(
		context.Background(),
		originalURL,
	)

	require.ErrorIs(t, err, ErrOriginalURLConflict)

	assert.Equal(
		t,
		baseUrl+"/"+"existingCode",
		result,
	)
}

func TestCreateUrlBatch_EmptyBatch(t *testing.T) {
	req := []CreateUrlBatchRequest{}

	repo := NewMockURLRepository(t)
	generator := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := &UrlShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validator.New(),
		log:       log,
		baseURL:   baseUrl,
	}

	_, err := svc.CreateUrlBatch(context.Background(), req)
	require.ErrorContains(t, err, "empty batch")

}

func TestCreateUrlBatch_CollisionLimitExceeded(t *testing.T) {
	ctx := context.Background()

	req := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
	}

	repo := NewMockURLRepository(t)
	generator := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := &UrlShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validator.New(),
		log:       log,
		baseURL:   baseUrl,
	}

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code1", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedUrl) bool {
			return len(batch) == 1 &&
				batch[0].ShortCode == "code1"
		})).
		Return(&urlrepo.BatchConflictError{
			Index: 0,
		}).
		Once()

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code2", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedUrl) bool {
			return len(batch) == 1 &&
				batch[0].ShortCode == "code2"
		})).
		Return(&urlrepo.BatchConflictError{
			Index: 0,
		}).
		Once()

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code3", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedUrl) bool {
			return len(batch) == 1 &&
				batch[0].ShortCode == "code3"
		})).
		Return(&urlrepo.BatchConflictError{
			Index: 0,
		}).
		Once()

	_, err := svc.CreateUrlBatch(ctx, req)
	require.ErrorIs(t, err, ErrShortCodeCollisionLimitExceeded)
}

func TestCreateUrlBatch_RetryOnBatchConflict(t *testing.T) {
	ctx := context.Background()

	req := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://github.com",
		},
	}

	repo := NewMockURLRepository(t)
	generator := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := &UrlShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validator.New(),
		log:       log,
		baseURL:   baseUrl,
	}

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code1", nil).
		Once()

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code2", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedUrl) bool {
			return len(batch) == 2 &&
				batch[0].ShortCode == "code1" &&
				batch[1].ShortCode == "code2"
		})).
		Return(&urlrepo.BatchConflictError{
			Index: 1,
		}).
		Once()

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code3", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedUrl) bool {
			return len(batch) == 2 &&
				batch[0].ShortCode == "code1" &&
				batch[1].ShortCode == "code3"
		})).
		Return(nil).
		Once()

	resp, err := svc.CreateUrlBatch(ctx, req)

	require.NoError(t, err)

	expected := []CreateUrlBatchResponse{
		{
			CorrelationID: "1",
			ShortURL:      baseUrl + "/" + "code1",
		},
		{
			CorrelationID: "2",
			ShortURL:      baseUrl + "/" + "code3",
		},
	}

	assert.Equal(t, expected, resp)
}

func TestCreateUrlBatch_RepositoryError(t *testing.T) {
	req := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockRepo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(errors.New("error")).Once()
	mockGen.EXPECT().Generate(alphabet, size).Return("BEzF9iSF", nil)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)
	_, err := svc.CreateUrlBatch(context.Background(), req)

	require.Contains(t, err.Error(), "repository create")
}

func TestCreateUrlBatch_GeneratorError(t *testing.T) {
	req := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockGen.EXPECT().Generate(alphabet, size).Return("", errors.New("error"))

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)
	_, err := svc.CreateUrlBatch(context.Background(), req)
	require.Contains(t, err.Error(), "generator generate")
}

func TestCreateUrlBatch_InvalidURL(t *testing.T) {
	req := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "abrakadabra",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)

	_, err := svc.CreateUrlBatch(context.Background(), req)
	require.ErrorIs(t, err, ErrIncorrectUrl)
}

func TestCreateUrlBatch_Success(t *testing.T) {

	req := []CreateUrlBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://github.com",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockGen.EXPECT().Generate(alphabet, size).Return("ABCD1234", nil).Once()
	mockGen.EXPECT().Generate(alphabet, size).Return("EFGH5678", nil).Once()

	mockRepo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedUrl) bool {
			return len(batch) == 2 &&
				batch[0].OriginalUrl == "https://google.com" &&
				batch[0].ShortCode == "ABCD1234" &&
				batch[1].OriginalUrl == "https://github.com" &&
				batch[1].ShortCode == "EFGH5678"
		})).
		Return(nil)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)

	result, err := svc.CreateUrlBatch(context.Background(), req)
	require.NoError(t, err)

	expectedResp := []CreateUrlBatchResponse{
		{
			CorrelationID: "1",
			ShortURL:      baseUrl + "/" + "ABCD1234",
		},
		{
			CorrelationID: "2",
			ShortURL:      baseUrl + "/" + "EFGH5678",
		},
	}

	for i, u := range result {
		assert.Equal(t, expectedResp[i].CorrelationID, u.CorrelationID)
		assert.Equal(t, expectedResp[i].ShortURL, u.ShortURL)
	}
}

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

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			svc := New(mockRepo, mockGen, baseUrl, validator.New(), logger)

			result, err := svc.CreateShortUrl(context.Background(), test.url)
			assert.Empty(t, result)
			assert.ErrorIs(t, err, test.expectedErr)
		})
	}
}

func TestGetOriginalUrl_UrlNotExists(t *testing.T) {
	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(mock.Anything, urlrepo.ErrNotExists)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), logger)

	result, err := svc.GetOriginalUrl(context.Background(), "fKM29FzE")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrUrlDoesNotExists)
}

func TestCreateShortUrl_ConflictRetry(t *testing.T) {

	mockRepo := NewMockURLRepository(t)
	mockGenerator := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	mockGenerator.EXPECT().Generate(alphabet, size).Return("ZEFIRMOY", nil).Once()
	mockGenerator.EXPECT().Generate(alphabet, size).Return("BULOCHKA", nil).Once()

	expectedUrl := baseUrl + "/" + "BULOCHKA"

	mockRepo.EXPECT().
		Create(
			mock.Anything,
			mock.MatchedBy(func(u model.ShortenedUrl) bool {
				return u.ShortCode == "ZEFIRMOY" &&
					u.OriginalUrl == "https://original.url"
			}),
		).
		Return(urlrepo.ErrConflictShortCode).
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

	svc := New(mockRepo, mockGenerator, baseUrl, validator.New(), logger)

	require.NotEmpty(t, expectedUrl)

	result, err := svc.CreateShortUrl(context.Background(), "https://original.url")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, result)
}

func TestGetOriginalUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), logger)

	result, err := svc.GetOriginalUrl(ctx, "shortCode")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}
func TestCreateShortUrl_ContextCanceled(t *testing.T) {
	// Params
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRepo := NewMockURLRepository(t)
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

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), logger)

	result, err := svc.CreateShortUrl(ctx, "anything")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGetOriginalUrl_Success(t *testing.T) {
	// Params
	expectedUrl := "https://google.com"

	mockRep := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockRep.EXPECT().Get(mock.Anything, mock.Anything).Return(expectedUrl, nil)

	svc := New(mockRep, mockGen, baseUrl, validator.New(), logger)

	originalUrl, err := svc.GetOriginalUrl(context.Background(), "FLeq19fl")
	require.NoError(t, err)

	assert.Equal(t, expectedUrl, originalUrl)
	_, err = url.ParseRequestURI(originalUrl)
	assert.NoError(t, err)
}

func TestCreateShortUrl_Success(t *testing.T) {
	// Params
	testUrl := "https://google.com"

	mockRep := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)
	mockGen.EXPECT().Generate(mock.Anything, mock.Anything).Return(mock.Anything, nil)
	mockRep.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	svc := New(mockRep, mockGen, baseUrl, validator.New(), logger)

	shortedUrl, err := svc.CreateShortUrl(context.Background(), testUrl)
	require.NoError(t, err)

	assert.NotEmpty(t, shortedUrl)
	_, err = url.ParseRequestURI(shortedUrl)
	assert.NoError(t, err)
}

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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const baseUrl = "https://base.com"

func TestGetUserURLs_UnexpectedError(t *testing.T) {
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

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	repo.EXPECT().
		GetURLs(mock.Anything, mock.Anything).
		Return(nil, errors.New("some error")).
		Once()

	resp, err := service.GetUserURLs(context.Background(), uid.String())
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestGetUserURLs_NoURLs(t *testing.T) {
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

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	repo.EXPECT().
		GetURLs(mock.Anything, mock.Anything).
		Return(nil, urlrepo.ErrNoRows).
		Once()

	resp, err := service.GetUserURLs(context.Background(), uid.String())
	require.ErrorIs(t, err, ErrURLsNotFound)
	require.Nil(t, resp)
}

func TestGetUserURLs_InvalidUid(t *testing.T) {
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

	uid := "invalid uuid/1!?,.>>>"

	resp, err := service.GetUserURLs(context.Background(), uid)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestGetUserURLs_Success(t *testing.T) {
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

	urlUUID, err := uuid.NewUUID()
	require.NoError(t, err)

	userUUID, err := uuid.NewUUID()
	require.NoError(t, err)

	shortCode := "short-url"
	OriginalURL := "original-url"

	shortURL := baseUrl + "/" + shortCode

	repo.EXPECT().
		GetURLs(mock.Anything, mock.Anything).
		Return([]model.ShortenedURL{
			{
				UUID:        urlUUID,
				ShortCode:   shortCode,
				OriginalURL: OriginalURL,
				UserID:      userUUID,
			},
		}, nil,
		)

	resp, err := service.GetUserURLs(context.Background(), userUUID.String())
	require.NoError(t, err)

	require.Equal(t, OriginalURL, resp[0].OriginalURL)
	require.Equal(t, shortURL, resp[0].ShortURL)
}

func TestCreateShortURL_OriginalURLConflict(t *testing.T) {
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

	OriginalURL := "https://google.com"

	generator.EXPECT().
		Generate(alphabet, size).
		Return("abc12345", nil)

	repo.EXPECT().
		Create(
			mock.Anything,
			mock.MatchedBy(func(u model.ShortenedURL) bool {
				return u.OriginalURL == OriginalURL &&
					u.ShortCode == "abc12345"
			}),
		).
		Return(&urlrepo.OriginalURLConflictError{
			ShortCode: "existingCode",
		})

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	result, err := service.CreateShortURL(
		context.Background(),
		OriginalURL,
		uid.String(),
	)

	require.ErrorIs(t, err, ErrOriginalURLConflict)

	assert.Equal(
		t,
		baseUrl+"/"+"existingCode",
		result,
	)
}

func TestCreateURLBatch_EmptyBatch(t *testing.T) {
	req := []CreateURLBatchRequest{}

	repo := NewMockURLRepository(t)
	generator := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := &URLShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validator.New(),
		log:       log,
		baseURL:   baseUrl,
	}

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	_, err = svc.CreateURLBatch(context.Background(), req, uid.String())
	require.ErrorContains(t, err, "empty batch")

}

func TestCreateURLBatch_CollisionLimitExceeded(t *testing.T) {
	ctx := context.Background()

	req := []CreateURLBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
	}

	repo := NewMockURLRepository(t)
	generator := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := &URLShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validator.New(),
		log:       log,
		baseURL:   baseUrl,
	}

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code1", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedURL) bool {
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
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedURL) bool {
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
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedURL) bool {
			return len(batch) == 1 &&
				batch[0].ShortCode == "code3"
		})).
		Return(&urlrepo.BatchConflictError{
			Index: 0,
		}).
		Once()

	_, err = svc.CreateURLBatch(ctx, req, uid.String())
	require.ErrorIs(t, err, ErrShortCodeCollisionLimitExceeded)
}

func TestCreateURLBatch_RetryOnBatchConflict(t *testing.T) {
	ctx := context.Background()

	req := []CreateURLBatchRequest{
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

	svc := &URLShortenerService{
		repo:      repo,
		generator: generator,
		validate:  validator.New(),
		log:       log,
		baseURL:   baseUrl,
	}

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code1", nil).
		Once()

	generator.EXPECT().
		Generate(alphabet, size).
		Return("code2", nil).
		Once()

	repo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedURL) bool {
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
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedURL) bool {
			return len(batch) == 2 &&
				batch[0].ShortCode == "code1" &&
				batch[1].ShortCode == "code3"
		})).
		Return(nil).
		Once()

	resp, err := svc.CreateURLBatch(ctx, req, uid.String())

	require.NoError(t, err)

	expected := []CreateURLBatchResponse{
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

func TestCreateURLBatch_RepositoryError(t *testing.T) {
	req := []CreateURLBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	mockRepo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(errors.New("error")).Once()
	mockGen.EXPECT().Generate(alphabet, size).Return("BEzF9iSF", nil)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)
	_, err = svc.CreateURLBatch(context.Background(), req, uid.String())

	require.Contains(t, err.Error(), "repository create")
}

func TestCreateURLBatch_GeneratorError(t *testing.T) {
	req := []CreateURLBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://google.com",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	mockGen.EXPECT().Generate(alphabet, size).Return("", errors.New("error"))

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)
	_, err = svc.CreateURLBatch(context.Background(), req, uid.String())
	require.Contains(t, err.Error(), "generator generate")
}

func TestCreateURLBatch_InvalidURL(t *testing.T) {
	req := []CreateURLBatchRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "abrakadabra",
		},
	}

	mockRepo := NewMockURLRepository(t)
	mockGen := NewMockGenerator(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)

	_, err = svc.CreateURLBatch(context.Background(), req, uid.String())
	require.ErrorIs(t, err, ErrIncorrectUrl)
}

func TestCreateURLBatch_Success(t *testing.T) {

	req := []CreateURLBatchRequest{
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

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	mockGen.EXPECT().Generate(alphabet, size).Return("ABCD1234", nil).Once()
	mockGen.EXPECT().Generate(alphabet, size).Return("EFGH5678", nil).Once()

	mockRepo.EXPECT().
		CreateBatch(mock.Anything, mock.MatchedBy(func(batch []model.ShortenedURL) bool {
			return len(batch) == 2 &&
				batch[0].OriginalURL == "https://google.com" &&
				batch[0].ShortCode == "ABCD1234" &&
				batch[1].OriginalURL == "https://github.com" &&
				batch[1].ShortCode == "EFGH5678"
		})).
		Return(nil)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), log)

	result, err := svc.CreateURLBatch(context.Background(), req, uid.String())
	require.NoError(t, err)

	expectedResp := []CreateURLBatchResponse{
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

func TestCreateShortURL_IncorrectUrl(t *testing.T) {
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

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			svc := New(mockRepo, mockGen, baseUrl, validator.New(), logger)

			result, err := svc.CreateShortURL(context.Background(), test.url, uid.String())
			assert.Empty(t, result)
			assert.ErrorIs(t, err, test.expectedErr)
		})
	}
}

func TestGetOriginalURL_UrlNotExists(t *testing.T) {
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

	result, err := svc.GetOriginalURL(context.Background(), "fKM29FzE")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrUrlDoesNotExists)
}

func TestCreateShortURL_ConflictRetry(t *testing.T) {

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
			mock.MatchedBy(func(u model.ShortenedURL) bool {
				return u.ShortCode == "ZEFIRMOY" &&
					u.OriginalURL == "https://original.url"
			}),
		).
		Return(urlrepo.ErrConflictShortCode).
		Once()
	mockRepo.EXPECT().
		Create(
			mock.Anything,
			mock.MatchedBy(func(u model.ShortenedURL) bool {
				return u.ShortCode == "BULOCHKA" &&
					u.OriginalURL == "https://original.url"
			}),
		).
		Return(nil).
		Once()

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	svc := New(mockRepo, mockGenerator, baseUrl, validator.New(), logger)

	require.NotEmpty(t, expectedUrl)

	result, err := svc.CreateShortURL(context.Background(), "https://original.url", uid.String())

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, result)
}

func TestGetOriginalURL_ContextCanceled(t *testing.T) {
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

	result, err := svc.GetOriginalURL(ctx, "shortCode")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}
func TestCreateShortURL_ContextCanceled(t *testing.T) {
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

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	svc := New(mockRepo, mockGen, baseUrl, validator.New(), logger)

	result, err := svc.CreateShortURL(ctx, "anything", uid.String())
	assert.Empty(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGetOriginalURL_Success(t *testing.T) {
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

	OriginalURL, err := svc.GetOriginalURL(context.Background(), "FLeq19fl")
	require.NoError(t, err)

	assert.Equal(t, expectedUrl, OriginalURL)
	_, err = url.ParseRequestURI(OriginalURL)
	assert.NoError(t, err)
}

func TestCreateShortURL_Success(t *testing.T) {
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

	uid, err := uuid.NewUUID()
	require.NoError(t, err)

	svc := New(mockRep, mockGen, baseUrl, validator.New(), logger)

	ShortedURL, err := svc.CreateShortURL(context.Background(), testUrl, uid.String())
	require.NoError(t, err)

	assert.NotEmpty(t, ShortedURL)
	_, err = url.ParseRequestURI(ShortedURL)
	assert.NoError(t, err)
}

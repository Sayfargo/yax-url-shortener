package url

import (
	"context"
	"testing"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBatch_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        uuid.New(),
			ShortCode:   "FaE9R129",
			OriginalUrl: "https://google.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "RbE9z121",
			OriginalUrl: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalUrl: "https://yandex.ru",
		},
	}

	err := repo.CreateBatch(ctx, shortenedUrls)
	require.NoError(t, err)

	gotUrls := make([]model.ShortenedUrl, 0, len(shortenedUrls))

	query := `SELECT uuid, short_code, original_url FROM shortened_urls`

	rows, err := testPool.Query(ctx, query)
	require.NoError(t, err)

	defer rows.Close()

	for rows.Next() {
		var url model.ShortenedUrl

		err = rows.Scan(&url.UUID, &url.ShortCode, &url.OriginalUrl)
		require.NoError(t, err)

		gotUrls = append(gotUrls, url)
	}

	err = rows.Err()
	require.NoError(t, err)

	require.Len(t, gotUrls, len(shortenedUrls))
	require.ElementsMatch(t, shortenedUrls, gotUrls)
}

func TestCreateBatch_Rollback(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	conflictedShortCode := "FaE9R129"
	existsUrl := "https://youtube.com"

	query := `INSERT INTO shortened_urls (uuid, short_code, original_url) VALUES ($1, $2, $3)`

	_, err := testPool.Exec(ctx, query, uuid.New(), conflictedShortCode, existsUrl)
	require.NoError(t, err)

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        uuid.New(),
			ShortCode:   "Ppr22Zp1",
			OriginalUrl: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalUrl: "https://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalUrl: "https://google.com",
		},
	}

	err = repo.CreateBatch(ctx, shortenedUrls)
	require.Error(t, err)

	queryCheck := `SELECT COUNT(*) FROM shortened_urls`

	var count int

	err = testPool.QueryRow(ctx, queryCheck).Scan(&count)
	require.NoError(t, err)

	require.Equal(t, 1, count)

	queryCheckRow := `SELECT original_url FROM shortened_urls WHERE short_code = $1`

	var url string

	err = testPool.QueryRow(ctx, queryCheckRow, "FaE9R129").Scan(&url)
	require.NoError(t, err)
	require.Equal(t, existsUrl, url)

}

func TestCreateBatch_ConflictShortCodeInDB(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	conflictedShortCode := "FaE9R129"
	conflictedIdx := 0

	query := `INSERT INTO shortened_urls (uuid, short_code, original_url) VALUES ($1, $2, $3)`

	_, err := testPool.Exec(ctx, query, uuid.New(), conflictedShortCode, "https://youtube.com")
	require.NoError(t, err)

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalUrl: "https://google.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "RbE9z121",
			OriginalUrl: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalUrl: "https://yandex.ru",
		},
	}

	err = repo.CreateBatch(ctx, shortenedUrls)
	var batchErr *BatchConflictError
	require.ErrorAs(t, err, &batchErr)
	require.ErrorIs(t, batchErr.Err, ErrConflictShortCode)
	require.Equal(t, conflictedIdx, batchErr.Index)
}

func TestCreateBatch_ConflictShortCodeInBatch(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	conflictedShortCode := "FaE9R129"
	conflictedIdx := 1

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalUrl: "https://google.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalUrl: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalUrl: "https://yandex.ru",
		},
	}

	err := repo.CreateBatch(ctx, shortenedUrls)
	var batchErr *BatchConflictError
	require.ErrorAs(t, err, &batchErr)
	require.ErrorIs(t, batchErr.Err, ErrConflictShortCode)
	require.Equal(t, conflictedIdx, batchErr.Index)

	queryCheck := `SELECT COUNT(*) FROM shortened_urls`

	var count int

	err = testPool.QueryRow(ctx, queryCheck).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)

}

func TestCreateBatch_ContextCanceled(t *testing.T) {
	cleanDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newRepo(t)

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        uuid.New(),
			ShortCode:   "Ppr22Zp1",
			OriginalUrl: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalUrl: "https://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "Far14Pp1",
			OriginalUrl: "https://google.com",
		},
	}

	err := repo.CreateBatch(ctx, shortenedUrls)
	require.ErrorIs(t, err, context.Canceled)
}

func TestGet_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	expectedShortCode := "FaE9R129"
	expectedUrl := "https://google.com"

	_, err := testPool.Exec(
		ctx,
		`INSERT INTO shortened_urls (uuid, short_code, original_url)
		VALUES ($1, $2, $3)`,
		uuid.New(),
		expectedShortCode,
		expectedUrl,
	)
	require.NoError(t, err)

	url, err := repo.Get(ctx, expectedShortCode)
	require.NoError(t, err)

	require.Equal(t, expectedUrl, url)
}

func TestGet_NotFound(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	url, err := repo.Get(ctx, "not exists")
	require.ErrorIs(t, err, ErrNotExists)
	require.Empty(t, url)
}

func TestGet_ContextCanceled(t *testing.T) {
	cleanDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newRepo(t)

	_, err := repo.Get(ctx, "anything")
	require.ErrorIs(t, err, context.Canceled)
}

func TestCreate_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	expectedUUID := uuid.New()
	expectedShortCode := "HaE9R121"
	expectedUrl := "https://github.com"

	shortenedUrl := model.ShortenedUrl{
		UUID:        expectedUUID,
		ShortCode:   expectedShortCode,
		OriginalUrl: expectedUrl,
	}

	err := repo.Create(ctx, shortenedUrl)
	require.NoError(t, err)

	query := `SELECT uuid, short_code, original_url FROM shortened_urls WHERE short_code = $1`

	var (
		gotUUID      uuid.UUID
		gotShortCode string
		gotUrl       string
	)

	err = testPool.QueryRow(ctx, query, expectedShortCode).Scan(&gotUUID, &gotShortCode, &gotUrl)
	require.NoError(t, err)

	assert.Equal(t, expectedUUID, gotUUID)
	assert.Equal(t, expectedShortCode, gotShortCode)
	assert.Equal(t, expectedUrl, gotUrl)
}

func TestCreate_DuplicateOriginalURL(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	UUID := uuid.New()
	ShortCode := "HaE9R121"
	URL := "https://github.com"

	shortenedUrl := model.ShortenedUrl{
		UUID:        UUID,
		ShortCode:   ShortCode,
		OriginalUrl: URL,
	}

	err := repo.Create(ctx, shortenedUrl)
	require.NoError(t, err)

	err = repo.Create(ctx, shortenedUrl)

	var conflictOrigURL *OriginalUrlConflictError
	require.ErrorAs(t, err, &conflictOrigURL)
	require.Equal(t, ShortCode, conflictOrigURL.ShortCode)
}

func TestCreate_ConflictShortCode(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	conflictedShortCode := "HaE9R121"

	shortenedUrl := model.ShortenedUrl{
		UUID:        uuid.New(),
		ShortCode:   conflictedShortCode,
		OriginalUrl: "https://github.com",
	}

	err := repo.Create(ctx, shortenedUrl)
	require.NoError(t, err)

	shortenedUrl1 := model.ShortenedUrl{
		UUID:        uuid.New(),
		ShortCode:   conflictedShortCode,
		OriginalUrl: "https://google.com",
	}

	err = repo.Create(ctx, shortenedUrl1)
	require.ErrorIs(t, err, ErrConflictShortCode)
}

func TestCreate_ContextCanceled(t *testing.T) {
	cleanDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newRepo(t)

	shortenedUrl := model.ShortenedUrl{
		UUID:        uuid.New(),
		ShortCode:   "HaE9R121",
		OriginalUrl: "https://google.com",
	}

	err := repo.Create(ctx, shortenedUrl)
	require.ErrorIs(t, err, context.Canceled)

}

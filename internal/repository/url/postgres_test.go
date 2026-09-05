package url

import (
	"context"
	"testing"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftDeleteURLs_DoesNotDeleteOtherUsersURLs(t *testing.T) {
	cleanDB(t)

	ctx := context.Background()
	repo := newRepo(t)

	userID := uuid.New()
	otherUserID := uuid.New()

	urls := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "MyURL123",
			OriginalURL: "https://google.com",
			UserID:      userID,
			IsDeleted:   false,
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "Other123",
			OriginalURL: "https://github.com",
			UserID:      otherUserID,
			IsDeleted:   false,
		},
	}

	queryCreate := `
		INSERT INTO shortened_urls 
			(uuid, short_code, original_url, user_id, is_deleted)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, u := range urls {
		_, err := testPool.Exec(
			ctx,
			queryCreate,
			u.UUID,
			u.ShortCode,
			u.OriginalURL,
			u.UserID,
			u.IsDeleted,
		)
		require.NoError(t, err)
	}

	err := repo.SoftDeleteURLs(
		ctx,
		userID,
		"MyURL123",
		"Other123",
	)
	require.NoError(t, err)

	queryCheck := `
		SELECT short_code, is_deleted
		FROM shortened_urls
		WHERE short_code = ANY($1)
	`

	rows, err := testPool.Query(
		ctx,
		queryCheck,
		[]string{
			"MyURL123",
			"Other123",
		},
	)
	require.NoError(t, err)

	defer rows.Close()

	deletedStatus := make(map[string]bool)

	for rows.Next() {
		var (
			shortCode string
			isDeleted bool
		)

		err := rows.Scan(&shortCode, &isDeleted)
		require.NoError(t, err)

		deletedStatus[shortCode] = isDeleted
	}

	require.NoError(t, rows.Err())

	assert.Equal(
		t,
		map[string]bool{
			"MyURL123": true,
			"Other123": false,
		},
		deletedStatus,
	)
}

func TestSoftDeleteURLs_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	userUUID, err := uuid.NewUUID()
	require.NoError(t, err)

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "FaE9R129",
			OriginalURL: "https://google.com",
			UserID:      userUUID,
			IsDeleted:   false,
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "RbE9z121",
			OriginalURL: "https://github.com",
			UserID:      userUUID,
			IsDeleted:   false,
		},
	}

	queryCreate := `INSERT INTO shortened_urls 
						(uuid, short_code, original_url, user_id, is_deleted)
					VALUES ($1, $2, $3, $4, $5)`

	for _, u := range ShortenedURLs {
		_, err = testPool.Exec(
			ctx,
			queryCreate,
			u.UUID,
			u.ShortCode,
			u.OriginalURL,
			u.UserID,
			u.IsDeleted,
		)
		require.NoError(t, err)
	}

	shortCodesToDelete := []string{"FaE9R129", "RbE9z121"}

	err = repo.SoftDeleteURLs(ctx, userUUID, shortCodesToDelete...)
	require.NoError(t, err)

	queryCheck := `SELECT is_deleted FROM shortened_urls WHERE user_id = $1 AND short_code = ANY($2)`

	rows, err := testPool.Query(ctx, queryCheck, userUUID, shortCodesToDelete)
	require.NoError(t, err)

	defer rows.Close()

	for rows.Next() {
		var isDeleted bool
		err = rows.Scan(&isDeleted)
		require.NoError(t, err)
		assert.True(t, isDeleted)
	}

	err = rows.Err()
	require.NoError(t, err)
}

func TestGetURLs_NoRows(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	existsUserID, err := uuid.NewUUID()
	require.NoError(t, err)

	notExistsUserID, err := uuid.NewUUID()
	require.NoError(t, err)

	queryCreate := `INSERT INTO shortened_urls 
						(uuid, short_code, original_url, user_id)
					VALUES ($1, $2, $3, $4)`

	_, err = testPool.Exec(
		ctx,
		queryCreate,
		uuid.New(),
		"short-code",
		"original-url",
		existsUserID,
	)
	require.NoError(t, err)

	result, err := repo.GetURLs(ctx, notExistsUserID)
	require.ErrorIs(t, err, ErrNoRows)
	require.Empty(t, result)
}

func TestGetURLs_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	urlUUID, err := uuid.NewUUID()
	require.NoError(t, err)

	userUUID, err := uuid.NewUUID()
	require.NoError(t, err)

	ShortenedURL := []model.ShortenedURL{
		{
			UUID:        urlUUID,
			ShortCode:   "short-code",
			OriginalURL: "original-url",
			UserID:      userUUID,
		},
	}

	queryCreate := `INSERT INTO shortened_urls 
						(uuid, short_code, original_url, user_id)
					VALUES ($1, $2, $3, $4)`

	_, err = testPool.Exec(
		ctx,
		queryCreate,
		ShortenedURL[0].UUID,
		ShortenedURL[0].ShortCode,
		ShortenedURL[0].OriginalURL,
		ShortenedURL[0].UserID,
	)
	require.NoError(t, err)

	result, err := repo.GetURLs(ctx, userUUID)
	require.NoError(t, err)

	require.ElementsMatch(t, ShortenedURL, result)
}

func TestCreateBatch_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "FaE9R129",
			OriginalURL: "https://google.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "RbE9z121",
			OriginalURL: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalURL: "https://yandex.ru",
		},
	}

	err := repo.CreateBatch(ctx, ShortenedURLs)
	require.NoError(t, err)

	gotURLs := make([]model.ShortenedURL, 0, len(ShortenedURLs))

	query := `SELECT uuid, short_code, original_url FROM shortened_urls`

	rows, err := testPool.Query(ctx, query)
	require.NoError(t, err)

	defer rows.Close()

	for rows.Next() {
		var url model.ShortenedURL

		err = rows.Scan(&url.UUID, &url.ShortCode, &url.OriginalURL)
		require.NoError(t, err)

		gotURLs = append(gotURLs, url)
	}

	err = rows.Err()
	require.NoError(t, err)

	require.Len(t, gotURLs, len(ShortenedURLs))
	require.ElementsMatch(t, ShortenedURLs, gotURLs)
}

func TestCreateBatch_Rollback(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	conflictedShortCode := "FaE9R129"
	existsURL := "https://youtube.com"

	query := `INSERT INTO shortened_urls (uuid, short_code, original_url) VALUES ($1, $2, $3)`

	_, err := testPool.Exec(ctx, query, uuid.New(), conflictedShortCode, existsURL)
	require.NoError(t, err)

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "Ppr22Zp1",
			OriginalURL: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalURL: "https://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalURL: "https://google.com",
		},
	}

	err = repo.CreateBatch(ctx, ShortenedURLs)
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
	require.Equal(t, existsURL, url)

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

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalURL: "https://google.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "RbE9z121",
			OriginalURL: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalURL: "https://yandex.ru",
		},
	}

	err = repo.CreateBatch(ctx, ShortenedURLs)
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

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalURL: "https://google.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   conflictedShortCode,
			OriginalURL: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalURL: "https://yandex.ru",
		},
	}

	err := repo.CreateBatch(ctx, ShortenedURLs)
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

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "Ppr22Zp1",
			OriginalURL: "https://github.com",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "zBA12Hp1",
			OriginalURL: "https://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "Far14Pp1",
			OriginalURL: "https://google.com",
		},
	}

	err := repo.CreateBatch(ctx, ShortenedURLs)
	require.ErrorIs(t, err, context.Canceled)
}

func TestGet_Success(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	expectedShortCode := "FaE9R129"
	expectedURL := "https://google.com"

	_, err := testPool.Exec(
		ctx,
		`INSERT INTO shortened_urls (uuid, short_code, original_url)
		VALUES ($1, $2, $3)`,
		uuid.New(),
		expectedShortCode,
		expectedURL,
	)
	require.NoError(t, err)

	url, err := repo.Get(ctx, expectedShortCode)
	require.NoError(t, err)

	require.Equal(t, expectedURL, url)
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
	expectedURL := "https://github.com"

	ShortenedURL := model.ShortenedURL{
		UUID:        expectedUUID,
		ShortCode:   expectedShortCode,
		OriginalURL: expectedURL,
	}

	err := repo.Create(ctx, ShortenedURL)
	require.NoError(t, err)

	query := `SELECT uuid, short_code, original_url FROM shortened_urls WHERE short_code = $1`

	var (
		gotUUID      uuid.UUID
		gotShortCode string
		gotURL       string
	)

	err = testPool.QueryRow(ctx, query, expectedShortCode).Scan(&gotUUID, &gotShortCode, &gotURL)
	require.NoError(t, err)

	assert.Equal(t, expectedUUID, gotUUID)
	assert.Equal(t, expectedShortCode, gotShortCode)
	assert.Equal(t, expectedURL, gotURL)
}

func TestCreate_DuplicateOriginalURL(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	UUID := uuid.New()
	ShortCode := "HaE9R121"
	URL := "https://github.com"

	ShortenedURL := model.ShortenedURL{
		UUID:        UUID,
		ShortCode:   ShortCode,
		OriginalURL: URL,
	}

	err := repo.Create(ctx, ShortenedURL)
	require.NoError(t, err)

	err = repo.Create(ctx, ShortenedURL)

	var conflictorigURL *OriginalURLConflictError
	require.ErrorAs(t, err, &conflictorigURL)
	require.Equal(t, ShortCode, conflictorigURL.ShortCode)
}

func TestCreate_ConflictShortCode(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := newRepo(t)

	conflictedShortCode := "HaE9R121"

	ShortenedURL := model.ShortenedURL{
		UUID:        uuid.New(),
		ShortCode:   conflictedShortCode,
		OriginalURL: "https://github.com",
	}

	err := repo.Create(ctx, ShortenedURL)
	require.NoError(t, err)

	ShortenedURL1 := model.ShortenedURL{
		UUID:        uuid.New(),
		ShortCode:   conflictedShortCode,
		OriginalURL: "https://google.com",
	}

	err = repo.Create(ctx, ShortenedURL1)
	require.ErrorIs(t, err, ErrConflictShortCode)
}

func TestCreate_ContextCanceled(t *testing.T) {
	cleanDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newRepo(t)

	ShortenedURL := model.ShortenedURL{
		UUID:        uuid.New(),
		ShortCode:   "HaE9R121",
		OriginalURL: "https://google.com",
	}

	err := repo.Create(ctx, ShortenedURL)
	require.ErrorIs(t, err, context.Canceled)

}

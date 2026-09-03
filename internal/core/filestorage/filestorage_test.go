package filestorage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertURLsEqual(t *testing.T, expected, actual []model.ShortenedURL) {
	t.Helper()
	require.Len(t, actual, len(expected))

	for i := range expected {
		assert.Equal(t, expected[i].UUID, actual[i].UUID)
		assert.Equal(t, expected[i].ShortCode, actual[i].ShortCode)
		assert.Equal(t, expected[i].OriginalURL, actual[i].OriginalURL)
	}
}

func TestFileStorage_CloseClosed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	err = fileStorage.Close()
	require.NoError(t, err)

	err = fileStorage.Close()
	require.Error(t, err)
}

func TestFileStorage_Close(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	err = fileStorage.Close()
	require.NoError(t, err)

	ShortenedURL := model.ShortenedURL{
		UUID:        uuid.New(),
		ShortCode:   "4rSPg8ap",
		OriginalURL: "http://yandex.ru",
	}

	err = fileStorage.WriteURL(ShortenedURL)
	require.Error(t, err)
}

func TestFileStorage_AppendAfterRead(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "4rSPg8ap",
			OriginalURL: "http://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "edVPg3ks",
			OriginalURL: "http://ya.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "dG56Hqxm",
			OriginalURL: "http://practicum.yandex.ru",
		},
	}

	// Сначала делаем первые три записи
	for _, ShortenedURL := range ShortenedURLs {
		err = fileStorage.WriteURL(ShortenedURL)
		require.NoError(t, err)
	}

	// Внутри ReadURLs указатель свдигается на начало файла - читаем записи
	results, err := fileStorage.ReadURLs()
	require.NoError(t, err)

	assert.Equal(t, len(ShortenedURLs), len(results))

	// Проверяем что первые три записи были прочитаны в правильном порядке

	assertURLsEqual(t, ShortenedURLs, results)

	newRecord := model.ShortenedURL{
		UUID:        uuid.New(),
		ShortCode:   "v256HZ3m",
		OriginalURL: "https://google.com",
	}

	ShortenedURLs = append(ShortenedURLs, newRecord)

	// По логике должны записать новую запись в конец файла
	err = fileStorage.WriteURL(newRecord)
	require.NoError(t, err)

	newResults, err := fileStorage.ReadURLs()
	require.NoError(t, err)

	assert.Equal(t, len(ShortenedURLs), len(newResults))

	// Снова проверяем были ли прочтены все записи в правильном порядке
	assertURLsEqual(t, ShortenedURLs, newResults)
}

func TestFileStorage_InvalidPath(t *testing.T) {

	file := ""

	cfg := &Config{
		FilePath: file,
	}

	fs, err := Init(cfg)
	require.Error(t, err)

	assert.Nil(t, fs)
}

func TestFileStorage_ReadURLs(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "4rSPg8ap",
			OriginalURL: "http://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "edVPg3ks",
			OriginalURL: "http://ya.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "dG56Hqxm",
			OriginalURL: "http://practicum.yandex.ru",
		},
	}

	for _, ShortenedURL := range ShortenedURLs {
		err = fileStorage.WriteURL(ShortenedURL)
		require.NoError(t, err)
	}

	// testing ReadURLs
	results, err := fileStorage.ReadURLs()
	require.NoError(t, err)

	assert.Equal(t, len(ShortenedURLs), len(results))

	assertURLsEqual(t, ShortenedURLs, results)

}

func TestFileStorage_WriteURL(t *testing.T) {

	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	ShortenedURLs := []model.ShortenedURL{
		{
			UUID:        uuid.New(),
			ShortCode:   "4rSPg8ap",
			OriginalURL: "http://yandex.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "edVPg3ks",
			OriginalURL: "http://ya.ru",
		},
		{
			UUID:        uuid.New(),
			ShortCode:   "dG56Hqxm",
			OriginalURL: "http://practicum.yandex.ru",
		},
	}

	for _, ShortenedURL := range ShortenedURLs {
		err = fileStorage.WriteURL(ShortenedURL)
		require.NoError(t, err)
	}

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	assert.NotEmpty(t, data)

	scanner := bufio.NewScanner(bytes.NewReader(data))

	var lineCount int

	for scanner.Scan() {

		line := scanner.Bytes()

		var readUrl model.ShortenedURL

		err := json.Unmarshal(line, &readUrl)
		assert.NoError(t, err)

		assert.Equal(t, ShortenedURLs[lineCount].UUID, readUrl.UUID)
		assert.Equal(t, ShortenedURLs[lineCount].ShortCode, readUrl.ShortCode)
		assert.Equal(t, ShortenedURLs[lineCount].OriginalURL, readUrl.OriginalURL)

		lineCount++
	}

	require.NoError(t, scanner.Err())

	assert.Equal(t, len(ShortenedURLs), lineCount)

}

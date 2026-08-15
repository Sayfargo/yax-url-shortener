package core_storage_file

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	config_storage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertURLsEqual(t *testing.T, expected, actual []model.ShortenedUrl) {
	require.Len(t, actual, len(expected))

	for i := range expected {
		assert.Equal(t, expected[i].UUID, actual[i].UUID)
		assert.Equal(t, expected[i].ShortCode, actual[i].ShortCode)
		assert.Equal(t, expected[i].OriginalUrl, actual[i].OriginalUrl)
	}
}

func TestFileStorage_CloseClosed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &config_storage.Config{
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

	cfg := &config_storage.Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	err = fileStorage.Close()
	require.NoError(t, err)

	shortenedUrl := model.ShortenedUrl{
		UUID:        "1",
		ShortCode:   "4rSPg8ap",
		OriginalUrl: "http://yandex.ru",
	}

	err = fileStorage.WriteURL(shortenedUrl)
	require.Error(t, err)
}

func TestFileStorage_AppendAfterRead(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &config_storage.Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        "1",
			ShortCode:   "4rSPg8ap",
			OriginalUrl: "http://yandex.ru",
		},
		{
			UUID:        "2",
			ShortCode:   "edVPg3ks",
			OriginalUrl: "http://ya.ru",
		},
		{
			UUID:        "3",
			ShortCode:   "dG56Hqxm",
			OriginalUrl: "http://practicum.yandex.ru",
		},
	}

	// Сначала делаем первые три записи
	for _, shortenedUrl := range shortenedUrls {
		err = fileStorage.WriteURL(shortenedUrl)
		require.NoError(t, err)
	}

	// Внутри ReadURLs указатель свдигается на начало файла - читаем записи
	results, err := fileStorage.ReadURLs()
	require.NoError(t, err)

	assert.Equal(t, len(shortenedUrls), len(results))

	// Проверяем что первые три записи были прочитаны в правильном порядке
	assertURLsEqual(t, shortenedUrls, results)

	newRecord := model.ShortenedUrl{
		UUID:        "4",
		ShortCode:   "v256HZ3m",
		OriginalUrl: "https://google.com",
	}

	shortenedUrls = append(shortenedUrls, newRecord)

	// По логике должны записать новую запись в конец файла
	err = fileStorage.WriteURL(newRecord)
	require.NoError(t, err)

	newResults, err := fileStorage.ReadURLs()
	require.NoError(t, err)

	assert.Equal(t, len(shortenedUrls), len(newResults))

	// Снова проверяем были ли прочтены все записи в правильном порядке
	assertURLsEqual(t, shortenedUrls, newResults)
}

func TestFileStorage_InvalidPath(t *testing.T) {

	file := ""

	cfg := &config_storage.Config{
		FilePath: file,
	}

	fs, err := Init(cfg)
	require.Error(t, err)

	assert.Nil(t, fs)
}

func TestFileStorage_ReadURLs(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &config_storage.Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        "1",
			ShortCode:   "4rSPg8ap",
			OriginalUrl: "http://yandex.ru",
		},
		{
			UUID:        "2",
			ShortCode:   "edVPg3ks",
			OriginalUrl: "http://ya.ru",
		},
		{
			UUID:        "3",
			ShortCode:   "dG56Hqxm",
			OriginalUrl: "http://practicum.yandex.ru",
		},
	}

	for _, shortenedUrl := range shortenedUrls {
		err = fileStorage.WriteURL(shortenedUrl)
		require.NoError(t, err)
	}

	// testing ReadURLs
	results, err := fileStorage.ReadURLs()
	require.NoError(t, err)

	assert.Equal(t, len(shortenedUrls), len(results))

	assertURLsEqual(t, shortenedUrls, results)

}

func TestFileStorage_WriteURL(t *testing.T) {

	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &config_storage.Config{
		FilePath: tmpFile,
	}

	fileStorage, err := Init(cfg)
	require.NoError(t, err)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	shortenedUrls := []model.ShortenedUrl{
		{
			UUID:        "1",
			ShortCode:   "4rSPg8ap",
			OriginalUrl: "http://yandex.ru",
		},
		{
			UUID:        "2",
			ShortCode:   "edVPg3ks",
			OriginalUrl: "http://ya.ru",
		},
		{
			UUID:        "3",
			ShortCode:   "dG56Hqxm",
			OriginalUrl: "http://practicum.yandex.ru",
		},
	}

	for _, shortenedUrl := range shortenedUrls {
		err = fileStorage.WriteURL(shortenedUrl)
		require.NoError(t, err)
	}

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	assert.NotEmpty(t, data)

	scanner := bufio.NewScanner(bytes.NewReader(data))

	var lineCount int

	for scanner.Scan() {

		line := scanner.Bytes()

		var readUrl model.ShortenedUrl

		err := json.Unmarshal(line, &readUrl)
		assert.NoError(t, err)

		assert.Equal(t, shortenedUrls[lineCount].UUID, readUrl.UUID)
		assert.Equal(t, shortenedUrls[lineCount].ShortCode, readUrl.ShortCode)
		assert.Equal(t, shortenedUrls[lineCount].OriginalUrl, readUrl.OriginalUrl)

		lineCount++
	}

	require.NoError(t, scanner.Err())

	assert.Equal(t, len(shortenedUrls), lineCount)

}

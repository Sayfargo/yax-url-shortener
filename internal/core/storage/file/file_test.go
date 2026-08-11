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

func TestFileStorage_WriteURL(t *testing.T) {

	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	cfg := &config_storage.Config{
		FilePath: tmpFile,
	}

	fileStorage, err := New(cfg)

	defer func() {
		err := fileStorage.Close()
		require.NoError(t, err)
	}()

	require.NoError(t, err)

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
		assert.NoError(t, err)
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

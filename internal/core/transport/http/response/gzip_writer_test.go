package httpresponse

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipWriter_Header(t *testing.T) {
	rec := httptest.NewRecorder()
	gw := NewGzipWriter(rec)

	gw.Header().Set("X-Test-Header", "golang")
	assert.Equal(t, "golang", rec.Header().Get("X-Test-Header"))
}

func TestGzipWriter_WrongContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	gw := NewGzipWriter(rec)

	gw.Header().Set("Content-Type", "text/plain")

	payload := []byte("do not compress this shot!")
	n, err := gw.Write(payload)

	require.NoError(t, err)
	assert.Equal(t, len(payload), n)

	require.NoError(t, gw.Close())

	assert.NotEqual(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "do not compress this shot!", rec.Body.String())
}

func TestGzipWriter_SuccessJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	gw := NewGzipWriter(rec)

	gw.Header().Set("Content-Type", "application/json")

	payload := []byte(`{"status":"ok","message":"hello"}`)
	_, err := gw.Write(payload)
	require.NoError(t, err)

	require.NoError(t, gw.Close())

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, http.StatusOK, rec.Code)

	gzReader, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gzReader.Close()

	uncompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, string(payload), string(uncompressedData))
}

func TestGzipWriter_SuccessHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	gw := NewGzipWriter(rec)

	gw.Header().Set("Content-Type", "text/html; charset=utf-8")

	gw.WriteHeader(http.StatusCreated)

	payload := []byte("<html><body>Hello</body></html>")
	_, err := gw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestGzipWriter_NoContentStatuses(t *testing.T) {
	statuses := []int{http.StatusNoContent, http.StatusNotModified}

	for _, code := range statuses {
		t.Run("StatusCode_", func(t *testing.T) {
			rec := httptest.NewRecorder()
			gw := NewGzipWriter(rec)

			gw.Header().Set("Content-Type", "application/json")
			gw.WriteHeader(code)

			require.NoError(t, gw.Close())

			assert.NotEqual(t, "gzip", rec.Header().Get("Content-Encoding"))
			assert.Equal(t, code, rec.Code)
		})
	}
}

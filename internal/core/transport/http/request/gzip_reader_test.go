package httprequest

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCloser struct {
	io.Reader
	closeErr error
}

func (m mockCloser) Close() error {
	return m.closeErr
}

func TestNewGzipReader_Error(t *testing.T) {
	badReader := io.NopCloser(bytes.NewReader([]byte("not-a-gzip-data")))

	gzReader, err := NewGzipReader(badReader)

	assert.Error(t, err)
	assert.Nil(t, gzReader)
}

func TestGzipReader_ReadAndClose_Success(t *testing.T) {
	expectedText := "hello!"

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write([]byte(expectedText))
	require.NoError(t, err)

	err = gzWriter.Close()

	require.NoError(t, err)

	body := io.NopCloser(&buf)

	gzReader, err := NewGzipReader(body)
	require.NoError(t, err)
	require.NotNil(t, gzReader)

	resultData, err := io.ReadAll(gzReader)
	require.NoError(t, err)
	assert.Equal(t, expectedText, string(resultData))

	err = gzReader.Close()

	assert.NoError(t, err)
}

func TestGzipReader_Close_WithErrors(t *testing.T) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	err := gzWriter.Close()

	require.NoError(t, err)

	expectedCloseErr := errors.New("failed to close reader")

	mockedBody := mockCloser{
		Reader:   &buf,
		closeErr: expectedCloseErr,
	}

	gzReader, err := NewGzipReader(mockedBody)
	require.NoError(t, err)

	err = gzReader.Close()
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedCloseErr)
}

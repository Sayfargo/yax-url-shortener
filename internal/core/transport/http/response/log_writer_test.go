package httpresponse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggingResponseWriter_WriteAndHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := New(rec)

	lrw.WriteHeader(http.StatusAccepted)

	payload := []byte("test body")
	size, err := lrw.Write(payload)

	assert.NoError(t, err)
	assert.Equal(t, len(payload), size)
	assert.Equal(t, len(payload), lrw.GetBodySize())

	assert.Equal(t, http.StatusAccepted, lrw.GetStatusCode())
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestLoggingResponseWriter_GetStatusCode_Panic(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := New(rec)

	assert.PanicsWithValue(t, "no status code set", func() {
		lrw.GetStatusCode()
	})
}

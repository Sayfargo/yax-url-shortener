package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPServer_Run_Success(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &Config{
		Addr: "127.0.0.1:0",
	}

	handler := http.NewServeMux()
	httpServer := New(handler, cfg, discardLogger)

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)

	go func() {
		errChan <- httpServer.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		require.NoError(t, err)

	case <-time.After(2 * time.Second):
		t.Fatal("server did not shutdown within timeout")
	}
}

func TestHTTPServer_Run_InvalidAddr(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &Config{
		Addr: "999.999.999.999:99999",
	}
	handler := http.NewServeMux()

	httpServer := New(handler, cfg, discardLogger)
	ctx := context.Background()

	err := httpServer.Run(ctx)
	require.Errorf(t, err, "expected error due to invalid address, but got nil")

}

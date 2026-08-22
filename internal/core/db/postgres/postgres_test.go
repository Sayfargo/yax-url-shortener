package postgres

import (
	"io"
	"log/slog"
	"testing"

	postgres "github.com/Sayfargo/yax-url-shortener/internal/config/db/postgres"
	"github.com/stretchr/testify/require"
)

func TestNew_ParseConfigError(t *testing.T) {

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := postgres.Config{
		DBDsn: "postgres://user:pass@host:port/db?invalid param >:) ",
	}

	pool, err := New(cfg, log)
	require.Error(t, err)
	require.Nil(t, pool)
}

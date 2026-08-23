package postgres

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_ParseConfigError(t *testing.T) {

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := Config{
		DBDsn: "postgres://user:pass@host:port/db?invalid param >:) ",
	}

	pool, err := New(cfg, log)
	require.Error(t, err)
	require.Nil(t, pool)
}

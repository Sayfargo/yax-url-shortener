package core_slogger

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	config_slogger "github.com/Sayfargo/yax-url-shortener/internal/config/slogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogger_CreateLogs(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()

	logger, closer, err := New(config_slogger.Config{
		Directory: dir,
		Stdout:    config_slogger.StdoutConfig{Enabled: true, Format: config_slogger.FormatText, Writer: &buf},
		Files: []config_slogger.FileConfig{
			{
				Name:    "app",
				Enabled: true,
				Format:  config_slogger.FormatText,
				Level:   slog.LevelInfo,
			},
		},
	})

	require.NoError(t, err)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Error("error message")

	l := logger.With(
		slog.String("key", "value"),
	)

	l.Debug("debug message")
	l.Info("info message")
	l.Error("error message")

	err = closer.Close()
	require.NoError(t, err)

	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	assert.Equalf(t, 1, len(files), "expected 1 log files, got %d", len(files))
	assert.NotEmpty(t, buf)
}

func TestSlogger_StdoutDisabled(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()

	l, c, err := New(config_slogger.Config{
		Directory: dir,

		Stdout: config_slogger.StdoutConfig{
			Enabled: false,
			Format:  config_slogger.FormatText,
			Level:   slog.LevelInfo,
			Writer:  &buf,
		},

		Files: []config_slogger.FileConfig{
			{
				Name:    "app",
				Enabled: true,
				Format:  config_slogger.FormatText,
				Level:   slog.LevelInfo,
			},
		},
	})
	require.NoError(t, err)

	l.Info("hello stdout???")

	err = c.Close()
	require.NoError(t, err)

	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	assert.Equalf(t, 1, len(files), "expected 1 log files, got %d", len(files))
	assert.Empty(t, buf.Bytes())
}

func TestSlogger_FilesDisabled(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()

	l, c, err := New(config_slogger.Config{
		Directory: dir,

		Stdout: config_slogger.StdoutConfig{
			Enabled: true,
			Format:  config_slogger.FormatText,
			Level:   slog.LevelInfo,
			Writer:  &buf,
		},

		Files: []config_slogger.FileConfig{
			{
				Name:    "app",
				Enabled: false,
				Format:  config_slogger.FormatText,
				Level:   slog.LevelInfo,
			},
		},
	})
	require.NoError(t, err)

	l.Info("hello?????")

	err = c.Close()
	require.NoError(t, err)

	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	assert.Equalf(t, 0, len(files), "expected 0 log files, got %d", len(files))
	assert.NotEmpty(t, buf.Bytes())
}

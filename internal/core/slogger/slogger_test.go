package core_slogger

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogger_BadCases(t *testing.T) {
	testcases := []struct {
		name        string // no file level
		levelFile   string
		levelStdout string
		Directory   string
	}{
		{
			name:        "no file level",
			levelFile:   "",
			levelStdout: "INFO",
			Directory:   t.TempDir(),
		},
		{
			name:        "no stdout level",
			levelFile:   "DEBUG",
			levelStdout: "",
			Directory:   t.TempDir(),
		},
		{
			name:        "no directory",
			levelFile:   "DEBUG",
			levelStdout: "INFO",
			Directory:   "",
		},
		{
			name:        "no levels, no directory",
			levelFile:   "",
			levelStdout: "",
			Directory:   "",
		},
		{
			name:        "incorrect file level name",
			levelFile:   "duck",
			levelStdout: "INFO",
			Directory:   t.TempDir(),
		},
		{
			name:        "incorrect stdout level name",
			levelFile:   "DEBUG",
			levelStdout: "cat",
			Directory:   t.TempDir(),
		},
		{
			name:        "incorrect file directory",
			levelFile:   "DEBUG",
			levelStdout: "INFO",
			Directory:   "\x00/invalid_path",
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.Directory, test.levelStdout, test.levelFile)
			assert.Error(t, err)
		})
	}
}
func TestSlogger_CreateLogs(t *testing.T) {
	dir := t.TempDir()

	logger, err := New(
		dir,
		"INFO",
		"DEBUG",
	)
	require.NoError(t, err)

	defer logger.Close()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Error("error message")

	l := logger.With(
		slog.String("key", "value"),
	)

	l.Debug("debug message")
	l.Info("info message")
	l.Error("error message")

	err = logger.Close()
	require.NoError(t, err)

	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	assert.Equalf(t, 2, len(files), "expected 2 log files, got %d", len(files))
}

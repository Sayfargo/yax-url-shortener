package slogger

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Closer struct {
	files []*os.File
	once  sync.Once
	errs  error
}

var (
	ErrNoLevelOrDirectory = errors.New("missing directory or level")
)

func New(cfg Config) (*slog.Logger, *Closer, error) {

	handler, closer, err := buildHandler(cfg)

	if err != nil {
		return nil, nil, err
	}

	return slog.New(handler), closer, err
}

// MustNew либо создаст экземпляр либо вызовет panic в случае ошибки
func MustNew(cfg Config) (*slog.Logger, *Closer) {
	slogger, closer, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return slogger, closer
}

func buildHandler(cfg Config) (handler slog.Handler, closer *Closer, err error) {

	handlers := make([]slog.Handler, 0)

	files := make([]*os.File, 0)

	defer func() {
		if err != nil {
			for _, file := range files {
				file.Close()
			}
		}
	}()

	if cfg.Stdout.Enabled {
		handlers = append(handlers, newStdoutHandler(cfg.Stdout))
	}

	for _, fileConfig := range cfg.Files {
		if fileConfig.Enabled {

			file, err := setupLogFile(cfg.Directory, fileConfig.Name, fileConfig.Level.String())
			if err != nil {
				return nil, nil, err
			}
			files = append(files, file)
			handlers = append(handlers, newFileHandler(fileConfig, file))
		}
	}

	handler = slog.NewMultiHandler(handlers...)

	return handler, &Closer{
		files: files,
	}, nil

}

func setupLogFile(dir, name, level string) (*os.File, error) {

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to make log dir: %w", err)
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")

	logFilePath := filepath.Join(dir, fmt.Sprintf("%s-%s-%s.log", timestamp, level, name))

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open app log file: %w", err)
	}

	return logFile, nil
}

func (c *Closer) Close() error {

	c.once.Do(func() {
		for _, file := range c.files {
			if err := file.Close(); err != nil {
				c.errs = errors.Join(c.errs, fmt.Errorf("failed to close file with name %s: %w", file.Name(), err))
			}
		}
	})

	return c.errs
}

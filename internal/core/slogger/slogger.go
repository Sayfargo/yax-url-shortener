package core_slogger

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Slogger struct {
	*slog.Logger

	files logFiles
}

type logFiles struct {
	appFile   *os.File
	errorFile *os.File
}

var (
	ErrNoLevelOrDirectory = errors.New("missing directory or level")
)

func New(dir, levelStdout, levelFile string) (*Slogger, error) {

	// Если ничего не передано, то возвращаем ошибку
	if levelStdout == "" || dir == "" || levelFile == "" {
		return nil, ErrNoLevelOrDirectory
	}

	var slogLevelStdout, slogLevelAppFile slog.Level
	// Преобразуем level в верхний регистр и парсим его в slog.Level
	if err := slogLevelStdout.UnmarshalText([]byte(strings.ToUpper(levelStdout))); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stdout level: %w", err)
	}

	if err := slogLevelAppFile.UnmarshalText([]byte(strings.ToUpper(levelFile))); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app log file level: %w", err)
	}

	// Создаем handler для вывода в stdout с заданным уровнем логирования
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevelStdout,
	})

	logFiles, err := makeLogFiles(dir, levelFile)
	if err != nil {
		return nil, fmt.Errorf("failed to make log file: %w", err)
	}

	// Обработчик для вывода в app.log (все детальные события)
	appFileHandler := slog.NewTextHandler(logFiles.appFile, &slog.HandlerOptions{
		Level: slogLevelAppFile,
	})
	// Обработчик для вывода в error.log (ошибки)
	errorFileHandler := slog.NewTextHandler(logFiles.errorFile, &slog.HandlerOptions{
		Level: slog.LevelError,
	})

	multiHandler := slog.NewMultiHandler(stdoutHandler, appFileHandler, errorFileHandler)

	return &Slogger{
		Logger: slog.New(multiHandler),
		files:  logFiles,
	}, nil
}

func makeLogFiles(dir, level string) (logFiles, error) {

	// Создаем директорию для логов, если она не существует
	if err := os.MkdirAll(dir, 0755); err != nil {
		return logFiles{}, fmt.Errorf("failed to make log dir: %w", err)
	}
	// Задаём формат времени для имени файла
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")

	appLogFilePath := filepath.Join(dir, fmt.Sprintf("%s-%s-app.log", timestamp, level))

	errorLogFilePath := filepath.Join(dir, fmt.Sprintf("%s-%s-error.log", timestamp, "ERROR"))

	// После создания файла, открываем его для записи
	applogFile, err := os.OpenFile(appLogFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return logFiles{}, fmt.Errorf("failed to open app log file: %w", err)
	}

	errorLogFile, err := os.OpenFile(errorLogFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		applogFile.Close()
		return logFiles{}, fmt.Errorf("failed to open error log file: %w", err)
	}

	return logFiles{
		appFile:   applogFile,
		errorFile: errorLogFile,
	}, nil
}

// MustNew либо создаст экземпляр либо вызовет panic в случае ошибки
func MustNew(dir, levelStdout, levelFile string) *Slogger {
	slogger, err := New(dir, levelStdout, levelFile)
	if err != nil {
		panic(err)
	}
	return slogger
}

// Обёртка для функции slog.Logger.With
func (l *Slogger) With(args ...any) *Slogger {
	return &Slogger{
		Logger: l.Logger.With(args...),
		files:  l.files,
	}
}

func (l *Slogger) Close() error {
	var errs error

	if l.files.appFile != nil {
		if err := l.files.appFile.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close %s log file: %w", l.files.appFile.Name(), err))
		}
	}

	if l.files.errorFile != nil {
		if err := l.files.errorFile.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close %s log file: %w", l.files.errorFile.Name(), err))
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}

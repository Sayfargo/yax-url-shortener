package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// FileSubscriber - структура, которая реализует интерфейс Subscriber и записывает event в файл
type FileSubscriber struct {
	file *os.File
}

func NewFileSubscriber(filePath string) (*FileSubscriber, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileSubscriber{file: file}, nil
}

func (fs *FileSubscriber) Notify(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json marshal event: %w", err)
	}

	_, err = fs.file.Write(
		append(data, '\n'),
	)

	return err
}

func (fs *FileSubscriber) Close() error {
	return fs.file.Close()
}

// HTTPSubscriber - структура, реализующая интерфейс Subscriber и отправляющая event на указанный URL (удалённый сервер)
type HTTPSubscriber struct {
	// client - HTTP клиент на resty для отправки POST запросов на другой сервер
	client *resty.Client
	url    string
}

func NewHTTPSubscriber(url string) *HTTPSubscriber {
	client := resty.New()
	client.SetRetryCount(5)
	client.SetRetryWaitTime(time.Second * 1)
	client.SetRetryMaxWaitTime(time.Second * 5)

	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return err != nil || r.StatusCode() >= 500
	})

	return &HTTPSubscriber{
		client: client,
		url:    url,
	}
}

func (hs *HTTPSubscriber) Notify(ctx context.Context, event *Event) error {
	_, err := hs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(event).
		Post(hs.url)
	if err != nil {
		return fmt.Errorf("send event via HTTP: %w", err)
	}
	return nil
}

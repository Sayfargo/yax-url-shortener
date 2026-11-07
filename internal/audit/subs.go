package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// FileSubscriber записывает входящие события аудита в файл в формате JSON Lines (JSONL).
// Каждое событие сериализуется в одну строку и завершается символом новой строки.
type FileSubscriber struct {
	file *os.File
}

// NewFileSubscriber открывает или создает файл по указанному пути [filePath]
// в режиме добавления данных (append-only) и возвращает инициализированный [FileSubscriber].
//
// Файл создается с правами доступа 0644. Возвращает ошибку, если не удалось открыть файл.
func NewFileSubscriber(filePath string) (*FileSubscriber, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileSubscriber{file: file}, nil
}

// Notify сериализует событие [Event] в JSON и записывает его в файл.
// Соответствует интерфейсу [Subscriber].
//
// Возвращает обернутую ошибку маршалинга или ошибку ввода-вывода при записи на диск.
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

// Close закрывает дескриптор файла, связанный с [FileSubscriber].
// После вызова этого метода любые последующие вызовы Notify будут завершаться ошибкой.
func (fs *FileSubscriber) Close() error {
	return fs.file.Close()
}

// HTTPSubscriber отправляет входящие события аудита на удаленный сервер
// посредством HTTP POST запросов с телом в формате JSON.
type HTTPSubscriber struct {
	client *resty.Client
	url    string
}

// NewHTTPSubscriber создает и настраивает новый [HTTPSubscriber] для отправки
// событий на указанный [url].
//
// Клиент по умолчанию настроен на 5 повторных попыток (retries) с начальным
// ожиданием в 1 секунду и максимальным в 5 секунд. Повторные попытки выполняются
// в случае сетевых ошибок или если сервер вернул HTTP статус >= 500.
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

// Notify отправляет HTTP POST запрос с сериализованным [Event] на настроенный URL.
// Соответствует интерфейсу [Subscriber].
//
// Запрос учитывает жизненный цикл переданного контекста [context.Context].
// Возвращает ошибку, если запрос не удался после всех повторных попыток.
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

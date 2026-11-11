package url

import (
	"errors"
	"fmt"
)

var (
	// ErrNotExists возвращается репозиторием, если запрашиваемый короткий код
	// отсутствует в базе данных или кэше.
	ErrNotExists = errors.New("row not found in db")

	// ErrConflictShortCode возвращается при попытке сохранить запись с коротким кодом,
	// который уже занят другой ссылкой (нарушение уникальности short_code).
	ErrConflictShortCode = errors.New("short code already exists")

	// ErrUnexpectedType возвращается In-Memory репозиторием, если объект,
	// извлеченный из кэша, не удалось привести к целевому типу [model.ShortenedURL].
	ErrUnexpectedType = errors.New("unexpected data type in cache")

	// ErrNoRows возвращается методом [GetURLs], если у запрашиваемого пользователя
	// не найдено ни одной созданной ссылки.
	ErrNoRows = errors.New("no rows")

	// ErrRowGone возвращается, если запись по короткому коду была найдена,
	// но ссылка имеет статус мягкого (логического) удаления (is_deleted = true).
	ErrRowGone = errors.New("row has been deleted")
)

// BatchConflictError описывает ошибку нарушения уникальности, возникшую в процессе
// пакетной (массовой) вставки данных через метод [CreateBatch].
type BatchConflictError struct {
	// Index хранит порядковый номер (индекс) элемента во входном слайсе,
	// на котором произошел конфликт уникальности.
	Index int
	// Err хранит нижележащую причину конфликта (обычно [ErrConflictShortCode]).
	Err error
}

// Error форматирует и возвращает текстовое представление ошибки [BatchConflictError].
// Соответствует стандартному интерфейсу [error].
func (e *BatchConflictError) Error() string {
	return fmt.Sprintf("conflict at index %d: %v", e.Index, e.Err)
}

// OriginalURLConflictError описывает ошибку дублирования оригинальной ссылки,
// когда пользователь пытается сократить URL, который уже присутствует в системе.
type OriginalURLConflictError struct {
	// ShortCode содержит существующий короткий код, который жестко связан
	// с этим оригинальным URL-адресом.
	ShortCode string
}

// Error форматирует и возвращает текстовое представление ошибки [OriginalURLConflictError].
// Соответствует стандартному интерфейсу [error].
func (o *OriginalURLConflictError) Error() string {
	return fmt.Sprintf("conflict at url: %s", o.ShortCode)
}

package url

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cache "github.com/Sayfargo/yax-url-shortener/internal/core/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
)

// CacheRepository реализует потокобезопасное хранилище сокращенных URL-адресов
// в оперативной памяти (in-memory) на базе пакета [cache.Cache].
//
// Структура дополнительно поддерживает мапу обратного поиска originalMap для быстрой
// проверки уникальности оригинальных (длинных) ссылок без полного сканирования кэша.
type CacheRepository struct {
	c           *cache.Cache
	originalMap map[string]string
	mu          sync.RWMutex
}

// NewInMemoryRepo создает и инициализирует новый экземпляр [CacheRepository].
func NewInMemoryRepo(cache *cache.Cache) *CacheRepository {
	return &CacheRepository{
		c:           cache,
		originalMap: make(map[string]string),
	}
}

// SoftDeleteURLs выполняет мягкое (логическое) удаление пачки коротких кодов,
// принадлежащих пользователю, переключая флаг IsDeleted в состояние true внутри кэша.
//
// Метод предварительно проверяет состояние контекста [context.Context].
// Если хотя бы один короткий код отсутствует в системе, операция прерывается с ошибкой [ErrNotExists].
func (r *CacheRepository) SoftDeleteURLs(ctx context.Context, uid uuid.UUID, shortCodes ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, shortCode := range shortCodes {
		value, err := r.c.Get(shortCode)
		if err != nil {
			if errors.Is(err, cache.ErrNotFound) {
				return ErrNotExists
			}
			return fmt.Errorf("cache storage get err: %w", err)
		}
		if shortenedURL, ok := value.(model.ShortenedURL); ok {
			shortenedURL.IsDeleted = true
			r.c.Set(shortCode, shortenedURL)
		}
	}
	return nil

}

// GetURLs возвращает список всех активных (не удаленных логически) записей [model.ShortenedURL],
// созданных указанным пользователем [uid].
//
// Метод предварительно проверяет состояние контекста [context.Context].
// Для фильтрации используется слепок данных [cache.Cache.Snapshot], благодаря чему метод
// не блокирует пишущие операции (Set) на время фильтрации.
// Возвращает [ErrUnexpectedType], если типы данных в кэше повреждены.
func (r *CacheRepository) GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error) {

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	urls := make([]model.ShortenedURL, 0, 32)

	for _, val := range r.c.Snapshot() {
		if rec, ok := val.(model.ShortenedURL); ok {
			if rec.UserID == uid && !rec.IsDeleted {
				urls = append(urls, rec)
			}
		} else {
			return nil, ErrUnexpectedType
		}
	}

	return urls, nil

}

// Get возвращает оригинальный длинный URL по его короткому буквенно-цифровому коду.
//
// Метод предварительно проверяет состояние контекста [context.Context].
// Использует оптимизированную неблокирующую блокировку чтения (RLock).
// Возвращаемые ошибки:
//   - [ErrNotExists]: если переданный короткий код отсутствует в кэше.
//   - [ErrUnexpectedType]: если тип объекта в кэше не приводится к [model.ShortenedURL].
//   - [ErrRowGone]: если ссылка найдена, но была мягко удалена пользователем.
func (r *CacheRepository) Get(ctx context.Context, shortCode string) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	value, err := r.c.Get(shortCode)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return "", ErrNotExists
		}
		return "", fmt.Errorf("cache storage get err: %w", err)
	}

	shortenedURL, ok := value.(model.ShortenedURL)
	if !ok {
		return "", ErrUnexpectedType
	}

	if shortenedURL.IsDeleted {
		return "", ErrRowGone
	}

	return shortenedURL.OriginalURL, nil
}

// CreateBatch атомарно сохраняет массив сокращенных ссылок в оперативной памяти.
//
// Метод предварительно проверяет состояние контекста [context.Context].
// Перед сохранением производится валидация на наличие дубликатов кодов как внутри
// самого пакета (слайса), так и в глобальном кэше. В случае обнаружения коллизии метод
// возвращает типизированную ошибку [BatchConflictError] с индексом проблемного элемента.
func (r *CacheRepository) CreateBatch(ctx context.Context, shortenedURLs []model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	seenShort := make(map[string]struct{})

	for i, u := range shortenedURLs {

		if _, ok := seenShort[u.ShortCode]; ok {
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		}

		_, err := r.c.Get(u.ShortCode)

		switch {
		case err == nil:
			return &BatchConflictError{
				Index: i,
				Err:   ErrConflictShortCode,
			}
		case !errors.Is(err, cache.ErrNotFound):
			return err
		}

		seenShort[u.ShortCode] = struct{}{}
	}

	for _, u := range shortenedURLs {
		r.c.Set(u.ShortCode, u)
		r.originalMap[u.OriginalURL] = u.ShortCode
	}

	return nil
}

// Create сохраняет новую сокращенную ссылку в оперативной памяти.
//
// Метод предварительно проверяет состояние контекста [context.Context].
// Логика обработки конфликтов уникальности:
//  1. Наличие переданного длинного URL проверяется по мапе [originalMap]. Если ссылка уже
//     существует, метод возвращает ошибку [OriginalURLConflictError], содержащую старый короткий код.
//  2. Если длинный URL уникален, но сгенерированный короткий код (ShortCode) уже занят другим
//     пользователем в кэше, метод возвращает ошибку [ErrConflictShortCode].
func (r *CacheRepository) Create(ctx context.Context, shortenedURL model.ShortenedURL) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.originalMap[shortenedURL.OriginalURL]; ok {
		return &OriginalURLConflictError{
			ShortCode: v,
		}
	}

	_, err := r.c.Get(shortenedURL.ShortCode)

	switch {
	case err == nil:
		return ErrConflictShortCode
	case !errors.Is(err, cache.ErrNotFound):
		return err
	}

	r.c.Set(shortenedURL.ShortCode, shortenedURL)
	r.originalMap[shortenedURL.OriginalURL] = shortenedURL.ShortCode

	return nil

}

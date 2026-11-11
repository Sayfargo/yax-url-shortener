// Package model определяет структуры данных и базовые модели предметной области,
// используемые на всех уровнях приложения (транспортном, сервисном и репозиторном).
package model

import "github.com/google/uuid"

// ShortenedURL представляет полную информацию о сокращенной ссылке в системе.
// Данная структура используется для сохранения записей в базе данных и файловом кэше.
type ShortenedURL struct {
	// UUID — уникальный системный идентификатор записи.
	UUID uuid.UUID `json:"uuid"`
	// ShortCode — короткий уникальный буквенный код ссылки.
	ShortCode string `json:"short_code"`
	// OriginalURL — исходный длинный веб-адрес, на который будет осуществляться перенаправление.
	OriginalURL string `json:"original_url"`
	// UserID — уникальный идентификатор пользователя, создавшего данную ссылку.
	UserID uuid.UUID `json:"user_id"`
	// IsDeleted — флаг мягкого удаления ссылки из системы.
	// Если равен true, то при попытке перехода по ShortCode клиенту вернется статус 410 Gone.
	IsDeleted bool `json:"is_deleted"`
}

// UserURL описывает структуру элемента для формирования списка ссылок,
// возвращаемого конкретному пользователю по его запросу.
type UserURL struct {
	// ShortURL — полный сформированный короткий URL-адрес (включая базовый домен сервиса).
	ShortURL string `json:"short_url"`
	// OriginalURL — исходный длинный веб-адрес, связанный с этой короткой ссылкой.
	OriginalURL string `json:"original_url"`
}

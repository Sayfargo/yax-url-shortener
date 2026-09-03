package model

import "github.com/google/uuid"

type ShortenedURL struct {
	UUID        uuid.UUID `json:"uuid"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	UserID      uuid.UUID `json:"user_Id"`
}

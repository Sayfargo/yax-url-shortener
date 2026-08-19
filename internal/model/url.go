package model

import "github.com/google/uuid"

type ShortenedUrl struct {
	UUID        uuid.UUID `json:"uuid"`
	ShortCode   string    `json:"short_code"`
	OriginalUrl string    `json:"original_url"`
}

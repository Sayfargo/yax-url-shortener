package model

type ShortenedUrl struct {
	UUID        string `json:"uuid"`
	ShortCode   string `json:"short_code"`
	OriginalUrl string `json:"original_url"`
}

package handler

type ShortUrlRequest struct {
	URL string `json:"url"`
}

type ShortUrlResponse struct {
	Result string `json:"result"`
}

type CreateUrlBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type CreateUrlBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

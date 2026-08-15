package handler

type ShortUrlRequest struct {
	URL string `json:"url"`
}

type ShortUrlResponse struct {
	Result string `json:"result"`
}

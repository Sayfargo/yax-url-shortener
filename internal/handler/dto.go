package handler

// ShortURLRequest представляет JSON-тело запроса для создания
// сокращенной ссылки для одного URL-адреса.
//
// Используется в эндпоинтах генерации коротких ссылок.
type ShortURLRequest struct {
	// URL — исходный (длинный) веб-адрес, который необходимо сократить.
	URL string `json:"url"`
}

// ShortURLResponse представляет JSON-ответ сервера, содержащий
// результат успешного сокращения одного URL-адреса.
type ShortURLResponse struct {
	// Result — готовый сокращенный URL-адрес, ведущий на сервис.
	Result string `json:"result"`
}

// CreateURLBatchRequest представляет один элемент в массиве JSON-запроса
// для массового (пакетного) сокращения ссылок.
type CreateURLBatchRequest struct {
	// CorrelationID — произвольный идентификатор строки, передаваемый клиентом
	// для сопоставления этого запроса с элементом ответа.
	CorrelationID string `json:"correlation_id"`
	// OriginalURL — исходный длинный веб-адрес для сокращения.
	OriginalURL string `json:"original_url"`
}

// CreateURLBatchResponse представляет один элемент в массиве JSON-ответа
// при успешном выполнении пакетного сокращения ссылок.
type CreateURLBatchResponse struct {
	// CorrelationID — идентификатор строки, вернувшийся из соответствующего [CreateURLBatchRequest].
	CorrelationID string `json:"correlation_id"`
	// ShortURL — сгенерированный короткий веб-адрес для данной строки.
	ShortURL string `json:"short_url"`
}

// GetURLsResponse описывает элемент структуры ответа, содержащий
// информацию о связи между оригинальным и сокращенным URL-адресами.
//
// Обычно используется в эндпоинтах получения списка ссылок пользователя.
type GetURLsResponse struct {
	// ShortURL — короткий идентификатор или полный адрес ссылки.
	ShortURL string `json:"short_url"`
	// OriginalURL — оригинальный целевой веб-адрес.
	OriginalURL string `json:"original_url"`
}

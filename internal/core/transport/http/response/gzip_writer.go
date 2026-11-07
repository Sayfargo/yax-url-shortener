package httpresponse

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type gzipWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer

	hasHeader bool
	compress  bool
}

// NewGzipWriter оборачивает исходный [http.ResponseWriter] и возвращает [gzipWriter].
// Метод инициализирует внутренний Gzip-компрессор, однако фактическое сжатие
// начнется только после проверки заголовка Content-Type в методе WriteHeader.
func NewGzipWriter(w http.ResponseWriter) *gzipWriter {
	zw := gzip.NewWriter(w)

	return &gzipWriter{
		w:         w,
		zw:        zw,
		hasHeader: false,
	}
}

// Header возвращает карту заголовков (HTTP Headers) нижележащего [http.ResponseWriter].
// Соответствует интерфейсу [http.ResponseWriter].
func (gw *gzipWriter) Header() http.Header {
	return gw.w.Header()
}

// Write записывает данные в поток ответа. Если метод WriteHeader еще не был вызван явным образом,
// Write автоматически вызывает его со статусом [http.StatusOK] перед отправкой байт.
//
// В зависимости от флага компрессии, данные пишутся либо напрямую в сеть, либо через Gzip-сжиматель.
// Соответствует интерфейсу [http.ResponseWriter].
func (gw *gzipWriter) Write(p []byte) (int, error) {

	if !gw.hasHeader {
		gw.WriteHeader(http.StatusOK)
	}

	if !gw.compress {
		return gw.w.Write(p)
	}

	return gw.zw.Write(p)
}

// WriteHeader отправляет клиенту HTTP-статус ответа. Метод автоматически проверяет
// заголовок Content-Type и включает сжатие только для "application/json" и "text/html".
//
// Если сжатие активировано, метод также выставляет исходящий заголовок "Content-Encoding: gzip".
// Статус-коды, не подразумевающие тела ответа ([http.StatusNoContent] и [http.StatusNotModified]),
// полностью игнорируются системой сжатия.
// Соответствует интерфейсу [http.ResponseWriter].
func (gw *gzipWriter) WriteHeader(statusCode int) {
	// Статус коды без тела ответа игноирруем
	if statusCode != http.StatusNoContent &&
		statusCode != http.StatusNotModified {

		// Смотрим есть ли заголовок с содержимом ответа от сервера
		contentType := gw.w.Header().Get("Content-Type")
		// Компрессим только json и html, остальное игнорируем
		if strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/html") {

			gw.compress = true
			// Выставляем для принимающего клиента ответ
			gw.w.Header().Set(
				"Content-Encoding",
				"gzip",
			)

		}
	}

	gw.w.WriteHeader(statusCode)
	gw.hasHeader = true
}

// Close финализирует процесс сжатия и сбрасывает оставшиеся буферы Gzip в сетевой поток.
// Если сжатие контента не было активировано, метод мгновенно возвращает nil без выполнения операций.
func (gw *gzipWriter) Close() error {

	if !gw.compress {
		return nil
	}

	return gw.zw.Close()
}

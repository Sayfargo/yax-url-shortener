package httpresponse

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type gzipWriter struct {
	// Имплементим методы от оригинала
	w  http.ResponseWriter
	zw *gzip.Writer

	// Если заголовок не был проставлен - сами поставим
	hasHeader bool
	// Если Content-Type НЕ application/json или text/html то не сжимаем
	// и пишем через оригинальный responswWriter
	compress bool
}

func NewGzipWriter(w http.ResponseWriter) *gzipWriter {
	zw := gzip.NewWriter(w)

	return &gzipWriter{
		w:         w,
		zw:        zw,
		hasHeader: false,
	}
}

func (gw *gzipWriter) Header() http.Header {
	return gw.w.Header()
}

func (gw *gzipWriter) Write(p []byte) (int, error) {

	if !gw.hasHeader {
		gw.WriteHeader(http.StatusOK)
	}

	if !gw.compress {
		return gw.w.Write(p)
	}

	return gw.zw.Write(p)
}

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

func (gw *gzipWriter) Close() error {

	if !gw.compress {
		return nil
	}

	return gw.zw.Close()
}

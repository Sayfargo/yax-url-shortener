// Модуль httprequest предоставляет вспомогательные инструменты и декораторы
// для обработки, трансформации и декомпрессии входящих HTTP-запросов.
package httprequest

import (
	"compress/gzip"
	"errors"
	"io"
)

type gzipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewGzipReader оборачивает исходный поток [io.ReadCloser] и возвращает инициализированный
// декоратор [gzipReader], готовый для чтения распакованных данных.
//
// Возвращает ошибку, если переданный поток данных не содержит валидного Gzip-заголовка.
func NewGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read считывает распакованные байты во временный буфер [p].
// Соответствует стандартному интерфейсу [io.Reader].
func (gr *gzipReader) Read(p []byte) (n int, err error) {
	return gr.zr.Read(p)
}

// Close последовательно закрывает как внутренний Gzip-декомпрессор,
// так и сам исходный поток данных [io.ReadCloser].
//
// Возможные ошибки закрытия объединяются с помощью [errors.Join].
// Соответствует стандартному интерфейсу [io.Closer].
func (gr *gzipReader) Close() error {
	var errs error

	if err := gr.r.Close(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := gr.zr.Close(); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return errs
	}

	return nil
}

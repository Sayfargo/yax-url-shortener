package core_transport_http_request

import (
	"compress/gzip"
	"errors"
	"io"
)

type gzipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

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

func (gr *gzipReader) Read(p []byte) (n int, err error) {
	return gr.zr.Read(p)
}

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

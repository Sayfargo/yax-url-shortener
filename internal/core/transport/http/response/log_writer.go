package core_transport_http_reponse

import "net/http"

type (
	reponseData struct {
		statusCode int
		bodySize   int
	}
	LoggingResponseWriter struct {
		http.ResponseWriter
		reponseData *reponseData
	}
)

var (
	StatusCodeUninitialized = -1
)

func New(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		reponseData: &reponseData{
			statusCode: StatusCodeUninitialized,
		},
	}
}

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.reponseData.bodySize += size
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.reponseData.statusCode = statusCode
}

func (r *LoggingResponseWriter) GetStatusCode() int {
	if r.reponseData.statusCode == StatusCodeUninitialized {
		panic("no status code set")
	}
	return r.reponseData.statusCode
}

func (r *LoggingResponseWriter) GetBodySize() int {
	return r.reponseData.bodySize
}

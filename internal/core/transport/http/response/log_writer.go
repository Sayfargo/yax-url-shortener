package httpresponse

import "net/http"

type (
	reponseData struct {
		statusCode int
		bodySize   int
	}
	// LoggingResponseWriter реализует интерфейс [http.ResponseWriter] и
	// перехватывает метаданные ответа для последующего логирования или сбора метрик.
	LoggingResponseWriter struct {
		http.ResponseWriter
		reponseData *reponseData
	}
)

var (
	// StatusCodeUninitialized — маркерное значение, указывающее на то,
	// что статус-код HTTP-ответа еще не был выставлен явным или неявным образом.
	StatusCodeUninitialized = -1
)

// New оборачивает исходный [http.ResponseWriter] и возвращает [LoggingResponseWriter]
// с дефолтным неинициализированным статусом ответа.
func New(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		reponseData: &reponseData{
			statusCode: StatusCodeUninitialized,
		},
	}
}

// Write записывает байты ответа в сеть и суммирует их количество для подсчета общего размера тела.
// Соответствует интерфейсу [http.ResponseWriter].
func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.reponseData.bodySize += size
	return size, err
}

// WriteHeader фиксирует отправляемый HTTP-статус в метриках и передает его нижележащему райтеру.
// Соответствует интерфейсу [http.ResponseWriter].
func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.reponseData.statusCode = statusCode
}

// GetStatusCode возвращает зафиксированный HTTP-статус ответа.
//
// Внимание: если на момент вызова метода статус-код еще не был выставлен
// (например, middleware логирования пытается прочитать статус до вызова обработчиков),
// метод вызовет panic с сообщением "no status code set".
func (r *LoggingResponseWriter) GetStatusCode() int {
	if r.reponseData.statusCode == StatusCodeUninitialized {
		panic("no status code set")
	}
	return r.reponseData.statusCode
}

// GetBodySize возвращает суммарный размер тела ответа (в байтах), отправленного клиенту.
func (r *LoggingResponseWriter) GetBodySize() int {
	return r.reponseData.bodySize
}

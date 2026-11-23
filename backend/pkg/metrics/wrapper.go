package metrics

import "net/http"

// statusWriter — обёртка для сохранения кода ответа.
type statusWriter struct {
	http.ResponseWriter     // Интерфейс по работе с HTTP.
	status              int // Код ответа
}

// WriteHeader записывает HTTP код.
func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write записывает в тело ответа сообщение.
func (w *statusWriter) Write(b []byte) (int, error) {
	// Если пользователь не вызывал WriteHeader — статус считается 200.
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.ResponseWriter.Write(b)
}

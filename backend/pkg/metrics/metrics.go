package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MiddlewareMetric middleware для работы с метриками.
type MiddlewareMetric struct {
	service string // Название сервиса

	httpDuration      *prometheus.HistogramVec // Гистограмма(path, method)
	httpRequestsTotal *prometheus.CounterVec   // Счетчик (path, method, status)
}

// NewMiddlewareMetric создаёт новый MiddlewareMetric.
func NewMiddlewareMetric(service string) *MiddlewareMetric {
	return &MiddlewareMetric{
		service: service,
		httpDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Длительность HTTP запросов в секундах.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "path", "method"},
		),
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_request_total",
				Help: "Общее количество HTTP запросов.",
			},
			[]string{"service", "path", "method", "status"},
		),
	}
}

// routePattern пытается получить паттерн chi (низкая кардинальность).
// Если не удалось — используем фактический путь без query.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}

	return r.URL.Path // fallback (может повысить кардинальность)
}

// Middleware — основной middleware для метрик.
// Игнорирует /metrics, чтобы не искажать статистику.
func (m *MiddlewareMetric) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		// Гарантируем корректный статус, даже если WriteHeader не вызывали.
		statusCode := sw.status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		path := routePattern(r)
		method := r.Method

		// Наблюдаем длительность
		m.httpDuration.WithLabelValues(m.service, path, method).Observe(time.Since(start).Seconds())

		// Инкрементируем счётчик
		m.httpRequestsTotal.WithLabelValues(m.service, path, method, strconv.Itoa(statusCode)).Inc()
	})
}

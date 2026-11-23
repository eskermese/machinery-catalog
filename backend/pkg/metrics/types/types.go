package types

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Определяют статусы для метрики.
const (
	Success = "success" // Успех
	Failure = "failure" // Ошибка
)

// Counter представляет собой счетчик.
type Counter interface {
	// Counter представляет собой счетчик Prometheus.
	prometheus.Counter
}

// CounterVec представляет собой вектор счетчиков.
type CounterVec interface {
	// WithLabelValues возвращает счетчик Prometheus с заданными метками.
	WithLabelValues(lvs ...string) prometheus.Counter
	// Describe отправляет описание вектора счетчиков в канал.
	Describe(chan<- *prometheus.Desc)
	// Collect отправляет собранные метрики в канал.
	Collect(chan<- prometheus.Metric)
}

// Histogram представляет собой гистограмму.
type Histogram interface {
	// Histogram представляет собой гистограмму Prometheus.
	prometheus.Histogram
}

// HistogramVec представляет собой вектор гистограмм.
type HistogramVec interface {
	// WithLabelValues возвращает гистограмму Prometheus с заданными метками.
	WithLabelValues(lvs ...string) prometheus.Observer
	// Describe отправляет описание вектора гистограмм в канал.
	Describe(chan<- *prometheus.Desc)
	// Collect отправляет собранные метрики в канал.
	Collect(chan<- prometheus.Metric)
}

// Observer представляет собой наблюдатель.
type Observer interface {
	// Observer представляет собой наблюдатель Prometheus.
	prometheus.Observer
}

// LatencyCallback представляет собой функцию обратного вызова для записи латентности.
type LatencyCallback func(err error)

// GetLatencyCallback возвращает функцию обратного вызова для записи латентности.
func GetLatencyCallback(start time.Time, latency HistogramVec) LatencyCallback {
	return func(err error) {
		elapsed := time.Since(start).Seconds()

		status := Success
		if err != nil {
			status = Failure
		}

		latency.WithLabelValues(status).Observe(elapsed)
	}
}

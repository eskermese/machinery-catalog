package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_routePattern(t *testing.T) {
	type tc struct {
		name     string
		setupReq func() *http.Request
		want     string
	}

	makeReq := func(path string) *http.Request {
		return httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
	}

	tests := []tc{
		{
			name: "использует chi RoutePattern, если он есть",
			setupReq: func() *http.Request {
				req := makeReq("/api/users/42")
				rctx := chi.NewRouteContext()
				rctx.RoutePatterns = []string{"/api/users/{userID}"}
				return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			},
			want: "/api/users/{userID}",
		},
		{
			name:     "fallback на URL.Path, если нет chi RouteContext",
			setupReq: func() *http.Request { return makeReq("/raw/path/123") },
			want:     "/raw/path/123",
		},
		{
			name: "fallback на URL.Path, если chi RoutePattern пустой",
			setupReq: func() *http.Request {
				req := makeReq("/empty/pattern")
				rctx := chi.NewRouteContext()
				rctx.RoutePatterns = nil
				return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			},
			want: "/empty/pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			got := routePattern(req)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Middleware metric tests ---

// helper: создаём изолированный MiddlewareMetric с собственным реестром и service
func newIsolatedMiddlewareMetric(service string) (*MiddlewareMetric, *prometheus.Registry) {
	reg := prometheus.NewRegistry()

	hv := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Длительность HTTP запросов в секундах.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "path", "method"},
	)
	cv := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Общее количество HTTP запросов.",
		},
		[]string{"service", "path", "method", "status"},
	)

	reg.MustRegister(hv, cv)

	m := &MiddlewareMetric{
		service:           service,
		httpDuration:      hv,
		httpRequestsTotal: cv,
	}
	return m, reg
}

func TestMiddleware_RecordsMetrics_WithAndWithoutChi(t *testing.T) {
	type tc struct {
		name         string
		service      string
		method       string
		status       int
		routePattern string // если непусто — кладём в chi.RouteContext
		urlPath      string
	}

	tests := []tc{
		{
			name:         "с chi маршрутным паттерном",
			service:      "api",
			method:       http.MethodDelete,
			status:       http.StatusNoContent,
			routePattern: "/api/haggle/orders/{orderId}",
			urlPath:      "/api/haggle/orders/123",
		},
		{
			name:         "без chi контекста, fallback на URL.Path",
			service:      "api",
			method:       http.MethodGet,
			status:       http.StatusOK,
			routePattern: "",
			urlPath:      "/api/raw/42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, reg := newIsolatedMiddlewareMetric(tt.service)

			// Тестовый handler c явным статусом
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(1 * time.Millisecond) // имитируем работу
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("ok"))
			})

			wrapped := m.Middleware(h)

			req := httptest.NewRequest(tt.method, "http://example.com"+tt.urlPath, nil)
			if tt.routePattern != "" {
				rctx := chi.NewRouteContext()
				rctx.RoutePatterns = []string{tt.routePattern}
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			// Проверяем counter
			pathLabel := tt.urlPath
			if tt.routePattern != "" {
				pathLabel = tt.routePattern
			}
			c, err := m.httpRequestsTotal.GetMetricWithLabelValues(
				tt.service, pathLabel, tt.method, strconv.Itoa(tt.status),
			)
			require.NoError(t, err)

			assert.Equal(t, float64(1), testutil.ToFloat64(c), "http_request_total should be 1")

			// Проверяем гистограмму: *_count == 1.
			// В экспозиции клиент сортирует лейблы по имени => method, path, service
			expected := `# HELP http_request_duration_seconds Длительность HTTP запросов в секундах.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_count{method="` + tt.method + `",path="` + pathLabel + `",service="` + tt.service + `"} 1
`
			err = testutil.CollectAndCompare(
				m.httpDuration,
				strings.NewReader(expected),
				"http_request_duration_seconds_count",
			)
			assert.NoError(t, err, "histogram count mismatch")

			// Дополнительно убеждаемся, что оба коллектора собираются из реестра
			assert.Greater(t, testutil.CollectAndCount(reg, "http_request_total"), 0)
			assert.Greater(t, testutil.CollectAndCount(reg, "http_request_duration_seconds"), 0)
		})
	}
}

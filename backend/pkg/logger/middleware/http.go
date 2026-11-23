package middleware

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// Logger возвращает middleware для логирования запросов.
func Logger(_ logger.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			defer func() {
				remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					remoteIP = r.RemoteAddr
				}

				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}

				statusCode := ww.Status()

				fields := logger.Fields{
					"status_code": statusCode,
					"bytes":       ww.BytesWritten(),
					"duration":    time.Since(t1).String(),
					"remote_ip":   remoteIP,
					"proto":       r.Proto,
					"method":      r.Method,
					"full_path":   fmt.Sprintf("%s://%s%s", scheme, r.Host, routePattern(r)),
				}

				log := logger.FromContext(r.Context()).WithFields(fields)

				switch {
				case statusCode >= http.StatusInternalServerError:
					log.Error("middleware log")
				case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
					log.Warn("middleware log")
				default:
					log.Info("middleware log")
				}
			}()

			h.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}

// routePattern возвращает шаблон маршрута вместо конкретного пути.
//
// Две причины. Первая — секреты: путь публичной ссылки на коммерческое
// предложение целиком состоит из токена, и запись конкретного пути в общий
// лог раздала бы доступ к предложению всем, кто видит логи. Вторая —
// кардинальность: шаблон делает логи группируемыми, тогда как путь
// с идентификатором даёт по строке на каждый запрос.
//
// Шаблон доступен уже после обработки запроса — эта функция вызывается
// в defer, когда chi заполнил контекст маршрутизации. Если шаблона нет
// (запрос не попал ни в один маршрут), пишется путь: секрета в нём быть
// не может, потому что такой ссылки сервис не выдавал.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}

	return r.URL.Path
}

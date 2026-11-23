package v1

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	httpx "github.com/eskermese/machinery-catalog/backend/pkg/httpx"
)

// Предел частоты публичного расчёта лизинга.
//
// Почему предел вообще нужен: расчёт заметно дороже отдачи карточки — чтение
// машины по слагу, чтение программы, чтение курса и построение графика,
// который на шестидесяти месяцах строится шестьюдесятью итерациями
// с десятичной арифметикой. Ручка при этом публичная и не требует ни токена,
// ни заявки, то есть ничего не стоит вызывающему.
//
// Почему тридцать в минуту: человек, двигающий ползунки калькулятора, шлёт
// запросы очередями по несколько штук и между ними читает результат. Тридцать
// в минуту — это один раз в две секунды подряд весь минутный интервал; такой
// темп не выдерживает человек, но в него укладывается любая серия движений
// ползунка. Перебор параметров упирается в предел за секунды.
//
// Почему по IP, а не по номеру, как у заявок: запрос не несёт никакого
// признака обратившегося — ни номера, ни токена. IP здесь единственный ключ,
// и его известная слабость (за одним адресом сидит целый офис) выбрана
// осознанно: цена ошибки — отказ в справочном расчёте с предложением
// повторить, а не потеря заявки.
const (
	calculateRateLimit  = 30
	calculateRateWindow = time.Minute
)

// rateLimiter считает обращения с адреса в скользящем окне.
//
// В памяти процесса, а не в кэше: предел защищает процессорное время самого
// процесса, и при нескольких репликах каждая справедливо считает свою
// нагрузку. Общий счётчик в кэше стоил бы обращения по сети на каждый расчёт —
// то есть добавил бы работы ровно там, где мы её ограничиваем.
type rateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time // Моменты обращений по адресу
	limit   int                    // Сколько обращений допускается в окне
	window  time.Duration          // Длина окна
	lastGC  time.Time              // Момент последней уборки
	gcEvery time.Duration          // Как часто убирать остывшие адреса
}

// newRateLimiter создаёт ограничитель частоты.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		hits:    make(map[string][]time.Time),
		limit:   limit,
		window:  window,
		gcEvery: window,
	}
}

// allow регистрирует обращение и сообщает, укладывается ли оно в предел.
func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.collect(now)

	fresh := withinWindow(l.hits[key], now.Add(-l.window))
	if len(fresh) >= l.limit {
		// Отклонённое обращение в окно НЕ записывается: иначе непрерывный
		// перебор продлевал бы себе наказание бесконечно, и адрес не вышел бы
		// из блокировки, даже перестав нарушать.
		l.hits[key] = fresh

		return false
	}

	l.hits[key] = append(fresh, now)

	return true
}

// collect выбрасывает адреса, не обращавшиеся дольше окна.
//
// Без уборки карта росла бы на каждый новый адрес и никогда не уменьшалась:
// у публичной ручки число разных адресов не ограничено ничем.
func (l *rateLimiter) collect(now time.Time) {
	if now.Sub(l.lastGC) < l.gcEvery {
		return
	}

	l.lastGC = now
	threshold := now.Add(-l.window)

	for key, moments := range l.hits {
		fresh := withinWindow(moments, threshold)
		if len(fresh) == 0 {
			delete(l.hits, key)

			continue
		}

		l.hits[key] = fresh
	}
}

// withinWindow отбрасывает моменты старше границы окна.
func withinWindow(moments []time.Time, threshold time.Time) []time.Time {
	fresh := moments[:0]

	for _, moment := range moments {
		if moment.After(threshold) {
			fresh = append(fresh, moment)
		}
	}

	return fresh
}

// RateLimitMiddleware ограничивает частоту обращений с одного адреса.
//
// Ставится на конкретный маршрут, а не на весь публичный роутер: остальные
// публичные ручки отдают готовые данные и такой защиты не требуют, а общий
// предел мешал бы обычному обходу каталога.
func RateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := newRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.allow(clientIP(r), time.Now()) {
				httpx.LogRequestError(r, "превышена частота обращений", entity.ErrTooManyRequests, nil)
				httpx.WriteError(w, entity.ErrTooManyRequests)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP возвращает адрес обратившегося.
//
// RemoteAddr, а не заголовок напрямую: chi middleware.RealIP уже подставил
// в него X-Forwarded-For или X-Real-IP, и читать заголовок второй раз значило
// бы завести второе место, где решается, кому верить.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

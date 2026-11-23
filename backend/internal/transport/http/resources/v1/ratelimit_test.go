package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLimitNow фиксированный момент отсчёта окна.
var testLimitNow = time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

// TestRateLimiter_AllowsWithinLimit фиксирует, что предел пропускает ровно
// заявленное число обращений и отвергает следующее.
func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(3, time.Minute)

	for i := range 3 {
		assert.True(t, limiter.allow("1.2.3.4", testLimitNow), "обращение %d в пределах", i+1)
	}

	assert.False(t, limiter.allow("1.2.3.4", testLimitNow), "четвёртое отвергается")
}

// TestRateLimiter_WindowSlides фиксирует, что окно скользит: обращения,
// вышедшие за его границу, перестают считаться.
func TestRateLimiter_WindowSlides(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(2, time.Minute)

	require.True(t, limiter.allow("1.2.3.4", testLimitNow))
	require.True(t, limiter.allow("1.2.3.4", testLimitNow))
	require.False(t, limiter.allow("1.2.3.4", testLimitNow))

	// Спустя окно предыдущие обращения не считаются.
	later := testLimitNow.Add(time.Minute + time.Second)

	assert.True(t, limiter.allow("1.2.3.4", later))
}

// TestRateLimiter_RejectedDoesNotExtendBan фиксирует, что отклонённое
// обращение не записывается в окно.
//
// Иначе непрерывный перебор продлевал бы себе наказание бесконечно, и адрес
// не вышел бы из блокировки, даже перестав нарушать.
func TestRateLimiter_RejectedDoesNotExtendBan(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(1, time.Minute)

	require.True(t, limiter.allow("1.2.3.4", testLimitNow))

	// Долбит весь интервал.
	for offset := time.Second; offset < time.Minute; offset += time.Second {
		require.False(t, limiter.allow("1.2.3.4", testLimitNow.Add(offset)))
	}

	// Ровно через окно после ЕДИНСТВЕННОГО засчитанного обращения адрес
	// снова проходит, хотя всё это время долбил.
	assert.True(t, limiter.allow("1.2.3.4", testLimitNow.Add(time.Minute+time.Second)))
}

// TestRateLimiter_AddressesAreIndependent фиксирует, что предел считается
// по адресам раздельно.
func TestRateLimiter_AddressesAreIndependent(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(1, time.Minute)

	require.True(t, limiter.allow("1.2.3.4", testLimitNow))
	require.False(t, limiter.allow("1.2.3.4", testLimitNow))

	assert.True(t, limiter.allow("5.6.7.8", testLimitNow), "чужой адрес не наказан")
}

// TestRateLimiter_CollectsStaleAddresses фиксирует уборку остывших адресов:
// без неё карта росла бы на каждый новый адрес и никогда не уменьшалась.
func TestRateLimiter_CollectsStaleAddresses(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(10, time.Minute)

	for _, addr := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		require.True(t, limiter.allow(addr, testLimitNow))
	}

	require.Len(t, limiter.hits, 3)

	// Спустя окно новый адрес запускает уборку остальных.
	later := testLimitNow.Add(time.Minute + time.Second)
	require.True(t, limiter.allow("4.4.4.4", later))

	assert.Len(t, limiter.hits, 1, "остались только свежие")
	assert.Contains(t, limiter.hits, "4.4.4.4")
}

// TestCalculateRateLimitIsSane фиксирует выбранный предел: он должен
// пропускать серию движений ползунка и упираться на переборе.
func TestCalculateRateLimitIsSane(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30, calculateRateLimit)
	assert.Equal(t, time.Minute, calculateRateWindow)

	limiter := newRateLimiter(calculateRateLimit, calculateRateWindow)

	// Серия из десяти движений ползунка подряд проходит целиком.
	for i := range 10 {
		require.True(t, limiter.allow("1.2.3.4", testLimitNow.Add(time.Duration(i)*100*time.Millisecond)),
			"движение ползунка %d не должно отвергаться", i+1)
	}
}

// TestClientIP фиксирует разбор адреса обратившегося.
//
// Читается RemoteAddr, а не заголовок: chi middleware.RealIP уже подставил
// в него X-Forwarded-For, и читать заголовок второй раз значило бы завести
// второе место, где решается, кому верить.
func TestClientIP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		remoteAddr string
		exp        string
	}{
		{name: "адрес с портом", remoteAddr: "203.0.113.7:54321", exp: "203.0.113.7"},
		{name: "ipv6 с портом", remoteAddr: "[2001:db8::1]:443", exp: "2001:db8::1"},
		{name: "без порта", remoteAddr: "203.0.113.7", exp: "203.0.113.7"},
		{name: "пустой", remoteAddr: "", exp: ""},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = c.remoteAddr

			assert.Equal(t, c.exp, clientIP(r))
		})
	}
}

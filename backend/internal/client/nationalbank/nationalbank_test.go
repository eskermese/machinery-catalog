package nationalbank

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// realResponse ответ Нацбанка, записанный с живой выдачи 30.08.2026.
//
// Сокращён до трёх валют: разбору всё равно, сколько их в списке, а полный
// ответ на полсотни валют в тесте только мешал бы читать.
const realResponse = `<?xml version="1.0" encoding="utf-8"?>
<rates>
    <title>Official exchange rates of National Bank of Republic Kazakhstan</title>
    <date>30.08.2026</date>
    <item>
        <fullname>ЕВРО</fullname>
        <title>EUR</title>
        <description>543.21</description>
        <quant>1</quant>
    </item>
    <item>
        <fullname>ДОЛЛАР США</fullname>
        <title>USD</title>
        <description>464.77</description>
        <quant>1</quant>
    </item>
    <item>
        <fullname>ЯПОНСКАЯ ЙЕНА</fullname>
        <title>JPY</title>
        <description>291.00</description>
        <quant>100</quant>
    </item>
</rates>`

// emptyResponse ответ на дату за горизонтом публикации.
//
// Записан с живой выдачи по 01.01.2027: источник отвечает двумя сотнями,
// эхом повторяет запрошенную дату и не даёт ни одной валюты. Это и есть
// «курса на эту дату нет» — не отказ и не поломка разбора.
const emptyResponse = `<?xml version="1.0" encoding="utf-8"?>
<rates>
    <title>Official exchange rates of National Bank of Republic Kazakhstan</title>
    <date>01.01.2027</date>
</rates>`

// testMoment момент, на который спрашивается курс.
var testMoment = time.Date(2026, 8, 30, 14, 0, 0, 0, time.UTC)

// serve поднимает источник, отдающий записанный ответ.
func serve(t *testing.T, body string, status int) (*Provider, *string) {
	t.Helper()

	var requestedDate string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedDate = r.URL.Query().Get("fdate")

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return New(server.URL, time.Second), &requestedDate
}

// TestProvider_FetchRate разбирает живой ответ источника.
func TestProvider_FetchRate(t *testing.T) {
	t.Parallel()

	provider, requested := serve(t, realResponse, http.StatusOK)

	rate, err := provider.FetchRate(context.Background(), "USD", "KZT", testMoment)
	require.NoError(t, err)

	assert.Equal(t, "30.08.2026", *requested, "дата уходит в запрос в формате источника")
	assert.Equal(t, "USD", rate.BaseCurrency)
	assert.Equal(t, "KZT", rate.QuoteCurrency)
	assert.Equal(t, Source, rate.Source, "источник записывается в курс и отличает его от демо-данных")
	assert.Equal(t, "464.77", rate.Rate.String())
	assert.Equal(t, testMoment, rate.FetchedAt)

	// Дата берётся ИЗ ОТВЕТА, а не из запроса: источник сам решает, за какие
	// сутки отдал курс.
	assert.Equal(t, time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC), rate.RateDate)
}

// TestProvider_FetchRateDividesByQuant фиксирует приведение к одной единице.
//
// В нынешней выдаче Нацбанка quant у всех валют равен единице, поэтому
// проверка идёт на составленном ответе: правило должно держаться само,
// а не потому, что источник сегодня присылает единицы. Без деления курс
// иены оказался бы завышен в сто раз.
func TestProvider_FetchRateDividesByQuant(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, realResponse, http.StatusOK)

	rate, err := provider.FetchRate(context.Background(), "JPY", "KZT", testMoment)
	require.NoError(t, err)

	assert.Equal(t, "2.91", rate.Rate.String(), "291 тенге за сто иен — это 2.91 за одну")
}

// TestProvider_FetchRateEmptyIsNotFound фиксирует главный крайний случай:
// на дату за горизонтом публикации источник отдаёт 200 и пустой список.
//
// Это отсутствие курса, а не поломка: воркер оставит в базе последний
// известный курс, а не уронит синхронизацию.
func TestProvider_FetchRateEmptyIsNotFound(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, emptyResponse, http.StatusOK)

	_, err := provider.FetchRate(context.Background(), "USD", "KZT", testMoment)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
}

// TestProvider_FetchRateUnknownCurrency — валюты нет в выдаче.
func TestProvider_FetchRateUnknownCurrency(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, realResponse, http.StatusOK)

	_, err := provider.FetchRate(context.Background(), "XYZ", "KZT", testMoment)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
}

// TestProvider_FetchRateRejectsForeignQuote отвергает пару, которой источник
// не знает.
//
// Нацбанк публикует стоимость валют В ТЕНГЕ и ничего больше. Отказ здесь
// отличается от «курс не найден»: повторять такой запрос завтра бессмысленно.
func TestProvider_FetchRateRejectsForeignQuote(t *testing.T) {
	t.Parallel()

	provider, requested := serve(t, realResponse, http.StatusOK)

	_, err := provider.FetchRate(context.Background(), "EUR", "USD", testMoment)

	require.ErrorIs(t, err, entity.ErrFXPairNotSupported)
	assert.Empty(t, *requested, "до сети такой запрос не доходит")
}

// TestProvider_FetchRateHTTPFailure заворачивает отказ источника.
func TestProvider_FetchRateHTTPFailure(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, "<html>502</html>", http.StatusBadGateway)

	_, err := provider.FetchRate(context.Background(), "USD", "KZT", testMoment)

	require.Error(t, err)
	assert.NotErrorIs(t, err, entity.ErrFXRateNotFound,
		"недоступный источник — это не «курса нет»: повторить стоит")
}

// TestProvider_FetchRateBrokenXML — испорченный ответ не выдаётся за пустой.
func TestProvider_FetchRateBrokenXML(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, "<rates><item><title>USD", http.StatusOK)

	_, err := provider.FetchRate(context.Background(), "USD", "KZT", testMoment)

	require.Error(t, err)
	assert.NotErrorIs(t, err, entity.ErrFXRateNotFound)
}

// TestProvider_FetchRateRejectsBadRate отвергает нечисловой и неположительный
// курс: подставить такой в цену хуже, чем не подставить ничего.
func TestProvider_FetchRateRejectsBadRate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{
			name: "не число",
			body: `<rates><date>30.08.2026</date><item><title>USD</title>` +
				`<description>н/д</description><quant>1</quant></item></rates>`,
		},
		{
			name: "ноль",
			body: `<rates><date>30.08.2026</date><item><title>USD</title>` +
				`<description>0</description><quant>1</quant></item></rates>`,
		},
		{
			// Нулевой quant уронил бы приложение делением на ноль, поэтому
			// проверка стоит ДО деления.
			name: "нулевой quant",
			body: `<rates><date>30.08.2026</date><item><title>USD</title>` +
				`<description>464.77</description><quant>0</quant></item></rates>`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			provider, _ := serve(t, c.body, http.StatusOK)

			_, err := provider.FetchRate(context.Background(), "USD", "KZT", testMoment)

			require.Error(t, err)
		})
	}
}

// TestProvider_FetchRateBrokenDate — дата ответа не разобралась.
func TestProvider_FetchRateBrokenDate(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, `<rates><date>вчера</date><item><title>USD</title>`+
		`<description>464.77</description><quant>1</quant></item></rates>`, http.StatusOK)

	_, err := provider.FetchRate(context.Background(), "USD", "KZT", testMoment)

	require.Error(t, err)
}

// TestProvider_FetchRateHonoursContext — отменённый контекст не уходит в сеть.
func TestProvider_FetchRateHonoursContext(t *testing.T) {
	t.Parallel()

	provider, _ := serve(t, realResponse, http.StatusOK)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.FetchRate(ctx, "USD", "KZT", testMoment)

	require.Error(t, err)
}

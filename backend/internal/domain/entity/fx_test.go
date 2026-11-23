package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRateDate дата курса в тестах.
var testRateDate = time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)

// newTestRate собирает курс источника для тестов.
func newTestRate(t *testing.T, rate string) *FXRate {
	t.Helper()

	value, err := decimal.NewFromString(rate)
	require.NoError(t, err)

	return NewFXRate("USD", "KZT", "nationalbank", value, testRateDate, testRateDate)
}

// TestFXRate_Applied фиксирует расчёт применённого курса.
//
// Наценка накручивается мультипликативно: applied = rate * (1 + bps/10000).
// Нулевая наценка обязана возвращать курс источника без изменений — иначе
// «продаём по курсу источника» отличалось бы от него в младшем разряде.
func TestFXRate_Applied(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		rate      string
		markupBps int
		expected  string
	}{
		{
			// 522.34 * 1.0 = 522.34
			name: "нулевая наценка отдаёт курс источника", rate: "522.34", markupBps: 0,
			expected: "522.340000",
		},
		{
			// 522.34 * 1.02 = 532.7868
			name: "наценка 200 bps", rate: "522.34", markupBps: 200,
			expected: "532.786800",
		},
		{
			// 522.34 * 1.015 = 522.34 + 7.8351 = 530.1751
			name: "наценка 150 bps", rate: "522.34", markupBps: 150,
			expected: "530.175100",
		},
		{
			// 100 * 1.0001 = 100.01
			name: "наценка в один базисный пункт", rate: "100", markupBps: 1,
			expected: "100.010000",
		},
		{
			// 522.34 * 2.0 = 1044.68
			name: "наценка 100 процентов", rate: "522.34", markupBps: 10000,
			expected: "1044.680000",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			applied := newTestRate(t, c.rate).Applied(c.markupBps)

			assert.Equal(t, c.expected, applied.StringFixed(FXRateScale))
		})
	}
}

// TestFXRate_Applied_NilIsZero фиксирует, что пустой курс не паникует.
func TestFXRate_Applied_NilIsZero(t *testing.T) {
	t.Parallel()

	var rate *FXRate

	assert.True(t, rate.Applied(200).IsZero())
}

// TestFXRate_Validate фиксирует проверки курса источника.
func TestFXRate_Validate(t *testing.T) {
	t.Parallel()

	valid := newTestRate(t, "522.34")

	cases := []struct {
		name   string
		mutate func(*FXRate)
		expErr error
	}{
		{name: "корректный курс", mutate: func(*FXRate) {}},
		{
			name:   "нулевой курс",
			mutate: func(r *FXRate) { r.Rate = decimal.Zero },
			expErr: ErrFXRateInvalid,
		},
		{
			name:   "отрицательный курс",
			mutate: func(r *FXRate) { r.Rate = decimal.NewFromInt(-1) },
			expErr: ErrFXRateInvalid,
		},
		{
			name:   "код валюты в нижнем регистре",
			mutate: func(r *FXRate) { r.BaseCurrency = "usd" },
			expErr: ErrCurrencyCodeInvalid,
		},
		{
			name:   "код валюты из двух букв",
			mutate: func(r *FXRate) { r.QuoteCurrency = "KZ" },
			expErr: ErrCurrencyCodeInvalid,
		},
		{
			name:   "одинаковые валюты пары",
			mutate: func(r *FXRate) { r.QuoteCurrency = r.BaseCurrency },
			expErr: ErrCurrencyPairInvalid,
		},
		{
			name:   "источник не указан",
			mutate: func(r *FXRate) { r.Source = "" },
			expErr: ErrFXSourceInvalid,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rate := *valid
			c.mutate(&rate)

			err := rate.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestFXRate_Validate_Nil фиксирует, что пустой курс не проходит проверку.
func TestFXRate_Validate_Nil(t *testing.T) {
	t.Parallel()

	var rate *FXRate

	require.ErrorIs(t, rate.Validate(), ErrNilPointer)
}

// TestFXMarkup_Validate фиксирует проверки наценки.
func TestFXMarkup_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		markup *FXMarkup
		expErr error
	}{
		{
			name:   "корректная наценка",
			markup: NewFXMarkup("USD", "KZT", 200, testRateDate, nil, testRateDate),
		},
		{
			name:   "нулевая наценка допустима",
			markup: NewFXMarkup("USD", "KZT", 0, testRateDate, nil, testRateDate),
		},
		{
			name:   "отрицательная наценка",
			markup: NewFXMarkup("USD", "KZT", -1, testRateDate, nil, testRateDate),
			expErr: ErrFXMarkupInvalid,
		},
		{
			name:   "неверный код валюты",
			markup: NewFXMarkup("US", "KZT", 200, testRateDate, nil, testRateDate),
			expErr: ErrCurrencyCodeInvalid,
		},
		{
			name:   "одинаковые валюты пары",
			markup: NewFXMarkup("USD", "USD", 200, testRateDate, nil, testRateDate),
			expErr: ErrCurrencyPairInvalid,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := c.markup.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestFXMarkup_Validate_Nil фиксирует, что пустая наценка не проходит проверку.
func TestFXMarkup_Validate_Nil(t *testing.T) {
	t.Parallel()

	var markup *FXMarkup

	require.ErrorIs(t, markup.Validate(), ErrNilPointer)
}

// TestNewAppliedFXRate фиксирует состав применённого курса: он несёт не только
// число, но и дату курса с наценкой — ровно то, что уходит в КП.
func TestNewAppliedFXRate(t *testing.T) {
	t.Parallel()

	applied := NewAppliedFXRate(newTestRate(t, "522.34"), 200)

	assert.Equal(t, "532.786800", applied.String())
	assert.Equal(t, testRateDate, applied.RateDate)
	assert.Equal(t, 200, applied.MarkupBps)
	assert.False(t, applied.IsZero())
}

// TestNewAppliedFXRate_Nil фиксирует, что отсутствие курса даёт пустое значение.
func TestNewAppliedFXRate_Nil(t *testing.T) {
	t.Parallel()

	applied := NewAppliedFXRate(nil, 200)

	assert.True(t, applied.IsZero())
}

// TestAppliedFXRate_String фиксирует numeric-совместимое представление:
// хвост дополняется до scale колонки, чтобы записанное и прочитанное значение
// совпадали посимвольно.
func TestAppliedFXRate_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rate string
		exp  string
	}{
		{name: "целое дополняется нулями", rate: "500", exp: "500.000000"},
		{name: "два знака дополняются", rate: "522.34", exp: "522.340000"},
		{name: "шесть знаков сохраняются", rate: "522.123456", exp: "522.123456"},
		{name: "седьмой знак округляется", rate: "522.1234565", exp: "522.123457"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			value, err := decimal.NewFromString(c.rate)
			require.NoError(t, err)

			applied := AppliedFXRate{Rate: value, RateDate: testRateDate}

			assert.Equal(t, c.exp, applied.String())
		})
	}
}

// TestAppliedFXRate_ConvertPriceCents фиксирует единственное правило пересчёта
// цены в тиыны: по нему считает и витрина, и выпуск предложения.
func TestAppliedFXRate_ConvertPriceCents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		rate          string
		priceUSDCents int64
		exp           int64
	}{
		{name: "целый курс", rate: "500", priceUSDCents: 100, exp: 50_000},
		{name: "дробный курс", rate: "522.34", priceUSDCents: 8_950_000, exp: 4_674_943_000},
		{name: "округление вверх", rate: "1.5", priceUSDCents: 1, exp: 2},
		{name: "округление половины от нуля", rate: "0.5", priceUSDCents: 1, exp: 1},
		{name: "нулевая цена", rate: "522.34", priceUSDCents: 0, exp: 0},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			applied := AppliedFXRate{Rate: decimal.RequireFromString(c.rate)}

			assert.Equal(t, c.exp, applied.ConvertPriceCents(c.priceUSDCents))
		})
	}
}

// TestAppliedFXRate_ConvertPriceCents_ZeroRate фиксирует, что пустой курс
// обнуляет результат: применять его нельзя, и потребитель обязан отсечь
// такой курс по IsZero до пересчёта.
func TestAppliedFXRate_ConvertPriceCents_ZeroRate(t *testing.T) {
	t.Parallel()

	var applied AppliedFXRate

	assert.True(t, applied.IsZero())
	assert.Equal(t, int64(0), applied.ConvertPriceCents(8_950_000))
}

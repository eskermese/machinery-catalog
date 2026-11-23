package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testQuoteTime момент операций в тестах предложений.
var testQuoteTime = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

// validQuote собирает пригодное к сохранению предложение.
func validQuote(t *testing.T) *Quote {
	t.Helper()

	token, err := NewQuoteToken()
	require.NoError(t, err)

	return &Quote{
		PublicToken:            token,
		MachineID:              "7c9e6679-7425-40de-944b-e07fc1f90ae7",
		PriceCNYCents:          10_000_000,
		FXRate:                 decimal.NewFromInt(500),
		FXRateDate:             testQuoteTime,
		FXMarkupBps:            200,
		AssetCostKZTCents:      5_000_000_000,
		ScheduleType:           LeasingScheduleAnnuity,
		DownPaymentBps:         2000,
		TermMonths:             36,
		RateBps:                1800,
		FinancedAmountKZTCents: 4_000_000_000,
		ExpiresAt:              testQuoteTime.AddDate(0, 0, 14),
		CreatedAt:              testQuoteTime,
	}
}

// TestNewQuoteToken фиксирует свойства публичного токена: достаточная длина
// и неповторяемость. Токен — единственная защита предложения, поэтому короткий
// или предсказуемый означал бы открытую ссылку.
func TestNewQuoteToken(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, 100)

	for range 100 {
		token, err := NewQuoteToken()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(token), MinQuoteTokenLength,
			"токен не короче границы CHECK-а схемы")
		assert.False(t, seen[token], "токены не повторяются")

		seen[token] = true
	}
}

// TestQuote_IsExpired фиксирует, что истечение вычисляется от момента,
// а не хранится флагом: флаг пришлось бы кому-то переключать, и до этого
// мгновения просроченное предложение выглядело бы действующим.
func TestQuote_IsExpired(t *testing.T) {
	t.Parallel()

	quote := &Quote{ExpiresAt: testQuoteTime}

	cases := []struct {
		name string
		at   time.Time
		exp  bool
	}{
		{name: "за час до срока", at: testQuoteTime.Add(-time.Hour), exp: false},
		{name: "за секунду до срока", at: testQuoteTime.Add(-time.Second), exp: false},
		{name: "ровно в момент срока", at: testQuoteTime, exp: true},
		{name: "через секунду после", at: testQuoteTime.Add(time.Second), exp: true},
		{name: "через неделю после", at: testQuoteTime.AddDate(0, 0, 7), exp: true},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, quote.IsExpired(c.at))
		})
	}
}

// TestQuote_IsExpired_Nil фиксирует, что пустое предложение не паникует.
func TestQuote_IsExpired_Nil(t *testing.T) {
	t.Parallel()

	var quote *Quote

	assert.False(t, quote.IsExpired(testQuoteTime))
}

// TestQuote_MarkOpened фиксирует однократность отметки первого открытия.
func TestQuote_MarkOpened(t *testing.T) {
	t.Parallel()

	quote := &Quote{}

	require.True(t, quote.MarkOpened(testQuoteTime), "первое открытие требует сохранения")
	require.NotNil(t, quote.FirstOpenedAt)
	assert.Equal(t, testQuoteTime, *quote.FirstOpenedAt)

	// Второе открытие часом позже.
	later := testQuoteTime.Add(time.Hour)

	assert.False(t, quote.MarkOpened(later), "повторное открытие сохранять нечего")
	assert.Equal(t, testQuoteTime, *quote.FirstOpenedAt,
		"момент первого открытия не перезаписывается")
}

// TestQuote_MarkOpened_Nil фиксирует, что пустое предложение не паникует.
func TestQuote_MarkOpened_Nil(t *testing.T) {
	t.Parallel()

	var quote *Quote

	assert.False(t, quote.MarkOpened(testQuoteTime))
}

// TestQuote_Validate фиксирует проверки предложения.
func TestQuote_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Quote)
		expErr error
	}{
		{name: "корректное предложение", mutate: func(*Quote) {}},
		{
			name:   "короткий токен",
			mutate: func(q *Quote) { q.PublicToken = "short" },
			expErr: ErrQuoteTokenInvalid,
		},
		{
			name:   "нет машины",
			mutate: func(q *Quote) { q.MachineID = "" },
			expErr: ErrQuoteMachineRequired,
		},
		{
			name:   "неизвестный тип графика",
			mutate: func(q *Quote) { q.ScheduleType = "balloon" },
			expErr: ErrQuoteScheduleTypeInvalid,
		},
		{
			name:   "отрицательная цена",
			mutate: func(q *Quote) { q.PriceCNYCents = -1 },
			expErr: ErrQuoteAmountInvalid,
		},
		{
			name:   "нулевой курс",
			mutate: func(q *Quote) { q.FXRate = decimal.Zero },
			expErr: ErrQuoteFXInvalid,
		},
		{
			name:   "отрицательная наценка",
			mutate: func(q *Quote) { q.FXMarkupBps = -1 },
			expErr: ErrQuoteFXInvalid,
		},
		{
			name:   "взнос больше 100 процентов",
			mutate: func(q *Quote) { q.DownPaymentBps = 10001 },
			expErr: ErrQuoteDownPaymentInvalid,
		},
		{
			name:   "нулевой срок",
			mutate: func(q *Quote) { q.TermMonths = 0 },
			expErr: ErrQuoteTermInvalid,
		},
		{
			name:   "отрицательная ставка",
			mutate: func(q *Quote) { q.RateBps = -1 },
			expErr: ErrQuoteTermInvalid,
		},
		{
			// Сумма финансирования — стоимость за вычетом взноса.
			name:   "финансирование больше стоимости",
			mutate: func(q *Quote) { q.FinancedAmountKZTCents = q.AssetCostKZTCents + 1 },
			expErr: ErrQuoteAmountInvalid,
		},
		{
			name:   "выкуп больше финансирования",
			mutate: func(q *Quote) { q.ResidualValueKZTCents = q.FinancedAmountKZTCents + 1 },
			expErr: ErrQuoteResidualInvalid,
		},
		{
			name:   "выкуп равен финансированию допустим",
			mutate: func(q *Quote) { q.ResidualValueKZTCents = q.FinancedAmountKZTCents },
		},
		{
			name:   "отрицательная комиссия",
			mutate: func(q *Quote) { q.CommissionKZTCents = -1 },
			expErr: ErrQuoteAmountInvalid,
		},
		{
			name:   "срок годности в прошлом",
			mutate: func(q *Quote) { q.ExpiresAt = q.CreatedAt.Add(-time.Hour) },
			expErr: ErrQuoteExpiresAtInvalid,
		},
		{
			name:   "срок годности равен моменту выпуска",
			mutate: func(q *Quote) { q.ExpiresAt = q.CreatedAt },
			expErr: ErrQuoteExpiresAtInvalid,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quote := validQuote(t)
			c.mutate(quote)

			err := quote.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestQuote_Validate_Nil фиксирует отказ на пустом предложении.
func TestQuote_Validate_Nil(t *testing.T) {
	t.Parallel()

	var quote *Quote

	require.ErrorIs(t, quote.Validate(), ErrNilPointer)
}

// TestLeasingScheduleType фиксирует закрытый перечень типов графика.
func TestLeasingScheduleType(t *testing.T) {
	t.Parallel()

	assert.True(t, LeasingScheduleAnnuity.IsValid())
	assert.True(t, LeasingScheduleDifferentiated.IsValid())
	assert.False(t, LeasingScheduleType("balloon").IsValid())
	assert.Equal(t, "annuity", LeasingScheduleAnnuity.String())
}

// TestLeasingProgram_CheckBounds фиксирует проверку параметров сделки против
// границ программы. Каждое нарушение — своя ошибка: клиенту нужно понять,
// что именно поправить.
func TestLeasingProgram_CheckBounds(t *testing.T) {
	t.Parallel()

	program := &LeasingProgram{
		IsActive:          true,
		MinDownPaymentBps: 1500,
		TermMonthsMin:     12,
		TermMonthsMax:     60,
		RateMinBps:        1800,
		RateMaxBps:        2400,
		MaxAmountCNYCents: Pointer(int64(50_000_000)),
	}

	valid := LeasingBounds{
		DownPaymentBps: 2000, TermMonths: 36, RateBps: 2000, AmountCNYCents: 10_000_000,
	}

	cases := []struct {
		name    string
		mutate  func(*LeasingBounds)
		program func(*LeasingProgram)
		expErr  error
	}{
		{name: "внутри границ", mutate: func(*LeasingBounds) {}},
		{
			name:   "взнос ровно минимальный",
			mutate: func(b *LeasingBounds) { b.DownPaymentBps = 1500 },
		},
		{
			name:   "взнос ниже минимального",
			mutate: func(b *LeasingBounds) { b.DownPaymentBps = 1499 },
			expErr: ErrLeasingDownPaymentTooLow,
		},
		{
			name:   "срок на нижней границе",
			mutate: func(b *LeasingBounds) { b.TermMonths = 12 },
		},
		{
			name:   "срок на верхней границе",
			mutate: func(b *LeasingBounds) { b.TermMonths = 60 },
		},
		{
			name:   "срок ниже границы",
			mutate: func(b *LeasingBounds) { b.TermMonths = 11 },
			expErr: ErrLeasingTermOutOfRange,
		},
		{
			name:   "срок выше границы",
			mutate: func(b *LeasingBounds) { b.TermMonths = 61 },
			expErr: ErrLeasingTermOutOfRange,
		},
		{
			name:   "ставка ниже границы",
			mutate: func(b *LeasingBounds) { b.RateBps = 1799 },
			expErr: ErrLeasingRateOutOfRange,
		},
		{
			name:   "ставка выше границы",
			mutate: func(b *LeasingBounds) { b.RateBps = 2401 },
			expErr: ErrLeasingRateOutOfRange,
		},
		{
			name:   "сумма ровно максимальная",
			mutate: func(b *LeasingBounds) { b.AmountCNYCents = 50_000_000 },
		},
		{
			name:   "сумма выше максимума",
			mutate: func(b *LeasingBounds) { b.AmountCNYCents = 50_000_001 },
			expErr: ErrLeasingAmountAboveMaximum,
		},
		{
			name:    "без ограничения суммы",
			mutate:  func(b *LeasingBounds) { b.AmountCNYCents = 1_000_000_000 },
			program: func(p *LeasingProgram) { p.MaxAmountCNYCents = nil },
		},
		{
			name:    "программа не действует",
			mutate:  func(*LeasingBounds) {},
			program: func(p *LeasingProgram) { p.IsActive = false },
			expErr:  ErrLeasingProgramInactive,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			bounds := valid
			c.mutate(&bounds)

			checked := *program
			if c.program != nil {
				c.program(&checked)
			}

			err := checked.CheckBounds(bounds)

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestLeasingProgram_CheckBounds_Nil фиксирует отказ на отсутствующей программе.
func TestLeasingProgram_CheckBounds_Nil(t *testing.T) {
	t.Parallel()

	var program *LeasingProgram

	require.ErrorIs(t, program.CheckBounds(LeasingBounds{}), ErrLeasingProgramNotFound)
}

// TestLeadFilter_Validate фиксирует умолчания и границы отбора заявок.
func TestLeadFilter_Validate(t *testing.T) {
	t.Parallel()

	t.Run("умолчание размера страницы", func(t *testing.T) {
		t.Parallel()

		filter := &LeadFilter{}
		require.NoError(t, filter.Validate())
		assert.Equal(t, DefaultLeadLimit, filter.Limit)
	})

	cases := []struct {
		name   string
		filter LeadFilter
		expErr error
	}{
		{name: "статус из перечня", filter: LeadFilter{Status: Pointer(LeadStatusNew)}},
		{
			name:   "неизвестный статус",
			filter: LeadFilter{Status: Pointer(LeadStatus("archived"))},
			expErr: ErrLeadStatusInvalid,
		},
		{name: "предельный размер", filter: LeadFilter{Limit: MaxLeadLimit}},
		{
			name:   "размер сверх предела",
			filter: LeadFilter{Limit: MaxLeadLimit + 1},
			expErr: ErrCatalogLimitInvalid,
		},
		{name: "отрицательный размер", filter: LeadFilter{Limit: -1}, expErr: ErrCatalogLimitInvalid},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := c.filter
			err := filter.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestLeadFilter_Validate_Nil фиксирует отказ на пустом фильтре.
func TestLeadFilter_Validate_Nil(t *testing.T) {
	t.Parallel()

	var filter *LeadFilter

	require.ErrorIs(t, filter.Validate(), ErrNilPointer)
}

// TestLeasingProgram_Validate фиксирует проверки программы лизинга.
//
// Диапазоны проверяются парой, а не по отдельности: программа с нижней
// границей выше верхней проходит поштучную проверку и оставляет калькулятору
// пустое множество допустимых значений — отказ на каждом расчёте вместо
// отказа при заведении.
func TestLeasingProgram_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*LeasingProgram)
		expErr error
	}{
		{name: "корректная программа", mutate: func(*LeasingProgram) {}},
		{name: "нет русского названия", mutate: func(p *LeasingProgram) { p.NameRU = "" }, expErr: ErrLeasingProgramNameRequired},
		// Непереведённая программа сохраняется: пустые казахское
		// и английское названия — обычное состояние, пока перевода нет.
		{name: "нет казахского и английского", mutate: func(p *LeasingProgram) { p.NameKK, p.NameEN = "", "" }},
		{
			name:   "отрицательный порядок",
			mutate: func(p *LeasingProgram) { p.Position = -1 },
			expErr: ErrLeasingProgramPositionInvalid,
		},
		{
			name:   "отрицательный взнос",
			mutate: func(p *LeasingProgram) { p.MinDownPaymentBps = -1 },
			expErr: ErrQuoteDownPaymentInvalid,
		},
		{
			name:   "взнос больше ста процентов",
			mutate: func(p *LeasingProgram) { p.MinDownPaymentBps = 10_001 },
			expErr: ErrQuoteDownPaymentInvalid,
		},
		{
			name:   "нулевой срок",
			mutate: func(p *LeasingProgram) { p.TermMonthsMin = 0 },
			expErr: ErrLeasingTermOutOfRange,
		},
		{
			name:   "срок наоборот",
			mutate: func(p *LeasingProgram) { p.TermMonthsMin, p.TermMonthsMax = 60, 12 },
			expErr: ErrLeasingTermOutOfRange,
		},
		{
			name:   "отрицательная ставка",
			mutate: func(p *LeasingProgram) { p.RateMinBps = -1 },
			expErr: ErrLeasingRateOutOfRange,
		},
		{
			name:   "ставка наоборот",
			mutate: func(p *LeasingProgram) { p.RateMinBps, p.RateMaxBps = 2400, 1800 },
			expErr: ErrLeasingRateOutOfRange,
		},
		{
			name:   "нулевой потолок суммы",
			mutate: func(p *LeasingProgram) { p.MaxAmountCNYCents = Pointer(int64(0)) },
			expErr: ErrLeasingAmountAboveMaximum,
		},
		{
			name:   "потолок суммы задан",
			mutate: func(p *LeasingProgram) { p.MaxAmountCNYCents = Pointer(int64(50_000_000)) },
		},
		{
			name:   "границы совпадают",
			mutate: func(p *LeasingProgram) { p.TermMonthsMin, p.TermMonthsMax = 36, 36 },
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			program := &LeasingProgram{
				NameRU: "Halyk", NameKK: "Halyk", NameEN: "Halyk",
				MinDownPaymentBps: 1500,
				TermMonthsMin:     12, TermMonthsMax: 60,
				RateMinBps: 1800, RateMaxBps: 2400,
				Position: 1,
			}
			c.mutate(program)

			err := program.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestLeasingProgram_Validate_Nil фиксирует, что пустая программа не паникует.
func TestLeasingProgram_Validate_Nil(t *testing.T) {
	t.Parallel()

	var program *LeasingProgram

	require.ErrorIs(t, program.Validate(), ErrNilPointer)
}

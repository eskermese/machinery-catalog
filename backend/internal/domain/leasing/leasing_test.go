package leasing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/leasing"
)

// Суммы в тестах — в минорных единицах: 100000.00 это 10_000_000.
const (
	assetCost100k = 10_000_000 // 100 000.00
	bps20Percent  = 2000       // 20.00%
	bps100Percent = 10000      // 100.00%
	rate18Percent = 1800       // 18% годовых
)

// TestCalculate_ControlCase — контрольный расчёт, посчитанный вручную.
//
// Вход: стоимость 100 000.00, взнос 20%, срок 36 мес, ставка 1800 bps, аннуитет,
// без выкупа и комиссии.
//
// Ручной расчёт:
//
//	взнос            = 100 000.00 * 0.20 = 20 000.00       → 2 000 000 мин. ед.
//	финансирование   = 100 000.00 - 20 000.00 = 80 000.00  → 8 000 000 мин. ед.
//	месячная ставка  = 1800 / 10000 / 12 = 0.015
//	f = 1.015^36, пошагово:
//	    ^4  = 1.061363550625
//	    ^8  = 1.126492586595307
//	    ^16 = 1.268985547654184
//	    ^32 = 1.610324320155075
//	    ^36 = ^32 * ^4 = 1.709139538097579
//	P = PV * i * f / (f - 1)
//	  = 8 000 000 * 0.015 * 1.709139538097579 / 0.709139538097579
//	  = 205 096.744571709 / 0.709139538097579
//	  = 289 219.16…                                        → 289 219 мин. ед.
//
// То есть регулярный платёж 2 892.19 в основной единице.
func TestCalculate_ControlCase(t *testing.T) {
	t.Parallel()

	result, err := leasing.Calculate(leasing.Params{
		AssetCostCents: assetCost100k,
		DownPaymentBps: bps20Percent,
		TermMonths:     36,
		AnnualRateBps:  rate18Percent,
		Schedule:       leasing.ScheduleAnnuity,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(2_000_000), result.DownPaymentCents, "взнос 20% от 100 000.00")
	assert.Equal(t, int64(8_000_000), result.FinancedAmountCents, "финансирование = стоимость минус взнос")
	assert.Equal(t, int64(289_219), result.FirstPaymentCents, "регулярный аннуитетный платёж, посчитан вручную выше")

	// Первый платёж: проценты на весь остаток, остальное — в тело.
	first := result.Schedule[0]
	assert.Equal(t, int64(120_000), first.InterestCents, "8 000 000 * 0.015")
	assert.Equal(t, int64(169_219), first.PrincipalCents, "289 219 - 120 000")

	require.Len(t, result.Schedule, 36)
	assert.Zero(t, result.Schedule[35].BalanceCents, "после последнего платежа долга не остаётся")
}

// calcCase описывает сценарий расчёта. Ожидания заданы только там, где значение
// проверяемо независимо от реализации; инварианты схождения проверяются для всех.
type calcCase struct {
	name   string
	params leasing.Params

	expDownPayment int64
	expFinanced    int64
	expFirst       int64
	expLast        int64
	expInterest    int64
	expTotalPaid   int64
	expOverpayment int64
}

//nolint:funlen // таблица сценариев: разбивать её на функции только вредит читаемости
func calcCases() []calcCase {
	return []calcCase{
		{
			// Ставка 0 — отдельная ветвь: общая формула аннуитета делит на ноль.
			// Тело делится на срок поровну, процентов нет вовсе.
			name: "аннуитет со ставкой 0",
			params: leasing.Params{
				AssetCostCents: 1_200_000,
				DownPaymentBps: 0,
				TermMonths:     12,
				AnnualRateBps:  0,
				Schedule:       leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    1_200_000,
			expFirst:       100_000,
			expLast:        100_000,
			expInterest:    0,
			expTotalPaid:   1_200_000,
			expOverpayment: 0,
		},
		{
			name: "дифференцированный со ставкой 0",
			params: leasing.Params{
				AssetCostCents: 1_200_000,
				TermMonths:     12,
				AnnualRateBps:  0,
				Schedule:       leasing.ScheduleDifferentiated,
			},
			expDownPayment: 0,
			expFinanced:    1_200_000,
			expFirst:       100_000,
			expLast:        100_000,
			expInterest:    0,
			expTotalPaid:   1_200_000,
			expOverpayment: 0,
		},
		{
			// Взнос 0%: финансируется вся стоимость.
			name: "взнос 0 процентов",
			params: leasing.Params{
				AssetCostCents: assetCost100k,
				DownPaymentBps: 0,
				TermMonths:     12,
				AnnualRateBps:  rate18Percent,
				Schedule:       leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    assetCost100k,
		},
		{
			// Взнос 100%: финансировать нечего, график состоит из нулевых платежей.
			// Вырожденный, но валидный вход — клиент платит сразу и целиком.
			name: "взнос 100 процентов",
			params: leasing.Params{
				AssetCostCents: assetCost100k,
				DownPaymentBps: bps100Percent,
				TermMonths:     12,
				AnnualRateBps:  rate18Percent,
				Schedule:       leasing.ScheduleAnnuity,
			},
			expDownPayment: assetCost100k,
			expFinanced:    0,
			expFirst:       0,
			expLast:        0,
			expInterest:    0,
			expTotalPaid:   0,
			expOverpayment: 0,
		},
		{
			// Срок 1 месяц: единственный платёж гасит всё тело и проценты за месяц.
			name: "срок 1 месяц",
			params: leasing.Params{
				AssetCostCents: 1_000_000,
				TermMonths:     1,
				AnnualRateBps:  rate18Percent,
				Schedule:       leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    1_000_000,
			expFirst:       1_015_000, // 1 000 000 + 1 000 000 * 0.015
			expLast:        1_015_000,
			expInterest:    15_000,
			expTotalPaid:   1_015_000,
			expOverpayment: 15_000,
		},
		{
			// Выкупная стоимость равна сумме финансирования: тело не амортизируется
			// вовсе, платежи состоят из одних процентов, выкуп уходит в последний.
			name: "выкупная стоимость равна сумме финансирования",
			params: leasing.Params{
				AssetCostCents:     1_000_000,
				TermMonths:         12,
				AnnualRateBps:      rate18Percent,
				ResidualValueCents: 1_000_000,
				Schedule:           leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    1_000_000,
			expFirst:       15_000,             // только проценты: 1 000 000 * 0.015
			expLast:        1_000_000 + 15_000, // проценты плюс выкуп
			expInterest:    15_000 * 12,
			expTotalPaid:   1_000_000 + 15_000*12,
			expOverpayment: 15_000 * 12,
		},
		{
			// Выкуп при нулевой ставке: платежи нулевые, всё уходит в последний.
			name: "выкупная стоимость равна финансированию при ставке 0",
			params: leasing.Params{
				AssetCostCents:     1_000_000,
				TermMonths:         6,
				AnnualRateBps:      0,
				ResidualValueCents: 1_000_000,
				Schedule:           leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    1_000_000,
			expFirst:       0,
			expLast:        1_000_000,
			expInterest:    0,
			expTotalPaid:   1_000_000,
			expOverpayment: 0,
		},
		{
			// Комиссия не входит в тело финансирования, но входит в итого и переплату.
			name: "ненулевая комиссия",
			params: leasing.Params{
				AssetCostCents:  1_200_000,
				TermMonths:      12,
				AnnualRateBps:   0,
				CommissionCents: 50_000,
				Schedule:        leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    1_200_000,
			expFirst:       100_000,
			expLast:        100_000,
			expInterest:    0,
			expTotalPaid:   1_250_000, // 1 200 000 по графику + 50 000 комиссии
			expOverpayment: 50_000,    // вся переплата — это комиссия
		},
		{
			// Дифференцированный график: тело равными долями, платёж убывает.
			name: "дифференцированный график с процентами",
			params: leasing.Params{
				AssetCostCents: assetCost100k,
				DownPaymentBps: bps20Percent,
				TermMonths:     36,
				AnnualRateBps:  rate18Percent,
				Schedule:       leasing.ScheduleDifferentiated,
			},
			expDownPayment: 2_000_000,
			expFinanced:    8_000_000,
			expFirst:       222_222 + 120_000, // 8 000 000/36 округлённо + 8 000 000*0.015
		},
		{
			// Дифференцированный с выкупом: делится только амортизируемая часть.
			name: "дифференцированный график с выкупом",
			params: leasing.Params{
				AssetCostCents:     1_000_000,
				TermMonths:         10,
				AnnualRateBps:      rate18Percent,
				ResidualValueCents: 500_000,
				CommissionCents:    1_000,
				Schedule:           leasing.ScheduleDifferentiated,
			},
			expDownPayment: 0,
			expFinanced:    1_000_000,
			expFirst:       50_000 + 15_000, // (1 000 000-500 000)/10 + 1 000 000*0.015
		},
		{
			// Сумма, не делящаяся на срок нацело: разница округления обязана
			// осесть в последнем платеже, а не потеряться.
			name: "неделимая сумма, разница округления в последнем платеже",
			params: leasing.Params{
				AssetCostCents: 1_000_000,
				TermMonths:     7,
				AnnualRateBps:  0,
				Schedule:       leasing.ScheduleDifferentiated,
			},
			expDownPayment: 0,
			expFinanced:    1_000_000,
			expFirst:       142_857, // округление 1 000 000 / 7 = 142 857.14…
			expLast:        142_858, // 1 000 000 - 142 857*6
			expInterest:    0,
			expTotalPaid:   1_000_000,
			expOverpayment: 0,
		},
		{
			// Взнос с дробной частью процента: 12.34% это 1234 bps.
			name: "взнос с сотыми долями процента",
			params: leasing.Params{
				AssetCostCents: assetCost100k,
				DownPaymentBps: 1234,
				TermMonths:     24,
				AnnualRateBps:  rate18Percent,
				Schedule:       leasing.ScheduleAnnuity,
			},
			expDownPayment: 1_234_000, // 10 000 000 * 0.1234
			expFinanced:    8_766_000,
		},
		{
			// Нулевая стоимость: вход валиден, весь график нулевой.
			name: "нулевая стоимость",
			params: leasing.Params{
				AssetCostCents: 0,
				TermMonths:     6,
				AnnualRateBps:  rate18Percent,
				Schedule:       leasing.ScheduleAnnuity,
			},
			expDownPayment: 0,
			expFinanced:    0,
			expFirst:       0,
			expLast:        0,
			expInterest:    0,
			expTotalPaid:   0,
			expOverpayment: 0,
		},
	}
}

// TestCalculate проверяет ожидаемые величины и инварианты схождения для каждого
// сценария. Схождение проверяется всегда: расхождение на цент в графике —
// это спор с клиентом, а не косметика.
func TestCalculate(t *testing.T) {
	t.Parallel()

	for _, c := range calcCases() {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			result, err := leasing.Calculate(c.params)
			require.NoError(t, err)

			require.Len(t, result.Schedule, c.params.TermMonths, "в графике ровно столько платежей, сколько месяцев")

			assertExpectations(t, c, result)
			assertConverges(t, result)
			assertScheduleConsistent(t, result)
		})
	}
}

// assertExpectations сверяет заданные в сценарии величины.
// Нулевое ожидание в сценарии означает «не проверяем», кроме полей, где ноль —
// осмысленный ответ; такие сценарии задают его явно вместе с остальными полями.
func assertExpectations(t *testing.T, c calcCase, result leasing.Result) {
	t.Helper()

	assert.Equal(t, c.expDownPayment, result.DownPaymentCents, "первоначальный взнос")
	assert.Equal(t, c.expFinanced, result.FinancedAmountCents, "сумма финансирования")

	if c.expFirst != 0 || c.expTotalPaid != 0 {
		assert.Equal(t, c.expFirst, result.FirstPaymentCents, "первый платёж")
	}

	if c.expTotalPaid != 0 || c.expLast != 0 {
		assert.Equal(t, c.expLast, result.LastPaymentCents, "последний платёж")
		assert.Equal(t, c.expInterest, result.TotalInterestCents, "проценты за весь срок")
		assert.Equal(t, c.expTotalPaid, result.TotalPaidCents, "итого выплат")
		assert.Equal(t, c.expOverpayment, result.OverpaymentCents, "переплата")
	}
}

// assertConverges проверяет главный инвариант: график сходится с итогами.
func assertConverges(t *testing.T, result leasing.Result) {
	t.Helper()

	var sumPrincipal, sumInterest, sumAmount int64

	for _, payment := range result.Schedule {
		sumPrincipal += payment.PrincipalCents
		sumInterest += payment.InterestCents
		sumAmount += payment.AmountCents
	}

	assert.Equal(t, result.FinancedAmountCents, sumPrincipal,
		"доли основного долга обязаны сложиться ровно в сумму финансирования")
	assert.Equal(t, result.TotalInterestCents, sumInterest,
		"проценты графика обязаны сойтись с итоговыми процентами")
	assert.Equal(t, result.FinancedAmountCents+sumInterest, sumAmount,
		"сумма платежей = финансирование плюс проценты")
	assert.Equal(t, sumAmount+commissionOf(result), result.TotalPaidCents,
		"итого выплат = сумма платежей графика плюс комиссия")
	assert.Equal(t, result.TotalPaidCents-result.FinancedAmountCents, result.OverpaymentCents,
		"переплата = итого выплат минус финансирование")
}

// commissionOf восстанавливает комиссию из результата: она равна разнице
// переплаты и процентов. Отдельного поля у неё нет — калькулятор возвращает
// то, что уплачено, а не разбор входных параметров.
func commissionOf(result leasing.Result) int64 {
	return result.OverpaymentCents - result.TotalInterestCents
}

// assertScheduleConsistent проверяет внутреннюю связность строк графика.
func assertScheduleConsistent(t *testing.T, result leasing.Result) {
	t.Helper()

	balance := result.FinancedAmountCents

	for i, payment := range result.Schedule {
		assert.Equal(t, i+1, payment.Number, "нумерация платежей идёт подряд с единицы")
		assert.Equal(t, payment.PrincipalCents+payment.InterestCents, payment.AmountCents,
			"платёж %d: сумма = тело плюс проценты", payment.Number)

		balance -= payment.PrincipalCents
		assert.Equal(t, balance, payment.BalanceCents, "платёж %d: остаток долга", payment.Number)
	}

	assert.Zero(t, result.Schedule[len(result.Schedule)-1].BalanceCents,
		"после последнего платежа долга не остаётся")
	assert.Equal(t, result.Schedule[0].AmountCents, result.FirstPaymentCents)
	assert.Equal(t, result.Schedule[len(result.Schedule)-1].AmountCents, result.LastPaymentCents)
}

// TestCalculate_Validation фиксирует, что недопустимый вход возвращает ошибку,
// а не панику: параметры приходят из формы.
func TestCalculate_Validation(t *testing.T) {
	t.Parallel()

	valid := leasing.Params{
		AssetCostCents: assetCost100k,
		DownPaymentBps: bps20Percent,
		TermMonths:     36,
		AnnualRateBps:  rate18Percent,
		Schedule:       leasing.ScheduleAnnuity,
	}

	withParams := func(mutate func(*leasing.Params)) leasing.Params {
		params := valid
		mutate(&params)

		return params
	}

	cases := []struct {
		name   string
		params leasing.Params
		expErr error
	}{
		{
			name:   "отрицательная стоимость",
			params: withParams(func(p *leasing.Params) { p.AssetCostCents = -1 }),
			expErr: leasing.ErrAssetCostInvalid,
		},
		{
			name:   "отрицательный взнос",
			params: withParams(func(p *leasing.Params) { p.DownPaymentBps = -1 }),
			expErr: leasing.ErrDownPaymentInvalid,
		},
		{
			name:   "взнос больше 100 процентов",
			params: withParams(func(p *leasing.Params) { p.DownPaymentBps = bps100Percent + 1 }),
			expErr: leasing.ErrDownPaymentInvalid,
		},
		{
			name:   "нулевой срок",
			params: withParams(func(p *leasing.Params) { p.TermMonths = 0 }),
			expErr: leasing.ErrTermInvalid,
		},
		{
			name:   "отрицательный срок",
			params: withParams(func(p *leasing.Params) { p.TermMonths = -12 }),
			expErr: leasing.ErrTermInvalid,
		},
		{
			// Срок входит в показатель степени: без верхней границы вход вроде
			// миллиарда месяцев съедает память, не давая осмысленного ответа.
			name:   "срок сверх допустимого предела",
			params: withParams(func(p *leasing.Params) { p.TermMonths = 1201 }),
			expErr: leasing.ErrTermInvalid,
		},
		{
			name:   "отрицательная ставка",
			params: withParams(func(p *leasing.Params) { p.AnnualRateBps = -1 }),
			expErr: leasing.ErrRateInvalid,
		},
		{
			name:   "отрицательная выкупная стоимость",
			params: withParams(func(p *leasing.Params) { p.ResidualValueCents = -1 }),
			expErr: leasing.ErrResidualValueInvalid,
		},
		{
			// 8 000 000 — сумма финансирования при взносе 20% от 100 000.00.
			name:   "выкупная стоимость больше суммы финансирования",
			params: withParams(func(p *leasing.Params) { p.ResidualValueCents = 8_000_001 }),
			expErr: leasing.ErrResidualValueInvalid,
		},
		{
			name:   "отрицательная комиссия",
			params: withParams(func(p *leasing.Params) { p.CommissionCents = -1 }),
			expErr: leasing.ErrCommissionInvalid,
		},
		{
			name:   "неизвестный тип графика",
			params: withParams(func(p *leasing.Params) { p.Schedule = "buyback" }),
			expErr: leasing.ErrScheduleTypeInvalid,
		},
		{
			name:   "пустой тип графика",
			params: withParams(func(p *leasing.Params) { p.Schedule = "" }),
			expErr: leasing.ErrScheduleTypeInvalid,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			result, err := leasing.Calculate(c.params)

			require.ErrorIs(t, err, c.expErr)
			assert.Empty(t, result.Schedule, "при ошибке график не строится")
		})
	}
}

// TestCalculate_ResidualEqualToFinancedIsAllowed фиксирует границу: выкуп,
// равный сумме финансирования, допустим, а больший — уже нет.
func TestCalculate_ResidualEqualToFinancedIsAllowed(t *testing.T) {
	t.Parallel()

	params := leasing.Params{
		AssetCostCents:     1_000_000,
		TermMonths:         12,
		AnnualRateBps:      rate18Percent,
		ResidualValueCents: 1_000_000,
		Schedule:           leasing.ScheduleAnnuity,
	}

	_, err := leasing.Calculate(params)
	require.NoError(t, err)

	params.ResidualValueCents++

	_, err = leasing.Calculate(params)
	require.ErrorIs(t, err, leasing.ErrResidualValueInvalid)
}

// TestCalculate_AnnuityPaymentIsConstant фиксирует смысл аннуитета: все платежи,
// кроме последнего, равны между собой.
func TestCalculate_AnnuityPaymentIsConstant(t *testing.T) {
	t.Parallel()

	result, err := leasing.Calculate(leasing.Params{
		AssetCostCents: assetCost100k,
		DownPaymentBps: bps20Percent,
		TermMonths:     36,
		AnnualRateBps:  rate18Percent,
		Schedule:       leasing.ScheduleAnnuity,
	})
	require.NoError(t, err)

	for _, payment := range result.Schedule[:len(result.Schedule)-1] {
		assert.Equal(t, result.FirstPaymentCents, payment.AmountCents,
			"платёж %d обязан совпадать с регулярным", payment.Number)
	}

	// Последний платёж вбирает накопленную разницу округления, поэтому от
	// регулярного он отличается — но не более чем на несколько минорных единиц.
	assert.InDelta(t, result.FirstPaymentCents, result.LastPaymentCents, 100,
		"последний платёж отличается только на разницу округления")
}

// TestCalculate_DifferentiatedPaymentDecreases фиксирует смысл дифференцированного
// графика: платёж убывает вместе с остатком долга.
func TestCalculate_DifferentiatedPaymentDecreases(t *testing.T) {
	t.Parallel()

	result, err := leasing.Calculate(leasing.Params{
		AssetCostCents: assetCost100k,
		DownPaymentBps: bps20Percent,
		TermMonths:     36,
		AnnualRateBps:  rate18Percent,
		Schedule:       leasing.ScheduleDifferentiated,
	})
	require.NoError(t, err)

	for i := 1; i < len(result.Schedule)-1; i++ {
		assert.Less(t, result.Schedule[i].AmountCents, result.Schedule[i-1].AmountCents,
			"платёж %d обязан быть меньше предыдущего", i+1)
	}

	assert.Less(t, result.LastPaymentCents, result.FirstPaymentCents)
}

// TestCalculate_DifferentiatedIsCheaperThanAnnuity фиксирует, что при равных
// условиях дифференцированный график даёт меньше процентов: тело гасится быстрее.
func TestCalculate_DifferentiatedIsCheaperThanAnnuity(t *testing.T) {
	t.Parallel()

	params := leasing.Params{
		AssetCostCents: assetCost100k,
		DownPaymentBps: bps20Percent,
		TermMonths:     36,
		AnnualRateBps:  rate18Percent,
		Schedule:       leasing.ScheduleAnnuity,
	}

	annuity, err := leasing.Calculate(params)
	require.NoError(t, err)

	params.Schedule = leasing.ScheduleDifferentiated

	differentiated, err := leasing.Calculate(params)
	require.NoError(t, err)

	assert.Less(t, differentiated.TotalInterestCents, annuity.TotalInterestCents)
}

// TestScheduleType_String фиксирует строковые значения, которые уходят в БД и логи.
func TestScheduleType_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "annuity", leasing.ScheduleAnnuity.String())
	assert.Equal(t, "differentiated", leasing.ScheduleDifferentiated.String())
}

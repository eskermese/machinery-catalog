package tco_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/tco"
)

// litres возвращает указатель на расход топлива.
func litres(value string) *decimal.Decimal {
	d := decimal.RequireFromString(value)

	return &d
}

func intPtr(value int) *int       { return &value }
func moneyPtr(value int64) *int64 { return &value }

// fullParams параметры, у которых заведено всё.
//
// Цифры подобраны так, чтобы каждую статью можно было проверить в уме:
// 2000 моточасов в год × 3 года = 6000 моточасов.
func fullParams() tco.Params {
	return tco.Params{
		PurchasePriceCents:     100_000_00, // 100 000 условных единиц
		Years:                  3,
		HoursPerYear:           2000,
		FuelPricePerLitreCents: 100, // 1 у.е. за литр
		OperatorHourlyCents:    500, // 5 у.е. за моточас
		FuelConsumptionLPH:     litres("20"),
		ServiceIntervalHours:   intPtr(500),
		ServiceCostCents:       moneyPtr(300_00),
		ResidualValueBps:       intPtr(4500), // 45% от цены покупки
	}
}

// TestCalculate_Full фиксирует расчёт при полном наборе данных.
//
// Каждая статья проверяется отдельно, а не только итог: сумма может сойтись
// и при двух взаимно компенсирующих ошибках.
func TestCalculate_Full(t *testing.T) {
	t.Parallel()

	result, err := tco.Calculate(fullParams())
	require.NoError(t, err)

	assert.True(t, result.Complete(), "при полных данных ничего не должно не хватать")
	assert.Equal(t, 6000, result.TotalHours)

	assert.Equal(t, int64(100_000_00), result.PurchaseCents)

	// 20 л/мч × 6000 мч × 1 у.е. = 120 000 у.е.
	require.NotNil(t, result.FuelCents)
	assert.Equal(t, int64(120_000_00), *result.FuelCents)

	// 6000 / 500 = 12 обслуживаний × 300 у.е. = 3600 у.е.
	require.NotNil(t, result.ServicesCount)
	assert.Equal(t, 12, *result.ServicesCount)
	require.NotNil(t, result.ServiceCents)
	assert.Equal(t, int64(3_600_00), *result.ServiceCents)

	// 5 у.е. × 6000 мч = 30 000 у.е.
	assert.Equal(t, int64(30_000_00), result.OperatorCents)

	// 45% от 100 000 = 45 000 у.е.
	require.NotNil(t, result.ResidualValueCents)
	assert.Equal(t, int64(45_000_00), *result.ResidualValueCents)

	// 100 000 + 120 000 + 3 600 + 30 000 − 45 000 = 208 600 у.е.
	assert.Equal(t, int64(208_600_00), result.TotalCents)
	// 208 600 / 6000 = 34,7666… → 3477 сотых
	assert.Equal(t, int64(3477), result.CostPerHourCents)
}

// TestCalculate_Missing фиксирует поведение при незаведённых величинах:
// статья не считается, но и не подменяется нулём, а расчёт не отказывает.
func TestCalculate_Missing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// mutate убирает одну или несколько величин карточки.
		mutate      func(*tco.Params)
		expMissing  []tco.Missing
		expFuelNil  bool
		expSvcNil   bool
		expResidNil bool
		// expTotal итог без непосчитанных статей.
		expTotal int64
	}{
		{
			name:       "нет расхода топлива",
			mutate:     func(p *tco.Params) { p.FuelConsumptionLPH = nil },
			expMissing: []tco.Missing{tco.MissingFuelConsumption},
			expFuelNil: true,
			// 100 000 + 3 600 + 30 000 − 45 000
			expTotal: 88_600_00,
		},
		{
			name:       "нет интервала ТО",
			mutate:     func(p *tco.Params) { p.ServiceIntervalHours = nil },
			expMissing: []tco.Missing{tco.MissingServiceInterval},
			expSvcNil:  true,
			// 100 000 + 120 000 + 30 000 − 45 000
			expTotal: 205_000_00,
		},
		{
			name:       "нет стоимости ТО",
			mutate:     func(p *tco.Params) { p.ServiceCostCents = nil },
			expMissing: []tco.Missing{tco.MissingServiceCost},
			expSvcNil:  true,
			expTotal:   205_000_00,
		},
		{
			name: "нет ни интервала, ни стоимости ТО",
			mutate: func(p *tco.Params) {
				p.ServiceIntervalHours = nil
				p.ServiceCostCents = nil
			},
			expMissing: []tco.Missing{tco.MissingServiceInterval, tco.MissingServiceCost},
			expSvcNil:  true,
			expTotal:   205_000_00,
		},
		{
			name:        "нет остаточной стоимости",
			mutate:      func(p *tco.Params) { p.ResidualValueBps = nil },
			expMissing:  []tco.Missing{tco.MissingResidualValue},
			expResidNil: true,
			// 100 000 + 120 000 + 3 600 + 30 000
			expTotal: 253_600_00,
		},
		{
			name: "не заведено ничего",
			mutate: func(p *tco.Params) {
				p.FuelConsumptionLPH = nil
				p.ServiceIntervalHours = nil
				p.ServiceCostCents = nil
				p.ResidualValueBps = nil
			},
			expMissing: []tco.Missing{
				tco.MissingFuelConsumption,
				tco.MissingServiceInterval,
				tco.MissingServiceCost,
				tco.MissingResidualValue,
			},
			expFuelNil:  true,
			expSvcNil:   true,
			expResidNil: true,
			// Остаются только покупка и оператор — то, что известно всегда.
			expTotal: 130_000_00,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			params := fullParams()
			c.mutate(&params)

			result, err := tco.Calculate(params)
			require.NoError(t, err, "незаведённая величина — не ошибка ввода")

			assert.Equal(t, c.expMissing, result.Missing)
			assert.False(t, result.Complete())

			assert.Equal(t, c.expFuelNil, result.FuelCents == nil, "топливо")
			assert.Equal(t, c.expSvcNil, result.ServiceCents == nil, "ТО")
			assert.Equal(t, c.expSvcNil, result.ServicesCount == nil, "число ТО")
			assert.Equal(t, c.expResidNil, result.ResidualValueCents == nil, "остаточная стоимость")

			assert.Equal(t, c.expTotal, result.TotalCents,
				"непосчитанная статья не входит в итог и не подменяется нулём")
		})
	}
}

// TestCalculate_InvalidInput фиксирует отказы на вводе клиента.
func TestCalculate_InvalidInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*tco.Params)
		expErr error
	}{
		{
			name:   "цена покупки ноль",
			mutate: func(p *tco.Params) { p.PurchasePriceCents = 0 },
			expErr: tco.ErrPurchasePriceInvalid,
		},
		{
			name:   "цена покупки отрицательная",
			mutate: func(p *tco.Params) { p.PurchasePriceCents = -1 },
			expErr: tco.ErrPurchasePriceInvalid,
		},
		{
			name:   "срок ноль",
			mutate: func(p *tco.Params) { p.Years = 0 },
			expErr: tco.ErrYearsInvalid,
		},
		{
			name:   "срок больше предела",
			mutate: func(p *tco.Params) { p.Years = 31 },
			expErr: tco.ErrYearsInvalid,
		},
		{
			name:   "наработка ноль",
			mutate: func(p *tco.Params) { p.HoursPerYear = 0 },
			expErr: tco.ErrHoursPerYearInvalid,
		},
		{
			name:   "наработка больше часов в году",
			mutate: func(p *tco.Params) { p.HoursPerYear = 8761 },
			expErr: tco.ErrHoursPerYearInvalid,
		},
		{
			name:   "отрицательная цена топлива",
			mutate: func(p *tco.Params) { p.FuelPricePerLitreCents = -1 },
			expErr: tco.ErrFuelPriceInvalid,
		},
		{
			name:   "отрицательная ставка оператора",
			mutate: func(p *tco.Params) { p.OperatorHourlyCents = -1 },
			expErr: tco.ErrOperatorRateInvalid,
		},
		{
			name:   "неположительный расход топлива",
			mutate: func(p *tco.Params) { p.FuelConsumptionLPH = litres("0") },
			expErr: tco.ErrFuelConsumptionInvalid,
		},
		{
			name:   "неположительный интервал ТО",
			mutate: func(p *tco.Params) { p.ServiceIntervalHours = intPtr(0) },
			expErr: tco.ErrServiceIntervalInvalid,
		},
		{
			name:   "доля остаточной стоимости больше 100%",
			mutate: func(p *tco.Params) { p.ResidualValueBps = intPtr(10001) },
			expErr: tco.ErrResidualRateInvalid,
		},
		{
			name:   "доля остаточной стоимости отрицательна",
			mutate: func(p *tco.Params) { p.ResidualValueBps = intPtr(-1) },
			expErr: tco.ErrResidualRateInvalid,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			params := fullParams()
			c.mutate(&params)

			_, err := tco.Calculate(params)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestCalculate_ServicesCountIsFloor фиксирует, что оплачивается проведённое
// ТО, а не начатый интервал: 5900 моточасов при интервале 500 — это одиннадцать
// обслуживаний, а не двенадцать.
func TestCalculate_ServicesCountIsFloor(t *testing.T) {
	t.Parallel()

	params := fullParams()
	params.HoursPerYear = 5900
	params.Years = 1

	result, err := tco.Calculate(params)
	require.NoError(t, err)

	require.NotNil(t, result.ServicesCount)
	assert.Equal(t, 11, *result.ServicesCount)
	require.NotNil(t, result.ServiceCents)
	assert.Equal(t, int64(3_300_00), *result.ServiceCents)
}

// TestCalculate_FractionalFuelConsumption фиксирует, что дробный расход
// не теряется: 12,7 л/мч — обычное паспортное значение, и округление его
// до целого на 6000 моточасах даёт разницу в 1800 литров.
func TestCalculate_FractionalFuelConsumption(t *testing.T) {
	t.Parallel()

	params := fullParams()
	params.FuelConsumptionLPH = litres("12.7")

	result, err := tco.Calculate(params)
	require.NoError(t, err)

	require.NotNil(t, result.FuelCents)
	// 12,7 × 6000 = 76 200 литров × 1 у.е.
	assert.Equal(t, int64(76_200_00), *result.FuelCents)
}

// TestCalculate_ResidualValueRounding фиксирует округление доли до минорной
// единицы: 33,33% от 100 000,01 — не целое число.
func TestCalculate_ResidualValueRounding(t *testing.T) {
	t.Parallel()

	params := fullParams()
	params.PurchasePriceCents = 100_000_01
	params.ResidualValueBps = intPtr(3333)

	result, err := tco.Calculate(params)
	require.NoError(t, err)

	require.NotNil(t, result.ResidualValueCents)
	// 10 000 001 × 3333 / 10 000 = 3 333 000,3333 → 3 333 000
	assert.Equal(t, int64(3_333_000), *result.ResidualValueCents)
}

// TestCalculate_ZeroClientRates фиксирует, что нулевые цена топлива и ставка
// оператора допустимы: это законный сценарий «интересует только железо»,
// а не ошибка ввода.
func TestCalculate_ZeroClientRates(t *testing.T) {
	t.Parallel()

	params := fullParams()
	params.FuelPricePerLitreCents = 0
	params.OperatorHourlyCents = 0

	result, err := tco.Calculate(params)
	require.NoError(t, err)

	require.NotNil(t, result.FuelCents)
	assert.Zero(t, *result.FuelCents)
	assert.Zero(t, result.OperatorCents)
	// 100 000 + 3 600 − 45 000
	assert.Equal(t, int64(58_600_00), result.TotalCents)
}

// TestMissing_String фиксирует, что перечень уходит наружу строками:
// клиент показывает человеку, чего именно не хватило.
func TestMissing_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fuel_consumption", tco.MissingFuelConsumption.String())
	assert.Equal(t, "residual_value", tco.MissingResidualValue.String())
}

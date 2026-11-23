package presenter

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/tco"
)

// testTCOApplied курс для представления расчёта.
func testTCOApplied() entity.AppliedFXRate {
	return entity.AppliedFXRate{
		Rate: decimal.RequireFromString("500"), RateDate: testCalcTime, MarkupBps: 200,
	}
}

// TestNewTCOResult_Complete фиксирует представление полного расчёта.
func TestNewTCOResult_Complete(t *testing.T) {
	t.Parallel()

	result := tco.Result{
		PurchaseCents:      50_000_000_00,
		FuelCents:          entity.Pointer(int64(36_000_000_00)),
		ServiceCents:       entity.Pointer(int64(6_000_000_00)),
		OperatorCents:      12_000_000_00,
		ResidualValueCents: entity.Pointer(int64(22_500_000_00)),
		ServicesCount:      entity.Pointer(12),
		TotalHours:         6000,
		TotalCents:         81_500_000_00,
		CostPerHourCents:   1_358_333,
	}

	machine := &entity.Machine{ID: "m1", Slug: "xcmg-xe215c", NameRU: "Экскаватор"}
	view := NewTCOResult(machine, "ekskavatory", testTCOApplied(), &result)

	assert.True(t, view.Complete)
	assert.Empty(t, view.Missing, "перечень пуст, а не отсутствует")

	require.NotNil(t, view.Fuel.AmountKztCents)
	assert.Equal(t, int64(36_000_000_00), *view.Fuel.AmountKztCents)
	assert.Empty(t, view.Fuel.Missing)

	require.NotNil(t, view.ServicesCount)
	assert.Equal(t, 12, *view.ServicesCount)

	require.NotNil(t, view.MachineRef)
	assert.Equal(t, "ekskavatory", view.MachineRef.CategorySlug)
	assert.Equal(t, "500.000000", view.FX.Rate)
}

// TestNewTCOResult_MissingIsNotZero фиксирует главное правило представления:
// непосчитанная статья приходит БЕЗ суммы и с перечнем недостающего.
//
// Ноль здесь был бы враньём: «нет данных о расходе» и «не расходует топлива» —
// разные сообщения, и клиент, увидев ноль, покажет человеку «0 ₸».
func TestNewTCOResult_MissingIsNotZero(t *testing.T) {
	t.Parallel()

	result := tco.Result{
		PurchaseCents: 50_000_000_00,
		OperatorCents: 12_000_000_00,
		TotalHours:    6000,
		TotalCents:    62_000_000_00,
		Missing: []tco.Missing{
			tco.MissingFuelConsumption,
			tco.MissingServiceInterval,
			tco.MissingServiceCost,
			tco.MissingResidualValue,
		},
	}

	view := NewTCOResult(nil, "", testTCOApplied(), &result)

	assert.False(t, view.Complete)
	assert.Equal(t, []string{
		"fuel_consumption", "service_interval", "service_cost", "residual_value",
	}, view.Missing)

	assert.Nil(t, view.Fuel.AmountKztCents, "суммы нет вовсе, а не ноль")
	assert.Equal(t, []string{"fuel_consumption"}, view.Fuel.Missing)

	// Статья обслуживания требует двух величин, и не хватить может обеих.
	assert.Nil(t, view.Service.AmountKztCents)
	assert.Equal(t, []string{"service_interval", "service_cost"}, view.Service.Missing)

	assert.Nil(t, view.ResidualValue.AmountKztCents)
	assert.Equal(t, []string{"residual_value"}, view.ResidualValue.Missing)

	// Покупка и оператор считаются всегда.
	require.NotNil(t, view.Purchase.AmountKztCents)
	require.NotNil(t, view.Operator.AmountKztCents)

	assert.Nil(t, view.MachineRef, "без слага раздела ссылку не собрать")
}

// TestNewTCOResult_PartialService фиксирует случай, когда не хватило только
// одной из двух величин обслуживания.
func TestNewTCOResult_PartialService(t *testing.T) {
	t.Parallel()

	result := tco.Result{
		PurchaseCents: 1,
		TotalHours:    100,
		Missing:       []tco.Missing{tco.MissingServiceCost},
	}

	view := NewTCOResult(nil, "", testTCOApplied(), &result)

	assert.Equal(t, []string{"service_cost"}, view.Service.Missing)
	assert.Empty(t, view.Fuel.Missing, "топливо посчитано и в перечень не попало")
}

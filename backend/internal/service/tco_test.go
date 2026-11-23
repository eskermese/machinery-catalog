package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/tco"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// newTCOService собирает сервис стоимости владения на моках.
func newTCOService(
	t *testing.T,
	machine *entity.Machine,
) (TCOService, *mock_repo.MockResidualValueRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	residual := mock_repo.NewMockResidualValueRepository(ctrl)

	// Курс 500 ₸ за доллар: цена 100 000 $ превращается в 50 000 000 ₸,
	// и все суммы в тестах считаются в уме.
	applied := entity.AppliedFXRate{Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow}

	return NewTCOService(&stubCatalogReader{machine: machine}, residual,
		&stubFXFixer{applied: applied}), residual
}

// tcoMachine собирает карточку с заведёнными величинами владения.
func tcoMachine() *entity.Machine {
	machine := testQuoteMachine()
	machine.FuelConsumptionLPH = entity.Pointer(decimal.RequireFromString("20"))
	machine.ServiceIntervalHours = entity.Pointer(500)
	machine.ServiceCostCNYCents = entity.Pointer(int64(100_000)) // 1000 $

	return machine
}

// tcoRequest собирает валидный запрос: 3 года по 2000 моточасов.
func tcoRequest() TCORequest {
	return TCORequest{
		MachineSlug:            testMachineSlug,
		Years:                  3,
		HoursPerYear:           2000,
		FuelPriceKZTCents:      30_000,  // 300 ₸ за литр
		OperatorHourlyKZTCents: 200_000, // 2000 ₸ за моточас
	}
}

// TestTCOService_Calculate фиксирует полный расчёт и приведение к тенге.
//
// Цена машины и стоимость ТО хранятся в долларах, а топливо и оператор
// приходят в тенге: обе долларовые величины обязаны пройти через один
// и тот же курс, иначе две статьи одной сметы считались бы в разных валютах.
func TestTCOService_Calculate(t *testing.T) {
	t.Parallel()

	srv, residual := newTCOService(t, tcoMachine())
	residual.EXPECT().RateFor(gomock.Any(), testCategoryID, 3).Return(entity.Pointer(4500), nil)

	result, err := srv.Calculate(context.Background(), tcoRequest(), testQuoteNow)
	require.NoError(t, err)

	assert.True(t, result.Result.Complete(), "все величины заведены")
	assert.Equal(t, 6000, result.Result.TotalHours)

	// 100 000 $ × 500 = 50 000 000 ₸
	assert.Equal(t, int64(50_000_000_00), result.Result.PurchaseCents)

	// 12 обслуживаний × 1000 $ × 500 = 6 000 000 ₸
	require.NotNil(t, result.Result.ServiceCents)
	assert.Equal(t, int64(6_000_000_00), *result.Result.ServiceCents)

	// 45% от 50 000 000 ₸
	require.NotNil(t, result.Result.ResidualValueCents)
	assert.Equal(t, int64(22_500_000_00), *result.Result.ResidualValueCents)
}

// TestTCOService_MissingCardValues фиксирует, что незаведённые величины
// карточки не отказывают в расчёте, а помечаются как недостающие.
func TestTCOService_MissingCardValues(t *testing.T) {
	t.Parallel()

	machine := testQuoteMachine() // без величин владения вовсе

	srv, residual := newTCOService(t, machine)
	residual.EXPECT().RateFor(gomock.Any(), testCategoryID, 3).Return(nil, nil)

	result, err := srv.Calculate(context.Background(), tcoRequest(), testQuoteNow)
	require.NoError(t, err, "нехватка данных карточки — не отказ")

	assert.False(t, result.Result.Complete())
	assert.ElementsMatch(t, []tco.Missing{
		tco.MissingFuelConsumption,
		tco.MissingServiceInterval,
		tco.MissingServiceCost,
		tco.MissingResidualValue,
	}, result.Result.Missing)

	// Покупка и оператор считаются всегда: цена известна по карточке,
	// ставка оператора пришла от клиента.
	assert.Positive(t, result.Result.PurchaseCents)
	assert.Positive(t, result.Result.OperatorCents)
	assert.Nil(t, result.Result.FuelCents)
}

// TestTCOService_HiddenPrice фиксирует отказ по карточке со скрытой ценой:
// стоимость владения начинается с покупки, и посчитать её «по запросу» нельзя.
func TestTCOService_HiddenPrice(t *testing.T) {
	t.Parallel()

	machine := tcoMachine()
	machine.IsPricePublic = false

	srv, _ := newTCOService(t, machine)

	_, err := srv.Calculate(context.Background(), tcoRequest(), testQuoteNow)

	require.ErrorIs(t, err, entity.ErrMachinePriceHidden)
}

// TestTCOService_MachineNotFound фиксирует, что отказ каталога доходит наружу.
func TestTCOService_MachineNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	srv := NewTCOService(
		&stubCatalogReader{err: entity.ErrMachineNotFound},
		mock_repo.NewMockResidualValueRepository(ctrl),
		&stubFXFixer{},
	)

	_, err := srv.Calculate(context.Background(), tcoRequest(), testQuoteNow)

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
}

// TestTCOService_ResidualLookupFails фиксирует, что отказ хранилища ставок
// роняет расчёт: это сбой, а не отсутствие данных, и молча считать без
// остаточной стоимости было бы подменой одного другим.
func TestTCOService_ResidualLookupFails(t *testing.T) {
	t.Parallel()

	srv, residual := newTCOService(t, tcoMachine())
	residual.EXPECT().RateFor(gomock.Any(), testCategoryID, 3).
		Return(nil, errors.New("база недоступна"))

	_, err := srv.Calculate(context.Background(), tcoRequest(), testQuoteNow)

	require.Error(t, err)
}

// TestTCOService_InvalidInput фиксирует, что ошибка калькулятора доходит
// до вызывающего кода неизменной.
func TestTCOService_InvalidInput(t *testing.T) {
	t.Parallel()

	srv, residual := newTCOService(t, tcoMachine())
	residual.EXPECT().RateFor(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(entity.Pointer(4500), nil)

	request := tcoRequest()
	request.Years = 0

	_, err := srv.Calculate(context.Background(), request, testQuoteNow)

	require.ErrorIs(t, err, tco.ErrYearsInvalid)
}

// TestTCOService_FXFails фиксирует, что без курса расчёт невозможен:
// цена хранится в долларах, а считается всё в тенге.
func TestTCOService_FXFails(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	srv := NewTCOService(
		&stubCatalogReader{machine: tcoMachine()},
		mock_repo.NewMockResidualValueRepository(ctrl),
		&stubFXFixer{err: entity.ErrFXRateNotFound},
	)

	_, err := srv.Calculate(context.Background(), tcoRequest(), testQuoteNow)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
}

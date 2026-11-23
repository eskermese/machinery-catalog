package service

import (
	"context"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/tco"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// tcoBaseCurrency и tcoQuoteCurrency — та же пара, что и у лизинга.
//
// Расчёт ведётся в тенге целиком: цена машины хранится в юанях, а топливо
// и оператор приходят от клиента в тенге, и складывать их можно только
// приведя к одной валюте.
const (
	tcoBaseCurrency  = entity.CurrencyCNY
	tcoQuoteCurrency = entity.CurrencyKZT
)

// TCORequest параметры расчёта стоимости владения, приходящие от клиента.
//
// Здесь только то, чего сервер знать не может: сколько лет клиент собирается
// владеть машиной, сколько нарабатывать, почём заправляться и сколько платить
// оператору. Всё остальное сервис берёт по карточке — как в расчёте лизинга.
type TCORequest struct {
	// MachineSlug карточка, по которой считается стоимость владения.
	MachineSlug string
	// Years срок владения в годах.
	Years int
	// HoursPerYear наработка в моточасах за год.
	HoursPerYear int
	// FuelPriceKZTCents цена литра топлива в тиынах.
	FuelPriceKZTCents int64
	// OperatorHourlyKZTCents оплата оператора за моточас в тиынах.
	OperatorHourlyKZTCents int64
}

// TCOResult результат расчёта вместе с тем, из чего он посчитан.
type TCOResult struct {
	// Machine карточка, по которой считали.
	Machine *entity.Machine
	// FX применённый курс: расчёт ведётся в тенге, а цена хранится в долларах.
	FX entity.AppliedFXRate
	// Result разложенная стоимость владения.
	Result tco.Result
}

// TCOService представляет интерфейс сервиса стоимости владения.
type TCOService interface {
	// Calculate считает стоимость владения по опубликованной карточке.
	Calculate(ctx context.Context, request TCORequest, currentTime time.Time) (*TCOResult, error)
}

// tcoService реализация сервиса стоимости владения.
type tcoService struct {
	catalog  CatalogService                     // Сервис каталога
	residual repository.ResidualValueRepository // Ставки остаточной стоимости
	fx       FXService                          // Курсы валют
}

// NewTCOService создаёт сервис стоимости владения.
func NewTCOService(
	catalog CatalogService,
	residual repository.ResidualValueRepository,
	fx FXService,
) TCOService {
	return &tcoService{catalog: catalog, residual: residual, fx: fx}
}

// Calculate считает стоимость владения по опубликованной карточке.
//
// Скрытая цена делает расчёт невозможным: стоимость владения начинается
// с покупки, и посчитать её «по запросу» нельзя. Отказ здесь тот же, что
// и у лизинга, — по той же причине.
//
// Незаведённые величины карточки отказом НЕ являются: расчёт возвращается
// неполным, и что именно не посчитано, лежит в Result.Missing. Иначе машина
// без расхода топлива не показала бы даже стоимости оператора, которую
// посчитать можно.
func (s *tcoService) Calculate(
	ctx context.Context,
	request TCORequest,
	currentTime time.Time,
) (*TCOResult, error) {
	machine, err := s.catalog.PublicMachineBySlug(ctx, request.MachineSlug)
	if err != nil {
		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	if !machine.IsPricePublic {
		return nil, entity.ErrMachinePriceHidden
	}

	applied, err := s.fx.AppliedRate(ctx, tcoBaseCurrency, tcoQuoteCurrency, currentTime)
	if err != nil {
		return nil, fmt.Errorf("получение курса: %w", err)
	}

	// Ставка остаточной стоимости ищется по разделу карточки и сроку владения.
	// Её отсутствие — не ошибка: расчёт скажет о нём в Missing.
	rate, err := s.residual.RateFor(ctx, machine.CategoryID, request.Years)
	if err != nil {
		return nil, fmt.Errorf("получение остаточной стоимости: %w", err)
	}

	params := tco.Params{
		PurchasePriceCents:     applied.ConvertPriceCents(machine.PriceCNYCents),
		Years:                  request.Years,
		HoursPerYear:           request.HoursPerYear,
		FuelPricePerLitreCents: request.FuelPriceKZTCents,
		OperatorHourlyCents:    request.OperatorHourlyKZTCents,
		FuelConsumptionLPH:     machine.FuelConsumptionLPH,
		ServiceIntervalHours:   machine.ServiceIntervalHours,
		ResidualValueBps:       rate,
	}

	// Стоимость ТО хранится в долларах, как и цена машины: это цена запчастей
	// и работ у поставщика. В расчёт она входит уже приведённой к тенге тем же
	// курсом, что и покупка, — иначе две статьи одной сметы считались бы
	// в разных валютах.
	if machine.ServiceCostCNYCents != nil {
		serviceCostKZT := applied.ConvertPriceCents(*machine.ServiceCostCNYCents)
		params.ServiceCostCents = &serviceCostKZT
	}

	result, err := tco.Calculate(params)
	if err != nil {
		return nil, err
	}

	return &TCOResult{Machine: machine, FX: applied, Result: result}, nil
}

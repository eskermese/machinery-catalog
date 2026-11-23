package presenter

import (
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/tco"
)

// TCOItem одна статья стоимости владения.
//
// Сумма необязательна: статья, которую не посчитали, приходит без неё.
// Ноль сюда не подставляется — «нет данных» и «ноль расходов» это разные
// сообщения, и подмена первого вторым превращает расчёт в выдумку.
type TCOItem struct {
	// AmountKztCents сумма статьи за срок. Отсутствует — статья не посчитана.
	AmountKztCents *int64 `json:"amountKztCents,omitempty" example:"12000000"`
	// Missing величина, которой не хватило. Пусто — статья посчитана.
	//
	// Список, а не одно значение: статья «обслуживание» требует и интервала,
	// и стоимости, и не хватить может обоих сразу.
	Missing []string `json:"missing,omitempty" example:"fuel_consumption"`
}

// TCOResult ответ расчёта стоимости владения.
type TCOResult struct {
	// TotalHours наработка за весь срок в моточасах.
	TotalHours int `json:"totalHours" validate:"required" example:"6000"`

	// Purchase покупка машины: цена карточки, приведённая к тенге.
	Purchase TCOItem `json:"purchase" validate:"required"`
	// Fuel топливо за срок.
	Fuel TCOItem `json:"fuel" validate:"required"`
	// Service плановое обслуживание за срок.
	Service TCOItem `json:"service" validate:"required"`
	// Operator оплата труда оператора за срок.
	Operator TCOItem `json:"operator" validate:"required"`
	// ResidualValue остаточная стоимость на конец срока: вычитается из итога.
	ResidualValue TCOItem `json:"residualValue" validate:"required"`

	// ServicesCount число плановых обслуживаний за срок.
	ServicesCount *int `json:"servicesCount,omitempty" example:"12"`

	// TotalKztCents итог за срок: сумма посчитанных статей за вычетом
	// остаточной стоимости. Непосчитанные статьи в него не входят — читать
	// итог, не посмотрев на missing, нельзя.
	TotalKztCents int64 `json:"totalKztCents" validate:"required" example:"20860000"`
	// CostPerHourKztCents стоимость моточаса.
	CostPerHourKztCents int64 `json:"costPerHourKztCents" validate:"required" example:"347700"`

	// Complete посчитаны ли все статьи.
	//
	// Отдельным полем, хотя выводится из missing: клиент решает по нему,
	// показывать ли расчёт вообще, и заставлять его собирать это из пяти
	// вложенных списков значило бы разложить одно решение по пяти местам.
	Complete bool `json:"complete" validate:"required" example:"false"`
	// Missing все недостающие величины одним списком.
	Missing []string `json:"missing" validate:"required" example:"fuel_consumption"`

	// FX курс, которым цена машины приведена к тенге.
	FX QuoteFX `json:"fx" validate:"required"`
	// Machine карточка, по которой считали.
	MachineRef *MachineRef `json:"machineRef,omitempty"`
}

// NewTCOResult создаёт представление расчёта стоимости владения.
func NewTCOResult(
	machine *entity.Machine,
	categorySlug string,
	applied entity.AppliedFXRate,
	result *tco.Result,
) TCOResult {
	view := TCOResult{
		TotalHours:          result.TotalHours,
		Purchase:            TCOItem{AmountKztCents: &result.PurchaseCents},
		Fuel:                item(result.FuelCents, missingOf(result, tco.MissingFuelConsumption)),
		Service:             serviceItem(result),
		Operator:            TCOItem{AmountKztCents: &result.OperatorCents},
		ResidualValue:       item(result.ResidualValueCents, missingOf(result, tco.MissingResidualValue)),
		ServicesCount:       result.ServicesCount,
		TotalKztCents:       result.TotalCents,
		CostPerHourKztCents: result.CostPerHourCents,
		Complete:            result.Complete(),
		Missing:             missingStrings(result.Missing),
		FX: QuoteFX{
			Rate:      applied.Rate.StringFixed(entity.FXRateScale),
			RateDate:  applied.RateDate,
			MarkupBps: applied.MarkupBps,
		},
	}

	if machine != nil && categorySlug != "" {
		view.MachineRef = &MachineRef{
			ID:           machine.ID,
			Slug:         machine.Slug,
			CategorySlug: categorySlug,
			NameRU:       machine.NameRU,
		}
	}

	return view
}

// item собирает статью: сумму либо перечень недостающего.
func item(amount *int64, missing []string) TCOItem {
	return TCOItem{AmountKztCents: amount, Missing: missing}
}

// serviceItem собирает статью обслуживания.
//
// Отдельно от остальных: ей нужны две величины сразу, и не хватить может
// как одной, так и обеих.
func serviceItem(result *tco.Result) TCOItem {
	missing := missingOf(result, tco.MissingServiceInterval, tco.MissingServiceCost)

	return TCOItem{AmountKztCents: result.ServiceCents, Missing: missing}
}

// missingOf отбирает из перечня расчёта те величины, что относятся к статье.
func missingOf(result *tco.Result, want ...tco.Missing) []string {
	var found []string

	for _, missing := range result.Missing {
		for _, candidate := range want {
			if missing == candidate {
				found = append(found, missing.String())
			}
		}
	}

	return found
}

// missingStrings переводит перечень недостающего в строки.
//
// Пустой срез, а не nil: поле объявлено обязательным, и клиенту проще
// работать с пустым списком, чем с отсутствующим полем.
func missingStrings(missing []tco.Missing) []string {
	items := make([]string, 0, len(missing))
	for _, value := range missing {
		items = append(items, value.String())
	}

	return items
}

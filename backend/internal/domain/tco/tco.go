// Package tco считает стоимость владения техникой за срок.
//
// Пакет намеренно чистый, по образцу leasing: он не знает ни о хранилище,
// ни о транспорте, не обращается ни к часам, ни к сети. Один и тот же вход
// всегда даёт один и тот же выход.
//
// Границы пакета работают в минорных единицах (int64) и целых базисных
// пунктах; decimal живёт только внутри расчёта.
//
// Главная особенность против leasing — отсутствующие данные. Расход топлива,
// интервал ТО и остаточная стоимость у карточки могут быть не заведены,
// и подставлять вместо них «типичное» значение нельзя: расчёт с выдуманным
// расходом выглядит как расчёт, а является догадкой, и это хуже, чем его
// отсутствие. Поэтому каждая статья расходов необязательна, а результат несёт
// перечень того, чего не хватило.
package tco

import (
	"github.com/shopspring/decimal"
)

const (
	// bpsInWhole базисных пунктов в единице: 10000 bps = 100%.
	bpsInWhole = 10000
	// maxYears верхняя граница срока владения.
	//
	// Ограничение техническое: срок умножается на годовую наработку, и без
	// границы вход вроде 10^9 лет даёт переполнение вместо ответа.
	maxYears = 30
	// maxHoursPerYear верхняя граница годовой наработки.
	//
	// В году 8760 часов; машина, работающая больше, — это опечатка,
	// а не режим эксплуатации.
	maxHoursPerYear = 8760
	// internalPrecision знаков после запятой во внутренних вычислениях.
	internalPrecision = 28
	// centsScale число знаков после запятой у минорной единицы.
	centsScale = 0
)

// Missing величина, без которой статья расходов не считается.
type Missing string

// Перечень величин, которых может не хватить.
//
// Строками, а не числами: они уходят наружу в ответе, и клиенту нужно
// показать человеку, чего именно не хватило.
const (
	// MissingFuelConsumption расход топлива на моточас не заведён у карточки.
	MissingFuelConsumption Missing = "fuel_consumption"
	// MissingServiceInterval интервал планового ТО не заведён у карточки.
	MissingServiceInterval Missing = "service_interval"
	// MissingServiceCost стоимость планового ТО не заведена у карточки.
	MissingServiceCost Missing = "service_cost"
	// MissingResidualValue доля остаточной стоимости не заведена для раздела.
	MissingResidualValue Missing = "residual_value"
)

// String возвращает строковое представление недостающей величины.
func (m Missing) String() string { return string(m) }

// Params входные параметры расчёта.
//
// Денежные величины — в минорных единицах одной валюты; пакет валюту не знает
// и не пересчитывает. Величины, приходящие от карточки, — указатели: nil
// означает «не заведено», и это не то же самое, что ноль.
type Params struct {
	// PurchasePriceCents цена покупки машины.
	PurchasePriceCents int64
	// Years срок владения в годах.
	Years int
	// HoursPerYear наработка в моточасах за год.
	HoursPerYear int
	// FuelPricePerLitreCents цена литра топлива.
	FuelPricePerLitreCents int64
	// OperatorHourlyCents зарплата оператора за моточас.
	OperatorHourlyCents int64

	// FuelConsumptionLPH расход топлива в литрах на моточас. nil — не заведён.
	FuelConsumptionLPH *decimal.Decimal
	// ServiceIntervalHours моточасов между плановыми ТО. nil — не заведён.
	ServiceIntervalHours *int
	// ServiceCostCents стоимость одного планового ТО. nil — не заведена.
	ServiceCostCents *int64
	// ResidualValueBps доля цены, сохраняемая к концу срока. nil — не заведена.
	ResidualValueBps *int
}

// Result результат расчёта.
//
// Каждая статья расходов необязательна: nil означает, что её не посчитали,
// и вызывающий код обязан показать это словами, а не нулём.
type Result struct {
	// PurchaseCents цена покупки. Считается всегда: она известна по карточке.
	PurchaseCents int64
	// FuelCents расходы на топливо за срок. nil — не хватило расхода топлива.
	FuelCents *int64
	// ServiceCents расходы на плановое ТО за срок.
	// nil — не хватило интервала или стоимости ТО.
	ServiceCents *int64
	// OperatorCents оплата труда оператора за срок. Считается всегда:
	// ставка приходит от клиента.
	OperatorCents int64
	// ResidualValueCents остаточная стоимость на конец срока — она вычитается
	// из итога. nil — доля остаточной стоимости не заведена.
	ResidualValueCents *int64

	// ServicesCount число плановых ТО за срок. nil вместе с ServiceCents.
	ServicesCount *int
	// TotalHours наработка за весь срок в моточасах.
	TotalHours int

	// TotalCents итог за срок: покупка плюс посчитанные статьи минус
	// остаточная стоимость. Непосчитанные статьи в него не входят —
	// см. Missing, без которого итог читать нельзя.
	TotalCents int64
	// CostPerHourCents стоимость моточаса: итог, делённый на наработку.
	CostPerHourCents int64

	// Missing величины, которых не хватило. Пустой список — посчитано всё.
	Missing []Missing
}

// Complete сообщает, что расчёт полон: ни одной величины не не хватило.
func (r *Result) Complete() bool { return len(r.Missing) == 0 }

// Calculate считает стоимость владения за срок.
//
// Возвращает ошибку только на недопустимом вводе клиента. Незаведённые
// величины карточки ошибкой не считаются: они дают неполный результат,
// а не отказ, — иначе машина без расхода топлива не показала бы даже
// стоимости оператора, которую посчитать можно.
func Calculate(params Params) (Result, error) {
	if err := params.validate(); err != nil {
		return Result{}, err
	}

	totalHours := params.Years * params.HoursPerYear

	result := Result{
		PurchaseCents: params.PurchasePriceCents,
		OperatorCents: params.OperatorHourlyCents * int64(totalHours),
		TotalHours:    totalHours,
	}

	result.FuelCents, result.Missing = fuelCost(params, totalHours, result.Missing)
	result.ServiceCents, result.ServicesCount, result.Missing = serviceCost(params, totalHours, result.Missing)
	result.ResidualValueCents, result.Missing = residualValue(params, result.Missing)

	result.TotalCents = total(&result)
	result.CostPerHourCents = perHour(result.TotalCents, totalHours)

	return result, nil
}

// fuelCost считает расходы на топливо за срок.
func fuelCost(params Params, totalHours int, missing []Missing) (*int64, []Missing) {
	if params.FuelConsumptionLPH == nil {
		return nil, append(missing, MissingFuelConsumption)
	}

	liters := params.FuelConsumptionLPH.Mul(decimal.NewFromInt(int64(totalHours)))
	cost := liters.Mul(decimal.NewFromInt(params.FuelPricePerLitreCents)).
		Round(centsScale).
		IntPart()

	return &cost, missing
}

// serviceCost считает расходы на плановое ТО за срок.
//
// Число обслуживаний — целая часть от деления наработки на интервал:
// оплачивается проведённое ТО, а не начатый интервал.
func serviceCost(
	params Params,
	totalHours int,
	missing []Missing,
) (cost *int64, count *int, stillMissing []Missing) {
	if params.ServiceIntervalHours == nil {
		missing = append(missing, MissingServiceInterval)
	}

	if params.ServiceCostCents == nil {
		missing = append(missing, MissingServiceCost)
	}

	if params.ServiceIntervalHours == nil || params.ServiceCostCents == nil {
		return nil, nil, missing
	}

	services := totalHours / *params.ServiceIntervalHours
	amount := int64(services) * *params.ServiceCostCents

	return &amount, &services, missing
}

// residualValue считает остаточную стоимость на конец срока.
func residualValue(params Params, missing []Missing) (*int64, []Missing) {
	if params.ResidualValueBps == nil {
		return nil, append(missing, MissingResidualValue)
	}

	value := decimal.NewFromInt(params.PurchasePriceCents).
		Mul(decimal.NewFromInt(int64(*params.ResidualValueBps))).
		DivRound(decimal.NewFromInt(bpsInWhole), internalPrecision).
		Round(centsScale).
		IntPart()

	return &value, missing
}

// total складывает посчитанные статьи.
//
// Непосчитанная статья в итог не входит и нулём не подменяется: ноль
// в сумме неотличим от «этой статьи расходов нет», а разница между
// «не знаем» и «нет» здесь и есть весь смысл.
func total(result *Result) int64 {
	sum := result.PurchaseCents + result.OperatorCents

	if result.FuelCents != nil {
		sum += *result.FuelCents
	}

	if result.ServiceCents != nil {
		sum += *result.ServiceCents
	}

	if result.ResidualValueCents != nil {
		sum -= *result.ResidualValueCents
	}

	return sum
}

// perHour считает стоимость моточаса.
func perHour(totalCents int64, totalHours int) int64 {
	if totalHours == 0 {
		return 0
	}

	return decimal.NewFromInt(totalCents).
		DivRound(decimal.NewFromInt(int64(totalHours)), internalPrecision).
		Round(centsScale).
		IntPart()
}

// validate проверяет то, что приходит от клиента.
//
// Величины карточки здесь не проверяются: их отсутствие — не ошибка ввода,
// а состояние данных, и отвечает за него Missing.
func (p Params) validate() error {
	if err := p.validateInput(); err != nil {
		return err
	}

	return p.validateCardValues()
}

// validateInput проверяет то, что задал клиент.
func (p Params) validateInput() error {
	if p.PurchasePriceCents <= 0 {
		return ErrPurchasePriceInvalid
	}

	if p.Years <= 0 || p.Years > maxYears {
		return ErrYearsInvalid
	}

	if p.HoursPerYear <= 0 || p.HoursPerYear > maxHoursPerYear {
		return ErrHoursPerYearInvalid
	}

	if p.FuelPricePerLitreCents < 0 {
		return ErrFuelPriceInvalid
	}

	if p.OperatorHourlyCents < 0 {
		return ErrOperatorRateInvalid
	}

	return nil
}

// validateCardValues проверяет заведённые величины карточки.
//
// Их ОТСУТСТВИЕ проверкой не считается — оно даёт неполный расчёт, а не отказ.
// Здесь ловится другое: величина заведена, но бессмысленна, — нулевой расход
// топлива или отрицательная доля остаточной стоимости. Такое приходит
// не от клиента, а из базы, и молча считать по нему нельзя.
func (p Params) validateCardValues() error {
	if p.FuelConsumptionLPH != nil && !p.FuelConsumptionLPH.IsPositive() {
		return ErrFuelConsumptionInvalid
	}

	if p.ServiceIntervalHours != nil && *p.ServiceIntervalHours <= 0 {
		return ErrServiceIntervalInvalid
	}

	if p.ResidualValueBps != nil && (*p.ResidualValueBps < 0 || *p.ResidualValueBps > bpsInWhole) {
		return ErrResidualRateInvalid
	}

	return nil
}

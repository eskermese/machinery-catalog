// Package leasing считает график лизинговых платежей.
//
// Пакет намеренно чистый: он не знает ни о хранилище, ни о транспорте, ни
// о конфигурации, и не обращается ни к часам, ни к сети. Один и тот же вход
// всегда даёт один и тот же выход — иначе выпущенное коммерческое предложение
// нельзя было бы пересчитать и проверить.
//
// Границы пакета работают в минорных единицах (int64) и целых базисных пунктах.
// decimal живёт только внутри расчёта: наружу он не протекает, чтобы
// вызывающий код не получал вместе с числом ещё и вопрос о его точности.
package leasing

import (
	"github.com/shopspring/decimal"
)

const (
	// bpsInWhole базисных пунктов в единице: 10000 bps = 100% = 1.0.
	bpsInWhole = 10000
	// maxDownPaymentBps соответствует взносу 100%.
	maxDownPaymentBps = 100 * 100
	// monthsInYear месяцев в году: годовая ставка приводится к месячной.
	monthsInYear = 12
	// maxTermMonths верхняя граница срока (100 лет).
	//
	// Ограничение техническое, а не бизнесовое: срок входит в показатель степени,
	// и без верхней границы вход вроде 10^9 месяцев занимает память и время
	// на возведении в степень, не давая осмысленного результата.
	maxTermMonths = 1200
	// internalPrecision знаков после запятой во внутренних вычислениях.
	//
	// Запас к минорной единице огромный: ошибка в 28-м знаке не может сдвинуть
	// округление до цента ни на одном шаге графика.
	internalPrecision = 28
	// centsScale число знаков после запятой у минорной единицы.
	centsScale = 0
)

// ScheduleType тип графика платежей.
type ScheduleType string

// Типы графиков.
const (
	// ScheduleAnnuity аннуитет: сумма платежа постоянна, доля процентов убывает.
	ScheduleAnnuity ScheduleType = "annuity"
	// ScheduleDifferentiated дифференцированный: тело гасится равными долями,
	// сумма платежа убывает вместе с остатком долга.
	ScheduleDifferentiated ScheduleType = "differentiated"
)

// String возвращает строковое представление типа графика.
func (t ScheduleType) String() string { return string(t) }

// Params входные параметры расчёта.
//
// Все денежные величины — в минорных единицах той валюты, в которой ведётся
// сделка; калькулятор валюту не знает и не пересчитывает. Взнос и ставка —
// в целых базисных пунктах: 20.00% из схемы БД это 2000 bps.
type Params struct {
	// AssetCostCents стоимость предмета лизинга.
	AssetCostCents int64
	// DownPaymentBps первоначальный взнос в базисных пунктах, 0..10000.
	DownPaymentBps int
	// TermMonths срок лизинга в месяцах.
	TermMonths int
	// AnnualRateBps годовая ставка в базисных пунктах.
	AnnualRateBps int
	// Schedule тип графика платежей.
	Schedule ScheduleType
	// ResidualValueCents выкупная стоимость. Не амортизируется в графике
	// и гасится последним платежом. Ноль — выкуп не предусмотрен.
	ResidualValueCents int64
	// CommissionCents разовая комиссия за организацию сделки.
	// В тело финансирования не входит: она уплачивается сверх графика.
	CommissionCents int64
}

// Payment строка графика платежей.
type Payment struct {
	// Number порядковый номер платежа, начиная с единицы.
	Number int
	// PrincipalCents часть платежа, гасящая основной долг.
	PrincipalCents int64
	// InterestCents часть платежа, приходящаяся на проценты.
	InterestCents int64
	// AmountCents сумма платежа целиком.
	AmountCents int64
	// BalanceCents остаток долга после платежа.
	BalanceCents int64
}

// Result результат расчёта.
type Result struct {
	// DownPaymentCents первоначальный взнос, посчитанный от стоимости.
	DownPaymentCents int64
	// FinancedAmountCents сумма финансирования: стоимость за вычетом взноса.
	FinancedAmountCents int64
	// FirstPaymentCents сумма первого платежа.
	// Для аннуитета это и есть регулярный ежемесячный платёж.
	FirstPaymentCents int64
	// LastPaymentCents сумма последнего платежа. У аннуитета отличается от
	// регулярного на выкупной платёж и накопленную разницу округления.
	LastPaymentCents int64
	// Schedule полный график платежей.
	Schedule []Payment
	// TotalInterestCents проценты за весь срок.
	TotalInterestCents int64
	// TotalPaidCents итого выплат по графику плюс комиссия.
	// Первоначальный взнос сюда не входит — он учтён отдельным полем.
	TotalPaidCents int64
	// OverpaymentCents переплата: всё, что уплачено сверх суммы финансирования,
	// то есть проценты вместе с комиссией.
	OverpaymentCents int64
}

// Calculate считает график лизинговых платежей по параметрам сделки.
//
// Возвращает ошибку на любом недопустимом входе и никогда не паникует:
// параметры приходят из формы, и падение процесса на «сроке -1» недопустимо.
func Calculate(params Params) (Result, error) {
	downPayment, financed, err := params.amounts()
	if err != nil {
		return Result{}, err
	}

	monthlyRate := monthlyRate(params.AnnualRateBps)

	var schedule []Payment

	switch params.Schedule {
	case ScheduleAnnuity:
		schedule = annuitySchedule(financed, params.ResidualValueCents, monthlyRate, params.TermMonths)
	case ScheduleDifferentiated:
		schedule = differentiatedSchedule(financed, params.ResidualValueCents, monthlyRate, params.TermMonths)
	default:
		return Result{}, ErrScheduleTypeInvalid
	}

	return newResult(downPayment, financed, params.CommissionCents, schedule), nil
}

// amounts проверяет параметры и возвращает первоначальный взнос и сумму финансирования.
//
// Проверка выкупной стоимости живёт здесь, а не в validate: она сравнивается
// с суммой финансирования, а та сама выводится из стоимости и взноса.
func (p Params) amounts() (downPayment, financed int64, err error) {
	if err := p.validate(); err != nil {
		return 0, 0, err
	}

	downPayment = downPaymentAmount(p.AssetCostCents, p.DownPaymentBps)
	financed = p.AssetCostCents - downPayment

	if p.ResidualValueCents > financed {
		return 0, 0, ErrResidualValueInvalid
	}

	return downPayment, financed, nil
}

// validate проверяет диапазоны входных параметров.
//
// Тип графика здесь не проверяется: он разбирается switch-ем в Calculate,
// и вторая проверка сделала бы одну из двух веток недостижимой.
// Согласованность выкупной стоимости с суммой финансирования тоже не здесь —
// сумма финансирования это уже результат расчёта, а не вход.
func (p Params) validate() error {
	if p.AssetCostCents < 0 {
		return ErrAssetCostInvalid
	}

	if p.DownPaymentBps < 0 || p.DownPaymentBps > maxDownPaymentBps {
		return ErrDownPaymentInvalid
	}

	if p.TermMonths <= 0 || p.TermMonths > maxTermMonths {
		return ErrTermInvalid
	}

	if p.AnnualRateBps < 0 {
		return ErrRateInvalid
	}

	if p.ResidualValueCents < 0 {
		return ErrResidualValueInvalid
	}

	if p.CommissionCents < 0 {
		return ErrCommissionInvalid
	}

	return nil
}

// newResult собирает итоги по готовому графику.
func newResult(downPayment, financed, commission int64, schedule []Payment) Result {
	var totalInterest, totalPayments int64

	for _, payment := range schedule {
		totalInterest += payment.InterestCents
		totalPayments += payment.AmountCents
	}

	totalPaid := totalPayments + commission

	return Result{
		DownPaymentCents:    downPayment,
		FinancedAmountCents: financed,
		FirstPaymentCents:   schedule[0].AmountCents,
		LastPaymentCents:    schedule[len(schedule)-1].AmountCents,
		Schedule:            schedule,
		TotalInterestCents:  totalInterest,
		TotalPaidCents:      totalPaid,
		OverpaymentCents:    totalPaid - financed,
	}
}

// downPaymentAmount считает первоначальный взнос от стоимости предмета лизинга.
func downPaymentAmount(assetCost int64, downPaymentBps int) int64 {
	if downPaymentBps == 0 {
		return 0
	}

	if downPaymentBps == maxDownPaymentBps {
		return assetCost
	}

	return toCents(
		decimal.NewFromInt(assetCost).
			Mul(decimal.NewFromInt(int64(downPaymentBps))).
			DivRound(decimal.NewFromInt(bpsInWhole), internalPrecision),
	)
}

// monthlyRate приводит годовую ставку в базисных пунктах к месячной доле.
//
// Деление годовой ставки на 12 — простая пропорция, а не приведение по формуле
// сложного процента: именно так ставка объявляется в договоре лизинга,
// и график должен сходиться с тем, что подписал клиент.
func monthlyRate(annualRateBps int) decimal.Decimal {
	if annualRateBps == 0 {
		return decimal.Zero
	}

	return decimal.NewFromInt(int64(annualRateBps)).
		DivRound(decimal.NewFromInt(bpsInWhole*monthsInYear), internalPrecision)
}

// toCents округляет внутреннее значение до минорной единицы.
//
// Округление половин от нуля (0.5 → 1, -0.5 → -1), а не банковское: график
// сверяют вручную на калькуляторе, и привычное арифметическое округление
// не вызывает вопросов там, где банковское разошлось бы на цент.
func toCents(value decimal.Decimal) int64 {
	return value.Round(centsScale).IntPart()
}

package entity

import (
	"github.com/shopspring/decimal"
)

// bpsScale базисные пункты в единице: 10000 bps = 100%.
const bpsScale = 10_000

// residualPrecision разрядность промежуточных вычислений кривой.
//
// Столько же, сколько в расчётах лизинга и стоимости владения: возведение
// в степень на длинном горизонте копит ошибку быстрее всего остального.
const residualPrecision = 28

// ResidualValueCurve кривая падения остаточной стоимости раздела.
//
// Кривая, а не таблица «год — доля», и это главное решение здесь.
// Строка на год означала бы тридцать чисел на раздел и больше тысячи
// на каталог: столько руками не заводят, таблица осталась бы наполовину
// пустой, а расчёт прыгал бы между годами без объяснимой причины.
//
// Тремя числами кривая задаётся так, как остаточную стоимость и называют
// в торговле: «за первый год теряет столько-то, дальше столько-то в год,
// ниже такой-то доли не падает». Она монотонна по построению — перепутать
// пятый год с четвёртым нечем — и продолжается на любой горизонт: спросили
// не пять лет, а десять, и новых данных для этого не нужно.
type ResidualValueCurve struct {
	// CategoryID раздел, для которого заведена кривая.
	CategoryID string
	// FirstYearDropBps сколько техника теряет за первый год.
	//
	// Отдельно от остальных лет не ради точности формулы: первый год
	// техника теряет заметно больше следующих, и одна ставка на все годы
	// либо завысила бы остаток свежей машины, либо занизила бы старой.
	FirstYearDropBps int
	// AnnualDropBps сколько теряется за каждый следующий год ОТ ОСТАТКА.
	AnnualDropBps int
	// FloorBps доля, ниже которой стоимость не опускается.
	//
	// Чистая геометрия на длинном горизонте уводит цену почти в ноль,
	// чего с техникой не происходит: у неё есть цена узлов и металла.
	FloorBps int
}

// Validate проверяет кривую.
func (c *ResidualValueCurve) Validate() error {
	if c == nil {
		return ErrNilPointer
	}

	if c.CategoryID == "" {
		return ErrResidualCurveCategoryRequired
	}

	if !inBps(c.FirstYearDropBps) || !inBps(c.AnnualDropBps) || !inBps(c.FloorBps) {
		return ErrResidualCurveBpsOutOfRange
	}

	// Порог выше того, что остаётся после первого года, сделал бы кривую
	// возрастающей: машина «дорожала» бы начиная со второго года.
	if c.FloorBps > bpsScale-c.FirstYearDropBps {
		return ErrResidualCurveFloorAboveFirstYear
	}

	return nil
}

// RateFor возвращает долю цены, которую техника сохраняет к концу N-го года,
// в базисных пунктах.
//
// Год ноль — это машина, которую ещё не эксплуатировали: она стоит своих
// ста процентов, и порог к ней не применяется. Дальше от остатка после
// первого года каждый год отнимается AnnualDropBps, пока не упрёмся в порог.
func (c *ResidualValueCurve) RateFor(ageYears int) int {
	if c == nil || ageYears <= 0 {
		return bpsScale
	}

	remaining := decimal.NewFromInt(int64(bpsScale - c.FirstYearDropBps))

	if ageYears > 1 {
		keep := decimal.NewFromInt(int64(bpsScale - c.AnnualDropBps)).
			Div(decimal.NewFromInt(bpsScale))

		remaining = remaining.Mul(keep.Pow(decimal.NewFromInt(int64(ageYears - 1)))).
			Round(residualPrecision)
	}

	// Дробная часть отбрасывается, а не округляется: заниженная остаточная
	// стоимость завышает стоимость владения, и ошибаться лучше в эту сторону.
	rate := int(remaining.IntPart())
	if rate < c.FloorBps {
		return c.FloorBps
	}

	return rate
}

// inBps сообщает, что значение лежит в границах базисных пунктов.
func inBps(value int) bool {
	return value >= 0 && value <= bpsScale
}

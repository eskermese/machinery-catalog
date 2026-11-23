package presenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
)

// testCurve кривая «за первый год минус 20%, дальше минус 12% в год,
// ниже 20% не падает».
func testCurve() *entity.ResidualValueCurve {
	return &entity.ResidualValueCurve{
		CategoryID:       "44444444-4444-4444-8444-444444444444",
		FirstYearDropBps: 2000,
		AnnualDropBps:    1200,
		FloorBps:         2000,
	}
}

// TestNewResidualValueCurve проверяет, что вместе с параметрами уходит
// посчитанная таблица долей по годам.
//
// Таблица здесь не украшение: «минус 12% от остатка» не даёт понять, сколько
// машина будет стоить через пять лет, а кривую заводят именно ради этого.
// Считает её сервер той же формулой, что и расчёт владения, чтобы на экране
// и в расчёте не оказалось двух разных цифр.
func TestNewResidualValueCurve(t *testing.T) {
	t.Parallel()

	view := presenter.NewResidualValueCurve(testCurve())

	require.NotNil(t, view)
	assert.Equal(t, 2000, view.FirstYearDropBps)
	assert.Equal(t, 1200, view.AnnualDropBps)
	assert.Equal(t, 2000, view.FloorBps)

	require.Len(t, view.Preview, 10, "горизонт таблицы — десять лет")
	assert.Equal(t, 1, view.Preview[0].Year)
	assert.Equal(t, 8000, view.Preview[0].RateBps)
	assert.Equal(t, 5, view.Preview[4].Year)
	assert.Equal(t, 4797, view.Preview[4].RateBps)

	// Таблица идёт по убыванию: это и есть то свойство, ради которого кривая
	// выбрана вместо строки на год.
	for i := 1; i < len(view.Preview); i++ {
		assert.LessOrEqual(t, view.Preview[i].RateBps, view.Preview[i-1].RateBps)
	}
}

// TestNewResidualValueCurve_Nil отдаёт пустоту, а не панику: раздел без своей
// кривой — законное состояние.
func TestNewResidualValueCurve_Nil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, presenter.NewResidualValueCurve(nil))
}

// TestNewResidualValueCurveList собирает список.
func TestNewResidualValueCurveList(t *testing.T) {
	t.Parallel()

	list := presenter.NewResidualValueCurveList([]*entity.ResidualValueCurve{testCurve()})

	require.Len(t, list.Items, 1)
	assert.Equal(t, "44444444-4444-4444-8444-444444444444", list.Items[0].CategoryID)
}

// TestNewResidualValueCurveList_Empty отдаёт пустой список, а не null:
// клиент обходит его циклом, и null сломал бы обход.
func TestNewResidualValueCurveList_Empty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, presenter.NewResidualValueCurveList(nil).Items)
	assert.NotNil(t, presenter.NewResidualValueCurveList(nil).Items)
}

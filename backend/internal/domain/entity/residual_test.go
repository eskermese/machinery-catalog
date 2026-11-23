package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
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

// TestResidualValueCurve_RateFor проходит кривую по годам.
func TestResidualValueCurve_RateFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		age      int
		expected int
	}{
		// Не эксплуатировали — машина стоит своих ста процентов.
		{"нулевой год", 0, 10_000},
		{"отрицательный возраст", -3, 10_000},
		// Первый год снимает ровно FirstYearDropBps.
		{"первый год", 1, 8_000},
		// Дальше 12% от ОСТАТКА, а не от исходной цены: 8000 * 0.88.
		{"второй год", 2, 7_040},
		{"третий год", 3, 6_195},
		{"пятый год", 5, 4_797},
		{"десятый год", 10, 2_531},
		// На длинном горизонте геометрия ушла бы почти в ноль, но порог держит.
		{"двадцатый год", 20, 2_000},
		{"тридцатый год", 30, 2_000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.expected, testCurve().RateFor(c.age))
		})
	}
}

// TestResidualValueCurve_RateForIsMonotonic фиксирует свойство, ради которого
// кривая и выбрана вместо строки на год: перепутать порядок лет нечем.
func TestResidualValueCurve_RateForIsMonotonic(t *testing.T) {
	t.Parallel()

	curve := testCurve()

	previous := curve.RateFor(1)
	for age := 2; age <= 30; age++ {
		current := curve.RateFor(age)
		assert.LessOrEqual(t, current, previous, "год %d не может стоить дороже предыдущего", age)

		previous = current
	}
}

// TestResidualValueCurve_RateForNil защищает от паники на незаведённой кривой.
func TestResidualValueCurve_RateForNil(t *testing.T) {
	t.Parallel()

	var curve *entity.ResidualValueCurve

	assert.Equal(t, 10_000, curve.RateFor(5))
}

// TestResidualValueCurve_ZeroDrops проверяет вырожденный, но законный случай:
// техника не теряет в цене вовсе.
func TestResidualValueCurve_ZeroDrops(t *testing.T) {
	t.Parallel()

	curve := &entity.ResidualValueCurve{CategoryID: "c", FirstYearDropBps: 0, AnnualDropBps: 0}

	assert.Equal(t, 10_000, curve.RateFor(1))
	assert.Equal(t, 10_000, curve.RateFor(15))
}

// TestResidualValueCurve_Validate проверяет границы кривой.
func TestResidualValueCurve_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		mutate  func(*entity.ResidualValueCurve)
		wantErr error
	}{
		{"корректная", func(*entity.ResidualValueCurve) {}, nil},
		{
			"без раздела",
			func(c *entity.ResidualValueCurve) { c.CategoryID = "" },
			entity.ErrResidualCurveCategoryRequired,
		},
		{
			"первый год выше ста процентов",
			func(c *entity.ResidualValueCurve) { c.FirstYearDropBps = 10_001 },
			entity.ErrResidualCurveBpsOutOfRange,
		},
		{
			"отрицательная годовая доля",
			func(c *entity.ResidualValueCurve) { c.AnnualDropBps = -1 },
			entity.ErrResidualCurveBpsOutOfRange,
		},
		{
			"порог вне границ",
			func(c *entity.ResidualValueCurve) { c.FloorBps = 10_001 },
			entity.ErrResidualCurveBpsOutOfRange,
		},
		{
			// 10000-2000 = 8000 остаётся после первого года, порог 8500 выше.
			"порог выше остатка первого года",
			func(c *entity.ResidualValueCurve) { c.FloorBps = 8_500 },
			entity.ErrResidualCurveFloorAboveFirstYear,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			curve := testCurve()
			c.mutate(curve)

			err := curve.Validate()
			if c.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.wantErr)
		})
	}
}

// TestResidualValueCurve_ValidateNil проверяет отказ на пустом указателе.
func TestResidualValueCurve_ValidateNil(t *testing.T) {
	t.Parallel()

	var curve *entity.ResidualValueCurve

	require.ErrorIs(t, curve.Validate(), entity.ErrNilPointer)
}

package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestFXPairStatus_HasRate фиксирует различие «курса нет» и «курс нулевой».
//
// Нулевой курс не курс: по нему цена в тенге вышла бы нулём, и витрина
// показала бы «0 ₸» вместо честного молчания.
func TestFXPairStatus_HasRate(t *testing.T) {
	t.Parallel()

	var missing *FXPairStatus
	assert.False(t, missing.HasRate(), "статуса нет — курса тем более")

	assert.False(t, (&FXPairStatus{}).HasRate(), "курс не заполнен")

	zero := &FXPairStatus{Effective: &AppliedFXRate{Rate: decimal.Zero}}
	assert.False(t, zero.HasRate(), "нулевой курс не курс")

	real := &FXPairStatus{Effective: &AppliedFXRate{Rate: decimal.RequireFromString("72.55")}}
	assert.True(t, real.HasRate())
}

// TestIsRateStale фиксирует, что возраст курса меряется КАЛЕНДАРНЫМИ сутками.
//
// Источник публикует курс на сутки, и «22 часа назад» для него не значит
// ничего: вчерашний курс вчерашний независимо от того, в котором часу
// его смотрят.
func TestIsRateStale(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	day := func(offset int) time.Time { return now.AddDate(0, 0, offset) }

	cases := []struct {
		name           string
		rateDate       time.Time
		staleAfterDays int
		want           bool
	}{
		{name: "сегодняшний", rateDate: now, staleAfterDays: 3},
		{
			name:     "вечерний вчерашний — всё ещё сутки, а не 22 часа",
			rateDate: time.Date(2026, 8, 31, 23, 0, 0, 0, time.UTC), staleAfterDays: 3,
		},
		{name: "ровно на пороге", rateDate: day(-3), staleAfterDays: 3},
		{name: "за порогом", rateDate: day(-4), staleAfterDays: 3, want: true},
		{name: "задан вперёд", rateDate: day(2), staleAfterDays: 3},
		{
			name:     "проверка выключена",
			rateDate: day(-100), staleAfterDays: 0,
		},
		{
			name:     "отрицательный порог тоже выключает",
			rateDate: day(-100), staleAfterDays: -1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.want, IsRateStale(c.rateDate, now, c.staleAfterDays))
		})
	}
}

package presenter

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

var fxDate = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// TestNewFXRate фиксирует форму курса на границе контракта: строка
// фиксированной точности, дата отдельно от момента получения.
//
// «72.55» и «72.550000» не должны быть двумя написаниями одного числа:
// клиент сравнивает строки, а не разбирает их обратно в дробь.
func TestNewFXRate(t *testing.T) {
	t.Parallel()

	rate := entity.NewFXRate(
		"CNY", "KZT", entity.FXSourceManual,
		decimal.RequireFromString("72.55"), fxDate,
		time.Date(2026, 9, 1, 9, 30, 0, 0, time.UTC),
	)

	view := NewFXRate(rate)
	assert.Equal(t, "CNY", view.BaseCurrency)
	assert.Equal(t, "KZT", view.QuoteCurrency)
	assert.Equal(t, entity.FXSourceManual, view.Source)
	assert.Equal(t, "72.550000", view.Rate)
	assert.Equal(t, "2026-09-01", view.RateDate, "дата курса — сутки, а не момент")
	assert.Equal(t, "2026-09-01T09:30:00Z", view.FetchedAt)

	assert.Equal(t, FXRate{}, NewFXRate(nil), "отсутствующий курс не роняет ответ")
}

// TestNewFXRateList фиксирует, что список переживает дыры.
func TestNewFXRateList(t *testing.T) {
	t.Parallel()

	rate := entity.NewFXRate(
		"CNY", "KZT", entity.FXSourceManual,
		decimal.RequireFromString("72.55"), fxDate, fxDate,
	)

	view := NewFXRateList(entity.FXRates{rate, nil})
	require.Len(t, view.Items, 1, "пустая строка выбрасывается, а не отдаётся пустым объектом")
	assert.Equal(t, "72.550000", view.Items[0].Rate)

	assert.Empty(t, NewFXRateList(nil).Items)
	assert.NotNil(t, NewFXRateList(nil).Items, "пустой список — [], а не null")
}

// TestNewFXMarkup фиксирует, что отсутствующий автор отдаётся пустой строкой,
// а не выдуманным идентификатором.
func TestNewFXMarkup(t *testing.T) {
	t.Parallel()

	author := "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	withAuthor := NewFXMarkup(entity.NewFXMarkup("CNY", "KZT", 200, fxDate, &author, fxDate))
	assert.Equal(t, 200, withAuthor.MarkupBps)
	assert.Equal(t, author, withAuthor.ChangedBy)
	assert.Equal(t, "2026-09-01T00:00:00Z", withAuthor.EffectiveFrom)

	// Наценку заводит и сам сервис при развёртывании — тогда автора нет.
	assert.Empty(t, NewFXMarkup(entity.NewFXMarkup("CNY", "KZT", 0, fxDate, nil, fxDate)).ChangedBy)

	assert.Equal(t, FXMarkup{}, NewFXMarkup(nil))
}

// TestNewFXMarkupList фиксирует форму списка наценок.
func TestNewFXMarkupList(t *testing.T) {
	t.Parallel()

	markup := entity.NewFXMarkup("CNY", "KZT", 200, fxDate, nil, fxDate)

	view := NewFXMarkupList(entity.FXMarkups{markup, nil})
	require.Len(t, view.Items, 1)
	assert.Equal(t, 200, view.Items[0].MarkupBps)

	assert.NotNil(t, NewFXMarkupList(nil).Items)
}

// TestNewFXStatusList фиксирует главное свойство ручки состояния: она
// показывает и то, что действует, и то, с чем это сравнивать.
func TestNewFXStatusList(t *testing.T) {
	t.Parallel()

	applied := entity.AppliedFXRate{
		Rate:      decimal.RequireFromString("74.001"),
		RateDate:  fxDate,
		MarkupBps: 200,
		Source:    entity.FXSourceManual,
	}

	full := &entity.FXPairStatus{
		BaseCurrency:    "CNY",
		QuoteCurrency:   "KZT",
		Effective:       &applied,
		EffectiveSource: entity.FXSourceManual,
		MarkupBps:       200,
		Stale:           true,
		Manual: entity.NewFXRate(
			"CNY", "KZT", entity.FXSourceManual,
			decimal.RequireFromString("72.55"), fxDate, fxDate,
		),
		Provider: entity.NewFXRate(
			"CNY", "KZT", "nationalbank",
			decimal.RequireFromString("70"), fxDate, fxDate,
		),
	}

	empty := &entity.FXPairStatus{BaseCurrency: "CNY", QuoteCurrency: "USD"}

	view := NewFXStatusList([]*entity.FXPairStatus{full, empty}, true)
	require.Len(t, view.Items, 2)
	assert.True(t, view.ShowUsdPrice)

	first := view.Items[0]
	assert.True(t, first.HasRate)
	assert.True(t, first.Stale)
	assert.Equal(t, "74.001000", first.EffectiveRate)
	assert.Equal(t, "2026-09-01", first.EffectiveRateDate)
	assert.Equal(t, entity.FXSourceManual, first.EffectiveSource)
	assert.Equal(t, 200, first.MarkupBps)
	require.NotNil(t, first.ManualRate)
	require.NotNil(t, first.ProviderRate)
	assert.Equal(t, "70.000000", first.ProviderRate.Rate)

	second := view.Items[1]
	assert.False(t, second.HasRate)
	assert.Empty(t, second.EffectiveRate)
	assert.Empty(t, second.EffectiveRateDate, "даты без курса не бывает")
	assert.Nil(t, second.ManualRate)
	assert.Nil(t, second.ProviderRate)
	assert.Equal(t, "CNY", second.BaseCurrency, "пара названа даже когда курса нет")
	assert.Equal(t, "USD", second.QuoteCurrency)

	assert.Equal(t, FXPairStatus{}, NewFXPairStatus(nil))
	assert.NotNil(t, NewFXStatusList(nil, false).Items)
}

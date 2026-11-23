package presenter

import (
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// FXRate представление курса валютной пары.
type FXRate struct {
	ID            string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	BaseCurrency  string `json:"baseCurrency" validate:"required" example:"CNY"`                        // Валюта, из которой считаем
	QuoteCurrency string `json:"quoteCurrency" validate:"required" example:"KZT"`                       // Валюта, в которую считаем
	Source        string `json:"source" validate:"required" example:"manual"`                           // Кто поставил курс
	Rate          string `json:"rate" validate:"required" example:"72.550000"`                          // Курс без наценки, строкой
	RateDate      string `json:"rateDate" validate:"required" example:"2026-09-01"`                     // Дата, на которую действует курс
	FetchedAt     string `json:"fetchedAt" validate:"required" example:"2026-09-01T09:00:00Z"`          // Момент получения
}

// NewFXRate создаёт представление курса.
func NewFXRate(rate *entity.FXRate) FXRate {
	if rate == nil {
		return FXRate{}
	}

	return FXRate{
		ID:            rate.ID,
		BaseCurrency:  rate.BaseCurrency,
		QuoteCurrency: rate.QuoteCurrency,
		Source:        rate.Source,
		// Строкой фиксированной точности: наружу decimal не уходит,
		// а «72.55» и «72.550000» не должны быть двумя написаниями одного.
		Rate:      rate.Rate.StringFixed(entity.FXRateScale),
		RateDate:  rate.RateDate.Format(time.DateOnly),
		FetchedAt: rate.FetchedAt.Format(time.RFC3339),
	}
}

// FXRateList ответ со списком курсов.
type FXRateList struct {
	Items []FXRate `json:"items" validate:"required"` // Курсы, свежие первыми
}

// NewFXRateList создаёт представление списка курсов.
func NewFXRateList(rates entity.FXRates) FXRateList {
	items := make([]FXRate, 0, len(rates))

	for _, rate := range rates {
		if rate == nil {
			continue
		}

		items = append(items, NewFXRate(rate))
	}

	return FXRateList{Items: items}
}

// FXMarkup представление наценки на валютную пару.
type FXMarkup struct {
	ID            string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	BaseCurrency  string `json:"baseCurrency" validate:"required" example:"CNY"`                        // Валюта, из которой считаем
	QuoteCurrency string `json:"quoteCurrency" validate:"required" example:"KZT"`                       // Валюта, в которую считаем
	MarkupBps     int    `json:"markupBps" validate:"required" example:"200"`                           // Наценка в базисных пунктах
	EffectiveFrom string `json:"effectiveFrom" validate:"required" example:"2026-09-01T00:00:00Z"`      // С какого момента действует
	ChangedBy     string `json:"changedBy" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`              // Кто задал; пусто — сервис
}

// NewFXMarkup создаёт представление наценки.
func NewFXMarkup(markup *entity.FXMarkup) FXMarkup {
	if markup == nil {
		return FXMarkup{}
	}

	view := FXMarkup{
		ID:            markup.ID,
		BaseCurrency:  markup.BaseCurrency,
		QuoteCurrency: markup.QuoteCurrency,
		MarkupBps:     markup.MarkupBps,
		EffectiveFrom: markup.EffectiveFrom.Format(time.RFC3339),
	}

	if markup.ChangedBy != nil {
		view.ChangedBy = *markup.ChangedBy
	}

	return view
}

// FXMarkupList ответ со списком наценок.
type FXMarkupList struct {
	Items []FXMarkup `json:"items" validate:"required"` // Наценки, свежие первыми
}

// NewFXMarkupList создаёт представление списка наценок.
func NewFXMarkupList(markups entity.FXMarkups) FXMarkupList {
	items := make([]FXMarkup, 0, len(markups))

	for _, markup := range markups {
		if markup == nil {
			continue
		}

		items = append(items, NewFXMarkup(markup))
	}

	return FXMarkupList{Items: items}
}

// FXPairStatus состояние курса одной пары.
//
// Отвечает на вопрос «почему витрина не показывает цену» до того, как его
// задаст клиент: видно и что действует, и что предложил источник, и что
// ввёл человек.
type FXPairStatus struct {
	BaseCurrency  string `json:"baseCurrency" validate:"required" example:"CNY"`  // Валюта, из которой считаем
	QuoteCurrency string `json:"quoteCurrency" validate:"required" example:"KZT"` // Валюта, в которую считаем

	// HasRate есть ли по паре действующий курс. False означает, что цена
	// в этой валюте на витрине не показывается вовсе.
	HasRate bool `json:"hasRate" validate:"required" example:"true"`
	// Stale курс есть, но старше допустимого возраста.
	Stale bool `json:"stale" example:"false"`
	// EffectiveRate применённый курс с наценкой — тот, по которому считается
	// цена. Пусто, если курса нет.
	EffectiveRate string `json:"effectiveRate" example:"74.001000"`
	// EffectiveRateDate дата действующего курса.
	EffectiveRateDate string `json:"effectiveRateDate" example:"2026-09-01"`
	// EffectiveSource кто поставил действующий курс.
	EffectiveSource string `json:"effectiveSource" example:"manual"`
	// MarkupBps действующая наценка на пару.
	MarkupBps int `json:"markupBps" example:"200"`

	// ProviderRate последний курс внешнего источника — то, что он предлагает.
	ProviderRate *FXRate `json:"providerRate,omitempty"`
	// ManualRate последний курс, введённый человеком.
	ManualRate *FXRate `json:"manualRate,omitempty"`
}

// NewFXPairStatus создаёт представление состояния пары.
func NewFXPairStatus(status *entity.FXPairStatus) FXPairStatus {
	if status == nil {
		return FXPairStatus{}
	}

	view := FXPairStatus{
		BaseCurrency:  status.BaseCurrency,
		QuoteCurrency: status.QuoteCurrency,
		HasRate:       status.HasRate(),
		Stale:         status.Stale,
		MarkupBps:     status.MarkupBps,
	}

	if status.Effective != nil {
		view.EffectiveRate = status.Effective.String()
		view.EffectiveRateDate = status.Effective.RateDate.Format(time.DateOnly)
		view.EffectiveSource = status.EffectiveSource
	}

	if status.Provider != nil {
		rate := NewFXRate(status.Provider)
		view.ProviderRate = &rate
	}

	if status.Manual != nil {
		rate := NewFXRate(status.Manual)
		view.ManualRate = &rate
	}

	return view
}

// FXStatusList ответ с состоянием курсов и настройками показа цены.
type FXStatusList struct {
	Items []FXPairStatus `json:"items" validate:"required"` // Состояние по каждой паре
	// ShowUsdPrice включён ли показ цены в долларах. Лежит здесь, а не
	// отдельной ручкой: админка показывает состояние пары CNY→USD и этот
	// флаг рядом — без него непонятно, важно ли отсутствие курса этой пары.
	ShowUsdPrice bool `json:"showUsdPrice" example:"false"`
}

// NewFXStatusList создаёт представление состояния курсов.
func NewFXStatusList(statuses []*entity.FXPairStatus, showUSD bool) FXStatusList {
	items := make([]FXPairStatus, 0, len(statuses))

	for _, status := range statuses {
		items = append(items, NewFXPairStatus(status))
	}

	return FXStatusList{Items: items, ShowUsdPrice: showUSD}
}

// PriceSettings настройки показа цены.
type PriceSettings struct {
	// ShowUSDPrice показывать ли цену в долларах на витрине.
	ShowUSDPrice bool `json:"showUsdPrice" example:"true"`
}

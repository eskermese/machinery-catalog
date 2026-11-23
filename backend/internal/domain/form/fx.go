package form

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// FXRateSave курс, введённый администратором вручную.
type FXRateSave struct {
	// BaseCurrency валюта, из которой считаем.
	BaseCurrency string `json:"baseCurrency" validate:"required,len=3" example:"CNY"`
	// QuoteCurrency валюта, в которую считаем.
	QuoteCurrency string `json:"quoteCurrency" validate:"required,len=3" example:"KZT"`
	// Rate курс строкой: сколько единиц котируемой валюты за одну базовую.
	//
	// Строкой, а не числом: JSON-число прошло бы через двоичную плавающую
	// точку и вернулось бы с хвостом в младшем разряде — а из этого курса
	// считается цена всего каталога.
	Rate string `json:"rate" validate:"required" example:"72.55"`
	// RateDate дата, на которую действует курс, в формате ГГГГ-ММ-ДД.
	// Пусто — сегодня.
	RateDate string `json:"rateDate" example:"2026-09-01"`
}

// ToEntity собирает курс ручного ввода.
//
// Источник проставляется здесь и не принимается снаружи: иначе клиент
// назначил бы своему курсу имя внешнего источника и подменил бы то,
// что загрузил воркер.
func (f FXRateSave) ToEntity(currentTime time.Time) (*entity.FXRate, error) {
	rate, err := decimal.NewFromString(f.Rate)
	if err != nil {
		return nil, entity.ErrFXRateInvalid
	}

	rateDate := currentTime

	if f.RateDate != "" {
		parsed, parseErr := time.Parse(time.DateOnly, f.RateDate)
		if parseErr != nil {
			return nil, entity.ErrFXRateDateInvalid
		}

		rateDate = parsed
	}

	return entity.NewFXRate(
		f.BaseCurrency,
		f.QuoteCurrency,
		entity.FXSourceManual,
		rate,
		rateDate,
		currentTime,
	), nil
}

// FXMarkupSave наценка на валютную пару.
type FXMarkupSave struct {
	// BaseCurrency валюта, из которой считаем.
	BaseCurrency string `json:"baseCurrency" validate:"required,len=3" example:"CNY"`
	// QuoteCurrency валюта, в которую считаем.
	QuoteCurrency string `json:"quoteCurrency" validate:"required,len=3" example:"KZT"`
	// MarkupBps наценка в базисных пунктах: 200 это 2 %.
	//
	// Базисные пункты, а не проценты дробью: целые участвуют в арифметике
	// без ошибки округления, и наценка оказывается в той же единице,
	// что ставка лизинга.
	MarkupBps int `json:"markupBps" validate:"gte=0" example:"200"`
	// EffectiveFrom момент вступления в силу; пусто — сейчас.
	EffectiveFrom *time.Time `json:"effectiveFrom"`
}

// ToEntity собирает наценку.
func (f FXMarkupSave) ToEntity(changedBy *string, currentTime time.Time) (*entity.FXMarkup, error) {
	effectiveFrom := currentTime
	if f.EffectiveFrom != nil {
		effectiveFrom = *f.EffectiveFrom
	}

	// Автор правки — указатель: наценку может завести и сам сервис
	// при развёртывании, и тогда автора нет.
	markup := entity.NewFXMarkup(
		f.BaseCurrency,
		f.QuoteCurrency,
		f.MarkupBps,
		effectiveFrom,
		changedBy,
		currentTime,
	)

	if err := markup.Validate(); err != nil {
		return nil, err
	}

	return markup, nil
}

// PriceSettingsSave настройки показа цены.
type PriceSettingsSave struct {
	// ShowUSDPrice показывать ли цену в долларах на витрине.
	ShowUSDPrice bool `json:"showUsdPrice" example:"true"`
}

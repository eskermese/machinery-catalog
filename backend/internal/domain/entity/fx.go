package entity

import (
	"regexp"
	"time"

	"github.com/shopspring/decimal"
)

// FXRateScale знаков после запятой у курса.
//
// Совпадает со scale колонок fx_rates.rate и quotes.fx_rate: приведение курса
// к строке обязано давать ровно то, что примет и вернёт БД, иначе значение,
// записанное сервисом, и значение, прочитанное обратно, разошлись бы в хвосте.
const FXRateScale = 6

// FXSourceManual источник курса, введённого администратором вручную.
//
// Отличается от кода внешнего источника намеренно: таблица уникальна по
// (source, пара, дата), и ручной курс обязан сосуществовать с загруженным
// за ту же дату, а не конфликтовать с ним. Заодно по строке видно, кто
// поставил курс — воркер или человек.
// ВАЛЮТЫ СЕРВИСА. Объявлены здесь и только здесь.
//
// Раньше пара расчёта была написана трижды — в предложении, в стоимости
// владения и в витрине, — и перевод прайса на юань означал бы найти все три.
// Разъехавшись, они не сломали бы сборку: сервис просто спрашивал бы курс
// пары, которой никто не заводил, и выпуск предложения отвечал бы «курс
// не найден» при заведённом курсе.
const (
	// CurrencyCNY юань — валюта прайса и единственная хранимая цена.
	CurrencyCNY = "CNY"
	// CurrencyKZT тенге — валюта, в которой клиент платит. В ней считаются
	// лизинг, стоимость владения и коммерческое предложение.
	CurrencyKZT = "KZT"
	// CurrencyUSD доллар — вторая, НЕОБЯЗАТЕЛЬНАЯ валюта отображения цены.
	//
	// Только показ: подменить его другой валютой означало бы поменять смысл
	// включателя, а не значение.
	CurrencyUSD = "USD"
)

const FXSourceManual = "manual"

// FXSourceDerived курс, выведенный сервисом через валюту-посредник.
//
// В базу такие курсы не пишутся: они получаются делением двух сохранённых
// и хранить их значило бы завести третье значение, которое разойдётся
// с исходными при правке любого из них.
const FXSourceDerived = "derived"

// ПРИОРИТЕТ ИСТОЧНИКОВ КУРСА — правило одно и живёт здесь.
//
//	 самая поздняя дата           →  берём её
//	 за эту дату есть ручной курс →  берём ручной
//	 нет                          →  берём курс внешнего источника
//	 нет ни одного                →  курса нет, и это отказ, а не единица
//
// То есть ручной ввод — ЗАМЕНА, а не единственный источник: воркер продолжает
// грузить курс, админ видит предложенное и либо соглашается, либо правит.
// Применяет правило запрос FXRateRepository.GetLatest; здесь оно записано,
// чтобы не искать его по коду хранилища.

// bpsInWhole базисных пунктов в единице: 10000 bps = 100%.
const bpsInWhole = 10000

// currencyCodePattern формат кода валюты: ровно три заглавные буквы (ISO 4217).
var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// FXRate курс валютной пары от внешнего источника.
//
// Хранит ЧИСТЫЙ курс источника, без наценки сервиса: наценка живёт отдельной
// сущностью со своей историей, потому что меняется по решению бизнеса,
// а не по расписанию котировок.
type FXRate struct {
	ID            string          // Идентификатор записи
	BaseCurrency  string          // Валюта, в которой ведётся прайс
	QuoteCurrency string          // Валюта, в которую пересчитывается цена
	Source        string          // Источник курса
	Rate          decimal.Decimal // Курс источника без наценки
	RateDate      time.Time       // Дата, на которую действует курс
	FetchedAt     time.Time       // Момент получения курса сервисом
}

// FXRates список курсов.
type FXRates []*FXRate

// NewFXRate создаёт курс источника.
func NewFXRate(
	base, quote, source string,
	rate decimal.Decimal,
	rateDate, fetchedAt time.Time,
) *FXRate {
	return &FXRate{
		BaseCurrency:  base,
		QuoteCurrency: quote,
		Source:        source,
		Rate:          rate,
		RateDate:      rateDate,
		FetchedAt:     fetchedAt,
	}
}

// Validate проверяет пригодность курса к сохранению.
func (r *FXRate) Validate() error {
	if r == nil {
		return ErrNilPointer
	}

	if err := validateCurrencyPair(r.BaseCurrency, r.QuoteCurrency); err != nil {
		return err
	}

	if r.Source == "" {
		return ErrFXSourceInvalid
	}

	// Строго положителен: нулевой курс обнулил бы цену, отрицательный
	// перевернул бы её знак — оба варианта тихо ломают расчёт, а не отказывают.
	if r.Rate.LessThanOrEqual(decimal.Zero) {
		return ErrFXRateInvalid
	}

	return nil
}

// Applied возвращает применённый курс: курс источника с наценкой.
//
// Наценка накручивается мультипликативно — applied = rate * (1 + bps/10000), —
// потому что она задаётся долей от курса, а не абсолютной надбавкой в тенге:
// при росте курса надбавка обязана расти вместе с ним.
func (r *FXRate) Applied(markupBps int) decimal.Decimal {
	if r == nil {
		return decimal.Zero
	}

	if markupBps == 0 {
		return r.Rate
	}

	multiplier := decimal.NewFromInt(1).Add(
		decimal.NewFromInt(int64(markupBps)).Div(decimal.NewFromInt(bpsInWhole)),
	)

	return r.Rate.Mul(multiplier)
}

// FXMarkup наценка сервиса к курсу источника.
//
// Запись истории, а не текущее значение: действующей считается строка
// с наибольшим EffectiveFrom, не превышающим момент расчёта. Прошлые записи
// остаются доказательством того, по какой наценке считалось выданное КП.
type FXMarkup struct {
	ID            string    // Идентификатор записи
	BaseCurrency  string    // Валюта, в которой ведётся прайс
	QuoteCurrency string    // Валюта, в которую пересчитывается цена
	MarkupBps     int       // Наценка в базисных пунктах
	EffectiveFrom time.Time // Момент, с которого наценка действует
	ChangedBy     *string   // Кто изменил; nil — изменение не от человека
	CreatedAt     time.Time // Момент внесения записи
}

// FXMarkups список наценок.
type FXMarkups []*FXMarkup

// NewFXMarkup создаёт запись наценки.
func NewFXMarkup(
	base, quote string,
	markupBps int,
	effectiveFrom time.Time,
	changedBy *string,
	currentTime time.Time,
) *FXMarkup {
	return &FXMarkup{
		BaseCurrency:  base,
		QuoteCurrency: quote,
		MarkupBps:     markupBps,
		EffectiveFrom: effectiveFrom,
		ChangedBy:     changedBy,
		CreatedAt:     currentTime,
	}
}

// Validate проверяет пригодность наценки к сохранению.
func (m *FXMarkup) Validate() error {
	if m == nil {
		return ErrNilPointer
	}

	if err := validateCurrencyPair(m.BaseCurrency, m.QuoteCurrency); err != nil {
		return err
	}

	// Отрицательная наценка означала бы продажу дешевле курса источника.
	// Такое решение возможно, но оно не наценка, и вводить его молча нельзя.
	if m.MarkupBps < 0 {
		return ErrFXMarkupInvalid
	}

	return nil
}

// AppliedFXRate применённый курс вместе с тем, из чего он получен.
//
// Ровно то, что уходит в КП: quotes.fx_rate, quotes.fx_rate_date
// и quotes.fx_markup_bps. Возвращается одним значением намеренно — курс
// без своей даты и наценки нельзя ни проверить, ни разложить обратно.
type AppliedFXRate struct {
	Rate      decimal.Decimal // Применённый курс: курс источника с наценкой
	RateDate  time.Time       // Дата курса источника
	MarkupBps int             // Наценка, зашитая в Rate

	// Source кто поставил курс, из которого он получен: код внешнего
	// источника, ручной ввод или вывод через посредника.
	//
	// Нужен админке, чтобы показать, что именно действует. Носится вместе
	// с курсом, а не вычисляется потом сравнением величин: два источника
	// вполне могут дать одинаковый курс за одну дату, и сравнение
	// назвало бы источником не того.
	Source string
}

// NewAppliedFXRate собирает применённый курс из курса источника и наценки.
func NewAppliedFXRate(rate *FXRate, markupBps int) AppliedFXRate {
	if rate == nil {
		return AppliedFXRate{}
	}

	return AppliedFXRate{
		Rate:      rate.Applied(markupBps),
		RateDate:  rate.RateDate,
		MarkupBps: markupBps,
		Source:    rate.Source,
	}
}

// ConvertPriceCents переводит сумму из минорных единиц базовой валюты пары
// в минорные единицы котируемой по применённому курсу.
//
// Валюты в имени намеренно нет: тот же метод считает и тенге из юаня
// (CNY→KZT), и доллары из юаня (CNY→USD). Пара задаётся курсом, а не
// методом, и приписать валюту к имени значило бы завести второй такой же
// для второй пары.
//
// Единственное место, где живёт это правило округления. И витрина, и выпуск
// коммерческого предложения зовут его, а не считают сами: разойдись они —
// клиент увидел бы на карточке одну сумму, а в предложении по той же машине
// и тому же курсу другую.
//
// Минорные единицы умножаются на курс: 1 фынь по курсу 72.5 — это 72.5 тиына.
// Округление половин от нуля, как во всей денежной арифметике сервиса.
func (a AppliedFXRate) ConvertPriceCents(priceCents int64) int64 {
	return decimal.NewFromInt(priceCents).Mul(a.Rate).Round(0).IntPart()
}

// String возвращает курс в виде, пригодном для numeric(18,6).
//
// Строка, а не decimal и не float: numeric принимает её без потерь и без
// предположений о драйвере, тогда как float потерял бы младшие разряды,
// а decimal протащил бы в подписи адаптеров зависимость домена от библиотеки.
// Хвост дополняется до FXRateScale знаков, чтобы записанное и прочитанное
// значение совпадали посимвольно.
func (a AppliedFXRate) String() string {
	return a.Rate.StringFixed(FXRateScale)
}

// IsZero сообщает, что применённый курс не заполнен.
func (a AppliedFXRate) IsZero() bool {
	return a.Rate.IsZero() && a.RateDate.IsZero()
}

// validateCurrencyPair проверяет коды валют и их различие.
func validateCurrencyPair(base, quote string) error {
	if !currencyCodePattern.MatchString(base) || !currencyCodePattern.MatchString(quote) {
		return ErrCurrencyCodeInvalid
	}

	// Пара из одинаковых валют бессмысленна: курс такой пары всегда единица,
	// и её появление означает ошибку конфигурации, а не редкий случай.
	if base == quote {
		return ErrCurrencyPairInvalid
	}

	return nil
}

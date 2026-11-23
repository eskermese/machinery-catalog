// Package nationalbank получает официальный курс валют Нацбанка Казахстана.
//
// Исходящий адаптер к порту service.RateProvider: сервису важен только факт
// получения курса, а откуда он взялся — не его дело. Поэтому замена источника
// (биржа, другой банк, ручной ввод) не затрагивает ни сценарий синхронизации,
// ни хранилище.
package nationalbank

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"resty.dev/v3"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	httpclient "github.com/eskermese/machinery-catalog/backend/pkg/clients/http"
)

// Source название источника, попадающее в запись курса.
//
// Оно уезжает в колонку fx_rates.source и отличает настоящий курс
// от демонстрационного, залитого в стенд: там источник stand-demo.
const Source = "nationalbank.kz"

// quoteCurrency единственная валюта котирования, которую даёт этот источник.
//
// Нацбанк публикует стоимость иностранных валют В ТЕНГЕ и ничего больше.
// Пара «евро к доллару» ему неизвестна, и просить её бессмысленно —
// поэтому запрос с другой валютой котирования отвергается сразу, а не
// возвращает пустоту, которую вызывающий примет за отсутствие курса.
const quoteCurrency = "KZT"

// ratesPath путь выдачи курсов на дату.
//
// Именно эта выдача, а не rates_all.xml, и это существенно. Порт требует курс
// НА ПЕРЕДАННЫЙ МОМЕНТ, а rates_all.xml всегда отдаёт «текущий» и помечает его
// датой следующих суток: Нацбанк публикует завтрашний курс вечером. Витрина
// подписывает цену словами «по курсу на такое-то число», и завтрашняя дата
// на сегодняшней странице выглядела бы опечаткой.
const ratesPath = "/rss/get_rates.cfm"

// dateFormat формат даты, который понимает выдача Нацбанка.
const dateFormat = "02.01.2006"

// ratesResponse разобранный ответ источника.
type ratesResponse struct {
	XMLName xml.Name    `xml:"rates"`
	Date    string      `xml:"date"`
	Items   []ratesItem `xml:"item"`
}

// ratesItem одна валюта в выдаче.
type ratesItem struct {
	Title string `xml:"title"`
	// Description стоимость Quant единиц валюты в тенге.
	Description string `xml:"description"`
	// Quant сколько единиц валюты стоят Description тенге.
	//
	// У доллара это единица, но у валют с мелким номиналом — сотня или тысяча,
	// и деление на Quant здесь не перестраховка: без него курс иены оказался бы
	// завышен в сто раз.
	Quant int64 `xml:"quant"`
}

// Provider источник курса Нацбанка Казахстана.
type Provider struct {
	client *resty.Client // HTTP клиент
}

// New создаёт источник курса.
func New(addr string, timeout time.Duration) *Provider {
	client := resty.New()
	client.SetTimeout(timeout)
	client.SetBaseURL(addr)

	return &Provider{client: client}
}

// FetchRate возвращает курс пары на дату, ближайшую к переданному моменту.
//
// Дату определяет источник: у него свои сутки, и выходные он «протягивает» —
// в субботу и воскресенье отдаёт курс пятницы. Поэтому отдельной логики
// «отступить на рабочий день» здесь нет и быть не должно: она дублировала бы
// решение источника и однажды разошлась бы с ним на празднике.
//
// Пустая выдача — это отсутствие курса на дату, а не поломка разбора: на дату
// за горизонтом публикации источник отвечает двумя сотнями и списком без
// единой валюты. Такой ответ превращается в ErrFXRateNotFound, и воркер
// синхронизации оставит в базе последний известный курс.
func (p *Provider) FetchRate(
	ctx context.Context,
	base, quote string,
	at time.Time,
) (*entity.FXRate, error) {
	if quote != quoteCurrency {
		return nil, entity.ErrFXPairNotSupported
	}

	resp, err := p.client.R().
		SetContext(ctx).
		SetQueryParam("fdate", at.Format(dateFormat)).
		Get(ratesPath)
	if err != nil {
		return nil, fmt.Errorf("запрос курса: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		raw := &http.Response{
			StatusCode: resp.StatusCode(),
			Body:       io.NopCloser(bytes.NewReader(resp.Bytes())),
		}

		external, convErr := httpclient.NewExternalServiceErrorFromHTTPResponse(raw)
		if convErr != nil {
			return nil, fmt.Errorf("разбор отказа источника: %w", convErr)
		}

		return nil, external
	}

	return parseRate(resp.Bytes(), base, at)
}

// parseRate достаёт курс нужной валюты из ответа источника.
//
// Вынесено из FetchRate, чтобы разбор проверялся тестами на записанных
// ответах, без обращения к сети.
func parseRate(body []byte, base string, at time.Time) (*entity.FXRate, error) {
	var parsed ratesResponse
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("разбор ответа источника: %w", err)
	}

	for _, item := range parsed.Items {
		if item.Title != base {
			continue
		}

		rate, err := unitRate(item)
		if err != nil {
			return nil, err
		}

		// Дата берётся из ответа, а не из запроса: источник сам решает,
		// за какие сутки отдал курс, и подставлять свою дату значило бы
		// подписать чужое число собственным предположением.
		rateDate, err := time.Parse(dateFormat, parsed.Date)
		if err != nil {
			return nil, fmt.Errorf("разбор даты курса %q: %w", parsed.Date, err)
		}

		return entity.NewFXRate(base, quoteCurrency, Source, rate, rateDate, at), nil
	}

	return nil, entity.ErrFXRateNotFound
}

// unitRate приводит стоимость Quant единиц к стоимости одной.
func unitRate(item ratesItem) (decimal.Decimal, error) {
	rate, err := decimal.NewFromString(item.Description)
	if err != nil {
		return decimal.Zero, fmt.Errorf("разбор курса %q: %w", item.Description, err)
	}

	// Ноль и отрицательное значение источник присылать не должен, но проверка
	// стоит до деления: нулевой Quant уронил бы приложение паникой.
	if !rate.IsPositive() || item.Quant <= 0 {
		return decimal.Zero, entity.ErrFXRateInvalid
	}

	return rate.Div(decimal.NewFromInt(item.Quant)), nil
}

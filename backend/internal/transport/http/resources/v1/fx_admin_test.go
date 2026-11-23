package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	v1 "github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources/v1"
)

// Пара, на которой проверяется админка курсов.
const (
	fxBase  = "CNY"
	fxQuote = "KZT"
	fxUSD   = "USD"
)

var fxTestDate = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// stubFXAdmin — заглушка сервиса курсов для админки.
type stubFXAdmin struct {
	service.FXService

	status     map[string]*entity.FXPairStatus
	rates      entity.FXRates
	markups    entity.FXMarkups
	savedRate  *entity.FXRate
	savedMkp   *entity.FXMarkup
	err        error
	historyLim int
}

func (s *stubFXAdmin) PairStatus(
	_ context.Context, base, quote string, _ time.Time,
) (*entity.FXPairStatus, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.status[base+quote], nil
}

func (s *stubFXAdmin) RateHistory(
	_ context.Context, _, _ string, limit int,
) (entity.FXRates, error) {
	s.historyLim = limit

	return s.rates, s.err
}

func (s *stubFXAdmin) MarkupHistory(
	_ context.Context, _, _ string, limit int,
) (entity.FXMarkups, error) {
	s.historyLim = limit

	return s.markups, s.err
}

func (s *stubFXAdmin) SaveRate(_ context.Context, rate *entity.FXRate) (*entity.FXRate, error) {
	s.savedRate = rate

	return rate, s.err
}

func (s *stubFXAdmin) SetMarkup(
	_ context.Context, markup *entity.FXMarkup,
) (*entity.FXMarkup, error) {
	s.savedMkp = markup

	return markup, s.err
}

// stubFXSettings — заглушка настроек показа цены.
type stubFXSettings struct {
	service.SettingsService

	showUSD bool
	err     error
}

func (s *stubFXSettings) ShowUSDPrice(context.Context) bool { return s.showUSD }

func (s *stubFXSettings) SetShowUSDPrice(_ context.Context, enabled bool) error {
	if s.err != nil {
		return s.err
	}

	s.showUSD = enabled

	return nil
}

// fxAdminCall выполняет запрос к админке курсов от имени актора.
func fxAdminCall(
	t *testing.T,
	fx service.FXService,
	settings service.SettingsService,
	actor entity.Actor,
	method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	r := httptest.NewRequest(method, path, reader)
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(v1.WithActorForTest(r.Context(), actor))

	w := httptest.NewRecorder()
	v1.NewFXAdminHandler(fx, settings, fxBase, fxQuote, fxUSD).Routes().ServeHTTP(w, r)

	return w
}

// adminActor администратор — ему открыто всё.
func adminActor() entity.Actor {
	return entity.NewActor("11111111-1111-1111-1111-111111111111", entity.ActorTypeAdmin)
}

// managerActor менеджер — ему открыт только ввод курса.
func managerActor() entity.Actor {
	return entity.NewActor("22222222-2222-2222-2222-222222222222", entity.ActorTypeManager)
}

// TestFXAdmin_Permissions фиксирует разделение прав, ради которого оно
// и сделано: ввод курса — ежедневная работа менеджера, наценка и включатель
// доллара меняют цену всего каталога разом и закрыты администратором.
func TestFXAdmin_Permissions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		managerGot int
	}{
		{name: "состояние", method: http.MethodGet, path: "/status", managerGot: http.StatusOK},
		{
			name: "история курсов", method: http.MethodGet,
			path: "/rates?base=CNY&quote=KZT", managerGot: http.StatusOK,
		},
		{
			name: "ввод курса", method: http.MethodPost, path: "/rates",
			body:       `{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"72.55"}`,
			managerGot: http.StatusCreated,
		},
		{
			name: "история наценок", method: http.MethodGet,
			path: "/markups?base=CNY&quote=KZT", managerGot: http.StatusForbidden,
		},
		{
			name: "правка наценки", method: http.MethodPost, path: "/markups",
			body:       `{"baseCurrency":"CNY","quoteCurrency":"KZT","markupBps":200}`,
			managerGot: http.StatusForbidden,
		},
		{
			name: "чтение настроек", method: http.MethodGet, path: "/settings",
			managerGot: http.StatusForbidden,
		},
		{
			name: "правка настроек", method: http.MethodPut, path: "/settings",
			body: `{"showUsdPrice":true}`, managerGot: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			manager := fxAdminCall(
				t, &stubFXAdmin{status: map[string]*entity.FXPairStatus{}},
				&stubFXSettings{}, managerActor(), c.method, c.path, c.body,
			)
			assert.Equal(t, c.managerGot, manager.Code, "менеджер")

			admin := fxAdminCall(
				t, &stubFXAdmin{status: map[string]*entity.FXPairStatus{}},
				&stubFXSettings{}, adminActor(), c.method, c.path, c.body,
			)
			assert.NotEqual(t, http.StatusForbidden, admin.Code, "администратору открыто всё")
		})
	}
}

// TestFXAdmin_Status фиксирует, что ручка состояния показывает ОБЕ пары
// витрины и включатель доллара рядом.
//
// Без пары CNY→USD в ответе непонятно, почему включённый доллар не появляется
// на витрине, а без включателя — важно ли вообще отсутствие её курса.
func TestFXAdmin_Status(t *testing.T) {
	t.Parallel()

	applied := entity.AppliedFXRate{
		Rate:      decimal.RequireFromString("74.001"),
		RateDate:  fxTestDate,
		MarkupBps: 200,
		Source:    entity.FXSourceManual,
	}

	fx := &stubFXAdmin{status: map[string]*entity.FXPairStatus{
		fxBase + fxQuote: {
			BaseCurrency:    fxBase,
			QuoteCurrency:   fxQuote,
			Effective:       &applied,
			EffectiveSource: entity.FXSourceManual,
			MarkupBps:       200,
			Manual: entity.NewFXRate(
				fxBase, fxQuote, entity.FXSourceManual,
				decimal.RequireFromString("72.55"), fxTestDate, fxTestDate,
			),
			Provider: entity.NewFXRate(
				fxBase, fxQuote, "nationalbank",
				decimal.RequireFromString("70"), fxTestDate, fxTestDate,
			),
			Stale: true,
		},
		// Пары CNY→USD в карте нет: сервис отдаёт nil, и ручка обязана
		// это пережить — это ровно то состояние, в котором стенд живёт,
		// пока курс доллара не заведён.
	}}

	w := fxAdminCall(t, fx, &stubFXSettings{showUSD: true}, adminActor(), http.MethodGet, "/status", "")
	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Items []struct {
			BaseCurrency      string `json:"baseCurrency"`
			QuoteCurrency     string `json:"quoteCurrency"`
			HasRate           bool   `json:"hasRate"`
			Stale             bool   `json:"stale"`
			EffectiveRate     string `json:"effectiveRate"`
			EffectiveRateDate string `json:"effectiveRateDate"`
			EffectiveSource   string `json:"effectiveSource"`
			MarkupBps         int    `json:"markupBps"`
			ProviderRate      *struct {
				Rate string `json:"rate"`
			} `json:"providerRate"`
			ManualRate *struct {
				Rate string `json:"rate"`
			} `json:"manualRate"`
		} `json:"items"`
		ShowUsdPrice bool `json:"showUsdPrice"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got.Items, 2, "обе пары витрины")

	kzt := got.Items[0]
	assert.Equal(t, fxBase, kzt.BaseCurrency)
	assert.Equal(t, fxQuote, kzt.QuoteCurrency)
	assert.True(t, kzt.HasRate)
	assert.True(t, kzt.Stale)
	assert.Equal(t, "74.001000", kzt.EffectiveRate)
	assert.Equal(t, "2026-09-01", kzt.EffectiveRateDate)
	assert.Equal(t, entity.FXSourceManual, kzt.EffectiveSource)
	assert.Equal(t, 200, kzt.MarkupBps)
	require.NotNil(t, kzt.ManualRate)
	assert.Equal(t, "72.550000", kzt.ManualRate.Rate)
	require.NotNil(t, kzt.ProviderRate,
		"курс источника виден рядом с ручным — иначе действующий не с чем сравнить")
	assert.Equal(t, "70.000000", kzt.ProviderRate.Rate)

	usd := got.Items[1]
	assert.Equal(t, fxUSD, usd.QuoteCurrency)
	assert.False(t, usd.HasRate, "курса пары нет — так и показываем")

	assert.True(t, got.ShowUsdPrice)
}

// TestFXAdmin_SaveRate фиксирует, что источник ручного курса проставляет
// сервер и не принимает снаружи.
//
// Иначе клиент назвал бы своему курсу имя воркера и подменил бы то,
// что загрузил источник.
func TestFXAdmin_SaveRate(t *testing.T) {
	t.Parallel()

	t.Run("источник не принимается от клиента", func(t *testing.T) {
		t.Parallel()

		fx := &stubFXAdmin{}
		w := fxAdminCall(t, fx, &stubFXSettings{}, managerActor(), http.MethodPost, "/rates",
			`{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"72.55",
			  "rateDate":"2026-09-01","source":"nationalbank"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, fx.savedRate)
		assert.Equal(t, entity.FXSourceManual, fx.savedRate.Source)
		assert.Equal(t, "72.55", fx.savedRate.Rate.String())
		assert.Equal(t, fxTestDate, fx.savedRate.RateDate)
	})

	t.Run("без даты берётся сегодняшняя", func(t *testing.T) {
		t.Parallel()

		fx := &stubFXAdmin{}
		w := fxAdminCall(t, fx, &stubFXSettings{}, managerActor(), http.MethodPost, "/rates",
			`{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"72.55"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, fx.savedRate)
		assert.WithinDuration(t, time.Now().UTC(), fx.savedRate.RateDate, time.Minute)
	})

	t.Run("отказы", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			body string
			fx   *stubFXAdmin
			want int
		}{
			{name: "не json", body: `{`, fx: &stubFXAdmin{}, want: http.StatusBadRequest},
			{
				name: "курс не число",
				body: `{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"много"}`,
				fx:   &stubFXAdmin{}, want: http.StatusBadRequest,
			},
			{
				name: "дата не разбирается",
				body: `{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"72.55","rateDate":"вчера"}`,
				fx:   &stubFXAdmin{}, want: http.StatusBadRequest,
			},
			{
				name: "хранилище недоступно",
				body: `{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"72.55"}`,
				fx:   &stubFXAdmin{err: entity.ErrDatabaseCommon}, want: http.StatusInternalServerError,
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				w := fxAdminCall(t, c.fx, &stubFXSettings{}, managerActor(),
					http.MethodPost, "/rates", c.body)
				assert.Equal(t, c.want, w.Code)
			})
		}
	})
}

// TestFXAdmin_SaveMarkup фиксирует, что автор правки берётся из токена,
// а не из тела запроса: «кто поменял наценку» — не то, что сообщает клиент.
func TestFXAdmin_SaveMarkup(t *testing.T) {
	t.Parallel()

	t.Run("автор берётся из токена", func(t *testing.T) {
		t.Parallel()

		fx := &stubFXAdmin{}
		actor := adminActor()

		w := fxAdminCall(t, fx, &stubFXSettings{}, actor, http.MethodPost, "/markups",
			`{"baseCurrency":"CNY","quoteCurrency":"KZT","markupBps":200}`)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, fx.savedMkp)
		require.NotNil(t, fx.savedMkp.ChangedBy)
		assert.Equal(t, *actor.ID, *fx.savedMkp.ChangedBy)
		assert.Equal(t, 200, fx.savedMkp.MarkupBps)
	})

	t.Run("отказы", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			body string
			fx   *stubFXAdmin
			want int
		}{
			{name: "не json", body: `{`, fx: &stubFXAdmin{}, want: http.StatusBadRequest},
			{
				name: "хранилище недоступно",
				body: `{"baseCurrency":"CNY","quoteCurrency":"KZT","markupBps":200}`,
				fx:   &stubFXAdmin{err: entity.ErrDatabaseCommon}, want: http.StatusInternalServerError,
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				w := fxAdminCall(t, c.fx, &stubFXSettings{}, adminActor(),
					http.MethodPost, "/markups", c.body)
				assert.Equal(t, c.want, w.Code)
			})
		}
	})
}

// TestFXAdmin_History фиксирует, что пара обязательна и что история
// ограничена сверху.
func TestFXAdmin_History(t *testing.T) {
	t.Parallel()

	rates := entity.FXRates{entity.NewFXRate(
		fxBase, fxQuote, entity.FXSourceManual,
		decimal.RequireFromString("72.55"), fxTestDate, fxTestDate,
	)}
	markups := entity.FXMarkups{entity.NewFXMarkup(fxBase, fxQuote, 200, fxTestDate, nil, fxTestDate)}

	t.Run("курсы", func(t *testing.T) {
		t.Parallel()

		fx := &stubFXAdmin{rates: rates}
		w := fxAdminCall(t, fx, &stubFXSettings{}, adminActor(),
			http.MethodGet, "/rates?base=CNY&quote=KZT", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.Positive(t, fx.historyLim, "история обязана быть ограничена сверху")

		var got struct {
			Items []struct {
				Rate   string `json:"rate"`
				Source string `json:"source"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Len(t, got.Items, 1)
		assert.Equal(t, "72.550000", got.Items[0].Rate)
		assert.Equal(t, entity.FXSourceManual, got.Items[0].Source)
	})

	t.Run("наценки", func(t *testing.T) {
		t.Parallel()

		fx := &stubFXAdmin{markups: markups}
		w := fxAdminCall(t, fx, &stubFXSettings{}, adminActor(),
			http.MethodGet, "/markups?base=CNY&quote=KZT", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got struct {
			Items []struct {
				MarkupBps int    `json:"markupBps"`
				ChangedBy string `json:"changedBy"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Len(t, got.Items, 1)
		assert.Equal(t, 200, got.Items[0].MarkupBps)
		assert.Empty(t, got.Items[0].ChangedBy, "автора нет — пусто, а не выдуманный идентификатор")
	})

	t.Run("пара обязательна", func(t *testing.T) {
		t.Parallel()

		for _, path := range []string{"/rates", "/markups", "/rates?base=CNY", "/markups?quote=KZT"} {
			w := fxAdminCall(t, &stubFXAdmin{}, &stubFXSettings{}, adminActor(),
				http.MethodGet, path, "")
			assert.Equal(t, http.StatusBadRequest, w.Code, path)
		}
	})

	t.Run("отказы хранилища", func(t *testing.T) {
		t.Parallel()

		for _, path := range []string{"/rates?base=CNY&quote=KZT", "/markups?base=CNY&quote=KZT"} {
			w := fxAdminCall(t, &stubFXAdmin{err: entity.ErrDatabaseCommon}, &stubFXSettings{},
				adminActor(), http.MethodGet, path, "")
			assert.Equal(t, http.StatusInternalServerError, w.Code, path)
		}
	})
}

// TestFXAdmin_Status_Error фиксирует, что отказ хранилища не выдаётся
// за «курса нет».
func TestFXAdmin_Status_Error(t *testing.T) {
	t.Parallel()

	w := fxAdminCall(t, &stubFXAdmin{err: entity.ErrDatabaseCommon}, &stubFXSettings{},
		adminActor(), http.MethodGet, "/status", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFXAdmin_Settings фиксирует чтение и правку включателя доллара.
func TestFXAdmin_Settings(t *testing.T) {
	t.Parallel()

	t.Run("читается", func(t *testing.T) {
		t.Parallel()

		w := fxAdminCall(t, &stubFXAdmin{}, &stubFXSettings{showUSD: true}, adminActor(),
			http.MethodGet, "/settings", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"showUsdPrice":true}`, w.Body.String())
	})

	t.Run("включается", func(t *testing.T) {
		t.Parallel()

		settings := &stubFXSettings{}
		w := fxAdminCall(t, &stubFXAdmin{}, settings, adminActor(),
			http.MethodPut, "/settings", `{"showUsdPrice":true}`)

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, settings.showUSD)
		assert.JSONEq(t, `{"showUsdPrice":true}`, w.Body.String())
	})

	t.Run("выключается", func(t *testing.T) {
		t.Parallel()

		settings := &stubFXSettings{showUSD: true}
		w := fxAdminCall(t, &stubFXAdmin{}, settings, adminActor(),
			http.MethodPut, "/settings", `{"showUsdPrice":false}`)

		require.Equal(t, http.StatusOK, w.Code)
		assert.False(t, settings.showUSD)
	})

	t.Run("отказы", func(t *testing.T) {
		t.Parallel()

		bad := fxAdminCall(t, &stubFXAdmin{}, &stubFXSettings{}, adminActor(),
			http.MethodPut, "/settings", `{`)
		assert.Equal(t, http.StatusBadRequest, bad.Code)

		broken := fxAdminCall(t, &stubFXAdmin{}, &stubFXSettings{err: entity.ErrDatabaseCommon},
			adminActor(), http.MethodPut, "/settings", `{"showUsdPrice":true}`)
		assert.Equal(t, http.StatusInternalServerError, broken.Code)
	})
}

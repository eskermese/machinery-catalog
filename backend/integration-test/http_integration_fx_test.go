//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	test_init "github.com/eskermese/machinery-catalog/backend/integration-test/test-init"
)

// Курс и наценка сценариев админки.
//
// Курс отличается от посеянного (72.5), чтобы было видно: цена на витрине
// пересчиталась по ВВЕДЁННОМУ курсу, а не осталась на курсе источника.
const (
	adminFXRate      = "80.000000"
	adminFXMarkupBps = 200
)

// withAdminFX заводит ручной курс и наценку на пару прайса, выполняет
// сценарий и возвращает набор в исходное состояние.
//
// Возврат обязателен: курс и наценка глобальны, а соседние сценарии считают
// цену по посеянному курсу без наценки. Через defer — падение проверки
// внутри не должно портить остальной прогон.
func (s *HTTPTestSuite) withAdminFX(admin authSession, action func()) {
	s.T().Helper()

	s.saveFXRate(admin, "CNY", "KZT", adminFXRate)
	s.saveFXMarkup(admin, "CNY", "KZT", adminFXMarkupBps)
	s.awaitFXCacheDropped()

	defer func() {
		require.NoError(s.T(), test_init.Exec(s.ctx, `DELETE FROM fx_markups`),
			"снятие наценки после сценария")
		require.NoError(s.T(), test_init.Exec(s.ctx,
			`DELETE FROM fx_rates WHERE source = 'manual'`),
			"снятие ручного курса после сценария")
		s.dropFXCache()
	}()

	action()
}

// saveFXRate вводит курс через админку.
func (s *HTTPTestSuite) saveFXRate(admin authSession, base, quote, rate string) {
	s.T().Helper()

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/fx/rates", admin.Token,
		fmt.Sprintf(`{"baseCurrency":%q,"quoteCurrency":%q,"rate":%q}`, base, quote, rate))
	require.Equal(s.T(), http.StatusCreated, status, "ввод курса: %s", payload)
}

// saveFXMarkup задаёт наценку через админку.
func (s *HTTPTestSuite) saveFXMarkup(admin authSession, base, quote string, bps int) {
	s.T().Helper()

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/fx/markups", admin.Token,
		fmt.Sprintf(`{"baseCurrency":%q,"quoteCurrency":%q,"markupBps":%d}`, base, quote, bps))
	require.Equal(s.T(), http.StatusCreated, status, "правка наценки: %s", payload)
}

// setShowUSD переключает показ цены в долларах.
func (s *HTTPTestSuite) setShowUSD(admin authSession, enabled bool) {
	s.T().Helper()

	status, payload := requestJSON(http.MethodPut, s.httpBasePath+"/v1/admin/fx/settings", admin.Token,
		fmt.Sprintf(`{"showUsdPrice":%t}`, enabled))
	require.Equal(s.T(), http.StatusOK, status, "переключение доллара: %s", payload)
}

// publishedMachine заводит опубликованную карточку с заданной ценой.
func (s *HTTPTestSuite) publishedMachine(admin authSession, prefix string, priceCents int64) string {
	s.T().Helper()

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, prefix+"-cat-"+suffix, "Раздел "+prefix)

	slug := prefix + "-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: priceCents,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	return slug
}

// TestHTTPFXManualRateWithMarkup — главный сценарий заказчика: курс приходит
// вручную, сверху наценка, и цена на витрине считается по ним.
//
// Проверяется и приоритет источников: курс Нацбанка за ту же дату посеян
// подготовкой набора, и введённый вручную обязан его перебить. Иначе «ручной
// курс главнее» осталось бы утверждением в комментарии.
func (s *HTTPTestSuite) TestHTTPFXManualRateWithMarkup() {
	admin := adminSession(s.T(), s.httpBasePath)
	slug := s.publishedMachine(admin, "manualfx", 1_000_000)

	s.withAdminFX(admin, func() {
		status, card := publicMachineBySlug(s.httpBasePath, slug)
		require.Equal(s.T(), http.StatusOK, status)

		assert.Equal(s.T(), float64(1_000_000), card["priceCnyCents"], "юань — хранимая цена")

		// 1 000 000 фыней по 80,00 с наценкой 2 % = 80 * 1,02 = 81,6.
		// Курс источника (72,5) в расчёт не идёт: ручной ввод главнее.
		require.Contains(s.T(), card, "priceKztCents")
		assert.Equal(s.T(), float64(81_600_000), card["priceKztCents"],
			"цена считается по ВВЕДЁННОМУ курсу с наценкой")

		// Состояние пары объясняет, откуда взялась цифра.
		status, payload := requestJSON(http.MethodGet,
			s.httpBasePath+"/v1/admin/fx/status", admin.Token, "")
		require.Equal(s.T(), http.StatusOK, status, "состояние курсов: %s", payload)

		state := decodeJSONMap(s.T(), payload)
		items, _ := state["items"].([]any)
		require.NotEmpty(s.T(), items)

		kzt, _ := items[0].(map[string]any)
		require.NotNil(s.T(), kzt)
		assert.Equal(s.T(), "KZT", kzt["quoteCurrency"])
		assert.Equal(s.T(), true, kzt["hasRate"])
		assert.Equal(s.T(), "manual", kzt["effectiveSource"], "действует ручной курс")
		assert.Equal(s.T(), "81.600000", kzt["effectiveRate"])
		assert.Equal(s.T(), float64(adminFXMarkupBps), kzt["markupBps"])

		require.NotNil(s.T(), kzt["manualRate"], "введённый курс виден")
		require.NotNil(s.T(), kzt["providerRate"],
			"курс источника виден рядом — иначе действующий не с чем сравнить")
	})
}

// TestHTTPFXUSDPriceToggle фиксирует доллар как ВТОРОЕ, необязательное
// отображение: он появляется по настройке и считается от юаня по своей паре.
func (s *HTTPTestSuite) TestHTTPFXUSDPriceToggle() {
	admin := adminSession(s.T(), s.httpBasePath)
	slug := s.publishedMachine(admin, "usdprice", 1_000_000)

	defer func() {
		s.setShowUSD(admin, false)
		require.NoError(s.T(), test_init.Exec(s.ctx,
			`DELETE FROM fx_rates WHERE source = 'manual'`), "снятие курса доллара")
		s.dropFXCache()
	}()

	// Курс доллара мог остаться в кэше от соседнего сценария: пары
	// кэшируются порознь.
	s.dropFXCache()

	// Выключено: доллара нет, даже когда курс пары заведён.
	s.saveFXRate(admin, "CNY", "USD", "0.140000")
	s.setShowUSD(admin, false)

	_, card := publicMachineBySlug(s.httpBasePath, slug)
	assert.NotContains(s.T(), card, "priceUsdCents", "выключенный доллар не показывается")

	// Включено: доллар считается ОТ ЮАНЯ по паре CNY→USD.
	s.setShowUSD(admin, true)

	_, card = publicMachineBySlug(s.httpBasePath, slug)
	require.Contains(s.T(), card, "priceUsdCents", "включённый доллар обязан прийти")

	// 1 000 000 фыней * 0,14 = 140 000 центов. Наценки на паре нет.
	assert.Equal(s.T(), float64(140_000), card["priceUsdCents"])
	assert.Contains(s.T(), card, "usdRateDate", "дата курса приходит вместе с ценой")

	// Тенге при этом считается по СВОЕЙ паре и не зависит от доллара.
	assert.Equal(s.T(), float64(72_500_000), card["priceKztCents"])
}

// TestHTTPFXUSDWithoutRate фиксирует, что включённый доллар без курса пары
// не ломает витрину и не подменяется ничем.
func (s *HTTPTestSuite) TestHTTPFXUSDWithoutRate() {
	admin := adminSession(s.T(), s.httpBasePath)
	slug := s.publishedMachine(admin, "usdnorate", 1_000_000)

	defer s.setShowUSD(admin, false)

	s.dropFXCache()
	s.setShowUSD(admin, true)

	status, card := publicMachineBySlug(s.httpBasePath, slug)
	require.Equal(s.T(), http.StatusOK, status, "карточка обязана открыться")

	assert.Contains(s.T(), card, "priceCnyCents")
	assert.NotContains(s.T(), card, "priceUsdCents",
		"курса пары нет — доллара нет, а не ноль и не цена в тенге")
	assert.NotContains(s.T(), card, "usdRateDate")
}

// TestHTTPQuoteFixesRateAndMarkup фиксирует главное свойство выпущенного
// предложения: его цифры не меняются вслед за курсом.
//
// Курс и наценка запоминаются в момент выпуска. Изменение курса после этого
// обязано пройти мимо: клиент получил конкретную сумму, и пересчитать её
// задним числом — значит выдать другое предложение под тем же токеном.
func (s *HTTPTestSuite) TestHTTPQuoteFixesRateAndMarkup() {
	admin := adminSession(s.T(), s.httpBasePath)
	slug := s.publishedMachine(admin, "fixfx", 10_000_000)

	var (
		publicToken string
		issuedRate  string
		issuedCost  float64
	)

	s.withAdminFX(admin, func() {
		body := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
			"downPaymentBps":2000,"termMonths":36,"rateBps":1800,"validForDays":14}`, slug)

		status, payload := requestJSON(http.MethodPost,
			s.httpBasePath+"/v1/admin/quotes", admin.Token, body)
		require.Equal(s.T(), http.StatusCreated, status, "выпуск предложения: %s", payload)

		issued := decodeJSONMap(s.T(), payload)
		publicToken, _ = issued["publicToken"].(string)
		require.NotEmpty(s.T(), publicToken)

		quote, _ := issued["quote"].(map[string]any)
		require.NotNil(s.T(), quote)

		fx, _ := quote["fx"].(map[string]any)
		require.NotNil(s.T(), fx, "предложение запоминает курс")

		issuedRate, _ = fx["rate"].(string)
		assert.Equal(s.T(), "81.600000", issuedRate, "зафиксирован курс с наценкой")
		assert.Equal(s.T(), float64(adminFXMarkupBps), fx["markupBps"],
			"наценка запоминается отдельно — иначе цену нечем объяснить")

		calculation, _ := quote["calculation"].(map[string]any)
		require.NotNil(s.T(), calculation)
		issuedCost, _ = calculation["assetCostKztCents"].(float64)
		assert.Equal(s.T(), float64(816_000_000), issuedCost,
			"10 000 000 фыней по 81,6 = 816 000 000 тиынов")
	})

	// Курс снят и наценка снята — выпущенное предложение обязано остаться
	// тем же.
	s.awaitFXCacheDropped()

	status, payload := requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/quotes/"+publicToken, "", "")
	require.Equal(s.T(), http.StatusOK, status, "чтение предложения: %s", payload)

	quote := decodeJSONMap(s.T(), payload)
	fx, _ := quote["fx"].(map[string]any)
	require.NotNil(s.T(), fx)

	assert.Equal(s.T(), issuedRate, fx["rate"], "курс предложения не меняется вслед за курсом дня")
	assert.Equal(s.T(), float64(adminFXMarkupBps), fx["markupBps"], "наценка тоже зафиксирована")

	calculation, _ := quote["calculation"].(map[string]any)
	require.NotNil(s.T(), calculation)
	assert.Equal(s.T(), issuedCost, calculation["assetCostKztCents"], "сумма та же")
}

// TestHTTPFXPermissions фиксирует разделение прав по цене ошибки: ввод курса —
// ежедневная работа менеджера, наценка и включатель доллара меняют цену всего
// каталога разом и закрыты администратором.
func (s *HTTPTestSuite) TestHTTPFXPermissions() {
	admin := adminSession(s.T(), s.httpBasePath)
	manager := managerSession(s.T(), s.httpBasePath)

	defer func() {
		require.NoError(s.T(), test_init.Exec(s.ctx,
			`DELETE FROM fx_rates WHERE source = 'manual'`), "снятие курса менеджера")
		s.dropFXCache()
	}()

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{
			name: "состояние курсов", method: http.MethodGet,
			path: "/v1/admin/fx/status", want: http.StatusOK,
		},
		{
			name: "ввод курса", method: http.MethodPost, path: "/v1/admin/fx/rates",
			body: `{"baseCurrency":"CNY","quoteCurrency":"KZT","rate":"73.10"}`,
			want: http.StatusCreated,
		},
		{
			name: "правка наценки", method: http.MethodPost, path: "/v1/admin/fx/markups",
			body: `{"baseCurrency":"CNY","quoteCurrency":"KZT","markupBps":500}`,
			want: http.StatusForbidden,
		},
		{
			name: "включатель доллара", method: http.MethodPut, path: "/v1/admin/fx/settings",
			body: `{"showUsdPrice":true}`, want: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			status, payload := requestJSON(c.method, s.httpBasePath+c.path, manager.Token, c.body)
			assert.Equal(t, c.want, status, "менеджер: %s", payload)

			adminStatus, adminPayload := requestJSON(c.method, s.httpBasePath+c.path, admin.Token, c.body)
			assert.NotEqual(t, http.StatusForbidden, adminStatus,
				"администратору открыто всё: %s", adminPayload)
		})
	}

	// Наценку, заведённую администратором в проверке прав, надо убрать:
	// она глобальна и сдвинула бы цены соседних сценариев.
	require.NoError(s.T(), test_init.Exec(s.ctx, `DELETE FROM fx_markups`), "снятие наценки")
	s.setShowUSD(admin, false)
}

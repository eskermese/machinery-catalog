//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPTCOFullData фиксирует расчёт стоимости владения при заведённых
// величинах карточки: все статьи посчитаны, перечень недостающего пуст.
func (s *HTTPTestSuite) TestHTTPTCOFullData() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "tco-cat-"+suffix, "Раздел владения")

	slug := "tco-machine-" + suffix
	// Цена 100 000 $, расход 20 л/мч, ТО каждые 500 мч по 1000 $.
	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":"XE215C","slug":%q,
		"nameRU":"Э","nameKK":"Э","nameEN":"E","priceCnyCents":10000000,"isPricePublic":true,
		"availability":"in_stock","condition":"new",
		"fuelConsumptionLph":"20","serviceIntervalHours":500,"serviceCostCnyCents":100000,
		"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`, categoryID, slug)

	status, payload := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/admin/catalog/machines", admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "создание карточки: %s", payload)

	machineID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// Величины дошли до карточки и возвращаются админке: без этого их
	// нельзя было бы ни проверить, ни поправить.
	stored := decodeJSONMap(s.T(), payload)
	assert.Equal(s.T(), "20", stored["fuelConsumptionLph"])
	assert.InDelta(s.T(), 500, stored["serviceIntervalHours"], 0)

	// Расчёт без авторизации: это публичная ручка.
	calc := `{"years":3,"hoursPerYear":2000,"fuelPriceKztCents":30000,"operatorHourlyKztCents":200000}`

	status, payload = requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/machines/%s/tco", s.httpBasePath, slug), "", calc)
	require.Equal(s.T(), http.StatusOK, status, "расчёт владения: %s", payload)

	result := decodeJSONMap(s.T(), payload)

	assert.InDelta(s.T(), 6000, result["totalHours"], 0, "3 года по 2000 моточасов")
	assert.Positive(s.T(), result["totalKztCents"])
	assert.Positive(s.T(), result["costPerHourKztCents"], "стоимость моточаса посчитана")

	// Остаточная стоимость не заведена ни для одного раздела: таблица ставок
	// пуста, и это законное состояние — расчёт говорит об этом, а не молчит.
	missing, _ := result["missing"].([]any)
	assert.Equal(s.T(), []any{"residual_value"}, missing,
		"не хватает только остаточной стоимости: её ставки не заведены")
	assert.Equal(s.T(), false, result["complete"])

	// Статьи, для которых данные есть, посчитаны и пришли суммами.
	fuel, _ := result["fuel"].(map[string]any)
	require.NotNil(s.T(), fuel)
	assert.Positive(s.T(), fuel["amountKztCents"], "топливо посчитано")
	assert.Nil(s.T(), fuel["missing"])

	service, _ := result["service"].(map[string]any)
	require.NotNil(s.T(), service)
	assert.Positive(s.T(), service["amountKztCents"])
	assert.InDelta(s.T(), 12, result["servicesCount"], 0, "6000 / 500 обслуживаний")

	// Ссылка на карточку собрана: клиенту есть куда вернуться.
	ref, _ := result["machineRef"].(map[string]any)
	require.NotNil(s.T(), ref)
	assert.Equal(s.T(), slug, ref["slug"])
}

// TestHTTPTCOPartialData фиксирует главное правило расчёта: незаведённая
// величина даёт статью БЕЗ суммы и запись в перечне, а не ноль.
//
// Ноль в строке «топливо» выглядел бы как расчёт, а был бы догадкой, —
// и это хуже отсутствия расчёта вовсе.
func (s *HTTPTestSuite) TestHTTPTCOPartialData() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "tco-part-"+suffix, "Без данных")

	slug := "tco-partial-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	calc := `{"years":3,"hoursPerYear":2000,"fuelPriceKztCents":30000,"operatorHourlyKztCents":200000}`

	status, payload := requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/machines/%s/tco", s.httpBasePath, slug), "", calc)
	require.Equal(s.T(), http.StatusOK, status, "нехватка данных — не отказ: %s", payload)

	result := decodeJSONMap(s.T(), payload)

	assert.Equal(s.T(), false, result["complete"])

	missing, _ := result["missing"].([]any)
	assert.ElementsMatch(s.T(),
		[]any{"fuel_consumption", "service_interval", "service_cost", "residual_value"}, missing)

	// Статья топлива приходит БЕЗ суммы и с указанием, чего не хватило.
	fuel, _ := result["fuel"].(map[string]any)
	require.NotNil(s.T(), fuel)
	assert.Nil(s.T(), fuel["amountKztCents"], "суммы нет вовсе, а не ноль")
	assert.Equal(s.T(), []any{"fuel_consumption"}, fuel["missing"])

	// Покупка и оператор считаются всегда: цена известна, ставка пришла
	// от клиента, — иначе расчёт не показал бы даже того, что может.
	purchase, _ := result["purchase"].(map[string]any)
	require.NotNil(s.T(), purchase)
	assert.Positive(s.T(), purchase["amountKztCents"])

	operator, _ := result["operator"].(map[string]any)
	require.NotNil(s.T(), operator)
	assert.Positive(s.T(), operator["amountKztCents"])
}

// TestHTTPTCOHiddenPrice фиксирует отказ по карточке со скрытой ценой:
// стоимость владения начинается с покупки.
func (s *HTTPTestSuite) TestHTTPTCOHiddenPrice() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "tco-hidden-"+suffix, "Скрытая цена")

	slug := "tco-hidden-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000, HidePrice: true,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	status, payload := requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/machines/%s/tco", s.httpBasePath, slug), "",
		`{"years":3,"hoursPerYear":2000,"fuelPriceKztCents":30000,"operatorHourlyKztCents":200000}`)

	assert.Equal(s.T(), http.StatusBadRequest, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"MACHINE_PRICE_HIDDEN")
}

// TestHTTPQuoteRecalculate проходит пересчёт предложения по публичной ссылке.
//
// Проверяется главное: пересчёт считает по ЗАФИКСИРОВАННОМУ курсу, ничего
// не сохраняет и само предложение не меняет.
func (s *HTTPTestSuite) TestHTTPQuoteRecalculate() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "recalc-cat-"+suffix, "Пересчёт")

	slug := "recalc-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	token := issueQuote(s.T(), s.httpBasePath, admin, slug)

	original := readQuote(s.T(), s.httpBasePath, token)
	originalCalc, _ := original["calculation"].(map[string]any)
	require.NotNil(s.T(), originalCalc)

	originalFX, _ := original["fx"].(map[string]any)
	require.NotNil(s.T(), originalFX)

	// Пересчёт: тот же взнос и ставка, но срок вдвое длиннее.
	status, payload := requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/quotes/%s/recalculate", s.httpBasePath, token), "",
		`{"scheduleType":"annuity","downPaymentBps":2000,"termMonths":72,"rateBps":1800}`)
	require.Equal(s.T(), http.StatusOK, status, "пересчёт по ссылке: %s", payload)

	recalculated := decodeJSONMap(s.T(), payload)
	recalcCalc, _ := recalculated["calculation"].(map[string]any)
	require.NotNil(s.T(), recalcCalc)

	assert.InDelta(s.T(), 72, recalcCalc["termMonths"], 0, "новый срок применился")
	assert.NotEqual(s.T(), originalCalc["monthlyPaymentKztCents"], recalcCalc["monthlyPaymentKztCents"],
		"платёж пересчитан")

	// Курс тот же, что зафиксирован в предложении: пересчёт отвечает
	// на вопрос об условиях сделки, а не о валюте.
	recalcFX, _ := recalculated["fx"].(map[string]any)
	require.NotNil(s.T(), recalcFX)
	assert.Equal(s.T(), originalFX["rate"], recalcFX["rate"], "курс зафиксированный")
	assert.Equal(s.T(), originalFX["rateDate"], recalcFX["rateDate"])
	assert.Equal(s.T(), originalCalc["assetCostKztCents"], recalcCalc["assetCostKztCents"],
		"стоимость машины в тенге не пересчитывается")

	// Само предложение по ссылке не изменилось: обязательство осталось.
	stored := readQuote(s.T(), s.httpBasePath, token)
	storedCalc, _ := stored["calculation"].(map[string]any)
	assert.Equal(s.T(), originalCalc["termMonths"], storedCalc["termMonths"],
		"выпущенное предложение не подменяется пересчётом")
	assert.Equal(s.T(), originalCalc["monthlyPaymentKztCents"], storedCalc["monthlyPaymentKztCents"])
}

// TestHTTPQuoteRecalculateUnknownToken фиксирует, что подобранный токен
// отвергается так же, как при чтении предложения.
func (s *HTTPTestSuite) TestHTTPQuoteRecalculateUnknownToken() {
	unknown := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	status, payload := requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/quotes/%s/recalculate", s.httpBasePath, unknown), "",
		`{"scheduleType":"annuity","downPaymentBps":2000,"termMonths":48,"rateBps":1800}`)

	assert.Equal(s.T(), http.StatusNotFound, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"QUOTE_NOT_FOUND")
}

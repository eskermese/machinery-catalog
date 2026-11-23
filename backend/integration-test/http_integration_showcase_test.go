//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPLeasingCalculateNeedsNoAuth проходит публичный расчёт целиком:
// без токена, с ценой и курсом, которые сервер берёт сам.
func (s *HTTPTestSuite) TestHTTPLeasingCalculateNeedsNoAuth() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "calc-cat-"+suffix, "Категория расчёта")

	slug := "calc-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	body := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800}`, slug)

	// Без токена доступа.
	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leasing/calculate", "", body)
	require.Equal(s.T(), http.StatusOK, status, "расчёт публичный: %s", payload)

	result := decodeJSONMap(s.T(), payload)

	// Цена взята сервером по слагу, а не принята от клиента.
	assert.Equal(s.T(), float64(10_000_000), result["priceCnyCents"])

	calculation, _ := result["calculation"].(map[string]any)
	require.NotNil(s.T(), calculation, "блок расчёта уходит в ответе")

	assert.Positive(s.T(), calculation["assetCostKztCents"], "стоимость пересчитана в тиыны")
	assert.Positive(s.T(), calculation["monthlyPaymentKztCents"])

	schedule, _ := calculation["schedule"].([]any)
	assert.Len(s.T(), schedule, 36, "график построен на весь срок")

	// Курс расчёта отдан вместе с датой, наценка — нет.
	fx, _ := result["fx"].(map[string]any)
	require.NotNil(s.T(), fx)
	assert.NotEmpty(s.T(), fx["rate"])
	assert.NotEmpty(s.T(), fx["rateDate"], "без даты вчерашний курс неотличим от сегодняшнего")
	assert.NotContains(s.T(), fx, "markupBps", "наценка наружу не выносится")

	// Ничего не выпущено.
	assert.NotContains(s.T(), result, "publicToken")
	assert.NotContains(s.T(), result, "id")
}

// TestHTTPLeasingCalculateRejectsHiddenPrice фиксирует, что по машине
// со скрытой ценой расчёт отказывает, а не отдаёт цифру обходным путём.
func (s *HTTPTestSuite) TestHTTPLeasingCalculateRejectsHiddenPrice() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "hidden-cat-"+suffix, "Скрытая цена")

	slug := "hidden-machine-" + suffix
	machineID := createMachineHiddenPrice(s.T(), s.httpBasePath, admin, categoryID, slug)
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// Карточка на витрине есть, но без цены.
	status, card := publicMachineBySlug(s.httpBasePath, slug)
	require.Equal(s.T(), http.StatusOK, status)

	assert.Equal(s.T(), false, card["isPricePublic"])
	assert.NotContains(s.T(), card, "priceCnyCents")
	assert.NotContains(s.T(), card, "priceKztCents", "скрытая цена не утекает пересчитанной")

	body := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800}`, slug)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leasing/calculate", "", body)

	assert.Equal(s.T(), http.StatusBadRequest, status, "расчёт по скрытой цене отвергается")
	assert.Contains(s.T(), payload, entity.CodePrefix+"MACHINE_PRICE_HIDDEN")
}

// TestHTTPLeasingCalculateChecksProgramBounds фиксирует проверку границ
// программы на живом API теми же кодами, что и у выпуска.
func (s *HTTPTestSuite) TestHTTPLeasingCalculateChecksProgramBounds() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "bounds-cat-"+suffix, "Границы")

	slug := "bounds-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	programID := createLeasingProgram(s.T(), s.httpBasePath, admin, "Программа "+suffix)

	cases := []struct {
		name    string
		params  string
		expCode string
	}{
		{
			name:    "взнос ниже минимального",
			params:  `"downPaymentBps":500,"termMonths":36,"rateBps":1800`,
			expCode: "LEASING_DOWN_PAYMENT_TOO_LOW",
		},
		{
			name:    "срок вне диапазона",
			params:  `"downPaymentBps":2000,"termMonths":120,"rateBps":1800`,
			expCode: "LEASING_TERM_OUT_OF_RANGE",
		},
		{
			name:    "ставка вне диапазона",
			params:  `"downPaymentBps":2000,"termMonths":36,"rateBps":100`,
			expCode: "LEASING_RATE_OUT_OF_RANGE",
		},
	}

	for _, c := range cases {
		body := fmt.Sprintf(`{"machineSlug":%q,"leasingProgramID":%q,"scheduleType":"annuity",%s}`,
			slug, programID, c.params)

		status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leasing/calculate", "", body)

		assert.Equal(s.T(), http.StatusBadRequest, status, "%s: %s", c.name, payload)
		assert.Contains(s.T(), payload, entity.CodePrefix+c.expCode, c.name)
	}

	// Внутри границ расчёт проходит.
	body := fmt.Sprintf(`{"machineSlug":%q,"leasingProgramID":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800}`, slug, programID)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leasing/calculate", "", body)
	assert.Equal(s.T(), http.StatusOK, status, "параметры внутри границ: %s", payload)
}

// TestHTTPCatalogPriceInTenge фиксирует пересчёт цены на витрине: курс посеян
// в подготовке набора, поэтому цена в тенге и дата курса обязаны прийти вместе.
func (s *HTTPTestSuite) TestHTTPCatalogPriceInTenge() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "tenge-cat-"+suffix, "Тенге")

	slug := "tenge-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 1_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	status, card := publicMachineBySlug(s.httpBasePath, slug)
	require.Equal(s.T(), http.StatusOK, status)

	assert.Equal(s.T(), float64(1_000_000), card["priceCnyCents"], "юани остаются")

	priceKZT, hasKZT := card["priceKztCents"]
	require.True(s.T(), hasKZT, "курс посеян в подготовке набора: цена в тенге обязана прийти")

	// 1 000 000 фыней по посеянному курсу 72.5 = 72 500 000 тиынов.
	assert.Equal(s.T(), float64(72_500_000), priceKZT)

	_, hasDate := card["fxRateDate"]
	assert.True(s.T(), hasDate, "дата курса приходит вместе с ценой, а не опционально при ней")

	assert.NotContains(s.T(), card, "markupBps", "наценка наружу не выносится")
}

// TestHTTPCatalogWorksWithoutRate фиксирует, что отсутствие курса роняет
// пересчёт, но НЕ каталог.
//
// Курс убирается на время теста и возвращается после: посев делает подготовка
// набора, и оставить базу без него значило бы уронить соседние сценарии.
func (s *HTTPTestSuite) TestHTTPCatalogWorksWithoutRate() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "norate-cat-"+suffix, "Без курса")

	slug := "norate-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 1_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	s.withoutFXRate(func() {
		status, card := publicMachineBySlug(s.httpBasePath, slug)
		require.Equal(s.T(), http.StatusOK, status, "карточка обязана открыться без курса")

		assert.Contains(s.T(), card, "priceCnyCents", "витрина показывает юани")
		assert.NotContains(s.T(), card, "priceKztCents")
		assert.NotContains(s.T(), card, "fxRateDate", "дата без цены бессмысленна")

		// Поиск тоже обязан отработать.
		status, searchPayload := requestJSON(http.MethodPost,
			s.httpBasePath+"/v1/catalog/machines/search", "",
			fmt.Sprintf(`{"categoryID":%q}`, categoryID))

		require.Equal(s.T(), http.StatusOK, status, "поиск обязан отработать без курса: %s", searchPayload)
		assert.Contains(s.T(), searchPayload, slug)
	})
}

// TestHTTPCatalogSubtreeAttributeFilter фиксирует, что фильтр по характеристике
// родителя находит карточки дочерних категорий.
//
// Ровно тот дефект, ради которого характеристики стали наследоваться: раньше
// у карточки потомка был свой идентификатор той же «мощности», и она выпадала
// из отбора по родительскому.
func (s *HTTPTestSuite) TestHTTPCatalogSubtreeAttributeFilter() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()

	parentID := createCategory(s.T(), s.httpBasePath, admin, "inh-parent-"+suffix, "Экскаваторы")
	childID := createChildCategory(s.T(), s.httpBasePath, admin,
		parentID, "inh-child-"+suffix, "Гусеничные")

	// Характеристика объявлена ТОЛЬКО у родителя.
	powerID := createNumberAttribute(s.T(), s.httpBasePath, admin, parentID, "engine_power")

	// Дочерняя категория видит её как свою.
	status, attributes := publicCategoryAttributes(s.httpBasePath, childID)
	require.Equal(s.T(), http.StatusOK, status)
	require.Len(s.T(), attributes, 1, "характеристика родителя наследуется потомком")
	assert.Equal(s.T(), powerID, attributes[0]["id"],
		"идентификатор тот же: иначе фильтр по поддереву терял бы потомков")

	// Карточка заводится в ДОЧЕРНЕЙ категории со значением унаследованной
	// характеристики.
	childSlug := "inh-child-machine-" + suffix
	childMachineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: childID, Slug: childSlug, PriceCents: 9_000_000,
		PowerID: powerID, Power: "200",
	})
	publishMachine(s.T(), s.httpBasePath, admin, childMachineID)

	// Фильтр по характеристике из РОДИТЕЛЬСКОЙ категории с обходом поддерева.
	body := fmt.Sprintf(`{"categoryID":%q,"includeSubcategories":true,
		"attributes":[{"attributeID":%q,"numberMin":"150"}]}`, parentID, powerID)

	status, payload := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", body)
	require.Equal(s.T(), http.StatusOK, status, "поиск по поддереву: %s", payload)

	assert.Contains(s.T(), payload, childSlug,
		"карточка дочерней категории обязана попасть в отбор по характеристике родителя")

	// Условие, которому карточка не удовлетворяет, её не возвращает —
	// иначе первая проверка прошла бы и на неработающем фильтре.
	body = fmt.Sprintf(`{"categoryID":%q,"includeSubcategories":true,
		"attributes":[{"attributeID":%q,"numberMin":"500"}]}`, parentID, powerID)

	status, payload = requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", body)
	require.Equal(s.T(), http.StatusOK, status)

	assert.NotContains(s.T(), payload, childSlug, "фильтр действительно отбирает")
}

// TestHTTPCatalogTotalIsOptional фиксирует, что итог считается только
// по просьбе клиента.
func (s *HTTPTestSuite) TestHTTPCatalogTotalIsOptional() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "total-cat-"+suffix, "Итог")

	for i := range 3 {
		slug := fmt.Sprintf("total-machine-%s-%d", suffix, i)
		machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID, Slug: slug, PriceCents: 1_000_000,
		})
		publishMachine(s.T(), s.httpBasePath, admin, machineID)
	}

	// Без просьбы итога нет.
	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/catalog/machines/search", "",
		fmt.Sprintf(`{"categoryID":%q,"limit":2}`, categoryID))
	require.Equal(s.T(), http.StatusOK, status)

	page := decodeJSONMap(s.T(), payload)
	assert.NotContains(s.T(), page, "total", "без просьбы итог не считается")

	// По просьбе — есть, и он больше размера страницы.
	status, payload = requestJSON(http.MethodPost, s.httpBasePath+"/v1/catalog/machines/search", "",
		fmt.Sprintf(`{"categoryID":%q,"limit":2,"withTotal":true}`, categoryID))
	require.Equal(s.T(), http.StatusOK, status)

	page = decodeJSONMap(s.T(), payload)
	assert.Equal(s.T(), float64(3), page["total"], "итог по всей выдаче, а не по странице")

	items, _ := page["items"].([]any)
	assert.Len(s.T(), items, 2, "страница осталась прежнего размера")
}

// TestHTTPMachineImagesCarryURL фиксирует, что фотографии адресуются URL-ом,
// а не только путём внутри хранилища.
func (s *HTTPTestSuite) TestHTTPMachineImagesCarryURL() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "img-cat-"+suffix, "Фото")

	slug := "img-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 1_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	status, card := publicMachineBySlug(s.httpBasePath, slug)
	require.Equal(s.T(), http.StatusOK, status)

	images, _ := card["images"].([]any)
	require.NotEmpty(s.T(), images, "карточка заводится с фотографией")

	image, _ := images[0].(map[string]any)
	require.NotNil(s.T(), image)

	assert.Equal(s.T(), "a.jpg", image["path"], "путь внутри хранилища остаётся")
	assert.Equal(s.T(), "/media/a.jpg", image["url"], "адрес относительно корня API")
}

// TestHTTPLeasingProgramCRUD фиксирует разделение прав в справочнике программ.
func (s *HTTPTestSuite) TestHTTPLeasingProgramCRUD() {
	admin := adminSession(s.T(), s.httpBasePath)
	manager := managerSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	programID := createLeasingProgram(s.T(), s.httpBasePath, manager, "Программа "+suffix)

	// Программа видна в публичном справочнике.
	status, payload := requestJSON(http.MethodGet, s.httpBasePath+"/v1/leasing-programs", "", "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.Contains(s.T(), payload, programID)

	// Менеджеру удаление запрещено.
	status, _ = requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/leasing-programs/%s", s.httpBasePath, programID), manager.Token, "")
	assert.Equal(s.T(), http.StatusForbidden, status, "удаление — только администратору")

	// Администратору разрешено.
	status, payload = requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/leasing-programs/%s", s.httpBasePath, programID), admin.Token, "")
	assert.Equal(s.T(), http.StatusNoContent, status, "удаление администратором: %s", payload)
}

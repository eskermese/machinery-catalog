//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPCatalogAdminFlow проходит путь редактора целиком: категория,
// характеристики, карточка, публикация — и проверяет, что до публикации
// карточки на витрине нет, а после она там появляется.
//
// Это же покрывает репозитории каталога, которые unit-тестами не проверяются:
// построитель фильтров вынесен отдельно и тестируется без базы, а сами
// запросы проверяются только здесь.
func (s *HTTPTestSuite) TestHTTPCatalogAdminFlow() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "excavators-"+suffix, "Экскаваторы")

	powerID := createNumberAttribute(s.T(), s.httpBasePath, admin, categoryID, "engine_power")
	driveID := createEnumAttribute(s.T(), s.httpBasePath, admin, categoryID, "drive_type",
		[]string{"crawler", "wheeled"})

	slug := "excavator-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID,
		Slug:       slug,
		PriceCents: 8_950_000,
		PowerID:    powerID,
		Power:      "155",
		DriveID:    driveID,
		DriveCode:  "crawler",
	})

	// До публикации карточки на витрине нет.
	status, _ := publicMachineBySlug(s.httpBasePath, slug)
	assert.Equal(s.T(), http.StatusNotFound, status, "черновик не виден публично")

	// Черновик при этом виден в админке.
	status, adminBody := adminMachineBySlug(s.httpBasePath, admin, slug)
	require.Equal(s.T(), http.StatusOK, status, "админка видит черновик")
	assert.Equal(s.T(), false, adminBody["isActive"])

	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// После публикации карточка отдаётся публично вместе с составом.
	status, body := publicMachineBySlug(s.httpBasePath, slug)
	require.Equal(s.T(), http.StatusOK, status, "опубликованная карточка видна публично")
	assert.Equal(s.T(), slug, body["slug"])
	assert.Equal(s.T(), float64(8_950_000), body["priceCnyCents"], "цена отдаётся в центах")

	values, ok := body["attributeValues"].([]any)
	require.True(s.T(), ok, "характеристики уходят вместе с карточкой")
	assert.Len(s.T(), values, 2)

	// Характеристики категории доступны публично без авторизации.
	status, attributes := publicCategoryAttributes(s.httpBasePath, categoryID)
	require.Equal(s.T(), http.StatusOK, status)
	require.Len(s.T(), attributes, 2)

	// Варианты enum приходят вместе с характеристикой: панель фильтров
	// собирается одним запросом.
	for _, attribute := range attributes {
		if attribute["id"] == driveID {
			options, optionsOK := attribute["options"].([]any)
			require.True(s.T(), optionsOK)
			assert.Len(s.T(), options, 2)

			// У enum границ не бывает: подсказка «от — до» под флажком
			// ничего не значит.
			assert.Empty(s.T(), attribute["numberMin"])
			assert.Empty(s.T(), attribute["numberMax"])
		}

		if attribute["id"] == powerID {
			// Границы значений раздела — из них строится подсказка под полем
			// фильтра. Считаются по опубликованным карточкам: карточка выше
			// одна, и обе границы равны её значению.
			assert.Equal(s.T(), "155", attribute["numberMin"])
			assert.Equal(s.T(), "155", attribute["numberMax"])
		}
	}

	assert.NotEmpty(s.T(), powerID)
}

// TestHTTPAttributeCodeConflictIsItsOwnCode фиксирует, что занятый код
// отличается от кривого формата отдельным кодом ошибки.
//
// Раньше оба случая приходили как ATTRIBUTE_CODE_INVALID, и клиент различал
// их только тем, что формат проверял у себя до отправки, — то есть
// по остаточному принципу.
func (s *HTTPTestSuite) TestHTTPAttributeCodeConflictIsItsOwnCode() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "code-cat-"+suffix, "Коды")

	code := "power_" + suffix
	createNumberAttribute(s.T(), s.httpBasePath, admin, categoryID, code)

	// Занятый код: формат верный, имя уже кем-то взято.
	taken := fmt.Sprintf(`{"categoryID":%q,"code":%q,"nameRU":"Х","nameKK":"Х","nameEN":"X",
		"dataType":"number","position":2}`, categoryID, code)

	status, payload := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/admin/catalog/attributes", admin.Token, taken)

	require.Equal(s.T(), http.StatusConflict, status, "занятый код — конфликт: %s", payload)
	assert.Contains(s.T(), payload, entity.CodePrefix+"ATTRIBUTE_CODE_ALREADY_USED")

	// Кривой формат: другая новость и другой код.
	malformed := fmt.Sprintf(`{"categoryID":%q,"code":"Bad Code!","nameRU":"Х","nameKK":"Х",
		"nameEN":"X","dataType":"number","position":3}`, categoryID)

	status, payload = requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/admin/catalog/attributes", admin.Token, malformed)

	require.Equal(s.T(), http.StatusBadRequest, status, "кривой формат — не конфликт: %s", payload)
	assert.Contains(s.T(), payload, entity.CodePrefix+"ATTRIBUTE_CODE_INVALID")
}

// TestHTTPAttributeTypeLockedWhenValuesFilled фиксирует, что смена типа
// у заполненной характеристики отвергается своим кодом.
//
// Защищает её составной внешний ключ (attribute_id, data_type): каскад
// по ON UPDATE переписал бы data_type в строках значений и оставил бы число
// в num_value при новом типе string.
func (s *HTTPTestSuite) TestHTTPAttributeTypeLockedWhenValuesFilled() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "locked-cat-"+suffix, "Смена типа")

	code := "mass_" + suffix
	attributeID := createNumberAttribute(s.T(), s.httpBasePath, admin, categoryID, code)

	// Пока значений нет, тип меняется свободно.
	toString := fmt.Sprintf(`{"categoryID":%q,"code":%q,"nameRU":"Х","nameKK":"Х","nameEN":"X",
		"dataType":"string","position":1}`, categoryID, code)

	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/attributes/%s", s.httpBasePath, attributeID),
		admin.Token, toString)
	require.Equal(s.T(), http.StatusOK, status, "тип без значений меняется: %s", payload)

	// Возвращаем обратно и заполняем значение в карточке.
	toNumber := fmt.Sprintf(`{"categoryID":%q,"code":%q,"nameRU":"Х","nameKK":"Х","nameEN":"X",
		"dataType":"number","unit":"кВт","position":1}`, categoryID, code)

	status, payload = requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/attributes/%s", s.httpBasePath, attributeID),
		admin.Token, toNumber)
	require.Equal(s.T(), http.StatusOK, status, "тип возвращается: %s", payload)

	createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: "locked-machine-" + suffix, PriceCents: 4_000_000,
		PowerID: attributeID, Power: "120",
	})

	// Теперь тип заперт заполненным значением.
	status, payload = requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/attributes/%s", s.httpBasePath, attributeID),
		admin.Token, toString)

	require.Equal(s.T(), http.StatusConflict, status, "смена типа отвергается: %s", payload)
	assert.Contains(s.T(), payload, entity.CodePrefix+"ATTRIBUTE_TYPE_LOCKED")
}

// TestHTTPCatalogAdminSearchReturnsTotal фиксирует, что админский поиск
// отдаёт общее число найденных, когда его просят.
//
// Оно терялось в представлении страницы, и «в разделе есть карточки»
// нельзя было превратить в «в разделе 3 карточки» — а именно это нужно
// сказать перед удалением раздела.
func (s *HTTPTestSuite) TestHTTPCatalogAdminSearchReturnsTotal() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "total-cat-"+suffix, "Счёт")

	const count = 3
	for i := range count {
		createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("total-machine-%s-%d", suffix, i),
			PriceCents: int64(1_000_000 * (i + 1)),
		})
	}

	// Лимит меньше числа карточек: итог считается по всей выдаче, а не
	// по странице, и подменить его длиной items нельзя.
	body := fmt.Sprintf(`{"categoryID":%q,"limit":1,"withTotal":true}`, categoryID)

	status, payload := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/admin/catalog/machines/search", admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "поиск с итогом: %s", payload)

	page := decodeJSONMap(s.T(), payload)
	items, _ := page["items"].([]any)

	assert.Len(s.T(), items, 1, "страница ограничена лимитом")
	assert.InDelta(s.T(), float64(count), page["total"], 0, "итог считается по всей выдаче")

	// Без флага числа нет вовсе: ноль читался бы как «ничего не нашлось».
	body = fmt.Sprintf(`{"categoryID":%q,"limit":1}`, categoryID)

	status, payload = requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/admin/catalog/machines/search", admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "поиск без итога: %s", payload)

	assert.NotContains(s.T(), payload, `"total"`, "без запроса итога поле не приходит")
}

// TestHTTPCatalogSlugConflict фиксирует, что занятый слаг возвращает 409,
// а не внутреннюю ошибку: конфликт приходит от уникального индекса.
func (s *HTTPTestSuite) TestHTTPCatalogSlugConflict() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "loaders-"+suffix, "Погрузчики")

	slug := "loader-" + suffix
	body := fmt.Sprintf(`{"categoryID":%q,"brand":"LiuGong","model":"ZL30","slug":%q,
		"nameRU":"П","nameKK":"П","nameEN":"L","priceCnyCents":3000000,"isPricePublic":true,
		"availability":"in_stock","condition":"new"}`, categoryID, slug)

	status, _ := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/catalog/machines", admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status)

	// Тот же слаг во второй раз.
	status, response := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/catalog/machines",
		admin.Token, body)

	assert.Equal(s.T(), http.StatusConflict, status, "занятый слаг — конфликт, а не 500")
	assert.Contains(s.T(), response, entity.CodePrefix+"SLUG_ALREADY_USED")
}

// TestHTTPCatalogFilters проверяет фильтры на живой базе: числовой диапазон,
// вариант enum и их комбинацию.
func (s *HTTPTestSuite) TestHTTPCatalogFilters() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "filters-"+suffix, "Фильтры")

	powerID := createNumberAttribute(s.T(), s.httpBasePath, admin, categoryID, "engine_power")
	driveID := createEnumAttribute(s.T(), s.httpBasePath, admin, categoryID, "drive_type",
		[]string{"crawler", "wheeled"})

	// Четыре карточки с разной мощностью и приводом.
	specs := []struct {
		power string
		drive string
		price int64
	}{
		{power: "100", drive: "crawler", price: 5_000_000},
		{power: "200", drive: "crawler", price: 7_000_000},
		{power: "300", drive: "wheeled", price: 9_000_000},
		{power: "400", drive: "wheeled", price: 11_000_000},
	}

	for i, spec := range specs {
		id := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("filter-%s-%d", suffix, i),
			PriceCents: spec.price,
			Power:      spec.power,
			DriveCode:  spec.drive,
			PowerID:    powerID,
			DriveID:    driveID,
		})
		publishMachine(s.T(), s.httpBasePath, admin, id)
	}

	// Числовой диапазон: мощность 150…350 отбирает вторую и третью.
	found := searchPublic(s.T(), s.httpBasePath, fmt.Sprintf(`{
		"categoryID":%q,
		"attributes":[{"attributeID":%q,"numberMin":"150","numberMax":"350"}],
		"sort":"price_asc"}`, categoryID, powerID))
	assert.Len(s.T(), found, 2, "числовой диапазон отбирает две карточки")

	// Вариант enum: гусеничные — первая и вторая.
	found = searchPublic(s.T(), s.httpBasePath, fmt.Sprintf(`{
		"categoryID":%q,
		"attributes":[{"attributeID":%q,"codes":["crawler"]}],
		"sort":"price_asc"}`, categoryID, driveID))
	assert.Len(s.T(), found, 2, "фильтр по варианту отбирает две карточки")

	// Комбинация: мощность 150…350 И гусеничный привод — только вторая.
	found = searchPublic(s.T(), s.httpBasePath, fmt.Sprintf(`{
		"categoryID":%q,
		"attributes":[
			{"attributeID":%q,"numberMin":"150","numberMax":"350"},
			{"attributeID":%q,"codes":["crawler"]}
		],
		"sort":"price_asc"}`, categoryID, powerID, driveID))
	require.Len(s.T(), found, 1, "комбинация условий сужает выдачу")
	assert.Equal(s.T(), float64(7_000_000), found[0]["priceCnyCents"])
}

// TestHTTPCatalogPagination проверяет keyset-пагинацию: две страницы подряд
// не пересекаются и вместе покрывают выдачу.
func (s *HTTPTestSuite) TestHTTPCatalogPagination() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "paging-"+suffix, "Пагинация")

	const total = 5

	for i := range total {
		id := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("page-%s-%d", suffix, i),
			PriceCents: int64(1_000_000 * (i + 1)),
		})
		publishMachine(s.T(), s.httpBasePath, admin, id)
	}

	// Первая страница из двух.
	firstPage, firstCursor, hasMore := searchPublicPage(s.T(), s.httpBasePath, fmt.Sprintf(
		`{"categoryID":%q,"sort":"price_asc","limit":2}`, categoryID))
	require.Len(s.T(), firstPage, 2)
	require.True(s.T(), hasMore, "за первой страницей есть ещё карточки")
	require.NotEmpty(s.T(), firstCursor, "курсор следующей страницы возвращён")

	// Вторая страница по курсору первой.
	secondPage, secondCursor, hasMore := searchPublicPage(s.T(), s.httpBasePath, fmt.Sprintf(
		`{"categoryID":%q,"sort":"price_asc","limit":2,"cursor":%q}`, categoryID, firstCursor))
	require.Len(s.T(), secondPage, 2)
	require.True(s.T(), hasMore)
	require.NotEmpty(s.T(), secondCursor)

	// Третья страница по курсору второй.
	thirdPage, _, hasMore := searchPublicPage(s.T(), s.httpBasePath, fmt.Sprintf(
		`{"categoryID":%q,"sort":"price_asc","limit":2,"cursor":%q}`, categoryID, secondCursor))
	require.Len(s.T(), thirdPage, 1, "на последней странице остаток")
	assert.False(s.T(), hasMore, "продолжения больше нет")

	// Страницы не пересекаются и покрывают всю выдачу.
	seen := make(map[string]bool, total)

	for _, page := range [][]map[string]any{firstPage, secondPage, thirdPage} {
		for _, machine := range page {
			id, _ := machine["id"].(string)
			require.False(s.T(), seen[id], "карточка %s встретилась на двух страницах", id)
			seen[id] = true
		}
	}

	assert.Len(s.T(), seen, total, "страницы вместе покрывают всю выдачу")
}

// TestHTTPCatalogSubtreeFilter проверяет фильтр по поддереву: клик по
// родительской категории показывает и карточки дочерней.
func (s *HTTPTestSuite) TestHTTPCatalogSubtreeFilter() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	parentID := createCategory(s.T(), s.httpBasePath, admin, "parent-"+suffix, "Родитель")
	childID := createChildCategory(s.T(), s.httpBasePath, admin, parentID, "child-"+suffix, "Потомок")

	parentMachine := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: parentID, Slug: "parent-machine-" + suffix, PriceCents: 5_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, parentMachine)

	childMachine := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: childID, Slug: "child-machine-" + suffix, PriceCents: 3_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, childMachine)

	// Без поддерева — только карточка самой категории.
	found := searchPublic(s.T(), s.httpBasePath, fmt.Sprintf(
		`{"categoryID":%q,"sort":"price_asc"}`, parentID))
	assert.Len(s.T(), found, 1, "без флага выдача ограничена одним узлом")

	// С поддеревом — обе.
	found = searchPublic(s.T(), s.httpBasePath, fmt.Sprintf(
		`{"categoryID":%q,"includeSubcategories":true,"sort":"price_asc"}`, parentID))
	require.Len(s.T(), found, 2, "поддерево добавляет карточки дочерней категории")

	// Дешевле — карточка потомка: сортировка по цене работает поверх поддерева.
	assert.Equal(s.T(), float64(3_000_000), found[0]["priceCnyCents"])
}

// TestHTTPCatalogPublicSearchNeedsNoAuth фиксирует, что витрина читается
// без токена: каталог смотрят до входа.
func (s *HTTPTestSuite) TestHTTPCatalogPublicSearchNeedsNoAuth() {
	status, _ := requestJSON(http.MethodPost, s.httpBasePath+"/v1/catalog/machines/search", "", `{}`)
	assert.Equal(s.T(), http.StatusOK, status, "публичный поиск не требует токена")

	status, _ = requestJSON(http.MethodGet, s.httpBasePath+"/v1/catalog/categories", "", "")
	assert.Equal(s.T(), http.StatusOK, status, "дерево категорий не требует токена")
}

// TestHTTPCatalogAdminNeedsAuth фиксирует, что админский раздел закрыт.
func (s *HTTPTestSuite) TestHTTPCatalogAdminNeedsAuth() {
	status, response := requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/admin/catalog/categories", "", "")

	assert.Equal(s.T(), http.StatusUnauthorized, status)
	assert.Contains(s.T(), response, entity.CodePrefix+"AUTH_REQUIRED")
}

// TestHTTPCatalogBrokenCursorIs400 фиксирует, что испорченный курсор даёт 400.
func (s *HTTPTestSuite) TestHTTPCatalogBrokenCursorIs400() {
	status, response := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", `{"cursor":"!!!не-курсор!!!"}`)

	assert.Equal(s.T(), http.StatusBadRequest, status, "битый курсор — ошибка клиента, а не сервера")
	assert.Contains(s.T(), response, entity.CodePrefix+"CATALOG_CURSOR_INVALID")
}

// TestHTTPCatalogLimitIsClamped фиксирует урезание размера страницы:
// запрос на десять тысяч карточек отвечает максимумом, а не отказом.
func (s *HTTPTestSuite) TestHTTPCatalogLimitIsClamped() {
	status, _ := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", `{"limit":10000}`)

	assert.Equal(s.T(), http.StatusOK, status, "перебор лимита — не повод отказывать")
}

// TestHTTPCatalogHiddenPriceSortsLast фиксирует место карточек без цены
// в сортировке по цене: они идут ПОСЛЕДНИМИ в обе стороны.
//
// Скрытая цена — не ноль и не бесконечность, а отдельное состояние: раньше
// она хранилась нулём и «сначала дешёвые» начинались с «Цены по запросу»
// вместо самой дешёвой машины.
//
// Три страницы подряд в каждом направлении, потому что проверяется не только
// порядок, но и курсор на границе между карточками с ценой и без: именно там
// сравнение с NULL даёт «неизвестно» вместо «ложь» и роняет из выдачи
// либо хвост, либо всё остальное.
func (s *HTTPTestSuite) TestHTTPCatalogHiddenPriceSortsLast() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "hidden-price-"+suffix, "Скрытая цена")

	const (
		priced = 4
		hidden = 2
		total  = priced + hidden
	)

	for i := range priced {
		id := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("hp-priced-%s-%d", suffix, i),
			PriceCents: int64(1_000_000 * (i + 1)),
		})
		publishMachine(s.T(), s.httpBasePath, admin, id)
	}

	// У скрытых цена НЕ нулевая: если бы порядок задавала сама колонка,
	// они встали бы в середину и подмена осталась бы незамеченной.
	for i := range hidden {
		id := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("hp-hidden-%s-%d", suffix, i),
			PriceCents: 2_500_000,
			HidePrice:  true,
		})
		publishMachine(s.T(), s.httpBasePath, admin, id)
	}

	for _, sort := range []string{"price_asc", "price_desc"} {
		s.Run(sort, func() {
			order := s.collectPages(categoryID, sort, total)

			require.Len(s.T(), order, total, "выдача не потеряла и не удвоила карточки")

			// Хвост без цены — ровно в конце и ровно нужной длины.
			for i, priceVisible := range order {
				if i < priced {
					assert.True(s.T(), priceVisible, "позиция %d: карточка с ценой", i)

					continue
				}

				assert.False(s.T(), priceVisible, "позиция %d: карточка без цены", i)
			}
		})
	}
}

// collectPages читает выдачу тремя страницами по курсору и возвращает признак
// «цена видна» для каждой карточки в порядке выдачи.
func (s *HTTPTestSuite) collectPages(categoryID, sort string, total int) []bool {
	s.T().Helper()

	visible := make([]bool, 0, total)
	seen := make(map[string]bool, total)

	cursor := ""

	for page := range 3 {
		body := fmt.Sprintf(`{"categoryID":%q,"sort":%q,"limit":2`, categoryID, sort)
		if cursor != "" {
			body += fmt.Sprintf(`,"cursor":%q`, cursor)
		}

		body += "}"

		items, next, hasMore := searchPublicPage(s.T(), s.httpBasePath, body)
		require.NotEmpty(s.T(), items, "страница %d не пуста", page+1)

		for _, machine := range items {
			id, _ := machine["id"].(string)
			require.False(s.T(), seen[id], "карточка %s встретилась на двух страницах", id)
			seen[id] = true

			_, hasPrice := machine["priceCnyCents"]
			visible = append(visible, hasPrice)
		}

		if !hasMore {
			break
		}

		cursor = next
	}

	return visible
}

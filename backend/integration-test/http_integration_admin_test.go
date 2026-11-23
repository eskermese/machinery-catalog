//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPAuthMe фиксирует чтение своей учётной записи по токену.
//
// Ручка существует затем, чтобы клиент не разбирал полезную нагрузку токена
// сам: разбор без проверки подписи — доверие к строке, которую он же прислал.
func (s *HTTPTestSuite) TestHTTPAuthMe() {
	manager := managerSession(s.T(), s.httpBasePath)

	status, payload := requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/me", manager.Token, "")
	require.Equal(s.T(), http.StatusOK, status, "своя запись: %s", payload)

	me := decodeJSONMap(s.T(), payload)

	assert.Equal(s.T(), manager.ID, me["id"], "идентификатор совпадает с выданным при входе")
	assert.Equal(s.T(), "manager", me["role"])
	assert.NotEmpty(s.T(), me["phone"], "номер приходит: в токене его нет")
}

// TestHTTPAuthMeNeedsToken фиксирует, что своя запись закрыта авторизацией.
func (s *HTTPTestSuite) TestHTTPAuthMeNeedsToken() {
	status, _ := requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/me", "", "")

	assert.Equal(s.T(), http.StatusUnauthorized, status)
}

// TestHTTPStaffNeedsAdmin фиксирует, что список сотрудников закрыт от менеджера.
func (s *HTTPTestSuite) TestHTTPStaffNeedsAdmin() {
	admin := adminSession(s.T(), s.httpBasePath)
	manager := managerSession(s.T(), s.httpBasePath)

	status, _ := requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/staff", manager.Token, "")
	assert.Equal(s.T(), http.StatusForbidden, status, "менеджеру список сотрудников закрыт")

	status, payload := requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/staff", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status, "администратору открыт: %s", payload)

	assert.Contains(s.T(), payload, adminPhone, "заведённый администратор в списке есть")

	// Отбор по роли.
	status, managersPayload := requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/auth/staff?role=manager", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.NotContains(s.T(), managersPayload, adminPhone, "отбор по роли работает")
}

// TestHTTPStaffNameIsSelfService фиксирует, что имя сотрудник задаёт себе сам
// и что оно доезжает до всех мест, где его показывают.
//
// Проверяется и обратный край: пока имя не задано, поля нет вовсе — клиенту
// есть что показать вместо него, телефон, и подставлять пустоту незачем.
func (s *HTTPTestSuite) TestHTTPStaffNameIsSelfService() {
	// Своя учётная запись, а не общая: имя — состояние записи, и сценарии,
	// которые его меняют, не должны видеть чужие правки.
	manager := freshManagerSession(s.T(), s.httpBasePath)

	// До правки имени поля нет.
	status, payload := requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/me", manager.Token, "")
	require.Equal(s.T(), http.StatusOK, status, "%s", payload)
	assert.NotContains(s.T(), decodeJSONMap(s.T(), payload), "name",
		"незаданное имя не приходит пустой строкой")

	name := "Менеджер " + uniqueSuffix()

	status, payload = requestJSON(http.MethodPatch, s.httpBasePath+"/v1/auth/me",
		manager.Token, fmt.Sprintf(`{"name":%q}`, name))
	require.Equal(s.T(), http.StatusOK, status, "смена имени: %s", payload)
	assert.Equal(s.T(), name, decodeJSONMap(s.T(), payload)["name"])

	// Имя видно в своей записи.
	status, payload = requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/me", manager.Token, "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.Equal(s.T(), name, decodeJSONMap(s.T(), payload)["name"])

	// И в списке сотрудников, который читает администратор.
	admin := adminSession(s.T(), s.httpBasePath)

	status, payload = requestJSON(http.MethodGet, s.httpBasePath+"/v1/auth/staff", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.Contains(s.T(), payload, name, "имя видно в списке сотрудников")

	// Пустая строка очищает имя, а не сохраняется как пустое значение.
	status, payload = requestJSON(http.MethodPatch, s.httpBasePath+"/v1/auth/me",
		manager.Token, `{"name":"   "}`)
	require.Equal(s.T(), http.StatusOK, status, "%s", payload)
	assert.NotContains(s.T(), decodeJSONMap(s.T(), payload), "name", "имя очищено")
}

// TestHTTPLeadCarriesManagerName фиксирует имя ответственного рядом с телефоном
// в заявке: в колонке списка номер читается хуже имени.
func (s *HTTPTestSuite) TestHTTPLeadCarriesManagerName() {
	admin := adminSession(s.T(), s.httpBasePath)
	manager := freshManagerSession(s.T(), s.httpBasePath)

	name := "Ответственный " + uniqueSuffix()

	status, payload := requestJSON(http.MethodPatch, s.httpBasePath+"/v1/auth/me",
		manager.Token, fmt.Sprintf(`{"name":%q}`, name))
	require.Equal(s.T(), http.StatusOK, status, "%s", payload)

	body := fmt.Sprintf(`{"name":"Иван","phone":%q,"source":"manager_name"}`, uniquePhone())

	status, payload = requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status, "%s", payload)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)

	status, payload = requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/admin/leads/%s/assign", s.httpBasePath, leadID), admin.Token,
		fmt.Sprintf(`{"managerID":%q}`, manager.ID))
	require.Equal(s.T(), http.StatusOK, status, "назначение: %s", payload)

	managerRef, _ := decodeJSONMap(s.T(), payload)["manager"].(map[string]any)
	require.NotNil(s.T(), managerRef)
	assert.Equal(s.T(), name, managerRef["name"], "имя приходит рядом с телефоном")
	assert.NotEmpty(s.T(), managerRef["phone"], "телефон остаётся: он есть всегда")
}

// TestHTTPLeadCarriesReadableRefs фиксирует, что рядом с идентификаторами
// приходит то, что можно показать человеку.
func (s *HTTPTestSuite) TestHTTPLeadCarriesReadableRefs() {
	admin := adminSession(s.T(), s.httpBasePath)
	manager := managerSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categorySlug := "ref-cat-" + suffix
	categoryID := createCategory(s.T(), s.httpBasePath, admin, categorySlug, "Ссылки")

	slug := "ref-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 5_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	body := fmt.Sprintf(`{"name":"Иван","phone":%q,"source":"machine_card","machineID":%q}`,
		uniquePhone(), machineID)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status, "приём заявки: %s", payload)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	require.NotEmpty(s.T(), leadID)

	// Назначаем ответственного, чтобы проверить и его ссылку.
	assignBody := fmt.Sprintf(`{"managerID":%q}`, manager.ID)
	status, assignPayload := requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/admin/leads/%s/assign", s.httpBasePath, leadID), admin.Token, assignBody)
	require.Equal(s.T(), http.StatusOK, status, "назначение: %s", assignPayload)

	status, leadPayload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/leads/%s", s.httpBasePath, leadID), admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)

	lead := decodeJSONMap(s.T(), leadPayload)

	machine, _ := lead["machine"].(map[string]any)
	require.NotNil(s.T(), machine, "название машины приходит рядом с идентификатором")
	assert.Equal(s.T(), machineID, machine["id"])
	assert.Equal(s.T(), slug, machine["slug"], "слаг нужен для перехода на карточку")
	assert.Equal(s.T(), categorySlug, machine["categorySlug"],
		"адрес карточки на витрине вложен в категорию: без её слага ссылку не собрать")
	assert.NotEmpty(s.T(), machine["nameRU"])

	managerRef, _ := lead["manager"].(map[string]any)
	require.NotNil(s.T(), managerRef, "ответственный приходит рядом с идентификатором")
	assert.Equal(s.T(), manager.ID, managerRef["id"])
	assert.NotEmpty(s.T(), managerRef["phone"])

	// Допустимые переходы приходят вместе с заявкой.
	transitions, _ := lead["allowedTransitions"].([]any)
	assert.ElementsMatch(s.T(), []any{"in_progress", "won", "lost"}, transitions,
		"перечень зависит от текущего статуса заявки")
}

// TestHTTPLeadTransitionsShrinkWithStatus фиксирует, что перечень переходов
// сужается вместе с продвижением заявки и пустеет на терминальном статусе.
func (s *HTTPTestSuite) TestHTTPLeadTransitionsShrinkWithStatus() {
	admin := adminSession(s.T(), s.httpBasePath)

	body := fmt.Sprintf(`{"name":"Иван","phone":%q,"source":"catalog"}`, uniquePhone())

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	statusURL := fmt.Sprintf("%s/v1/admin/leads/%s/status", s.httpBasePath, leadID)

	status, inProgress := requestJSON(http.MethodPost, statusURL, admin.Token, `{"status":"in_progress"}`)
	require.Equal(s.T(), http.StatusOK, status, "%s", inProgress)

	transitions, _ := decodeJSONMap(s.T(), inProgress)["allowedTransitions"].([]any)
	assert.ElementsMatch(s.T(), []any{"won", "lost"}, transitions, "назад по воронке нельзя")

	status, won := requestJSON(http.MethodPost, statusURL, admin.Token, `{"status":"won"}`)
	require.Equal(s.T(), http.StatusOK, status, "%s", won)

	transitions, _ = decodeJSONMap(s.T(), won)["allowedTransitions"].([]any)
	assert.Empty(s.T(), transitions, "из терминального статуса выхода нет")
}

// TestHTTPLeadsPagination фиксирует keyset-пагинацию списка заявок:
// страницы не пересекаются и покрывают всю выдачу.
func (s *HTTPTestSuite) TestHTTPLeadsPagination() {
	admin := adminSession(s.T(), s.httpBasePath)

	created := make(map[string]bool, 3)

	for range 3 {
		body := fmt.Sprintf(`{"name":"Иван","phone":%q,"source":"pagination"}`, uniquePhone())

		status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
		require.Equal(s.T(), http.StatusCreated, status, "%s", payload)

		id, _ := decodeJSONMap(s.T(), payload)["id"].(string)
		created[id] = true
	}

	seen := make(map[string]bool, len(created))
	cursor := ""

	// Три страницы по одной: каждая приносит новые заявки, повторов нет.
	for page := range 3 {
		url := fmt.Sprintf("%s/v1/admin/leads?limit=1", s.httpBasePath)
		if cursor != "" {
			url += "&cursor=" + cursor
		}

		status, payload := requestJSON(http.MethodGet, url, admin.Token, "")
		require.Equal(s.T(), http.StatusOK, status, "страница %d: %s", page+1, payload)

		body := decodeJSONMap(s.T(), payload)

		items, _ := body["items"].([]any)
		require.Len(s.T(), items, 1, "страница %d", page+1)

		item, _ := items[0].(map[string]any)
		id, _ := item["id"].(string)

		require.False(s.T(), seen[id], "страницы не пересекаются: %s встретился дважды", id)
		seen[id] = true

		cursor, _ = body["nextCursor"].(string)
		require.NotEmpty(s.T(), cursor, "за страницей %d есть продолжение", page+1)
	}

	// Все три только что созданные заявки попали в первые три страницы:
	// список идёт свежими сверху.
	for id := range created {
		assert.True(s.T(), seen[id], "заявка %s не попала в выдачу", id)
	}
}

// TestHTTPLeadsTotalIsOptional фиксирует, что итог считается по просьбе.
func (s *HTTPTestSuite) TestHTTPLeadsTotalIsOptional() {
	admin := adminSession(s.T(), s.httpBasePath)

	status, payload := requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/admin/leads?limit=1", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.NotContains(s.T(), decodeJSONMap(s.T(), payload), "total", "без просьбы итог не считается")

	status, payload = requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/admin/leads?limit=1&withTotal=true", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)

	body := decodeJSONMap(s.T(), payload)
	total, ok := body["total"].(float64)
	require.True(s.T(), ok, "по просьбе итог приходит: %s", payload)
	assert.GreaterOrEqual(s.T(), total, float64(1), "итог по всей выдаче, а не по странице")
}

// TestHTTPCatalogMixedPagination фиксирует пагинацию по выдаче, где черновики
// и опубликованные карточки идут вперемешку.
//
// Проверка нужна именно интеграционная: сравнение с NULL — свойство базы,
// а не построителя запроса, и тест на текст SQL его бы не поймал. Раньше
// здесь ломалось всё: у черновика нет даты публикации, поле не попадало
// в курсор, и следующий запрос отвергал курсор, который сам же выпустил.
//
// Три страницы по две карточки на выдаче из трёх черновиков и трёх
// опубликованных проходят оба края порядка: страница 1 обрывается на
// черновике, страница 2 — на опубликованной карточке.
func (s *HTTPTestSuite) TestHTTPCatalogMixedPagination() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "mixed-cat-"+suffix, "Смешанная")

	const (
		draftCount     = 3
		publishedCount = 3
		pageSize       = 2
		pages          = (draftCount + publishedCount) / pageSize
	)

	expected := make(map[string]bool, draftCount+publishedCount)

	for i := range draftCount {
		slug := fmt.Sprintf("mixed-draft-%d-%s", i, suffix)
		createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID, Slug: slug, PriceCents: int64(1_000_000 + i),
		})
		expected[slug] = true
	}

	for i := range publishedCount {
		slug := fmt.Sprintf("mixed-published-%d-%s", i, suffix)
		id := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID, Slug: slug, PriceCents: int64(2_000_000 + i),
		})
		publishMachine(s.T(), s.httpBasePath, admin, id)
		expected[slug] = true
	}

	seen := make(map[string]bool, len(expected))
	cursor := ""

	for page := range pages {
		body := fmt.Sprintf(`{"categoryID":%q,"limit":%d,"sort":"published_desc","cursor":%q}`,
			categoryID, pageSize, cursor)

		status, payload := requestJSON(http.MethodPost,
			s.httpBasePath+"/v1/admin/catalog/machines/search", admin.Token, body)
		require.Equal(s.T(), http.StatusOK, status, "страница %d: %s", page+1, payload)

		response := decodeJSONMap(s.T(), payload)

		items, _ := response["items"].([]any)
		require.Len(s.T(), items, pageSize, "страница %d полная", page+1)

		for _, raw := range items {
			item, _ := raw.(map[string]any)
			slug, _ := item["slug"].(string)

			require.False(s.T(), seen[slug], "страницы не пересекаются: %s встретился дважды", slug)
			seen[slug] = true
		}

		hasMore, _ := response["hasMore"].(bool)
		cursor, _ = response["nextCursor"].(string)

		if page+1 < pages {
			require.True(s.T(), hasMore, "за страницей %d есть продолжение", page+1)
			require.NotEmpty(s.T(), cursor, "страница %d отдаёт курсор", page+1)

			continue
		}

		assert.False(s.T(), hasMore, "последняя страница не обещает продолжения")
	}

	// Покрытие полное: ни одна карточка не потерялась между страницами.
	assert.Equal(s.T(), expected, seen, "все карточки категории попали в выдачу")
}

// TestHTTPCatalogSearchByQuery фиксирует поиск по подстроке на витрине
// и в админке одним и тем же полем.
func (s *HTTPTestSuite) TestHTTPCatalogSearchByQuery() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "query-cat-"+suffix, "Поиск")

	slug := "query-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 3_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// Бренд задаётся хелпером как XCMG — ищем по его подстроке.
	body := fmt.Sprintf(`{"categoryID":%q,"query":"XCM"}`, categoryID)

	status, payload := requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", body)
	require.Equal(s.T(), http.StatusOK, status, "поиск на витрине: %s", payload)
	assert.Contains(s.T(), payload, slug)

	// Подстрока, которой нет, ничего не возвращает — иначе первая проверка
	// прошла бы и на неработающем поиске.
	body = fmt.Sprintf(`{"categoryID":%q,"query":"такого-бренда-нет"}`, categoryID)

	status, payload = requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", body)
	require.Equal(s.T(), http.StatusOK, status)
	assert.NotContains(s.T(), payload, slug, "поиск действительно отбирает")

	// Символы шаблона в запросе не превращают его в «найти всё».
	body = fmt.Sprintf(`{"categoryID":%q,"query":"%%"}`, categoryID)

	status, payload = requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "", body)
	require.Equal(s.T(), http.StatusOK, status)
	assert.NotContains(s.T(), payload, slug, "процент ищется как символ, а не как шаблон")
}

// TestHTTPCatalogFilterByIsActive фиксирует отбор черновиков в админке
// и его игнорирование на витрине.
func (s *HTTPTestSuite) TestHTTPCatalogFilterByIsActive() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "active-cat-"+suffix, "Активность")

	draftSlug := "active-draft-" + suffix
	createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: draftSlug, PriceCents: 1_000_000,
	})

	publishedSlug := "active-published-" + suffix
	publishedID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: publishedSlug, PriceCents: 2_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, publishedID)

	adminSearch := s.httpBasePath + "/v1/admin/catalog/machines/search"

	// Только черновики.
	status, payload := requestJSON(http.MethodPost, adminSearch, admin.Token,
		fmt.Sprintf(`{"categoryID":%q,"isActive":false}`, categoryID))
	require.Equal(s.T(), http.StatusOK, status, "%s", payload)
	assert.Contains(s.T(), payload, draftSlug)
	assert.NotContains(s.T(), payload, publishedSlug)

	// Только опубликованные.
	status, payload = requestJSON(http.MethodPost, adminSearch, admin.Token,
		fmt.Sprintf(`{"categoryID":%q,"isActive":true}`, categoryID))
	require.Equal(s.T(), http.StatusOK, status, "%s", payload)
	assert.Contains(s.T(), payload, publishedSlug)
	assert.NotContains(s.T(), payload, draftSlug)

	// На витрине отбор игнорируется: черновик не отдаётся даже по прямой просьбе.
	status, payload = requestJSON(http.MethodPost,
		s.httpBasePath+"/v1/catalog/machines/search", "",
		fmt.Sprintf(`{"categoryID":%q,"isActive":false}`, categoryID))
	require.Equal(s.T(), http.StatusOK, status, "%s", payload)
	assert.NotContains(s.T(), payload, draftSlug, "черновик не отдаётся параметром запроса")
}

// TestHTTPQuoteCarriesProgramName фиксирует название программы рядом с её
// идентификатором в расчёте.
func (s *HTTPTestSuite) TestHTTPQuoteCarriesProgramName() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "prog-cat-"+suffix, "Программа")

	slug := "prog-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	programName := "Программа " + suffix
	programID := createLeasingProgram(s.T(), s.httpBasePath, admin, programName)

	body := fmt.Sprintf(`{"machineSlug":%q,"leasingProgramID":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800,"validForDays":14}`, slug, programID)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/quotes", admin.Token, body)
	require.Equal(s.T(), http.StatusCreated, status, "выпуск: %s", payload)

	quote, _ := decodeJSONMap(s.T(), payload)["quote"].(map[string]any)
	require.NotNil(s.T(), quote)

	calculation, _ := quote["calculation"].(map[string]any)
	require.NotNil(s.T(), calculation)

	assert.Equal(s.T(), programID, calculation["leasingProgramID"])
	assert.Equal(s.T(), programName, calculation["leasingProgramName"],
		"название нужно показать: публичный справочник отдаёт только действующие программы")
}

// TestHTTPLeadCommentIsNotClientMessage фиксирует, что комментарий менеджера
// и сообщение клиента — разные поля.
//
// Проверка выглядит тавтологией ровно до того дня, когда кто-нибудь решит
// «сэкономить» и свести их в одно: тогда правка менеджера затрёт запрос
// клиента, а восстановить его будет неоткуда.
func (s *HTTPTestSuite) TestHTTPLeadCommentIsNotClientMessage() {
	const clientMessage = "Нужен масляный фильтр на XE215C, серийный 1234567"

	suffix := uniqueSuffix()
	// Номер уникален на прогон: приём заявок ограничен по частоте
	// с одного номера, и общий телефон уронил бы соседние сценарии.
	phone := fmt.Sprintf("+7701%07s", uniqueSuffix())

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", fmt.Sprintf(
		`{"name":"Клиент %s","phone":%q,"source":"spare_parts","message":%q}`,
		suffix, phone, clientMessage))
	require.Equal(s.T(), http.StatusCreated, status, "приём заявки: %s", payload)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	require.NotEmpty(s.T(), leadID)

	admin := adminSession(s.T(), s.httpBasePath)

	const managerComment = "Клиент постоянный, дать скидку"

	status, payload = requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/admin/leads/%s/comment", s.httpBasePath, leadID),
		admin.Token, fmt.Sprintf(`{"comment":%q}`, managerComment))
	require.Equal(s.T(), http.StatusOK, status, "комментарий менеджера: %s", payload)

	lead := decodeJSONMap(s.T(), payload)
	assert.Equal(s.T(), clientMessage, lead["message"], "запрос клиента не затёрт")
	assert.Equal(s.T(), managerComment, lead["comment"])

	// Пустая строка стирает комментарий, а сообщение клиента остаётся.
	status, payload = requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/admin/leads/%s/comment", s.httpBasePath, leadID),
		admin.Token, `{"comment":""}`)
	require.Equal(s.T(), http.StatusOK, status, "очистка комментария: %s", payload)

	lead = decodeJSONMap(s.T(), payload)
	assert.Empty(s.T(), lead["comment"], "комментарий стёрт")
	assert.Equal(s.T(), clientMessage, lead["message"], "запрос клиента цел")
}

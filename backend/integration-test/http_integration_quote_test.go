//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPQuoteIssueAndRead проходит путь предложения целиком: выпуск через
// админку, чтение клиентом по публичной ссылке, повторное чтение.
//
// Требует загруженного курса: без него выпуск отказывает — подставлять единицу
// нельзя, это тихо продало бы технику по цене в юанях как за тенге.
func (s *HTTPTestSuite) TestHTTPQuoteIssueAndRead() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "quote-cat-"+suffix, "Категория КП")

	slug := "quote-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// Выпуск предложения.
	body := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800,"validForDays":14}`, slug)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/quotes", admin.Token, body)
	require.Equal(s.T(), http.StatusCreated, status, "выпуск предложения: %s", payload)

	issued := decodeJSONMap(s.T(), payload)

	token, _ := issued["publicToken"].(string)
	require.NotEmpty(s.T(), token, "выпуск возвращает токен публичной ссылки")
	require.GreaterOrEqual(s.T(), len(token), entity.MinQuoteTokenLength,
		"токен не короче границы схемы")

	quote, _ := issued["quote"].(map[string]any)
	require.NotNil(s.T(), quote)

	// Расчёт сохранён целиком.
	calculation, _ := quote["calculation"].(map[string]any)
	require.NotNil(s.T(), calculation, "блок расчёта уходит в ответе")

	assert.Positive(s.T(), calculation["assetCostKztCents"], "стоимость в тиынах зафиксирована")
	assert.Positive(s.T(), calculation["financedAmountKztCents"])
	assert.Positive(s.T(), calculation["monthlyPaymentKztCents"])

	schedule, _ := calculation["schedule"].([]any)
	assert.Len(s.T(), schedule, 36, "полный график сохраняется и отдаётся")

	// Курс зафиксирован вместе с датой и наценкой.
	fx, _ := quote["fx"].(map[string]any)
	require.NotNil(s.T(), fx)
	assert.NotEmpty(s.T(), fx["rate"], "применённый курс зафиксирован")
	assert.NotEmpty(s.T(), fx["rateDate"])

	// Чтение по публичной ссылке — без авторизации.
	status, publicPayload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/quotes/%s", s.httpBasePath, token), "", "")
	require.Equal(s.T(), http.StatusOK, status, "предложение читается по ссылке без токена доступа")

	public := decodeJSONMap(s.T(), publicPayload)
	assert.Equal(s.T(), quote["id"], public["id"])
	assert.Equal(s.T(), false, public["isExpired"], "свежее предложение не истекло")

	machine, _ := public["machine"].(map[string]any)
	require.NotNil(s.T(), machine, "машина уходит вместе с предложением")
	assert.Equal(s.T(), slug, machine["slug"])
}

// TestHTTPQuoteFirstOpenedAtIsSetOnce фиксирует однократность отметки первого
// открытия: повторное чтение по ссылке её не сдвигает.
func (s *HTTPTestSuite) TestHTTPQuoteFirstOpenedAtIsSetOnce() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "opened-cat-"+suffix, "Категория")

	slug := "opened-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 5_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	token := issueQuote(s.T(), s.httpBasePath, admin, slug)

	// Первое открытие. Ответ отдаёт момент, который сервис только что положил
	// в память, ещё не прочитав его обратно, — поэтому сравнение идёт
	// с точностью хранения, а не посимвольно: timestamptz хранит микросекунды,
	// а time.Now() в Go даёт наносекунды.
	first := readQuote(s.T(), s.httpBasePath, token)
	firstOpened := openedAt(s.T(), first)

	// Второе открытие: момент не меняется.
	second := readQuote(s.T(), s.httpBasePath, token)
	secondOpened := openedAt(s.T(), second)

	assert.Equal(s.T(), firstOpened, secondOpened,
		"повторное чтение не сдвигает момент первого открытия")

	// Третье — для верности.
	third := readQuote(s.T(), s.httpBasePath, token)
	thirdOpened := openedAt(s.T(), third)

	assert.Equal(s.T(), firstOpened, thirdOpened)
}

// openedAt читает момент первого открытия с точностью хранения.
//
// Округление до микросекунд обязательно: первый ответ отдаёт значение прямо
// из памяти (наносекунды time.Now()), последующие — прочитанное из
// timestamptz, где хвост наносекунд уже потерян. Посимвольное сравнение
// сравнивало бы точность представления, а не тот факт, что отметка не сдвинулась.
func openedAt(t require.TestingT, quote map[string]any) time.Time {
	raw, _ := quote["firstOpenedAt"].(string)
	require.NotEmpty(t, raw, "первое открытие фиксируется")

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	require.NoError(t, err, "момент открытия разбирается как RFC3339: %s", raw)

	return parsed.Truncate(time.Microsecond)
}

// TestHTTPQuoteListCarriesMachineRef фиксирует, что список менеджера называет
// машину и даёт по чему на неё перейти.
//
// Раньше список отдавал пустой объект «машины»: в выборке её не было вовсе,
// а в ответе она стояла как заполненная — менеджер видел строку без названия
// и не мог понять, о какой технике предложение.
func (s *HTTPTestSuite) TestHTTPQuoteListCarriesMachineRef() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categorySlug := "list-cat-" + suffix
	categoryID := createCategory(s.T(), s.httpBasePath, admin, categorySlug, "Категория списка")

	slug := "list-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 7_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	issueQuote(s.T(), s.httpBasePath, admin, slug)

	status, payload := requestJSON(http.MethodGet, s.httpBasePath+"/v1/admin/quotes", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status, "список предложений: %s", payload)

	items, _ := decodeJSONMap(s.T(), payload)["items"].([]any)
	require.NotEmpty(s.T(), items, "выпущенное предложение попадает в список")

	// График в списке не нужен и не приходит: он длиной в срок сделки,
	// а таблица списка его не показывает.
	assert.NotContains(s.T(), payload, `"schedule"`,
		"график платежей в списке не отдаётся")

	var found map[string]any

	for _, item := range items {
		quote, _ := item.(map[string]any)

		ref, _ := quote["machineRef"].(map[string]any)
		if ref != nil && ref["slug"] == slug {
			found = ref

			break
		}
	}

	require.NotNil(s.T(), found, "в списке есть предложение по нашей машине: %s", payload)
	assert.Equal(s.T(), categorySlug, found["categorySlug"],
		"слаг категории нужен для адреса карточки: /catalog/{категория}/{машина}")
	assert.NotEmpty(s.T(), found["nameRU"], "название показывается в строке списка")

	assert.NotContains(s.T(), payload, `"machine":{`,
		"полной карточки в списке нет: её там не читают")
}

// TestHTTPQuoteUnknownTokenIs404 фиксирует, что подобранный токен не отличим
// от несуществующего.
func (s *HTTPTestSuite) TestHTTPQuoteUnknownTokenIs404() {
	// Токен нужной длины, но не выданный сервисом.
	unknown := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	status, payload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/quotes/%s", s.httpBasePath, unknown), "", "")

	assert.Equal(s.T(), http.StatusNotFound, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"QUOTE_NOT_FOUND")
}

// TestHTTPQuoteAdminNeedsAuth фиксирует, что выпуск закрыт авторизацией.
func (s *HTTPTestSuite) TestHTTPQuoteAdminNeedsAuth() {
	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/quotes", "",
		`{"machineSlug":"x","scheduleType":"annuity","termMonths":36}`)

	assert.Equal(s.T(), http.StatusUnauthorized, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"AUTH_REQUIRED")
}

// TestHTTPLeadCreateWritesOutbox фиксирует приём заявки и запись события
// в очередь уведомлений.
func (s *HTTPTestSuite) TestHTTPLeadCreateWritesOutbox() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	phone := uniquePhone()

	// Заявка принимается без авторизации.
	body := fmt.Sprintf(`{"name":"Иван","phone":%q,"city":"Алматы","source":"machine_card"}`, phone)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status, "приём заявки: %s", payload)

	accepted := decodeJSONMap(s.T(), payload)
	leadID, _ := accepted["id"].(string)

	require.NotEmpty(s.T(), leadID)
	assert.Equal(s.T(), "new", accepted["status"], "заявка начинается со статуса new")

	// Заявка видна в админке, номер нормализован.
	status, listPayload := requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/admin/leads?status=new", admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.Contains(s.T(), listPayload, phone, "номер сохранён в нормализованном виде")

	// Событие ушло в очередь уведомлений: канал доставки не подключён,
	// поэтому строка остаётся необработанной — это и проверяем.
	status, leadPayload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/leads/%s", s.httpBasePath, leadID), admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)

	lead := decodeJSONMap(s.T(), leadPayload)
	assert.Equal(s.T(), "Иван", lead["name"])
	assert.Equal(s.T(), "machine_card", lead["source"])

	assert.NotEmpty(s.T(), suffix)
}

// TestHTTPLeadNormalizesPhone фиксирует, что разные написания номера дают
// одну и ту же запись.
func (s *HTTPTestSuite) TestHTTPLeadNormalizesPhone() {
	admin := adminSession(s.T(), s.httpBasePath)

	// Номер в формате «8 7XX…» обязан сохраниться как «+77XX…».
	//
	// Строится из uniquePhone, а не склейкой со счётчиком суффиксов: замена
	// ведущей восьмёрки срабатывает только на номере ровно нужной длины,
	// и склейка «8 701 » + суффикс давала номер, длина которого зависела
	// от разрядности счётчика — то есть от порядка выполнения тестов.
	raw := localPhoneForm(uniquePhone())
	body := fmt.Sprintf(`{"name":"Пётр","phone":%q,"source":"catalog"}`, raw)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status, "приём заявки: %s", payload)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)

	status, leadPayload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/leads/%s", s.httpBasePath, leadID), admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)

	stored, _ := decodeJSONMap(s.T(), leadPayload)["phone"].(string)

	assert.Equal(s.T(), entity.NormalizePhone(raw), stored,
		"номер сохранён в том же виде, в каком его нормализует домен")
	assert.True(s.T(), strings.HasPrefix(stored, "+7"),
		"ведущая восьмёрка заменена на семёрку: %s", stored)
	assert.NotContains(s.T(), stored, " ", "разделители убраны")
}

// TestHTTPLeadRateLimit фиксирует защиту от спама: четвёртая заявка с одного
// номера за час отклоняется.
func (s *HTTPTestSuite) TestHTTPLeadRateLimit() {
	phone := uniquePhone()
	body := fmt.Sprintf(`{"name":"Спам","phone":%q,"source":"catalog"}`, phone)

	// Три заявки проходят.
	for i := range 3 {
		status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
		require.Equal(s.T(), http.StatusCreated, status, "заявка %d: %s", i+1, payload)
	}

	// Четвёртая — отклоняется.
	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)

	assert.Equal(s.T(), http.StatusConflict, status, "четвёртая заявка с номера отклоняется")
	assert.Contains(s.T(), payload, entity.CodePrefix+"LEAD_TOO_FREQUENT")
}

// TestHTTPLeadStatusTransitions фиксирует таблицу переходов на живом API.
func (s *HTTPTestSuite) TestHTTPLeadStatusTransitions() {
	admin := adminSession(s.T(), s.httpBasePath)

	phone := uniquePhone()
	body := fmt.Sprintf(`{"name":"Иван","phone":%q,"source":"catalog"}`, phone)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	statusURL := fmt.Sprintf("%s/v1/admin/leads/%s/status", s.httpBasePath, leadID)

	// new → in_progress: разрешено.
	status, payload = requestJSON(http.MethodPost, statusURL, admin.Token, `{"status":"in_progress"}`)
	require.Equal(s.T(), http.StatusOK, status, "переход в работу: %s", payload)
	assert.Equal(s.T(), "in_progress", decodeJSONMap(s.T(), payload)["status"])

	// in_progress → new: запрещено, назад по воронке нельзя.
	status, payload = requestJSON(http.MethodPost, statusURL, admin.Token, `{"status":"new"}`)
	assert.Equal(s.T(), http.StatusBadRequest, status, "обратный ход запрещён")
	assert.Contains(s.T(), payload, entity.CodePrefix+"LEAD_STATUS_TRANSITION_DENIED")

	// in_progress → won: разрешено.
	status, payload = requestJSON(http.MethodPost, statusURL, admin.Token, `{"status":"won"}`)
	require.Equal(s.T(), http.StatusOK, status)

	// won → lost: запрещено, терминальный статус не воскрешается.
	status, payload = requestJSON(http.MethodPost, statusURL, admin.Token, `{"status":"lost"}`)
	assert.Equal(s.T(), http.StatusBadRequest, status, "из терминального статуса выхода нет")
}

// TestHTTPLeadAssignManager фиксирует назначение ответственного.
func (s *HTTPTestSuite) TestHTTPLeadAssignManager() {
	admin := adminSession(s.T(), s.httpBasePath)

	phone := uniquePhone()
	body := fmt.Sprintf(`{"name":"Иван","phone":%q,"source":"catalog"}`, phone)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)

	assignBody := fmt.Sprintf(`{"managerID":%q}`, admin.ID)
	status, payload = requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/admin/leads/%s/assign", s.httpBasePath, leadID), admin.Token, assignBody)

	require.Equal(s.T(), http.StatusOK, status, "назначение ответственного: %s", payload)

	lead := decodeJSONMap(s.T(), payload)
	assert.Equal(s.T(), admin.ID, lead["managerID"])
	assert.Equal(s.T(), "new", lead["status"], "назначение не двигает стадию")
}

// TestHTTPLeadsAdminNeedsAuth фиксирует, что список заявок закрыт.
func (s *HTTPTestSuite) TestHTTPLeadsAdminNeedsAuth() {
	status, payload := requestJSON(http.MethodGet, s.httpBasePath+"/v1/admin/leads", "", "")

	assert.Equal(s.T(), http.StatusUnauthorized, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"AUTH_REQUIRED")
}

// TestHTTPLeasingProgramsArePublic фиксирует, что справочник программ читается
// без авторизации: калькулятор на витрине работает до входа.
func (s *HTTPTestSuite) TestHTTPLeasingProgramsArePublic() {
	status, _ := requestJSON(http.MethodGet, s.httpBasePath+"/v1/leasing-programs", "", "")

	assert.Equal(s.T(), http.StatusOK, status)
}

// issueQuote выпускает предложение и возвращает токен публичной ссылки.
func issueQuote(t require.TestingT, httpBasePath string, admin authSession, slug string) string {
	body := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":12,"rateBps":1800}`, slug)

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/quotes", admin.Token, body)
	require.Equal(t, http.StatusCreated, status, "выпуск предложения: %s", payload)

	token, _ := decodeJSONMap(t, payload)["publicToken"].(string)
	require.NotEmpty(t, token)

	return token
}

// readQuote читает предложение по публичной ссылке.
func readQuote(t require.TestingT, httpBasePath, token string) map[string]any {
	status, payload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/quotes/%s", httpBasePath, token), "", "")
	require.Equal(t, http.StatusOK, status, "чтение предложения: %s", payload)

	return decodeJSONMap(t, payload)
}

//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// adminPhone телефон администратора интеграционного набора.
// Администратор не может зарегистрироваться сам, поэтому номер заводится заранее
// через ADMIN_PHONES в окружении приложения.
//
// Префикс +7700 закреплён за администратором и НЕ пересекается с диапазоном
// сгенерированных номеров (+7701, см. uniquePhone). Раньше диапазоны совпадали:
// первое значение счётчика давало ровно этот номер, и сценарий «незаведённый
// администратор не может войти» проходил лишь потому, что алфавитный порядок
// методов набора отдавал ему не первое значение.
const adminPhone = "+77000000001"

// managerPhone телефон менеджера интеграционного набора.
// Диапазон +7702 отделён и от администраторского +7700, и от генерируемых
// +7701: сценарии, считающие заявки по номеру, не должны пересекаться с этим.
const managerPhone = "+77020000001"

// phoneCounter обеспечивает уникальность телефонов в пределах прогона.
//
// Начинается со случайного смещения, а не с нуля: контейнер базы между
// прогонами переиспользуется, и второй прогон выдал бы ровно те же номера —
// заявки первого прогона к этому моменту ещё не вышли из часового окна,
// и проверка частоты отклонила бы первую же заявку теста.
var phoneCounter = newPhoneCounter()

// newPhoneCounter создаёт счётчик телефонов со случайным началом.
func newPhoneCounter() *atomic.Int64 {
	counter := &atomic.Int64{}
	counter.Store(rand.Int64N(phoneRange))

	return counter
}

// phoneRange размер диапазона номеров: семь цифр после префикса +7701.
const phoneRange = 10_000_000

// uniquePhone выдаёт новый телефон обычного пользователя.
// Актора определяет вход по номеру, а идентификатор назначает сервис. Диапазон
// +7701 намеренно отделён от администраторского +7700 — совпадение сделало бы
// менеджера администратором и зависело бы от порядка выполнения тестов.
func uniquePhone() string {
	return fmt.Sprintf("+7701%07d", phoneCounter.Add(1)%phoneRange)
}

// localPhoneForm переписывает номер «+7701XXXXXXX» в местное написание
// «8 701 XXX XX XX»: то же значение, другой формат записи.
func localPhoneForm(phone string) string {
	digits := strings.TrimPrefix(phone, "+")

	return fmt.Sprintf("8 %s %s %s %s", digits[1:4], digits[4:7], digits[7:9], digits[9:])
}

// authSession результат входа: токен и выданный сервисом идентификатор.
type authSession struct {
	ID    string // Идентификатор пользователя
	Token string // Токен доступа
}

var (
	sessionsMu sync.Mutex                     // Защищает кэш сессий от конкурентных сценариев
	sessions   = make(map[string]authSession) // Кэш выданных токенов на время прогона
)

// login выпускает токен по паре роль и телефон и кэширует его на время прогона.
// Повторный вход тем же номером возвращает ту же учётную запись, поэтому кэш
// не меняет наблюдаемого поведения, но экономит запросы.
func login(httpBasePath string, role entity.ActorType, phone string) authSession {
	key := role.String() + ":" + phone

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	if session, ok := sessions[key]; ok {
		return session
	}

	url := fmt.Sprintf("%s/v1/auth/%s/token", httpBasePath, role)
	body := fmt.Sprintf(`{"phone": %q}`, phone)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return authSession{}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return authSession{}
	}

	defer resp.Body.Close()

	var issued struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&issued); err != nil {
		return authSession{}
	}

	session := authSession{ID: issued.User.ID, Token: issued.Token}
	sessions[key] = session

	return session
}

// loginFresh выполняет вход в обход кэша: нужен, чтобы убедиться, что повторный
// вход тем же номером возвращает ту же учётную запись, а не новую.
func loginFresh(httpBasePath string, role entity.ActorType, phone string) authSession {
	sessionsMu.Lock()
	delete(sessions, role.String()+":"+phone)
	sessionsMu.Unlock()

	return login(httpBasePath, role, phone)
}

// adminLoginStatus возвращает код ответа на попытку входа администратором.
func adminLoginStatus(httpBasePath, phone string) int {
	url := fmt.Sprintf("%s/v1/auth/admin/token", httpBasePath)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url,
		strings.NewReader(fmt.Sprintf(`{"phone": %q}`, phone)))
	if err != nil {
		return 0
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}

	defer resp.Body.Close()

	return resp.StatusCode
}

// suffixCounter обеспечивает уникальность слагов в пределах прогона.
var suffixCounter atomic.Int64

// uniqueSuffix выдаёт суффикс для слагов и кодов.
//
// Нужен потому, что слаги уникальны глобально: тесты набора идут по одной базе,
// и фиксированное имя категории сделало бы второй тест зависимым от порядка
// выполнения первого.
func uniqueSuffix() string {
	return fmt.Sprintf("%d", suffixCounter.Add(1))
}

// adminSession выполняет вход администратором.
func adminSession(t require.TestingT, httpBasePath string) authSession {
	session := login(httpBasePath, entity.ActorTypeAdmin, adminPhone)
	require.NotEmpty(t, session.Token, "администратор должен входить: номер задан в ADMIN_PHONES")

	return session
}

// requestJSON выполняет запрос с телом JSON и возвращает код и тело ответа.
// Пустой токен означает запрос без авторизации.
func requestJSON(method, url, token, body string) (int, string) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		return 0, ""
	}

	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, ""
	}

	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, ""
	}

	return resp.StatusCode, string(payload)
}

// decodeJSONMap разбирает тело ответа в отображение.
func decodeJSONMap(t require.TestingT, payload string) map[string]any {
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload), &body))

	return body
}

// createCategory заводит корневую категорию и возвращает её идентификатор.
func createCategory(t require.TestingT, httpBasePath string, admin authSession, slug, name string) string {
	body := fmt.Sprintf(`{"slug":%q,"nameRU":%q,"nameKK":%q,"nameEN":%q,"position":1,"isActive":true}`,
		slug, name, name, name)

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/catalog/categories",
		admin.Token, body)
	require.Equal(t, http.StatusOK, status, "создание категории: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// createChildCategory заводит категорию внутри родительской.
func createChildCategory(
	t require.TestingT,
	httpBasePath string,
	admin authSession,
	parentID, slug, name string,
) string {
	body := fmt.Sprintf(`{"parentID":%q,"slug":%q,"nameRU":%q,"nameKK":%q,"nameEN":%q,"isActive":true}`,
		parentID, slug, name, name, name)

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/catalog/categories",
		admin.Token, body)
	require.Equal(t, http.StatusOK, status, "создание дочерней категории: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// createNumberAttribute заводит числовую характеристику категории.
func createNumberAttribute(
	t require.TestingT,
	httpBasePath string,
	admin authSession,
	categoryID, code string,
) string {
	body := fmt.Sprintf(`{"categoryID":%q,"code":%q,"nameRU":"Х","nameKK":"Х","nameEN":"X",
		"dataType":"number","unit":"кВт","isFilterable":true,"position":1}`, categoryID, code)

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/catalog/attributes",
		admin.Token, body)
	require.Equal(t, http.StatusOK, status, "создание характеристики: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// createEnumAttribute заводит характеристику типа enum вместе с вариантами.
func createEnumAttribute(
	t require.TestingT,
	httpBasePath string,
	admin authSession,
	categoryID, code string,
	codes []string,
) string {
	options := make([]string, 0, len(codes))
	for _, optionCode := range codes {
		options = append(options,
			fmt.Sprintf(`{"code":%q,"nameRU":"В","nameKK":"В","nameEN":"O","position":1}`, optionCode))
	}

	body := fmt.Sprintf(`{"categoryID":%q,"code":%q,"nameRU":"Х","nameKK":"Х","nameEN":"X",
		"dataType":"enum","isFilterable":true,"position":2,"options":[%s]}`,
		categoryID, code, strings.Join(options, ","))

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/catalog/attributes",
		admin.Token, body)
	require.Equal(t, http.StatusOK, status, "создание характеристики enum: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// machineRequest описывает карточку, создаваемую в тесте.
type machineRequest struct {
	CategoryID string // Категория
	Slug       string // Слаг
	PriceCents int64  // Цена в фынях (1/100 юаня)

	// HidePrice скрыть цену за формой запроса.
	//
	// Такие карточки идут в конце выдачи в обе стороны сортировки по цене,
	// и без них проверить границу между «с ценой» и «без цены» нечем.
	HidePrice bool

	PowerID   string // Числовая характеристика; пусто — значение не задаётся
	Power     string // Значение числовой характеристики
	DriveID   string // Характеристика enum; пусто — значение не задаётся
	DriveCode string // Код варианта

	// ImagePath путь к фотографии в хранилище; пусто — заглушка "a.jpg".
	//
	// Заглушка годится там, где важен сам факт наличия фото. Сценариям
	// удаления нужен настоящий загруженный файл: они проверяют, что он
	// исчезает с диска — или не исчезает, если на него ссылается соседняя
	// карточка.
	ImagePath string
}

// createMachine заводит карточку и возвращает её идентификатор.
func createMachine(t require.TestingT, httpBasePath string, admin authSession, req machineRequest) string {
	values := make([]string, 0, 2)

	if req.PowerID != "" && req.Power != "" {
		values = append(values, fmt.Sprintf(`{"attributeID":%q,"number":%q}`, req.PowerID, req.Power))
	}

	if req.DriveID != "" && req.DriveCode != "" {
		values = append(values, fmt.Sprintf(`{"attributeID":%q,"code":%q}`, req.DriveID, req.DriveCode))
	}

	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":"XE215C","slug":%q,
		"nameRU":"Э","nameKK":"Э","nameEN":"E","priceCnyCents":%d,"isPricePublic":%t,
		"availability":"in_stock","condition":"new","productionYear":2024,
		"attributeValues":[%s],
		"images":[{"path":%q,"position":0,"isPrimary":true}]}`,
		req.CategoryID, req.Slug, req.PriceCents, !req.HidePrice, strings.Join(values, ","),
		imagePathOrStub(req.ImagePath))

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/catalog/machines",
		admin.Token, body)
	require.Equal(t, http.StatusOK, status, "создание карточки: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// imagePathOrStub подставляет заглушку, когда путь не задан.
func imagePathOrStub(path string) string {
	if path == "" {
		return "a.jpg"
	}

	return path
}

// publishMachine публикует карточку.
func publishMachine(t require.TestingT, httpBasePath string, admin authSession, machineID string) {
	url := fmt.Sprintf("%s/v1/admin/catalog/machines/%s/publish", httpBasePath, machineID)

	status, payload := requestJSON(http.MethodPost, url, admin.Token, "")
	require.Equal(t, http.StatusOK, status, "публикация карточки: %s", payload)
}

// managerSession выполняет вход менеджером.
//
// Отдельный номер на каждый вызов не нужен: сценариям важна роль, а не
// конкретная учётная запись, и общий кэш сессий отдаёт одного и того же
// менеджера всему набору.
func managerSession(t require.TestingT, httpBasePath string) authSession {
	session := login(httpBasePath, entity.ActorTypeManager, managerPhone)
	require.NotEmpty(t, session.Token, "менеджер должен входить свободно")

	return session
}

// freshManagerSession выполняет вход менеджером с новым номером.
//
// Нужен сценариям, которые меняют состояние самой учётной записи: общий
// менеджер из managerSession один на весь набор, и правка из одного теста
// была бы видна другому.
func freshManagerSession(t require.TestingT, httpBasePath string) authSession {
	session := login(httpBasePath, entity.ActorTypeManager, uniquePhone())
	require.NotEmpty(t, session.Token, "менеджер должен входить свободно")

	return session
}

// createMachineHiddenPrice заводит карточку со скрытой ценой.
func createMachineHiddenPrice(
	t require.TestingT,
	httpBasePath string,
	admin authSession,
	categoryID, slug string,
) string {
	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":"XE215C","slug":%q,
		"nameRU":"Э","nameKK":"Э","nameEN":"E","priceCnyCents":9500000,"isPricePublic":false,
		"availability":"in_stock","condition":"new","productionYear":2024,
		"attributeValues":[],"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`,
		categoryID, slug)

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/catalog/machines",
		admin.Token, body)
	require.Equal(t, http.StatusOK, status, "создание карточки со скрытой ценой: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// createLeasingProgram заводит программу лизинга и возвращает её идентификатор.
func createLeasingProgram(t require.TestingT, httpBasePath string, actor authSession, name string) string {
	body := fmt.Sprintf(`{"nameRU":%q,"nameKK":%q,"nameEN":%q,
		"minDownPaymentBps":1500,"termMonthsMin":12,"termMonthsMax":60,
		"rateMinBps":1500,"rateMaxBps":2400,"isActive":true,"position":1}`, name, name, name)

	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/admin/leasing-programs",
		actor.Token, body)
	require.Equal(t, http.StatusOK, status, "создание программы лизинга: %s", payload)

	id, _ := decodeJSONMap(t, payload)["id"].(string)
	require.NotEmpty(t, id)

	return id
}

// publicMachineBySlug читает карточку с витрины.
func publicMachineBySlug(httpBasePath, slug string) (int, map[string]any) {
	status, payload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/catalog/machines/%s", httpBasePath, slug), "", "")

	var body map[string]any
	_ = json.Unmarshal([]byte(payload), &body)

	return status, body
}

// adminMachineBySlug читает карточку из админки, включая черновик.
func adminMachineBySlug(httpBasePath string, admin authSession, slug string) (int, map[string]any) {
	status, payload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", httpBasePath, slug), admin.Token, "")

	var body map[string]any
	_ = json.Unmarshal([]byte(payload), &body)

	return status, body
}

// publicCategoryAttributes читает характеристики категории с витрины.
func publicCategoryAttributes(httpBasePath, categoryID string) (int, []map[string]any) {
	status, payload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/catalog/categories/%s/attributes", httpBasePath, categoryID), "", "")

	var body struct {
		Items []map[string]any `json:"items"`
	}

	_ = json.Unmarshal([]byte(payload), &body)

	return status, body.Items
}

// searchPublic выполняет поиск по витрине и возвращает карточки страницы.
func searchPublic(t require.TestingT, httpBasePath, filter string) []map[string]any {
	items, _, _ := searchPublicPage(t, httpBasePath, filter)

	return items
}

// searchPublicPage выполняет поиск и возвращает карточки, курсор и признак продолжения.
func searchPublicPage(t require.TestingT, httpBasePath, filter string) ([]map[string]any, string, bool) {
	status, payload := requestJSON(http.MethodPost, httpBasePath+"/v1/catalog/machines/search", "", filter)
	require.Equal(t, http.StatusOK, status, "поиск по каталогу: %s", payload)

	var body struct {
		Items      []map[string]any `json:"items"`
		NextCursor string           `json:"nextCursor"`
		HasMore    bool             `json:"hasMore"`
	}

	require.NoError(t, json.Unmarshal([]byte(payload), &body))

	return body.Items, body.NextCursor, body.HasMore
}

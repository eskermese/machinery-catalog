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
	"github.com/eskermese/machinery-catalog/backend/internal/domain/tco"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	v1 "github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources/v1"
	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
)

// testQuoteMachineSlug слаг карточки, по которой выпускается предложение в тестах.
const testQuoteMachineSlug = "xcmg-xe215c"

// testValidToken токен нужной длины: короче MinQuoteTokenLength сервис
// отвергает не глядя, и такой токен проверял бы не транспорт.
const testValidToken = "IEkYYm4kQ0lLZ1hnR2dnUlZUR0FBQUFBQUFBQUFBQQ"

// stubQuoteService — заглушка сервиса предложений.
//
// Рукописная по образцу stubCatalogService: тестам транспорта важно не число
// вызовов, а ЧТО именно ушло в сервис, — поэтому заглушка запоминает аргументы
// и подменяет ответ.
type stubQuoteService struct {
	service.QuoteService // Остальные методы транспорту этих тестов не нужны

	lastIssue  *service.QuoteIssue // Параметры, с которыми позвали выпуск
	lastToken  string              // Токен, с которым позвали публичное чтение
	lastID     string              // Идентификатор, с которым позвали чтение
	lastLimit  int                 // Размер страницы, дошедший до сервиса
	lastAuthor string              // Автор, чей список запросили
	issueCalls int                 // Сколько раз звали выпуск

	lastCalculate  *service.QuoteCalculate // Параметры публичного расчёта
	calculateCalls int                     // Сколько раз звали расчёт

	savedProgram     *entity.LeasingProgram // Сохранённая программа
	deletedProgramID string                 // Удалённая программа

	quote    *entity.Quote
	quotes   entity.Quotes
	programs entity.LeasingPrograms
	err      error
}

func (s *stubQuoteService) Issue(
	_ context.Context, issue *service.QuoteIssue, _ time.Time,
) (*entity.Quote, error) {
	s.lastIssue = issue
	s.issueCalls++

	return s.quote, s.err
}

func (s *stubQuoteService) ByPublicToken(
	_ context.Context, token string, _ time.Time,
) (*entity.Quote, error) {
	s.lastToken = token

	return s.quote, s.err
}

func (s *stubQuoteService) RecalculateByToken(
	_ context.Context, token string, _ service.QuoteRecalculate,
) (*entity.Quote, error) {
	s.lastToken = token

	return s.quote, s.err
}

func (s *stubQuoteService) ByID(_ context.Context, id string) (*entity.Quote, error) {
	s.lastID = id

	return s.quote, s.err
}

func (s *stubQuoteService) ListByAuthor(
	_ context.Context, authorID string, limit int,
) (entity.Quotes, error) {
	s.lastAuthor = authorID
	s.lastLimit = limit

	return s.quotes, s.err
}

func (s *stubQuoteService) LeasingPrograms(context.Context) (entity.LeasingPrograms, error) {
	return s.programs, s.err
}

func (s *stubQuoteService) AllLeasingPrograms(context.Context) (entity.LeasingPrograms, error) {
	return s.programs, s.err
}

func (s *stubQuoteService) SaveLeasingProgram(
	_ context.Context, program *entity.LeasingProgram,
) (*entity.LeasingProgram, error) {
	s.savedProgram = program

	if s.err != nil {
		return nil, s.err
	}

	return program, nil
}

func (s *stubQuoteService) DeleteLeasingProgram(_ context.Context, id string) error {
	s.deletedProgramID = id

	return s.err
}

func (s *stubQuoteService) Calculate(
	_ context.Context, calc *service.QuoteCalculate, _ time.Time,
) (*entity.Quote, error) {
	s.lastCalculate = calc
	s.calculateCalls++

	return s.quote, s.err
}

// stubLeadService — заглушка сервиса заявок.
type stubLeadService struct {
	service.LeadService // Остальные методы транспорту этих тестов не нужны

	lastCreate    *service.LeadCreate // Заявка, дошедшая до сервиса
	deletedLeadID string              // Заявка, которую попросили удалить
	lastFilter    *entity.LeadFilter  // Фильтр списка
	lastID        string              // Идентификатор заявки
	lastStatus    entity.LeadStatus   // Целевой статус
	lastManager   string
	lastComment   string // Назначаемый менеджер
	createCalls   int    // Сколько раз звали приём заявки

	lead       *entity.Lead
	leads      entity.Leads
	nextCursor string
	hasMore    bool
	err        error
}

func (s *stubLeadService) Create(
	_ context.Context, create *service.LeadCreate, _ time.Time,
) (*entity.Lead, error) {
	s.lastCreate = create
	s.createCalls++

	return s.lead, s.err
}

func (s *stubLeadService) List(_ context.Context, filter *entity.LeadFilter) (entity.LeadPage, error) {
	s.lastFilter = filter

	return entity.LeadPage{Leads: s.leads, NextCursor: s.nextCursor, HasMore: s.hasMore}, s.err
}

func (s *stubLeadService) ByID(_ context.Context, id string) (*entity.Lead, error) {
	s.lastID = id

	return s.lead, s.err
}

func (s *stubLeadService) ChangeStatus(
	_ context.Context, id string, target entity.LeadStatus, _ time.Time,
) (*entity.Lead, error) {
	s.lastID = id
	s.lastStatus = target

	return s.lead, s.err
}

func (s *stubLeadService) AssignManager(
	_ context.Context, id, managerID string, _ time.Time,
) (*entity.Lead, error) {
	s.lastID = id
	s.lastManager = managerID

	return s.lead, s.err
}

func (s *stubLeadService) SetComment(
	_ context.Context, id, comment string, _ time.Time,
) (*entity.Lead, error) {
	s.lastID = id
	s.lastComment = comment

	return s.lead, s.err
}

// publicQuoteRequest выполняет запрос к публичному роутеру предложений и заявок.
func publicQuoteRequest(
	t *testing.T, quotes *stubQuoteService, leads *stubLeadService,
	method, target, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	v1.NewQuoteHandler(quotes, leads, nil, nil).Routes().ServeHTTP(w, r)

	return w
}

// adminQuoteRequest выполняет запрос к административному роутеру от имени актора.
func adminQuoteRequest(
	t *testing.T, quotes *stubQuoteService, leads *stubLeadService,
	actor entity.Actor, method, target, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(v1.WithActorForTest(r.Context(), actor))

	w := httptest.NewRecorder()
	v1.NewQuoteAdminHandler(quotes, leads).Routes().ServeHTTP(w, r)

	return w
}

// testManagerActor возвращает актора-менеджера.
func testManagerActor() entity.Actor {
	return entity.NewActor("7c9e6679-7425-40de-944b-e07fc1f90ae7", entity.ActorTypeManager)
}

// validIssueBody корректное тело выпуска предложения.
const validIssueBody = `{
	"machineSlug": "` + testQuoteMachineSlug + `",
	"scheduleType": "annuity",
	"downPaymentBps": 2000,
	"termMonths": 36,
	"rateBps": 1800
}`

// testQuote собирает предложение, пригодное для представления.
func testQuote(expiresAt, createdAt time.Time) *entity.Quote {
	return &entity.Quote{
		ID:                     "quote-1",
		PublicToken:            testValidToken,
		MachineID:              "machine-1",
		PriceCNYCents:          8_950_000,
		FXRate:                 decimal.RequireFromString("532.786800"),
		FXRateDate:             createdAt,
		FXMarkupBps:            200,
		AssetCostKZTCents:      4_675_943_000,
		ScheduleType:           entity.LeasingScheduleAnnuity,
		DownPaymentBps:         2000,
		TermMonths:             36,
		RateBps:                1800,
		FinancedAmountKZTCents: 3_740_754_400,
		MonthlyPaymentKZTCents: 143_214_400,
		LastPaymentKZTCents:    143_214_500,
		TotalPaidKZTCents:      5_155_718_500,
		OverpaymentKZTCents:    1_414_964_100,
		PaymentSchedule: entity.QuotePayments{
			{Number: 1, PrincipalCents: 7_871_440, InterestCents: 6_450_000, AmountCents: 14_321_440},
		},
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
		Machine: &entity.Machine{
			ID: "machine-1", Slug: testQuoteMachineSlug, PriceCNYCents: 8_950_000, IsPricePublic: true,
		},
	}
}

// ---- Выпуск предложения ----

// TestQuoteAdminHandler_IssueInvalidFormIs400 фиксирует, что невалидная форма
// отвергается транспортом и до сервиса не доходит: сервис не должен разбираться
// в том, что клиент прислал заведомый мусор.
func TestQuoteAdminHandler_IssueInvalidFormIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "битый json", body: `{не json`},
		{name: "нет слага карточки", body: `{"scheduleType":"annuity","termMonths":36}`},
		{name: "неизвестный тип графика", body: `{"machineSlug":"x","scheduleType":"bullet","termMonths":36}`},
		{name: "нулевой срок", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":0}`},
		{name: "взнос больше ста процентов", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"downPaymentBps":10001}`},
		{name: "отрицательная ставка", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"rateBps":-1}`},
		{name: "срок годности больше года", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"validForDays":366}`},
		{name: "программа не uuid", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"leasingProgramID":"not-a-uuid"}`},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{}

			w := adminQuoteRequest(t, quotes, &stubLeadService{},
				testManagerActor(), http.MethodPost, "/quotes", c.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Zero(t, quotes.issueCalls, "невалидная форма до сервиса не доходит")
		})
	}
}

// TestQuoteAdminHandler_IssueUnknownMachineIs404 фиксирует, что отсутствующая
// карточка даёт 404, а не 500.
func TestQuoteAdminHandler_IssueUnknownMachineIs404(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrMachineNotFound}

	w := adminQuoteRequest(t, quotes, &stubLeadService{},
		testManagerActor(), http.MethodPost, "/quotes", validIssueBody)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"MACHINE_NOT_FOUND")
}

// TestQuoteAdminHandler_IssueProgramBoundsAre400 фиксирует, что нарушение
// границ программы возвращается конкретной ошибкой, а не общим отказом:
// менеджеру нужно понять, какой параметр поправить.
func TestQuoteAdminHandler_IssueProgramBoundsAre400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     error
		expCode string
	}{
		{name: "взнос ниже минимального", err: entity.ErrLeasingDownPaymentTooLow, expCode: "LEASING_DOWN_PAYMENT_TOO_LOW"},
		{name: "срок вне диапазона", err: entity.ErrLeasingTermOutOfRange, expCode: "LEASING_TERM_OUT_OF_RANGE"},
		{name: "ставка вне диапазона", err: entity.ErrLeasingRateOutOfRange, expCode: "LEASING_RATE_OUT_OF_RANGE"},
		{name: "сумма выше потолка", err: entity.ErrLeasingAmountAboveMaximum, expCode: "LEASING_AMOUNT_ABOVE_MAXIMUM"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{err: c.err}

			w := adminQuoteRequest(t, quotes, &stubLeadService{},
				testManagerActor(), http.MethodPost, "/quotes", validIssueBody)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), entity.CodePrefix+c.expCode)
			assert.Equal(t, 1, quotes.issueCalls, "проверку границ делает сервис, а не транспорт")
		})
	}
}

// TestQuoteAdminHandler_IssueReturnsToken фиксирует, что токен уходит наружу
// ровно один раз — в ответе на выпуск, тому, кто отправит ссылку клиенту.
func TestQuoteAdminHandler_IssueReturnsToken(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(14*24*time.Hour), now)}
	actor := testManagerActor()

	w := adminQuoteRequest(t, quotes, &stubLeadService{},
		actor, http.MethodPost, "/quotes", validIssueBody)

	require.Equal(t, http.StatusCreated, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, testValidToken, body["publicToken"])

	require.NotNil(t, quotes.lastIssue)
	assert.Equal(t, testQuoteMachineSlug, quotes.lastIssue.MachineSlug)
	assert.Equal(t, entity.LeasingScheduleAnnuity, quotes.lastIssue.ScheduleType)
	assert.Equal(t, 2000, quotes.lastIssue.DownPaymentBps)
	assert.Equal(t, *actor.ID, quotes.lastIssue.AuthorID, "автор берётся из контекста, а не из тела")
}

// TestQuoteAdminHandler_IssueValidForDaysBecomesDuration фиксирует перевод
// срока годности из дней в длительность.
func TestQuoteAdminHandler_IssueValidForDaysBecomesDuration(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}

	w := adminQuoteRequest(t, quotes, &stubLeadService{}, testManagerActor(),
		http.MethodPost, "/quotes",
		`{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"validForDays":7}`)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, quotes.lastIssue)
	assert.Equal(t, 7*24*time.Hour, quotes.lastIssue.ValidFor)
}

// TestQuoteAdminHandler_IssueWithoutActor фиксирует пустого автора, когда
// запрос пришёл мимо аутентификации: обработчик не падает, а отдаёт решение
// сервису — сам сценарий отсекается middleware до обработчика.
func TestQuoteAdminHandler_IssueWithoutActor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}

	r := httptest.NewRequest(http.MethodPost, "/quotes", strings.NewReader(validIssueBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	v1.NewQuoteAdminHandler(quotes, &stubLeadService{}).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, quotes.lastIssue)
	assert.Empty(t, quotes.lastIssue.AuthorID)
}

// ---- Чтение по публичной ссылке ----

// TestQuoteHandler_ShortTokenIs404 фиксирует, что короткий токен неотличим
// от несуществующего.
//
// Транспорт токен не судит: он передаёт его сервису как есть, а тот отвергает
// короткий до похода в базу — это проверено в TestQuoteService_ByPublicToken_ShortTokenRejected.
// Здесь фиксируется, что транспорт не режет и не подменяет токен по дороге,
// а отказ сервиса доезжает до клиента как 404.
func TestQuoteHandler_ShortTokenIs404(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrQuoteNotFound}

	w := publicQuoteRequest(t, quotes, &stubLeadService{}, http.MethodGet, "/quotes/short", "")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"QUOTE_NOT_FOUND")
	assert.Equal(t, "short", quotes.lastToken, "транспорт передаёт токен как есть")
}

// TestQuoteHandler_UnknownTokenIs404 фиксирует, что подобранный токен нужной
// длины неотличим от короткого: оба дают один и тот же ответ.
func TestQuoteHandler_UnknownTokenIs404(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrQuoteNotFound}

	w := publicQuoteRequest(t, quotes, &stubLeadService{},
		http.MethodGet, "/quotes/"+testValidToken, "")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, testValidToken, quotes.lastToken)
}

// TestQuoteHandler_ExpiredQuoteIs200 фиксирует, что истёкшее предложение
// отдаётся с признаком истечения, а не отказом: ссылка из переписки не должна
// выглядеть сломанной — клиент должен увидеть, что предложение было.
func TestQuoteHandler_ExpiredQuoteIs200(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(-time.Hour), now.Add(-30*24*time.Hour))}

	w := publicQuoteRequest(t, quotes, &stubLeadService{},
		http.MethodGet, "/quotes/"+testValidToken, "")

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["isExpired"])
}

// TestQuoteHandler_LiveQuoteHasNoToken фиксирует, что действующее предложение
// отдаётся без признака истечения и БЕЗ токена: тому, кто по ссылке уже
// пришёл, повторять право доступа в теле незачем.
func TestQuoteHandler_LiveQuoteHasNoToken(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(14*24*time.Hour), now)}

	w := publicQuoteRequest(t, quotes, &stubLeadService{},
		http.MethodGet, "/quotes/"+testValidToken, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), testValidToken, "токен в представление не попадает")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["isExpired"])

	calculation, ok := body["calculation"].(map[string]any)
	require.True(t, ok, "блок расчёта присутствует целиком")
	assert.Equal(t, float64(143_214_400), calculation["monthlyPaymentKztCents"])
}

// TestQuoteHandler_ServiceFailureIs500 фиксирует, что недоступное хранилище
// даёт 500, а не подменённый ответ.
func TestQuoteHandler_ServiceFailureIs500(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrDatabaseCommon}

	w := publicQuoteRequest(t, quotes, &stubLeadService{},
		http.MethodGet, "/quotes/"+testValidToken, "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- Программы лизинга ----

// TestQuoteHandler_LeasingPrograms фиксирует публичную выдачу справочника.
func TestQuoteHandler_LeasingPrograms(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{programs: entity.LeasingPrograms{
		{ID: "program-1", NameRU: "Партнёр", MinDownPaymentBps: 2000, TermMonthsMin: 12, TermMonthsMax: 60},
	}}

	w := publicQuoteRequest(t, quotes, &stubLeadService{}, http.MethodGet, "/leasing-programs", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "program-1")
}

// TestQuoteHandler_LeasingProgramsFailureIs500 фиксирует отказ справочника.
func TestQuoteHandler_LeasingProgramsFailureIs500(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrDatabaseCommon}

	w := publicQuoteRequest(t, quotes, &stubLeadService{}, http.MethodGet, "/leasing-programs", "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- Заявки с сайта ----

// TestQuoteHandler_LeadPhonePassesThroughUntouched фиксирует, что транспорт
// номер не трогает: нормализация — работа домена, и повторять её правило
// в обработчике значило бы держать формат в двух местах.
func TestQuoteHandler_LeadPhonePassesThroughUntouched(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		phone string
	}{
		{name: "с разделителями", phone: "+7 (700) 111-22-33"},
		{name: "с ведущей восьмёркой", phone: "8 701 222 33 44"},
		{name: "без плюса", phone: "77002223344"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Сервис возвращает уже нормализованный номер — как это делает
			// entity.NewLead в TestNewLead_NormalizesPhone.
			leads := &stubLeadService{lead: &entity.Lead{
				ID: "lead-1", Name: "Иван", Phone: "+77001112233", Status: entity.LeadStatusNew,
			}}

			w := publicQuoteRequest(t, &stubQuoteService{}, leads, http.MethodPost, "/leads",
				`{"name":"Иван","phone":"`+c.phone+`","source":"machine_card"}`)

			require.Equal(t, http.StatusCreated, w.Code)
			require.NotNil(t, leads.lastCreate)
			assert.Equal(t, c.phone, leads.lastCreate.Phone, "в сервис номер уходит как есть")

			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "lead-1", body["id"])
			assert.Equal(t, "new", body["status"])
		})
	}
}

// TestQuoteHandler_LeadTooFrequentIs409 фиксирует отказ по частоте: превышение
// лимита — конфликт состояния, а не ошибка запроса, поэтому 409, а не 400.
func TestQuoteHandler_LeadTooFrequentIs409(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{err: entity.ErrLeadTooFrequent}

	w := publicQuoteRequest(t, &stubQuoteService{}, leads, http.MethodPost, "/leads",
		`{"name":"Иван","phone":"+77001112233","source":"machine_card"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"LEAD_TOO_FREQUENT")
}

// TestQuoteHandler_LeadInvalidFormIs400 фиксирует, что заведомо неполная заявка
// до сервиса не доходит.
func TestQuoteHandler_LeadInvalidFormIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "битый json", body: `{не json`},
		{name: "нет имени", body: `{"phone":"+77001112233","source":"machine_card"}`},
		{name: "нет номера", body: `{"name":"Иван","source":"machine_card"}`},
		{name: "нет источника", body: `{"name":"Иван","phone":"+77001112233"}`},
		{name: "машина не uuid", body: `{"name":"Иван","phone":"+77001112233","source":"x","machineID":"nope"}`},
		{name: "предложение не uuid", body: `{"name":"Иван","phone":"+77001112233","source":"x","quoteID":"nope"}`},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			leads := &stubLeadService{}

			w := publicQuoteRequest(t, &stubQuoteService{}, leads, http.MethodPost, "/leads", c.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Zero(t, leads.createCalls, "невалидная форма до сервиса не доходит")
		})
	}
}

// TestQuoteHandler_LeadCarriesOptionalFields фиксирует, что необязательные поля
// доезжают до сервиса.
func TestQuoteHandler_LeadCarriesOptionalFields(t *testing.T) {
	t.Parallel()

	const (
		machineID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
		quoteID   = "1b4e28ba-2fa1-4d1b-883f-e07fc1f90ae7"
	)

	leads := &stubLeadService{lead: &entity.Lead{ID: "lead-1", Status: entity.LeadStatusNew}}

	w := publicQuoteRequest(t, &stubQuoteService{}, leads, http.MethodPost, "/leads",
		`{"name":"Иван","phone":"+77001112233","source":"quote_page","city":"Алматы",
		  "machineID":"`+machineID+`","quoteID":"`+quoteID+`"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, leads.lastCreate)
	require.Equal(t, &service.LeadCreate{
		Name:      "Иван",
		Phone:     "+77001112233",
		City:      "Алматы",
		MachineID: machineID,
		QuoteID:   quoteID,
		Source:    "quote_page",
	}, leads.lastCreate)
}

// ---- Административная часть ----

// TestQuoteAdminHandler_ListQuotesLimit фиксирует урезание размера страницы
// и то, что список ограничен автором из контекста.
func TestQuoteAdminHandler_ListQuotesLimit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		query     string
		expStatus int
		expLimit  int
	}{
		{name: "лимит не задан", query: "", expStatus: http.StatusOK, expLimit: entity.DefaultMachineLimit},
		{name: "лимит в пределах", query: "?limit=7", expStatus: http.StatusOK, expLimit: 7},
		{name: "лимит сверх предела", query: "?limit=10000", expStatus: http.StatusOK, expLimit: entity.MaxMachineLimit},
		{name: "лимит не число", query: "?limit=много", expStatus: http.StatusBadRequest},
		{name: "отрицательный лимит", query: "?limit=-1", expStatus: http.StatusBadRequest},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{}
			actor := testManagerActor()

			w := adminQuoteRequest(t, quotes, &stubLeadService{},
				actor, http.MethodGet, "/quotes"+c.query, "")

			require.Equal(t, c.expStatus, w.Code)

			if c.expStatus != http.StatusOK {
				return
			}

			assert.Equal(t, c.expLimit, quotes.lastLimit)
			assert.Equal(t, *actor.ID, quotes.lastAuthor, "менеджер видит только свои предложения")
		})
	}
}

// TestQuoteAdminHandler_ListQuotesFailureIs500 фиксирует отказ хранилища.
func TestQuoteAdminHandler_ListQuotesFailureIs500(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrDatabaseCommon}

	w := adminQuoteRequest(t, quotes, &stubLeadService{},
		testManagerActor(), http.MethodGet, "/quotes", "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestQuoteAdminHandler_QuoteByID фиксирует чтение по идентификатору.
func TestQuoteAdminHandler_QuoteByID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}

	w := adminQuoteRequest(t, quotes, &stubLeadService{},
		testManagerActor(), http.MethodGet, "/quotes/quote-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "quote-1", quotes.lastID)
	assert.NotContains(t, w.Body.String(), testValidToken, "токен не уходит и в админский ответ")
}

// TestQuoteAdminHandler_QuoteByIDNotFound фиксирует 404 на чужом идентификаторе.
func TestQuoteAdminHandler_QuoteByIDNotFound(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrQuoteNotFound}

	w := adminQuoteRequest(t, quotes, &stubLeadService{},
		testManagerActor(), http.MethodGet, "/quotes/quote-1", "")

	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestQuoteAdminHandler_ListLeadsFilter фиксирует разбор фильтра заявок.
func TestQuoteAdminHandler_ListLeadsFilter(t *testing.T) {
	t.Parallel()

	const managerID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	leads := &stubLeadService{}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads?status=in_progress&managerID="+managerID+"&limit=5", "")

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, leads.lastFilter)
	require.NotNil(t, leads.lastFilter.Status)
	assert.Equal(t, entity.LeadStatusInProgress, *leads.lastFilter.Status)
	require.NotNil(t, leads.lastFilter.ManagerID)
	assert.Equal(t, managerID, *leads.lastFilter.ManagerID)
	assert.Equal(t, 5, leads.lastFilter.Limit)
}

// TestQuoteAdminHandler_ListLeadsEmptyFilter фиксирует, что пустые параметры
// не превращаются в условие «статус равен пустой строке».
func TestQuoteAdminHandler_ListLeadsEmptyFilter(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads?status=&managerID=", "")

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, leads.lastFilter)
	assert.Nil(t, leads.lastFilter.Status)
	assert.Nil(t, leads.lastFilter.ManagerID)
}

// TestQuoteAdminHandler_ListLeadsBadLimitIs400 фиксирует отказ на битом лимите.
func TestQuoteAdminHandler_ListLeadsBadLimitIs400(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads?limit=много", "")

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, leads.lastFilter, "до сервиса запрос с битым лимитом не доходит")
}

// TestQuoteAdminHandler_ListLeadsFailureIs500 фиксирует отказ хранилища заявок.
func TestQuoteAdminHandler_ListLeadsFailureIs500(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{err: entity.ErrDatabaseCommon}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads", "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestQuoteAdminHandler_LeadByID фиксирует чтение заявки по идентификатору.
func TestQuoteAdminHandler_LeadByID(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{lead: &entity.Lead{
		ID: "lead-1", Name: "Иван", Phone: "+77001112233", Status: entity.LeadStatusNew,
	}}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads/lead-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "lead-1", leads.lastID)
	assert.Contains(t, w.Body.String(), "+77001112233")
}

// TestQuoteAdminHandler_LeadByIDNotFound фиксирует 404 на несуществующей заявке.
func TestQuoteAdminHandler_LeadByIDNotFound(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{err: entity.ErrLeadNotFound}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads/lead-1", "")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"LEAD_NOT_FOUND")
}

// TestQuoteAdminHandler_ChangeStatusDeniedIs400 фиксирует, что недопустимый
// переход отвергается со своим кодом: клиенту нужно отличить его от «заявка
// не найдена» и от битого тела.
func TestQuoteAdminHandler_ChangeStatusDeniedIs400(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{err: entity.ErrLeadStatusTransitionDenied}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/status", `{"status":"new"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"LEAD_STATUS_TRANSITION_DENIED")
	assert.Equal(t, entity.LeadStatusNew, leads.lastStatus, "решение о переходе принимает домен")
}

// TestQuoteAdminHandler_ChangeStatusOK фиксирует успешный перевод.
func TestQuoteAdminHandler_ChangeStatusOK(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{lead: &entity.Lead{ID: "lead-1", Status: entity.LeadStatusInProgress}}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/status", `{"status":"in_progress"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "lead-1", leads.lastID)
	assert.Equal(t, entity.LeadStatusInProgress, leads.lastStatus)
	assert.Contains(t, w.Body.String(), `"status":"in_progress"`)
}

// TestQuoteAdminHandler_ChangeStatusInvalidFormIs400 фиксирует, что статус вне
// перечня отвергается формой — до сервиса такой запрос не доходит.
func TestQuoteAdminHandler_ChangeStatusInvalidFormIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "битый json", body: `{не json`},
		{name: "нет статуса", body: `{}`},
		{name: "статус вне перечня", body: `{"status":"archived"}`},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			leads := &stubLeadService{}

			w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
				http.MethodPost, "/leads/lead-1/status", c.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Empty(t, leads.lastStatus, "невалидная форма до сервиса не доходит")
		})
	}
}

// TestQuoteAdminHandler_AssignLead фиксирует назначение ответственного.
func TestQuoteAdminHandler_AssignLead(t *testing.T) {
	t.Parallel()

	const managerID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	leads := &stubLeadService{lead: &entity.Lead{
		ID: "lead-1", Status: entity.LeadStatusNew, ManagerID: entity.Pointer(managerID),
	}}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/assign", `{"managerID":"`+managerID+`"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, managerID, leads.lastManager)
	assert.Contains(t, w.Body.String(), `"status":"new"`, "назначение не двигает стадию")
}

// TestQuoteAdminHandler_AssignLeadInvalidFormIs400 фиксирует отказ на форме
// назначения без корректного идентификатора.
func TestQuoteAdminHandler_AssignLeadInvalidFormIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "битый json", body: `{не json`},
		{name: "нет менеджера", body: `{}`},
		{name: "менеджер не uuid", body: `{"managerID":"nope"}`},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			leads := &stubLeadService{}

			w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
				http.MethodPost, "/leads/lead-1/assign", c.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Empty(t, leads.lastManager, "невалидная форма до сервиса не доходит")
		})
	}
}

// TestQuoteAdminHandler_AssignLeadNotFound фиксирует 404 при назначении
// на несуществующую заявку.
func TestQuoteAdminHandler_AssignLeadNotFound(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{err: entity.ErrLeadNotFound}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/assign",
		`{"managerID":"7c9e6679-7425-40de-944b-e07fc1f90ae7"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Отображение доменных ошибок в статусы ----

// TestQuoteHandler_DomainErrorsMapToStatuses фиксирует отображение доменных
// ошибок предложений, программ и заявок в HTTP-статусы существующим механизмом.
//
// Таблицей по образцу TestCatalogHandler_DomainErrorsMapToStatuses: статус
// берётся из категории ошибки, а не задаётся обработчиком, и проверяется
// не только код ответа, но и то, что механизм тот же самый.
func TestQuoteHandler_DomainErrorsMapToStatuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		err       error
		expStatus int
		expCode   string
	}{
		// Коммерческие предложения.
		{
			name: "предложение не найдено", err: entity.ErrQuoteNotFound,
			expStatus: http.StatusNotFound, expCode: "QUOTE_NOT_FOUND",
		},
		{
			name: "ссылка испорчена", err: entity.ErrQuoteTokenInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_TOKEN_INVALID",
		},
		{
			name: "предложение без машины", err: entity.ErrQuoteMachineRequired,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_MACHINE_REQUIRED",
		},
		{
			name: "неизвестный тип графика", err: entity.ErrQuoteScheduleTypeInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_SCHEDULE_TYPE_INVALID",
		},
		{
			name: "денежная величина неверна", err: entity.ErrQuoteAmountInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_AMOUNT_INVALID",
		},
		{
			name: "курс предложения неверен", err: entity.ErrQuoteFXInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_FX_INVALID",
		},
		{
			name: "взнос вне диапазона", err: entity.ErrQuoteDownPaymentInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_DOWN_PAYMENT_INVALID",
		},
		{
			name: "срок или ставка неверны", err: entity.ErrQuoteTermInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_TERM_INVALID",
		},
		{
			name: "выкупная выше финансирования", err: entity.ErrQuoteResidualInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_RESIDUAL_INVALID",
		},
		{
			name: "срок годности в прошлом", err: entity.ErrQuoteExpiresAtInvalid,
			expStatus: http.StatusBadRequest, expCode: "QUOTE_EXPIRES_AT_INVALID",
		},
		{
			// Не хватило энтропии на токен — отказ системы, а не запроса.
			name: "токен не выпущен", err: entity.ErrQuoteTokenGeneration,
			expStatus: http.StatusInternalServerError, expCode: "QUOTE_TOKEN_GENERATION",
		},

		// Границы программы лизинга.
		{
			name: "программа не найдена", err: entity.ErrLeasingProgramNotFound,
			expStatus: http.StatusNotFound, expCode: "LEASING_PROGRAM_NOT_FOUND",
		},
		{
			// 400, а не 409: это несоответствие состояния (StateMismatch),
			// и общий механизм apperrors отдаёт для него ошибку запроса —
			// клиент прислал ссылку на программу, которой нельзя пользоваться.
			name: "программа не действует", err: entity.ErrLeasingProgramInactive,
			expStatus: http.StatusBadRequest, expCode: "LEASING_PROGRAM_INACTIVE",
		},
		{
			name: "взнос ниже минимального", err: entity.ErrLeasingDownPaymentTooLow,
			expStatus: http.StatusBadRequest, expCode: "LEASING_DOWN_PAYMENT_TOO_LOW",
		},
		{
			name: "срок вне диапазона программы", err: entity.ErrLeasingTermOutOfRange,
			expStatus: http.StatusBadRequest, expCode: "LEASING_TERM_OUT_OF_RANGE",
		},
		{
			name: "ставка вне диапазона программы", err: entity.ErrLeasingRateOutOfRange,
			expStatus: http.StatusBadRequest, expCode: "LEASING_RATE_OUT_OF_RANGE",
		},
		{
			name: "сумма выше потолка программы", err: entity.ErrLeasingAmountAboveMaximum,
			expStatus: http.StatusBadRequest, expCode: "LEASING_AMOUNT_ABOVE_MAXIMUM",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{err: c.err}

			w := adminQuoteRequest(t, quotes, &stubLeadService{},
				testManagerActor(), http.MethodPost, "/quotes", validIssueBody)

			assert.Equal(t, c.expStatus, w.Code)
			assert.Contains(t, w.Body.String(), entity.CodePrefix+c.expCode)

			var domainErr *apperrors.DomainError
			require.ErrorAs(t, c.err, &domainErr)
			assert.Equal(t, c.expStatus, domainErr.ErrorType.HTTPStatusCode())
		})
	}
}

// TestQuoteHandler_LeadErrorsMapToStatuses фиксирует отображение ошибок заявок.
//
// Отдельной таблицей: они приходят с другого роутера, и проверять их через
// выпуск предложения значило бы проверять не тот путь.
func TestQuoteHandler_LeadErrorsMapToStatuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		err       error
		expStatus int
		expCode   string
	}{
		{
			name: "заявка не найдена", err: entity.ErrLeadNotFound,
			expStatus: http.StatusNotFound, expCode: "LEAD_NOT_FOUND",
		},
		{
			name: "нет имени", err: entity.ErrLeadNameRequired,
			expStatus: http.StatusBadRequest, expCode: "LEAD_NAME_REQUIRED",
		},
		{
			name: "нет источника", err: entity.ErrLeadSourceRequired,
			expStatus: http.StatusBadRequest, expCode: "LEAD_SOURCE_REQUIRED",
		},
		{
			name: "неизвестный статус", err: entity.ErrLeadStatusInvalid,
			expStatus: http.StatusBadRequest, expCode: "LEAD_STATUS_INVALID",
		},
		{
			name: "недопустимый переход", err: entity.ErrLeadStatusTransitionDenied,
			expStatus: http.StatusBadRequest, expCode: "LEAD_STATUS_TRANSITION_DENIED",
		},
		{
			name: "нет ответственного", err: entity.ErrLeadManagerRequired,
			expStatus: http.StatusBadRequest, expCode: "LEAD_MANAGER_REQUIRED",
		},
		{
			name: "предложение без машины", err: entity.ErrLeadQuoteRequiresMachine,
			expStatus: http.StatusBadRequest, expCode: "LEAD_QUOTE_REQUIRES_MACHINE",
		},
		{
			name: "слишком частые заявки", err: entity.ErrLeadTooFrequent,
			expStatus: http.StatusConflict, expCode: "LEAD_TOO_FREQUENT",
		},
		{
			name: "неверный номер", err: entity.ErrPhoneInvalid,
			expStatus: http.StatusBadRequest, expCode: "PHONE_INVALID",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			leads := &stubLeadService{err: c.err}

			w := publicQuoteRequest(t, &stubQuoteService{}, leads, http.MethodPost, "/leads",
				`{"name":"Иван","phone":"+77001112233","source":"machine_card"}`)

			assert.Equal(t, c.expStatus, w.Code)
			assert.Contains(t, w.Body.String(), entity.CodePrefix+c.expCode)

			var domainErr *apperrors.DomainError
			require.ErrorAs(t, c.err, &domainErr)
			assert.Equal(t, c.expStatus, domainErr.ErrorType.HTTPStatusCode())
		})
	}
}

// ---- Публичный расчёт лизинга ----

// validCalculateBody корректное тело расчёта.
const validCalculateBody = `{
	"machineSlug": "` + testQuoteMachineSlug + `",
	"scheduleType": "annuity",
	"downPaymentBps": 2000,
	"termMonths": 36,
	"rateBps": 1800
}`

// TestQuoteHandler_CalculateNeedsNoAuth фиксирует, что расчёт публичный
// и не принимает цену снаружи: в форме такого поля нет вовсе.
func TestQuoteHandler_CalculateNeedsNoAuth(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}

	// Клиент пытается протащить свою цену — поля нет, оно игнорируется.
	body := `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,
		"priceCnyCents":1,"assetCostKztCents":1}`

	w := publicQuoteRequest(t, quotes, &stubLeadService{}, http.MethodPost, "/leasing/calculate", body)

	require.Equal(t, http.StatusOK, w.Code, "расчёт доступен без токена")
	assert.Equal(t, 1, quotes.calculateCalls)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))

	calculation, ok := payload["calculation"].(map[string]any)
	require.True(t, ok, "блок расчёта присутствует")
	assert.Equal(t, float64(143_214_400), calculation["monthlyPaymentKztCents"])

	fx, ok := payload["fx"].(map[string]any)
	require.True(t, ok, "курс расчёта присутствует")
	assert.Equal(t, "532.786800", fx["rate"])
	assert.NotEmpty(t, fx["rateDate"])
	assert.NotContains(t, fx, "markupBps", "наценка наружу не выносится")

	// Ни ссылки, ни идентификатора расчёт не отдаёт.
	assert.NotContains(t, payload, "publicToken")
	assert.NotContains(t, payload, "id")
}

// TestQuoteHandler_CalculateHiddenPriceIs400 фиксирует отказ по машине
// со скрытой ценой отдельным кодом.
func TestQuoteHandler_CalculateHiddenPriceIs400(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrMachinePriceHidden}

	w := publicQuoteRequest(t, quotes, &stubLeadService{},
		http.MethodPost, "/leasing/calculate", validCalculateBody)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"MACHINE_PRICE_HIDDEN")
	assert.Contains(t, w.Body.String(), "по запросу", "клиенту объясняют, почему отказ")
}

// TestQuoteHandler_CalculateProgramBoundsAre400 фиксирует, что публичный
// расчёт отвергает параметры вне границ программы теми же кодами, что и выпуск.
func TestQuoteHandler_CalculateProgramBoundsAre400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     error
		expCode string
	}{
		{name: "взнос ниже минимального", err: entity.ErrLeasingDownPaymentTooLow, expCode: "LEASING_DOWN_PAYMENT_TOO_LOW"},
		{name: "срок вне диапазона", err: entity.ErrLeasingTermOutOfRange, expCode: "LEASING_TERM_OUT_OF_RANGE"},
		{name: "ставка вне диапазона", err: entity.ErrLeasingRateOutOfRange, expCode: "LEASING_RATE_OUT_OF_RANGE"},
		{name: "сумма выше потолка", err: entity.ErrLeasingAmountAboveMaximum, expCode: "LEASING_AMOUNT_ABOVE_MAXIMUM"},
		{name: "программа не действует", err: entity.ErrLeasingProgramInactive, expCode: "LEASING_PROGRAM_INACTIVE"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{err: c.err}

			w := publicQuoteRequest(t, quotes, &stubLeadService{},
				http.MethodPost, "/leasing/calculate", validCalculateBody)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), entity.CodePrefix+c.expCode)
		})
	}
}

// TestQuoteHandler_CalculateInvalidFormIs400 фиксирует, что заведомо негодные
// параметры до сервиса не доходят.
func TestQuoteHandler_CalculateInvalidFormIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "битый json", body: `{не json`},
		{name: "нет слага", body: `{"scheduleType":"annuity","termMonths":36}`},
		{name: "неизвестный график", body: `{"machineSlug":"x","scheduleType":"bullet","termMonths":36}`},
		{name: "нулевой срок", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":0}`},
		{name: "срок больше тридцати лет", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":361}`},
		{name: "взнос больше ста процентов", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"downPaymentBps":10001}`},
		{name: "отрицательная комиссия", body: `{"machineSlug":"x","scheduleType":"annuity","termMonths":36,"commissionKztCents":-1}`},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{}

			w := publicQuoteRequest(t, quotes, &stubLeadService{},
				http.MethodPost, "/leasing/calculate", c.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Zero(t, quotes.calculateCalls, "невалидная форма до сервиса не доходит")
		})
	}
}

// TestQuoteHandler_CalculateCarriesParameters фиксирует, что параметры доезжают
// до сервиса разобранными.
func TestQuoteHandler_CalculateCarriesParameters(t *testing.T) {
	t.Parallel()

	const programID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}

	body := `{"machineSlug":"` + testQuoteMachineSlug + `","leasingProgramID":"` + programID + `",
		"scheduleType":"differentiated","downPaymentBps":3000,"termMonths":24,"rateBps":2100,
		"residualValueKztCents":1000,"commissionKztCents":2000}`

	w := publicQuoteRequest(t, quotes, &stubLeadService{}, http.MethodPost, "/leasing/calculate", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, &service.QuoteCalculate{
		MachineSlug:           testQuoteMachineSlug,
		LeasingProgramID:      programID,
		ScheduleType:          entity.LeasingScheduleDifferentiated,
		DownPaymentBps:        3000,
		TermMonths:            24,
		RateBps:               2100,
		ResidualValueKZTCents: 1000,
		CommissionKZTCents:    2000,
	}, quotes.lastCalculate)
}

// TestQuoteHandler_CalculateRateLimited фиксирует, что превышение частоты
// с адреса даёт 429, а не 409: конфликт не проходит при повторе, а превышение
// частоты проходит — нужно лишь подождать.
func TestQuoteHandler_CalculateRateLimited(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}
	router := v1.NewQuoteHandler(quotes, &stubLeadService{}, nil, nil).Routes()

	send := func() int {
		r := httptest.NewRequest(http.MethodPost, "/leasing/calculate",
			strings.NewReader(validCalculateBody))
		r.Header.Set("Content-Type", "application/json")
		r.RemoteAddr = "203.0.113.7:54321"

		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		return w.Code
	}

	for i := range 30 {
		require.Equal(t, http.StatusOK, send(), "обращение %d в пределах", i+1)
	}

	require.Equal(t, http.StatusTooManyRequests, send(), "31-е отвергается")
}

// ---- Программы лизинга в админке ----

// TestQuoteAdminHandler_SaveLeasingProgram фиксирует сохранение программы.
func TestQuoteAdminHandler_SaveLeasingProgram(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{}

	body := `{"nameRU":"Halyk","nameKK":"Halyk","nameEN":"Halyk",
		"minDownPaymentBps":1500,"termMonthsMin":12,"termMonthsMax":60,
		"rateMinBps":1800,"rateMaxBps":2400,"isActive":true,"position":1}`

	w := adminQuoteRequest(t, quotes, &stubLeadService{}, testManagerActor(),
		http.MethodPost, "/leasing-programs", body)

	require.Equal(t, http.StatusOK, w.Code, "заводить программы может менеджер")
	require.NotNil(t, quotes.savedProgram)
	assert.Equal(t, "Halyk", quotes.savedProgram.NameRU)
	assert.Equal(t, 1500, quotes.savedProgram.MinDownPaymentBps)
	assert.True(t, quotes.savedProgram.IsActive)
}

// TestQuoteAdminHandler_SaveLeasingProgramInvalidFormIs400 фиксирует, что
// негодная форма до сервиса не доходит.
func TestQuoteAdminHandler_SaveLeasingProgramInvalidFormIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "битый json", body: `{не json`},
		{name: "нет названий", body: `{"termMonthsMin":12,"termMonthsMax":60}`},
		{name: "нулевой срок", body: `{"nameRU":"a","nameKK":"a","nameEN":"a","termMonthsMin":0,"termMonthsMax":60}`},
		{name: "взнос больше ста процентов", body: `{"nameRU":"a","nameKK":"a","nameEN":"a","termMonthsMin":12,"termMonthsMax":60,"minDownPaymentBps":10001}`},
		{name: "идентификатор не uuid", body: `{"id":"nope","nameRU":"a","nameKK":"a","nameEN":"a","termMonthsMin":12,"termMonthsMax":60}`},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{}

			w := adminQuoteRequest(t, quotes, &stubLeadService{}, testManagerActor(),
				http.MethodPost, "/leasing-programs", c.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Nil(t, quotes.savedProgram, "невалидная форма до сервиса не доходит")
		})
	}
}

// TestQuoteAdminHandler_ListLeasingPrograms фиксирует, что админский список
// отдаёт и выключенные программы.
func TestQuoteAdminHandler_ListLeasingPrograms(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{programs: entity.LeasingPrograms{
		{ID: "program-1", NameRU: "Действующая", IsActive: true},
		{ID: "program-2", NameRU: "Выключенная", IsActive: false},
	}}

	w := adminQuoteRequest(t, quotes, &stubLeadService{}, testManagerActor(),
		http.MethodGet, "/leasing-programs", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "program-2", "выключенную нужно видеть, чтобы включить")
	assert.Contains(t, w.Body.String(), `"isActive":false`)
}

// TestQuoteAdminHandler_DeleteLeasingProgramNeedsAdmin фиксирует разделение
// прав: удаление необратимо и стирает условия, по которым считали сделку.
func TestQuoteAdminHandler_DeleteLeasingProgramNeedsAdmin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		actor     entity.Actor
		expStatus int
	}{
		{
			name:      "администратор удаляет",
			actor:     entity.NewActor("1b4e28ba-2fa1-4d1b-883f-e07fc1f90ae7", entity.ActorTypeAdmin),
			expStatus: http.StatusNoContent,
		},
		{
			name:      "менеджеру отказано",
			actor:     testManagerActor(),
			expStatus: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			quotes := &stubQuoteService{}

			w := adminQuoteRequest(t, quotes, &stubLeadService{}, c.actor,
				http.MethodDelete, "/leasing-programs/program-1", "")

			require.Equal(t, c.expStatus, w.Code)

			if c.expStatus == http.StatusNoContent {
				assert.Equal(t, "program-1", quotes.deletedProgramID)

				return
			}

			assert.Empty(t, quotes.deletedProgramID, "до сервиса запрос менеджера не доходит")
		})
	}
}

// TestQuoteAdminHandler_DeleteLeasingProgramInUse фиксирует отказ по программе
// с выпущенными предложениями: её выключают флагом, а не удаляют.
func TestQuoteAdminHandler_DeleteLeasingProgramInUse(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrLeasingProgramInUse}

	w := adminQuoteRequest(t, quotes, &stubLeadService{},
		entity.NewActor("1b4e28ba-2fa1-4d1b-883f-e07fc1f90ae7", entity.ActorTypeAdmin),
		http.MethodDelete, "/leasing-programs/program-1", "")

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"LEASING_PROGRAM_IN_USE")
}

// TestQuoteAdminHandler_LeasingProgramFailures фиксирует, что отказы сервиса
// доезжают до клиента, а не превращаются в пустой ответ.
func TestQuoteAdminHandler_LeasingProgramFailures(t *testing.T) {
	t.Parallel()

	t.Run("отказ списка", func(t *testing.T) {
		t.Parallel()

		quotes := &stubQuoteService{err: entity.ErrDatabaseCommon}

		w := adminQuoteRequest(t, quotes, &stubLeadService{}, testManagerActor(),
			http.MethodGet, "/leasing-programs", "")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("отказ сохранения", func(t *testing.T) {
		t.Parallel()

		quotes := &stubQuoteService{err: entity.ErrLeasingTermOutOfRange}

		body := `{"nameRU":"a","nameKK":"a","nameEN":"a","termMonthsMin":60,"termMonthsMax":12}`

		w := adminQuoteRequest(t, quotes, &stubLeadService{}, testManagerActor(),
			http.MethodPost, "/leasing-programs", body)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), entity.CodePrefix+"LEASING_TERM_OUT_OF_RANGE")
	})

	t.Run("программа не найдена", func(t *testing.T) {
		t.Parallel()

		quotes := &stubQuoteService{err: entity.ErrLeasingProgramNotFound}

		w := adminQuoteRequest(t, quotes, &stubLeadService{},
			entity.NewActor("1b4e28ba-2fa1-4d1b-883f-e07fc1f90ae7", entity.ActorTypeAdmin),
			http.MethodDelete, "/leasing-programs/program-1", "")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestQuoteAdminHandler_ListLeadsPagination фиксирует keyset-пагинацию заявок:
// курсор доезжает до сервиса разобранным, а признаки продолжения — до клиента.
func TestQuoteAdminHandler_ListLeadsPagination(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

	encoded, err := entity.LeadCursor{CreatedAt: created, ID: "lead-1"}.Encode()
	require.NoError(t, err)

	leads := &stubLeadService{
		leads:      entity.Leads{{ID: "lead-2", Status: entity.LeadStatusNew}},
		nextCursor: "следующий",
		hasMore:    true,
	}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads?cursor="+encoded, "")

	require.Equal(t, http.StatusOK, w.Code)

	require.NotNil(t, leads.lastFilter)
	require.NotNil(t, leads.lastFilter.Cursor, "курсор доезжает до сервиса разобранным")
	assert.Equal(t, "lead-1", leads.lastFilter.Cursor.ID)
	assert.True(t, created.Equal(leads.lastFilter.Cursor.CreatedAt))

	var page map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	assert.Equal(t, "следующий", page["nextCursor"])
	assert.Equal(t, true, page["hasMore"])
	assert.NotContains(t, page, "total", "итог не запрашивали")
}

// TestQuoteAdminHandler_ListLeadsBrokenCursorIs400 фиксирует, что испорченный
// курсор даёт 400, а не 500: его прислал клиент.
func TestQuoteAdminHandler_ListLeadsBrokenCursorIs400(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodGet, "/leads?cursor=!!!не-курсор!!!", "")

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"CATALOG_CURSOR_INVALID")
	assert.Nil(t, leads.lastFilter, "до сервиса запрос с битым курсором не доходит")
}

// TestQuoteAdminHandler_ListLeadsWithTotal фиксирует, что итог запрашивается
// флагом, а не считается всегда.
func TestQuoteAdminHandler_ListLeadsWithTotal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		query    string
		expTotal bool
	}{
		{name: "без флага", query: "", expTotal: false},
		{name: "флаг выключен", query: "?withTotal=false", expTotal: false},
		{name: "флаг включён", query: "?withTotal=true", expTotal: true},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			leads := &stubLeadService{}

			w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
				http.MethodGet, "/leads"+c.query, "")

			require.Equal(t, http.StatusOK, w.Code)
			require.NotNil(t, leads.lastFilter)
			assert.Equal(t, c.expTotal, leads.lastFilter.WithTotal)
		})
	}
}

// TestQuoteAdminHandler_SetLeadComment фиксирует запись комментария менеджера.
func TestQuoteAdminHandler_SetLeadComment(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{lead: &entity.Lead{
		ID: "lead-1", Status: entity.LeadStatusNew,
		Message: entity.Pointer("Нужен фильтр"),
		Comment: entity.Pointer("Перезвонить после обеда"),
	}}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/comment", `{"comment":"Перезвонить после обеда"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Перезвонить после обеда", leads.lastComment)
	assert.Contains(t, w.Body.String(), `"message":"Нужен фильтр"`,
		"сообщение клиента приходит рядом и не подменяется комментарием")
}

// TestQuoteAdminHandler_SetLeadCommentClears фиксирует, что пустая строка
// доходит до сервиса: она стирает комментарий, а не означает «не менять».
func TestQuoteAdminHandler_SetLeadCommentClears(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{lead: &entity.Lead{ID: "lead-1", Status: entity.LeadStatusNew}}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/comment", `{"comment":""}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, leads.lastComment)
	assert.Equal(t, "lead-1", leads.lastID, "запрос дошёл до сервиса")
}

// TestQuoteAdminHandler_SetLeadCommentTooLongIs400 фиксирует отказ на слишком
// длинном комментарии до обращения к сервису.
func TestQuoteAdminHandler_SetLeadCommentTooLongIs400(t *testing.T) {
	t.Parallel()

	leads := &stubLeadService{lead: &entity.Lead{ID: "lead-1", Status: entity.LeadStatusNew}}

	body := `{"comment":"` + strings.Repeat("a", entity.MaxLeadCommentLength+1) + `"}`

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodPost, "/leads/lead-1/comment", body)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, leads.lastID, "невалидная форма до сервиса не доходит")
}

// stubTCOService отдаёт готовый расчёт стоимости владения.
type stubTCOService struct {
	result  *service.TCOResult
	err     error
	lastReq service.TCORequest // Запрос, дошедший до сервиса
	calls   int
}

func (s *stubTCOService) Calculate(
	_ context.Context, request service.TCORequest, _ time.Time,
) (*service.TCOResult, error) {
	s.calls++
	s.lastReq = request

	return s.result, s.err
}

// Справочник кривых этим тестам не нужен: они про публичный расчёт.
// Методы объявлены пустыми, чтобы заглушка удовлетворяла интерфейсу.
func (s *stubTCOService) ResidualCurves(
	_ context.Context,
) ([]*entity.ResidualValueCurve, error) {
	return nil, nil
}

func (s *stubTCOService) ResidualCurveByCategory(
	_ context.Context, _ string,
) (*entity.ResidualValueCurve, error) {
	return nil, nil
}

func (s *stubTCOService) SaveResidualCurve(
	_ context.Context, curve *entity.ResidualValueCurve,
) (*entity.ResidualValueCurve, error) {
	return curve, nil
}

func (s *stubTCOService) DeleteResidualCurve(_ context.Context, _ string) error {
	return nil
}

func (s *stubLeadService) Delete(_ context.Context, id string) error {
	s.deletedLeadID = id

	return s.err
}

// stubCategoryReader отдаёт дерево категорий для сборки ссылки на карточку.
type stubCategoryReader struct {
	service.CatalogService // Остальные методы обработчику этих тестов не нужны

	categories entity.Categories
	err        error
}

func (s *stubCategoryReader) CategoryTree(context.Context) (entity.Categories, error) {
	return s.categories, s.err
}

// testTCOResult собирает неполный расчёт: расхода топлива у карточки нет.
func testTCOResult() *service.TCOResult {
	return &service.TCOResult{
		Machine: &entity.Machine{
			ID: "m1", Slug: "xcmg-xe215c", CategoryID: "c1", NameRU: "Экскаватор",
		},
		FX: entity.AppliedFXRate{
			Rate: decimal.RequireFromString("500"), RateDate: time.Now().UTC(), MarkupBps: 200,
		},
		Result: tco.Result{
			PurchaseCents: 50_000_000_00,
			OperatorCents: 12_000_000_00,
			TotalHours:    6000,
			TotalCents:    62_000_000_00,
			Missing:       []tco.Missing{tco.MissingFuelConsumption},
		},
	}
}

// validTCOBody валидное тело расчёта стоимости владения.
const validTCOBody = `{"years":3,"hoursPerYear":2000,` +
	`"fuelPriceKztCents":30000,"operatorHourlyKztCents":200000}`

// TestQuoteHandler_CalculateTCO фиксирует расчёт стоимости владения:
// слаг берётся из пути, параметры из тела, а нехватка данных карточки
// доходит до клиента перечнем, а не нулём.
func TestQuoteHandler_CalculateTCO(t *testing.T) {
	t.Parallel()

	tcoStub := &stubTCOService{result: testTCOResult()}
	catalog := &stubCategoryReader{categories: entity.Categories{
		{ID: "c1", Slug: "ekskavatory", NameRU: "Экскаваторы"},
	}}

	router := v1.NewQuoteHandler(&stubQuoteService{}, &stubLeadService{}, tcoStub, catalog).Routes()

	r := httptest.NewRequest(http.MethodPost, "/machines/xcmg-xe215c/tco",
		strings.NewReader(validTCOBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "xcmg-xe215c", tcoStub.lastReq.MachineSlug, "слаг берётся из пути")
	assert.Equal(t, 3, tcoStub.lastReq.Years)

	body := w.Body.String()
	assert.Contains(t, body, `"complete":false`)
	assert.Contains(t, body, `"fuel_consumption"`)
	assert.Contains(t, body, `"ekskavatory"`, "ссылка на карточку собрана")
	// Непосчитанная статья приходит без суммы: ноль здесь был бы враньём.
	assert.NotContains(t, body, `"fuel":{"amountKztCents"`)
}

// TestQuoteHandler_CalculateTCO_WithoutCategoryTree фиксирует, что отказ
// дерева категорий не роняет расчёт: пропадает только ссылка на карточку,
// а цифры, ради которых запрос и делался, остаются.
func TestQuoteHandler_CalculateTCO_WithoutCategoryTree(t *testing.T) {
	t.Parallel()

	router := v1.NewQuoteHandler(&stubQuoteService{}, &stubLeadService{},
		&stubTCOService{result: testTCOResult()},
		&stubCategoryReader{err: entity.ErrCategoryNotFound}).Routes()

	r := httptest.NewRequest(http.MethodPost, "/machines/xcmg-xe215c/tco",
		strings.NewReader(validTCOBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), `"machineRef"`)
}

// TestQuoteHandler_CalculateTCO_ServiceError фиксирует, что отказ сервиса
// разворачивается доменным кодом.
func TestQuoteHandler_CalculateTCO_ServiceError(t *testing.T) {
	t.Parallel()

	router := v1.NewQuoteHandler(&stubQuoteService{}, &stubLeadService{},
		&stubTCOService{err: entity.ErrMachinePriceHidden}, &stubCategoryReader{}).Routes()

	r := httptest.NewRequest(http.MethodPost, "/machines/xcmg-xe215c/tco",
		strings.NewReader(validTCOBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"MACHINE_PRICE_HIDDEN")
}

// TestQuoteHandler_RecalculateQuote фиксирует пересчёт предложения по ссылке.
func TestQuoteHandler_RecalculateQuote(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	quotes := &stubQuoteService{quote: testQuote(now.Add(time.Hour), now)}

	router := v1.NewQuoteHandler(quotes, &stubLeadService{}, nil, nil).Routes()

	r := httptest.NewRequest(http.MethodPost, "/quotes/"+testValidToken+"/recalculate",
		strings.NewReader(`{"scheduleType":"annuity","downPaymentBps":3000,"termMonths":48,"rateBps":1800}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, testValidToken, quotes.lastToken, "токен берётся из пути")
	// Ответ — расчёт, а не предложение: публичного токена в нём нет.
	assert.NotContains(t, w.Body.String(), testValidToken)
}

// TestQuoteHandler_RecalculateQuote_NotFound фиксирует, что несуществующий
// токен отвергается тем же кодом, что и при чтении предложения.
func TestQuoteHandler_RecalculateQuote_NotFound(t *testing.T) {
	t.Parallel()

	quotes := &stubQuoteService{err: entity.ErrQuoteNotFound}
	router := v1.NewQuoteHandler(quotes, &stubLeadService{}, nil, nil).Routes()

	r := httptest.NewRequest(http.MethodPost, "/quotes/"+testValidToken+"/recalculate",
		strings.NewReader(`{"scheduleType":"annuity","termMonths":48}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"QUOTE_NOT_FOUND")
}

// TestQuoteHandler_CalculateTCO_UnknownCategory фиксирует, что чужой раздел
// в дереве не превращается в ссылку: слаг берётся только у своего.
func TestQuoteHandler_CalculateTCO_UnknownCategory(t *testing.T) {
	t.Parallel()

	router := v1.NewQuoteHandler(&stubQuoteService{}, &stubLeadService{},
		&stubTCOService{result: testTCOResult()},
		&stubCategoryReader{categories: entity.Categories{
			{ID: "другой", Slug: "krany", NameRU: "Краны"},
		}}).Routes()

	r := httptest.NewRequest(http.MethodPost, "/machines/xcmg-xe215c/tco",
		strings.NewReader(validTCOBody))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), `"machineRef"`)
	assert.NotContains(t, w.Body.String(), "krany")
}

// TestQuoteHandler_CalculateTCO_InvalidBody фиксирует отказ на невалидном
// вводе: срок обязателен и положителен.
func TestQuoteHandler_CalculateTCO_InvalidBody(t *testing.T) {
	t.Parallel()

	tcoStub := &stubTCOService{result: testTCOResult()}
	router := v1.NewQuoteHandler(&stubQuoteService{}, &stubLeadService{},
		tcoStub, &stubCategoryReader{}).Routes()

	r := httptest.NewRequest(http.MethodPost, "/machines/xcmg-xe215c/tco",
		strings.NewReader(`{"years":0,"hoursPerYear":2000}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Zero(t, tcoStub.calls, "до сервиса невалидный ввод не доходит")
}

// TestQuoteHandler_RecalculateQuote_InvalidBody фиксирует отказ на неизвестном
// типе графика.
func TestQuoteHandler_RecalculateQuote_InvalidBody(t *testing.T) {
	t.Parallel()

	router := v1.NewQuoteHandler(&stubQuoteService{}, &stubLeadService{}, nil, nil).Routes()

	r := httptest.NewRequest(http.MethodPost, "/quotes/"+testValidToken+"/recalculate",
		strings.NewReader(`{"scheduleType":"весёлый","termMonths":48}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// testAdminActor возвращает актора-администратора.
func testAdminActor() entity.Actor {
	return entity.NewActor("7c9e6679-7425-40de-944b-e07fc1f90ae7", entity.ActorTypeAdmin)
}

// TestQuoteAdminHandler_DeleteLead удаляет заявку.
func TestQuoteAdminHandler_DeleteLead(t *testing.T) {
	t.Parallel()

	const leadID = "88888888-8888-4888-8888-888888888888"

	leads := &stubLeadService{}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testAdminActor(),
		http.MethodDelete, "/leads/"+leadID, "")

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, leadID, leads.deletedLeadID)
}

// TestQuoteAdminHandler_DeleteLead_NotFound отдаёт 404.
func TestQuoteAdminHandler_DeleteLead_NotFound(t *testing.T) {
	t.Parallel()

	const leadID = "88888888-8888-4888-8888-888888888888"

	leads := &stubLeadService{err: entity.ErrLeadNotFound}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testAdminActor(),
		http.MethodDelete, "/leads/"+leadID, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestQuoteAdminHandler_DeleteLeadRequiresAdmin закрывает удаление
// от менеджера: оно необратимо и стирает контакт живого клиента.
func TestQuoteAdminHandler_DeleteLeadRequiresAdmin(t *testing.T) {
	t.Parallel()

	const leadID = "88888888-8888-4888-8888-888888888888"

	leads := &stubLeadService{}

	w := adminQuoteRequest(t, &stubQuoteService{}, leads, testManagerActor(),
		http.MethodDelete, "/leads/"+leadID, "")

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, leads.deletedLeadID, "до сервиса запрос не дошёл")
}

// TestQuoteAdminHandler_QuotesAreNotDeletable фиксирует РЕШЕНИЕ, а не пробел:
// предложение удалить нельзя.
//
// Предложение — зафиксированное обязательство, ссылку на которое клиент уже
// держит в руках. Удаление превратило бы присланный ему документ в 404 —
// ровно то «у меня ничего не открывается», от которого страницу и защищали
// пометкой об истечении. Жизненный цикл у предложения свой: оно истекает
// само и после этого открывается с пометкой.
func TestQuoteAdminHandler_QuotesAreNotDeletable(t *testing.T) {
	t.Parallel()

	const quoteID = "99999999-9999-4999-8999-999999999999"

	w := adminQuoteRequest(t, &stubQuoteService{}, &stubLeadService{}, testAdminActor(),
		http.MethodDelete, "/quotes/"+quoteID, "")

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code,
		"ручки удаления предложения нет и быть не должно")
}

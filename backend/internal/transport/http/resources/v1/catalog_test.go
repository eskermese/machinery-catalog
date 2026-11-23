package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	v1 "github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources/v1"
	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
)

// Валюты витрины в тестах. Те же, что и в конфигурации по умолчанию.
const (
	testBaseCurrency  = "CNY"
	testQuoteCurrency = "KZT"
	testUSDCurrency   = "USD"
)

// stubCatalogService — заглушка сервиса каталога.
//
// Рукописная, а не сгенерированная: тестам транспорта нужно не число вызовов,
// а то, ЧТО именно ушло в сервис — поэтому заглушка запоминает фильтр
// и подменяет ответ.
type stubCatalogService struct {
	service.CatalogService // Остальные методы транспорту этих тестов не нужны

	lastFilter  *entity.MachineFilter // Фильтр, с которым позвали поиск
	publicCalls int                   // Сколько раз звали публичный поиск
	adminCalls  int                   // Сколько раз звали админский поиск

	page       entity.MachinePage
	machine    *entity.Machine
	searchErr  error
	machineErr error

	categories entity.Categories
	attributes entity.Attributes
	images     entity.MachineImages
	category   *entity.Category
	attribute  *entity.Attribute
	opErr      error

	curves     []*entity.ResidualValueCurve
	savedCurve *entity.ResidualValueCurve

	savedMachine   *entity.Machine
	savedCategory  *entity.Category
	savedAttribute *entity.Attribute
	primaryImageID string
}

func (s *stubCatalogService) CategoryTree(context.Context) (entity.Categories, error) {
	return s.categories, s.opErr
}

func (s *stubCatalogService) PublicCategoryAttributes(
	context.Context, string,
) (entity.Attributes, error) {
	return s.attributes, s.opErr
}

func (s *stubCatalogService) CategoryAttributes(context.Context, string) (entity.Attributes, error) {
	return s.attributes, s.opErr
}

func (s *stubCatalogService) SaveCategory(_ context.Context, c *entity.Category) (*entity.Category, error) {
	s.savedCategory = c

	if s.opErr != nil {
		return nil, s.opErr
	}

	if s.category != nil {
		return s.category, nil
	}

	return c, nil
}

func (s *stubCatalogService) DeleteCategory(context.Context, string) error { return s.opErr }

func (s *stubCatalogService) DeleteMachine(context.Context, string) error { return s.opErr }

func (s *stubCatalogService) ResidualCurves(
	context.Context,
) ([]*entity.ResidualValueCurve, error) {
	return s.curves, s.opErr
}

func (s *stubCatalogService) ResidualCurveByCategory(
	_ context.Context, categoryID string,
) (*entity.ResidualValueCurve, error) {
	if s.opErr != nil {
		return nil, s.opErr
	}

	return &entity.ResidualValueCurve{CategoryID: categoryID, FirstYearDropBps: 2000}, nil
}

func (s *stubCatalogService) SaveResidualCurve(
	_ context.Context, curve *entity.ResidualValueCurve,
) (*entity.ResidualValueCurve, error) {
	if s.opErr != nil {
		return nil, s.opErr
	}

	s.savedCurve = curve

	return curve, nil
}

func (s *stubCatalogService) DeleteResidualCurve(context.Context, string) error {
	return s.opErr
}

func (s *stubCatalogService) SaveAttribute(_ context.Context, a *entity.Attribute) (*entity.Attribute, error) {
	s.savedAttribute = a

	if s.opErr != nil {
		return nil, s.opErr
	}

	if s.attribute != nil {
		return s.attribute, nil
	}

	return a, nil
}

func (s *stubCatalogService) DeleteAttribute(context.Context, string) error { return s.opErr }

func (s *stubCatalogService) SaveMachine(_ context.Context, m *entity.Machine) (*entity.Machine, error) {
	s.savedMachine = m

	if s.opErr != nil {
		return nil, s.opErr
	}

	return m, nil
}

func (s *stubCatalogService) PublishMachine(context.Context, string, time.Time) (*entity.Machine, error) {
	return s.machine, s.opErr
}

func (s *stubCatalogService) UnpublishMachine(context.Context, string) (*entity.Machine, error) {
	return s.machine, s.opErr
}

func (s *stubCatalogService) MachineImages(context.Context, string) (entity.MachineImages, error) {
	return s.images, s.opErr
}

func (s *stubCatalogService) ReplaceMachineImages(
	_ context.Context, _ string, images entity.MachineImages,
) (entity.MachineImages, error) {
	if s.opErr != nil {
		return nil, s.opErr
	}

	s.images = images

	return images, nil
}

func (s *stubCatalogService) SetPrimaryImage(
	_ context.Context, _, imageID string,
) (entity.MachineImages, error) {
	s.primaryImageID = imageID

	return s.images, s.opErr
}

func (s *stubCatalogService) SearchPublicMachines(
	_ context.Context, filter *entity.MachineFilter,
) (entity.MachinePage, error) {
	s.lastFilter = filter
	s.publicCalls++

	return s.page, s.searchErr
}

func (s *stubCatalogService) SearchAllMachines(
	_ context.Context, filter *entity.MachineFilter,
) (entity.MachinePage, error) {
	s.lastFilter = filter
	s.adminCalls++

	return s.page, s.searchErr
}

func (s *stubCatalogService) PublicMachineBySlug(_ context.Context, _ string) (*entity.Machine, error) {
	return s.machine, s.machineErr
}

func (s *stubCatalogService) MachineBySlug(_ context.Context, _ string) (*entity.Machine, error) {
	return s.machine, s.machineErr
}

// postSearch выполняет запрос поиска к публичному роутеру.
func postSearch(t *testing.T, srv *stubCatalogService, body string) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodPost, "/machines/search", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	v1.NewCatalogHandler(srv, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	return w
}

// TestCatalogHandler_SearchForcesPublicFilter — главная проверка публичного
// роутера: он зовёт только публичный метод сервиса, каким бы ни было тело.
func TestCatalogHandler_SearchForcesPublicFilter(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	// Клиент пытается протащить признак админского чтения — в форме такого
	// поля нет вовсе, поэтому оно молча игнорируется.
	w := postSearch(t, stub, `{"onlyPublished": false, "limit": 5}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, stub.publicCalls, "публичный роутер зовёт только публичный поиск")
	assert.Zero(t, stub.adminCalls, "админский метод с публичного роутера недостижим")
}

// TestCatalogHandler_LimitIsClamped фиксирует урезание размера страницы:
// клиент, запросивший десять тысяч, получает максимум, а не отказ.
func TestCatalogHandler_LimitIsClamped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		expLimit int
	}{
		{name: "лимит не задан", body: `{}`, expLimit: entity.DefaultMachineLimit},
		{name: "нулевой лимит", body: `{"limit": 0}`, expLimit: entity.DefaultMachineLimit},
		{name: "лимит в пределах", body: `{"limit": 7}`, expLimit: 7},
		{name: "лимит на границе", body: `{"limit": 100}`, expLimit: entity.MaxMachineLimit},
		{name: "лимит сверх предела", body: `{"limit": 10000}`, expLimit: entity.MaxMachineLimit},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{}

			w := postSearch(t, stub, c.body)

			require.Equal(t, http.StatusOK, w.Code, "перебор лимита — не повод отказывать")
			require.NotNil(t, stub.lastFilter)
			assert.Equal(t, c.expLimit, stub.lastFilter.Limit)
		})
	}
}

// TestCatalogHandler_BrokenCursorIs400 фиксирует, что испорченный курсор даёт
// 400, а не 500: его прислал клиент.
func TestCatalogHandler_BrokenCursorIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		cursor string
	}{
		{name: "не base64", cursor: "!!!не-курсор!!!"},
		{name: "base64, но не json", cursor: "aGVsbG8"},
		{name: "json без идентификатора", cursor: "eyJzb3J0IjoicHJpY2VfYXNjIn0"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{}

			w := postSearch(t, stub, `{"cursor":"`+c.cursor+`"}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), entity.CodePrefix+"CATALOG_CURSOR_INVALID")
			assert.Zero(t, stub.publicCalls, "до сервиса запрос с битым курсором не доходит")
		})
	}
}

// TestCatalogHandler_ValidCursorReachesService фиксирует, что корректный курсор
// доезжает до сервиса разобранным.
func TestCatalogHandler_ValidCursorReachesService(t *testing.T) {
	t.Parallel()

	encoded, err := entity.MachineCursor{
		Sort: entity.SortPriceAsc, PriceCNYCents: entity.Pointer(int64(7_500_000)), ID: "machine-1",
	}.Encode()
	require.NoError(t, err)

	stub := &stubCatalogService{}

	w := postSearch(t, stub, `{"sort":"price_asc","cursor":"`+encoded+`"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.lastFilter)
	require.NotNil(t, stub.lastFilter.Cursor)
	assert.Equal(t, "machine-1", stub.lastFilter.Cursor.ID)
	require.NotNil(t, stub.lastFilter.Cursor.PriceCNYCents)
	assert.Equal(t, int64(7_500_000), *stub.lastFilter.Cursor.PriceCNYCents)
}

// TestCatalogHandler_CursorSortIsInherited фиксирует, что сортировка берётся
// из курсора, если клиент её не повторил: иначе вторая страница отвергалась бы
// как несовместимая с курсором.
func TestCatalogHandler_CursorSortIsInherited(t *testing.T) {
	t.Parallel()

	encoded, err := entity.MachineCursor{
		Sort: entity.SortPriceDesc, PriceCNYCents: entity.Pointer(int64(100)), ID: "machine-1",
	}.Encode()
	require.NoError(t, err)

	stub := &stubCatalogService{}

	w := postSearch(t, stub, `{"cursor":"`+encoded+`"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.lastFilter)
	assert.Equal(t, entity.SortPriceDesc, stub.lastFilter.Sort)
}

// TestCatalogHandler_FilterIsParsed фиксирует разбор условий по характеристикам.
func TestCatalogHandler_FilterIsParsed(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{
		"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7",
		"includeSubcategories":true,
		"priceMinCents":5000000,
		"availability":"in_stock",
		"condition":"new",
		"attributes":[
			{"attributeID":"11111111-1111-4111-8111-111111111111","numberMin":"150","numberMax":"250.5"},
			{"attributeID":"22222222-2222-4222-8222-222222222222","codes":["crawler","wheeled"]},
			{"attributeID":"33333333-3333-4333-8333-333333333333","flag":true}
		]
	}`

	w := postSearch(t, stub, body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.lastFilter)

	filter := stub.lastFilter
	assert.True(t, filter.IncludeSubcategories)
	require.NotNil(t, filter.PriceMinCents)
	assert.Equal(t, int64(5_000_000), *filter.PriceMinCents)
	require.Len(t, filter.Attributes, 3)

	numberRange, ok := filter.Attributes[0].(entity.NumberRangeCondition)
	require.True(t, ok, "первое условие — числовой диапазон")
	require.NotNil(t, numberRange.Min)
	assert.Equal(t, "150", numberRange.Min.String())

	enumCondition, ok := filter.Attributes[1].(entity.EnumCondition)
	require.True(t, ok, "второе условие — набор вариантов")
	assert.Equal(t, []string{"crawler", "wheeled"}, enumCondition.Codes)

	boolCondition, ok := filter.Attributes[2].(entity.BoolCondition)
	require.True(t, ok, "третье условие — флаг")
	assert.True(t, boolCondition.Flag)
}

// TestCatalogHandler_AmbiguousAttributeFilterIs400 фиксирует отказ на условии,
// задающем несколько видов отбора сразу: молча выбирать один из них нельзя.
func TestCatalogHandler_AmbiguousAttributeFilterIs400(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"attributes":[{"attributeID":"11111111-1111-4111-8111-111111111111",
		"numberMin":"150","codes":["crawler"]}]}`

	w := postSearch(t, stub, body)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Zero(t, stub.publicCalls)
}

// TestCatalogHandler_DraftBySlugIs404 фиксирует, что черновик по публичному
// слагу отдаётся как несуществующий, а не как запрещённый: иначе перебором
// можно было бы узнать, какие карточки готовятся к публикации.
func TestCatalogHandler_DraftBySlugIs404(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{machineErr: entity.ErrMachineNotFound}

	r := httptest.NewRequest(http.MethodGet, "/machines/xcmg-xe215c", nil)
	w := httptest.NewRecorder()

	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"MACHINE_NOT_FOUND")
	assert.NotContains(t, w.Body.String(), "FORBIDDEN", "черновик не должен выглядеть как запрещённый")
}

// TestCatalogHandler_PriceIsHiddenWhenNotPublic фиксирует, что скрытая цена
// не подменяется нулём, а опускается: ноль клиент показал бы как «бесплатно».
func TestCatalogHandler_PriceIsHiddenWhenNotPublic(t *testing.T) {
	t.Parallel()

	published := time.Now().UTC()

	stub := &stubCatalogService{machine: &entity.Machine{
		ID: "m1", Slug: "xcmg-xe215c", PriceCNYCents: 8_950_000,
		IsPricePublic: false, IsActive: true, PublishedAt: &published,
		Availability: entity.AvailabilityInStock, Condition: entity.ConditionNew,
	}}

	r := httptest.NewRequest(http.MethodGet, "/machines/xcmg-xe215c", nil)
	w := httptest.NewRecorder()

	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	_, hasPrice := body["priceCnyCents"]
	assert.False(t, hasPrice, "скрытая цена опускается целиком")
	assert.Equal(t, false, body["isPricePublic"])
}

// TestCatalogHandler_DomainErrorsMapToStatuses фиксирует отображение доменных
// ошибок каталога в HTTP-статусы существующим механизмом.
func TestCatalogHandler_DomainErrorsMapToStatuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		err       error
		expStatus int
		expCode   string
	}{
		{
			name: "карточка не найдена", err: entity.ErrMachineNotFound,
			expStatus: http.StatusNotFound, expCode: "MACHINE_NOT_FOUND",
		},
		{
			name: "категория не найдена", err: entity.ErrCategoryNotFound,
			expStatus: http.StatusNotFound, expCode: "CATEGORY_NOT_FOUND",
		},
		{
			name: "слаг занят", err: entity.ErrSlugAlreadyUsed,
			expStatus: http.StatusConflict, expCode: "SLUG_ALREADY_USED",
		},
		{
			name: "неверный слаг", err: entity.ErrSlugInvalid,
			expStatus: http.StatusBadRequest, expCode: "SLUG_INVALID",
		},
		{
			name: "испорченный курсор", err: entity.ErrCatalogCursorInvalid,
			expStatus: http.StatusBadRequest, expCode: "CATALOG_CURSOR_INVALID",
		},
		{
			name: "неизвестная сортировка", err: entity.ErrCatalogSortInvalid,
			expStatus: http.StatusBadRequest, expCode: "CATALOG_SORT_INVALID",
		},
		{
			name: "размер страницы вне границ", err: entity.ErrCatalogLimitInvalid,
			expStatus: http.StatusBadRequest, expCode: "CATALOG_LIMIT_INVALID",
		},
		{
			name: "активная карточка без даты публикации", err: entity.ErrMachinePublishedAtRequired,
			expStatus: http.StatusBadRequest, expCode: "MACHINE_PUBLISHED_AT_REQUIRED",
		},
		{
			name: "категория с потомками", err: entity.ErrCategoryHasChildren,
			expStatus: http.StatusBadRequest, expCode: "CATEGORY_HAS_CHILDREN",
		},
		{
			name: "цикл в дереве категорий", err: entity.ErrCategoryCycle,
			expStatus: http.StatusBadRequest, expCode: "CATEGORY_CYCLE",
		},
		{
			name: "неизвестный код варианта", err: entity.ErrAttributeOptionUnknown,
			expStatus: http.StatusBadRequest, expCode: "ATTRIBUTE_OPTION_UNKNOWN",
		},
		{
			name: "тип значения не совпадает", err: entity.ErrAttributeValueTypeMismatch,
			expStatus: http.StatusBadRequest, expCode: "ATTRIBUTE_VALUE_TYPE_MISMATCH",
		},
		{
			name: "характеристика чужой категории", err: entity.ErrAttributeForeignCategory,
			expStatus: http.StatusBadRequest, expCode: "ATTRIBUTE_FOREIGN_CATEGORY",
		},
		{
			name: "две главные фотографии", err: entity.ErrMachineImagePrimaryConflict,
			expStatus: http.StatusBadRequest, expCode: "MACHINE_IMAGE_PRIMARY_CONFLICT",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{searchErr: c.err}

			w := postSearch(t, stub, `{}`)

			assert.Equal(t, c.expStatus, w.Code)
			assert.Contains(t, w.Body.String(), entity.CodePrefix+c.expCode)

			// Статус берётся из категории доменной ошибки, а не задаётся
			// обработчиком: проверяем, что механизм тот же самый.
			var domainErr *apperrors.DomainError
			require.ErrorAs(t, c.err, &domainErr)
			assert.Equal(t, c.expStatus, domainErr.ErrorType.HTTPStatusCode())
		})
	}
}

// TestCatalogHandler_MalformedBodyIs400 фиксирует отказ на битом JSON.
func TestCatalogHandler_MalformedBodyIs400(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	w := postSearch(t, stub, `{не json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Zero(t, stub.publicCalls)
}

// TestCatalogAdminHandler_SearchIncludesDrafts фиксирует, что админский роутер
// зовёт метод, отдающий черновики.
func TestCatalogAdminHandler_SearchIncludesDrafts(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	r := httptest.NewRequest(http.MethodPost, "/machines/search", strings.NewReader(`{"limit":5}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	v1.NewCatalogAdminHandler(stub).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, stub.adminCalls, "админский роутер зовёт поиск по всему каталогу")
	assert.Zero(t, stub.publicCalls)
}

// TestCatalogAdminHandler_AdminOnlyRoutesRequireAdmin фиксирует разделение прав
// внутри админки: необратимые операции закрыты от менеджера.
func TestCatalogAdminHandler_AdminOnlyRoutesRequireAdmin(t *testing.T) {
	t.Parallel()

	const machineID = "66666666-6666-4666-8666-666666666666"

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "удаление категории", method: http.MethodDelete, path: "/categories/" + machineID},
		{name: "удаление характеристики", method: http.MethodDelete, path: "/attributes/" + machineID},
		{name: "удаление карточки", method: http.MethodDelete, path: "/machines/" + machineID},
		{name: "публикация карточки", method: http.MethodPost, path: "/machines/" + machineID + "/publish"},
		// Кривая остаточной стоимости участвует в расчёте всегда и молча:
		// занижение задирает стоимость владения сразу на всех карточках
		// раздела, и по одной карточке этого не увидеть.
		{name: "список кривых", method: http.MethodGet, path: "/residual-curves"},
		{name: "кривая раздела", method: http.MethodGet, path: "/residual-curves/" + machineID},
		{name: "сохранение кривой", method: http.MethodPut, path: "/residual-curves/" + machineID},
		{name: "удаление кривой", method: http.MethodDelete, path: "/residual-curves/" + machineID},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{}
			router := v1.NewCatalogAdminHandler(stub).Routes()

			// Менеджер: роль есть, но операция необратимая — 403.
			r := httptest.NewRequest(c.method, c.path, nil)
			r = r.WithContext(v1.WithActorForTest(r.Context(),
				entity.NewActor(machineID, entity.ActorTypeManager)))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			assert.Equal(t, http.StatusForbidden, w.Code, "менеджеру необратимые операции закрыты")
			assert.Contains(t, w.Body.String(), entity.CodePrefix+"ROLE_FORBIDDEN")
		})
	}
}

// TestCatalogAdminHandler_ManagerRoutesAreOpenToManager фиксирует обратное:
// обратимые операции менеджеру доступны.
func TestCatalogAdminHandler_ManagerRoutesAreOpenToManager(t *testing.T) {
	t.Parallel()

	const actorID = "66666666-6666-4666-8666-666666666666"

	stub := &stubCatalogService{}

	r := httptest.NewRequest(http.MethodPost, "/machines/search", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(v1.WithActorForTest(r.Context(),
		entity.NewActor(actorID, entity.ActorTypeManager)))

	w := httptest.NewRecorder()
	v1.NewCatalogAdminHandler(stub).Routes().ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "поиск по каталогу доступен менеджеру")
}

// adminRequest выполняет запрос к админскому роутеру от имени администратора.
func adminRequest(t *testing.T, srv *stubCatalogService, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	const adminID = "77777777-7777-4777-8777-777777777777"

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}

	r := httptest.NewRequest(method, path, reader)
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(v1.WithActorForTest(r.Context(), entity.NewActor(adminID, entity.ActorTypeAdmin)))

	w := httptest.NewRecorder()
	v1.NewCatalogAdminHandler(srv).Routes().ServeHTTP(w, r)

	return w
}

// TestCatalogHandler_Categories фиксирует выдачу дерева категорий на витрине.
func TestCatalogHandler_Categories(t *testing.T) {
	t.Parallel()

	parent := "a1"
	stub := &stubCatalogService{categories: entity.Categories{
		{ID: "a1", Slug: "excavators", NameRU: "Экскаваторы", IsActive: true},
		{ID: "a2", ParentID: &parent, Slug: "mini-excavators", NameRU: "Мини", IsActive: true},
	}}

	r := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()
	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Items []struct {
			ID       string `json:"id"`
			ParentID string `json:"parentID"`
			Slug     string `json:"slug"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	require.Len(t, body.Items, 2)
	assert.Empty(t, body.Items[0].ParentID, "корень отдаётся с пустым родителем")
	assert.Equal(t, "a1", body.Items[1].ParentID, "потомок несёт ссылку на родителя")
}

// TestCatalogHandler_CategoryAttributes фиксирует выдачу характеристик с вариантами.
func TestCatalogHandler_CategoryAttributes(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{attributes: entity.Attributes{
		{
			ID: "b1", Code: "drive_type", DataType: entity.AttributeTypeEnum, IsFilterable: true,
			Options: entity.AttributeOptions{{Code: "crawler"}, {Code: "wheeled"}},
		},
		{ID: "b2", Code: "engine_power", DataType: entity.AttributeTypeNumber, Unit: entity.Pointer("кВт")},
	}}

	r := httptest.NewRequest(http.MethodGet, "/categories/a1/attributes", nil)
	w := httptest.NewRecorder()
	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Items []struct {
			Code     string `json:"code"`
			DataType string `json:"dataType"`
			Unit     string `json:"unit"`
			Options  []struct {
				Code string `json:"code"`
			} `json:"options"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	require.Len(t, body.Items, 2)
	assert.Len(t, body.Items[0].Options, 2, "варианты enum уходят вместе с характеристикой")
	assert.Equal(t, "кВт", body.Items[1].Unit)
}

// TestCatalogHandler_CategoriesFailure фиксирует проброс отказа сервиса.
func TestCatalogHandler_CategoriesFailure(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrCategoryNotFound}

	r := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()
	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCatalogAdminHandler_SaveCategory фиксирует создание категории.
func TestCatalogAdminHandler_SaveCategory(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"slug":"excavators","nameRU":"Экскаваторы","nameKK":"Экскаваторлар",
		"nameEN":"Excavators","position":1,"isActive":true}`

	w := adminRequest(t, stub, http.MethodPost, "/categories", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.savedCategory)
	assert.Equal(t, "excavators", stub.savedCategory.Slug)
	assert.Empty(t, stub.savedCategory.ID, "при создании идентификатор пуст")
}

// TestCatalogAdminHandler_SaveCategory_SlugConflictIs409 фиксирует, что занятый
// слаг отдаётся конфликтом, а не внутренней ошибкой.
func TestCatalogAdminHandler_SaveCategory_SlugConflictIs409(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrSlugAlreadyUsed}

	body := `{"slug":"excavators","nameRU":"Э","nameKK":"Э","nameEN":"E"}`

	w := adminRequest(t, stub, http.MethodPost, "/categories", body)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"SLUG_ALREADY_USED")
}

// TestCatalogAdminHandler_SaveCategory_InvalidIs400 фиксирует отказ на неполной форме.
func TestCatalogAdminHandler_SaveCategory_InvalidIs400(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	w := adminRequest(t, stub, http.MethodPost, "/categories", `{"slug":"excavators"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, stub.savedCategory, "до сервиса невалидная форма не доходит")
}

// TestCatalogAdminHandler_UpdateCategory фиксирует, что при PUT идентификатор
// берётся из пути.
func TestCatalogAdminHandler_UpdateCategory(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"slug":"excavators","nameRU":"Э","nameKK":"Э","nameEN":"E"}`

	w := adminRequest(t, stub, http.MethodPut, "/categories/a1", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.savedCategory)
	assert.Equal(t, "a1", stub.savedCategory.ID)
}

// TestCatalogAdminHandler_DeleteCategory фиксирует удаление категории.
func TestCatalogAdminHandler_DeleteCategory(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	w := adminRequest(t, stub, http.MethodDelete, "/categories/a1", "")

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestCatalogAdminHandler_DeleteCategoryWithChildren фиксирует отказ удаления
// категории с содержимым.
func TestCatalogAdminHandler_DeleteCategoryWithChildren(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrCategoryHasChildren}

	w := adminRequest(t, stub, http.MethodDelete, "/categories/a1", "")

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"CATEGORY_HAS_CHILDREN")
}

// TestCatalogAdminHandler_DeleteMachine фиксирует удаление карточки.
func TestCatalogAdminHandler_DeleteMachine(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	w := adminRequest(t, stub, http.MethodDelete, "/machines/m1", "")

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestCatalogAdminHandler_DeleteMachineWithQuotes фиксирует отказ удаления
// карточки, по которой выпущены предложения, и код этого отказа.
//
// 409, а не 400: состояние карточки допустимо, конфликтует с ним запрос.
func TestCatalogAdminHandler_DeleteMachineWithQuotes(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrMachineHasQuotes}

	w := adminRequest(t, stub, http.MethodDelete, "/machines/m1", "")

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"MACHINE_HAS_QUOTES")
}

// TestCatalogAdminHandler_CategoryAttributes фиксирует выдачу характеристик в админке.
func TestCatalogAdminHandler_CategoryAttributes(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{attributes: entity.Attributes{
		{ID: "b1", Code: "engine_power", DataType: entity.AttributeTypeNumber},
	}}

	w := adminRequest(t, stub, http.MethodGet, "/categories/a1/attributes", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "engine_power")
}

// TestCatalogAdminHandler_SaveAttribute фиксирует сохранение характеристики
// вместе с вариантами enum.
func TestCatalogAdminHandler_SaveAttribute(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7","code":"drive_type",
		"nameRU":"Привод","nameKK":"Жетек","nameEN":"Drive","dataType":"enum",
		"isFilterable":true,
		"options":[{"code":"crawler","nameRU":"Гусеничный","nameKK":"Ш","nameEN":"Crawler"}]}`

	w := adminRequest(t, stub, http.MethodPost, "/attributes", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.savedAttribute)
	assert.Equal(t, entity.AttributeTypeEnum, stub.savedAttribute.DataType)
	require.Len(t, stub.savedAttribute.Options, 1)
	assert.Equal(t, "crawler", stub.savedAttribute.Options[0].Code)
}

// TestCatalogAdminHandler_SaveAttribute_UnknownTypeIs400 фиксирует отказ
// на типе вне перечня.
func TestCatalogAdminHandler_SaveAttribute_UnknownTypeIs400(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7","code":"x",
		"nameRU":"X","nameKK":"X","nameEN":"X","dataType":"json"}`

	w := adminRequest(t, stub, http.MethodPost, "/attributes", body)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, stub.savedAttribute)
}

// TestCatalogAdminHandler_DeleteAttribute фиксирует удаление характеристики.
func TestCatalogAdminHandler_DeleteAttribute(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	w := adminRequest(t, stub, http.MethodDelete, "/attributes/b1", "")

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestCatalogAdminHandler_SaveMachine фиксирует сохранение карточки: цена
// приходит целым числом центов, значения характеристик — типизированными.
func TestCatalogAdminHandler_SaveMachine(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7","brand":"XCMG","model":"XE215C",
		"slug":"xcmg-xe215c","nameRU":"X","nameKK":"X","nameEN":"X",
		"priceCnyCents":8950000,"isPricePublic":true,
		"availability":"in_stock","condition":"new","productionYear":2024,
		"attributeValues":[
			{"attributeID":"11111111-1111-4111-8111-111111111111","number":"155.5"},
			{"attributeID":"22222222-2222-4222-8222-222222222222","code":"crawler"},
			{"attributeID":"33333333-3333-4333-8333-333333333333","flag":true}
		],
		"images":[{"path":"a.jpg","isPrimary":true}]}`

	w := adminRequest(t, stub, http.MethodPost, "/machines", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.savedMachine)

	assert.Equal(t, int64(8_950_000), stub.savedMachine.PriceCNYCents, "цена в центах, без разбора строки")
	require.Len(t, stub.savedMachine.AttributeValues, 3)

	number, ok := stub.savedMachine.AttributeValues[0].Value.(entity.NumberValue)
	require.True(t, ok)
	assert.Equal(t, "155.5", number.Number.String(), "дробная часть не теряется")

	enumValue, ok := stub.savedMachine.AttributeValues[1].Value.(entity.EnumValue)
	require.True(t, ok, "код варианта приходит как enum, а не свободная строка")
	assert.Equal(t, "crawler", enumValue.Code)

	require.Len(t, stub.savedMachine.Images, 1)
	assert.True(t, stub.savedMachine.Images[0].IsPrimary)
}

// TestCatalogAdminHandler_SaveMachine_AmbiguousValueIs400 фиксирует отказ
// на значении, заданном сразу несколькими полями.
func TestCatalogAdminHandler_SaveMachine_AmbiguousValueIs400(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{}

	body := `{"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7","brand":"X","model":"X",
		"slug":"x","nameRU":"X","nameKK":"X","nameEN":"X","priceCnyCents":1,
		"availability":"in_stock","condition":"new",
		"attributeValues":[{"attributeID":"11111111-1111-4111-8111-111111111111",
			"number":"1","code":"crawler"}]}`

	w := adminRequest(t, stub, http.MethodPost, "/machines", body)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, stub.savedMachine)
}

// TestCatalogAdminHandler_SaveMachine_SlugConflictIs409 фиксирует конфликт слага.
func TestCatalogAdminHandler_SaveMachine_SlugConflictIs409(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrSlugAlreadyUsed}

	body := `{"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7","brand":"X","model":"X",
		"slug":"x","nameRU":"X","nameKK":"X","nameEN":"X","priceCnyCents":1,
		"availability":"in_stock","condition":"new"}`

	w := adminRequest(t, stub, http.MethodPost, "/machines", body)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestCatalogAdminHandler_MachineBySlugShowsPriceAlways фиксирует, что админка
// видит цену даже когда та скрыта на витрине: редактор обязан видеть то,
// что редактирует.
func TestCatalogAdminHandler_MachineBySlugShowsPriceAlways(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{machine: &entity.Machine{
		ID: "m1", Slug: "x", PriceCNYCents: 8_950_000, IsPricePublic: false,
		Availability: entity.AvailabilityInStock, Condition: entity.ConditionNew,
	}}

	w := adminRequest(t, stub, http.MethodGet, "/machines/x", "")

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	price, hasPrice := body["priceCnyCents"]
	require.True(t, hasPrice, "в админке цена отдаётся всегда")
	assert.InDelta(t, float64(8_950_000), price, 0.5)
}

// TestCatalogAdminHandler_PublishAndUnpublish фиксирует публикацию и снятие.
func TestCatalogAdminHandler_PublishAndUnpublish(t *testing.T) {
	t.Parallel()

	published := time.Now().UTC()

	stub := &stubCatalogService{machine: &entity.Machine{
		ID: "m1", Slug: "x", IsActive: true, PublishedAt: &published,
		Availability: entity.AvailabilityInStock, Condition: entity.ConditionNew,
	}}

	w := adminRequest(t, stub, http.MethodPost, "/machines/m1/publish", "")
	assert.Equal(t, http.StatusOK, w.Code)

	w = adminRequest(t, stub, http.MethodPost, "/machines/m1/unpublish", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestCatalogAdminHandler_Images фиксирует чтение, замену и назначение главного фото.
func TestCatalogAdminHandler_Images(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{images: entity.MachineImages{
		{ID: "i1", Path: "a.jpg", IsPrimary: true},
	}}

	w := adminRequest(t, stub, http.MethodGet, "/machines/m1/images", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "a.jpg")

	w = adminRequest(t, stub, http.MethodPut, "/machines/m1/images",
		`{"images":[{"path":"b.jpg","position":0,"isPrimary":true},{"path":"c.jpg","position":1}]}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, stub.images, 2)

	w = adminRequest(t, stub, http.MethodPost, "/machines/m1/images/i2/primary", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "i2", stub.primaryImageID)
}

// TestCatalogAdminHandler_ReplaceImages_TwoPrimariesIs400 фиксирует отказ
// на двух главных фотографиях: это свойство набора, и проверяется оно на наборе.
func TestCatalogAdminHandler_ReplaceImages_TwoPrimariesIs400(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrMachineImagePrimaryConflict}

	w := adminRequest(t, stub, http.MethodPut, "/machines/m1/images",
		`{"images":[{"path":"a.jpg","isPrimary":true},{"path":"b.jpg","isPrimary":true}]}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"MACHINE_IMAGE_PRIMARY_CONFLICT")
}

// TestCatalogAdminHandler_MalformedBodyIs400 фиксирует отказ на битом JSON
// во всех формах админки.
func TestCatalogAdminHandler_MalformedBodyIs400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
	}{
		{name: "категория", path: "/categories"},
		{name: "характеристика", path: "/attributes"},
		{name: "карточка", path: "/machines"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{}

			w := adminRequest(t, stub, http.MethodPost, c.path, `{не json`)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestCatalogAdminHandler_Categories фиксирует выдачу категорий в админке,
// включая скрытые с сайта.
func TestCatalogAdminHandler_Categories(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{categories: entity.Categories{
		{ID: "a1", Slug: "excavators", IsActive: true},
		{ID: "a2", Slug: "hidden", IsActive: false},
	}}

	w := adminRequest(t, stub, http.MethodGet, "/categories", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "hidden", "админка видит скрытые ветки")
}

// TestCatalogAdminHandler_ServiceFailuresPropagate фиксирует, что отказ сервиса
// на каждом маршруте админки доходит до клиента, а не превращается в 200.
func TestCatalogAdminHandler_ServiceFailuresPropagate(t *testing.T) {
	t.Parallel()

	const validMachine = `{"categoryID":"7c9e6679-7425-40de-944b-e07fc1f90ae7","brand":"X","model":"X",
		"slug":"x","nameRU":"X","nameKK":"X","nameEN":"X","priceCnyCents":1,
		"availability":"in_stock","condition":"new"}`

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "категории", method: http.MethodGet, path: "/categories"},
		{name: "характеристики категории", method: http.MethodGet, path: "/categories/a1/attributes"},
		{name: "удаление характеристики", method: http.MethodDelete, path: "/attributes/b1"},
		{name: "поиск", method: http.MethodPost, path: "/machines/search", body: `{}`},
		{name: "карточка по слагу", method: http.MethodGet, path: "/machines/x"},
		{name: "сохранение карточки", method: http.MethodPost, path: "/machines", body: validMachine},
		{name: "публикация", method: http.MethodPost, path: "/machines/m1/publish"},
		{name: "снятие с публикации", method: http.MethodPost, path: "/machines/m1/unpublish"},
		{name: "фотографии", method: http.MethodGet, path: "/machines/m1/images"},
		{name: "замена фотографий", method: http.MethodPut, path: "/machines/m1/images", body: `{"images":[]}`},
		{name: "главное фото", method: http.MethodPost, path: "/machines/m1/images/i1/primary"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{
				opErr:      entity.ErrMachineNotFound,
				searchErr:  entity.ErrMachineNotFound,
				machineErr: entity.ErrMachineNotFound,
			}

			w := adminRequest(t, stub, c.method, c.path, c.body)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

// TestCatalogHandler_CategoryAttributesFailure фиксирует проброс отказа
// на публичном маршруте характеристик.
func TestCatalogHandler_CategoryAttributesFailure(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrCategoryNotFound}

	r := httptest.NewRequest(http.MethodGet, "/categories/a1/attributes", nil)
	w := httptest.NewRecorder()
	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRoleAnyMiddleware фиксирует общий для нескольких ролей раздел:
// перечисленные роли проходят, остальные — нет, неаутентифицированный — 401.
func TestRoleAnyMiddleware(t *testing.T) {
	t.Parallel()

	const actorID = "88888888-8888-4888-8888-888888888888"

	cases := []struct {
		name      string
		actor     entity.Actor
		expStatus int
		expCode   string
	}{
		{
			name:      "менеджер проходит",
			actor:     entity.NewActor(actorID, entity.ActorTypeManager),
			expStatus: http.StatusOK,
		},
		{
			name:      "администратор проходит",
			actor:     entity.NewActor(actorID, entity.ActorTypeAdmin),
			expStatus: http.StatusOK,
		},
		{
			name:      "системная роль не проходит",
			actor:     entity.NewActor(actorID, entity.ActorTypeSystem),
			expStatus: http.StatusForbidden,
			expCode:   "ROLE_FORBIDDEN",
		},
		{
			name:      "актора нет",
			expStatus: http.StatusUnauthorized,
			expCode:   "AUTH_REQUIRED",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var passed bool

			handler := v1.RoleAnyMiddleware(entity.ActorTypeManager, entity.ActorTypeAdmin)(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					passed = true

					w.WriteHeader(http.StatusOK)
				}))

			r := httptest.NewRequest(http.MethodGet, "/admin/catalog/categories", nil)
			r = r.WithContext(v1.WithActorForTest(r.Context(), c.actor))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			require.Equal(t, c.expStatus, w.Code)

			if c.expStatus != http.StatusOK {
				assert.Contains(t, w.Body.String(), entity.CodePrefix+c.expCode)
				assert.False(t, passed)

				return
			}

			assert.True(t, passed)
		})
	}
}

// stubFXService — заглушка сервиса курсов для витрины.
type stubFXService struct {
	service.FXService // Остальные методы витрине не нужны

	applied entity.AppliedFXRate
	err     error
	calls   int
}

func (s *stubFXService) AppliedRate(
	context.Context, string, string, time.Time,
) (entity.AppliedFXRate, error) {
	s.calls++

	return s.applied, s.err
}

// TestCatalogHandler_RateIsFetchedOncePerResponse — проверка требования
// «курс запрашивается один раз на выдачу, не на карточку».
func TestCatalogHandler_RateIsFetchedOncePerResponse(t *testing.T) {
	t.Parallel()

	fx := &stubFXService{applied: entity.AppliedFXRate{
		Rate:     decimal.RequireFromString("500"),
		RateDate: time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
	}}

	stub := &stubCatalogService{page: entity.MachinePage{Machines: entity.Machines{
		{ID: "m-1", PriceCNYCents: 100, IsPricePublic: true},
		{ID: "m-2", PriceCNYCents: 200, IsPricePublic: true},
		{ID: "m-3", PriceCNYCents: 300, IsPricePublic: true},
	}}}

	r := httptest.NewRequest(http.MethodPost, "/machines/search", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	v1.NewCatalogHandler(stub, fx, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, fx.calls, "курс читается один раз на ответ, а не на карточку")

	var page struct {
		Items []struct {
			PriceKZTCents *int64 `json:"priceKztCents"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page.Items, 3)

	for i, item := range page.Items {
		require.NotNil(t, item.PriceKZTCents, "карточка %d пересчитана", i)
	}
}

// TestCatalogHandler_WithoutRateSearchStillWorks фиксирует главное требование
// пересчёта: отсутствие курса роняет пересчёт, но НЕ каталог.
func TestCatalogHandler_WithoutRateSearchStillWorks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fx   *stubFXService
	}{
		{name: "курс не найден", fx: &stubFXService{err: entity.ErrFXRateNotFound}},
		{name: "хранилище недоступно", fx: &stubFXService{err: entity.ErrDatabaseCommon}},
		{name: "курс пустой", fx: &stubFXService{}},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubCatalogService{page: entity.MachinePage{Machines: entity.Machines{
				{ID: "m-1", PriceCNYCents: 100, IsPricePublic: true},
			}}}

			r := httptest.NewRequest(http.MethodPost, "/machines/search", strings.NewReader(`{}`))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			v1.NewCatalogHandler(stub, c.fx, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

			require.Equal(t, http.StatusOK, w.Code, "каталог обязан открыться без курса")

			var page struct {
				Items []map[string]any `json:"items"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
			require.Len(t, page.Items, 1)

			assert.NotContains(t, page.Items[0], "priceKztCents")
			assert.NotContains(t, page.Items[0], "fxRateDate", "дата без цены бессмысленна")
			assert.Contains(t, page.Items[0], "priceCnyCents", "доллары остаются")
		})
	}
}

// TestCatalogHandler_WithoutFXServiceWorks фиксирует, что витрина поднимается
// и без сервиса курсов вовсе.
func TestCatalogHandler_WithoutFXServiceWorks(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{machine: &entity.Machine{
		ID: "m-1", Slug: "x", PriceCNYCents: 100, IsPricePublic: true,
	}}

	r := httptest.NewRequest(http.MethodGet, "/machines/x", nil)
	w := httptest.NewRecorder()
	v1.NewCatalogHandler(stub, nil, nil, testBaseCurrency, testQuoteCurrency, testUSDCurrency).Routes().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "priceKztCents")
}

// adminCurveRequest прогоняет запрос к справочнику кривых от администратора.
func adminCurveRequest(
	t *testing.T, stub *stubCatalogService, method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	const adminID = "66666666-6666-4666-8666-666666666666"

	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(v1.WithActorForTest(r.Context(),
		entity.NewActor(adminID, entity.ActorTypeAdmin)))

	w := httptest.NewRecorder()
	v1.NewCatalogAdminHandler(stub).Routes().ServeHTTP(w, r)

	return w
}

// TestCatalogAdminHandler_ResidualCurves отдаёт список кривых вместе
// с посчитанной таблицей долей по годам.
//
// Таблица здесь и нужна: три числа кривой не дают понять, сколько машина
// будет стоить через пять лет, а заводят кривую именно ради этого.
func TestCatalogAdminHandler_ResidualCurves(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{curves: []*entity.ResidualValueCurve{{
		CategoryID:       "44444444-4444-4444-8444-444444444444",
		FirstYearDropBps: 2000,
		AnnualDropBps:    1200,
		FloorBps:         2000,
	}}}

	w := adminCurveRequest(t, stub, http.MethodGet, "/residual-curves", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"firstYearDropBps":2000`)
	assert.Contains(t, w.Body.String(), `"preview"`)
	// Пятый год: 8000 * 0.88^4 = 4797.
	assert.Contains(t, w.Body.String(), `{"year":5,"rateBps":4797}`)
}

// TestCatalogAdminHandler_ResidualCurveByCategory отдаёт кривую раздела.
func TestCatalogAdminHandler_ResidualCurveByCategory(t *testing.T) {
	t.Parallel()

	const categoryID = "44444444-4444-4444-8444-444444444444"

	w := adminCurveRequest(t, &stubCatalogService{}, http.MethodGet,
		"/residual-curves/"+categoryID, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), categoryID)
}

// TestCatalogAdminHandler_SaveResidualCurve берёт раздел ИЗ АДРЕСА.
//
// В теле раздела нет вовсе: адрес и есть ключ кривой, и второе место,
// где написано, кому её правят, только породило бы вопрос, что делать
// при расхождении.
func TestCatalogAdminHandler_SaveResidualCurve(t *testing.T) {
	t.Parallel()

	const urlCategory = "44444444-4444-4444-8444-444444444444"

	stub := &stubCatalogService{}
	body := `{"firstYearDropBps":2000,"annualDropBps":1200,"floorBps":2000}`

	w := adminCurveRequest(t, stub, http.MethodPut, "/residual-curves/"+urlCategory, body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.savedCurve)
	assert.Equal(t, urlCategory, stub.savedCurve.CategoryID, "раздел берётся из адреса")
	assert.Equal(t, 1200, stub.savedCurve.AnnualDropBps)
}

// TestCatalogAdminHandler_SaveResidualCurve_Invalid отдаёт отказ домена как 400.
func TestCatalogAdminHandler_SaveResidualCurve_Invalid(t *testing.T) {
	t.Parallel()

	const categoryID = "44444444-4444-4444-8444-444444444444"

	stub := &stubCatalogService{opErr: entity.ErrResidualCurveFloorAboveFirstYear}
	body := `{"firstYearDropBps":2000,"annualDropBps":1200,"floorBps":9000}`

	w := adminCurveRequest(t, stub, http.MethodPut, "/residual-curves/"+categoryID, body)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"RESIDUAL_CURVE_FLOOR_ABOVE_FIRST_YEAR")
}

// TestCatalogAdminHandler_DeleteResidualCurve убирает кривую раздела.
func TestCatalogAdminHandler_DeleteResidualCurve(t *testing.T) {
	t.Parallel()

	const categoryID = "44444444-4444-4444-8444-444444444444"

	w := adminCurveRequest(t, &stubCatalogService{}, http.MethodDelete,
		"/residual-curves/"+categoryID, "")

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestCatalogAdminHandler_DeleteResidualCurve_NotFound отдаёт 404.
func TestCatalogAdminHandler_DeleteResidualCurve_NotFound(t *testing.T) {
	t.Parallel()

	const categoryID = "44444444-4444-4444-8444-444444444444"

	stub := &stubCatalogService{opErr: entity.ErrResidualCurveNotFound}

	w := adminCurveRequest(t, stub, http.MethodDelete, "/residual-curves/"+categoryID, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCatalogAdminHandler_ResidualCurves_Failure отдаёт отказ сервиса.
func TestCatalogAdminHandler_ResidualCurves_Failure(t *testing.T) {
	t.Parallel()

	stub := &stubCatalogService{opErr: entity.ErrCategoryNotFound}

	w := adminCurveRequest(t, stub, http.MethodGet, "/residual-curves", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCatalogAdminHandler_ResidualCurveByCategory_NotFound отдаёт 404,
// когда своей кривой у раздела нет.
//
// Это не поломка: раздел наследует кривую предка, и справочнику нужно
// показать пустую форму, а не чужие числа.
func TestCatalogAdminHandler_ResidualCurveByCategory_NotFound(t *testing.T) {
	t.Parallel()

	const categoryID = "44444444-4444-4444-8444-444444444444"

	stub := &stubCatalogService{opErr: entity.ErrResidualCurveNotFound}

	w := adminCurveRequest(t, stub, http.MethodGet, "/residual-curves/"+categoryID, "")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"RESIDUAL_CURVE_NOT_FOUND")
}

// TestCatalogAdminHandler_SaveResidualCurve_BadJSON не доходит до сервиса.
func TestCatalogAdminHandler_SaveResidualCurve_BadJSON(t *testing.T) {
	t.Parallel()

	const categoryID = "44444444-4444-4444-8444-444444444444"

	stub := &stubCatalogService{}

	w := adminCurveRequest(t, stub, http.MethodPut, "/residual-curves/"+categoryID, "{")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, stub.savedCurve, "испорченное тело до сервиса не доходит")
}

// stubPairFX — заглушка курсов, различающая пары.
//
// Отдельно от stubFXService: там один курс на все пары, и на нём нельзя
// показать главное — что тенге и доллар считаются по РАЗНЫМ курсам от одной
// цены в юанях, а не один из другого.
type stubPairFX struct {
	service.FXService

	rates map[string]entity.AppliedFXRate
	pairs []string
}

func (s *stubPairFX) AppliedRate(
	_ context.Context, base, quote string, _ time.Time,
) (entity.AppliedFXRate, error) {
	pair := base + quote
	s.pairs = append(s.pairs, pair)

	rate, ok := s.rates[pair]
	if !ok {
		return entity.AppliedFXRate{}, entity.ErrFXRateNotFound
	}

	return rate, nil
}

// stubSettings — заглушка настроек показа цены.
type stubSettings struct {
	service.SettingsService

	showUSD bool
}

func (s *stubSettings) ShowUSDPrice(context.Context) bool { return s.showUSD }

// TestCatalogHandler_USDPriceFollowsSetting фиксирует требование A5: доллар —
// НЕОБЯЗАТЕЛЬНОЕ второе отображение, и включает его настройка.
func TestCatalogHandler_USDPriceFollowsSetting(t *testing.T) {
	t.Parallel()

	rateDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		showUSD     bool
		rates       map[string]entity.AppliedFXRate
		wantUSD     *int64
		wantUSDPair bool
	}{
		{
			name:    "выключено — доллара нет и курс за ним не ходят",
			showUSD: false,
			rates: map[string]entity.AppliedFXRate{
				"CNYKZT": {Rate: decimal.RequireFromString("74"), RateDate: rateDate},
				"CNYUSD": {Rate: decimal.RequireFromString("0.14"), RateDate: rateDate},
			},
			wantUSD:     nil,
			wantUSDPair: false,
		},
		{
			name:    "включено — доллар считается по своей паре",
			showUSD: true,
			rates: map[string]entity.AppliedFXRate{
				"CNYKZT": {Rate: decimal.RequireFromString("74"), RateDate: rateDate},
				"CNYUSD": {Rate: decimal.RequireFromString("0.14"), RateDate: rateDate},
			},
			// 100 фыней * 0.14 = 14 центов: цена получена ОТ ЮАНЯ,
			// а не из тенге — иначе к ней прилипла бы наценка пары CNY→KZT.
			wantUSD:     entity.Pointer(int64(14)),
			wantUSDPair: true,
		},
		{
			name:    "включено, но курса нет — карточка без доллара",
			showUSD: true,
			rates: map[string]entity.AppliedFXRate{
				"CNYKZT": {Rate: decimal.RequireFromString("74"), RateDate: rateDate},
			},
			wantUSD:     nil,
			wantUSDPair: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			fx := &stubPairFX{rates: c.rates}
			stub := &stubCatalogService{page: entity.MachinePage{Machines: entity.Machines{
				{ID: "m-1", PriceCNYCents: 100, IsPricePublic: true},
			}}}

			r := httptest.NewRequest(http.MethodPost, "/machines/search", strings.NewReader(`{}`))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			v1.NewCatalogHandler(
				stub, fx, &stubSettings{showUSD: c.showUSD},
				testBaseCurrency, testQuoteCurrency, testUSDCurrency,
			).Routes().ServeHTTP(w, r)

			require.Equal(t, http.StatusOK, w.Code, "каталог обязан открыться в любом случае")

			var page struct {
				Items []struct {
					PriceCNYCents *int64 `json:"priceCnyCents"`
					PriceKZTCents *int64 `json:"priceKztCents"`
					PriceUSDCents *int64 `json:"priceUsdCents"`
				} `json:"items"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
			require.Len(t, page.Items, 1)

			item := page.Items[0]
			require.NotNil(t, item.PriceCNYCents, "юань — единственная хранимая цена, он есть всегда")
			assert.Equal(t, int64(7400), *item.PriceKZTCents, "тенге считается по своей паре")

			if c.wantUSD == nil {
				assert.Nil(t, item.PriceUSDCents)
			} else {
				require.NotNil(t, item.PriceUSDCents)
				assert.Equal(t, *c.wantUSD, *item.PriceUSDCents)
			}

			assert.Equal(t, c.wantUSDPair, slices.Contains(fx.pairs, "CNYUSD"),
				"за курсом доллара ходят только при включённой настройке")
		})
	}
}

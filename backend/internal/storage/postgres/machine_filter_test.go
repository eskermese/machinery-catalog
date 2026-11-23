package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// Идентификаторы характеристик в тестах построителя.
const (
	powerAttrID = "11111111-1111-4111-8111-111111111111"
	driveAttrID = "22222222-2222-4222-8222-222222222222"
	gpsAttrID   = "33333333-3333-4333-8333-333333333333"
	categoryID  = "44444444-4444-4444-8444-444444444444"
	lastID      = "55555555-5555-4555-8555-555555555555"
)

// decimalPtr собирает указатель на десятичное число.
func decimalPtr(t *testing.T, value string) *decimal.Decimal {
	t.Helper()

	parsed, err := decimal.NewFromString(value)
	require.NoError(t, err)

	return &parsed
}

// normalize сжимает пробелы, чтобы сравнивать запрос по смыслу, а не по вёрстке.
func normalize(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

// baseFilter собирает минимальный валидный фильтр.
func baseFilter() *entity.MachineFilter {
	return &entity.MachineFilter{Sort: entity.SortPublishedDesc, Limit: 20}
}

// TestBuildMachineSearchQuery_NoFilters фиксирует запрос без условий:
// предикат обязан остаться валидным, а не выродиться в висящий WHERE.
// priceOrder ключ сортировки по цене в том виде, в каком его пишет
// построитель: скрытая цена превращается в NULL.
const priceOrder = `(CASE WHEN m.is_price_public THEN m.price_cny_cents END)`

func TestBuildMachineSearchQuery_NoFilters(t *testing.T) {
	t.Parallel()

	query, args, err := buildMachineSearchQuery(baseFilter())
	require.NoError(t, err)

	assert.Contains(t, normalize(query), "WHERE TRUE ORDER BY m.published_at DESC NULLS FIRST, m.id DESC LIMIT $1")
	assert.Equal(t, []any{20}, args)
}

// TestBuildMachineSearchQuery_OnlyPublished фиксирует предикат витрины.
func TestBuildMachineSearchQuery_OnlyPublished(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.OnlyPublished = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.Contains(t, normalize(query), "WHERE m.is_active AND m.published_at IS NOT NULL")
	assert.Equal(t, []any{20}, args, "предикат витрины не добавляет аргументов")
}

// TestBuildMachineSearchQuery_MachineColumns фиксирует условия по колонкам карточки:
// текст запроса и порядок аргументов проверяются вместе, потому что плейсхолдер
// без своего аргумента — это либо сдвиг всех последующих, либо ошибка драйвера.
func TestBuildMachineSearchQuery_MachineColumns(t *testing.T) {
	t.Parallel()

	availability := entity.AvailabilityInStock
	condition := entity.ConditionNew

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.PriceMinCents = entity.Pointer(int64(1_000_000))
	filter.PriceMaxCents = entity.Pointer(int64(9_000_000))
	filter.Availability = &availability
	filter.Condition = &condition
	filter.ProductionYearMin = entity.Pointer(2018)
	filter.ProductionYearMax = entity.Pointer(2026)

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "m.category_id = $1")
	assert.Contains(t, normalized, "m.price_cny_cents >= $2")
	assert.Contains(t, normalized, "m.price_cny_cents <= $3")
	assert.Contains(t, normalized, "m.availability = $4")
	assert.Contains(t, normalized, "m.condition = $5")
	assert.Contains(t, normalized, "m.production_year >= $6")
	assert.Contains(t, normalized, "m.production_year <= $7")
	assert.Contains(t, normalized, "LIMIT $8")

	assert.Equal(t, []any{
		categoryID, int64(1_000_000), int64(9_000_000), "in_stock", "new", 2018, 2026, 20,
	}, args)
}

// TestBuildMachineSearchQuery_NumberRange фиксирует EXISTS по числовому диапазону.
func TestBuildMachineSearchQuery_NumberRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		minValue *decimal.Decimal
		maxValue *decimal.Decimal
		expWhere string
		expArgs  []any
	}{
		{
			name:     "обе границы",
			minValue: decimalPtr(t, "100"),
			maxValue: decimalPtr(t, "250.5"),
			expWhere: "EXISTS (SELECT 1 FROM machine_attribute_values v WHERE v.machine_id = m.id " +
				"AND v.attribute_id = $1 AND v.num_value >= $2 AND v.num_value <= $3)",
			expArgs: []any{powerAttrID, "100", "250.5", 20},
		},
		{
			name:     "только нижняя граница",
			minValue: decimalPtr(t, "100"),
			expWhere: "EXISTS (SELECT 1 FROM machine_attribute_values v WHERE v.machine_id = m.id " +
				"AND v.attribute_id = $1 AND v.num_value >= $2)",
			expArgs: []any{powerAttrID, "100", 20},
		},
		{
			name:     "только верхняя граница",
			maxValue: decimalPtr(t, "250.5"),
			expWhere: "EXISTS (SELECT 1 FROM machine_attribute_values v WHERE v.machine_id = m.id " +
				"AND v.attribute_id = $1 AND v.num_value <= $2)",
			expArgs: []any{powerAttrID, "250.5", 20},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.Attributes = []entity.AttributeCondition{
				entity.NewNumberRangeCondition(powerAttrID, c.minValue, c.maxValue),
			}

			query, args, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			assert.Contains(t, normalize(query), c.expWhere)
			assert.Equal(t, c.expArgs, args)
		})
	}
}

// TestBuildMachineSearchQuery_NumberRangeEmptyIsSkipped фиксирует, что диапазон
// без границ не добавляет ни условия, ни аргумента.
func TestBuildMachineSearchQuery_NumberRangeEmptyIsSkipped(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Attributes = []entity.AttributeCondition{
		entity.NewNumberRangeCondition(powerAttrID, nil, nil),
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.NotContains(t, query, "EXISTS")
	assert.Equal(t, []any{20}, args)
}

// TestBuildMachineSearchQuery_Enum фиксирует EXISTS по вариантам.
//
// Коды уходят одним массивом: число выбранных флажков не меняет текст запроса,
// поэтому план разбирается один раз на все комбинации панели фильтров.
func TestBuildMachineSearchQuery_Enum(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		codes   []string
		expArgs []any
	}{
		{
			name:    "один вариант",
			codes:   []string{"crawler"},
			expArgs: []any{driveAttrID, []string{"crawler"}, 20},
		},
		{
			name:    "несколько вариантов не меняют текст запроса",
			codes:   []string{"crawler", "wheeled"},
			expArgs: []any{driveAttrID, []string{"crawler", "wheeled"}, 20},
		},
	}

	const expWhere = "EXISTS (SELECT 1 FROM machine_attribute_values v WHERE v.machine_id = m.id " +
		"AND v.attribute_id = $1 AND v.text_value = ANY($2))"

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.Attributes = []entity.AttributeCondition{
				entity.NewEnumCondition(driveAttrID, c.codes...),
			}

			query, args, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			assert.Contains(t, normalize(query), expWhere)
			assert.Equal(t, c.expArgs, args)
		})
	}
}

// TestBuildMachineSearchQuery_EnumEmptyIsSkipped фиксирует, что пустой набор
// вариантов не сужает выборку до пустой.
func TestBuildMachineSearchQuery_EnumEmptyIsSkipped(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Attributes = []entity.AttributeCondition{entity.NewEnumCondition(driveAttrID)}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.NotContains(t, query, "EXISTS")
	assert.Equal(t, []any{20}, args)
}

// TestBuildMachineSearchQuery_Bool фиксирует EXISTS по флаговой характеристике.
func TestBuildMachineSearchQuery_Bool(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Attributes = []entity.AttributeCondition{entity.NewBoolCondition(gpsAttrID, true)}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.Contains(t, normalize(query),
		"EXISTS (SELECT 1 FROM machine_attribute_values v WHERE v.machine_id = m.id "+
			"AND v.attribute_id = $1 AND v.bool_value = $2)")
	assert.Equal(t, []any{gpsAttrID, true, 20}, args)
}

// TestBuildMachineSearchQuery_Combination фиксирует комбинацию всех видов условий.
//
// Главная проверка: N характеристик дают N независимых EXISTS и ни одного
// JOIN к таблице значений — иначе строки карточки размножились бы и сломали
// и сортировку, и keyset-пагинацию.
func TestBuildMachineSearchQuery_Combination(t *testing.T) {
	t.Parallel()

	availability := entity.AvailabilityInStock

	filter := baseFilter()
	filter.OnlyPublished = true
	filter.CategoryID = entity.Pointer(categoryID)
	filter.PriceMinCents = entity.Pointer(int64(5_000_000))
	filter.Availability = &availability
	filter.Sort = entity.SortPriceAsc
	filter.Attributes = []entity.AttributeCondition{
		entity.NewNumberRangeCondition(powerAttrID, decimalPtr(t, "100"), decimalPtr(t, "250")),
		entity.NewEnumCondition(driveAttrID, "crawler"),
		entity.NewBoolCondition(gpsAttrID, true),
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Equal(t, 3, strings.Count(normalized, "EXISTS ("), "по одному EXISTS на характеристику")
	assert.NotContains(t, normalized, "GROUP BY", "агрегация выдаче не нужна")
	assert.NotContains(t, normalized, "DISTINCT", "строки карточек не размножаются")

	// Соединение в запросе ровно одно — дочитывание главного фото поверх
	// готовой страницы. Отбор смотрится до него: фильтр по характеристикам
	// обязан оставаться полусоединениями EXISTS, иначе строки карточек
	// размножатся по числу значений.
	page := normalized[:strings.Index(normalized, "LEFT JOIN machine_images")]
	assert.NotContains(t, page, "JOIN", "фильтр по характеристикам не превращается в join-ы")
	assert.Equal(t, 1, strings.Count(normalized, "JOIN"), "соединение только за фотографией")

	assert.Contains(t, normalized, "ORDER BY "+priceOrder+" ASC NULLS LAST, m.id ASC")

	assert.Equal(t, []any{
		categoryID, int64(5_000_000), "in_stock", // условия по карточке
		powerAttrID, "100", "250", // числовой диапазон
		driveAttrID, []string{"crawler"}, // вариант
		gpsAttrID, true, // флаг
		20, // limit
	}, args)
}

// TestBuildMachineSearchQuery_Sorting фиксирует закрытый перечень сортировок
// и тай-брейкер по идентификатору в каждой из них.
func TestBuildMachineSearchQuery_Sorting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		sort     entity.MachineSort
		expOrder string
	}{
		{sort: entity.SortPriceAsc, expOrder: "ORDER BY " + priceOrder + " ASC NULLS LAST, m.id ASC"},
		{sort: entity.SortPriceDesc, expOrder: "ORDER BY " + priceOrder + " DESC NULLS LAST, m.id DESC"},
		{sort: entity.SortPublishedDesc, expOrder: "ORDER BY m.published_at DESC NULLS FIRST, m.id DESC"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.sort.String(), func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.Sort = c.sort

			query, _, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			assert.Contains(t, normalize(query), c.expOrder)
		})
	}
}

// TestBuildMachineSearchQuery_UnknownSort фиксирует отказ на сортировке вне перечня.
func TestBuildMachineSearchQuery_UnknownSort(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Sort = "price_asc; DROP TABLE machines"

	_, _, err := buildMachineSearchQuery(filter)

	require.ErrorIs(t, err, entity.ErrCatalogSortInvalid)
}

// TestBuildMachineSearchQuery_Cursor фиксирует keyset-условие для каждой сортировки.
//
// Сравнение кортежем, а не раскрытым OR: кортеж атомарен и позволяет
// планировщику встать на нужную позицию составного индекса.
func TestBuildMachineSearchQuery_Cursor(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		sort     entity.MachineSort
		cursor   entity.MachineCursor
		expWhere string
		expArgs  []any
	}{
		{
			name: "по возрастанию цены",
			sort: entity.SortPriceAsc,
			cursor: entity.MachineCursor{
				Sort: entity.SortPriceAsc, PriceCNYCents: entity.Pointer(int64(7_500_000)), ID: lastID,
			},
			expWhere: "((" + priceOrder + " IS NOT NULL AND (" + priceOrder +
				", m.id) > ($1, $2)) OR " + priceOrder + " IS NULL)",
			expArgs: []any{int64(7_500_000), lastID, 20},
		},
		{
			name: "по убыванию цены",
			sort: entity.SortPriceDesc,
			cursor: entity.MachineCursor{
				Sort: entity.SortPriceDesc, PriceCNYCents: entity.Pointer(int64(7_500_000)), ID: lastID,
			},
			expWhere: "((" + priceOrder + " IS NOT NULL AND (" + priceOrder +
				", m.id) < ($1, $2)) OR " + priceOrder + " IS NULL)",
			expArgs: []any{int64(7_500_000), lastID, 20},
		},
		{
			// Курсор остановился на карточке без цены: хвост уже начался,
			// и продолжение — только оставшийся хвост. Карточки с ценой все
			// показаны на предыдущих страницах.
			name: "по возрастанию цены, курсор на карточке без цены",
			sort: entity.SortPriceAsc,
			cursor: entity.MachineCursor{
				Sort: entity.SortPriceAsc, ID: lastID,
			},
			expWhere: "(" + priceOrder + " IS NULL AND m.id > $1)",
			expArgs:  []any{lastID, 20},
		},
		{
			name: "по убыванию цены, курсор на карточке без цены",
			sort: entity.SortPriceDesc,
			cursor: entity.MachineCursor{
				Sort: entity.SortPriceDesc, ID: lastID,
			},
			expWhere: "(" + priceOrder + " IS NULL AND m.id < $1)",
			expArgs:  []any{lastID, 20},
		},
		{
			name: "по убыванию даты публикации",
			sort: entity.SortPublishedDesc,
			cursor: entity.MachineCursor{
				Sort: entity.SortPublishedDesc, PublishedAt: &publishedAt, ID: lastID,
			},
			expWhere: "(m.published_at IS NOT NULL AND (m.published_at, m.id) < ($1, $2))",
			expArgs:  []any{publishedAt, lastID, 20},
		},
		{
			// Курсор остановился на черновике: даты у него нет. Продолжение —
			// оставшиеся черновики с меньшим id, а за ними все опубликованные.
			name: "по убыванию даты публикации, курсор на черновике",
			sort: entity.SortPublishedDesc,
			cursor: entity.MachineCursor{
				Sort: entity.SortPublishedDesc, ID: lastID,
			},
			expWhere: "((m.published_at IS NULL AND m.id < $1) OR m.published_at IS NOT NULL)",
			expArgs:  []any{lastID, 20},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.Sort = c.sort
			filter.Cursor = &c.cursor

			query, args, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			assert.Contains(t, normalize(query), c.expWhere)
			assert.Equal(t, c.expArgs, args)
		})
	}
}

// TestBuildMachineSearchQuery_PublishedOrderPutsDraftsFirst фиксирует порядок,
// на котором держится условие продолжения выдачи.
//
// NULLS FIRST выписан в запросе явно: это и есть умолчание PostgreSQL для DESC,
// но условие с курсора рассчитано именно на такой порядок, и он не должен
// зависеть от умолчания, которое читателю нужно помнить.
func TestBuildMachineSearchQuery_PublishedOrderPutsDraftsFirst(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Sort = entity.SortPublishedDesc

	query, _, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.Contains(t, normalize(query), "ORDER BY m.published_at DESC NULLS FIRST, m.id DESC")
}

// TestBuildMachineSearchQuery_NoUserValueInSQL — главная проверка безопасности:
// значения, пришедшие снаружи, обязаны оказаться в аргументах, а не в тексте.
func TestBuildMachineSearchQuery_NoUserValueInSQL(t *testing.T) {
	t.Parallel()

	const injection = "'; DROP TABLE machines; --"

	availability := entity.MachineAvailability(injection)
	condition := entity.MachineCondition(injection)

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(injection)
	filter.Availability = &availability
	filter.Condition = &condition
	filter.Attributes = []entity.AttributeCondition{
		entity.NewEnumCondition(injection, injection),
		entity.NewNumberRangeCondition(injection, decimalPtr(t, "1"), nil),
		entity.NewBoolCondition(injection, true),
	}
	filter.Cursor = &entity.MachineCursor{
		Sort: entity.SortPublishedDesc, ID: injection,
		PublishedAt: entity.Pointer(time.Now().UTC()),
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.NotContains(t, query, "DROP TABLE", "пользовательское значение не попадает в текст запроса")
	assert.NotContains(t, query, injection)

	// Каждое значение обязано найтись среди аргументов.
	found := 0

	for _, arg := range args {
		if arg == injection {
			found++
		}
	}

	// Категория, наличие, состояние, три идентификатора характеристик и курсор.
	// Коды варианта уходят массивом, поэтому отдельным аргументом не считаются.
	assert.Equal(t, 7, found, "все значения ушли в аргументы")
}

// TestBuildMachineSearchQuery_PlaceholdersMatchArgs фиксирует соответствие
// числа плейсхолдеров числу аргументов: расхождение здесь — либо сдвиг
// значений по условиям, либо отказ драйвера в рантайме.
func TestBuildMachineSearchQuery_PlaceholdersMatchArgs(t *testing.T) {
	t.Parallel()

	availability := entity.AvailabilityOnOrder

	filter := baseFilter()
	filter.OnlyPublished = true
	filter.CategoryID = entity.Pointer(categoryID)
	filter.PriceMinCents = entity.Pointer(int64(1))
	filter.PriceMaxCents = entity.Pointer(int64(2))
	filter.Availability = &availability
	filter.ProductionYearMin = entity.Pointer(2000)
	filter.Attributes = []entity.AttributeCondition{
		entity.NewNumberRangeCondition(powerAttrID, decimalPtr(t, "1"), decimalPtr(t, "2")),
		entity.NewEnumCondition(driveAttrID, "crawler", "wheeled"),
		entity.NewBoolCondition(gpsAttrID, false),
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	for i := range args {
		assert.Contains(t, query, "$"+itoa(i+1), "плейсхолдер $%d обязан присутствовать в запросе", i+1)
	}

	assert.NotContains(t, query, "$"+itoa(len(args)+1), "лишних плейсхолдеров быть не должно")
}

// itoa переводит небольшое число в строку без импорта strconv в тесте.
func itoa(value int) string {
	if value < 10 {
		return string(rune('0' + value))
	}

	return itoa(value/10) + string(rune('0'+value%10))
}

// TestBuildMachineSearchQuery_CategoryWithoutSubtree фиксирует, что без флага
// поддерева отбор идёт по одному узлу и CTE в запросе не появляется.
func TestBuildMachineSearchQuery_CategoryWithoutSubtree(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.NotContains(t, query, "WITH RECURSIVE")
	assert.Contains(t, normalize(query), "m.category_id = $1")
	assert.Equal(t, []any{categoryID, 20}, args)
}

// TestBuildMachineSearchQuery_CategorySubtree фиксирует обход поддерева:
// клик по родительской категории обязан показывать и дочерние.
func TestBuildMachineSearchQuery_CategorySubtree(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "WITH RECURSIVE subtree(id, depth) AS ( "+
		"SELECT id, 0 FROM categories WHERE id = $1 UNION "+
		"SELECT c.id, s.depth + 1 FROM categories c JOIN subtree s ON c.parent_id = s.id "+
		"WHERE s.depth < $2 )")

	// Условие по категории — равенство, а не вхождение: именно оно позволяет
	// планировщику взять готовый порядок из индекса витрины.
	assert.Contains(t, normalized, "m.category_id = s.id")
	assert.NotContains(t, normalized, "IN (SELECT id FROM subtree)")

	// Обход идёт по уникальным идентификаторам: узел, попавший в CTE дважды
	// с разной глубиной, иначе повторил бы свои карточки в выдаче.
	assert.Contains(t, normalized, "FROM (SELECT DISTINCT id FROM subtree) s")
	assert.Contains(t, normalized, "CROSS JOIN LATERAL")

	// Предел глубины — аргумент, а не число в тексте: он приходит константой
	// пакета, но идёт через тот же путь, что и остальные значения.
	assert.Equal(t, []any{categoryID, maxCategoryDepth, 20}, args)
}

// TestBuildMachineSearchQuery_SubtreeLimitIsSharedByBothLevels фиксирует первое
// условие корректности слияния: внутренний LIMIT не меньше внешнего.
//
// Здесь это буквально один плейсхолдер на оба уровня. Общий top-N в вырожденном
// случае целиком приходит из одной категории, поэтому меньшее число строк
// с категории теряло бы выдачу.
func TestBuildMachineSearchQuery_SubtreeLimitIsSharedByBothLevels(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	// Внутренний LIMIT — в латерали, внешний — после её ORDER BY.
	assert.Contains(t, normalized,
		"ORDER BY m.published_at DESC NULLS FIRST, m.id DESC LIMIT $3 ) m ORDER BY m.published_at DESC NULLS FIRST, m.id DESC LIMIT $3")

	assert.Equal(t, 2, strings.Count(normalized, "LIMIT $3"), "лимит подставлен дважды")
	assert.Equal(t, 2, strings.Count(normalized, "LIMIT $"), "других лимитов в запросе нет")
	assert.Equal(t, []any{categoryID, maxCategoryDepth, 20}, args, "значение лимита в аргументах одно")
}

// TestBuildMachineSearchQuery_SubtreeCursorGoesInsideLateral фиксирует второе
// условие корректности: условие курсора уходит ВНУТРЬ латерали.
//
// Снаружи оно отсекало бы уже собранные top-N каждой категории, и вторая
// страница собралась бы не из тех строк: категория, чьи свежие карточки
// целиком остались до курсора, не отдала бы ничего вместо своих следующих.
func TestBuildMachineSearchQuery_SubtreeCursorGoesInsideLateral(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true
	filter.Cursor = &entity.MachineCursor{
		Sort: entity.SortPublishedDesc, PublishedAt: &publishedAt, ID: lastID,
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)
	cursorCondition := "(m.published_at, m.id) < ($3, $4)"

	require.Contains(t, normalized, cursorCondition)

	// Условие стоит между открытием латерали и её закрывающей скобкой,
	// то есть внутри подзапроса, а не в предикате внешнего запроса.
	lateralStart := strings.Index(normalized, "CROSS JOIN LATERAL (")
	lateralEnd := strings.Index(normalized, ") m ORDER BY")
	cursorAt := strings.Index(normalized, cursorCondition)

	require.Positive(t, lateralStart)
	require.Greater(t, lateralEnd, lateralStart)
	assert.Greater(t, cursorAt, lateralStart, "курсор после открытия латерали")
	assert.Less(t, cursorAt, lateralEnd, "курсор до её закрытия")

	assert.Equal(t, []any{categoryID, maxCategoryDepth, publishedAt, lastID, 20}, args)
}

// TestBuildMachineSearchQuery_SubtreeConditionsGoInsideLateral фиксирует, что
// внутрь латерали уходит ВЕСЬ предикат, а не только курсор: снаружи латерали
// условий нет вовсе, иначе они отсекали бы уже урезанные top-N.
func TestBuildMachineSearchQuery_SubtreeConditionsGoInsideLateral(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true
	filter.OnlyPublished = true
	filter.PriceMinCents = entity.Pointer(int64(5_000_000))
	filter.Attributes = []entity.AttributeCondition{
		entity.NewEnumCondition(driveAttrID, "crawler"),
	}

	query, _, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	lateralStart := strings.Index(normalized, "CROSS JOIN LATERAL (")
	lateralEnd := strings.Index(normalized, ") m ORDER BY")
	require.Positive(t, lateralStart)
	require.Greater(t, lateralEnd, lateralStart)

	// Сравнивается тело латерали целиком, а не позиция подстроки во всём
	// запросе: «m.is_active» встречается и в списке колонок внешнего SELECT,
	// и поиск по всему тексту находил бы колонку вместо условия.
	body := normalized[lateralStart:lateralEnd]

	for _, condition := range []string{
		"WHERE m.category_id = s.id",
		"AND m.is_active AND m.published_at IS NOT NULL",
		"m.price_cny_cents >= $",
		"EXISTS (SELECT 1 FROM machine_attribute_values v",
	} {
		assert.Contains(t, body, condition, "условие внутри латерали")
	}

	// Снаружи латерали — только слияние, общий top-N и дочитывание главного
	// фото. Предиката там нет: он весь ушёл внутрь.
	assert.Equal(t, ") m ORDER BY m.published_at DESC NULLS FIRST, m.id DESC LIMIT $6"+
		") m LEFT JOIN machine_images img ON img.machine_id = m.id AND img.is_primary "+
		"ORDER BY m.published_at DESC NULLS FIRST, m.id DESC",
		normalized[lateralEnd:], "снаружи латерали предиката нет")
}

// TestBuildMachineSearchQuery_SingleCategoryStaysFlat фиксирует, что фильтр
// по одной категории латералью не оборачивается: там план уже оптимален —
// равенство по ведущей колонке индекса витрины, — и лишние узлы не нужны.
func TestBuildMachineSearchQuery_SingleCategoryStaysFlat(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.OnlyPublished = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.NotContains(t, normalized, "LATERAL")
	assert.NotContains(t, normalized, "WITH RECURSIVE")
	assert.Equal(t, "SELECT "+normalize(machineColumns)+", m.relevance, "+normalize(machinePrimaryImageColumns)+
		" FROM (SELECT "+normalize(machineColumns)+", 0 AS relevance FROM machines m "+
		"WHERE m.category_id = $1 AND m.is_active AND m.published_at IS NOT NULL "+
		"ORDER BY m.published_at DESC NULLS FIRST, m.id DESC LIMIT $2) m "+
		"LEFT JOIN machine_images img ON img.machine_id = m.id AND img.is_primary "+
		"ORDER BY m.published_at DESC NULLS FIRST, m.id DESC", normalized)
	assert.Equal(t, []any{categoryID, 20}, args)
}

// TestBuildMachineSearchQuery_PrimaryImageJoinsOutsideThePage фиксирует главное
// свойство дочитывания фотографии: соединение стоит НАД готовой страницей.
//
// Если бы оно оказалось в том же FROM, что и латераль, планировщик был бы
// вправе занести его внутрь — и фотографии дочитывались бы на каждую категорию
// поддерева, а не на каждую строку страницы. ORDER BY с LIMIT внутри подзапроса
// эту перестановку запрещают, поэтому проверяется именно вложенность.
func TestBuildMachineSearchQuery_PrimaryImageJoinsOutsideThePage(t *testing.T) {
	t.Parallel()

	for name, includeSubcategories := range map[string]bool{
		"плоский запрос":  false,
		"обход поддерева": true,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.CategoryID = entity.Pointer(categoryID)
			filter.IncludeSubcategories = includeSubcategories

			query, _, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			normalized := normalize(query)

			join := strings.Index(normalized, "LEFT JOIN machine_images")
			require.Positive(t, join, "фотография дочитывается соединением")

			// Условие соединения повторяет предикат частичного индекса
			// machine_images_primary_uidx: без is_primary соединение вернуло бы
			// все фотографии карточки и размножило бы строку выдачи.
			assert.Contains(t, normalized,
				"LEFT JOIN machine_images img ON img.machine_id = m.id AND img.is_primary")

			// Страница целиком — включая свой LIMIT — лежит в подзапросе
			// ПЕРЕД соединением.
			page := normalized[:join]
			assert.Contains(t, page, "LIMIT $", "LIMIT страницы стоит до соединения")
			assert.Contains(t, page, "FROM (SELECT", "страница обёрнута подзапросом")

			// После соединения — только порядок. Своего LIMIT снаружи нет:
			// соединение один к одному, строк не прибавляется.
			after := normalized[join:]
			assert.NotContains(t, after, "LIMIT", "снаружи соединения лимита нет")
			assert.Contains(t, after, "ORDER BY m.published_at DESC NULLS FIRST, m.id DESC")
		})
	}
}

// TestBuildMachineSearchQuery_SubtreePlaceholdersComeFirst фиксирует порядок
// плейсхолдеров: CTE стоит в начале запроса, поэтому его аргументы обязаны
// быть первыми — иначе драйвер подставит значения не в те места.
func TestBuildMachineSearchQuery_SubtreePlaceholdersComeFirst(t *testing.T) {
	t.Parallel()

	availability := entity.AvailabilityInStock

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true
	filter.PriceMinCents = entity.Pointer(int64(5_000_000))
	filter.Availability = &availability
	filter.Attributes = []entity.AttributeCondition{
		entity.NewEnumCondition(driveAttrID, "crawler"),
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "WHERE id = $1")
	assert.Contains(t, normalized, "WHERE s.depth < $2")
	assert.Contains(t, normalized, "m.price_cny_cents >= $3")
	assert.Contains(t, normalized, "m.availability = $4")
	assert.Contains(t, normalized, "v.attribute_id = $5")

	assert.Equal(t, []any{
		categoryID, maxCategoryDepth, // CTE
		int64(5_000_000), "in_stock", // условия карточки
		driveAttrID, []string{"crawler"}, // характеристика
		20, // limit
	}, args)
}

// TestBuildMachineSearchQuery_SubtreeWithoutCategoryIsSkipped фиксирует, что
// флаг поддерева без самой категории не добавляет ни CTE, ни условия.
func TestBuildMachineSearchQuery_SubtreeWithoutCategoryIsSkipped(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.IncludeSubcategories = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.NotContains(t, query, "WITH RECURSIVE")
	assert.NotContains(t, query, "subtree")
	assert.Equal(t, []any{20}, args)
}

// TestBuildMachineSearchQuery_SubtreeDepthIsBounded фиксирует наличие предела
// глубины: рекурсивный CTE на цикле, попавшем в данные в обход триггера,
// не завершается и съедает память.
func TestBuildMachineSearchQuery_SubtreeDepthIsBounded(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.Contains(t, normalize(query), "WHERE s.depth < $2", "обход ограничен по уровню")
	assert.Contains(t, normalize(query), "UNION SELECT", "UNION, а не UNION ALL: повторы отбрасываются")
	assert.Contains(t, args, maxCategoryDepth)
	assert.Positive(t, maxCategoryDepth)
}

// TestBuildMachineCountQuery фиксирует форму запроса подсчёта: без сортировки,
// без лимита и без курсора — итог считается по всей выдаче, а не по её остатку.
func TestBuildMachineCountQuery(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

	filter := baseFilter()
	filter.OnlyPublished = true
	filter.PriceMinCents = entity.Pointer(int64(5_000_000))
	filter.Cursor = &entity.MachineCursor{
		Sort: entity.SortPublishedDesc, PublishedAt: &publishedAt, ID: lastID,
	}

	query, args, err := buildMachineCountQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Equal(t, "SELECT count(*) FROM machines m "+
		"WHERE m.is_active AND m.published_at IS NOT NULL AND m.price_cny_cents >= $1", normalized)
	assert.NotContains(t, normalized, "ORDER BY")
	assert.NotContains(t, normalized, "LIMIT")
	assert.NotContains(t, normalized, "published_at, m.id) <", "курсор в подсчёт не входит")
	assert.Equal(t, []any{int64(5_000_000)}, args, "ни курсор, ни лимит не попали в аргументы")
}

// TestBuildMachineCountQuery_Subtree фиксирует, что подсчёт по поддереву идёт
// вхождением, а не латералью: латераль существует ради top-N по индексу,
// а подсчёт обязан просмотреть все подходящие строки.
func TestBuildMachineCountQuery_Subtree(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)
	filter.IncludeSubcategories = true

	query, args, err := buildMachineCountQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "WITH RECURSIVE subtree(id, depth)")
	assert.Contains(t, normalized, "m.category_id IN (SELECT id FROM subtree)")
	assert.NotContains(t, normalized, "LATERAL")
	assert.Equal(t, []any{categoryID, maxCategoryDepth}, args)
}

// TestBuildMachineCountQuery_SingleCategory фиксирует подсчёт по одной категории.
func TestBuildMachineCountQuery_SingleCategory(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.CategoryID = entity.Pointer(categoryID)

	query, args, err := buildMachineCountQuery(filter)
	require.NoError(t, err)

	assert.Equal(t, "SELECT count(*) FROM machines m WHERE m.category_id = $1", normalize(query))
	assert.Equal(t, []any{categoryID}, args)
}

// TestBuildMachineCountQuery_BadAttribute фиксирует, что битое условие
// по характеристике отвергается и в подсчёте.
func TestBuildMachineCountQuery_BadAttribute(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Attributes = []entity.AttributeCondition{nil}

	_, _, err := buildMachineCountQuery(filter)

	require.ErrorIs(t, err, entity.ErrAttributeValueInvalid)
}

// TestBuildMachineSearchQuery_Query фиксирует основной путь поиска:
// полнотекстовый по конфигурации russian, с префиксом у каждой лексемы.
//
// Префикс здесь не украшение: без него «xe370» не нашёл бы «XE370G» —
// лексема одна и совпадать должна целиком.
func TestBuildMachineSearchQuery_Query(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Query = "экскаватор"

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "to_tsvector('russian', m.search_text) @@ to_tsquery('russian'")
	assert.Contains(t, normalized, "plainto_tsquery('russian', $1)",
		"запрос собирается из plainto_tsquery: операторы tsquery во вводе не должны становиться синтаксисом")
	assert.Contains(t, normalized, "':*", "к лексемам дописывается префиксный признак")
	// Однословный запрос ищется ещё и подстрокой: род техники в русском
	// стоит в конце сложного слова («автокран» — это кран), и стеммер
	// их не связывает.
	assert.Contains(t, normalized, "m.search_text ILIKE $2")

	// Два аргумента на запрос: текст для tsquery и шаблон подстроки.
	// Ступени релевантности здесь не считаются — порядок задан датой
	// публикации, и их плейсхолдеры стали бы лишними аргументами.
	assert.Equal(t, []any{"экскаватор", "%экскаватор%", 20}, args)
}

// TestBuildMachineSearchQuery_FuzzyQuery фиксирует запасной путь.
//
// Он включается, только когда основной ничего не нашёл, и берёт на себя
// опечатки и склонения, которых не умеет снежковый стеммер, — плюс подстроку,
// которую полнотекстовый поиск не возьмёт никогда («370» внутри «XE370G»
// не префикс).
func TestBuildMachineSearchQuery_FuzzyQuery(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Query = "эскаватор"
	filter.Fuzzy = true

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "word_similarity($1, m.search_text) >= 0.5")
	assert.Contains(t, normalized, "m.search_text ILIKE $2")
	assert.NotContains(t, normalized, "@@", "запасной путь полнотекстовым поиском не пользуется")
	assert.Equal(t, []any{"эскаватор", "%эскаватор%", 20}, args)
}

// TestBuildMachineSearchQuery_QueryEscapesPattern — проверка того, что
// символы шаблона в пользовательском вводе обезврежены.
//
// Проверяется запасной путь: подстрока живёт там. Без экранирования запрос
// «100%» превратился бы в шаблон, находящий всё подряд, а «a_b» совпал бы
// с «axb».
func TestBuildMachineSearchQuery_QueryEscapesPattern(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		exp  string
	}{
		{name: "процент", in: "100%", exp: `%100\%%`},
		{name: "подчёркивание", in: "a_b", exp: `%a\_b%`},
		{name: "обратный слэш", in: `a\b`, exp: `%a\\b%`},
		{name: "всё сразу", in: `50%_\`, exp: `%50\%\_\\%`},
		{name: "обычный текст не трогается", in: "экскаватор", exp: "%экскаватор%"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.Query = c.in
			filter.Fuzzy = true

			_, args, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			// Первый аргумент — сырой текст для похожести, второй —
			// экранированный шаблон подстроки.
			assert.Equal(t, c.exp, args[1])
		})
	}
}

// TestBuildMachineSearchQuery_HomoglyphVariant фиксирует второй вариант
// написания: обозначения латинские, а раскладка у человека русская.
func TestBuildMachineSearchQuery_HomoglyphVariant(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Query = "ХЕ370"

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	assert.Contains(t, normalize(query), "OR (to_tsvector", "оба написания ищутся вместе")
	assert.Contains(t, args, "ХЕ370")
	assert.Contains(t, args, "XE370", "латинский вариант уходит в запрос вторым")
}

// TestBuildMachineSearchQuery_RelevanceOrder фиксирует порядок ранжирования
// и то, что снаружи сортировка идёт по КОЛОНКЕ.
//
// Выражение ссылается на search_text, которой во внешней обёртке уже нет:
// там доступны только выбранные колонки, и сортировка выражением там упала бы.
func TestBuildMachineSearchQuery_RelevanceOrder(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Query = "XE370G"
	filter.Sort = entity.SortRelevance

	query, _, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "AS relevance")
	assert.Contains(t, normalized, "ORDER BY m.relevance DESC, m.published_at DESC NULLS FIRST, m.id DESC")
	assert.Contains(t, normalized, "THEN 5", "точная модель — верхняя ступень")
	assert.Contains(t, normalized, "lower(m.brand) = lower(", "бренд сужает мало и стоит низко")
}

// TestBuildMachineSearchQuery_RelevanceWithoutQuery отвергает порядок
// по релевантности без запроса: считать её не из чего.
func TestBuildMachineSearchQuery_RelevanceWithoutQuery(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.Sort = entity.SortRelevance

	_, _, err := buildMachineSearchQuery(filter)

	require.ErrorIs(t, err, entity.ErrCatalogCursorInvalid)
}

// TestBuildMachineSearchQuery_RelevanceCursor фиксирует продолжение
// ранжированной выдачи: три ключа, и хвост тот же, что у сортировки по дате.
func TestBuildMachineSearchQuery_RelevanceCursor(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)

	filter := baseFilter()
	filter.Query = "экскаватор"
	filter.Sort = entity.SortRelevance
	filter.Cursor = &entity.MachineCursor{
		Sort:        entity.SortRelevance,
		Relevance:   entity.Pointer(3087),
		PublishedAt: &published,
		ID:          "66666666-6666-4666-8666-666666666666",
	}

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "m.published_at IS NOT NULL AND (m.published_at, m.id) <",
		"внутри одной релевантности продолжение идёт тем же ключом, что и по дате")
	assert.Contains(t, args, 3087)
}

// TestBuildMachineSearchQuery_EmptyQueryIsSkipped фиксирует, что пустой поиск
// не добавляет условия: иначе каждая страница каталога платила бы за ILIKE
// по пяти колонкам без причины.
func TestBuildMachineSearchQuery_EmptyQueryIsSkipped(t *testing.T) {
	t.Parallel()

	query, args, err := buildMachineSearchQuery(baseFilter())
	require.NoError(t, err)

	assert.NotContains(t, query, "ILIKE")
	assert.Equal(t, []any{20}, args)
}

// TestBuildMachineSearchQuery_IsActive фиксирует отбор по признаку публикации
// в административном чтении.
func TestBuildMachineSearchQuery_IsActive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		flag bool
	}{
		{name: "только опубликованные", flag: true},
		{name: "только черновики", flag: false},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := baseFilter()
			filter.IsActive = entity.Pointer(c.flag)

			query, args, err := buildMachineSearchQuery(filter)
			require.NoError(t, err)

			assert.Contains(t, normalize(query), "m.is_active = $1")
			assert.Equal(t, []any{c.flag, 20}, args)
		})
	}
}

// TestBuildMachineSearchQuery_IsActiveIgnoredOnShowcase — проверка того, что
// витрина не отдаёт черновики параметром запроса.
//
// На публичном чтении признак уже наложен предикатом витрины, и позволить
// клиенту переопределить его значило бы отдать неопубликованные карточки.
func TestBuildMachineSearchQuery_IsActiveIgnoredOnShowcase(t *testing.T) {
	t.Parallel()

	filter := baseFilter()
	filter.OnlyPublished = true
	filter.IsActive = entity.Pointer(false)

	query, args, err := buildMachineSearchQuery(filter)
	require.NoError(t, err)

	normalized := normalize(query)

	assert.Contains(t, normalized, "m.is_active AND m.published_at IS NOT NULL")
	assert.NotContains(t, normalized, "m.is_active = $", "переопределение витрины игнорируется")
	assert.Equal(t, []any{20}, args, "значение клиента даже не уходит аргументом")
}

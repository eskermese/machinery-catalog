package postgres

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// machineColumns список колонок карточки в порядке выборки.
//
// Перечислены явно, а не звёздочкой: порядок колонок здесь — часть контракта
// со сканером, и добавление колонки в таблицу не должно молча сдвигать разбор.
const machineColumns = `m.id, m.category_id, m.brand, m.model, m.slug,
	m.name_ru, m.name_kk, m.name_en,
	m.description_ru, m.description_kk, m.description_en,
	m.price_cny_cents, m.is_price_public, m.availability, m.delivery_weeks,
	m.production_year, m.condition,
	m.fuel_consumption_lph, m.service_interval_hours, m.service_cost_cny_cents,
	m.is_active, m.published_at,
	m.created_at, m.updated_at`

// machinePrimaryImageColumns колонки главного фото в выдаче поиска.
//
// Соединение необязательное, поэтому все колонки читаются как NULL-совместимые:
// у карточки может не быть ни одной фотографии.
const machinePrimaryImageColumns = `img.id, img.path, img.position,
	img.alt_ru, img.alt_kk, img.alt_en`

// machinePrimaryImageWrap дочитывание главного фото к готовой странице выдачи.
//
// Соединение, а не отдельный запрос на карточку: страница в двадцать карточек
// иначе стоила бы двадцать обращений к базе. Условие точно повторяет предикат
// частичного уникального индекса machine_images_primary_uidx (machine_id)
// WHERE is_primary — благодаря ему соединение читает не больше одной строки
// на карточку по индексу, и планировщику не нужно ни сортировать, ни
// группировать выдачу, чтобы выбрать из нескольких фотографий одну.
//
// Именно уникальность делает соединение безопасным для порядка: строк
// не становится больше. Обычный LEFT JOIN по machine_id размножил бы карточку
// по числу её фотографий и потребовал бы DISTINCT поверх выдачи, что сломало бы
// и keyset-пагинацию, и план.
//
// Запрос страницы обёрнут подзапросом, а не дополнен соединением на месте,
// и это не оформление. Соединение, дописанное в тот же FROM, планировщик
// волен переставить ВНУТРЬ латерали — и тогда фотографии дочитываются
// на каждую категорию поддерева, то есть «число категорий × LIMIT» раз вместо
// LIMIT. ORDER BY с LIMIT внутри подзапроса такую перестановку запрещают:
// соединение видит ровно страницу. На 30 000 карточек это 20 обращений
// к machine_images против 100 и 184 страницы буфера против 424.
//
// Внешний ORDER BY повторяется намеренно: соединение не обязано сохранять
// порядок подзапроса, и без него выдача была бы упорядочена случайно.
// Внешний LIMIT не нужен — соединение один к одному, строк не прибавляется.
const machinePrimaryImageWrap = `SELECT %s, %s FROM (%s) m
	LEFT JOIN machine_images img ON img.machine_id = m.id AND img.is_primary
	ORDER BY %s`

// maxCategoryDepth предел глубины обхода дерева категорий.
//
// Защита от неограниченной рекурсии: цикл в дереве отвергает триггер
// categories_reject_cycle_trg, но строка может попасть в данные в обход
// него — восстановлением дампа, отключённым триггером при массовой заливке
// или прямым SQL при разборе инцидента. Рекурсивный CTE на цикле не
// завершается и съедает память, поэтому обход ограничен по уровню, а не
// полагается на чистоту данных. Шестнадцать уровней заведомо перекрывают
// каталог спецтехники, где вложенность — это «тип → подтип».
const maxCategoryDepth = 16

// categorySubtreeCTE рекурсивный обход поддерева категорий.
//
// Собирает идентификаторы узла и всех его потомков. UNION, а не UNION ALL:
// он отбрасывает уже встреченные строки и потому останавливает обход, даже
// если цикл всё-таки попал в данные; ограничение по глубине страхует случай,
// когда цикл проходит через узлы, ещё не попавшие в результат.
const categorySubtreeCTE = `WITH RECURSIVE subtree(id, depth) AS (
		SELECT id, 0 FROM %s WHERE id = %s
		UNION
		SELECT c.id, s.depth + 1 FROM %s c JOIN subtree s ON c.parent_id = s.id
		WHERE s.depth < %s
	)`

// machineSearchLateral шаблон поиска с обходом поддерева категорий.
//
// Форма выбрана из-за индекса витрины machines_public_catalog_idx —
// (category_id, published_at DESC, id DESC). Готовый порядок он даёт только
// ВНУТРИ одной категории: при равенстве по ней планировщик заходит в индекс
// один раз и читает ровно LIMIT строк. Отбор через IN (SELECT id FROM subtree)
// лишает ведущую колонку константы — нужен сквозной порядок по всем категориям
// поддерева, а его такой индекс не даёт, и планировщик читает таблицу целиком
// и сортирует. На 30 000 карточках это Seq Scan по 30 000 строк вместо
// 400 строк по индексу.
//
// Латераль возвращает порядок к форме «равенство по категории»: для каждой
// категории берётся её собственный top-N по индексу, затем результаты
// сливаются и берётся общий top-N. Сортируется число категорий × LIMIT строк,
// и от общего объёма каталога это не зависит.
//
// Два условия корректности, оба держатся конструкцией запроса:
//
//  1. внутренний LIMIT не меньше внешнего — здесь это буквально один и тот же
//     плейсхолдер: общий top-N в вырожденном случае целиком приходит из одной
//     категории, поэтому меньшее число строк с категории теряло бы выдачу;
//  2. условие курсора уходит ВНУТРЬ латерали вместе с остальным предикатом —
//     снаружи оно отсекало бы уже собранные top-N, и вторая страница
//     собралась бы не из тех строк.
//
// DISTINCT по идентификатору категории обязателен: UNION в CTE различает
// строки по паре (id, depth), и узел, до которого при испорченных данных есть
// два пути разной длины, попал бы в обход дважды — а латераль, в отличие
// от IN, повторила бы его карточки в выдаче.
const machineSearchLateral = `%sSELECT %s
	FROM (SELECT DISTINCT id FROM subtree) s
	CROSS JOIN LATERAL (
		SELECT %s FROM %s m WHERE %s ORDER BY %s LIMIT %s
	) m
	ORDER BY %s LIMIT %s`

// machinePriceOrder значение, по которому идёт сортировка по цене.
//
// Скрытая цена превращается в NULL, и это решает сразу две задачи.
//
// Первая — порядок. Раньше скрытая цена хранилась нулём и участвовала
// в сортировке как ноль: «сначала дешёвые» начинались с «Цены по запросу»
// вместо самого дешёвого крана. Отсутствие цены — не ноль и не бесконечность,
// а отдельное состояние, и NULL выражает его прямо. Тот же приём уже применён
// к пустой дате публикации — второй способ для той же задачи заводить незачем.
//
// Вторая — утечка. Админка хранит цену и у скрытых карточек: редактор обязан
// видеть то, что редактирует. Сортировка по сырой колонке расставляла бы такие
// карточки по их тайной цене, и подбором границ фильтра её можно было бы
// восстановить с витрины. Через NULL сортировать нечего.
//
// Выражение выписано ровно так же, как в индексах machines_public_price_idx
// и machines_public_price_desc_idx: планировщик сопоставляет выражение
// с индексом текстуально, и любое расхождение молча лишает выдачу
// готового порядка.
const machinePriceOrder = `(CASE WHEN m.is_price_public THEN m.price_cny_cents END)`

// Выражения сортировки. Закрытый перечень: порядок приходит перечислимым
// типом домена, и в SQL попадает только константа отсюда — ни одна часть
// ORDER BY не выводится из входных данных.
//
// Тай-брейкером всюду идёт m.id: ни цена, ни дата публикации не уникальны,
// а keyset-пагинация требует строгого полного порядка. Без него две машины
// с одинаковой ценой на границе страниц дали бы пропуск одной и повтор другой.
var machineOrderExpressions = map[entity.MachineSort]string{
	// NULLS LAST в обе стороны: карточка без цены не дешевле и не дороже
	// остальных, её цены просто нет, и место ей в конце и при возрастании,
	// и при убывании.
	entity.SortPriceAsc:  machinePriceOrder + " ASC NULLS LAST, m.id ASC",
	entity.SortPriceDesc: machinePriceOrder + " DESC NULLS LAST, m.id DESC",
	// NULLS FIRST выписан явно, хотя для DESC это и есть умолчание PostgreSQL:
	// на этом порядке держится условие продолжения выдачи с курсора, и он
	// не должен зависеть от умолчания, которое читателю нужно помнить.
	// Индексу это ничего не меняет: DESC в объявлении индекса означает то же.
	entity.SortPublishedDesc: "m.published_at DESC NULLS FIRST, m.id DESC",
	// По релевантности — и дальше тем же ключом, что и по умолчанию.
	// Совпадение релевантности у 106 карточек по запросу «XCMG» — обычное
	// дело, и внутри ступени свежие должны идти первыми, а не в порядке
	// идентификаторов.
	//
	// Сортировка идёт по КОЛОНКЕ relevance, а не по выражению: выражение
	// ссылается на search_text, которой во внешней обёртке уже нет —
	// там доступны только выбранные колонки.
	entity.SortRelevance: "m.relevance DESC, m.published_at DESC NULLS FIRST, m.id DESC",
}

// buildMachineSearchQuery собирает запрос поиска по каталогу.
//
// Возвращает текст запроса и аргументы отдельно — благодаря этому построитель
// проверяется тестами без базы, а пользовательские значения физически не могут
// попасть в текст: в него уходят только номера плейсхолдеров.
func buildMachineSearchQuery(filter *entity.MachineFilter) (query string, args []any, err error) {
	order, ok := machineOrderExpressions[filter.Sort]
	if !ok {
		return "", nil, entity.ErrCatalogSortInvalid
	}

	builder := newConditionBuilder()

	// Обход поддерева регистрируется первым: его плейсхолдеры обязаны идти
	// раньше остальных, потому что CTE стоит в начале запроса.
	prefix := appendCategorySubtree(builder, filter)

	search := appendMachineConditions(builder, filter)

	// Порядок по релевантности без запроса не определён: считать её не из
	// чего. Курсор с таким порядком и пустым запросом — испорченный курсор,
	// а не законная страница.
	if filter.Sort == entity.SortRelevance && len(search.ranks) == 0 {
		return "", nil, entity.ErrCatalogCursorInvalid
	}

	// Релевантность строится ТОЛЬКО здесь и только когда по ней сортируют:
	// её плейсхолдеры в подсчёте строк не нужны и оказались бы лишними
	// аргументами запроса.
	var relevance string
	if filter.Sort == entity.SortRelevance {
		relevance = machineRelevanceExpression(builder, search.query, search.ranks)
	}

	if attrErr := appendAttributeConditions(builder, filter.Attributes); attrErr != nil {
		return "", nil, attrErr
	}

	if cursorErr := appendCursorCondition(builder, filter, relevance); cursorErr != nil {
		return "", nil, cursorErr
	}

	// Лимит регистрируется один раз и подставляется дважды: внутрь латерали
	// и наружу. Один плейсхолдер вместо двух делает равенство лимитов
	// свойством запроса, а не соглашением, которое можно нарушить правкой.
	limit := builder.placeholder(filter.Limit)

	// Релевантность выбирается всегда — нулём, когда её не считали.
	// Постоянный список колонок держит один сканер на всю выдачу поиска:
	// иначе их стало бы два, и они разошлись бы при первой правке.
	columns := machineColumns + ", " + relevanceColumn(relevance)
	// Во внутреннем запросе сортируем выражением, снаружи — уже колонкой.
	innerOrder := strings.Replace(order, "m.relevance", "("+relevance+")", 1)

	if !usesCategorySubtree(filter) {
		page := fmt.Sprintf(`SELECT %s FROM %s m WHERE %s ORDER BY %s LIMIT %s`,
			columns, machineTable, builder.where(), innerOrder, limit)

		return withPrimaryImage(page, order), builder.args, nil
	}

	page := fmt.Sprintf(machineSearchLateral,
		prefix, machineColumns+", m.relevance",
		columns, machineTable, builder.where(), innerOrder, limit,
		order, limit)

	return withPrimaryImage(page, order), builder.args, nil
}

// relevanceColumn отдаёт выражение релевантности под именем колонки.
//
// Ноль, когда ранжировать нечем: колонка обязана быть в выдаче всегда,
// иначе сканер выдачи поиска пришлось бы держать в двух вариантах.
func relevanceColumn(expression string) string {
	if expression == "" {
		return "0 AS relevance"
	}

	return "(" + expression + ") AS relevance"
}

// withPrimaryImage оборачивает запрос страницы дочитыванием главного фото.
//
// Отдельная функция, а не две подстановки на месте: обе формы поиска — плоская
// и с обходом поддерева — дочитывают фото одинаково, и это должно оставаться
// одним местом, а не совпадением двух.
func withPrimaryImage(page, order string) string {
	return fmt.Sprintf(machinePrimaryImageWrap,
		machineColumns+", m.relevance", machinePrimaryImageColumns, page, order)
}

// buildMachineCountQuery собирает запрос подсчёта найденных карточек.
//
// Отличается от поиска тремя вещами, и каждая существенна:
//
//  1. нет курсора — итог считается по всей выдаче, а не по её остатку:
//     на второй странице клиент ждёт того же числа, что и на первой;
//  2. нет сортировки и лимита — считать порядок незачем;
//  3. поддерево отбирается вхождением, а не латералью. Латераль существует
//     ради того, чтобы взять top-N по индексу, а подсчёт обязан просмотреть
//     все подходящие строки — брать их по одной категории за раз здесь
//     не выигрыш, а лишние узлы плана.
func buildMachineCountQuery(filter *entity.MachineFilter) (query string, args []any, err error) {
	builder := newConditionBuilder()

	prefix := appendCategorySubtreeForCount(builder, filter)

	// Релевантность подсчёту не нужна: считается число строк, а не их
	// порядок. Условие отбора при этом ровно то же, что и у выдачи, —
	// его собирает та же функция.
	appendMachineConditions(builder, filter)

	if attrErr := appendAttributeConditions(builder, filter.Attributes); attrErr != nil {
		return "", nil, attrErr
	}

	return fmt.Sprintf(`%sSELECT count(*) FROM %s m WHERE %s`,
		prefix, machineTable, builder.where()), builder.args, nil
}

// appendCategorySubtreeForCount добавляет условие по категории для подсчёта.
func appendCategorySubtreeForCount(b *conditionBuilder, filter *entity.MachineFilter) string {
	if filter.CategoryID == nil {
		return ""
	}

	if !filter.IncludeSubcategories {
		b.addf("m.category_id = %s", *filter.CategoryID)

		return ""
	}

	prefix := fmt.Sprintf(categorySubtreeCTE,
		categoryTable, b.placeholder(*filter.CategoryID),
		categoryTable, b.placeholder(maxCategoryDepth))

	b.add("m.category_id IN (SELECT id FROM subtree)")

	return prefix + " "
}

// usesCategorySubtree сообщает, что запрос обходит поддерево категорий.
//
// Один флаг решает форму запроса целиком, поэтому вынесен отдельно: фильтр
// по одной категории остаётся прежним плоским запросом — там план уже
// оптимален, и латераль вокруг единственной категории только добавила бы
// узлов без выигрыша.
func usesCategorySubtree(filter *entity.MachineFilter) bool {
	return filter.CategoryID != nil && filter.IncludeSubcategories
}

// appendMachineConditions добавляет условия по колонкам самой карточки.
func appendMachineConditions(
	b *conditionBuilder,
	filter *entity.MachineFilter,
) (search machineSearchCondition) {
	// Публичная витрина требует обоих признаков сразу. Проверка published_at
	// здесь не избыточна при наличии CHECK-а в схеме: она делает предикат
	// пригодным для частичных индексов витрины и не полагается на инвариант,
	// который в будущем может ослабнуть.
	if filter.OnlyPublished {
		b.add("m.is_active")
		b.add("m.published_at IS NOT NULL")
	}

	// Условие по категории добавляет appendCategorySubtree: форма условия
	// зависит от того, обходится поддерево или нет.

	appendActiveCondition(b, filter)

	search = appendQueryCondition(b, filter)

	if filter.PriceMinCents != nil {
		b.addf("m.price_cny_cents >= %s", *filter.PriceMinCents)
	}

	if filter.PriceMaxCents != nil {
		b.addf("m.price_cny_cents <= %s", *filter.PriceMaxCents)
	}

	if filter.Availability != nil {
		b.addf("m.availability = %s", filter.Availability.String())
	}

	if filter.Condition != nil {
		b.addf("m.condition = %s", filter.Condition.String())
	}

	if filter.ProductionYearMin != nil {
		b.addf("m.production_year >= %s", *filter.ProductionYearMin)
	}

	if filter.ProductionYearMax != nil {
		b.addf("m.production_year <= %s", *filter.ProductionYearMax)
	}

	return search
}

// appendActiveCondition добавляет отбор по признаку публикации.
//
// Игнорируется на публичной витрине: там is_active уже наложен
// предикатом OnlyPublished, и позволить клиенту переопределить его значило бы
// отдать черновики параметром запроса. Молча, а не отказом: клиент публичного
// поиска не обязан знать про существование черновиков вовсе.
func appendActiveCondition(b *conditionBuilder, filter *entity.MachineFilter) {
	if filter.IsActive == nil || filter.OnlyPublished {
		return
	}

	b.addf("m.is_active = %s", *filter.IsActive)
}

// machineSearchText колонка, по которой идёт поиск.
//
// Генерируемая колонка, а не склейка на месте: по ней построены оба индекса,
// и повторять выражение в запросе значило бы держать третье место, которое
// разойдётся с ними.
const machineSearchText = "m.search_text"

// Дописывание префиксного признака к каждой лексеме готового tsquery.
//
// plainto_tsquery отдаёт текст вида 'гусеничн' & 'кран'. Замена ставит «:*»
// после закрывающей кавычки каждой лексемы — перед пробелом или в конце
// строки, — и запрос становится префиксным.
//
// Литералы вынесены в константы, потому что в них по три уровня экранирования
// сразу: кавычка SQL удваивается, обратный слэш принадлежит регулярному
// выражению, и разбирать это внутри Sprintf невозможно.
const (
	// prefixPattern SQL-литерал '''(\s|$)': закрывающая кавычка лексемы,
	// за которой пробел или конец строки.
	prefixPattern = `'''(\s|$)'`
	// prefixReplacement SQL-литерал ''':*\1': та же кавычка, префиксный
	// признак и сохранённый разделитель.
	prefixReplacement = `''':*\1'`
)

// fuzzySimilarityThreshold порог похожести для запасного пути.
//
// Подобран замером на живом каталоге. Ниже 0,5 запрос из двух слов начинает
// тащить лишнее («гусеничный кран» давал 73 карточки вместо 19), выше 0,6 —
// перестаёт находить то, ради чего путь и заведён («катки» теряет «каток»).
const fuzzySimilarityThreshold = 0.5

// machineSearchCondition условие отбора по тексту запроса и выражение
// релевантности к нему.
type machineSearchCondition struct {
	// match предикат отбора; пусто — запроса нет и отбирать нечего.
	match string
	// query первое написание запроса: по нему считаются ступени релевантности.
	query string
	// ranks выражения точности совпадения по каждому написанию запроса.
	//
	// Заготовки, а не готовая релевантность: подсчёт строк её не заказывает,
	// а зарегистрированные плейсхолдеры ушли бы в аргументы запроса, где
	// их некому подставить, — и count падал бы на «expected 1 arguments,
	// got 3».
	ranks []string
}

// appendQueryCondition добавляет поиск по тексту запроса.
//
// Два пути, и переключает между ними filter.Fuzzy.
//
// ОСНОВНОЙ — полнотекстовый поиск по конфигурации russian. Он закрывает
// склонения, число и порядок слов: «погрузчики» находит «погрузчик»,
// а «кран гусеничный» — то же, что «гусеничный кран». Обозначения моделей
// он не ломает: парсер видит «XE370G» как numword и кладёт лексему целиком,
// а «XC8-S2570» — и целиком, и по частям.
//
// К каждой лексеме дописывается префиксный признак. Без него «xe370»
// не нашёл бы «XE370G»: лексема одна и совпадать должна целиком. С префиксом
// поиск ведёт себя так, как от него и ждут при наборе, а на полных словах
// ничего не меняет — замерено, все прочие запросы дали ровно те же числа.
//
// ЗАПАСНОЙ — триграммы. Включается, только когда основной не нашёл ничего,
// и берёт на себя то, чего снежковый стеммер не умеет: опечатки
// («эскаватор») и беглую гласную («катки» против «каток»; полноценный
// ispell-словарь это чинил бы, но его в образе нет — только образцы).
// Тем же условием идёт подстрока: «370» внутри «XE370G» не префикс,
// и полнотекстовый поиск его не возьмёт никогда.
//
// Постоянно включать запасной путь нельзя: на «гусеничный кран» триграммы
// дают 73 карточки вместо 19, и точная выдача утонула бы в похожей.
func appendQueryCondition(b *conditionBuilder, filter *entity.MachineFilter) machineSearchCondition {
	variants := entity.SearchQueryVariants(filter.Query)
	if len(variants) == 0 {
		return machineSearchCondition{}
	}

	matches := make([]string, 0, len(variants))
	ranks := make([]string, 0, len(variants))

	for _, variant := range variants {
		match, rank := searchVariantCondition(b, variant, filter.Fuzzy)

		matches = append(matches, match)
		ranks = append(ranks, rank)
	}

	condition := machineSearchCondition{
		match: "(" + strings.Join(matches, " OR ") + ")",
		query: variants[0],
		ranks: ranks,
	}

	b.add(condition.match)

	return condition
}

// searchVariantCondition собирает предикат и выражение точности для одного
// написания запроса.
func searchVariantCondition(b *conditionBuilder, query string, fuzzy bool) (match, rank string) {
	if fuzzy {
		similar := b.placeholder(query)
		pattern := b.placeholder("%" + escapeLikePattern(query) + "%")

		match = fmt.Sprintf("(word_similarity(%[1]s, %[3]s) >= %[4]v OR %[3]s ILIKE %[2]s)",
			similar, pattern, machineSearchText, fuzzySimilarityThreshold)
		rank = fmt.Sprintf("word_similarity(%s, %s)", similar, machineSearchText)

		return match, rank
	}

	// Запрос собирается из plainto_tsquery, а не из текста напрямую:
	// операторы tsquery в пользовательском вводе иначе означали бы синтаксис,
	// и «(» стал бы отказом разбора. plainto_tsquery видит во вводе только
	// слова.
	//
	// Префикс дописывается правкой готового текста запроса: своей функции
	// «plainto с префиксом» в PostgreSQL нет, а собирать tsquery из строк
	// на стороне Go значило бы вернуть тот самый разбор пользовательского
	// текста, от которого plainto_tsquery и избавляет.
	term := b.placeholder(query)
	tsquery := fmt.Sprintf(
		"to_tsquery('russian', regexp_replace(plainto_tsquery('russian', %s)::text, %s, %s, 'g'))",
		term, prefixPattern, prefixReplacement)

	match = fmt.Sprintf("to_tsvector('russian', %s) @@ %s", machineSearchText, tsquery)
	rank = fmt.Sprintf("ts_rank(to_tsvector('russian', %s), %s)", machineSearchText, tsquery)

	// Однословный запрос ищется ещё и подстрокой.
	//
	// В русском языке род техники стоит в КОНЦЕ сложного слова: «автокран» —
	// это кран, «электропогрузчик» — погрузчик. Стеммер их не связывает,
	// и «кран» терял 26 автокранов, которые прежний поиск подстрокой находил.
	//
	// Только для однословного запроса, и это не осторожность, а замер:
	// на «гусеничный кран» подстрока не добавляет ничего (такой строки
	// в названиях нет), зато на многословных запросах общее правило начало
	// бы тащить случайные совпадения. На однословных проверено по всему
	// списку запросов — меняется ровно «кран», остальные до одной карточки
	// те же.
	if !strings.ContainsAny(query, " \t") {
		pattern := b.placeholder("%" + escapeLikePattern(query) + "%")
		match = fmt.Sprintf("(%s OR %s ILIKE %s)", match, machineSearchText, pattern)
	}

	return match, rank
}

// machineRelevanceExpression собирает целочисленную релевантность.
//
// Ступень важнее точности совпадения, поэтому она стоит в старших разрядах,
// а точность — в младших. Порядок ступеней задан тем, насколько однозначно
// запрос выражает намерение:
//
//	5 — модель совпала целиком. «XE370G» — это просьба показать XE370G,
//	    и любой другой ответ выше неё был бы неверным;
//	4 — модель начинается с запроса. «xe370» — то же намерение, просто
//	    набранное не до конца;
//	3 — название начинается с запроса. Карточка, чьё имя ВЕДЁТ с этого
//	    слова, скорее и есть эта техника, а не та, где слово в середине;
//	2 — совпал бренд. Ответ верный, но почти ничего не сужающий: под XCMG
//	    подходит 439 карточки из 456;
//	1 — остальное.
//
// Целое, а не дробное, ради курсора: дробь пришлось бы сравнивать
// с пересчитанной на следующей странице, и расхождение в младшем разряде
// молча теряло бы строки на границе страниц.
func machineRelevanceExpression(b *conditionBuilder, query string, ranks []string) string {
	exact := b.placeholder(query)
	prefix := b.placeholder(escapeLikePattern(query) + "%")

	tier := fmt.Sprintf("CASE"+
		" WHEN lower(m.model) = lower(%[1]s) THEN 5"+
		" WHEN lower(m.model) LIKE lower(%[2]s) THEN 4"+
		" WHEN lower(m.name_ru) LIKE lower(%[2]s) THEN 3"+
		" WHEN lower(m.brand) = lower(%[1]s) THEN 2"+
		" ELSE 1 END", exact, prefix)

	// Из нескольких написаний берётся лучшее: раскладку человек перепутал,
	// а искал одно и то же.
	best := ranks[0]
	if len(ranks) > 1 {
		best = fmt.Sprintf("GREATEST(%s)", strings.Join(ranks, ", "))
	}

	return fmt.Sprintf("((%s) * 1000 + LEAST(GREATEST(((%s) * 1000)::int, 0), 999))", tier, best)
}

// escapeLikePattern обезвреживает символы шаблона в пользовательском вводе.
func escapeLikePattern(value string) string {
	// В raw-строках обратный слэш литерален: слева ищется ОДИН символ,
	// справа подставляются ДВА — то есть экранированный сам собой.
	// Экранирующий символ ILIKE по умолчанию именно обратный слэш,
	// поэтому отдельного ESCAPE в запросе не нужно.
	replacer := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

	return replacer.Replace(value)
}

// appendCategorySubtree добавляет условие по категории и, при обходе поддерева,
// возвращает префикс запроса с рекурсивным CTE.
//
// В обоих случаях условие — РАВЕНСТВО по category_id, и в этом весь смысл:
// именно равенство позволяет планировщику взять готовый порядок из индекса
// витрины. При обходе поддерева равенство берётся со строкой латерали
// (см. machineSearchLateral), поэтому запрос выполняется один раз на каждую
// категорию поддерева, а не один раз на всю таблицу.
func appendCategorySubtree(b *conditionBuilder, filter *entity.MachineFilter) string {
	if filter.CategoryID == nil {
		return ""
	}

	if !filter.IncludeSubcategories {
		b.addf("m.category_id = %s", *filter.CategoryID)

		return ""
	}

	prefix := fmt.Sprintf(categorySubtreeCTE,
		categoryTable, b.placeholder(*filter.CategoryID),
		categoryTable, b.placeholder(maxCategoryDepth))

	// Корреляция с латералью: s — категория текущей итерации обхода.
	b.add("m.category_id = s.id")

	return prefix + " "
}

// appendAttributeConditions добавляет условия по характеристикам.
//
// Каждое условие — отдельный EXISTS-подзапрос, а не join к таблице значений.
// Три причины:
//
//  1. join по N характеристикам размножил бы строки карточки в N раз и потребовал
//     DISTINCT или GROUP BY поверх выдачи, ломая и keyset-пагинацию, и сортировку;
//  2. EXISTS — полусоединение: планировщик останавливается на первой подошедшей
//     строке значения и волен начать с самого селективного условия, тогда как
//     порядок join-ов пришлось бы задавать самим;
//  3. каждый подзапрос независимо ложится на свой частичный индекс
//     (attribute_id, num_value) или (attribute_id, text_value) — условия внутри
//     одного EXISTS относятся к одной строке значения, и составной индекс
//     покрывает их целиком.
//
// Рассматривалась агрегация с HAVING count(DISTINCT attribute_id): она читает
// значения всех упомянутых характеристик и группирует их, то есть платит за
// полный проход даже когда первое же условие отсекает почти всё. Выигрыш
// у неё появляется на десятках условий в одном запросе, а панель фильтров
// каталога — это единицы.
func appendAttributeConditions(b *conditionBuilder, conditions []entity.AttributeCondition) error {
	for _, condition := range conditions {
		switch c := condition.(type) {
		case entity.NumberRangeCondition:
			appendNumberRangeCondition(b, c)
		case entity.EnumCondition:
			appendEnumCondition(b, c)
		case entity.BoolCondition:
			appendBoolCondition(b, c)
		default:
			// Набор реализаций закрыт приватным методом-маркером, поэтому сюда
			// можно попасть только с nil-условием.
			return entity.ErrAttributeValueInvalid
		}
	}

	return nil
}

// appendNumberRangeCondition добавляет EXISTS по числовому диапазону.
func appendNumberRangeCondition(b *conditionBuilder, c entity.NumberRangeCondition) {
	if c.IsEmpty() {
		return
	}

	inner := []string{
		"v.attribute_id = " + b.placeholder(c.AttributeID),
	}

	// Строгие сравнения подразумевают num_value IS NOT NULL, поэтому частичный
	// индекс machine_attribute_values_num_filter_idx применим без отдельного
	// предиката: планировщик выводит это сам.
	if c.Min != nil {
		inner = append(inner, "v.num_value >= "+b.placeholder(c.Min.String()))
	}

	if c.Max != nil {
		inner = append(inner, "v.num_value <= "+b.placeholder(c.Max.String()))
	}

	b.addExists(inner)
}

// appendEnumCondition добавляет EXISTS по набору кодов варианта.
func appendEnumCondition(b *conditionBuilder, c entity.EnumCondition) {
	if c.IsEmpty() {
		return
	}

	// Массив уходит одним аргументом, а не списком плейсхолдеров: число
	// выбранных вариантов не меняет текст запроса, поэтому план разбирается
	// один раз на все комбинации флажков в панели фильтров.
	b.addExists([]string{
		"v.attribute_id = " + b.placeholder(c.AttributeID),
		"v.text_value = ANY(" + b.placeholder(c.Codes) + ")",
	})
}

// appendBoolCondition добавляет EXISTS по флаговой характеристике.
func appendBoolCondition(b *conditionBuilder, c entity.BoolCondition) {
	b.addExists([]string{
		"v.attribute_id = " + b.placeholder(c.AttributeID),
		"v.bool_value = " + b.placeholder(c.Flag),
	})
}

// appendCursorCondition добавляет условие продолжения выдачи с курсора.
//
// Сравнение записано кортежем ((a, b) > (x, y)), а не раскрыто в
// «a > x OR (a = x AND b > y)»: кортежная форма атомарна, читается как одно
// сравнение и позволяет планировщику пройти по составному индексу с нужной
// позиции вместо разбора двух ветвей OR.
func appendCursorCondition(b *conditionBuilder, filter *entity.MachineFilter, relevance string) error {
	if filter.Cursor == nil {
		return nil
	}

	cursor := filter.Cursor

	switch filter.Sort {
	case entity.SortPriceAsc:
		appendPriceCursorCondition(b, cursor, ">")
	case entity.SortPriceDesc:
		appendPriceCursorCondition(b, cursor, "<")
	case entity.SortPublishedDesc:
		appendPublishedCursorCondition(b, cursor)
	case entity.SortRelevance:
		if cursor.Relevance == nil {
			return entity.ErrCatalogCursorInvalid
		}

		appendRelevanceCursorCondition(b, cursor, relevance)
	default:
		return entity.ErrCatalogSortInvalid
	}

	return nil
}

// appendRelevanceCursorCondition добавляет условие продолжения ранжированной
// выдачи.
//
// Порядок здесь трёхключевой: релевантность DESC, дата публикации DESC NULLS
// FIRST, id DESC. Первый ключ отсекается сравнением, а хвост — ровно то же
// условие, что и у сортировки по дате, со всеми его краями: сравнение с NULL
// даёт «неизвестно», а не «ложь», и кортежным сравнением такой порядок
// не записывается.
//
// Выражение релевантности повторяется здесь и в ORDER BY. Это не дублирование
// текста, а то же самое выражение, собранное один раз и подставленное дважды:
// разойтись они не могут, а сравнивать в WHERE колонку из SELECT нельзя —
// её там ещё нет.
func appendRelevanceCursorCondition(b *conditionBuilder, cursor *entity.MachineCursor, relevance string) {
	rank := b.placeholder(*cursor.Relevance)

	if cursor.PublishedAt == nil {
		b.add(fmt.Sprintf(
			"((%[1]s) < %[2]s OR ((%[1]s) = %[2]s"+
				" AND ((m.published_at IS NULL AND m.id < %[3]s) OR m.published_at IS NOT NULL)))",
			relevance, rank, b.placeholder(cursor.ID)))

		return
	}

	b.add(fmt.Sprintf(
		"((%[1]s) < %[2]s OR ((%[1]s) = %[2]s"+
			" AND m.published_at IS NOT NULL AND (m.published_at, m.id) < (%[3]s, %[4]s)))",
		relevance, rank, b.placeholder(*cursor.PublishedAt), b.placeholder(cursor.ID)))
}

// appendPriceCursorCondition добавляет условие продолжения выдачи,
// отсортированной по цене.
//
// Порядок здесь — цена ASC или DESC, но NULLS LAST в обе стороны: карточки
// без цены всегда в хвосте. Отсюда два края, и каждый требует своего условия —
// ровно как у пустой даты публикации, только хвост с другой стороны.
//
// Край первый — курсор стоит на карточке С ценой. Продолжение это «оставшиеся
// карточки с ценой за ней, а за ними весь хвост без цены». Одним кортежным
// сравнением это не записать: сравнение с NULL даёт не «ложь», а «неизвестно»,
// и карточки без цены выпали бы из выдачи целиком.
//
// Край второй — курсор стоит на карточке БЕЗ цены. Значит, хвост уже начался,
// и продолжение это только оставшийся хвост: карточки с ценой все показаны
// на предыдущих страницах.
//
// Направление сравнения по id совпадает с направлением сортировки — иначе
// внутри хвоста, где цены нет и решает только id, страницы пошли бы
// в обратную сторону.
func appendPriceCursorCondition(b *conditionBuilder, cursor *entity.MachineCursor, cmp string) {
	if cursor.PriceCNYCents == nil {
		b.add(fmt.Sprintf("(%s IS NULL AND m.id %s %s)",
			machinePriceOrder, cmp, b.placeholder(cursor.ID)))

		return
	}

	// Скобки внутри выписаны, хотя AND и так связывает крепче OR:
	// на приоритете операторов такое условие держаться не должно.
	b.add(fmt.Sprintf("((%[1]s IS NOT NULL AND (%[1]s, m.id) %[2]s (%[3]s, %[4]s)) OR %[1]s IS NULL)",
		machinePriceOrder, cmp,
		b.placeholder(*cursor.PriceCNYCents), b.placeholder(cursor.ID)))
}

// appendPublishedCursorCondition добавляет условие продолжения выдачи,
// отсортированной по дате публикации.
//
// Порядок здесь — published_at DESC NULLS FIRST, id DESC, то есть черновики
// идут перед опубликованными: у них даты нет, а в убывающем порядке PostgreSQL
// ставит пустые значения первыми. Отсюда два края, и каждый требует своего
// условия.
//
// Край первый — курсор стоит на черновике. Продолжение это «оставшиеся
// черновики с меньшим id, а за ними все опубликованные». Кортежным сравнением
// это не записать: сравнение с NULL даёт не «ложь», а «неизвестно», и строка
// выпадает из выдачи — потерялись бы либо черновики, либо опубликованные.
//
// Край второй — курсор стоит на опубликованной карточке. Здесь работает
// прежнее кортежное сравнение, а черновики отсекаются сами: они уже показаны
// на предыдущих страницах, и NULL в сравнении как раз их и исключает.
// Проверка IS NOT NULL выписана явно, чтобы это следовало из текста условия,
// а не из знания трёхзначной логики SQL.
func appendPublishedCursorCondition(b *conditionBuilder, cursor *entity.MachineCursor) {
	if cursor.PublishedAt == nil {
		// Скобки внутри выписаны, хотя AND и так связывает крепче OR:
		// на приоритете операторов такое условие держаться не должно.
		b.add(fmt.Sprintf("((m.published_at IS NULL AND m.id < %s) OR m.published_at IS NOT NULL)",
			b.placeholder(cursor.ID)))

		return
	}

	b.add(fmt.Sprintf("(m.published_at IS NOT NULL AND (m.published_at, m.id) < (%s, %s))",
		b.placeholder(*cursor.PublishedAt), b.placeholder(cursor.ID)))
}

// conditionBuilder собирает условия WHERE и список аргументов.
//
// Единственный способ поместить значение в запрос — placeholder: он кладёт
// значение в args и возвращает его номер. Никакой метод не принимает текст,
// собранный из входных данных, поэтому конкатенация пользовательского значения
// в SQL здесь невыразима, а не запрещена соглашением.
type conditionBuilder struct {
	conditions []string // Условия WHERE
	args       []any    // Аргументы запроса в порядке плейсхолдеров
}

// newConditionBuilder создаёт построитель условий.
func newConditionBuilder() *conditionBuilder {
	return &conditionBuilder{}
}

// placeholder регистрирует аргумент и возвращает его плейсхолдер.
func (b *conditionBuilder) placeholder(value any) string {
	b.args = append(b.args, value)

	return "$" + strconv.Itoa(len(b.args))
}

// add добавляет готовое условие.
func (b *conditionBuilder) add(condition string) {
	b.conditions = append(b.conditions, condition)
}

// addf добавляет условие с одним значением: шаблон обязан содержать ровно один
// глагол подстановки, куда встанет плейсхолдер, а не само значение.
func (b *conditionBuilder) addf(template string, value any) {
	b.add(fmt.Sprintf(template, b.placeholder(value)))
}

// addExists добавляет EXISTS-подзапрос к значениям характеристик.
func (b *conditionBuilder) addExists(inner []string) {
	b.add(fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s v WHERE v.machine_id = m.id AND %s)",
		machineAttributeValueTable, strings.Join(inner, " AND "),
	))
}

// where собирает условия в предикат.
// Пустой набор даёт TRUE: запрос без фильтров обязан вернуть весь каталог,
// а не сломаться на висящем WHERE.
func (b *conditionBuilder) where() string {
	if len(b.conditions) == 0 {
		return "TRUE"
	}

	return strings.Join(b.conditions, " AND ")
}

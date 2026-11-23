package entity

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Границы размера страницы каталога.
const (
	// DefaultMachineLimit размер страницы по умолчанию.
	DefaultMachineLimit = 20
	// MaxMachineLimit верхняя граница размера страницы.
	//
	// Ограничение обязательно: без него один запрос с limit в миллион выгружает
	// весь каталог вместе с характеристиками и укладывает базу.
	MaxMachineLimit = 100
)

// MachineSort порядок сортировки выдачи каталога.
//
// Перечень закрыт: порядок попадает в ORDER BY, и принимать его из входных
// данных как есть означало бы позволить клиенту диктовать текст запроса.
type MachineSort string

// Допустимые порядки сортировки.
const (
	// SortPriceAsc по возрастанию цены.
	SortPriceAsc MachineSort = "price_asc"
	// SortPriceDesc по убыванию цены.
	SortPriceDesc MachineSort = "price_desc"
	// SortPublishedDesc по убыванию даты публикации: свежие сверху.
	SortPublishedDesc MachineSort = "published_desc"
	// SortRelevance по релевантности запросу: сначала точная модель.
	//
	// Клиент его не запрашивает — в форме поиска его нет, и добавлять некуда:
	// без запроса он бессмыслен, а с запросом нужен всегда. Сервис включает
	// его САМ, когда есть текст поиска и не выбран явный порядок. Наружу
	// он выходит только внутри курсора, чтобы следующая страница собралась
	// в том же порядке, что и первая.
	SortRelevance MachineSort = "relevance"
)

// String возвращает строковое представление порядка сортировки.
func (s MachineSort) String() string { return string(s) }

// IsValid сообщает, что порядок сортировки известен домену.
func (s MachineSort) IsValid() bool {
	switch s {
	case SortPriceAsc, SortPriceDesc, SortPublishedDesc, SortRelevance:
		return true
	default:
		return false
	}
}

// IsByPrice сообщает, что порядок задаётся ценой.
//
// Обе ценовые сортировки требуют от курсора одного и того же ключа, и
// перечислять их по одной в каждом месте — способ однажды забыть одну.
func (s MachineSort) IsByPrice() bool {
	return s == SortPriceAsc || s == SortPriceDesc
}

// MachineCursor позиция, с которой продолжается выдача.
//
// Keyset-пагинация вместо OFFSET: OFFSET заставляет базу прочитать и отбросить
// все пропускаемые строки, поэтому цена страницы растёт с её номером, а вставка
// новой карточки во время листания сдвигает окно и показывает одну и ту же
// машину дважды. Курсор же указывает на конкретную строку, и обе беды исчезают.
//
// Курсор несёт свою сортировку: продолжить выдачу, отсортированную по цене,
// курсором от сортировки по дате нельзя — сравнение пошло бы по другому полю
// и выдало бы произвольный кусок каталога.
type MachineCursor struct {
	Sort MachineSort `json:"sort"` // Сортировка, в которой курсор получен

	// PriceCNYCents цена последней строки страницы; nil — цена скрыта.
	//
	// Указатель по той же причине, что и у PublishedAt: скрытая цена — это
	// не ноль и не бесконечность, а отдельное состояние. Такие карточки идут
	// в конце выдачи в обе стороны сортировки, и курсор обязан отличать
	// «остановились на карточке за 150 000» от «остановились на карточке
	// без цены» — иначе продолжение собралось бы не с той границы.
	//
	// Без omitempty намеренно: у карточки без цены поле иначе исчезало бы
	// из курсора, а отсутствие ключа неотличимо от испорченного курсора.
	PriceCNYCents *int64 `json:"priceCnyCents"`

	// PublishedAt дата публикации последней строки страницы; nil — карточка
	// не опубликована.
	//
	// Без omitempty намеренно: у черновика даты нет, и раньше поле просто
	// исчезало из курсора. Отсутствие поля неотличимо от испорченного курсора,
	// поэтому разбор такой курсор отвергал — то есть сервер выпускал курсор,
	// который сам же не принимал, и пагинация по выдаче с черновиками
	// обрывалась на первой же странице. Теперь пустая дата выражена явно,
	// значением null, и остаётся законной позицией курсора.
	PublishedAt *time.Time `json:"publishedAt"`

	// Relevance релевантность последней строки; nil — порядок не по релевантности.
	//
	// Целое, а не дробное, и это осознанно: релевантность собирается из ступени
	// (точная модель, префикс модели, начало названия, бренд, остальное)
	// и дробной части, приведённой к целому. Дробь в курсоре сравнивалась бы
	// с пересчитанной на следующей странице и расходилась в младшем разряде,
	// а целое сравнивается точно.
	//
	// Без omitempty намеренно, по той же причине, что и у соседей: отсутствие
	// ключа неотличимо от испорченного курсора.
	Relevance *int `json:"relevance"`

	// Fuzzy страница получена ЗАПАСНЫМ путём — триграммами, а не полнотекстовым
	// поиском.
	//
	// Путь обязан сохраняться между страницами. Первая страница по запросу
	// «эскаватор» собирается триграммами, потому что полнотекстовый поиск
	// не нашёл ничего; если вторую собрать основным путём, продолжение пойдёт
	// по другой выдаче — то есть по пустой.
	Fuzzy bool `json:"fuzzy"`

	// ID последней строки страницы. Тай-брейкер: ни цена, ни дата публикации
	// не уникальны, и без него две машины с одинаковой ценой давали бы
	// неустойчивый порядок — на границе страниц одна терялась бы, другая
	// показывалась дважды.
	ID string `json:"id"`
}

// Encode упаковывает курсор в непрозрачную для клиента строку.
//
// Клиент не должен собирать курсор сам: его содержимое — деталь реализации
// пагинации, и разбор строки на стороне клиента закрепил бы её как контракт.
func (c MachineCursor) Encode() (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", ErrCatalogCursorInvalid
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// machineCursorPayload промежуточное представление курсора при разборе.
//
// PublishedAt здесь json.RawMessage, а не *time.Time, ради одного различия:
// для указателя отсутствующий ключ и ключ со значением null неразличимы —
// оба дают nil. А различать их нужно: null означает «последняя строка страницы
// не опубликована» и это законная позиция, а отсутствие ключа означает курсор,
// собранный не нами.
type machineCursorPayload struct {
	Sort          MachineSort     `json:"sort"`
	PriceCNYCents json.RawMessage `json:"priceCnyCents"`
	PublishedAt   json.RawMessage `json:"publishedAt"`
	Relevance     json.RawMessage `json:"relevance"`
	Fuzzy         bool            `json:"fuzzy"`
	ID            string          `json:"id"`
}

// DecodeMachineCursor разбирает курсор из строки.
func DecodeMachineCursor(encoded string) (*MachineCursor, error) {
	if encoded == "" {
		return nil, nil //nolint:nilnil // пустой курсор — первая страница, а не отказ
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrCatalogCursorInvalid
	}

	var payload machineCursorPayload
	if err = json.Unmarshal(raw, &payload); err != nil {
		return nil, ErrCatalogCursorInvalid
	}

	if payload.ID == "" || !payload.Sort.IsValid() {
		return nil, ErrCatalogCursorInvalid
	}

	cursor := MachineCursor{Sort: payload.Sort, ID: payload.ID}

	if keyErr := payload.applySortKeys(&cursor); keyErr != nil {
		return nil, keyErr
	}

	return &cursor, nil
}

// applySortKeys переносит ключи сортировки из разобранного JSON в курсор.
//
// Ключ, по которому идёт сортировка, обязан присутствовать хотя бы значением
// null: иначе продолжать выдачу не от чего, и курсор без ключа увёл бы её
// к началу каталога — молча и на каждой странице. Пустая дата публикации
// и скрытая цена при этом законные позиции, и отличить их от «ключа нет»
// можно только по наличию самого поля.
func (p *machineCursorPayload) applySortKeys(cursor *MachineCursor) error {
	if err := p.requireSortKey(); err != nil {
		return err
	}

	if len(p.PublishedAt) > 0 {
		if err := json.Unmarshal(p.PublishedAt, &cursor.PublishedAt); err != nil {
			return ErrCatalogCursorInvalid
		}
	}

	if len(p.PriceCNYCents) > 0 {
		if err := json.Unmarshal(p.PriceCNYCents, &cursor.PriceCNYCents); err != nil {
			return ErrCatalogCursorInvalid
		}
	}

	if len(p.Relevance) > 0 {
		if err := json.Unmarshal(p.Relevance, &cursor.Relevance); err != nil {
			return ErrCatalogCursorInvalid
		}
	}

	cursor.Fuzzy = p.Fuzzy

	return nil
}

// requireSortKey проверяет, что ключ выбранного порядка в курсоре есть.
//
// Отдельно от разбора значений, потому что это разные вопросы: здесь —
// «ключ вообще передан», ниже — «его удалось прочитать». Слитно они дают
// функцию, в которой каждая сортировка добавляет по две ветки.
func (p *machineCursorPayload) requireSortKey() error {
	switch {
	case p.Sort == SortPublishedDesc && len(p.PublishedAt) == 0:
		return ErrCatalogCursorInvalid
	case p.Sort.IsByPrice() && len(p.PriceCNYCents) == 0:
		return ErrCatalogCursorInvalid
	case p.Sort == SortRelevance && len(p.Relevance) == 0:
		return ErrCatalogCursorInvalid
	default:
		return nil
	}
}

// AttributeCondition условие фильтрации по одной характеристике.
//
// Замкнутый набор реализаций, как и у AttributeValue: построитель запроса
// разбирает его одним переключателем, и «условие неизвестного вида» не
// существует. Благодаря этому в SQL не может попасть ни оператор, ни имя
// колонки, выведенные из входных данных.
type AttributeCondition interface {
	// TargetAttributeID возвращает характеристику, к которой применяется условие.
	TargetAttributeID() string
	// isAttributeCondition закрывает набор реализаций границами пакета.
	isAttributeCondition()
}

// NumberRangeCondition диапазон значений числовой характеристики.
// Обе границы включающие; nil означает отсутствие границы.
type NumberRangeCondition struct {
	AttributeID string           // Характеристика
	Min         *decimal.Decimal // Нижняя граница включительно
	Max         *decimal.Decimal // Верхняя граница включительно
}

// NewNumberRangeCondition создаёт условие диапазона.
func NewNumberRangeCondition(attributeID string, minValue, maxValue *decimal.Decimal) NumberRangeCondition {
	return NumberRangeCondition{AttributeID: attributeID, Min: minValue, Max: maxValue}
}

// TargetAttributeID возвращает характеристику условия.
func (c NumberRangeCondition) TargetAttributeID() string { return c.AttributeID }

func (NumberRangeCondition) isAttributeCondition() {}

// IsEmpty сообщает, что условие не ограничивает выборку.
func (c NumberRangeCondition) IsEmpty() bool { return c.Min == nil && c.Max == nil }

// EnumCondition набор допустимых кодов варианта.
// Несколько кодов означают «любой из», как флажки в панели фильтров.
type EnumCondition struct {
	AttributeID string   // Характеристика
	Codes       []string // Допустимые коды вариантов
}

// NewEnumCondition создаёт условие по вариантам.
func NewEnumCondition(attributeID string, codes ...string) EnumCondition {
	return EnumCondition{AttributeID: attributeID, Codes: codes}
}

// TargetAttributeID возвращает характеристику условия.
func (c EnumCondition) TargetAttributeID() string { return c.AttributeID }

func (EnumCondition) isAttributeCondition() {}

// IsEmpty сообщает, что условие не ограничивает выборку.
func (c EnumCondition) IsEmpty() bool { return len(c.Codes) == 0 }

// BoolCondition условие по флаговой характеристике.
type BoolCondition struct {
	AttributeID string // Характеристика
	Flag        bool   // Требуемое значение
}

// NewBoolCondition создаёт условие по флагу.
func NewBoolCondition(attributeID string, flag bool) BoolCondition {
	return BoolCondition{AttributeID: attributeID, Flag: flag}
}

// TargetAttributeID возвращает характеристику условия.
func (c BoolCondition) TargetAttributeID() string { return c.AttributeID }

func (BoolCondition) isAttributeCondition() {}

// MachineFilter набор условий поиска по каталогу.
//
// Все поля-указатели необязательны: nil означает «не фильтровать».
// Условия соединяются по И — как флажки в панели фильтров, каждый из которых
// сужает выдачу.
type MachineFilter struct {
	// CategoryID ограничивает выдачу категорией.
	CategoryID *string
	// IncludeSubcategories расширяет отбор на всё поддерево CategoryID.
	//
	// Клик по «Экскаваторы» обязан показывать и мини-экскаваторы: дерево
	// с parent_id и триггер против циклов заведены именно под вложенность,
	// то есть это заложенный сценарий, а не гипотеза. Без флага выдача
	// ограничена ровно одним узлом.
	IncludeSubcategories bool
	// PriceMinCents нижняя граница цены в фынях (1/100 юаня) включительно.
	PriceMinCents *int64
	// PriceMaxCents верхняя граница цены в фынях (1/100 юаня) включительно.
	PriceMaxCents *int64
	// Availability наличие.
	Availability *MachineAvailability
	// Condition состояние.
	Condition *MachineCondition
	// ProductionYearMin нижняя граница года выпуска включительно.
	ProductionYearMin *int
	// ProductionYearMax верхняя граница года выпуска включительно.
	ProductionYearMax *int
	// Attributes условия по характеристикам, произвольный набор.
	Attributes []AttributeCondition

	// OnlyPublished ограничивает выдачу опубликованными карточками.
	// Ставится сервисом, а не приходит снаружи: путать публичное и админское
	// чтение нельзя, и решает это метод сервиса, а не поле формы.
	OnlyPublished bool

	// Query подстрока для поиска по названию, бренду и модели; пусто — без поиска.
	//
	// Одно поле на витрину и на админку: искомые колонки одни и те же,
	// а различие между публичным и административным чтением — только в том,
	// какие строки видны, и это решает OnlyPublished, а не клиент. Второе
	// поле означало бы два пути через один и тот же предикат, и второй
	// рано или поздно разошёлся бы с первым.
	Query string

	// Fuzzy искать ЗАПАСНЫМ путём: триграммами вместо полнотекстового поиска.
	//
	// Ставит его репозиторий, когда основной путь ничего не нашёл, и курсор,
	// когда страница продолжает уже начатую нечёткую выдачу. Снаружи он
	// не приходит: клиент не выбирает, каким способом искать.
	Fuzzy bool

	// IsActive ограничивает выдачу опубликованными или черновиками.
	//
	// Учитывается только в административном чтении: на витрине признак уже
	// зафиксирован OnlyPublished, и позволить клиенту его переопределить
	// значило бы отдать черновики параметром запроса.
	IsActive *bool

	// Sort порядок сортировки.
	Sort MachineSort
	// Cursor позиция, с которой продолжается выдача; nil — первая страница.
	Cursor *MachineCursor
	// Limit размер страницы.
	Limit int
	// WithTotal просить ли общее число найденных карточек.
	//
	// По запросу клиента, а не всегда: точный подсчёт — второй запрос по тому
	// же предикату, и он обязан просмотреть ВСЕ подходящие строки, тогда как
	// сама страница читает ровно Limit. Бесконечной прокрутке итог не нужен,
	// нумерованной выдаче нужен — и платит за него та, что его просит.
	WithTotal bool
}

// Validate проверяет параметры поиска и подставляет умолчания.
func (f *MachineFilter) Validate() error {
	if f == nil {
		return ErrNilPointer
	}

	if f.Sort == "" {
		f.Sort = SortPublishedDesc
	}

	if !f.Sort.IsValid() {
		return ErrCatalogSortInvalid
	}

	if f.Limit == 0 {
		f.Limit = DefaultMachineLimit
	}

	if f.Limit < 0 || f.Limit > MaxMachineLimit {
		return ErrCatalogLimitInvalid
	}

	// Курсор от другой сортировки продолжил бы выдачу сравнением по чужому
	// полю и вернул бы произвольный кусок каталога.
	if f.Cursor != nil && f.Cursor.Sort != f.Sort {
		return ErrCatalogCursorInvalid
	}

	return nil
}

// MachinePage страница выдачи каталога.
type MachinePage struct {
	// Machines карточки страницы.
	Machines Machines
	// NextCursor курсор следующей страницы; пуст, если страница последняя.
	NextCursor string
	// HasMore сообщает, что за этой страницей есть ещё карточки.
	HasMore bool
	// Total общее число найденных карточек; nil — не считали.
	//
	// Указатель, а не ноль: «не считали» и «ничего не нашли» — разные ответы,
	// и ноль клиент показал бы как «ничего не найдено».
	Total *int
}

// NewMachinePage собирает страницу из выдачи, запрошенной с запасом в одну строку.
//
// Лишняя строка отвечает на вопрос «есть ли продолжение» без отдельного
// count(*): на фильтрах по характеристикам подсчёт стоил бы столько же,
// сколько сама выдача, а клиенту достаточно знать, показывать ли кнопку.
func NewMachinePage(machines Machines, limit int, filter *MachineFilter) (MachinePage, error) {
	page := MachinePage{Machines: machines}

	if len(machines) > limit {
		page.Machines = machines[:limit]
		page.HasMore = true
	}

	if !page.HasMore || len(page.Machines) == 0 {
		return page, nil
	}

	last := page.Machines[len(page.Machines)-1]

	cursor := MachineCursor{
		Sort:        filter.Sort,
		PublishedAt: last.PublishedAt,
		// Путь поиска переносится в курсор: продолжение обязано идти тем же,
		// иначе вторая страница соберётся по другой выдаче.
		Fuzzy: filter.Fuzzy,
		ID:    last.ID,
	}

	// Релевантность попадает в курсор только в своём порядке: в остальных
	// она не считалась, и ноль означал бы позицию, которой в выдаче нет.
	if filter.Sort == SortRelevance {
		cursor.Relevance = Pointer(last.Relevance)
	}

	// Цена попадает в курсор, только если она видна: скрытая для выдачи
	// не существует, и подставить её значение значило бы продолжить
	// с позиции, которой в порядке нет.
	if last.IsPricePublic {
		cursor.PriceCNYCents = Pointer(last.PriceCNYCents)
	}

	encoded, err := cursor.Encode()
	if err != nil {
		return MachinePage{}, err
	}

	page.NextCursor = encoded

	return page, nil
}

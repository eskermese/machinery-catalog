package entity

import (
	"regexp"
	"time"

	"github.com/shopspring/decimal"
)

// slugPattern формат слага: строчные буквы, цифры и дефис-разделитель.
// Совпадает с CHECK-ами categories_slug_format и machines_slug_format.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsValidSlug проверяет формат слага.
func IsValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

// codePattern формат кода: строчные буквы, цифры и подчёркивание-разделитель.
// Совпадает с CHECK-ами attributes_code_format и attribute_options_code_format.
//
// Отличается от слага разделителем: код уходит в имена полей и ключи фильтров,
// где дефис пришлось бы экранировать, а слаг — в адрес страницы, где дефис
// принят.
var codePattern = regexp.MustCompile(`^[a-z0-9]+(_[a-z0-9]+)*$`)

// IsValidCode проверяет формат кода характеристики или варианта.
func IsValidCode(code string) bool {
	return codePattern.MatchString(code)
}

// Category узел дерева категорий спецтехники.
type Category struct {
	ID        string    // Идентификатор
	ParentID  *string   // Родительский узел; nil — корень
	Slug      string    // Идентификатор в URL
	NameRU    string    // Название на русском
	NameKK    string    // Название на казахском
	NameEN    string    // Название на английском
	Position  int       // Ручной порядок вывода
	IsActive  bool      // Видимость ветки на сайте
	ImagePath *string   // Обложка, выбранная вручную; nil — подставит сервис
	CreatedAt time.Time // Дата создания
	UpdatedAt time.Time // Дата изменения

	// CoverPath обложка, которую увидит клиент: заданная вручную, а если
	// её нет — главное фото первой машины поддерева. Считает сервис,
	// в базе такого поля нет.
	//
	// Отдельно от ImagePath намеренно: админке нужно видеть, задана обложка
	// или подставлена, иначе редактор не отличит «выбрал этот снимок»
	// от «так совпало».
	CoverPath *string
}

// CategoryCover кандидат в обложку раздела — лучшая машина категории.
//
// Хранит не только путь: обложку раздела выбирает МАКСИМУМ по поддереву,
// и сравнивать кандидатов надо по тому же порядку, в котором витрина
// показывает карточки, — иначе обложка раздела и первая плитка в нём
// оказались бы разными снимками.
type CategoryCover struct {
	Path        string     // Путь главного фото
	PublishedAt *time.Time // Дата публикации машины
	MachineID   string     // Идентификатор машины — тай-брейкер
}

// Newer сообщает, что кандидат идёт раньше другого в порядке витрины
// (published_at DESC NULLS FIRST, id DESC).
func (c CategoryCover) Newer(other CategoryCover) bool {
	if c.PublishedAt == nil || other.PublishedAt == nil {
		// NULLS FIRST: неопубликованная дата идёт раньше любой заданной.
		return c.PublishedAt == nil && other.PublishedAt != nil
	}

	if !c.PublishedAt.Equal(*other.PublishedAt) {
		return c.PublishedAt.After(*other.PublishedAt)
	}

	return c.MachineID > other.MachineID
}

// Categories список категорий.
type Categories []*Category

// Validate проверяет пригодность категории к сохранению.
func (c *Category) Validate() error {
	if c == nil {
		return ErrNilPointer
	}

	if !IsValidSlug(c.Slug) {
		return ErrSlugInvalid
	}

	// Обязателен только русский. Казахское и английское названия пусты,
	// пока их не перевели: «перевода нет» — обычное состояние каталога
	// на четыреста пятьдесят карточек, а не ошибка ввода. Витрина
	// на непереведённом языке показывает русское.
	if c.NameRU == "" {
		return ErrCategoryNameRequired
	}

	if c.Position < 0 {
		return ErrPositionInvalid
	}

	return nil
}

// AttributeDataType тип значения характеристики.
type AttributeDataType string

// Типы значений характеристик. Значения совпадают с CHECK attributes_data_type_allowed.
const (
	AttributeTypeNumber AttributeDataType = "number" // Числовая величина
	AttributeTypeString AttributeDataType = "string" // Свободная строка
	AttributeTypeBool   AttributeDataType = "bool"   // Флаг
	AttributeTypeEnum   AttributeDataType = "enum"   // Код варианта из attribute_options
)

// String возвращает строковое представление типа значения.
func (t AttributeDataType) String() string { return string(t) }

// IsValid сообщает, что тип значения известен домену.
func (t AttributeDataType) IsValid() bool {
	switch t {
	case AttributeTypeNumber, AttributeTypeString, AttributeTypeBool, AttributeTypeEnum:
		return true
	default:
		return false
	}
}

// AttributeBounds границы значений числовой характеристики в разделе.
type AttributeBounds struct {
	Min decimal.Decimal // Наименьшее значение в разделе
	Max decimal.Decimal // Наибольшее значение в разделе
}

// Attribute метаданные характеристики, привязанной к категории.
type Attribute struct {
	ID           string            // Идентификатор
	CategoryID   string            // Категория, которой принадлежит характеристика
	Code         string            // Машинный ключ
	NameRU       string            // Название на русском
	NameKK       string            // Название на казахском
	NameEN       string            // Название на английском
	DataType     AttributeDataType // Тип значения
	Unit         *string           // Единица измерения; только у числовых
	IsFilterable bool              // Выносить ли в панель фильтров
	Position     int               // Порядок вывода
	CreatedAt    time.Time         // Дата создания
	UpdatedAt    time.Time         // Дата изменения
	Options      AttributeOptions  // Варианты для типа enum

	// NumberMin и NumberMax границы значений характеристики в разделе.
	//
	// Не свойство характеристики, а свойство ВЫДАЧИ: заполняются только
	// публичным каталогом и только для числовых, потому что нужны ровно
	// одному — подсказке под полем фильтра. Пустая пара «от — до» без единой
	// цифры не говорит, идёт ли речь о сотне или о сотне тысяч, и человек
	// либо угадывает, либо не трогает фильтр вовсе.
	//
	// nil означает «в разделе нет ни одного значения», а не ноль: у фильтра
	// без данных подсказки быть не должно.
	NumberMin *decimal.Decimal
	NumberMax *decimal.Decimal
}

// Attributes список характеристик.
type Attributes []*Attribute

// Validate проверяет пригодность характеристики к сохранению.
func (a *Attribute) Validate() error {
	if a == nil {
		return ErrNilPointer
	}

	if !a.DataType.IsValid() {
		return ErrAttributeTypeInvalid
	}

	// Формат, а не только непустота: тот же формат проверяет CHECK
	// attributes_code_format, и без проверки на входе нарушение доходило бы
	// до базы и возвращалось клиенту внутренней ошибкой вместо понятного 400.
	if !IsValidCode(a.Code) {
		return ErrAttributeCodeInvalid
	}

	// Раньше имя характеристики не проверялось вовсе, и безымянная
	// характеристика доходила до фильтров пустой подписью над парой полей.
	// Теперь правило одно на все сущности: русское обязательно, остальные
	// пусты до перевода — и витрине всегда есть на что откатиться.
	if a.NameRU == "" {
		return ErrAttributeNameRequired
	}

	// Единица измерения осмысленна только у числовой характеристики:
	// «тип привода: гусеничный, кВт» — ошибка ввода, а не редкий случай.
	// То же ограничение стоит CHECK-ом attributes_unit_number_only.
	if a.Unit != nil && a.DataType != AttributeTypeNumber {
		return ErrAttributeUnitNotAllowed
	}

	if a.Position < 0 {
		return ErrPositionInvalid
	}

	// Варианты проверяются здесь, а не там, где они пишутся: их запись —
	// отдельный запрос, но приходят они внутри характеристики, и проверка
	// обязана случиться до того, как характеристика уже сохранена.
	return a.Options.Validate()
}

// HasOption сообщает, что среди вариантов характеристики есть код.
// Существование кода проверяет домен и сервис: внешнего ключа на
// attribute_options у значения нет — см. COMMENT ON TABLE machine_attribute_values.
func (a *Attribute) HasOption(code string) bool {
	if a == nil {
		return false
	}

	for _, option := range a.Options {
		if option.Code == code {
			return true
		}
	}

	return false
}

// AttributeOption вариант значения для характеристики типа enum.
type AttributeOption struct {
	ID          string    // Идентификатор
	AttributeID string    // Характеристика, которой принадлежит вариант
	Code        string    // Код варианта
	NameRU      string    // Название на русском
	NameKK      string    // Название на казахском
	NameEN      string    // Название на английском
	Position    int       // Порядок вывода
	CreatedAt   time.Time // Дата создания
}

// AttributeOptions список вариантов.
type AttributeOptions []*AttributeOption

// Validate проверяет пригодность варианта к сохранению.
func (o *AttributeOption) Validate() error {
	if o == nil {
		return ErrNilPointer
	}

	// Тот же формат стоит CHECK-ом attribute_options_code_format.
	if !IsValidCode(o.Code) {
		return ErrAttributeOptionCodeInvalid
	}

	if o.NameRU == "" {
		return ErrAttributeOptionNameRequired
	}

	if o.Position < 0 {
		return ErrPositionInvalid
	}

	return nil
}

// Validate проверяет пригодность набора вариантов к сохранению.
//
// Коды сверяются на повторы: в схеме это UNIQUE (attribute_id, code),
// и без проверки здесь повтор дошёл бы до базы отказом уникальности,
// который клиенту ничего не объясняет.
func (o AttributeOptions) Validate() error {
	seen := make(map[string]struct{}, len(o))

	for _, option := range o {
		if err := option.Validate(); err != nil {
			return err
		}

		if _, duplicate := seen[option.Code]; duplicate {
			return ErrAttributeOptionDuplicate
		}

		seen[option.Code] = struct{}{}
	}

	return nil
}

// MachineAvailability наличие машины.
type MachineAvailability string

// Варианты наличия. Совпадают с CHECK machines_availability_allowed.
const (
	AvailabilityInStock MachineAvailability = "in_stock" // На складе
	AvailabilityOnOrder MachineAvailability = "on_order" // Под заказ
)

// String возвращает строковое представление наличия.
func (a MachineAvailability) String() string { return string(a) }

// IsValid сообщает, что наличие известно домену.
func (a MachineAvailability) IsValid() bool {
	return a == AvailabilityInStock || a == AvailabilityOnOrder
}

// MachineCondition состояние машины.
type MachineCondition string

// Варианты состояния. Совпадают с CHECK machines_condition_allowed.
const (
	ConditionNew  MachineCondition = "new"  // Новая
	ConditionUsed MachineCondition = "used" // Бывшая в употреблении
)

// String возвращает строковое представление состояния.
func (c MachineCondition) String() string { return string(c) }

// IsValid сообщает, что состояние известно домену.
func (c MachineCondition) IsValid() bool {
	return c == ConditionNew || c == ConditionUsed
}

// Machine карточка единицы техники в каталоге.
type Machine struct {
	ID             string              // Идентификатор
	CategoryID     string              // Категория
	Brand          string              // Бренд
	Model          string              // Модель
	Slug           string              // Идентификатор в URL
	NameRU         string              // Название на русском
	NameKK         string              // Название на казахском
	NameEN         string              // Название на английском
	DescriptionRU  *string             // Описание на русском
	DescriptionKK  *string             // Описание на казахском
	DescriptionEN  *string             // Описание на английском
	PriceCNYCents  int64               // Цена в фынях (1/100 юаня)
	IsPricePublic  bool                // Показывать ли цену
	Availability   MachineAvailability // Наличие
	DeliveryWeeks  *int                // Срок поставки в неделях
	ProductionYear *int                // Год выпуска
	Condition      MachineCondition    // Состояние

	// FuelConsumptionLPH, ServiceIntervalHours и ServiceCostCNYCents —
	// величины для расчёта стоимости владения.
	//
	// Указатели, и это существенно: nil означает «не заведено», и расчёт
	// обязан сказать об этом словами, а не показать ноль. Ноль здесь имел бы
	// собственный смысл — «не расходует топлива», — и путать его с незнанием
	// нельзя.
	FuelConsumptionLPH   *decimal.Decimal
	ServiceIntervalHours *int
	ServiceCostCNYCents  *int64

	IsActive    bool       // Видимость в публичном каталоге
	PublishedAt *time.Time // Момент первой публикации
	CreatedAt   time.Time  // Дата создания
	UpdatedAt   time.Time  // Дата изменения

	AttributeValues MachineAttributeValues // Значения характеристик
	Images          MachineImages          // Фотографии

	// Relevance релевантность запросу, заполняется только выдачей поиска.
	//
	// Не колонка и не свойство машины: величина существует лишь в паре
	// с конкретным запросом. Живёт здесь потому, что курсор следующей
	// страницы собирается из последней строки выдачи, а собирать его
	// иначе значило бы возвращать выдачу двумя разными типами.
	Relevance int

	// PrimaryImage главное фото карточки, дочитанное выдачей поиска.
	//
	// Отдельное поле, а не первый элемент Images: списку нужна одна
	// фотография, и Images в выдаче остаётся пустым. Иначе одно и то же поле
	// значило бы «вся галерея» на карточке и «только главное» в списке —
	// и клиент не мог бы отличить карточку с единственным фото от урезанной.
	PrimaryImage *MachineImage
}

// Machines список карточек.
type Machines []*Machine

// Validate проверяет пригодность карточки к сохранению.
func (m *Machine) Validate() error {
	if m == nil {
		return ErrNilPointer
	}

	if err := m.validateIdentity(); err != nil {
		return err
	}

	if err := m.validateSupply(); err != nil {
		return err
	}

	// Опубликованная карточка обязана иметь дату публикации: витрина
	// сортируется по published_at, и NULL в ней означал бы машину, которая
	// видна в каталоге, но не встаёт в ленту. Тот же инвариант стоит CHECK-ом
	// machines_active_requires_published_at.
	if m.IsActive && m.PublishedAt == nil {
		return ErrMachinePublishedAtRequired
	}

	return nil
}

// validateIdentity проверяет то, чем карточка представлена снаружи.
func (m *Machine) validateIdentity() error {
	if !IsValidSlug(m.Slug) {
		return ErrSlugInvalid
	}

	if m.Brand == "" || m.Model == "" {
		return ErrMachineBrandModelRequired
	}

	// Обязателен только русский — см. Category.Validate.
	if m.NameRU == "" {
		return ErrMachineNameRequired
	}

	return nil
}

// validateSupply проверяет цену и условия поставки.
// Вынесено из Validate: вместе набор ветвей читается как сплошной список,
// в котором легко потерять одну при правке.
func (m *Machine) validateSupply() error {
	if m.PriceCNYCents < 0 {
		return ErrMachinePriceInvalid
	}

	if !m.Availability.IsValid() {
		return ErrMachineAvailabilityInvalid
	}

	if !m.Condition.IsValid() {
		return ErrMachineConditionInvalid
	}

	if m.DeliveryWeeks != nil && *m.DeliveryWeeks <= 0 {
		return ErrMachineDeliveryWeeksInvalid
	}

	if m.ProductionYear != nil && (*m.ProductionYear < minProductionYear || *m.ProductionYear > maxProductionYear) {
		return ErrMachineProductionYearInvalid
	}

	return nil
}

// Границы года выпуска. Совпадают с CHECK machines_production_year_sane.
const (
	minProductionYear = 1950
	maxProductionYear = 2100
)

// Publish публикует карточку, проставляя дату первой публикации.
//
// Повторная публикация дату не сбрасывает: published_at задаёт порядок ленты,
// и возвращение снятой карточки не должно поднимать её наверх как новинку.
func (m *Machine) Publish(currentTime time.Time) {
	if m == nil {
		return
	}

	m.IsActive = true

	if m.PublishedAt == nil {
		m.PublishedAt = Pointer(currentTime)
	}
}

// Unpublish снимает карточку с публикации, сохраняя дату первой публикации.
func (m *Machine) Unpublish() {
	if m == nil {
		return
	}

	m.IsActive = false
}

// IsPubliclyVisible сообщает, что карточку можно показывать на витрине.
func (m *Machine) IsPubliclyVisible() bool {
	return m != nil && m.IsActive && m.PublishedAt != nil
}

// MachineImage фотография машины.
type MachineImage struct {
	ID        string    // Идентификатор
	MachineID string    // Машина
	Path      string    // Путь в файловом хранилище
	Position  int       // Порядок показа
	IsPrimary bool      // Главное фото
	AltRU     *string   // Подпись на русском
	AltKK     *string   // Подпись на казахском
	AltEN     *string   // Подпись на английском
	CreatedAt time.Time // Дата создания
}

// MachineImages список фотографий.
type MachineImages []*MachineImage

// Validate проверяет набор фотографий целиком.
//
// Проверяется весь набор, а не каждая по отдельности: «не больше одного
// главного фото» — свойство набора, и на одной строке его не увидеть.
// В БД это же ограничение держит частичный уникальный индекс
// machine_images_primary_uidx.
func (images MachineImages) Validate() error {
	primaries := 0

	for _, image := range images {
		if image == nil {
			return ErrNilPointer
		}

		if image.Path == "" {
			return ErrMachineImagePathRequired
		}

		if image.Position < 0 {
			return ErrPositionInvalid
		}

		if image.IsPrimary {
			primaries++
		}
	}

	if primaries > 1 {
		return ErrMachineImagePrimaryConflict
	}

	return nil
}

// Primary возвращает главное фото набора или nil.
func (images MachineImages) Primary() *MachineImage {
	for _, image := range images {
		if image != nil && image.IsPrimary {
			return image
		}
	}

	return nil
}

// MachineAttributeValue значение характеристики машины.
//
// Значение хранится типизированным (см. AttributeValue): комбинация «тип
// number с заполненной строкой» здесь не выражается в принципе, а не
// отвергается проверкой постфактум.
type MachineAttributeValue struct {
	ID          string         // Идентификатор
	MachineID   string         // Машина
	AttributeID string         // Характеристика
	Value       AttributeValue // Типизированное значение
}

// MachineAttributeValues список значений характеристик.
type MachineAttributeValues []*MachineAttributeValue

// Validate проверяет, что у каждого значения заполнена характеристика и само значение.
func (values MachineAttributeValues) Validate() error {
	for _, value := range values {
		if value == nil || value.Value == nil {
			return ErrNilPointer
		}

		if value.AttributeID == "" {
			return ErrAttributeValueInvalid
		}
	}

	return nil
}

// AttributeValue типизированное значение характеристики.
//
// Замкнутый набор реализаций: NumberValue, StringValue, BoolValue, EnumValue.
// Приватный метод-маркер закрывает набор снаружи пакета, поэтому «значение
// неизвестного типа» не существует, а каждая реализация несёт ровно одно поле
// своего типа — рассогласование типа и заполненной колонки невыразимо.
//
// Разбор обратно в колонки БД делает адаптер хранилища одним переключателем
// по DataType, и CHECK machine_attribute_values_type_matches_column
// оказывается не проверкой, а следствием формы типа.
type AttributeValue interface {
	// DataType возвращает тип значения.
	DataType() AttributeDataType
	// isAttributeValue закрывает набор реализаций границами пакета.
	isAttributeValue()
}

// AttributeNumberScale знаков после запятой у числовой характеристики.
// Совпадает с numeric(14,3) в machine_attribute_values.num_value.
const AttributeNumberScale = 3

// NumberValue числовое значение характеристики.
type NumberValue struct {
	Number decimal.Decimal // Значение в единице, объявленной характеристикой
}

// NewNumberValue создаёт числовое значение.
func NewNumberValue(number decimal.Decimal) NumberValue { return NumberValue{Number: number} }

// DataType возвращает тип значения.
func (NumberValue) DataType() AttributeDataType { return AttributeTypeNumber }

// String возвращает значение строкой фиксированной точности.
//
// Единственная точка перевода числа в строку для хранения — по образцу
// AppliedFXRate.String(). Фиксированный хвост, а не decimal.String(): без него
// «155» и «155.000» — одно значение в двух написаниях, и записанное перестаёт
// совпадать с прочитанным при строковом сравнении. Округление до scale делает
// не PostgreSQL при вставке, а домен, поэтому записанное известно до похода
// в базу.
func (v NumberValue) String() string {
	return v.Number.StringFixed(AttributeNumberScale)
}

func (NumberValue) isAttributeValue() {}

// StringValue строковое значение характеристики.
type StringValue struct {
	Text string // Значение
}

// NewStringValue создаёт строковое значение.
func NewStringValue(text string) StringValue { return StringValue{Text: text} }

// DataType возвращает тип значения.
func (StringValue) DataType() AttributeDataType { return AttributeTypeString }

func (StringValue) isAttributeValue() {}

// BoolValue флаговое значение характеристики.
type BoolValue struct {
	Flag bool // Значение
}

// NewBoolValue создаёт флаговое значение.
func NewBoolValue(flag bool) BoolValue { return BoolValue{Flag: flag} }

// DataType возвращает тип значения.
func (BoolValue) DataType() AttributeDataType { return AttributeTypeBool }

func (BoolValue) isAttributeValue() {}

// EnumValue значение-вариант характеристики.
//
// Отдельный тип, а не StringValue: код варианта обязан существовать
// в attribute_options, и отличать его от свободной строки нужно до того,
// как значение уйдёт в хранилище. В колонках БД оба лежат в text_value.
type EnumValue struct {
	Code string // Код варианта из attribute_options
}

// NewEnumValue создаёт значение-вариант.
func NewEnumValue(code string) EnumValue { return EnumValue{Code: code} }

// DataType возвращает тип значения.
func (EnumValue) DataType() AttributeDataType { return AttributeTypeEnum }

func (EnumValue) isAttributeValue() {}

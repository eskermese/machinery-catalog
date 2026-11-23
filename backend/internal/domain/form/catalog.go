package form

import (
	"errors"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// MachineSearch форма поиска по каталогу.
//
// Приходит телом POST, а не query: набор условий по характеристикам
// переменный и вложенный, и в строке запроса он превратился бы в разбор
// повторяющихся параметров вручную. Ссылочность выдачи это не ломает —
// см. комментарий к маршруту поиска.
type MachineSearch struct {
	// CategoryID ограничивает выдачу категорией.
	CategoryID string `json:"categoryID" validate:"omitempty,uuid4" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	// IncludeSubcategories расширяет отбор на всё поддерево категории.
	IncludeSubcategories bool `json:"includeSubcategories" example:"true"`

	// PriceMinCents нижняя граница цены в фынях (1/100 юаня) включительно.
	PriceMinCents *int64 `json:"priceMinCents" validate:"omitempty,gte=0" example:"5000000"`
	// PriceMaxCents верхняя граница цены в фынях (1/100 юаня) включительно.
	PriceMaxCents *int64 `json:"priceMaxCents" validate:"omitempty,gte=0" example:"15000000"`

	// Availability наличие: in_stock или on_order.
	Availability string `json:"availability" validate:"omitempty,oneof=in_stock on_order" example:"in_stock"`
	// Condition состояние: new или used.
	Condition string `json:"condition" validate:"omitempty,oneof=new used" example:"new"`

	// ProductionYearMin нижняя граница года выпуска включительно.
	ProductionYearMin *int `json:"productionYearMin" validate:"omitempty,gte=1950,lte=2100" example:"2018"`
	// ProductionYearMax верхняя граница года выпуска включительно.
	ProductionYearMax *int `json:"productionYearMax" validate:"omitempty,gte=1950,lte=2100" example:"2026"`

	// Attributes условия по характеристикам, произвольный набор.
	Attributes []AttributeFilter `json:"attributes" validate:"omitempty,dive"`

	// Sort порядок сортировки: price_asc, price_desc или published_desc.
	Sort string `json:"sort" validate:"omitempty,oneof=price_asc price_desc published_desc" example:"price_asc"`
	// Cursor курсор следующей страницы, полученный в предыдущем ответе.
	Cursor string `json:"cursor" example:""`
	// Limit размер страницы. Значение сверх допустимого урезается до максимума.
	Limit int `json:"limit" validate:"omitempty,gte=0" example:"20"`
	// Query подстрока для поиска по названию, бренду и модели.
	//
	// Одно поле и для витрины, и для админки: ищутся одни и те же колонки,
	// а видимость черновиков решает не оно.
	Query string `json:"query" validate:"omitempty,max=200" example:"экскаватор"`

	// IsActive отбирает опубликованные (true) или черновики (false).
	//
	// На публичной витрине игнорируется: там признак уже зафиксирован,
	// и переопределить его параметром запроса нельзя.
	IsActive *bool `json:"isActive" example:"false"`

	// WithTotal просить ли общее число найденных карточек.
	//
	// Стоит второго запроса по тому же предикату, поэтому по умолчанию
	// выключено: бесконечной прокрутке итог не нужен.
	WithTotal bool `json:"withTotal" example:"false"`
}

// AttributeFilter условие по одной характеристике.
//
// Вид условия выбирается по заполненным полям: числовые границы, набор кодов
// варианта либо флаг. Заполнять несколько видов сразу бессмысленно — форма
// отвергает такое как неоднозначное, а не молча выбирает одно из них.
type AttributeFilter struct {
	// AttributeID характеристика, к которой применяется условие.
	AttributeID string `json:"attributeID" validate:"required,uuid4" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`

	// NumberMin нижняя граница числовой характеристики включительно.
	NumberMin *string `json:"numberMin" example:"150"`
	// NumberMax верхняя граница числовой характеристики включительно.
	NumberMax *string `json:"numberMax" example:"250.5"`

	// Codes допустимые коды варианта; несколько означают «любой из».
	Codes []string `json:"codes" example:"crawler"`

	// Flag требуемое значение флаговой характеристики.
	Flag *bool `json:"flag" example:"true"`
}

// Validate проверяет форму поиска и приводит её к доменному фильтру.
func (f *MachineSearch) Validate() error {
	if err := validatorDefault.Validate(f); err != nil {
		return err
	}

	for i := range f.Attributes {
		if err := f.Attributes[i].validate(); err != nil {
			return err
		}
	}

	return nil
}

// validate проверяет, что условие задаёт ровно один вид отбора.
func (f *AttributeFilter) validate() error {
	kinds := 0

	if f.NumberMin != nil || f.NumberMax != nil {
		kinds++
	}

	if len(f.Codes) > 0 {
		kinds++
	}

	if f.Flag != nil {
		kinds++
	}

	if kinds != 1 {
		return entity.ErrAttributeValueInvalid
	}

	return nil
}

// ToFilter собирает доменный фильтр из формы.
//
// Границы страницы и признак публичности здесь не выставляются: лимит
// урезает вызывающий ресурс, а OnlyPublished — сервис, и дублировать это
// в форме значило бы завести второе место, где решается видимость черновиков.
func (f *MachineSearch) ToFilter() (*entity.MachineFilter, error) {
	filter := &entity.MachineFilter{
		IncludeSubcategories: f.IncludeSubcategories,
		PriceMinCents:        f.PriceMinCents,
		PriceMaxCents:        f.PriceMaxCents,
		ProductionYearMin:    f.ProductionYearMin,
		ProductionYearMax:    f.ProductionYearMax,
		Query:                strings.TrimSpace(f.Query),
		IsActive:             f.IsActive,
		Sort:                 entity.MachineSort(f.Sort),
		Limit:                f.Limit,
		WithTotal:            f.WithTotal,
	}

	if f.CategoryID != "" {
		filter.CategoryID = entity.Pointer(f.CategoryID)
	}

	if f.Availability != "" {
		filter.Availability = entity.Pointer(entity.MachineAvailability(f.Availability))
	}

	if f.Condition != "" {
		filter.Condition = entity.Pointer(entity.MachineCondition(f.Condition))
	}

	cursor, err := entity.DecodeMachineCursor(f.Cursor)
	if err != nil {
		return nil, err
	}

	filter.Cursor = cursor

	// Курсор несёт свою сортировку: если клиент её не передал, берём из курсора,
	// иначе первая же следующая страница отвергнется как несовместимая.
	if f.Sort == "" && cursor != nil {
		filter.Sort = cursor.Sort
	}

	if filter.Attributes, err = f.attributeConditions(); err != nil {
		return nil, err
	}

	return filter, nil
}

// attributeConditions собирает условия по характеристикам.
func (f *MachineSearch) attributeConditions() ([]entity.AttributeCondition, error) {
	if len(f.Attributes) == 0 {
		return nil, nil
	}

	conditions := make([]entity.AttributeCondition, 0, len(f.Attributes))

	for i := range f.Attributes {
		condition, err := f.Attributes[i].toCondition()
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, condition)
	}

	return conditions, nil
}

// toCondition приводит условие формы к доменному.
func (f *AttributeFilter) toCondition() (entity.AttributeCondition, error) {
	switch {
	case f.NumberMin != nil || f.NumberMax != nil:
		minValue, err := parseOptionalDecimal(f.NumberMin)
		if err != nil {
			return nil, err
		}

		maxValue, err := parseOptionalDecimal(f.NumberMax)
		if err != nil {
			return nil, err
		}

		return entity.NewNumberRangeCondition(f.AttributeID, minValue, maxValue), nil
	case len(f.Codes) > 0:
		return entity.NewEnumCondition(f.AttributeID, f.Codes...), nil
	case f.Flag != nil:
		return entity.NewBoolCondition(f.AttributeID, *f.Flag), nil
	default:
		return nil, entity.ErrAttributeValueInvalid
	}
}

// parseOptionalDecimal разбирает необязательное десятичное значение.
//
// Границы диапазона приходят строкой, а не числом: JSON-число прошло бы через
// float64 и потеряло бы младшие разряды, ради точности которых характеристики
// объявлены numeric.
func parseOptionalDecimal(raw *string) (*decimal.Decimal, error) {
	if raw == nil {
		return nil, nil //nolint:nilnil // отсутствие границы — не отказ
	}

	parsed, err := decimal.NewFromString(strings.TrimSpace(*raw))
	if err != nil {
		return nil, entity.ErrAttributeValueInvalid
	}

	return &parsed, nil
}

// CategorySave форма создания и изменения категории.
type CategorySave struct {
	// ParentID родительская категория; пусто — корень.
	ParentID string `json:"parentID" validate:"omitempty,uuid4" example:""`
	// Slug идентификатор в URL.
	Slug string `json:"slug" validate:"required" example:"excavators"`
	// NameRU название на русском.
	NameRU string `json:"nameRU" validate:"required" example:"Экскаваторы"`
	// NameKK и NameEN названия на казахском и английском.
	//
	// НЕОБЯЗАТЕЛЬНЫ, в отличие от русского. Русский — источник правды:
	// витрина показывает его, пока перевода нет. Требовать перевод здесь
	// значило бы запретить заводить раздел до того, как его перевели,
	// а заглушка вместо перевода хуже пустоты — витрина не отличит её
	// от настоящего названия.
	NameKK string `json:"nameKK" example:"Экскаваторлар"`
	NameEN string `json:"nameEN" example:"Excavators"`
	// Position порядок вывода.
	Position int `json:"position" validate:"gte=0" example:"1"`
	// IsActive видимость ветки на сайте.
	IsActive bool `json:"isActive" example:"true"`
	// ImagePath обложка раздела: путь в хранилище из ответа
	// POST /v1/admin/media/categories. Пусто — обложку подставит сервис
	// по первой опубликованной машине поддерева.
	ImagePath string `json:"imagePath" example:"categories/7c9e6679.jpg"`
}

// Validate проверяет форму категории.
func (f *CategorySave) Validate() error {
	return validatorDefault.Validate(f)
}

// ToEntity собирает категорию из формы.
func (f *CategorySave) ToEntity(id string) *entity.Category {
	category := &entity.Category{
		ID:       id,
		Slug:     f.Slug,
		NameRU:   f.NameRU,
		NameKK:   f.NameKK,
		NameEN:   f.NameEN,
		Position: f.Position,
		IsActive: f.IsActive,
	}

	if f.ParentID != "" {
		category.ParentID = entity.Pointer(f.ParentID)
	}

	// Пустая строка — это «обложки нет», а не «оставить как было»: форма
	// описывает категорию целиком, и пропущенное поле означает сброс.
	if f.ImagePath != "" {
		category.ImagePath = entity.Pointer(f.ImagePath)
	}

	return category
}

// AttributeSave форма создания и изменения характеристики.
type AttributeSave struct {
	// CategoryID категория, которой принадлежит характеристика.
	CategoryID string `json:"categoryID" validate:"required,uuid4" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	// Code машинный ключ характеристики.
	Code string `json:"code" validate:"required" example:"engine_power"`
	// NameRU название на русском.
	NameRU string `json:"nameRU" validate:"required" example:"Мощность"`
	// NameKK название на казахском.
	NameKK string `json:"nameKK" example:"Қуаты"`
	// NameEN название на английском.
	NameEN string `json:"nameEN" example:"Engine power"`
	// DataType тип значения: number, string, bool или enum.
	DataType string `json:"dataType" validate:"required,oneof=number string bool enum" example:"number"`
	// Unit единица измерения; допустима только у числовой характеристики.
	Unit string `json:"unit" example:"кВт"`
	// IsFilterable выносить ли характеристику в панель фильтров.
	IsFilterable bool `json:"isFilterable" example:"true"`
	// Position порядок вывода.
	Position int `json:"position" validate:"gte=0" example:"1"`
	// Options варианты для типа enum.
	Options []AttributeOptionSave `json:"options" validate:"omitempty,dive"`
}

// AttributeOptionSave форма варианта характеристики.
type AttributeOptionSave struct {
	// Code код варианта.
	Code string `json:"code" validate:"required" example:"crawler"`
	// NameRU название на русском.
	NameRU string `json:"nameRU" validate:"required" example:"Гусеничный"`
	// NameKK название на казахском.
	NameKK string `json:"nameKK" example:"Шынжыр табанды"`
	// NameEN название на английском.
	NameEN string `json:"nameEN" example:"Crawler"`
	// Position порядок вывода.
	Position int `json:"position" validate:"gte=0" example:"1"`
}

// Validate проверяет форму характеристики.
func (f *AttributeSave) Validate() error {
	return validatorDefault.Validate(f)
}

// ToEntity собирает характеристику из формы.
func (f *AttributeSave) ToEntity(id string) *entity.Attribute {
	attribute := &entity.Attribute{
		ID:           id,
		CategoryID:   f.CategoryID,
		Code:         f.Code,
		NameRU:       f.NameRU,
		NameKK:       f.NameKK,
		NameEN:       f.NameEN,
		DataType:     entity.AttributeDataType(f.DataType),
		IsFilterable: f.IsFilterable,
		Position:     f.Position,
	}

	if f.Unit != "" {
		attribute.Unit = entity.Pointer(f.Unit)
	}

	for _, option := range f.Options {
		attribute.Options = append(attribute.Options, &entity.AttributeOption{
			Code:     option.Code,
			NameRU:   option.NameRU,
			NameKK:   option.NameKK,
			NameEN:   option.NameEN,
			Position: option.Position,
		})
	}

	return attribute
}

// MachineSave форма создания и изменения карточки техники.
type MachineSave struct {
	// CategoryID категория карточки.
	CategoryID string `json:"categoryID" validate:"required,uuid4" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	// Brand бренд.
	Brand string `json:"brand" validate:"required" example:"XCMG"`
	// Model модель.
	Model string `json:"model" validate:"required" example:"XE215C"`
	// Slug идентификатор в URL.
	Slug string `json:"slug" validate:"required" example:"xcmg-xe215c"`
	// NameRU название на русском.
	NameRU string `json:"nameRU" validate:"required" example:"Экскаватор XCMG XE215C"`
	// NameKK название на казахском.
	NameKK string `json:"nameKK" example:"XCMG XE215C экскаваторы"`
	// NameEN название на английском.
	NameEN string `json:"nameEN" example:"XCMG XE215C excavator"`
	// DescriptionRU описание на русском.
	DescriptionRU string `json:"descriptionRU" example:""`
	// DescriptionKK описание на казахском.
	DescriptionKK string `json:"descriptionKK" example:""`
	// DescriptionEN описание на английском.
	DescriptionEN string `json:"descriptionEN" example:""`

	// PriceCNYCents цена в фынях (1/100 юаня).
	//
	// Целое в центах, а не доллары строкой или числом: на границе не должно
	// быть места для потери точности, а «89500.00» пришлось бы разбирать
	// и округлять — то есть заводить ровно тот шаг, где точность и теряется.
	PriceCNYCents int64 `json:"priceCnyCents" validate:"gte=0" example:"8950000"`
	// IsPricePublic показывать ли цену на витрине.
	IsPricePublic bool `json:"isPricePublic" example:"true"`

	// Availability наличие: in_stock или on_order.
	Availability string `json:"availability" validate:"required,oneof=in_stock on_order" example:"in_stock"`
	// DeliveryWeeks срок поставки в неделях.
	DeliveryWeeks *int `json:"deliveryWeeks" validate:"omitempty,gt=0" example:"8"`
	// ProductionYear год выпуска.
	ProductionYear *int `json:"productionYear" validate:"omitempty,gte=1950,lte=2100" example:"2024"`
	// Condition состояние: new или used.
	Condition string `json:"condition" validate:"required,oneof=new used" example:"new"`

	// FuelConsumptionLph расход топлива в литрах на моточас.
	//
	// Строкой, а не числом: расход паспортный и дробный («12.7»), и float64
	// на границе контракта — лишний повод для расхождения в младшем разряде.
	// Пусто — не заведён, и расчёт стоимости владения так и скажет.
	FuelConsumptionLph *string `json:"fuelConsumptionLph" example:"12.7"`
	// ServiceIntervalHours моточасов между плановыми ТО; пусто — не заведён.
	ServiceIntervalHours *int `json:"serviceIntervalHours" validate:"omitempty,gt=0" example:"500"`
	// ServiceCostCnyCents стоимость одного планового ТО в фынях (1/100 юаня);
	// пусто — не заведена.
	ServiceCostCnyCents *int64 `json:"serviceCostCnyCents" validate:"omitempty,gte=0" example:"120000"`

	// AttributeValues значения характеристик.
	AttributeValues []MachineAttributeValueSave `json:"attributeValues" validate:"omitempty,dive"`
	// Images фотографии карточки.
	Images []MachineImageSave `json:"images" validate:"omitempty,dive"`
}

// MachineAttributeValueSave форма значения характеристики.
//
// Вид значения выбирается по заполненному полю — ровно одному. Так же устроено
// и доменное значение: комбинация «тип number со строкой» не выражается.
type MachineAttributeValueSave struct {
	// AttributeID характеристика.
	AttributeID string `json:"attributeID" validate:"required,uuid4" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	// Number числовое значение строкой, чтобы не терять разряды на float64.
	Number *string `json:"number" example:"155"`
	// Text строковое значение.
	Text *string `json:"text" example:""`
	// Code код варианта для характеристики типа enum.
	Code *string `json:"code" example:"crawler"`
	// Flag значение флаговой характеристики.
	Flag *bool `json:"flag" example:"true"`
}

// MachineImageSave форма фотографии карточки.
type MachineImageSave struct {
	// Path путь в файловом хранилище.
	Path string `json:"path" validate:"required" example:"machines/xcmg-xe215c-1.jpg"`
	// Position порядок показа.
	Position int `json:"position" validate:"gte=0" example:"0"`
	// IsPrimary главное фото; допустимо не более одного на карточку.
	IsPrimary bool `json:"isPrimary" example:"true"`
	// AltRU подпись на русском.
	AltRU string `json:"altRU" example:""`
	// AltKK подпись на казахском.
	AltKK string `json:"altKK" example:""`
	// AltEN подпись на английском.
	AltEN string `json:"altEN" example:""`
}

// Validate проверяет форму карточки.
func (f *MachineSave) Validate() error {
	if err := validatorDefault.Validate(f); err != nil {
		return err
	}

	for i := range f.AttributeValues {
		if err := f.AttributeValues[i].validate(); err != nil {
			return err
		}
	}

	return nil
}

// validate проверяет, что значение задано ровно одним полем.
func (f *MachineAttributeValueSave) validate() error {
	filled := 0

	for _, isSet := range []bool{f.Number != nil, f.Text != nil, f.Code != nil, f.Flag != nil} {
		if isSet {
			filled++
		}
	}

	if filled != 1 {
		return entity.ErrAttributeValueInvalid
	}

	return nil
}

// ToEntity собирает карточку из формы.
func (f *MachineSave) ToEntity(id string) (*entity.Machine, error) {
	machine := &entity.Machine{
		ID:             id,
		CategoryID:     f.CategoryID,
		Brand:          f.Brand,
		Model:          f.Model,
		Slug:           f.Slug,
		NameRU:         f.NameRU,
		NameKK:         f.NameKK,
		NameEN:         f.NameEN,
		DescriptionRU:  optionalString(f.DescriptionRU),
		DescriptionKK:  optionalString(f.DescriptionKK),
		DescriptionEN:  optionalString(f.DescriptionEN),
		PriceCNYCents:  f.PriceCNYCents,
		IsPricePublic:  f.IsPricePublic,
		Availability:   entity.MachineAvailability(f.Availability),
		DeliveryWeeks:  f.DeliveryWeeks,
		ProductionYear: f.ProductionYear,
		Condition:      entity.MachineCondition(f.Condition),

		ServiceIntervalHours: f.ServiceIntervalHours,
		ServiceCostCNYCents:  f.ServiceCostCnyCents,
	}

	// Расход разбирается здесь, а не проверяется тегом: тег умеет требовать
	// число, но не умеет превратить строку в decimal, а тащить разбор ниже
	// значило бы пустить в домен непроверенную строку.
	consumption, err := optionalDecimal(f.FuelConsumptionLph)
	if err != nil {
		return nil, entity.ErrMachineFuelConsumptionInvalid
	}

	machine.FuelConsumptionLPH = consumption

	values, err := f.attributeValues()
	if err != nil {
		return nil, err
	}

	machine.AttributeValues = values
	machine.Images = f.images()

	return machine, nil
}

// attributeValues собирает значения характеристик.
func (f *MachineSave) attributeValues() (entity.MachineAttributeValues, error) {
	values := make(entity.MachineAttributeValues, 0, len(f.AttributeValues))

	for i := range f.AttributeValues {
		value, err := f.AttributeValues[i].toEntity()
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

// toEntity приводит значение формы к доменному.
func (f *MachineAttributeValueSave) toEntity() (*entity.MachineAttributeValue, error) {
	value := &entity.MachineAttributeValue{AttributeID: f.AttributeID}

	switch {
	case f.Number != nil:
		parsed, err := decimal.NewFromString(strings.TrimSpace(*f.Number))
		if err != nil {
			return nil, entity.ErrAttributeValueInvalid
		}

		value.Value = entity.NewNumberValue(parsed)
	case f.Text != nil:
		value.Value = entity.NewStringValue(*f.Text)
	case f.Code != nil:
		value.Value = entity.NewEnumValue(*f.Code)
	case f.Flag != nil:
		value.Value = entity.NewBoolValue(*f.Flag)
	default:
		return nil, entity.ErrAttributeValueInvalid
	}

	return value, nil
}

// images собирает фотографии карточки.
func (f *MachineSave) images() entity.MachineImages {
	images := make(entity.MachineImages, 0, len(f.Images))

	for _, image := range f.Images {
		images = append(images, &entity.MachineImage{
			Path:      image.Path,
			Position:  image.Position,
			IsPrimary: image.IsPrimary,
			AltRU:     optionalString(image.AltRU),
			AltKK:     optionalString(image.AltKK),
			AltEN:     optionalString(image.AltEN),
		})
	}

	return images
}

// MachineImagesReplace форма замены набора фотографий.
type MachineImagesReplace struct {
	// Images новый набор фотографий; пустой набор очищает галерею.
	Images []MachineImageSave `json:"images" validate:"omitempty,dive"`
}

// Validate проверяет форму замены фотографий.
func (f *MachineImagesReplace) Validate() error {
	return validatorDefault.Validate(f)
}

// ToEntity собирает набор фотографий из формы.
func (f *MachineImagesReplace) ToEntity() entity.MachineImages {
	save := MachineSave{Images: f.Images}

	return save.images()
}

// optionalString приводит пустую строку к отсутствующему значению.
// Пустое описание и отсутствующее описание — одно и то же, и хранить
// пустую строку значило бы отличать их без причины.
func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return entity.Pointer(value)
}

// optionalDecimal разбирает необязательное дробное число.
//
// Пусто и «не передано» — одно и то же: nil означает «не заведено», и это
// состояние первого класса, а не ошибка.
func optionalDecimal(value *string) (*decimal.Decimal, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil //nolint:nilnil // «не заведено» — не ошибка
	}

	parsed, err := decimal.NewFromString(strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}

	if !parsed.IsPositive() {
		return nil, errNonPositiveDecimal
	}

	return &parsed, nil
}

// errNonPositiveDecimal внутренняя ошибка разбора: наружу уходит доменная.
var errNonPositiveDecimal = errors.New("значение должно быть положительным")

// ParseLimit разбирает размер страницы из строки запроса.
//
// Значение сверх максимума урезается, а не отвергается: клиент, попросивший
// десять тысяч карточек, ошибся в масштабе, а не в синтаксисе — отдать ему
// максимум полезнее, чем отказ. Нечитаемое значение при этом остаётся отказом:
// «limit=много» — это опечатка, и молча подставлять умолчание значило бы
// скрыть её.
func ParseLimit(raw string) (int, error) {
	if raw == "" {
		return entity.DefaultMachineLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		return 0, entity.ErrCatalogLimitInvalid
	}

	if limit == 0 {
		return entity.DefaultMachineLimit, nil
	}

	return min(limit, entity.MaxMachineLimit), nil
}

// ResidualValueCurveSave форма кривой остаточной стоимости раздела.
//
// Кривая целиком, а не строка на год: тридцать чисел на раздел и больше
// тысячи на каталог руками не заводят, а наполовину заполненная таблица
// заставляла бы расчёт прыгать между годами без объяснимой причины.
//
// Раздела в форме НЕТ: у раздела кривая ровно одна, раздел и есть её ключ,
// и приходит он адресом запроса. Поле в теле означало бы два места, где
// написано, какому разделу правят кривую, — и вопрос, что делать, когда
// они разошлись.
type ResidualValueCurveSave struct {
	// FirstYearDropBps сколько техника теряет за первый год, в базисных пунктах.
	//
	// Отдельно от остальных лет: за первый год техника теряет заметно больше,
	// и одна ставка на все годы либо завысила бы остаток свежей машины,
	// либо занизила бы старой.
	FirstYearDropBps int `json:"firstYearDropBps" validate:"gte=0,lte=10000" example:"2000"`
	// AnnualDropBps сколько теряется за каждый следующий год ОТ ОСТАТКА.
	AnnualDropBps int `json:"annualDropBps" validate:"gte=0,lte=10000" example:"1200"`
	// FloorBps доля, ниже которой стоимость не опускается.
	//
	// Чистая геометрия на длинном горизонте уводит цену почти в ноль, чего
	// с техникой не происходит: у неё есть цена узлов и металла.
	FloorBps int `json:"floorBps" validate:"gte=0,lte=10000" example:"2000"`
}

// Validate проверяет форму кривой.
func (f *ResidualValueCurveSave) Validate() error {
	return validatorDefault.Validate(f)
}

// ToEntity собирает кривую из формы. Раздел приходит адресом запроса.
func (f *ResidualValueCurveSave) ToEntity(categoryID string) *entity.ResidualValueCurve {
	return &entity.ResidualValueCurve{
		CategoryID:       categoryID,
		FirstYearDropBps: f.FirstYearDropBps,
		AnnualDropBps:    f.AnnualDropBps,
		FloorBps:         f.FloorBps,
	}
}

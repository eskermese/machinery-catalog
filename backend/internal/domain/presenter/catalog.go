package presenter

import (
	"strings"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// Category представление категории в ответе.
type Category struct {
	ID       string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	ParentID string `json:"parentID" example:""`                                                   // Родительская категория; пусто — корень
	Slug     string `json:"slug" validate:"required" example:"excavators"`                         // Идентификатор в URL
	NameRU   string `json:"nameRU" validate:"required" example:"Экскаваторы"`                      // Название на русском
	NameKK   string `json:"nameKK" validate:"required" example:"Экскаваторлар"`                    // Название на казахском; пусто, пока нет перевода
	NameEN   string `json:"nameEN" validate:"required" example:"Excavators"`                       // Название на английском; пусто, пока нет перевода
	Position int    `json:"position" validate:"required" example:"1"`                              // Порядок вывода
	IsActive bool   `json:"isActive" validate:"required" example:"true"`                           // Видимость на сайте

	// ImagePath обложка, выбранная вручную. Пусто — её не выбирали.
	//
	// Нужна админке: без неё редактор не отличит «я выбрал этот снимок»
	// от «сервис подставил, и так совпало».
	ImagePath string `json:"imagePath,omitempty" example:"categories/7c9e6679.jpg"`
	// ImageURL адрес обложки раздела относительно корня API.
	//
	// Заданная вручную, а если её нет — главное фото первой машины
	// поддерева. Клиент не знает, откуда она взялась, и знать не должен:
	// это одна картинка раздела. Пусто — в разделе нет ни одной
	// опубликованной машины с фотографией.
	ImageURL string `json:"imageURL,omitempty" example:"/media/categories/7c9e6679.jpg"`
}

// CategoryList ответ со списком категорий.
type CategoryList struct {
	Items []Category `json:"items" validate:"required"` // Категории в порядке (родитель, position)
}

// NewCategory создаёт представление категории.
func NewCategory(category *entity.Category) Category {
	if category == nil {
		return Category{}
	}

	view := Category{
		ID:       category.ID,
		Slug:     category.Slug,
		NameRU:   category.NameRU,
		NameKK:   category.NameKK,
		NameEN:   category.NameEN,
		Position: category.Position,
		IsActive: category.IsActive,
	}

	if category.ParentID != nil {
		view.ParentID = *category.ParentID
	}

	if category.ImagePath != nil {
		view.ImagePath = *category.ImagePath
	}

	if category.CoverPath != nil {
		view.ImageURL = mediaURL(*category.CoverPath)
	}

	return view
}

// NewCategoryList создаёт представление списка категорий.
//
// Плоский список с parent_id, а не вложенное дерево: собрать дерево из плоского
// списка клиент может сам одним проходом, а разобрать вложенный JSON обратно
// в плоский для таблицы админки — уже нет.
func NewCategoryList(categories entity.Categories) CategoryList {
	items := make([]Category, 0, len(categories))

	for _, category := range categories {
		if category == nil {
			continue
		}

		items = append(items, NewCategory(category))
	}

	return CategoryList{Items: items}
}

// AttributeOption представление варианта характеристики.
type AttributeOption struct {
	ID       string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	Code     string `json:"code" validate:"required" example:"crawler"`                            // Код варианта
	NameRU   string `json:"nameRU" validate:"required" example:"Гусеничный"`                       // Название на русском
	NameKK   string `json:"nameKK" validate:"required" example:"Шынжыр табанды"`                   // Название на казахском; пусто, пока нет перевода
	NameEN   string `json:"nameEN" validate:"required" example:"Crawler"`                          // Название на английском; пусто, пока нет перевода
	Position int    `json:"position" validate:"required" example:"1"`                              // Порядок вывода
}

// Attribute представление характеристики.
type Attribute struct {
	ID           string            `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`         // Идентификатор
	CategoryID   string            `json:"categoryID" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Категория
	Code         string            `json:"code" validate:"required" example:"engine_power"`                               // Машинный ключ
	NameRU       string            `json:"nameRU" validate:"required" example:"Мощность"`                                 // Название на русском
	NameKK       string            `json:"nameKK" validate:"required" example:"Қуаты"`                                    // Название на казахском; пусто, пока нет перевода
	NameEN       string            `json:"nameEN" validate:"required" example:"Engine power"`                             // Название на английском; пусто, пока нет перевода
	DataType     string            `json:"dataType" validate:"required" example:"number"`                                 // Тип значения
	Unit         string            `json:"unit" example:"кВт"`                                                            // Единица измерения
	IsFilterable bool              `json:"isFilterable" validate:"required" example:"true"`                               // Показывать в фильтрах
	Position     int               `json:"position" validate:"required" example:"1"`                                      // Порядок вывода
	Options      []AttributeOption `json:"options" validate:"required"`                                                   // Варианты для типа enum

	// NumberMin и NumberMax границы значений в разделе; пусто — значений нет
	// или характеристика не числовая. Заполняются только публичной ручкой:
	// из них витрина строит подсказку под полем фильтра.
	NumberMin string `json:"numberMin" example:"55"`
	NumberMax string `json:"numberMax" example:"420"`
}

// AttributeList ответ со списком характеристик.
type AttributeList struct {
	Items []Attribute `json:"items" validate:"required"` // Характеристики в порядке вывода
}

// NewAttribute создаёт представление характеристики.
func NewAttribute(attribute *entity.Attribute) Attribute {
	if attribute == nil {
		return Attribute{}
	}

	view := Attribute{
		ID:           attribute.ID,
		CategoryID:   attribute.CategoryID,
		Code:         attribute.Code,
		NameRU:       attribute.NameRU,
		NameKK:       attribute.NameKK,
		NameEN:       attribute.NameEN,
		DataType:     attribute.DataType.String(),
		IsFilterable: attribute.IsFilterable,
		Position:     attribute.Position,
		Options:      make([]AttributeOption, 0, len(attribute.Options)),
	}

	if attribute.Unit != nil {
		view.Unit = *attribute.Unit
	}

	// Границы приходят только с публичной ручки и только у числовых:
	// панели фильтров они нужны, справочнику в админке — нет.
	//
	// Строками, как и остальные десятичные в этом API: JSON-число прошло бы
	// через двоичную плавающую точку и вернулось бы с хвостом в младшем
	// разряде.
	if attribute.NumberMin != nil {
		view.NumberMin = attribute.NumberMin.String()
	}

	if attribute.NumberMax != nil {
		view.NumberMax = attribute.NumberMax.String()
	}

	for _, option := range attribute.Options {
		view.Options = append(view.Options, AttributeOption{
			ID:       option.ID,
			Code:     option.Code,
			NameRU:   option.NameRU,
			NameKK:   option.NameKK,
			NameEN:   option.NameEN,
			Position: option.Position,
		})
	}

	return view
}

// NewAttributeList создаёт представление списка характеристик.
func NewAttributeList(attributes entity.Attributes) AttributeList {
	items := make([]Attribute, 0, len(attributes))

	for _, attribute := range attributes {
		if attribute == nil {
			continue
		}

		items = append(items, NewAttribute(attribute))
	}

	return AttributeList{Items: items}
}

// MachineAttributeValue представление значения характеристики.
//
// Заполнено ровно одно из полей значения — то, которое соответствует типу
// характеристики. Числовое отдаётся строкой: JSON-число прошло бы через
// float64 и потеряло бы младшие разряды.
type MachineAttributeValue struct {
	AttributeID string  `json:"attributeID" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Характеристика
	DataType    string  `json:"dataType" validate:"required" example:"number"`                                  // Тип значения
	Number      *string `json:"number,omitempty" example:"155"`                                                 // Числовое значение
	Text        *string `json:"text,omitempty" example:""`                                                      // Строковое значение
	Code        *string `json:"code,omitempty" example:"crawler"`                                               // Код варианта
	Flag        *bool   `json:"flag,omitempty" example:"true"`                                                  // Значение флага
}

// MediaRoutePrefix префикс, по которому сервис раздаёт загруженные файлы.
//
// Изображения адресуются URL-ом относительно корня API, а не абсолютным
// адресом из конфигурации. Абсолютный пришлось бы собирать из настроенного
// хоста, и он оказался бы неверным при обращении через второе доменное имя
// или через прокси, а в выгрузках протухал бы вместе со сменой домена.
// Относительный клиент разрешает от того же адреса, по которому и так
// обращается к API, а поставить впереди CDN можно не трогая контракт.
//
// Раздаёт сам сервис: развёртывание — один VPS без объектного хранилища,
// и заводить адрес внешнего хранилища ради адресации того, чего нет,
// значило бы усложнять контракт под несуществующую инфраструктуру.
const MediaRoutePrefix = "/media"

// mediaURL собирает адрес файла из его пути внутри хранилища.
func mediaURL(path string) string {
	if path == "" {
		return ""
	}

	return MediaRoutePrefix + "/" + strings.TrimPrefix(path, "/")
}

// MachineImage представление фотографии карточки.
type MachineImage struct {
	ID   string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	Path string `json:"path" validate:"required" example:"machines/xcmg-xe215c-1.jpg"`         // Путь внутри хранилища
	// URL адрес фотографии относительно корня API: клиент дописывает его
	// к тому же адресу, по которому обращается к каталогу.
	URL       string `json:"url" validate:"required" example:"/media/machines/xcmg-xe215c-1.jpg"`
	Position  int    `json:"position" validate:"required" example:"0"`     // Порядок показа
	IsPrimary bool   `json:"isPrimary" validate:"required" example:"true"` // Главное фото
	AltRU     string `json:"altRU" example:""`                             // Подпись на русском
	AltKK     string `json:"altKK" example:""`                             // Подпись на казахском
	AltEN     string `json:"altEN" example:""`                             // Подпись на английском
}

// MachineImageList ответ со списком фотографий.
type MachineImageList struct {
	Items []MachineImage `json:"items" validate:"required"` // Фотографии в порядке показа
}

// Machine представление карточки техники.
type Machine struct {
	ID            string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`         // Идентификатор
	CategoryID    string `json:"categoryID" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Категория
	Brand         string `json:"brand" validate:"required" example:"XCMG"`                                      // Бренд
	Model         string `json:"model" validate:"required" example:"XE215C"`                                    // Модель
	Slug          string `json:"slug" validate:"required" example:"xcmg-xe215c"`                                // Идентификатор в URL
	NameRU        string `json:"nameRU" validate:"required" example:"Экскаватор XCMG XE215C"`                   // Название на русском
	NameKK        string `json:"nameKK" validate:"required" example:"XCMG XE215C экскаваторы"`                  // Название на казахском; пусто, пока нет перевода
	NameEN        string `json:"nameEN" validate:"required" example:"XCMG XE215C excavator"`                    // Название на английском; пусто, пока нет перевода
	DescriptionRU string `json:"descriptionRU" example:""`                                                      // Описание на русском
	DescriptionKK string `json:"descriptionKK" example:""`                                                      // Описание на казахском
	DescriptionEN string `json:"descriptionEN" example:""`                                                      // Описание на английском

	// PriceCNYCents цена в фынях (1/100 юаня). Пусто, если цена скрыта за формой запроса.
	PriceCNYCents *int64 `json:"priceCnyCents,omitempty" example:"8950000"`
	// IsPricePublic показывается ли цена на витрине.
	IsPricePublic bool `json:"isPricePublic" validate:"required" example:"true"`

	// PriceKZTCents цена в тиынах по действующему курсу; отсутствует, когда
	// курса нет или цена скрыта на витрине.
	//
	// Пересчёт справочный: он не фиксируется и меняется вместе с курсом.
	// Зафиксированную цену даёт только выпущенное коммерческое предложение.
	PriceKZTCents *int64 `json:"priceKztCents,omitempty" example:"4675943000"`
	// FXRateDate дата курса, по которому пересчитана цена.
	//
	// Приходит только вместе с priceKztCents и обязательна при ней: применённый
	// курс при отсутствии свежего — это последний известный, и без даты
	// вчерашний курс неотличим от сегодняшнего. Наценка наружу не выносится:
	// она уже внутри курса, а её размер посетителю знать незачем.
	FXRateDate *time.Time `json:"fxRateDate,omitempty" example:"2026-08-27T00:00:00Z"`

	// PriceUSDCents цена в центах — второе, НЕОБЯЗАТЕЛЬНОЕ отображение.
	//
	// Приходит, только когда показ доллара включён настройкой и курс пары
	// есть. Отображением всё и ограничивается: ни лизинг, ни стоимость
	// владения, ни коммерческое предложение доллар не используют — они
	// считаются от юаня и тенге.
	PriceUSDCents *int64 `json:"priceUsdCents,omitempty" example:"1243000"`
	// USDRateDate дата курса, по которому получена цена в долларах.
	//
	// Своя, а не общая с тенге: пары разные, и курс одной бывает свежее
	// курса другой.
	USDRateDate *time.Time `json:"usdRateDate,omitempty" example:"2026-08-27T00:00:00Z"`

	Availability   string `json:"availability" validate:"required" example:"in_stock"` // Наличие
	DeliveryWeeks  *int   `json:"deliveryWeeks,omitempty" example:"8"`                 // Срок поставки в неделях
	ProductionYear *int   `json:"productionYear,omitempty" example:"2024"`             // Год выпуска
	Condition      string `json:"condition" validate:"required" example:"new"`         // Состояние

	// FuelConsumptionLph, ServiceIntervalHours и ServiceCostCnyCents —
	// величины для расчёта стоимости владения.
	//
	// Отсутствуют, если не заведены: пустое поле означает «нет данных»,
	// и подменять его нулём нельзя — ноль здесь читался бы как «не расходует
	// топлива». Расход строкой по той же причине, что и число характеристики:
	// он дробный, и float64 на границе контракта теряет младший разряд.
	FuelConsumptionLph   *string `json:"fuelConsumptionLph,omitempty" example:"12.7"`
	ServiceIntervalHours *int    `json:"serviceIntervalHours,omitempty" example:"500"`
	ServiceCostCnyCents  *int64  `json:"serviceCostCnyCents,omitempty" example:"120000"`

	IsActive    bool       `json:"isActive" validate:"required" example:"true"`          // Опубликована ли карточка
	PublishedAt *time.Time `json:"publishedAt,omitempty" example:"2026-08-27T10:00:00Z"` // Дата первой публикации

	AttributeValues []MachineAttributeValue `json:"attributeValues,omitempty"` // Значения характеристик
	Images          []MachineImage          `json:"images,omitempty"`          // Фотографии

	// PrimaryImage главное фото карточки.
	//
	// Существует ради списков: плитке каталога нужна одна картинка, и тянуть
	// ради неё всю галерею — это десятки лишних строк на страницу. В выдаче
	// поиска приходит только оно, images там пусто; на самой карточке приходят
	// оба поля, и primaryImage там — та же фотография, что помечена isPrimary
	// внутри images.
	//
	// Необязательное: у карточки может не быть ни одной фотографии.
	PrimaryImage *MachineImage `json:"primaryImage,omitempty"`
}

// MachinePage страница выдачи каталога.
type MachinePage struct {
	// Items карточки страницы.
	Items []Machine `json:"items" validate:"required"`
	// NextCursor курсор следующей страницы; пусто — страница последняя.
	//
	// Непрозрачная строка: её содержимое — деталь реализации пагинации,
	// и клиент обязан возвращать её как есть, а не собирать сам.
	NextCursor string `json:"nextCursor" example:""`
	// HasMore есть ли ещё карточки за этой страницей.
	HasMore bool `json:"hasMore" validate:"required" example:"true"`
	// Total общее число найденных карточек.
	//
	// Приходит только если клиент попросил его флагом withTotal: точный
	// подсчёт при keyset-пагинации — отдельный запрос по тому же предикату,
	// и он обязан просмотреть все подходящие строки.
	Total *int `json:"total,omitempty" example:"137"`
}

// NewMachine создаёт представление карточки.
//
// Скрытая цена не подменяется нулём и не остаётся в ответе: поле опускается
// целиком. Ноль клиент показал бы как «бесплатно», а «по запросу» — это
// отсутствие цены, а не её значение.
// PriceRates курсы, по которым цена карточки показывается в других валютах.
//
// Структурой, а не двумя указателями подряд: перепутать местами два nil
// одного типа нельзя, если у каждого есть имя.
type PriceRates struct {
	// KZT курс в тенге. Пусто, когда курса нет — витрина в этом случае
	// показывает только юани.
	KZT *entity.AppliedFXRate
	// USD курс в долларах. Пусто, когда показ доллара выключен настройкой
	// ИЛИ когда курса пары нет: и то и другое означает одно — второй строки
	// цены не будет.
	USD *entity.AppliedFXRate
}

func NewMachine(machine *entity.Machine, rates PriceRates) Machine {
	if machine == nil {
		return Machine{}
	}

	view := Machine{
		ID:             machine.ID,
		CategoryID:     machine.CategoryID,
		Brand:          machine.Brand,
		Model:          machine.Model,
		Slug:           machine.Slug,
		NameRU:         machine.NameRU,
		NameKK:         machine.NameKK,
		NameEN:         machine.NameEN,
		DescriptionRU:  stringValue(machine.DescriptionRU),
		DescriptionKK:  stringValue(machine.DescriptionKK),
		DescriptionEN:  stringValue(machine.DescriptionEN),
		IsPricePublic:  machine.IsPricePublic,
		Availability:   machine.Availability.String(),
		DeliveryWeeks:  machine.DeliveryWeeks,
		ProductionYear: machine.ProductionYear,

		FuelConsumptionLph:   fuelConsumption(machine),
		ServiceIntervalHours: machine.ServiceIntervalHours,
		ServiceCostCnyCents:  machine.ServiceCostCNYCents,
		Condition:            machine.Condition.String(),
		IsActive:             machine.IsActive,
		PublishedAt:          machine.PublishedAt,
	}

	if machine.IsPricePublic {
		view.PriceCNYCents = entity.Pointer(machine.PriceCNYCents)
		view.applyRates(machine.PriceCNYCents, rates)
	}

	view.AttributeValues = newAttributeValues(machine.AttributeValues)
	view.Images = newImages(machine.Images)
	view.PrimaryImage = newPrimaryImage(machine)

	return view
}

// newPrimaryImage выбирает главное фото для представления.
//
// Два источника, и порядок между ними важен. Выдача поиска дочитывает фото
// соединением и кладёт его в PrimaryImage, оставляя галерею пустой. Карточка
// читает галерею целиком, и главное фото ищется в ней — иначе поле было бы
// заполнено в списке и пусто на карточке, хотя фотография одна и та же.
func newPrimaryImage(machine *entity.Machine) *MachineImage {
	image := machine.PrimaryImage

	if image == nil {
		for _, candidate := range machine.Images {
			if candidate != nil && candidate.IsPrimary {
				image = candidate

				break
			}
		}
	}

	if image == nil {
		return nil
	}

	view := newImages(entity.MachineImages{image})
	if len(view) == 0 {
		return nil
	}

	return &view[0]
}

// applyRates дописывает цену в тенге и в долларах вместе с датами курсов.
//
// Каждая валюта считается ОТ ЮАНЯ по своей паре и своей наценке, а не одна
// из другой: цепочка юань-тенге-доллар наложила бы наценку пары CNY-KZT
// на долларовую цену, к которой она отношения не имеет.
//
// Цена и дата курса появляются только вместе: цена без даты не сообщает,
// на какой момент она верна, а дата без цены бессмысленна. Отсутствие курса
// не ошибка — витрина в этом случае показывает только юани.
func (m *Machine) applyRates(priceCNYCents int64, rates PriceRates) {
	if rates.KZT != nil && !rates.KZT.IsZero() {
		m.PriceKZTCents = entity.Pointer(rates.KZT.ConvertPriceCents(priceCNYCents))
		m.FXRateDate = entity.Pointer(rates.KZT.RateDate)
	}

	if rates.USD != nil && !rates.USD.IsZero() {
		m.PriceUSDCents = entity.Pointer(rates.USD.ConvertPriceCents(priceCNYCents))
		m.USDRateDate = entity.Pointer(rates.USD.RateDate)
	}
}

// NewAdminMachine создаёт представление карточки для админки.
//
// Отличается одним: цена отдаётся всегда, даже когда скрыта на витрине —
// редактор обязан видеть то, что редактирует.
func NewAdminMachine(machine *entity.Machine) Machine {
	view := NewMachine(machine, PriceRates{})

	if machine != nil {
		view.PriceCNYCents = entity.Pointer(machine.PriceCNYCents)
	}

	return view
}

// NewMachinePage создаёт представление страницы выдачи.
func NewMachinePage(page entity.MachinePage, rates PriceRates) MachinePage {
	items := make([]Machine, 0, len(page.Machines))

	for _, machine := range page.Machines {
		if machine == nil {
			continue
		}

		items = append(items, NewMachine(machine, rates))
	}

	return MachinePage{Items: items, NextCursor: page.NextCursor, HasMore: page.HasMore, Total: page.Total}
}

// NewAdminMachinePage создаёт представление страницы выдачи для админки.
//
// Итог переносится наравне с курсором: withTotal — параметр поиска, а не
// свойство витрины, и админке он нужен даже больше. «В разделе есть
// карточки» и «в разделе 17 карточек» — разные сообщения, когда решают,
// удалять раздел или нет.
func NewAdminMachinePage(page entity.MachinePage) MachinePage {
	items := make([]Machine, 0, len(page.Machines))

	for _, machine := range page.Machines {
		if machine == nil {
			continue
		}

		items = append(items, NewAdminMachine(machine))
	}

	return MachinePage{
		Items:      items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
		Total:      page.Total,
	}
}

// fuelConsumption отдаёт расход топлива строкой или nil, если он не заведён.
func fuelConsumption(machine *entity.Machine) *string {
	if machine.FuelConsumptionLPH == nil {
		return nil
	}

	return entity.Pointer(machine.FuelConsumptionLPH.String())
}

// NewMachineImageList создаёт представление списка фотографий.
func NewMachineImageList(images entity.MachineImages) MachineImageList {
	return MachineImageList{Items: newImages(images)}
}

// newAttributeValues создаёт представления значений характеристик.
func newAttributeValues(values entity.MachineAttributeValues) []MachineAttributeValue {
	if len(values) == 0 {
		return nil
	}

	items := make([]MachineAttributeValue, 0, len(values))

	for _, value := range values {
		if value == nil || value.Value == nil {
			continue
		}

		item := MachineAttributeValue{
			AttributeID: value.AttributeID,
			DataType:    value.Value.DataType().String(),
		}

		switch typed := value.Value.(type) {
		case entity.NumberValue:
			item.Number = entity.Pointer(typed.Number.String())
		case entity.StringValue:
			item.Text = entity.Pointer(typed.Text)
		case entity.EnumValue:
			item.Code = entity.Pointer(typed.Code)
		case entity.BoolValue:
			item.Flag = entity.Pointer(typed.Flag)
		}

		items = append(items, item)
	}

	return items
}

// newImages создаёт представления фотографий.
func newImages(images entity.MachineImages) []MachineImage {
	if len(images) == 0 {
		return nil
	}

	items := make([]MachineImage, 0, len(images))

	for _, image := range images {
		if image == nil {
			continue
		}

		items = append(items, MachineImage{
			ID:        image.ID,
			Path:      image.Path,
			URL:       mediaURL(image.Path),
			Position:  image.Position,
			IsPrimary: image.IsPrimary,
			AltRU:     stringValue(image.AltRU),
			AltKK:     stringValue(image.AltKK),
			AltEN:     stringValue(image.AltEN),
		})
	}

	return items
}

// stringValue разворачивает необязательную строку в значение.
func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// ResidualValueCurve кривая остаточной стоимости раздела.
//
// Вместе с параметрами уходит и посчитанная по ним таблица долей по годам.
// Кривая тремя числами читается плохо: «минус 12% от остатка» не даёт понять,
// сколько машина будет стоить через пять лет, а редактор заводит её именно
// ради этого. Считает сервер — той же формулой, что и расчёт владения,
// чтобы на экране и в расчёте не оказалось двух разных цифр.
type ResidualValueCurve struct {
	// CategoryID раздел, для которого заведена кривая.
	CategoryID string `json:"categoryID" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	// FirstYearDropBps сколько техника теряет за первый год.
	FirstYearDropBps int `json:"firstYearDropBps" example:"2000"`
	// AnnualDropBps сколько теряется за каждый следующий год от остатка.
	AnnualDropBps int `json:"annualDropBps" example:"1200"`
	// FloorBps доля, ниже которой стоимость не опускается.
	FloorBps int `json:"floorBps" example:"2000"`
	// Preview доли по годам, посчитанные по кривой.
	Preview []ResidualValueYear `json:"preview"`
}

// ResidualValueYear доля остаточной стоимости на конец года.
type ResidualValueYear struct {
	// Year год владения.
	Year int `json:"year" example:"3"`
	// RateBps доля цены, которую техника сохраняет к концу года.
	RateBps int `json:"rateBps" example:"6195"`
}

// residualPreviewYears горизонт таблицы долей.
//
// Десять лет: столько живёт техника в разговоре о покупке, а дальше кривая
// упирается в порог и новых строк не добавляет.
const residualPreviewYears = 10

// NewResidualValueCurve собирает кривую для ответа.
func NewResidualValueCurve(curve *entity.ResidualValueCurve) *ResidualValueCurve {
	if curve == nil {
		return nil
	}

	preview := make([]ResidualValueYear, 0, residualPreviewYears)
	for year := 1; year <= residualPreviewYears; year++ {
		preview = append(preview, ResidualValueYear{Year: year, RateBps: curve.RateFor(year)})
	}

	return &ResidualValueCurve{
		CategoryID:       curve.CategoryID,
		FirstYearDropBps: curve.FirstYearDropBps,
		AnnualDropBps:    curve.AnnualDropBps,
		FloorBps:         curve.FloorBps,
		Preview:          preview,
	}
}

// ResidualValueCurveList список кривых остаточной стоимости.
type ResidualValueCurveList struct {
	// Items кривые, заведённые в справочнике.
	Items []*ResidualValueCurve `json:"items"`
}

// NewResidualValueCurveList собирает список кривых.
func NewResidualValueCurveList(curves []*entity.ResidualValueCurve) ResidualValueCurveList {
	items := make([]*ResidualValueCurve, 0, len(curves))
	for _, curve := range curves {
		items = append(items, NewResidualValueCurve(curve))
	}

	return ResidualValueCurveList{Items: items}
}

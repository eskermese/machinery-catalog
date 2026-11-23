package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCatalogTime момент операций в тестах каталога.
var testCatalogTime = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

// TestIsValidSlug фиксирует формат слага, общий с CHECK-ами схемы.
func TestIsValidSlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		slug string
		exp  bool
	}{
		{slug: "excavators", exp: true},
		{slug: "mini-excavators", exp: true},
		{slug: "xcmg-xe215c", exp: true},
		{slug: "a1-b2-c3", exp: true},
		{slug: "Excavators", exp: false},
		{slug: "экскаваторы", exp: false},
		{slug: "excavators--mini", exp: false},
		{slug: "-excavators", exp: false},
		{slug: "excavators-", exp: false},
		{slug: "excavators mini", exp: false},
		{slug: "", exp: false},
	}

	for _, c := range cases {
		c := c

		t.Run(c.slug, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, IsValidSlug(c.slug))
		})
	}
}

// TestAttributeValue_DataType фиксирует связь типа значения и его представления.
//
// Главное свойство модели: комбинация «тип number с заполненной строкой»
// не выражается в типах — у NumberValue просто нет строкового поля.
func TestAttributeValue_DataType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value AttributeValue
		exp   AttributeDataType
	}{
		{name: "число", value: NewNumberValue(decimal.NewFromInt(155)), exp: AttributeTypeNumber},
		{name: "строка", value: NewStringValue("гидромолот"), exp: AttributeTypeString},
		{name: "флаг", value: NewBoolValue(true), exp: AttributeTypeBool},
		{name: "вариант", value: NewEnumValue("crawler"), exp: AttributeTypeEnum},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, c.value.DataType())
		})
	}
}

// TestAttributeValue_EnumIsNotString фиксирует, что вариант и свободная строка —
// разные типы, хотя в БД лежат в одной колонке: код варианта обязан существовать
// в справочнике, и отличать его нужно до записи.
func TestAttributeValue_EnumIsNotString(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, NewStringValue("crawler").DataType(), NewEnumValue("crawler").DataType())
}

// TestAttributeDataType_IsValid фиксирует закрытый перечень типов.
func TestAttributeDataType_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, AttributeTypeNumber.IsValid())
	assert.True(t, AttributeTypeString.IsValid())
	assert.True(t, AttributeTypeBool.IsValid())
	assert.True(t, AttributeTypeEnum.IsValid())
	assert.False(t, AttributeDataType("json").IsValid())
	assert.Equal(t, "number", AttributeTypeNumber.String())
}

// TestCategory_Validate фиксирует проверки категории.
func TestCategory_Validate(t *testing.T) {
	t.Parallel()

	valid := &Category{Slug: "excavators", NameRU: "Э", NameKK: "Э", NameEN: "E"}

	cases := []struct {
		name   string
		mutate func(*Category)
		expErr error
	}{
		{name: "корректная категория", mutate: func(*Category) {}},
		{name: "неверный слаг", mutate: func(c *Category) { c.Slug = "Экскаваторы" }, expErr: ErrSlugInvalid},
		{name: "нет русского названия", mutate: func(c *Category) { c.NameRU = "" }, expErr: ErrCategoryNameRequired},
		// Непереведённая категория сохраняется: пустые казахское
		// и английское названия — обычное состояние, пока перевода нет.
		{name: "нет казахского и английского", mutate: func(c *Category) { c.NameKK, c.NameEN = "", "" }},
		{name: "отрицательный порядок", mutate: func(c *Category) { c.Position = -1 }, expErr: ErrPositionInvalid},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			category := *valid
			c.mutate(&category)

			err := category.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestCategory_Validate_Nil фиксирует отказ на пустой категории.
func TestCategory_Validate_Nil(t *testing.T) {
	t.Parallel()

	var category *Category

	require.ErrorIs(t, category.Validate(), ErrNilPointer)
}

// TestAttribute_Validate фиксирует проверки характеристики.
func TestAttribute_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		attribute *Attribute
		expErr    error
	}{
		{
			name:      "числовая с единицей",
			attribute: &Attribute{NameRU: "Мощность", Code: "engine_power", DataType: AttributeTypeNumber, Unit: Pointer("кВт")},
		},
		{
			name:      "вариантная без единицы",
			attribute: &Attribute{NameRU: "Привод", Code: "drive_type", DataType: AttributeTypeEnum},
		},
		{
			// Единица у нечисловой характеристики — ошибка ввода, а не редкий случай.
			name:      "единица у вариантной характеристики",
			attribute: &Attribute{NameRU: "Привод", Code: "drive_type", DataType: AttributeTypeEnum, Unit: Pointer("кВт")},
			expErr:    ErrAttributeUnitNotAllowed,
		},
		{
			name:      "неизвестный тип",
			attribute: &Attribute{Code: "x", NameRU: "Х", DataType: "json"},
			expErr:    ErrAttributeTypeInvalid,
		},
		{
			name:      "нет кода",
			attribute: &Attribute{NameRU: "Х", DataType: AttributeTypeNumber},
			expErr:    ErrAttributeCodeInvalid,
		},
		{
			// Раньше имя характеристики не проверялось вовсе, и безымянная
			// доходила до фильтров пустой подписью над парой полей «от — до».
			name:      "нет русского названия",
			attribute: &Attribute{Code: "engine_power", DataType: AttributeTypeNumber},
			expErr:    ErrAttributeNameRequired,
		},
		{
			// Непереведённая характеристика сохраняется.
			name: "нет казахского и английского",
			attribute: &Attribute{
				NameRU: "Мощность", Code: "engine_power", DataType: AttributeTypeNumber,
			},
		},
		{
			name:      "отрицательный порядок",
			attribute: &Attribute{NameRU: "Х", Code: "x", DataType: AttributeTypeNumber, Position: -1},
			expErr:    ErrPositionInvalid,
		},
		{
			// Формат кода проверяет и CHECK attributes_code_format; без проверки
			// здесь нарушение возвращалось бы клиенту внутренней ошибкой.
			name:      "заглавные буквы в коде",
			attribute: &Attribute{NameRU: "Х", Code: "EnginePower", DataType: AttributeTypeNumber},
			expErr:    ErrAttributeCodeInvalid,
		},
		{
			name:      "дефис в коде",
			attribute: &Attribute{NameRU: "Х", Code: "engine-power", DataType: AttributeTypeNumber},
			expErr:    ErrAttributeCodeInvalid,
		},
		{
			// Ровно тот случай, на котором ручка отвечала 500: обрезка длинного
			// названия оставляла хвостовой разделитель.
			name:      "хвостовое подчёркивание в коде",
			attribute: &Attribute{NameRU: "Х", Code: "engine_power_", DataType: AttributeTypeNumber},
			expErr:    ErrAttributeCodeInvalid,
		},
		{
			name:      "кириллица в коде",
			attribute: &Attribute{NameRU: "Х", Code: "мощность", DataType: AttributeTypeNumber},
			expErr:    ErrAttributeCodeInvalid,
		},
		{
			name: "вариант с неверным кодом",
			attribute: &Attribute{
				NameRU: "Привод", Code: "drive_type", DataType: AttributeTypeEnum,
				Options: AttributeOptions{{NameRU: "Г", Code: "Crawler"}},
			},
			expErr: ErrAttributeOptionCodeInvalid,
		},
		{
			name: "повтор кода среди вариантов",
			attribute: &Attribute{
				NameRU: "Привод", Code: "drive_type", DataType: AttributeTypeEnum,
				Options: AttributeOptions{{NameRU: "Г", Code: "crawler"}, {NameRU: "Г", Code: "crawler"}},
			},
			expErr: ErrAttributeOptionDuplicate,
		},
		{
			name: "варианты в порядке",
			attribute: &Attribute{
				NameRU: "Привод", Code: "drive_type", DataType: AttributeTypeEnum,
				Options: AttributeOptions{{NameRU: "Г", Code: "crawler"}, {NameRU: "К", Code: "wheeled", Position: 1}},
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := c.attribute.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestIsValidCode фиксирует формат кода, общий с CHECK-ами attributes_code_format
// и attribute_options_code_format.
//
// Разделитель здесь подчёркивание, а не дефис: код уходит в имена полей
// и ключи фильтров, слаг — в адрес страницы. Проверка отдельным тестом,
// потому что перепутать эти два формата легко, а последствия видны только
// отказом базы.
func TestIsValidCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		code string
		exp  bool
	}{
		{code: "power", exp: true},
		{code: "engine_power", exp: true},
		{code: "a1_b2_c3", exp: true},
		{code: "EnginePower", exp: false},
		{code: "engine-power", exp: false},
		{code: "engine_power_", exp: false},
		{code: "_engine_power", exp: false},
		{code: "engine__power", exp: false},
		{code: "engine power", exp: false},
		{code: "мощность", exp: false},
		{code: "", exp: false},
	}

	for _, c := range cases {
		c := c

		t.Run(c.code, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, IsValidCode(c.code))
		})
	}
}

// TestAttributeOption_Validate фиксирует проверку отдельного варианта.
func TestAttributeOption_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&AttributeOption{NameRU: "Гусеничный", Code: "crawler"}).Validate())
	require.ErrorIs(t, (&AttributeOption{NameRU: "Г", Code: "Crawler"}).Validate(), ErrAttributeOptionCodeInvalid)
	require.ErrorIs(t,
		(&AttributeOption{NameRU: "Г", Code: "crawler", Position: -1}).Validate(), ErrPositionInvalid)

	// Русское название обязательно: без него в панели фильтров появлялся
	// флажок без подписи.
	require.ErrorIs(t,
		(&AttributeOption{Code: "crawler"}).Validate(), ErrAttributeOptionNameRequired)

	// Непереведённый вариант сохраняется.
	require.NoError(t, (&AttributeOption{NameRU: "Гусеничный", Code: "crawler"}).Validate())

	var option *AttributeOption

	require.ErrorIs(t, option.Validate(), ErrNilPointer)
}

// TestAttribute_Validate_Nil фиксирует отказ на пустой характеристике.
func TestAttribute_Validate_Nil(t *testing.T) {
	t.Parallel()

	var attribute *Attribute

	require.ErrorIs(t, attribute.Validate(), ErrNilPointer)
}

// TestAttribute_HasOption фиксирует проверку кода варианта: внешнего ключа
// на attribute_options нет, и существование кода проверяет домен.
func TestAttribute_HasOption(t *testing.T) {
	t.Parallel()

	attribute := &Attribute{
		DataType: AttributeTypeEnum,
		Options:  AttributeOptions{{Code: "crawler"}, {Code: "wheeled"}},
	}

	assert.True(t, attribute.HasOption("crawler"))
	assert.True(t, attribute.HasOption("wheeled"))
	assert.False(t, attribute.HasOption("hovercraft"))

	var empty *Attribute

	assert.False(t, empty.HasOption("crawler"))
}

// TestMachine_Validate фиксирует проверки карточки.
func TestMachine_Validate(t *testing.T) {
	t.Parallel()

	valid := &Machine{
		Slug: "xcmg-xe215c", Brand: "XCMG", Model: "XE215C",
		NameRU: "X", NameKK: "X", NameEN: "X",
		PriceCNYCents: 8_950_000,
		Availability:  AvailabilityInStock, Condition: ConditionNew,
	}

	cases := []struct {
		name   string
		mutate func(*Machine)
		expErr error
	}{
		{name: "корректная карточка", mutate: func(*Machine) {}},
		{
			// Главный инвариант: витрина сортируется по published_at, и NULL
			// в ней означал бы карточку, которая видна, но не встаёт в ленту.
			name:   "активная без даты публикации",
			mutate: func(m *Machine) { m.IsActive = true },
			expErr: ErrMachinePublishedAtRequired,
		},
		{
			name:   "активная с датой публикации",
			mutate: func(m *Machine) { m.IsActive = true; m.PublishedAt = Pointer(testCatalogTime) },
		},
		{name: "неверный слаг", mutate: func(m *Machine) { m.Slug = "XCMG" }, expErr: ErrSlugInvalid},
		{name: "нет бренда", mutate: func(m *Machine) { m.Brand = "" }, expErr: ErrMachineBrandModelRequired},
		{name: "нет русского названия", mutate: func(m *Machine) { m.NameRU = "" }, expErr: ErrMachineNameRequired},
		{name: "нет казахского и английского", mutate: func(m *Machine) { m.NameKK, m.NameEN = "", "" }},
		{name: "отрицательная цена", mutate: func(m *Machine) { m.PriceCNYCents = -1 }, expErr: ErrMachinePriceInvalid},
		{
			name:   "неизвестное наличие",
			mutate: func(m *Machine) { m.Availability = "reserved" },
			expErr: ErrMachineAvailabilityInvalid,
		},
		{
			name:   "неизвестное состояние",
			mutate: func(m *Machine) { m.Condition = "refurbished" },
			expErr: ErrMachineConditionInvalid,
		},
		{
			name:   "нулевой срок поставки",
			mutate: func(m *Machine) { m.DeliveryWeeks = Pointer(0) },
			expErr: ErrMachineDeliveryWeeksInvalid,
		},
		{
			name:   "год выпуска до нижней границы",
			mutate: func(m *Machine) { m.ProductionYear = Pointer(1949) },
			expErr: ErrMachineProductionYearInvalid,
		},
		{
			name:   "год выпуска на нижней границе",
			mutate: func(m *Machine) { m.ProductionYear = Pointer(1950) },
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			machine := *valid
			c.mutate(&machine)

			err := machine.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestMachine_Validate_Nil фиксирует отказ на пустой карточке.
func TestMachine_Validate_Nil(t *testing.T) {
	t.Parallel()

	var machine *Machine

	require.ErrorIs(t, machine.Validate(), ErrNilPointer)
}

// TestMachine_Publish фиксирует простановку даты первой публикации и то,
// что повторная публикация её не сбрасывает.
func TestMachine_Publish(t *testing.T) {
	t.Parallel()

	machine := &Machine{}

	machine.Publish(testCatalogTime)

	assert.True(t, machine.IsActive)
	require.NotNil(t, machine.PublishedAt)
	assert.Equal(t, testCatalogTime, *machine.PublishedAt)

	machine.Unpublish()
	assert.False(t, machine.IsActive)
	assert.NotNil(t, machine.PublishedAt, "дата первой публикации сохраняется")

	// Повторная публикация позже: дата остаётся прежней, иначе возвращённая
	// карточка поднялась бы в ленте как новинка.
	machine.Publish(testCatalogTime.AddDate(0, 1, 0))
	assert.Equal(t, testCatalogTime, *machine.PublishedAt)
}

// TestMachine_PublishNil фиксирует, что пустая карточка не паникует.
func TestMachine_PublishNil(t *testing.T) {
	t.Parallel()

	var machine *Machine

	assert.NotPanics(t, func() {
		machine.Publish(testCatalogTime)
		machine.Unpublish()
	})
	assert.False(t, machine.IsPubliclyVisible())
}

// TestMachine_IsPubliclyVisible фиксирует условие показа на витрине.
func TestMachine_IsPubliclyVisible(t *testing.T) {
	t.Parallel()

	assert.False(t, (&Machine{}).IsPubliclyVisible(), "черновик не показывается")
	assert.False(t, (&Machine{IsActive: true}).IsPubliclyVisible(), "активная без даты не показывается")
	assert.True(t, (&Machine{IsActive: true, PublishedAt: Pointer(testCatalogTime)}).IsPubliclyVisible())
}

// TestMachineImages_Validate фиксирует проверку набора фотографий целиком:
// «не больше одного главного» — свойство набора, на одной строке его не увидеть.
func TestMachineImages_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		images MachineImages
		expErr error
	}{
		{name: "пустой набор", images: nil},
		{name: "одно главное", images: MachineImages{{Path: "a.jpg", IsPrimary: true}, {Path: "b.jpg"}}},
		{name: "ни одного главного", images: MachineImages{{Path: "a.jpg"}}},
		{
			name: "два главных",
			images: MachineImages{
				{Path: "a.jpg", IsPrimary: true},
				{Path: "b.jpg", IsPrimary: true},
			},
			expErr: ErrMachineImagePrimaryConflict,
		},
		{name: "нет пути", images: MachineImages{{}}, expErr: ErrMachineImagePathRequired},
		{
			name:   "отрицательный порядок",
			images: MachineImages{{Path: "a.jpg", Position: -1}},
			expErr: ErrPositionInvalid,
		},
		{name: "пустой элемент", images: MachineImages{nil}, expErr: ErrNilPointer},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := c.images.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestMachineImages_Primary фиксирует поиск главного фото.
func TestMachineImages_Primary(t *testing.T) {
	t.Parallel()

	images := MachineImages{{Path: "a.jpg"}, {Path: "b.jpg", IsPrimary: true}}

	primary := images.Primary()
	require.NotNil(t, primary)
	assert.Equal(t, "b.jpg", primary.Path)

	assert.Nil(t, MachineImages{{Path: "a.jpg"}}.Primary())
}

// TestMachineAttributeValues_Validate фиксирует проверку набора значений.
func TestMachineAttributeValues_Validate(t *testing.T) {
	t.Parallel()

	valid := MachineAttributeValues{
		{AttributeID: "attr", Value: NewNumberValue(decimal.NewFromInt(1))},
	}
	require.NoError(t, valid.Validate())

	require.ErrorIs(t, MachineAttributeValues{nil}.Validate(), ErrNilPointer)
	require.ErrorIs(t, MachineAttributeValues{{AttributeID: "attr"}}.Validate(), ErrNilPointer)
	require.ErrorIs(t,
		MachineAttributeValues{{Value: NewBoolValue(true)}}.Validate(), ErrAttributeValueInvalid)
}

// TestMachineAvailabilityAndCondition фиксирует закрытые перечни.
func TestMachineAvailabilityAndCondition(t *testing.T) {
	t.Parallel()

	assert.True(t, AvailabilityInStock.IsValid())
	assert.True(t, AvailabilityOnOrder.IsValid())
	assert.False(t, MachineAvailability("reserved").IsValid())
	assert.Equal(t, "in_stock", AvailabilityInStock.String())

	assert.True(t, ConditionNew.IsValid())
	assert.True(t, ConditionUsed.IsValid())
	assert.False(t, MachineCondition("refurbished").IsValid())
	assert.Equal(t, "new", ConditionNew.String())
}

// TestNumberValue_String фиксирует запись числовой характеристики строкой
// фиксированной точности: без хвоста «155» и «155.000» были бы одним значением
// в двух написаниях, и записанное не совпало бы с прочитанным при сравнении.
func TestNumberValue_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		exp  string
	}{
		{name: "целое дополняется хвостом", raw: "155", exp: "155.000"},
		{name: "точность сохраняется", raw: "155.500", exp: "155.500"},
		{name: "короткий хвост дополняется", raw: "0.5", exp: "0.500"},
		{name: "лишние знаки округляются", raw: "1.23456", exp: "1.235"},
		{name: "округление от нуля", raw: "-1.2345", exp: "-1.235"},
		{name: "ноль", raw: "0", exp: "0.000"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			value := NewNumberValue(decimal.RequireFromString(c.raw))

			assert.Equal(t, c.exp, value.String())
		})
	}
}

// TestNumberValue_StringMatchesSchemaScale фиксирует, что точность записи
// совпадает с numeric(14,3): расхождение означало бы, что округляет база,
// а не домен, и записанное становилось бы известно только после вставки.
func TestNumberValue_StringMatchesSchemaScale(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, AttributeNumberScale)

	value := NewNumberValue(decimal.RequireFromString("1"))

	assert.Len(t, value.String()[len("1."):], AttributeNumberScale)
}

// TestCategoryCover_Newer фиксирует порядок кандидатов в обложку раздела.
//
// Он обязан совпадать с порядком выдачи каталога (published_at DESC
// NULLS FIRST, id DESC): обложка раздела — это максимум по поддереву,
// и сравнивать кандидатов по другому правилу значило бы показать на плитке
// не ту машину, что стоит в разделе первой.
func TestCategoryCover_Newer(t *testing.T) {
	t.Parallel()

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		a    CategoryCover
		b    CategoryCover
		exp  bool
	}{
		{
			name: "свежая дата раньше старой",
			a:    CategoryCover{PublishedAt: &newer, MachineID: "a"},
			b:    CategoryCover{PublishedAt: &older, MachineID: "b"},
			exp:  true,
		},
		{
			name: "старая дата позже свежей",
			a:    CategoryCover{PublishedAt: &older, MachineID: "a"},
			b:    CategoryCover{PublishedAt: &newer, MachineID: "b"},
			exp:  false,
		},
		{
			// NULLS FIRST: пустая дата идёт раньше любой заданной — то же,
			// что делает сортировка выдачи.
			name: "пустая дата раньше заданной",
			a:    CategoryCover{MachineID: "a"},
			b:    CategoryCover{PublishedAt: &newer, MachineID: "b"},
			exp:  true,
		},
		{
			name: "заданная дата позже пустой",
			a:    CategoryCover{PublishedAt: &newer, MachineID: "a"},
			b:    CategoryCover{MachineID: "b"},
			exp:  false,
		},
		{
			// Даты равны — решает идентификатор, как тай-брейкер выдачи.
			name: "при равных датах побеждает больший идентификатор",
			a:    CategoryCover{PublishedAt: &newer, MachineID: "b"},
			b:    CategoryCover{PublishedAt: &newer, MachineID: "a"},
			exp:  true,
		},
		{
			name: "обе даты пустые — ни одна не раньше",
			a:    CategoryCover{MachineID: "a"},
			b:    CategoryCover{MachineID: "b"},
			exp:  false,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, c.a.Newer(c.b))
		})
	}
}

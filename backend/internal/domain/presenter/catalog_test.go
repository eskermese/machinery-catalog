package presenter

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestMediaURL фиксирует сборку адреса фотографии из пути в хранилище.
//
// Адрес относительный: клиент разрешает его от того же корня, по которому
// обращается к API, — абсолютный пришлось бы собирать из настроенного хоста
// и он оказался бы неверным при обращении через второе доменное имя.
func TestMediaURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		exp  string
	}{
		{name: "обычный путь", path: "machines/xcmg-xe215c-1.jpg", exp: "/media/machines/xcmg-xe215c-1.jpg"},
		{name: "ведущий слеш не удваивается", path: "/machines/a.jpg", exp: "/media/machines/a.jpg"},
		{name: "пустой путь даёт пустой адрес", path: "", exp: ""},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, mediaURL(c.path))
		})
	}
}

// TestNewMachine_ImagesCarryURL фиксирует, что адрес уходит вместе с путём:
// путь — деталь хранилища, по нему клиент ничего загрузить не может.
func TestNewMachine_ImagesCarryURL(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{
		ID: "machine-1", Slug: "xcmg-xe215c", IsPricePublic: true, PriceCNYCents: 8_950_000,
		Images: entity.MachineImages{
			{ID: "img-1", Path: "machines/a.jpg", Position: 0, IsPrimary: true},
			nil, // Пустая строка набора не должна ронять представление.
			{ID: "img-2", Path: "machines/b.jpg", Position: 1},
		},
	}

	view := NewMachine(machine, PriceRates{})

	require.Len(t, view.Images, 2)
	assert.Equal(t, "machines/a.jpg", view.Images[0].Path)
	assert.Equal(t, "/media/machines/a.jpg", view.Images[0].URL)
	assert.Equal(t, "/media/machines/b.jpg", view.Images[1].URL)
}

// TestNewMachine_PrimaryImageFromGallery фиксирует, что на карточке главное
// фото берётся из галереи: клиент не должен получать поле, заполненное
// в списке и пустое на карточке, хотя фотография одна и та же.
func TestNewMachine_PrimaryImageFromGallery(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{
		ID: "machine-1", Slug: "xcmg-xe215c",
		Images: entity.MachineImages{
			{ID: "img-1", Path: "machines/a.jpg", Position: 0},
			{ID: "img-2", Path: "machines/b.jpg", Position: 1, IsPrimary: true},
		},
	}

	view := NewMachine(machine, PriceRates{})

	require.NotNil(t, view.PrimaryImage)
	assert.Equal(t, "img-2", view.PrimaryImage.ID)
	assert.Equal(t, "/media/machines/b.jpg", view.PrimaryImage.URL)
	assert.True(t, view.PrimaryImage.IsPrimary)
}

// TestNewMachine_PrimaryImageFromSearch фиксирует выдачу поиска: фото приходит
// дочитанным отдельно, галерея пуста, и в ответе оказывается только главное.
func TestNewMachine_PrimaryImageFromSearch(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{
		ID: "machine-1", Slug: "xcmg-xe215c",
		PrimaryImage: &entity.MachineImage{
			ID: "img-1", Path: "machines/a.jpg", IsPrimary: true,
		},
	}

	view := NewMachine(machine, PriceRates{})

	require.NotNil(t, view.PrimaryImage)
	assert.Equal(t, "/media/machines/a.jpg", view.PrimaryImage.URL)
	assert.Empty(t, view.Images, "в выдаче поиска галереи нет")
}

// TestNewMachine_WithoutImagesHasNoPrimary фиксирует, что карточка без
// фотографий не получает пустого объекта: поле опускается целиком.
func TestNewMachine_WithoutImagesHasNoPrimary(t *testing.T) {
	t.Parallel()

	view := NewMachine(&entity.Machine{ID: "machine-1", Slug: "xcmg-xe215c"}, PriceRates{})

	assert.Nil(t, view.PrimaryImage)
}

// TestNewMachine_GalleryWithoutPrimaryHasNoPrimary фиксирует, что главным
// не назначается первое попавшееся фото: признак ставит редактор, и подмена
// его порядком показала бы на витрине не то, что выбрали.
func TestNewMachine_GalleryWithoutPrimaryHasNoPrimary(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{
		ID: "machine-1", Slug: "xcmg-xe215c",
		Images: entity.MachineImages{
			{ID: "img-1", Path: "machines/a.jpg", Position: 0},
		},
	}

	view := NewMachine(machine, PriceRates{})

	assert.Nil(t, view.PrimaryImage)
	require.Len(t, view.Images, 1)
}

// TestNewMachine_HiddenPriceIsOmitted фиксирует, что скрытая цена опускается
// целиком, а не подменяется нулём: ноль клиент показал бы как «бесплатно».
func TestNewMachine_HiddenPriceIsOmitted(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{ID: "machine-1", PriceCNYCents: 8_950_000, IsPricePublic: false}

	view := NewMachine(machine, PriceRates{})

	assert.Nil(t, view.PriceCNYCents)
	assert.False(t, view.IsPricePublic)

	// Админское представление показывает цену всегда: редактор обязан видеть
	// то, что редактирует.
	admin := NewAdminMachine(machine)
	require.NotNil(t, admin.PriceCNYCents)
	assert.Equal(t, int64(8_950_000), *admin.PriceCNYCents)
}

// TestNewMachine_Nil фиксирует, что пустая карточка не роняет представление.
func TestNewMachine_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Machine{}, NewMachine(nil, PriceRates{}))
	assert.Empty(t, NewMachineImageList(nil).Items)
}

// TestNewMachine_PriceInTenge фиксирует пересчёт цены в тенге.
//
// Цена и дата курса появляются только вместе: цена без даты не сообщает,
// на какой момент она верна. Наценка наружу не выносится — она уже внутри
// курса, а её размер посетителю знать незачем.
func TestNewMachine_PriceInTenge(t *testing.T) {
	t.Parallel()

	rateDate := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	rate := &entity.AppliedFXRate{
		Rate:      decimal.RequireFromString("522.340000"),
		RateDate:  rateDate,
		MarkupBps: 200,
	}

	machine := &entity.Machine{ID: "m-1", PriceCNYCents: 8_950_000, IsPricePublic: true}

	view := NewMachine(machine, PriceRates{KZT: rate})

	require.NotNil(t, view.PriceKZTCents)
	// 8 950 000 центов × 522.34 = 4 674 943 000 тиынов.
	assert.Equal(t, int64(4_674_943_000), *view.PriceKZTCents)

	require.NotNil(t, view.FXRateDate)
	assert.Equal(t, rateDate, *view.FXRateDate)

	assert.Equal(t, *view.PriceKZTCents, rate.ConvertPriceCents(machine.PriceCNYCents),
		"витрина считает тем же правилом, что и выпуск предложения")
}

// TestNewMachine_WithoutRateKeepsDollars фиксирует, что отсутствие курса
// роняет пересчёт, но не карточку: витрина показывает доллары.
func TestNewMachine_WithoutRateKeepsDollars(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{ID: "m-1", PriceCNYCents: 8_950_000, IsPricePublic: true}

	cases := []struct {
		name string
		rate *entity.AppliedFXRate
	}{
		{name: "курса нет", rate: nil},
		{name: "курс пустой", rate: &entity.AppliedFXRate{}},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			view := NewMachine(machine, PriceRates{KZT: c.rate})

			require.NotNil(t, view.PriceCNYCents, "цена в долларах остаётся")
			assert.Nil(t, view.PriceKZTCents)
			assert.Nil(t, view.FXRateDate, "дата курса без цены бессмысленна")
		})
	}
}

// TestNewMachine_HiddenPriceHasNoTenge фиксирует, что скрытая цена не утекает
// пересчитанной: иначе форма «цена по запросу» обходилась бы делением
// на курс.
func TestNewMachine_HiddenPriceHasNoTenge(t *testing.T) {
	t.Parallel()

	rate := &entity.AppliedFXRate{
		Rate:     decimal.RequireFromString("522.340000"),
		RateDate: time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
	}

	machine := &entity.Machine{ID: "m-1", PriceCNYCents: 8_950_000, IsPricePublic: false}

	view := NewMachine(machine, PriceRates{KZT: rate})

	assert.Nil(t, view.PriceCNYCents)
	assert.Nil(t, view.PriceKZTCents)
	assert.Nil(t, view.FXRateDate)
}

// TestNewMachinePage_RateAppliedToEveryItem фиксирует, что курс один на всю
// выдачу: карточки страницы не могут оказаться пересчитанными по разным курсам.
func TestNewMachinePage_RateAppliedToEveryItem(t *testing.T) {
	t.Parallel()

	rate := &entity.AppliedFXRate{
		Rate:     decimal.RequireFromString("500.000000"),
		RateDate: time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
	}

	page := entity.MachinePage{Machines: entity.Machines{
		{ID: "m-1", PriceCNYCents: 100, IsPricePublic: true},
		{ID: "m-2", PriceCNYCents: 200, IsPricePublic: true},
		{ID: "m-3", PriceCNYCents: 300, IsPricePublic: false},
	}}

	view := NewMachinePage(page, PriceRates{KZT: rate})

	require.Len(t, view.Items, 3)
	require.NotNil(t, view.Items[0].PriceKZTCents)
	assert.Equal(t, int64(50_000), *view.Items[0].PriceKZTCents)
	require.NotNil(t, view.Items[1].PriceKZTCents)
	assert.Equal(t, int64(100_000), *view.Items[1].PriceKZTCents)
	assert.Nil(t, view.Items[2].PriceKZTCents, "скрытая цена не пересчитывается")
}

// TestNewAdminMachine_HasNoTenge фиксирует, что админское представление тенге
// не показывает: редактор работает с прайсом в долларах, а справочный пересчёт
// только сбивал бы его с толку при правке цены.
func TestNewAdminMachine_HasNoTenge(t *testing.T) {
	t.Parallel()

	machine := &entity.Machine{ID: "m-1", PriceCNYCents: 8_950_000, IsPricePublic: false}

	view := NewAdminMachine(machine)

	require.NotNil(t, view.PriceCNYCents)
	assert.Nil(t, view.PriceKZTCents)
}

// TestNewCategory_Cover фиксирует, что клиент получает готовый адрес обложки
// и не знает, откуда та взялась.
func TestNewCategory_Cover(t *testing.T) {
	t.Parallel()

	t.Run("подставленная обложка", func(t *testing.T) {
		t.Parallel()

		view := NewCategory(&entity.Category{
			ID: "c1", Slug: "excavators",
			CoverPath: entity.Pointer("machines/a.jpg"),
		})

		assert.Equal(t, "/media/machines/a.jpg", view.ImageURL)
		assert.Empty(t, view.ImagePath, "вручную обложку не выбирали")
	})

	t.Run("выбранная вручную", func(t *testing.T) {
		t.Parallel()

		view := NewCategory(&entity.Category{
			ID: "c1", Slug: "excavators",
			ImagePath: entity.Pointer("categories/b.jpg"),
			CoverPath: entity.Pointer("categories/b.jpg"),
		})

		assert.Equal(t, "/media/categories/b.jpg", view.ImageURL)
		assert.Equal(t, "categories/b.jpg", view.ImagePath,
			"админке нужно отличать выбранную обложку от подставленной")
	})

	t.Run("пустой раздел", func(t *testing.T) {
		t.Parallel()

		view := NewCategory(&entity.Category{ID: "c1", Slug: "excavators"})

		assert.Empty(t, view.ImageURL, "обложки нет — поля нет")
		assert.Empty(t, view.ImagePath)
	})
}

// TestNewAdminMachinePage_CarriesTotal фиксирует, что итог доходит и до
// админки.
//
// Он терялся: публичное представление страницы его переносило, админское —
// нет, и клиент, попросивший withTotal, получал ответ без числа. Отличить
// «не просили» от «потеряли по дороге» по такому ответу нельзя.
func TestNewAdminMachinePage_CarriesTotal(t *testing.T) {
	t.Parallel()

	total := 17
	page := entity.MachinePage{
		Machines: entity.Machines{{ID: "m1", Slug: "xcmg-xe215c"}},
		HasMore:  true,
		Total:    &total,
	}

	view := NewAdminMachinePage(page)

	require.NotNil(t, view.Total)
	assert.Equal(t, 17, *view.Total)
}

// TestNewAdminMachinePage_TotalStaysAbsentWhenNotAsked фиксирует обратное:
// без withTotal поле не появляется вовсе, а не приходит нулём.
func TestNewAdminMachinePage_TotalStaysAbsentWhenNotAsked(t *testing.T) {
	t.Parallel()

	view := NewAdminMachinePage(entity.MachinePage{
		Machines: entity.Machines{{ID: "m1", Slug: "xcmg-xe215c"}},
	})

	assert.Nil(t, view.Total, "ноль читался бы как «ничего не нашлось»")
}

// TestNewAttributeList фиксирует, что именно уходит в панель фильтров.
//
// Границы числовой характеристики отдаются строками: JSON-число прошло бы
// через двоичную плавающую точку и «13.3» вернулось бы с хвостом в младшем
// разряде — а из этих строк подписывается подсказка под полем.
//
// Пустые границы значат «значений в разделе нет», и подставлять вместо них
// ноль нельзя: ноль — это тоже значение, и подсказка «от 0» соврала бы.
func TestNewAttributeList(t *testing.T) {
	t.Parallel()

	unit := "кН"
	power := &entity.Attribute{
		ID:           "power",
		CategoryID:   "excavators",
		Code:         "digging_force",
		NameRU:       "Сила копания",
		DataType:     entity.AttributeTypeNumber,
		Unit:         &unit,
		IsFilterable: true,
		NumberMin:    entity.Pointer(decimal.RequireFromString("13.3")),
		NumberMax:    entity.Pointer(decimal.RequireFromString("310")),
	}

	drive := &entity.Attribute{
		ID:       "drive",
		Code:     "drive_type",
		NameRU:   "Привод",
		DataType: entity.AttributeTypeEnum,
		Options: []*entity.AttributeOption{
			{ID: "crawler", Code: "crawler", NameRU: "Гусеничный", Position: 1},
		},
	}

	// nil в списке пропускается, а не роняет ответ.
	list := NewAttributeList(entity.Attributes{power, nil, drive})

	require.Len(t, list.Items, 2)

	assert.Equal(t, "13.3", list.Items[0].NumberMin)
	assert.Equal(t, "310", list.Items[0].NumberMax)
	assert.Equal(t, "кН", list.Items[0].Unit)
	assert.Equal(t, "number", list.Items[0].DataType)

	assert.Empty(t, list.Items[1].NumberMin, "у enum границ не бывает")
	assert.Empty(t, list.Items[1].NumberMax)
	assert.Empty(t, list.Items[1].Unit, "единицы у характеристики может не быть")
	require.Len(t, list.Items[1].Options, 1)
	assert.Equal(t, "Гусеничный", list.Items[1].Options[0].NameRU)

	// Отсутствующая характеристика — пустое представление, а не паника.
	assert.Equal(t, Attribute{}, NewAttribute(nil))
}

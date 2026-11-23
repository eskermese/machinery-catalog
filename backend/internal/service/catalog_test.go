package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// Идентификаторы в тестах каталога.
const (
	testCategoryID   = "44444444-4444-4444-8444-444444444444"
	testPowerAttrID  = "11111111-1111-4111-8111-111111111111"
	testDriveAttrID  = "22222222-2222-4222-8222-222222222222"
	testMachineID    = "66666666-6666-4666-8666-666666666666"
	testMachineSlug  = "xcmg-xe215c"
	testForeignAttr  = "99999999-9999-4999-8999-999999999999"
	testMachineBrand = "XCMG"
)

// testCatalogNow момент операций в тестах каталога.
var testCatalogNow = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

// catalogMocks набор моков для сервиса каталога.
type catalogMocks struct {
	categories *mock_repo.MockCategoryRepository
	attributes *mock_repo.MockAttributeRepository
	machines   *mock_repo.MockMachineRepository
	values     *mock_repo.MockMachineAttributeValueRepository
	images     *mock_repo.MockMachineImageRepository
	media      *mock_repo.MockMediaStorage
	cache      *mock_repo.MockCatalogCache
	residual   *mock_repo.MockResidualValueRepository
}

// newCatalogService собирает сервис каталога на моках с настоящим стабом транзакций.
func newCatalogService(t *testing.T) (CatalogService, catalogMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mocks := catalogMocks{
		categories: mock_repo.NewMockCategoryRepository(ctrl),
		attributes: mock_repo.NewMockAttributeRepository(ctrl),
		machines:   mock_repo.NewMockMachineRepository(ctrl),
		values:     mock_repo.NewMockMachineAttributeValueRepository(ctrl),
		images:     mock_repo.NewMockMachineImageRepository(ctrl),
		media:      mock_repo.NewMockMediaStorage(ctrl),
		cache:      mock_repo.NewMockCatalogCache(ctrl),
		residual:   mock_repo.NewMockResidualValueRepository(ctrl),
	}

	srv := NewCatalogService(mocks.categories, mocks.attributes, mocks.machines,
		mocks.values, mocks.images, mocks.media, mocks.cache, mocks.residual,
		repository.NewTxStub())

	return srv, mocks
}

// testMachine собирает валидную карточку.
func testMachine(published bool) *entity.Machine {
	machine := &entity.Machine{
		ID:            testMachineID,
		CategoryID:    testCategoryID,
		Brand:         testMachineBrand,
		Model:         "XE215C",
		Slug:          testMachineSlug,
		NameRU:        "XCMG XE215C",
		NameKK:        "XCMG XE215C",
		NameEN:        "XCMG XE215C",
		PriceCNYCents: 8_950_000,
		Availability:  entity.AvailabilityInStock,
		Condition:     entity.ConditionNew,
		CreatedAt:     testCatalogNow,
		UpdatedAt:     testCatalogNow,
	}

	if published {
		machine.IsActive = true
		machine.PublishedAt = entity.Pointer(testCatalogNow)
	}

	return machine
}

// testAttributes собирает описание характеристик категории.
func testAttributes() entity.Attributes {
	return entity.Attributes{
		{
			ID: testPowerAttrID, CategoryID: testCategoryID, Code: "engine_power",
			DataType: entity.AttributeTypeNumber, Unit: entity.Pointer("кВт"),
		},
		{
			ID: testDriveAttrID, CategoryID: testCategoryID, Code: "drive_type",
			DataType: entity.AttributeTypeEnum,
			Options: entity.AttributeOptions{
				{Code: "crawler"},
				{Code: "wheeled"},
			},
		},
	}
}

// TestCatalogService_SearchPublicMachines_ForcesPublishedFilter — главная
// проверка разделения витрины и админки: публичный метод выставляет фильтр сам
// и не даёт вызывающему коду его снять.
func TestCatalogService_SearchPublicMachines_ForcesPublishedFilter(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	// Вызывающий код пытается передать false — метод обязан это перезаписать.
	filter := entity.MachineFilter{OnlyPublished: false, Limit: 5}

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
			assert.True(t, f.OnlyPublished, "витрина обязана запрашивать только опубликованные")

			return entity.Machines{testMachine(true)}, nil
		})

	page, err := srv.SearchPublicMachines(context.Background(), &filter)
	require.NoError(t, err)

	assert.Len(t, page.Machines, 1)
	assert.False(t, page.HasMore)
	assert.Empty(t, page.NextCursor)
}

// TestCatalogService_SearchAllMachines_IncludesDrafts фиксирует, что админское
// чтение снимает ограничение витрины даже если в фильтре пришло true.
func TestCatalogService_SearchAllMachines_IncludesDrafts(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
			assert.False(t, f.OnlyPublished, "админка видит и черновики")

			return entity.Machines{testMachine(false)}, nil
		})

	page, err := srv.SearchAllMachines(context.Background(),
		&entity.MachineFilter{OnlyPublished: true, Limit: 5})
	require.NoError(t, err)

	assert.Len(t, page.Machines, 1)
}

// TestCatalogService_Search_RequestsOneExtraRow фиксирует, что признак
// следующей страницы берётся из лишней строки, а не из отдельного count(*).
func TestCatalogService_Search_RequestsOneExtraRow(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	const limit = 2

	first := testMachine(true)
	second := testMachine(true)
	second.ID = "77777777-7777-4777-8777-777777777777"
	third := testMachine(true)
	third.ID = "88888888-8888-4888-8888-888888888888"

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
			assert.Equal(t, limit+1, f.Limit, "у базы запрашивается на строку больше")

			return entity.Machines{first, second, third}, nil
		})

	page, err := srv.SearchPublicMachines(context.Background(), &entity.MachineFilter{Limit: limit})
	require.NoError(t, err)

	assert.Len(t, page.Machines, limit, "лишняя строка в выдачу не попадает")
	assert.True(t, page.HasMore)
	assert.NotEmpty(t, page.NextCursor, "курсор строится по последней строке страницы")

	// Курсор обязан указывать на вторую карточку, а не на третью.
	cursor, err := entity.DecodeMachineCursor(page.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, second.ID, cursor.ID)
}

// TestCatalogService_Search_InvalidFilter фиксирует отказ на неверных параметрах.
func TestCatalogService_Search_InvalidFilter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		filter entity.MachineFilter
		expErr error
	}{
		{
			name:   "неизвестная сортировка",
			filter: entity.MachineFilter{Sort: "rating_desc"},
			expErr: entity.ErrCatalogSortInvalid,
		},
		{
			name:   "размер страницы сверх предела",
			filter: entity.MachineFilter{Limit: entity.MaxMachineLimit + 1},
			expErr: entity.ErrCatalogLimitInvalid,
		},
		{
			name:   "отрицательный размер страницы",
			filter: entity.MachineFilter{Limit: -1},
			expErr: entity.ErrCatalogLimitInvalid,
		},
		{
			name: "курсор от другой сортировки",
			filter: entity.MachineFilter{
				Sort:   entity.SortPriceAsc,
				Cursor: &entity.MachineCursor{Sort: entity.SortPublishedDesc, ID: testMachineID},
			},
			expErr: entity.ErrCatalogCursorInvalid,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, _ := newCatalogService(t)

			_, err := srv.SearchPublicMachines(context.Background(), &c.filter)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestCatalogService_PublicMachineBySlug_HidesDraft фиксирует, что черновик
// снаружи неотличим от несуществующей карточки.
func TestCatalogService_PublicMachineBySlug_HidesDraft(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().GetBySlug(gomock.Any(), testMachineSlug).Return(testMachine(false), nil)

	_, err := srv.PublicMachineBySlug(context.Background(), testMachineSlug)

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
}

// TestCatalogService_PublicMachineBySlug_ReturnsPublished фиксирует выдачу
// опубликованной карточки.
func TestCatalogService_PublicMachineBySlug_ReturnsPublished(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().GetBySlug(gomock.Any(), testMachineSlug).Return(testMachine(true), nil)

	machine, err := srv.PublicMachineBySlug(context.Background(), testMachineSlug)
	require.NoError(t, err)

	assert.Equal(t, testMachineSlug, machine.Slug)
}

// TestCatalogService_MachineBySlug_ReturnsDraft фиксирует, что админское чтение
// отдаёт и черновик.
func TestCatalogService_MachineBySlug_ReturnsDraft(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().GetBySlug(gomock.Any(), testMachineSlug).Return(testMachine(false), nil)

	machine, err := srv.MachineBySlug(context.Background(), testMachineSlug)
	require.NoError(t, err)

	assert.False(t, machine.IsActive)
}

// TestCatalogService_CategoryTree_UsesCache фиксирует чтение дерева из кэша.
func TestCatalogService_CategoryTree_UsesCache(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	cached := entity.Categories{{ID: testCategoryID, Slug: "excavators"}}

	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(cached, true)
	mocks.categories.EXPECT().Tree(gomock.Any()).Times(0)

	tree, err := srv.CategoryTree(context.Background())
	require.NoError(t, err)

	assert.Equal(t, cached, tree)
}

// TestCatalogService_CategoryTree_CacheMissFallsBackToDB фиксирует, что промах
// и отказ кэша одинаково ведут к чтению из базы и не ломают запрос.
func TestCatalogService_CategoryTree_CacheMissFallsBackToDB(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	stored := entity.Categories{{ID: testCategoryID, Slug: "excavators"}}

	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(nil, false)
	mocks.categories.EXPECT().Tree(gomock.Any()).Return(stored, nil)
	mocks.categories.EXPECT().CoverCandidates(gomock.Any()).Return(nil, nil)
	mocks.cache.EXPECT().SetCategoryTree(gomock.Any(), stored)

	tree, err := srv.CategoryTree(context.Background())
	require.NoError(t, err)

	assert.Equal(t, stored, tree)
}

// TestCatalogService_CategoryAttributes_UsesCache фиксирует чтение характеристик из кэша.
func TestCatalogService_CategoryAttributes_UsesCache(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	cached := testAttributes()

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(cached, true)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), gomock.Any()).Times(0)

	attributes, err := srv.CategoryAttributes(context.Background(), testCategoryID)
	require.NoError(t, err)

	assert.Equal(t, cached, attributes)
}

// TestCatalogService_SaveMachine_Atomic фиксирует, что карточка, характеристики
// и фото пишутся вместе: карточка с половиной характеристик хуже, чем отказ.
func TestCatalogService_SaveMachine_Atomic(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)
	machine.ID = ""
	machine.AttributeValues = entity.MachineAttributeValues{
		{AttributeID: testPowerAttrID, Value: entity.NewNumberValue(decimal.NewFromInt(155))},
		{AttributeID: testDriveAttrID, Value: entity.NewEnumValue("crawler")},
	}
	machine.Images = entity.MachineImages{{Path: "a.jpg", IsPrimary: true}}

	created := testMachine(true)

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(testAttributes(), true)
	mocks.machines.EXPECT().Create(gomock.Any(), machine).Return(created, nil)
	mocks.values.EXPECT().Replace(gomock.Any(), created.ID, machine.AttributeValues).Return(nil)
	mocks.images.EXPECT().Replace(gomock.Any(), created.ID, machine.Images).Return(nil)

	saved, err := srv.SaveMachine(context.Background(), machine)
	require.NoError(t, err)

	assert.Equal(t, testMachineID, saved.ID)
	assert.Len(t, saved.AttributeValues, 2)
	assert.Len(t, saved.Images, 1)
}

// TestCatalogService_SaveMachine_KeepsPublication фиксирует главное правило
// правки: сохранение карточки не трогает её публикацию.
//
// Форма сохранения признака публикации не описывает вовсе, поэтому в сущность
// он приходит нулевым. Пока Update писал is_active вместе с остальными полями,
// правка опечатки молча снимала карточку с витрины. Теперь публикацию меняет
// только SetPublication, и Update её не видит.
func TestCatalogService_SaveMachine_KeepsPublication(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	// Ровно то, что приходит из формы: идентификатор есть, признака публикации
	// нет — он в MachineSave не объявлен.
	machine := testMachine(false)
	machine.NameRU = "XCMG XE215C, исправленная опечатка"

	mocks.machines.EXPECT().Update(gomock.Any(), machine).Return(nil)
	mocks.values.EXPECT().Replace(gomock.Any(), testMachineID, gomock.Any()).Return(nil)
	mocks.images.EXPECT().Replace(gomock.Any(), testMachineID, gomock.Any()).Return(nil)
	// SetPublication не ожидается: если сервис его вызовет, gomock уронит тест.

	_, err := srv.SaveMachine(context.Background(), machine)
	require.NoError(t, err)
}

// TestCatalogService_SaveMachine_ValidatesAttributeValues фиксирует три проверки,
// которых схема сделать не может или делает не полностью.
func TestCatalogService_SaveMachine_ValidatesAttributeValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		values entity.MachineAttributeValues
		expErr error
	}{
		{
			name: "характеристика чужой категории",
			values: entity.MachineAttributeValues{
				{AttributeID: testForeignAttr, Value: entity.NewNumberValue(decimal.NewFromInt(1))},
			},
			expErr: entity.ErrAttributeForeignCategory,
		},
		{
			name: "тип значения не совпадает с типом характеристики",
			values: entity.MachineAttributeValues{
				{AttributeID: testPowerAttrID, Value: entity.NewStringValue("много")},
			},
			expErr: entity.ErrAttributeValueTypeMismatch,
		},
		{
			// Внешнего ключа на attribute_options нет — проверку делает сервис.
			name: "код варианта отсутствует в справочнике",
			values: entity.MachineAttributeValues{
				{AttributeID: testDriveAttrID, Value: entity.NewEnumValue("hovercraft")},
			},
			expErr: entity.ErrAttributeOptionUnknown,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newCatalogService(t)

			machine := testMachine(true)
			machine.AttributeValues = c.values

			mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).
				Return(testAttributes(), true)
			// До записи дело не доходит.
			mocks.machines.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			mocks.machines.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
			mocks.values.EXPECT().Replace(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := srv.SaveMachine(context.Background(), machine)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestCatalogService_SaveMachine_ValidatesMachine фиксирует отказ на невалидной карточке.
func TestCatalogService_SaveMachine_ValidatesMachine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*entity.Machine)
		expErr error
	}{
		{
			name:   "активная карточка без даты публикации",
			mutate: func(m *entity.Machine) { m.IsActive = true; m.PublishedAt = nil },
			expErr: entity.ErrMachinePublishedAtRequired,
		},
		{
			name:   "неверный формат слага",
			mutate: func(m *entity.Machine) { m.Slug = "XCMG XE215C" },
			expErr: entity.ErrSlugInvalid,
		},
		{
			name:   "отрицательная цена",
			mutate: func(m *entity.Machine) { m.PriceCNYCents = -1 },
			expErr: entity.ErrMachinePriceInvalid,
		},
		{
			name: "два главных фото",
			mutate: func(m *entity.Machine) {
				m.Images = entity.MachineImages{
					{Path: "a.jpg", IsPrimary: true},
					{Path: "b.jpg", IsPrimary: true},
				}
			},
			expErr: entity.ErrMachineImagePrimaryConflict,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, _ := newCatalogService(t)

			machine := testMachine(true)
			c.mutate(machine)

			_, err := srv.SaveMachine(context.Background(), machine)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestCatalogService_SaveMachine_SlugConflictComesFromDB фиксирует, что занятый
// слаг определяется конфликтом от БД, а не предварительным SELECT: между
// проверкой и вставкой его может занять кто-то другой.
func TestCatalogService_SaveMachine_SlugConflictComesFromDB(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)
	machine.ID = ""

	mocks.machines.EXPECT().Create(gomock.Any(), machine).Return(nil, entity.ErrSlugAlreadyUsed)

	_, err := srv.SaveMachine(context.Background(), machine)

	require.ErrorIs(t, err, entity.ErrSlugAlreadyUsed)
}

// TestCatalogService_SaveMachine_Nil фиксирует отказ на пустой карточке.
func TestCatalogService_SaveMachine_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	_, err := srv.SaveMachine(context.Background(), nil)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestCatalogService_PublishMachine фиксирует простановку даты первой публикации.
func TestCatalogService_PublishMachine(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	draft := testMachine(false)

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(draft, nil)
	mocks.machines.EXPECT().SetPublication(gomock.Any(), gomock.Any()).Return(nil)
	// Публикация меняет подставленную обложку раздела, поэтому дерево
	// категорий сбрасывается.
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	published, err := srv.PublishMachine(context.Background(), testMachineID, testCatalogNow)
	require.NoError(t, err)

	assert.True(t, published.IsActive)
	require.NotNil(t, published.PublishedAt)
	assert.Equal(t, testCatalogNow, *published.PublishedAt)
}

// TestCatalogService_PublishMachine_KeepsFirstDate фиксирует, что повторная
// публикация не сбрасывает дату: иначе возвращённая карточка поднялась бы
// в ленте как новинка.
func TestCatalogService_PublishMachine_KeepsFirstDate(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	firstPublication := testCatalogNow.AddDate(0, -2, 0)

	machine := testMachine(false)
	machine.PublishedAt = entity.Pointer(firstPublication)

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(machine, nil)
	mocks.machines.EXPECT().SetPublication(gomock.Any(), gomock.Any()).Return(nil)
	// Публикация меняет подставленную обложку раздела, поэтому дерево
	// категорий сбрасывается.
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	published, err := srv.PublishMachine(context.Background(), testMachineID, testCatalogNow)
	require.NoError(t, err)

	require.NotNil(t, published.PublishedAt)
	assert.Equal(t, firstPublication, *published.PublishedAt, "дата первой публикации сохраняется")
}

// TestCatalogService_UnpublishMachine фиксирует снятие с публикации без потери даты.
func TestCatalogService_UnpublishMachine(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(machine, nil)
	mocks.machines.EXPECT().SetPublication(gomock.Any(), gomock.Any()).Return(nil)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	unpublished, err := srv.UnpublishMachine(context.Background(), testMachineID)
	require.NoError(t, err)

	assert.False(t, unpublished.IsActive)
	assert.NotNil(t, unpublished.PublishedAt, "дата первой публикации остаётся")
}

// TestCatalogService_SaveCategory_InvalidatesTree фиксирует сброс кэша дерева.
func TestCatalogService_SaveCategory_InvalidatesTree(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	category := &entity.Category{Slug: "excavators", NameRU: "Э", NameKK: "Э", NameEN: "E"}

	mocks.categories.EXPECT().Create(gomock.Any(), category).Return(category, nil)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	_, err := srv.SaveCategory(context.Background(), category)
	require.NoError(t, err)
}

// TestCatalogService_SaveAttribute_InvalidatesCategoryAttributes фиксирует
// сброс кэша характеристик именно той категории, к которой они относятся.
func TestCatalogService_SaveAttribute_InvalidatesCategoryAttributes(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		CategoryID: testCategoryID, Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	mocks.attributes.EXPECT().Create(gomock.Any(), attribute).Return(attribute, nil)
	// Сброс идёт по поддереву, поэтому читается дерево категорий.
	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(entity.Categories{}, true)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), testCategoryID)

	_, err := srv.SaveAttribute(context.Background(), attribute)
	require.NoError(t, err)
}

// TestCatalogService_SaveMachine_ValuesFailureAborts фиксирует, что отказ записи
// характеристик поднимается наверх и карточка не остаётся наполовину сохранённой.
func TestCatalogService_SaveMachine_ValuesFailureAborts(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)
	machine.AttributeValues = entity.MachineAttributeValues{
		{AttributeID: testPowerAttrID, Value: entity.NewNumberValue(decimal.NewFromInt(155))},
	}

	valuesErr := errors.New("база недоступна")

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(testAttributes(), true)
	mocks.machines.EXPECT().Update(gomock.Any(), machine).Return(nil)
	mocks.values.EXPECT().Replace(gomock.Any(), testMachineID, machine.AttributeValues).Return(valuesErr)
	mocks.images.EXPECT().Replace(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.SaveMachine(context.Background(), machine)

	require.ErrorIs(t, err, valuesErr)
}

// TestCatalogService_CategoryBySlug фиксирует чтение категории по слагу.
func TestCatalogService_CategoryBySlug(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	category := &entity.Category{ID: testCategoryID, Slug: "excavators"}

	mocks.categories.EXPECT().GetBySlug(gomock.Any(), "excavators").Return(category, nil)

	found, err := srv.CategoryBySlug(context.Background(), "excavators")
	require.NoError(t, err)

	assert.Equal(t, testCategoryID, found.ID)
}

// TestCatalogService_CategoryBySlug_NotFound фиксирует проброс отсутствия категории.
func TestCatalogService_CategoryBySlug_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.categories.EXPECT().GetBySlug(gomock.Any(), "unknown").Return(nil, entity.ErrCategoryNotFound)

	_, err := srv.CategoryBySlug(context.Background(), "unknown")

	require.ErrorIs(t, err, entity.ErrCategoryNotFound)
}

// TestCatalogService_CategoryAttributes_CacheMissFallsBackToDB фиксирует чтение
// характеристик из базы при промахе кэша и последующее их сохранение.
func TestCatalogService_CategoryAttributes_CacheMissFallsBackToDB(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	stored := testAttributes()

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(nil, false)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).Return(stored, nil)
	mocks.cache.EXPECT().SetCategoryAttributes(gomock.Any(), testCategoryID, stored)

	attributes, err := srv.CategoryAttributes(context.Background(), testCategoryID)
	require.NoError(t, err)

	assert.Equal(t, stored, attributes)
}

// TestCatalogService_CategoryAttributes_RepoFailure фиксирует проброс отказа базы.
func TestCatalogService_CategoryAttributes_RepoFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	repoErr := errors.New("база недоступна")

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(nil, false)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).Return(nil, repoErr)

	_, err := srv.CategoryAttributes(context.Background(), testCategoryID)

	require.ErrorIs(t, err, repoErr)
}

// TestCatalogService_CategoryTree_RepoFailure фиксирует проброс отказа базы.
func TestCatalogService_CategoryTree_RepoFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	repoErr := errors.New("база недоступна")

	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(nil, false)
	mocks.categories.EXPECT().Tree(gomock.Any()).Return(nil, repoErr)

	_, err := srv.CategoryTree(context.Background())

	require.ErrorIs(t, err, repoErr)
}

// TestCatalogService_Search_RepoFailure фиксирует проброс отказа поиска.
func TestCatalogService_Search_RepoFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	repoErr := errors.New("база недоступна")

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, repoErr)

	_, err := srv.SearchPublicMachines(context.Background(), &entity.MachineFilter{})

	require.ErrorIs(t, err, repoErr)
}

// TestCatalogService_MachineBySlug_NotFound фиксирует проброс отсутствия карточки.
func TestCatalogService_MachineBySlug_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().GetBySlug(gomock.Any(), "unknown").Return(nil, entity.ErrMachineNotFound)

	_, err := srv.PublicMachineBySlug(context.Background(), "unknown")

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
}

// TestCatalogService_SaveMachine_ImagesFailureAborts фиксирует, что отказ записи
// фотографий поднимается наверх.
func TestCatalogService_SaveMachine_ImagesFailureAborts(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)
	machine.Images = entity.MachineImages{{Path: "a.jpg", IsPrimary: true}}

	imagesErr := errors.New("база недоступна")

	mocks.machines.EXPECT().Update(gomock.Any(), machine).Return(nil)
	mocks.values.EXPECT().Replace(gomock.Any(), testMachineID, gomock.Any()).Return(nil)
	mocks.images.EXPECT().Replace(gomock.Any(), testMachineID, machine.Images).Return(imagesErr)

	_, err := srv.SaveMachine(context.Background(), machine)

	require.ErrorIs(t, err, imagesErr)
}

// TestCatalogService_SaveMachine_UpdateFailureAborts фиксирует проброс отказа
// обновления карточки.
func TestCatalogService_SaveMachine_UpdateFailureAborts(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)
	updateErr := errors.New("база недоступна")

	mocks.machines.EXPECT().Update(gomock.Any(), machine).Return(updateErr)
	mocks.values.EXPECT().Replace(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.SaveMachine(context.Background(), machine)

	require.ErrorIs(t, err, updateErr)
}

// TestCatalogService_SaveMachine_AttributesReadFailure фиксирует, что отказ
// чтения описания характеристик не даёт сохранить карточку вслепую.
func TestCatalogService_SaveMachine_AttributesReadFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	machine := testMachine(true)
	machine.AttributeValues = entity.MachineAttributeValues{
		{AttributeID: testPowerAttrID, Value: entity.NewNumberValue(decimal.NewFromInt(155))},
	}

	repoErr := errors.New("база недоступна")

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(nil, false)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).Return(nil, repoErr)
	mocks.machines.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.SaveMachine(context.Background(), machine)

	require.ErrorIs(t, err, repoErr)
}

// TestCatalogService_SaveMachine_EmptyValueIsRejected фиксирует отказ на
// значении без самого значения.
func TestCatalogService_SaveMachine_EmptyValueIsRejected(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	machine := testMachine(true)
	machine.AttributeValues = entity.MachineAttributeValues{{AttributeID: testPowerAttrID}}

	_, err := srv.SaveMachine(context.Background(), machine)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestCatalogService_PublishMachine_NotFound фиксирует проброс отсутствия карточки.
func TestCatalogService_PublishMachine_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(nil, entity.ErrMachineNotFound)

	_, err := srv.PublishMachine(context.Background(), testMachineID, testCatalogNow)

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
}

// TestCatalogService_PublishMachine_UpdateFailure фиксирует проброс отказа записи.
func TestCatalogService_PublishMachine_UpdateFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	updateErr := errors.New("база недоступна")

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(testMachine(false), nil)
	mocks.machines.EXPECT().SetPublication(gomock.Any(), gomock.Any()).Return(updateErr)

	_, err := srv.PublishMachine(context.Background(), testMachineID, testCatalogNow)

	require.ErrorIs(t, err, updateErr)
}

// TestCatalogService_UnpublishMachine_NotFound фиксирует проброс отсутствия карточки.
func TestCatalogService_UnpublishMachine_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(nil, entity.ErrMachineNotFound)

	_, err := srv.UnpublishMachine(context.Background(), testMachineID)

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
}

// TestCatalogService_UnpublishMachine_UpdateFailure фиксирует проброс отказа записи.
func TestCatalogService_UnpublishMachine_UpdateFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	updateErr := errors.New("база недоступна")

	mocks.machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(testMachine(true), nil)
	mocks.machines.EXPECT().SetPublication(gomock.Any(), gomock.Any()).Return(updateErr)

	_, err := srv.UnpublishMachine(context.Background(), testMachineID)

	require.ErrorIs(t, err, updateErr)
}

// TestCatalogService_SaveCategory_Update фиксирует обновление существующей категории.
func TestCatalogService_SaveCategory_Update(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	category := &entity.Category{ID: testCategoryID, Slug: "excavators", NameRU: "Э", NameKK: "Э", NameEN: "E"}

	mocks.categories.EXPECT().Update(gomock.Any(), category).Return(nil)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	saved, err := srv.SaveCategory(context.Background(), category)
	require.NoError(t, err)

	assert.Equal(t, testCategoryID, saved.ID)
}

// TestCatalogService_SaveCategory_Failures фиксирует пробросы отказов и отсутствие
// сброса кэша при неудаче: кэш сбрасывать незачем, если ничего не изменилось.
func TestCatalogService_SaveCategory_Failures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		category *entity.Category
		setup    func(catalogMocks, *entity.Category)
		expErr   error
	}{
		{
			name:     "конфликт слага при создании",
			category: &entity.Category{Slug: "excavators", NameRU: "Э", NameKK: "Э", NameEN: "E"},
			setup: func(m catalogMocks, c *entity.Category) {
				m.categories.EXPECT().Create(gomock.Any(), c).Return(nil, entity.ErrSlugAlreadyUsed)
			},
			expErr: entity.ErrSlugAlreadyUsed,
		},
		{
			name: "цикл в дереве при обновлении",
			category: &entity.Category{
				ID: testCategoryID, Slug: "excavators", NameRU: "Э", NameKK: "Э", NameEN: "E",
			},
			setup: func(m catalogMocks, c *entity.Category) {
				m.categories.EXPECT().Update(gomock.Any(), c).Return(entity.ErrCategoryCycle)
			},
			expErr: entity.ErrCategoryCycle,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newCatalogService(t)

			c.setup(mocks, c.category)
			mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any()).Times(0)

			_, err := srv.SaveCategory(context.Background(), c.category)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestCatalogService_SaveCategory_Nil фиксирует отказ на пустой категории.
func TestCatalogService_SaveCategory_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	_, err := srv.SaveCategory(context.Background(), nil)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestCatalogService_SaveAttribute_Update фиксирует обновление характеристики.
func TestCatalogService_SaveAttribute_Update(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		ID: testPowerAttrID, CategoryID: testCategoryID,
		Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	mocks.attributes.EXPECT().Update(gomock.Any(), attribute).Return(nil)
	// Сброс идёт по поддереву, поэтому читается дерево категорий.
	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(entity.Categories{}, true)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), testCategoryID)

	saved, err := srv.SaveAttribute(context.Background(), attribute)
	require.NoError(t, err)

	assert.Equal(t, testPowerAttrID, saved.ID)
}

// TestCatalogService_SaveAttribute_TypeChangeRejected фиксирует, что смена типа
// у характеристики с заполненными значениями отвергается составным внешним ключом.
func TestCatalogService_SaveAttribute_TypeChangeRejected(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		ID: testPowerAttrID, CategoryID: testCategoryID,
		Code: "engine_power", DataType: entity.AttributeTypeString,
	}

	mocks.attributes.EXPECT().Update(gomock.Any(), attribute).
		Return(entity.ErrAttributeValueTypeMismatch)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.SaveAttribute(context.Background(), attribute)

	require.ErrorIs(t, err, entity.ErrAttributeValueTypeMismatch)
}

// TestCatalogService_SaveAttribute_CreateFailure фиксирует проброс отказа создания.
func TestCatalogService_SaveAttribute_CreateFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		CategoryID: testCategoryID, Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	mocks.attributes.EXPECT().Create(gomock.Any(), attribute).Return(nil, entity.ErrAttributeCodeInvalid)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.SaveAttribute(context.Background(), attribute)

	require.ErrorIs(t, err, entity.ErrAttributeCodeInvalid)
}

// TestCatalogService_SaveAttribute_Nil фиксирует отказ на пустой характеристике.
func TestCatalogService_SaveAttribute_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	_, err := srv.SaveAttribute(context.Background(), nil)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestCatalogService_WorksWithoutCache фиксирует, что сервис собирается
// и работает вообще без кэша.
func TestCatalogService_WorksWithoutCache(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	categories := mock_repo.NewMockCategoryRepository(ctrl)
	attributes := mock_repo.NewMockAttributeRepository(ctrl)

	stored := entity.Categories{{ID: testCategoryID, Slug: "excavators"}}

	categories.EXPECT().Tree(gomock.Any()).Return(stored, nil)
	categories.EXPECT().CoverCandidates(gomock.Any()).Return(nil, nil)
	attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).Return(testAttributes(), nil)

	machines := mock_repo.NewMockMachineRepository(ctrl)
	machines.EXPECT().GetByID(gomock.Any(), testMachineID).Return(testMachine(false), nil)
	machines.EXPECT().SetPublication(gomock.Any(), gomock.Any()).Return(nil)

	srv := NewCatalogService(categories, attributes,
		machines,
		mock_repo.NewMockMachineAttributeValueRepository(ctrl),
		mock_repo.NewMockMachineImageRepository(ctrl),
		nil, nil, nil, repository.NewTxStub())

	tree, err := srv.CategoryTree(context.Background())
	require.NoError(t, err)
	assert.Equal(t, stored, tree)

	attrs, err := srv.CategoryAttributes(context.Background(), testCategoryID)
	require.NoError(t, err)
	assert.Len(t, attrs, 2)

	// Публикация сбрасывает кэш дерева категорий, потому что меняет
	// подставленную обложку раздела. Без кэша сбрасывать нечего,
	// и операция обязана пройти, а не упасть на пустой ссылке.
	published, err := srv.PublishMachine(context.Background(), testMachineID, testCatalogNow)
	require.NoError(t, err)
	assert.True(t, published.IsActive)
}

// TestCatalogService_DeleteCategory фиксирует удаление со сбросом кэша дерева.
func TestCatalogService_DeleteCategory(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.categories.EXPECT().Delete(gomock.Any(), testCategoryID).Return(nil)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	require.NoError(t, srv.DeleteCategory(context.Background(), testCategoryID))
}

// TestCatalogService_DeleteCategory_Failure фиксирует, что при отказе удаления
// кэш не сбрасывается: сбрасывать нечего, если ничего не изменилось.
func TestCatalogService_DeleteCategory_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.categories.EXPECT().Delete(gomock.Any(), testCategoryID).Return(entity.ErrCategoryHasChildren)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any()).Times(0)

	err := srv.DeleteCategory(context.Background(), testCategoryID)

	require.ErrorIs(t, err, entity.ErrCategoryHasChildren)
}

// TestCatalogService_DeleteAttribute фиксирует, что характеристика читается
// до удаления ради своей категории: после удаления строки эту связь
// уже не восстановить, а сбросить нужно кэш конкретной категории.
func TestCatalogService_DeleteAttribute(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{ID: testPowerAttrID, CategoryID: testCategoryID}

	mocks.attributes.EXPECT().GetByID(gomock.Any(), testPowerAttrID).Return(attribute, nil)
	mocks.attributes.EXPECT().Delete(gomock.Any(), testPowerAttrID).Return(nil)
	// Сброс идёт по поддереву, поэтому читается дерево категорий.
	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(entity.Categories{}, true)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), testCategoryID)

	require.NoError(t, srv.DeleteAttribute(context.Background(), testPowerAttrID))
}

// TestCatalogService_DeleteAttribute_NotFound фиксирует, что отсутствующая
// характеристика не доходит до удаления.
func TestCatalogService_DeleteAttribute_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.attributes.EXPECT().GetByID(gomock.Any(), testPowerAttrID).
		Return(nil, entity.ErrAttributeNotFound)
	mocks.attributes.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	err := srv.DeleteAttribute(context.Background(), testPowerAttrID)

	require.ErrorIs(t, err, entity.ErrAttributeNotFound)
}

// TestCatalogService_DeleteAttribute_DeleteFailure фиксирует проброс отказа удаления.
func TestCatalogService_DeleteAttribute_DeleteFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	deleteErr := errors.New("база недоступна")

	mocks.attributes.EXPECT().GetByID(gomock.Any(), testPowerAttrID).
		Return(&entity.Attribute{ID: testPowerAttrID, CategoryID: testCategoryID}, nil)
	mocks.attributes.EXPECT().Delete(gomock.Any(), testPowerAttrID).Return(deleteErr)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), gomock.Any()).Times(0)

	err := srv.DeleteAttribute(context.Background(), testPowerAttrID)

	require.ErrorIs(t, err, deleteErr)
}

// TestCatalogService_SaveAttribute_ReplacesOptions фиксирует, что варианты enum
// сохраняются вместе с характеристикой одной транзакцией.
func TestCatalogService_SaveAttribute_ReplacesOptions(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		CategoryID: testCategoryID, Code: "drive_type", DataType: entity.AttributeTypeEnum,
		Options: entity.AttributeOptions{{Code: "crawler"}, {Code: "wheeled"}},
	}
	created := &entity.Attribute{ID: testDriveAttrID, CategoryID: testCategoryID}

	mocks.attributes.EXPECT().Create(gomock.Any(), attribute).Return(created, nil)
	mocks.attributes.EXPECT().ReplaceOptions(gomock.Any(), testDriveAttrID, attribute.Options).Return(nil)
	// Сброс идёт по поддереву, поэтому читается дерево категорий.
	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(entity.Categories{}, true)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), testCategoryID)

	saved, err := srv.SaveAttribute(context.Background(), attribute)
	require.NoError(t, err)

	assert.Len(t, saved.Options, 2)
}

// TestCatalogService_SaveAttribute_NonEnumSkipsOptions фиксирует, что у нечисловых
// типов замена вариантов не вызывается: их не бывает, и лишний запрос ни к чему.
func TestCatalogService_SaveAttribute_NonEnumSkipsOptions(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		CategoryID: testCategoryID, Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	mocks.attributes.EXPECT().Create(gomock.Any(), attribute).Return(attribute, nil)
	mocks.attributes.EXPECT().ReplaceOptions(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	// Сброс идёт по поддереву, поэтому читается дерево категорий.
	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(entity.Categories{}, true)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), testCategoryID)

	_, err := srv.SaveAttribute(context.Background(), attribute)
	require.NoError(t, err)
}

// TestCatalogService_SaveAttribute_OptionsFailureAborts фиксирует, что отказ
// записи вариантов поднимается наверх и кэш не сбрасывается.
func TestCatalogService_SaveAttribute_OptionsFailureAborts(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	optionsErr := errors.New("база недоступна")

	attribute := &entity.Attribute{
		ID: testDriveAttrID, CategoryID: testCategoryID,
		Code: "drive_type", DataType: entity.AttributeTypeEnum,
		Options: entity.AttributeOptions{{Code: "crawler"}},
	}

	mocks.attributes.EXPECT().Update(gomock.Any(), attribute).Return(nil)
	mocks.attributes.EXPECT().ReplaceOptions(gomock.Any(), testDriveAttrID, attribute.Options).
		Return(optionsErr)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.SaveAttribute(context.Background(), attribute)

	require.ErrorIs(t, err, optionsErr)
}

// TestCatalogService_MachineImages фиксирует чтение фотографий карточки.
func TestCatalogService_MachineImages(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	stored := entity.MachineImages{{ID: "i1", Path: "a.jpg", IsPrimary: true}}

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).Return(stored, nil)

	images, err := srv.MachineImages(context.Background(), testMachineID)
	require.NoError(t, err)

	assert.Equal(t, stored, images)
}

// TestCatalogService_MachineImages_Failure фиксирует проброс отказа чтения.
func TestCatalogService_MachineImages_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	repoErr := errors.New("база недоступна")

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).Return(nil, repoErr)

	_, err := srv.MachineImages(context.Background(), testMachineID)

	require.ErrorIs(t, err, repoErr)
}

// TestCatalogService_ReplaceMachineImages фиксирует замену набора и перечитывание.
func TestCatalogService_ReplaceMachineImages(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	images := entity.MachineImages{{Path: "a.jpg", IsPrimary: true}, {Path: "b.jpg", Position: 1}}
	stored := entity.MachineImages{{ID: "i1", Path: "a.jpg", IsPrimary: true}}

	mocks.images.EXPECT().Replace(gomock.Any(), testMachineID, images).Return(nil)
	// Смена фотографий меняет подставленную обложку раздела.
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())
	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).Return(stored, nil)

	result, err := srv.ReplaceMachineImages(context.Background(), testMachineID, images)
	require.NoError(t, err)

	assert.Equal(t, stored, result, "возвращается перечитанный набор с идентификаторами")
}

// TestCatalogService_ReplaceMachineImages_TwoPrimaries фиксирует отказ на двух
// главных фотографиях до обращения к базе.
func TestCatalogService_ReplaceMachineImages_TwoPrimaries(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	images := entity.MachineImages{
		{Path: "a.jpg", IsPrimary: true},
		{Path: "b.jpg", IsPrimary: true},
	}

	mocks.images.EXPECT().Replace(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.ReplaceMachineImages(context.Background(), testMachineID, images)

	require.ErrorIs(t, err, entity.ErrMachineImagePrimaryConflict)
}

// TestCatalogService_ReplaceMachineImages_Failure фиксирует проброс отказа записи.
func TestCatalogService_ReplaceMachineImages_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	replaceErr := errors.New("база недоступна")
	images := entity.MachineImages{{Path: "a.jpg"}}

	mocks.images.EXPECT().Replace(gomock.Any(), testMachineID, images).Return(replaceErr)

	_, err := srv.ReplaceMachineImages(context.Background(), testMachineID, images)

	require.ErrorIs(t, err, replaceErr)
}

// TestCatalogService_SetPrimaryImage фиксирует назначение главного фото.
func TestCatalogService_SetPrimaryImage(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	stored := entity.MachineImages{{ID: "i2", Path: "b.jpg", IsPrimary: true}}

	mocks.images.EXPECT().SetPrimary(gomock.Any(), testMachineID, "i2").Return(nil)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())
	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).Return(stored, nil)

	images, err := srv.SetPrimaryImage(context.Background(), testMachineID, "i2")
	require.NoError(t, err)

	assert.Equal(t, stored, images)
}

// TestCatalogService_SetPrimaryImage_Failure фиксирует проброс отказа.
func TestCatalogService_SetPrimaryImage_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.images.EXPECT().SetPrimary(gomock.Any(), testMachineID, "i2").
		Return(entity.ErrMachineNotFound)

	_, err := srv.SetPrimaryImage(context.Background(), testMachineID, "i2")

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
}

// TestCatalogService_SaveAttribute_InvalidatesSubtree фиксирует, что правка
// характеристики у родителя сбрасывает кэш и у потомков.
//
// С тех пор как характеристики наследуются, набор родителя отдаётся каждому
// потомку: оставленный у них кэш показывал бы панель фильтров, которой уже нет.
func TestCatalogService_SaveAttribute_InvalidatesSubtree(t *testing.T) {
	t.Parallel()

	const (
		rootID  = "11111111-1111-4111-8111-111111111111"
		midID   = "22222222-2222-4222-8222-222222222222"
		leafID  = "33333333-3333-4333-8333-333333333333"
		otherID = "44444444-4444-4444-8444-444444444444"
	)

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		CategoryID: rootID, Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	mocks.attributes.EXPECT().Create(gomock.Any(), attribute).Return(attribute, nil)

	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(entity.Categories{
		{ID: rootID},
		{ID: midID, ParentID: entity.Pointer(rootID)},
		{ID: leafID, ParentID: entity.Pointer(midID)},
		{ID: otherID}, // Соседнее дерево — его трогать не нужно.
	}, true)

	// Сама категория и оба потомка вглубь, но не соседнее дерево.
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), rootID)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), midID)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), leafID)
	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), otherID).Times(0)

	_, err := srv.SaveAttribute(context.Background(), attribute)

	require.NoError(t, err)
}

// TestCatalogService_InvalidateSubtree_TreeFailureDoesNotBreakSave фиксирует,
// что недоступное дерево не отменяет уже выполненную операцию: кэш —
// ускоритель, и в худшем случае набор обновится по истечении срока жизни.
func TestCatalogService_InvalidateSubtree_TreeFailureDoesNotBreakSave(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	attribute := &entity.Attribute{
		CategoryID: testCategoryID, Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	mocks.attributes.EXPECT().Create(gomock.Any(), attribute).Return(attribute, nil)

	mocks.cache.EXPECT().DeleteCategoryAttributes(gomock.Any(), testCategoryID)
	mocks.cache.EXPECT().GetCategoryTree(gomock.Any()).Return(nil, false)
	mocks.categories.EXPECT().Tree(gomock.Any()).Return(nil, errors.New("база недоступна"))

	_, err := srv.SaveAttribute(context.Background(), attribute)

	require.NoError(t, err, "отказ чтения дерева не отменяет сохранение")
}

// TestDescendantIDs фиксирует обход потомков по плоскому списку категорий.
func TestDescendantIDs(t *testing.T) {
	t.Parallel()

	categories := entity.Categories{
		{ID: "root"},
		{ID: "a", ParentID: entity.Pointer("root")},
		{ID: "b", ParentID: entity.Pointer("root")},
		{ID: "a1", ParentID: entity.Pointer("a")},
		{ID: "a2", ParentID: entity.Pointer("a")},
		{ID: "a1x", ParentID: entity.Pointer("a1")},
		{ID: "other"},
	}

	assert.ElementsMatch(t, []string{"a", "b", "a1", "a2", "a1x"}, descendantIDs(categories, "root"))
	assert.ElementsMatch(t, []string{"a1", "a2", "a1x"}, descendantIDs(categories, "a"))
	assert.Empty(t, descendantIDs(categories, "a1x"), "у листа потомков нет")
	assert.Empty(t, descendantIDs(categories, "нет-такой"))
	assert.Empty(t, descendantIDs(nil, "root"))
}

// TestCatalogService_SaveAttribute_WithoutCache фиксирует, что сервис без кэша
// работает: сброс — вспомогательный эффект, а не часть операции.
func TestCatalogService_SaveAttribute_WithoutCache(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	attributes := mock_repo.NewMockAttributeRepository(ctrl)

	srv := NewCatalogService(
		mock_repo.NewMockCategoryRepository(ctrl), attributes,
		mock_repo.NewMockMachineRepository(ctrl),
		mock_repo.NewMockMachineAttributeValueRepository(ctrl),
		mock_repo.NewMockMachineImageRepository(ctrl),
		nil, nil, nil, repository.NewTxStub())

	attribute := &entity.Attribute{
		CategoryID: testCategoryID, Code: "engine_power", DataType: entity.AttributeTypeNumber,
	}

	attributes.EXPECT().Create(gomock.Any(), attribute).Return(attribute, nil)

	_, err := srv.SaveAttribute(context.Background(), attribute)

	require.NoError(t, err)
}

// TestCatalogService_SearchTotal фиксирует, что итог считается только
// по просьбе клиента: точный подсчёт — второй запрос по тому же предикату,
// и он обязан просмотреть все подходящие строки.
func TestCatalogService_SearchTotal(t *testing.T) {
	t.Parallel()

	t.Run("без просьбы не считается", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newCatalogService(t)

		mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Return(entity.Machines{testMachine(true)}, nil)
		mocks.machines.EXPECT().Count(gomock.Any(), gomock.Any()).Times(0)

		page, err := srv.SearchPublicMachines(context.Background(),
			&entity.MachineFilter{Sort: entity.SortPublishedDesc, Limit: 20})

		require.NoError(t, err)
		assert.Nil(t, page.Total, "не считали — не «ничего не нашли»")
	})

	t.Run("по просьбе считается", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newCatalogService(t)

		mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Return(entity.Machines{testMachine(true)}, nil)
		mocks.machines.EXPECT().Count(gomock.Any(), gomock.Any()).Return(137, nil)

		page, err := srv.SearchPublicMachines(context.Background(),
			&entity.MachineFilter{Sort: entity.SortPublishedDesc, Limit: 20, WithTotal: true})

		require.NoError(t, err)
		require.NotNil(t, page.Total)
		assert.Equal(t, 137, *page.Total)
	})

	t.Run("отказ подсчёта роняет ответ", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newCatalogService(t)

		mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Return(entity.Machines{}, nil)
		mocks.machines.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, errors.New("база недоступна"))

		_, err := srv.SearchPublicMachines(context.Background(),
			&entity.MachineFilter{Sort: entity.SortPublishedDesc, Limit: 20, WithTotal: true})

		require.Error(t, err, "клиент запросил число явно: неполный ответ был бы хуже отказа")
	})
}

// TestCatalogService_SearchTotal_AdminAlso фиксирует, что итог доступен
// и админскому поиску: флаг относится к выдаче, а не к её видимости.
func TestCatalogService_SearchTotal_AdminAlso(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Return(entity.Machines{}, nil)
	mocks.machines.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil)

	page, err := srv.SearchAllMachines(context.Background(),
		&entity.MachineFilter{Sort: entity.SortPublishedDesc, Limit: 20, WithTotal: true})

	require.NoError(t, err)
	require.NotNil(t, page.Total)
	assert.Equal(t, 0, *page.Total, "ноль найденных — это ответ, а не отсутствие ответа")
}

// TestCatalogService_Search_NilFilter фиксирует отказ на пустом фильтре
// в обоих методах поиска.
func TestCatalogService_Search_NilFilter(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	_, publicErr := srv.SearchPublicMachines(context.Background(), nil)
	require.ErrorIs(t, publicErr, entity.ErrNilPointer)

	_, adminErr := srv.SearchAllMachines(context.Background(), nil)
	require.ErrorIs(t, adminErr, entity.ErrNilPointer)
}

// TestResolveCovers фиксирует правила подстановки обложки раздела.
//
// Дерево: раздел -> два подраздела, у каждого своя машина. Обложка раздела
// обязана совпасть с той, что витрина покажет в нём первой, — иначе плитка
// раздела и первая карточка в нём окажутся разными снимками.
func TestResolveCovers(t *testing.T) {
	t.Parallel()

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	section := &entity.Category{ID: "s"}
	left := &entity.Category{ID: "l", ParentID: entity.Pointer("s")}
	right := &entity.Category{ID: "r", ParentID: entity.Pointer("s")}
	empty := &entity.Category{ID: "e", ParentID: entity.Pointer("s")}

	covers := map[string]entity.CategoryCover{
		"l": {Path: "machines/old.jpg", PublishedAt: &older, MachineID: "m1"},
		"r": {Path: "machines/new.jpg", PublishedAt: &newer, MachineID: "m2"},
	}

	resolveCovers(entity.Categories{section, left, right, empty}, covers)

	require.NotNil(t, section.CoverPath)
	assert.Equal(t, "machines/new.jpg", *section.CoverPath,
		"разделу достаётся машина, которую витрина показывает первой")
	assert.Equal(t, "machines/old.jpg", *left.CoverPath)
	assert.Equal(t, "machines/new.jpg", *right.CoverPath)
	assert.Nil(t, empty.CoverPath, "у пустого подраздела обложки нет")
}

// TestResolveCovers_ManualWins фиксирует, что выбранная вручную обложка
// сильнее подставленной: её выбрал человек, а подстановка — запасной вариант.
//
// И обратное: выбор у потомка не меняет обложку предка. Иначе редактор,
// поправив один подраздел, молча переписал бы плитку всего раздела.
func TestResolveCovers_ManualWins(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	section := &entity.Category{ID: "s"}
	child := &entity.Category{
		ID: "c", ParentID: entity.Pointer("s"),
		ImagePath: entity.Pointer("categories/hand-picked.jpg"),
	}

	covers := map[string]entity.CategoryCover{
		"c": {Path: "machines/auto.jpg", PublishedAt: &published, MachineID: "m1"},
	}

	resolveCovers(entity.Categories{section, child}, covers)

	assert.Equal(t, "categories/hand-picked.jpg", *child.CoverPath)
	require.NotNil(t, section.CoverPath)
	assert.Equal(t, "machines/auto.jpg", *section.CoverPath,
		"наверх уходит машина, а не выбор редактора у потомка")
}

// TestResolveCovers_Cycle фиксирует, что цикл в данных не вешает обход.
//
// Циклы отвергает триггер categories_reject_cycle_trg, но строка может попасть
// в данные в обход него — восстановлением дампа или прямым SQL.
func TestResolveCovers_Cycle(t *testing.T) {
	t.Parallel()

	first := &entity.Category{ID: "a", ParentID: entity.Pointer("b")}
	second := &entity.Category{ID: "b", ParentID: entity.Pointer("a")}

	resolveCovers(entity.Categories{first, second}, nil)

	assert.Nil(t, first.CoverPath)
	assert.Nil(t, second.CoverPath)
}

// testMachineImages набор фотографий карточки для тестов удаления.
func testMachineImages(paths ...string) entity.MachineImages {
	images := make(entity.MachineImages, 0, len(paths))
	for index, path := range paths {
		images = append(images, &entity.MachineImage{
			ID: fmt.Sprintf("image-%d", index), MachineID: testMachineID, Path: path,
		})
	}

	return images
}

// TestCatalogService_DeleteMachine_RemovesOnlyOrphanFiles фиксирует главное
// правило удаления файлов: один снимок может стоять у нескольких карточек,
// и с диска уходит только тот, на который больше никто не ссылается.
//
// В импортированном каталоге это не гипотеза: один файл стоит сразу у семи
// тоннажей одной модели, и удаление без проверки оставило бы шесть карточек
// с битой картинкой.
func TestCatalogService_DeleteMachine_RemovesOnlyOrphanFiles(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	own := "machines/own.jpg"
	shared := "machines/shared.jpg"

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).
		Return(testMachineImages(own, shared), nil)
	mocks.machines.EXPECT().Delete(gomock.Any(), testMachineID).Return(nil)
	// После удаления карточки на общий снимок всё ещё ссылается соседняя.
	mocks.images.EXPECT().UsedPaths(gomock.Any(), []string{own, shared}).
		Return([]string{shared}, nil)
	mocks.media.EXPECT().Delete(gomock.Any(), own).Return(nil)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	require.NoError(t, srv.DeleteMachine(context.Background(), testMachineID))
}

// TestCatalogService_DeleteMachine_KeepsFilesOnUnknownUsage фиксирует, что
// при отказе проверки ссылок файлы остаются на диске.
//
// Лишний файл — потерянные килобайты, удалённый чужой — потерянная работа.
func TestCatalogService_DeleteMachine_KeepsFilesOnUnknownUsage(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).
		Return(testMachineImages("machines/one.jpg"), nil)
	mocks.machines.EXPECT().Delete(gomock.Any(), testMachineID).Return(nil)
	mocks.images.EXPECT().UsedPaths(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("база недоступна"))
	mocks.media.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	require.NoError(t, srv.DeleteMachine(context.Background(), testMachineID))
}

// TestCatalogService_DeleteMachine_FileErrorDoesNotFailDeletion фиксирует, что
// отказ файловой системы не превращается в отказ операции: карточки в базе
// уже нет, и сказать клиенту «не удалось» значило бы соврать.
func TestCatalogService_DeleteMachine_FileErrorDoesNotFailDeletion(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).
		Return(testMachineImages("machines/one.jpg"), nil)
	mocks.machines.EXPECT().Delete(gomock.Any(), testMachineID).Return(nil)
	mocks.images.EXPECT().UsedPaths(gomock.Any(), gomock.Any()).Return(nil, nil)
	mocks.media.EXPECT().Delete(gomock.Any(), "machines/one.jpg").
		Return(errors.New("диск только для чтения"))
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	require.NoError(t, srv.DeleteMachine(context.Background(), testMachineID))
}

// TestCatalogService_DeleteMachine_QuotesBlockDeletion фиксирует, что при
// отказе удаления файлы не трогаются вовсе: карточка осталась на месте.
func TestCatalogService_DeleteMachine_QuotesBlockDeletion(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).
		Return(testMachineImages("machines/one.jpg"), nil)
	mocks.machines.EXPECT().Delete(gomock.Any(), testMachineID).
		Return(entity.ErrMachineHasQuotes)
	mocks.images.EXPECT().UsedPaths(gomock.Any(), gomock.Any()).Times(0)
	mocks.media.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any()).Times(0)

	err := srv.DeleteMachine(context.Background(), testMachineID)

	require.ErrorIs(t, err, entity.ErrMachineHasQuotes)
}

// TestCatalogService_DeleteMachine_WithoutImages фиксирует, что карточка
// без фотографий удаляется, не спрашивая хранилище ни о чём.
func TestCatalogService_DeleteMachine_WithoutImages(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).Return(nil, nil)
	mocks.machines.EXPECT().Delete(gomock.Any(), testMachineID).Return(nil)
	mocks.images.EXPECT().UsedPaths(gomock.Any(), gomock.Any()).Times(0)
	mocks.media.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
	mocks.cache.EXPECT().DeleteCategoryTree(gomock.Any())

	require.NoError(t, srv.DeleteMachine(context.Background(), testMachineID))
}

// TestCatalogService_DeleteMachine_ImagesUnreadable фиксирует, что удаление
// не начинается, пока неизвестно, какие файлы принадлежали карточке.
func TestCatalogService_DeleteMachine_ImagesUnreadable(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.images.EXPECT().ListByMachine(gomock.Any(), testMachineID).
		Return(nil, errors.New("база недоступна"))
	mocks.machines.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	require.Error(t, srv.DeleteMachine(context.Background(), testMachineID))
}

// testResidualCurve собирает валидную кривую.
func testResidualCurve() *entity.ResidualValueCurve {
	return &entity.ResidualValueCurve{
		CategoryID:       testCategoryID,
		FirstYearDropBps: 2000,
		AnnualDropBps:    1200,
		FloorBps:         2000,
	}
}

// TestCatalogService_ResidualCurves отдаёт список кривых.
func TestCatalogService_ResidualCurves(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.residual.EXPECT().List(gomock.Any()).
		Return([]*entity.ResidualValueCurve{testResidualCurve()}, nil)

	curves, err := srv.ResidualCurves(context.Background())
	require.NoError(t, err)
	require.Len(t, curves, 1)
	assert.Equal(t, testCategoryID, curves[0].CategoryID)
}

// TestCatalogService_ResidualCurves_Failure заворачивает отказ репозитория.
func TestCatalogService_ResidualCurves_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.residual.EXPECT().List(gomock.Any()).Return(nil, errors.New("база недоступна"))

	_, err := srv.ResidualCurves(context.Background())
	require.Error(t, err)
}

// TestCatalogService_ResidualCurveByCategory отдаёт кривую раздела.
//
// Именно GetByCategory, а не CurveFor: справочник показывает свою кривую,
// а не унаследованную от предка.
func TestCatalogService_ResidualCurveByCategory(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.residual.EXPECT().GetByCategory(gomock.Any(), testCategoryID).
		Return(testResidualCurve(), nil)

	curve, err := srv.ResidualCurveByCategory(context.Background(), testCategoryID)
	require.NoError(t, err)
	assert.Equal(t, 2000, curve.FirstYearDropBps)
}

// TestCatalogService_ResidualCurveByCategory_NotFound пропускает отказ наружу.
func TestCatalogService_ResidualCurveByCategory_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.residual.EXPECT().GetByCategory(gomock.Any(), testCategoryID).
		Return(nil, entity.ErrResidualCurveNotFound)

	_, err := srv.ResidualCurveByCategory(context.Background(), testCategoryID)
	require.ErrorIs(t, err, entity.ErrResidualCurveNotFound)
}

// TestCatalogService_SaveResidualCurve сохраняет кривую.
func TestCatalogService_SaveResidualCurve(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	curve := testResidualCurve()
	mocks.residual.EXPECT().Save(gomock.Any(), curve).Return(curve, nil)

	saved, err := srv.SaveResidualCurve(context.Background(), curve)
	require.NoError(t, err)
	assert.Equal(t, testCategoryID, saved.CategoryID)
}

// TestCatalogService_SaveResidualCurve_Invalid не пускает кривую в базу.
//
// Проверка до репозитория: порог выше остатка первого года сделал бы кривую
// возрастающей, и техника «дорожала» бы со второго года.
func TestCatalogService_SaveResidualCurve_Invalid(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	curve := testResidualCurve()
	curve.FloorBps = 9000

	_, err := srv.SaveResidualCurve(context.Background(), curve)
	require.ErrorIs(t, err, entity.ErrResidualCurveFloorAboveFirstYear)
}

// TestCatalogService_SaveResidualCurve_Nil отказывает на пустом указателе.
func TestCatalogService_SaveResidualCurve_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newCatalogService(t)

	_, err := srv.SaveResidualCurve(context.Background(), nil)
	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestCatalogService_SaveResidualCurve_Failure пропускает отказ репозитория.
func TestCatalogService_SaveResidualCurve_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.residual.EXPECT().Save(gomock.Any(), gomock.Any()).
		Return(nil, entity.ErrCategoryNotFound)

	_, err := srv.SaveResidualCurve(context.Background(), testResidualCurve())
	require.ErrorIs(t, err, entity.ErrCategoryNotFound)
}

// TestCatalogService_DeleteResidualCurve убирает кривую раздела.
func TestCatalogService_DeleteResidualCurve(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.residual.EXPECT().Delete(gomock.Any(), testCategoryID).Return(nil)

	require.NoError(t, srv.DeleteResidualCurve(context.Background(), testCategoryID))
}

// TestCatalogService_SearchRanksByRelevance фиксирует главное: с текстом
// запроса порядок подменяется на релевантность, и делает это сервис сам.
//
// Клиент про такой порядок не знает — в форме поиска его нет и быть не может:
// без запроса он недействителен.
func TestCatalogService_SearchRanksByRelevance(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		query    string
		sort     entity.MachineSort
		expected entity.MachineSort
	}{
		{
			name:     "запрос без явного порядка ранжируется",
			query:    "экскаватор",
			sort:     "",
			expected: entity.SortRelevance,
		},
		{
			name:     "порядок по умолчанию тоже подменяется",
			query:    "экскаватор",
			sort:     entity.SortPublishedDesc,
			expected: entity.SortRelevance,
		},
		{
			// Явно выбранный порядок — это решение человека, и подменять
			// его нельзя: он попросил дешёвые сверху, а не подходящие.
			name:     "явная сортировка по цене остаётся",
			query:    "экскаватор",
			sort:     entity.SortPriceAsc,
			expected: entity.SortPriceAsc,
		},
		{
			name:     "без запроса ранжировать нечем",
			query:    "",
			sort:     "",
			expected: entity.SortPublishedDesc,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newCatalogService(t)

			var used entity.MachineSort

			mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
					used = f.Sort

					return entity.Machines{testMachine(true)}, nil
				})

			_, err := srv.SearchPublicMachines(context.Background(), &entity.MachineFilter{
				Query: c.query, Sort: c.sort, Limit: 10,
			})
			require.NoError(t, err)

			assert.Equal(t, c.expected, used)
		})
	}
}

// TestCatalogService_SearchFallsBackToFuzzy фиксирует запасной путь: пустая
// выдача по тексту запроса переспрашивается триграммами.
//
// Именно вторым запросом, а не одним с OR: постоянно включённые триграммы
// тащили бы в точную выдачу похожее.
func TestCatalogService_SearchFallsBackToFuzzy(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	var attempts []bool

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Times(2).
		DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
			attempts = append(attempts, f.Fuzzy)

			if f.Fuzzy {
				return entity.Machines{testMachine(true)}, nil
			}

			return nil, nil
		})

	page, err := srv.SearchPublicMachines(context.Background(),
		&entity.MachineFilter{Query: "эскаватор", Limit: 10})
	require.NoError(t, err)

	assert.Equal(t, []bool{false, true}, attempts, "сначала точный путь, потом запасной")
	assert.Len(t, page.Machines, 1)
}

// TestCatalogService_SearchKeepsFuzzyFromCursor фиксирует, что путь поиска
// живёт в курсоре: продолжение нечёткой выдачи идёт тем же путём.
//
// Иначе вторая страница «эскаватора» собралась бы точным поиском — то есть
// оказалась бы пустой, и выдача оборвалась бы на первой странице.
func TestCatalogService_SearchKeepsFuzzyFromCursor(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	var used bool

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
			used = f.Fuzzy

			return entity.Machines{testMachine(true)}, nil
		})

	_, err := srv.SearchPublicMachines(context.Background(), &entity.MachineFilter{
		Query: "эскаватор",
		Sort:  entity.SortRelevance,
		Limit: 10,
		Cursor: &entity.MachineCursor{
			Sort:      entity.SortRelevance,
			Relevance: entity.Pointer(2087),
			Fuzzy:     true,
			ID:        testMachineID,
		},
	})
	require.NoError(t, err)

	assert.True(t, used, "путь взят из курсора, а не выбран заново")
}

// TestCatalogService_SearchDoesNotRetryOnCursor фиксирует обратное: пустая
// страница продолжения — это конец выдачи, а не промах.
func TestCatalogService_SearchDoesNotRetryOnCursor(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	// Ровно один вызов: переспрашивать продолжение нельзя.
	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

	page, err := srv.SearchPublicMachines(context.Background(), &entity.MachineFilter{
		Query: "эскаватор",
		Sort:  entity.SortRelevance,
		Limit: 10,
		Cursor: &entity.MachineCursor{
			Sort:      entity.SortRelevance,
			Relevance: entity.Pointer(2087),
			ID:        testMachineID,
		},
	})
	require.NoError(t, err)

	assert.Empty(t, page.Machines)
}

// TestCatalogService_SearchFuzzyFailure заворачивает отказ запасного запроса.
func TestCatalogService_SearchFuzzyFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.machines.EXPECT().Search(gomock.Any(), gomock.Any()).Times(2).
		DoAndReturn(func(_ context.Context, f *entity.MachineFilter) (entity.Machines, error) {
			if f.Fuzzy {
				return nil, errors.New("база недоступна")
			}

			return nil, nil
		})

	_, err := srv.SearchPublicMachines(context.Background(),
		&entity.MachineFilter{Query: "эскаватор", Limit: 10})

	require.Error(t, err)
}

// TestCatalogService_PublicCategoryAttributes дочитывает границы значений
// к характеристикам раздела.
//
// Границы получают только числовые: подсказка «от — до» под флажком enum
// бессмысленна.
func TestCatalogService_PublicCategoryAttributes(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(nil, false)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).Return(testAttributes(), nil)
	mocks.cache.EXPECT().SetCategoryAttributes(gomock.Any(), testCategoryID, gomock.Any())
	mocks.values.EXPECT().NumberBounds(gomock.Any(), testCategoryID).Return(
		map[string]entity.AttributeBounds{
			testPowerAttrID: {Min: decimal.NewFromInt(55), Max: decimal.NewFromInt(420)},
		}, nil)

	attributes, err := srv.PublicCategoryAttributes(context.Background(), testCategoryID)
	require.NoError(t, err)

	byID := make(map[string]*entity.Attribute, len(attributes))
	for _, attribute := range attributes {
		byID[attribute.ID] = attribute
	}

	power := byID[testPowerAttrID]
	require.NotNil(t, power.NumberMin)
	assert.Equal(t, "55", power.NumberMin.String())
	assert.Equal(t, "420", power.NumberMax.String())

	// Характеристика типа enum границ не получает.
	drive := byID[testDriveAttrID]
	assert.Nil(t, drive.NumberMin)
}

// TestCatalogService_PublicCategoryAttributes_BoundsFailure — отказ границ
// не роняет фильтры.
//
// Панель без подсказок работает ровно так, как работала раньше, и терять
// её целиком из-за неполученной подписи нельзя.
func TestCatalogService_PublicCategoryAttributes_BoundsFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(nil, false)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).Return(testAttributes(), nil)
	mocks.cache.EXPECT().SetCategoryAttributes(gomock.Any(), testCategoryID, gomock.Any())
	mocks.values.EXPECT().NumberBounds(gomock.Any(), testCategoryID).
		Return(nil, errors.New("база недоступна"))

	attributes, err := srv.PublicCategoryAttributes(context.Background(), testCategoryID)

	require.NoError(t, err)
	assert.NotEmpty(t, attributes)
}

// TestCatalogService_PublicCategoryAttributes_ListFailure — отказ самого
// списка характеристик отдаётся наружу.
//
// В отличие от границ: без подсказок панель фильтров работает, без
// характеристик её не из чего собрать, и молча вернуть пустой список
// значило бы показать раздел без фильтров как раздел, где их нет.
func TestCatalogService_PublicCategoryAttributes_ListFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newCatalogService(t)

	mocks.cache.EXPECT().GetCategoryAttributes(gomock.Any(), testCategoryID).Return(nil, false)
	mocks.attributes.EXPECT().ListByCategory(gomock.Any(), testCategoryID).
		Return(nil, errors.New("база недоступна"))

	attributes, err := srv.PublicCategoryAttributes(context.Background(), testCategoryID)

	require.Error(t, err)
	assert.Nil(t, attributes)
}

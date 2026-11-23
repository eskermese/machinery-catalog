package entity

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMachineSort_IsValid фиксирует закрытый перечень сортировок.
func TestMachineSort_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, SortPriceAsc.IsValid())
	assert.True(t, SortPriceDesc.IsValid())
	assert.True(t, SortPublishedDesc.IsValid())
	assert.False(t, MachineSort("rating_desc").IsValid())
	assert.False(t, MachineSort("").IsValid())
	assert.Equal(t, "price_asc", SortPriceAsc.String())
}

// TestMachineCursor_RoundTrip фиксирует, что курсор переживает кодирование
// и разбор без потери позиции.
func TestMachineCursor_RoundTrip(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		cursor MachineCursor
	}{
		{
			name:   "по цене",
			cursor: MachineCursor{Sort: SortPriceAsc, PriceCNYCents: Pointer(int64(7_500_000)), ID: "machine-1"},
		},
		{
			name:   "по дате публикации",
			cursor: MachineCursor{Sort: SortPublishedDesc, PublishedAt: &publishedAt, ID: "machine-2"},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := c.cursor.Encode()
			require.NoError(t, err)
			assert.NotContains(t, encoded, "machine-", "курсор непрозрачен для клиента")

			decoded, err := DecodeMachineCursor(encoded)
			require.NoError(t, err)
			require.NotNil(t, decoded)

			assert.Equal(t, c.cursor.Sort, decoded.Sort)
			assert.Equal(t, c.cursor.ID, decoded.ID)
			assert.Equal(t, c.cursor.PriceCNYCents, decoded.PriceCNYCents)

			if c.cursor.PublishedAt != nil {
				require.NotNil(t, decoded.PublishedAt)
				assert.True(t, c.cursor.PublishedAt.Equal(*decoded.PublishedAt))
			}
		})
	}
}

// TestDecodeMachineCursor_Invalid фиксирует отказы разбора курсора.
func TestDecodeMachineCursor_Invalid(t *testing.T) {
	t.Parallel()

	validPayload, err := MachineCursor{Sort: SortPriceAsc, ID: "machine-1"}.Encode()
	require.NoError(t, err)

	noID, err := MachineCursor{Sort: SortPriceAsc}.Encode()
	require.NoError(t, err)

	unknownSort, err := MachineCursor{Sort: "rating_desc", ID: "machine-1"}.Encode()
	require.NoError(t, err)

	// Курсор без ключа publishedAt вовсе: собран не нами. Продолжать выдачу
	// не от чего, и без отказа она начиналась бы с нулевого времени,
	// то есть с начала каталога — молча и на каждой странице.
	noDateKey := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"sort":"published_desc","id":"machine-1"}`))

	// А вот пустая дата ключом со значением null — законная позиция:
	// так выглядит курсор, остановившийся на черновике.
	nullDate, err := MachineCursor{Sort: SortPublishedDesc, ID: "machine-1"}.Encode()
	require.NoError(t, err)

	cases := []struct {
		name    string
		encoded string
		expNil  bool
		expErr  bool
	}{
		{name: "пустая строка — первая страница", encoded: "", expNil: true},
		{name: "корректный курсор", encoded: validPayload},
		{name: "не base64", encoded: "!!!", expErr: true},
		{name: "не json", encoded: "aGVsbG8", expErr: true},
		{name: "без идентификатора", encoded: noID, expErr: true},
		{name: "неизвестная сортировка", encoded: unknownSort, expErr: true},
		{name: "сортировка по дате без ключа даты", encoded: noDateKey, expErr: true},
		{name: "сортировка по дате с пустой датой", encoded: nullDate},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			cursor, err := DecodeMachineCursor(c.encoded)

			if c.expErr {
				require.ErrorIs(t, err, ErrCatalogCursorInvalid)

				return
			}

			require.NoError(t, err)

			if c.expNil {
				assert.Nil(t, cursor)

				return
			}

			assert.NotNil(t, cursor)
		})
	}
}

// TestMachineFilter_Validate фиксирует умолчания и границы параметров поиска.
func TestMachineFilter_Validate(t *testing.T) {
	t.Parallel()

	t.Run("умолчания", func(t *testing.T) {
		t.Parallel()

		filter := MachineFilter{}
		require.NoError(t, filter.Validate())

		assert.Equal(t, SortPublishedDesc, filter.Sort, "по умолчанию свежие сверху")
		assert.Equal(t, DefaultMachineLimit, filter.Limit)
	})

	cases := []struct {
		name   string
		filter MachineFilter
		expErr error
	}{
		{name: "предельный размер страницы", filter: MachineFilter{Limit: MaxMachineLimit}},
		{
			name:   "размер страницы сверх предела",
			filter: MachineFilter{Limit: MaxMachineLimit + 1},
			expErr: ErrCatalogLimitInvalid,
		},
		{name: "отрицательный размер", filter: MachineFilter{Limit: -1}, expErr: ErrCatalogLimitInvalid},
		{name: "неизвестная сортировка", filter: MachineFilter{Sort: "rating"}, expErr: ErrCatalogSortInvalid},
		{
			name: "курсор от другой сортировки",
			filter: MachineFilter{
				Sort:   SortPriceAsc,
				Cursor: &MachineCursor{Sort: SortPriceDesc, ID: "machine-1"},
			},
			expErr: ErrCatalogCursorInvalid,
		},
		{
			name: "курсор своей сортировки",
			filter: MachineFilter{
				Sort:   SortPriceAsc,
				Cursor: &MachineCursor{Sort: SortPriceAsc, ID: "machine-1"},
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			filter := c.filter
			err := filter.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestMachineFilter_Validate_Nil фиксирует отказ на пустом фильтре.
func TestMachineFilter_Validate_Nil(t *testing.T) {
	t.Parallel()

	var filter *MachineFilter

	require.ErrorIs(t, filter.Validate(), ErrNilPointer)
}

// TestAttributeConditions фиксирует конструкторы условий и признак пустоты.
func TestAttributeConditions(t *testing.T) {
	t.Parallel()

	minValue := decimal.NewFromInt(100)

	numberRange := NewNumberRangeCondition("attr-1", &minValue, nil)
	assert.Equal(t, "attr-1", numberRange.TargetAttributeID())
	assert.False(t, numberRange.IsEmpty())
	assert.True(t, NewNumberRangeCondition("attr-1", nil, nil).IsEmpty())

	enum := NewEnumCondition("attr-2", "crawler", "wheeled")
	assert.Equal(t, "attr-2", enum.TargetAttributeID())
	assert.False(t, enum.IsEmpty())
	assert.True(t, NewEnumCondition("attr-2").IsEmpty())

	flag := NewBoolCondition("attr-3", true)
	assert.Equal(t, "attr-3", flag.TargetAttributeID())
}

// TestNewMachinePage фиксирует сборку страницы из выдачи с запасом в одну строку.
func TestNewMachinePage(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

	machines := Machines{
		{ID: "m1", PriceCNYCents: 100, IsPricePublic: true, PublishedAt: &publishedAt},
		{ID: "m2", PriceCNYCents: 200, IsPricePublic: true, PublishedAt: &publishedAt},
		{ID: "m3", PriceCNYCents: 300, IsPricePublic: true, PublishedAt: &publishedAt},
	}

	t.Run("есть продолжение", func(t *testing.T) {
		t.Parallel()

		page, err := NewMachinePage(machines, 2, &MachineFilter{Sort: SortPriceAsc})
		require.NoError(t, err)

		assert.Len(t, page.Machines, 2, "лишняя строка в выдачу не попадает")
		assert.True(t, page.HasMore)

		cursor, err := DecodeMachineCursor(page.NextCursor)
		require.NoError(t, err)
		assert.Equal(t, "m2", cursor.ID, "курсор указывает на последнюю строку страницы")
		require.NotNil(t, cursor.PriceCNYCents)
		assert.Equal(t, int64(200), *cursor.PriceCNYCents)
	})

	t.Run("последняя страница", func(t *testing.T) {
		t.Parallel()

		page, err := NewMachinePage(machines, 3, &MachineFilter{Sort: SortPriceAsc})
		require.NoError(t, err)

		assert.Len(t, page.Machines, 3)
		assert.False(t, page.HasMore)
		assert.Empty(t, page.NextCursor, "курсора у последней страницы нет")
	})

	t.Run("пустая выдача", func(t *testing.T) {
		t.Parallel()

		page, err := NewMachinePage(nil, 20, &MachineFilter{Sort: SortPriceAsc})
		require.NoError(t, err)

		assert.Empty(t, page.Machines)
		assert.False(t, page.HasMore)
		assert.Empty(t, page.NextCursor)
	})
}

// TestDecodeMachineCursor_PriceKey фиксирует обращение с ключом цены.
//
// Скрытая цена — законная позиция курсора, и выражена она значением null.
// Отсутствие ключа означает курсор, собранный не нами, и такой отвергается:
// иначе продолжение выдачи молча уехало бы к началу каталога.
func TestDecodeMachineCursor_PriceKey(t *testing.T) {
	t.Parallel()

	encode := func(payload string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(payload))
	}

	t.Run("цена null — законная позиция", func(t *testing.T) {
		t.Parallel()

		cursor, err := DecodeMachineCursor(
			encode(`{"sort":"price_asc","priceCnyCents":null,"id":"m1"}`))
		require.NoError(t, err)
		assert.Nil(t, cursor.PriceCNYCents, "карточка без цены")
		assert.Equal(t, "m1", cursor.ID)
	})

	t.Run("ключа цены нет — курсор отвергается", func(t *testing.T) {
		t.Parallel()

		_, err := DecodeMachineCursor(encode(`{"sort":"price_desc","id":"m1"}`))
		require.ErrorIs(t, err, ErrCatalogCursorInvalid)
	})

	t.Run("цена не число", func(t *testing.T) {
		t.Parallel()

		_, err := DecodeMachineCursor(
			encode(`{"sort":"price_asc","priceCnyCents":"дорого","id":"m1"}`))
		require.ErrorIs(t, err, ErrCatalogCursorInvalid)
	})

	t.Run("сортировка по дате цену не требует", func(t *testing.T) {
		t.Parallel()

		cursor, err := DecodeMachineCursor(
			encode(`{"sort":"published_desc","publishedAt":null,"id":"m1"}`))
		require.NoError(t, err)
		assert.Nil(t, cursor.PriceCNYCents)
	})
}

// TestMachineSort_IsByPrice фиксирует, какие порядки задаются ценой.
//
// От этого зависит, требовать ли у курсора ключ цены, и перечислять обе
// сортировки в каждом месте — способ однажды забыть одну.
func TestMachineSort_IsByPrice(t *testing.T) {
	t.Parallel()

	assert.True(t, SortPriceAsc.IsByPrice())
	assert.True(t, SortPriceDesc.IsByPrice())
	assert.False(t, SortPublishedDesc.IsByPrice())
}

// TestNewMachinePage_HiddenPriceCursor фиксирует курсор на карточке без цены.
//
// Значение цены в такой курсор не попадает: скрытой цены для выдачи
// не существует, и подставить её значило бы продолжить с позиции,
// которой в порядке нет.
func TestNewMachinePage_HiddenPriceCursor(t *testing.T) {
	t.Parallel()

	machines := Machines{
		{ID: "m1", PriceCNYCents: 100, IsPricePublic: true},
		{ID: "m2", PriceCNYCents: 900, IsPricePublic: false},
		{ID: "m3", PriceCNYCents: 300, IsPricePublic: true},
	}

	page, err := NewMachinePage(machines, 2, &MachineFilter{Sort: SortPriceAsc})
	require.NoError(t, err)

	cursor, err := DecodeMachineCursor(page.NextCursor)
	require.NoError(t, err)

	assert.Equal(t, "m2", cursor.ID)
	assert.Nil(t, cursor.PriceCNYCents, "тайная цена в курсор не уходит")
}

// TestNewMachinePage_RelevanceCursor фиксирует курсор ранжированной выдачи:
// в нём и релевантность последней строки, и выбранный путь поиска.
//
// Без релевантности продолжение сравнивать не с чем, без пути — вторая
// страница собралась бы другим способом, то есть по другой выдаче.
func TestNewMachinePage_RelevanceCursor(t *testing.T) {
	t.Parallel()

	machines := Machines{
		{ID: "11111111-1111-4111-8111-111111111111", Relevance: 5087},
		{ID: "22222222-2222-4222-8222-222222222222", Relevance: 2061},
		{ID: "33333333-3333-4333-8333-333333333333", Relevance: 1000},
	}

	page, err := NewMachinePage(machines, 2, &MachineFilter{Sort: SortRelevance, Fuzzy: true})
	require.NoError(t, err)
	require.True(t, page.HasMore)

	cursor, err := DecodeMachineCursor(page.NextCursor)
	require.NoError(t, err)

	assert.Equal(t, SortRelevance, cursor.Sort)
	require.NotNil(t, cursor.Relevance)
	assert.Equal(t, 2061, *cursor.Relevance, "релевантность ПОСЛЕДНЕЙ показанной строки")
	assert.True(t, cursor.Fuzzy, "путь поиска переносится в курсор")
}

// TestNewMachinePage_RelevanceOnlyInItsOwnOrder — проверка того, что
// релевантность не попадает в курсор чужого порядка: там её не считали,
// и ноль означал бы позицию, которой в выдаче нет.
func TestNewMachinePage_RelevanceOnlyInItsOwnOrder(t *testing.T) {
	t.Parallel()

	machines := Machines{
		{ID: "11111111-1111-4111-8111-111111111111", IsPricePublic: true, PriceCNYCents: 100},
		{ID: "22222222-2222-4222-8222-222222222222", IsPricePublic: true, PriceCNYCents: 200},
	}

	page, err := NewMachinePage(machines, 1, &MachineFilter{Sort: SortPriceAsc})
	require.NoError(t, err)

	cursor, err := DecodeMachineCursor(page.NextCursor)
	require.NoError(t, err)

	assert.Nil(t, cursor.Relevance)
	assert.False(t, cursor.Fuzzy)
}

// TestDecodeMachineCursor_RelevanceRequired отвергает курсор порядка
// по релевантности без самой релевантности: продолжать выдачу не от чего.
func TestDecodeMachineCursor_RelevanceRequired(t *testing.T) {
	t.Parallel()

	encoded, err := MachineCursor{
		Sort: SortRelevance,
		ID:   "11111111-1111-4111-8111-111111111111",
	}.Encode()
	require.NoError(t, err)

	// Encode кладёт relevance значением null; подменяем его на отсутствие ключа.
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	delete(payload, "relevance")

	stripped, err := json.Marshal(payload)
	require.NoError(t, err)

	_, err = DecodeMachineCursor(base64.RawURLEncoding.EncodeToString(stripped))

	require.ErrorIs(t, err, ErrCatalogCursorInvalid)
}

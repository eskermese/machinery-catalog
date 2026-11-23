package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// stubRow строка выдачи для проверки сканера без базы.
//
// Значения раздаются по порядку и присваиваются приёмникам напрямую: сканеру
// от строки нужен только Scan, а разбор форматов pgx здесь не проверяется —
// проверяется, как сканер обходится с NULL-ами левого соединения.
type stubRow struct {
	values []any
}

func (r stubRow) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r stubRow) Values() ([]any, error)                       { return r.values, nil }
func (r stubRow) RawValues() [][]byte                          { return nil }

func (r stubRow) Scan(dest ...any) error {
	for i, target := range dest {
		assign(target, r.values[i])
	}

	return nil
}

// assign копирует значение в приёмник нужного типа.
func assign(target, value any) {
	switch dst := target.(type) {
	case *string:
		*dst = value.(string)
	case *bool:
		*dst = value.(bool)
	case *int:
		*dst = value.(int)
	case *int64:
		*dst = value.(int64)
	case *time.Time:
		*dst = value.(time.Time)
	case *entity.MachineAvailability:
		*dst = value.(entity.MachineAvailability)
	case *entity.MachineCondition:
		*dst = value.(entity.MachineCondition)
	case **string:
		*dst, _ = value.(*string)
	case **int:
		*dst, _ = value.(*int)
	case **int64:
		*dst, _ = value.(*int64)
	case **decimal.Decimal:
		*dst, _ = value.(*decimal.Decimal)
	case **time.Time:
		*dst, _ = value.(*time.Time)
	}
}

// machineRowValues собирает строку выдачи: карточка плюс колонки фотографии.
func machineRowValues(image []any) []any {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

	row := []any{
		"machine-1", "category-1", "XCMG", "XE215C", "xcmg-xe215c",
		"Экскаватор", "Экскаватор", "Excavator",
		(*string)(nil), (*string)(nil), (*string)(nil),
		int64(8_950_000), true, entity.AvailabilityInStock, (*int)(nil),
		(*int)(nil), entity.ConditionNew,
		// Величины стоимости владения: у карточки их может не быть,
		// и незаполненность здесь — обычный случай, а не край.
		(*decimal.Decimal)(nil), (*int)(nil), (*int64)(nil),
		true, (*time.Time)(nil),
		now, now,
		// Релевантность: выдача поиска отдаёт её всегда, нулём — когда
		// ранжировать было нечем.
		0,
	}

	return append(row, image...)
}

// TestScanMachineSearchRow_WithPrimaryImage фиксирует чтение дочитанной фотографии.
func TestScanMachineSearchRow_WithPrimaryImage(t *testing.T) {
	t.Parallel()

	alt := "Экскаватор на площадке"
	row := stubRow{values: machineRowValues([]any{
		entity.Pointer("img-1"), entity.Pointer("machines/a.jpg"), entity.Pointer(2),
		&alt, (*string)(nil), (*string)(nil),
	})}

	machine, err := scanMachineSearchRow(row)
	require.NoError(t, err)

	require.NotNil(t, machine.PrimaryImage)
	assert.Equal(t, "img-1", machine.PrimaryImage.ID)
	assert.Equal(t, "machines/a.jpg", machine.PrimaryImage.Path)
	assert.Equal(t, 2, machine.PrimaryImage.Position)
	assert.Equal(t, "machine-1", machine.PrimaryImage.MachineID, "связь проставляется из карточки")
	assert.True(t, machine.PrimaryImage.IsPrimary, "соединение отбирает только главные")
	assert.Equal(t, &alt, machine.PrimaryImage.AltRU)

	// Галерея выдачей поиска не читается: в списке она не нужна.
	assert.Empty(t, machine.Images)
}

// TestScanMachineSearchRow_WithoutImage фиксирует карточку без фотографий.
//
// Левое соединение отдаёт NULL во всех колонках фотографии, включая
// объявленные в схеме NOT NULL, — и это обычный случай, а не ошибка.
func TestScanMachineSearchRow_WithoutImage(t *testing.T) {
	t.Parallel()

	row := stubRow{values: machineRowValues([]any{
		(*string)(nil), (*string)(nil), (*int)(nil),
		(*string)(nil), (*string)(nil), (*string)(nil),
	})}

	machine, err := scanMachineSearchRow(row)
	require.NoError(t, err)

	assert.Nil(t, machine.PrimaryImage)
	assert.Equal(t, "xcmg-xe215c", machine.Slug, "карточка читается как обычно")
}

// TestUsedPaths_EmptyInputSkipsQuery фиксирует, что пустой набор путей
// не доходит до базы.
//
// Проверка не декоративная: метод зовут после удаления карточки, у которой
// фотографий могло не быть вовсе, а `path = ANY('{}')` — запрос, который
// заведомо ничего не найдёт. Репозиторий здесь без пула: если бы запрос
// всё-таки уходил, тест упал бы на разыменовании nil.
func TestUsedPaths_EmptyInputSkipsQuery(t *testing.T) {
	t.Parallel()

	used, err := machineImageRepository{}.UsedPaths(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, used)
}

// TestResidualCurveQuery фиксирует правило поиска кривой остаточной стоимости:
// идём по цепочке предков и берём ближайшую заведённую.
//
// Проверяется сам запрос, а не результат: правило выражено порядком по depth
// и пределом в одну строку, и сломать его правкой соседней строки не должно
// быть возможно молча.
func TestResidualCurveQuery(t *testing.T) {
	t.Parallel()

	query := residualCurveQuery()

	assert.Contains(t, query, "WITH RECURSIVE ancestors", "кривая ищется по предкам")
	assert.Contains(t, query, "ORDER BY a.depth", "побеждает ближайший предок")
	assert.Contains(t, query, "LIMIT 1")
	assert.Contains(t, query, "residual_value_curves")
	// Возраста в запросе нет и быть не должно: кривая заводится на раздел
	// целиком, а долю за конкретный год считает уже сущность.
	assert.NotContains(t, query, "age_years")
}

// TestResidualCurveSaveQuery фиксирует правило «завести или переписать».
//
// У раздела кривая ровно одна, поэтому создание и правка снаружи не различаются.
// Голый INSERT вместо ON CONFLICT начал бы отказывать редактору при второй
// правке того же раздела, и заметно это было бы только на живом стенде.
func TestResidualCurveSaveQuery(t *testing.T) {
	t.Parallel()

	query := residualCurveSaveQuery()

	assert.Contains(t, query, "residual_value_curves")
	assert.Contains(t, query, "ON CONFLICT (category_id) DO UPDATE",
		"вторая правка того же раздела переписывает кривую, а не падает")
	assert.Contains(t, query, "updated_at = now()")
	assert.Contains(t, query, "RETURNING", "ответ отдаёт то, что реально записано")
}

// TestResidualCurveByCategoryQuery фиксирует главное отличие собственной
// кривой от действующей: обхода предков здесь НЕТ.
//
// Справочник показывает то, что заведено у самого раздела. Подмешай сюда
// ancestors — и редактор принял бы кривую предка за свою, «поправил» бы её
// и завёл копию в дочернем разделе, после чего правка в родителе перестала
// бы на него влиять.
func TestResidualCurveByCategoryQuery(t *testing.T) {
	t.Parallel()

	query := residualCurveByCategoryQuery()

	assert.Contains(t, query, "residual_value_curves")
	assert.Contains(t, query, "WHERE category_id = $1")
	assert.NotContains(t, query, "ancestors", "своя кривая, а не унаследованная")
	assert.NotContains(t, query, "RECURSIVE")
}

// TestResidualCurveListQuery проверяет запрос списка.
func TestResidualCurveListQuery(t *testing.T) {
	t.Parallel()

	query := residualCurveListQuery()

	assert.Contains(t, query, "residual_value_curves")
	assert.Contains(t, query, "ORDER BY category_id", "порядок выдачи устойчив")
}

// TestScanResidualCurve собирает кривую из строки выдачи.
//
// Порядок колонок здесь единственная возможная ошибка, и она молчаливая:
// перепутанные местами «первый год» и «годовая доля» дают правдоподобную,
// но неверную кривую на всех карточках раздела сразу.
func TestScanResidualCurve(t *testing.T) {
	t.Parallel()

	curve, err := scanResidualCurve(stubRow{values: []any{
		"44444444-4444-4444-8444-444444444444", 2000, 1200, 2000,
	}})

	require.NoError(t, err)
	assert.Equal(t, "44444444-4444-4444-8444-444444444444", curve.CategoryID)
	assert.Equal(t, 2000, curve.FirstYearDropBps)
	assert.Equal(t, 1200, curve.AnnualDropBps)
	assert.Equal(t, 2000, curve.FloorBps)

	// Колонки читаются в том же порядке, в каком перечислены в запросе:
	// иначе первая же правка списка колонок молча перепутала бы параметры.
	assert.Equal(t, "category_id, first_year_drop_bps, annual_drop_bps, floor_bps",
		residualCurveColumns)
}

// TestAttributeNumberBoundsQuery фиксирует правила подсчёта границ.
//
// Три из них молчаливые, и каждая ошибка в них даёт подсказку, которой
// нельзя доверять: диапазон по черновикам покажет значения, которых в выдаче
// нет; без обхода поддерева раздел «Экскаваторы» останется без подсказок,
// потому что сами карточки лежат в подразделах; без отсева NULL минимум
// уедет в пустоту.
func TestAttributeNumberBoundsQuery(t *testing.T) {
	t.Parallel()

	query := normalize(attributeNumberBoundsQuery())

	assert.Contains(t, query, "WITH RECURSIVE subtree", "границы считаются по поддереву")
	assert.Contains(t, query, "m.is_active AND m.published_at IS NOT NULL",
		"черновики в подсказку не попадают")
	assert.Contains(t, query, "v.num_value IS NOT NULL")
	assert.Contains(t, query, "GROUP BY v.attribute_id",
		"один запрос на весь раздел, а не по запросу на характеристику")
	assert.Contains(t, query, "min(v.num_value), max(v.num_value)")
}

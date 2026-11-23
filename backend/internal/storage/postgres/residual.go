package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// Набор колонок кривой в порядке сканирования — с префиксом таблицы и без.
const (
	residualCurveColumns        = `category_id, first_year_drop_bps, annual_drop_bps, floor_bps`
	residualCurveAliasedColumns = `curves.category_id, curves.first_year_drop_bps,
		curves.annual_drop_bps, curves.floor_bps`
)

// residualValueRepository репозиторий кривых остаточной стоимости.
type residualValueRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewResidualValueRepository создаёт новый объект residualValueRepository.
func NewResidualValueRepository(pool *pgxpool.Pool) repository.ResidualValueRepository {
	return &residualValueRepository{pool: pool}
}

// residualCurveQuery собирает запрос ближайшей заведённой кривой.
//
// Отдельной функцией, чтобы запрос можно было прочитать и проверить, не поднимая
// базу: правило «побеждает ближайший предок» выражено здесь порядком по depth,
// и сломать его молча правкой соседней строки не должно быть возможно.
func residualCurveQuery() string {
	prefix := fmt.Sprintf(categoryAncestorsCTE, categoryTable, "$1", categoryTable, "$2")

	return fmt.Sprintf(`%s
		SELECT %s
		FROM %s curves JOIN ancestors a ON a.id = curves.category_id
		ORDER BY a.depth
		LIMIT 1`, prefix, residualCurveAliasedColumns, residualValueCurveTable)
}

// CurveFor возвращает кривую, действующую для раздела.
//
// Ищет по цепочке предков от самого раздела вверх и берёт ближайшую
// заведённую. Иначе кривую пришлось бы дублировать в каждый из 44 разделов:
// «экскаваторы теряют столько-то» верно и для гусеничных, и для колёсных,
// и повторять это по веткам значило бы заводить 44 места, где одно и то же
// число расходится.
//
// Кривой нет — не ошибка: nil означает «не заведена», и расчёт стоимости
// владения обязан сказать об этом словами, а не подставить догадку.
func (r residualValueRepository) CurveFor(
	ctx context.Context,
	categoryID string,
) (*entity.ResidualValueCurve, error) {
	curve, err := scanResidualCurve(querierFromContext(ctx, r.pool).
		QueryRow(ctx, residualCurveQuery(), categoryID, maxCategoryDepth))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // «кривой нет» — не ошибка, а состояние данных
		}

		return nil, fmt.Errorf("получение кривой остаточной стоимости: %w", err)
	}

	return curve, nil
}

// RateFor возвращает долю остаточной стоимости раздела на конец N-го года.
//
// Тонкая обёртка над кривой: расчёт стоимости владения спрашивает долю
// за конкретный год и о том, что она выведена из трёх параметров, знать
// не обязан.
func (r residualValueRepository) RateFor(
	ctx context.Context,
	categoryID string,
	ageYears int,
) (*int, error) {
	curve, err := r.CurveFor(ctx, categoryID)
	if err != nil || curve == nil {
		return nil, err
	}

	return entity.Pointer(curve.RateFor(ageYears)), nil
}

// residualCurveByCategoryQuery собирает запрос собственной кривой раздела.
//
// Отдельной функцией, чтобы отсутствие обхода предков было видно и проверяемо:
// именно этим запрос отличается от residualCurveQuery, и подмешать сюда
// ancestors — значит показать редактору чужую кривую как его собственную.
func residualCurveByCategoryQuery() string {
	return fmt.Sprintf(`SELECT %s FROM %s WHERE category_id = $1`,
		residualCurveColumns, residualValueCurveTable)
}

// residualCurveListQuery собирает запрос списка кривых.
func residualCurveListQuery() string {
	return fmt.Sprintf(`SELECT %s FROM %s ORDER BY category_id`,
		residualCurveColumns, residualValueCurveTable)
}

// GetByCategory возвращает кривую, заведённую ИМЕННО для этого раздела.
//
// В отличие от CurveFor вверх по дереву не поднимается: экрану справочника
// нужно показать, заведена ли своя кривая, а не какая на разделе действует.
// Иначе редактор увидел бы унаследованную кривую как собственную и,
// «поправив» её, завёл бы копию в дочернем разделе.
func (r residualValueRepository) GetByCategory(
	ctx context.Context,
	categoryID string,
) (*entity.ResidualValueCurve, error) {
	curve, err := scanResidualCurve(querierFromContext(ctx, r.pool).
		QueryRow(ctx, residualCurveByCategoryQuery(), categoryID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrResidualCurveNotFound
		}

		return nil, fmt.Errorf("получение кривой раздела: %w", err)
	}

	return curve, nil
}

// List возвращает все заведённые кривые.
func (r residualValueRepository) List(ctx context.Context) ([]*entity.ResidualValueCurve, error) {
	rows, err := querierFromContext(ctx, r.pool).Query(ctx, residualCurveListQuery())
	if err != nil {
		return nil, fmt.Errorf("список кривых остаточной стоимости: %w", err)
	}
	defer rows.Close()

	curves := make([]*entity.ResidualValueCurve, 0)

	for rows.Next() {
		curve, scanErr := scanResidualCurve(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("чтение кривой: %w", scanErr)
		}

		curves = append(curves, curve)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("обход кривых: %w", err)
	}

	return curves, nil
}

// residualCurveSaveQuery собирает запрос сохранения кривой.
//
// Отдельной функцией по той же причине, что и поиск по предкам: правило
// «у раздела кривая одна, и сохранение либо заводит её, либо переписывает»
// выражено здесь через ON CONFLICT, и подменить его на голый INSERT — значит
// начать отказывать редактору при второй правке.
func residualCurveSaveQuery() string {
	return fmt.Sprintf(`INSERT INTO %s (category_id, first_year_drop_bps, annual_drop_bps, floor_bps)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (category_id) DO UPDATE SET
			first_year_drop_bps = EXCLUDED.first_year_drop_bps,
			annual_drop_bps = EXCLUDED.annual_drop_bps,
			floor_bps = EXCLUDED.floor_bps,
			updated_at = now()
		RETURNING %s`, residualValueCurveTable, residualCurveColumns)
}

// Save заводит кривую раздела или переписывает её.
//
// Одним запросом, а не «проверить и вставить»: у раздела кривая ровно одна,
// и различать создание с правкой снаружи незачем — редактор просто задаёт,
// как этот раздел теряет в цене.
func (r residualValueRepository) Save(
	ctx context.Context,
	curve *entity.ResidualValueCurve,
) (*entity.ResidualValueCurve, error) {
	if err := curve.Validate(); err != nil {
		return nil, err
	}

	saved, err := scanResidualCurve(querierFromContext(ctx, r.pool).QueryRow(ctx, residualCurveSaveQuery(),
		curve.CategoryID, curve.FirstYearDropBps, curve.AnnualDropBps, curve.FloorBps))
	if err != nil {
		// Раздела нет — это отказ внешнего ключа, а не поломка.
		if isForeignKeyViolation(err) {
			return nil, entity.ErrCategoryNotFound
		}

		return nil, fmt.Errorf("сохранение кривой остаточной стоимости: %w", err)
	}

	return saved, nil
}

// Delete убирает кривую раздела.
//
// После удаления раздел наследует кривую предка, а если её нет ни у кого —
// расчёт снова честно скажет, что остаточной стоимости не знает.
func (r residualValueRepository) Delete(ctx context.Context, categoryID string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE category_id = $1`, residualValueCurveTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, categoryID)
	if err != nil {
		return fmt.Errorf("удаление кривой остаточной стоимости: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrResidualCurveNotFound
	}

	return nil
}

// scanResidualCurve собирает кривую из строки выдачи.
func scanResidualCurve(row pgx.Row) (*entity.ResidualValueCurve, error) {
	var curve entity.ResidualValueCurve

	if err := row.Scan(&curve.CategoryID, &curve.FirstYearDropBps,
		&curve.AnnualDropBps, &curve.FloorBps); err != nil {
		return nil, err
	}

	return &curve, nil
}

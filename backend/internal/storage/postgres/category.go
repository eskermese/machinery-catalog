package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// categoryColumns список колонок категории в порядке выборки.
const categoryColumns = `id, parent_id, slug, name_ru, name_kk, name_en,
	position, is_active, image_path, created_at, updated_at`

// foreignKeyViolationCode код ошибки PostgreSQL при нарушении внешнего ключа.
const foreignKeyViolationCode = "23503"

// categoryRepository репозиторий категорий каталога.
type categoryRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewCategoryRepository создаёт новый объект categoryRepository.
func NewCategoryRepository(pool *pgxpool.Pool) repository.CategoryRepository {
	return &categoryRepository{pool: pool}
}

// Tree возвращает всё дерево категорий в порядке отображения.
func (r categoryRepository) Tree(ctx context.Context) (entity.Categories, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s ORDER BY parent_id NULLS FIRST, position, id`,
		categoryColumns, categoryTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("получение дерева категорий: %w", err)
	}

	categories, err := pgx.CollectRows(rows, scanCategoryRow)
	if err != nil {
		return nil, fmt.Errorf("чтение дерева категорий: %w", err)
	}

	return categories, nil
}

// CoverCandidates возвращает по одной машине на категорию — ту, что витрина
// показывает в этой категории первой и у которой есть главное фото.
//
// DISTINCT ON, а не оконная функция с фильтром: он читает по строке на группу
// и отдаёт ровно то, что нужно, — первую в заданном порядке. Порядок тот же,
// что у выдачи каталога (published_at DESC NULLS FIRST, id DESC), поэтому
// обложка раздела совпадает с первой плиткой в нём.
//
// Соединение по условию частичного уникального индекса
// machine_images_primary_uidx: у машины не больше одного главного фото,
// строк не прибавляется.
func (r categoryRepository) CoverCandidates(
	ctx context.Context,
) (map[string]entity.CategoryCover, error) {
	rows, err := querierFromContext(ctx, r.pool).Query(ctx, buildCategoryCoverQuery())
	if err != nil {
		return nil, fmt.Errorf("получение обложек категорий: %w", err)
	}

	candidates, err := pgx.CollectRows(rows, scanCategoryCoverRow)
	if err != nil {
		return nil, fmt.Errorf("чтение обложек категорий: %w", err)
	}

	covers := make(map[string]entity.CategoryCover, len(candidates))
	for _, candidate := range candidates {
		covers[candidate.categoryID] = candidate.cover
	}

	return covers, nil
}

// categoryCoverRow строка выдачи кандидатов: категория и её лучшая машина.
type categoryCoverRow struct {
	categoryID string
	cover      entity.CategoryCover
}

// scanCategoryCoverRow читает строку кандидата в обложку.
func scanCategoryCoverRow(row pgx.CollectableRow) (categoryCoverRow, error) {
	var candidate categoryCoverRow

	if err := row.Scan(&candidate.categoryID, &candidate.cover.Path,
		&candidate.cover.PublishedAt, &candidate.cover.MachineID); err != nil {
		return categoryCoverRow{}, err
	}

	return candidate, nil
}

// buildCategoryCoverQuery собирает запрос кандидатов в обложку.
//
// Вынесен из метода, чтобы порядок кандидатов проверялся тестом без базы:
// он обязан совпадать с порядком выдачи каталога, иначе обложка раздела
// и первая плитка в нём окажутся разными снимками, а заметить это можно
// было бы только глазами на живых данных.
func buildCategoryCoverQuery() string {
	return fmt.Sprintf(`SELECT DISTINCT ON (m.category_id)
			m.category_id, i.path, m.published_at, m.id
		FROM %s m
		JOIN %s i ON i.machine_id = m.id AND i.is_primary
		WHERE m.is_active AND m.published_at IS NOT NULL
		ORDER BY m.category_id, %s`,
		machineTable, machineImageTable, machineOrderExpressions[entity.SortPublishedDesc])
}

// GetBySlug возвращает категорию по слагу.
func (r categoryRepository) GetBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	return r.getBy(ctx, "slug", slug)
}

// GetByID возвращает категорию по идентификатору.
func (r categoryRepository) GetByID(ctx context.Context, id string) (*entity.Category, error) {
	return r.getBy(ctx, "id", id)
}

// getBy читает категорию по значению одной из колонок.
// Имя колонки приходит константой из кода этого файла, а не снаружи.
func (r categoryRepository) getBy(ctx context.Context, column, value string) (*entity.Category, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`, categoryColumns, categoryTable, column)

	row := querierFromContext(ctx, r.pool).QueryRow(ctx, q, value)

	category, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrCategoryNotFound
		}

		return nil, fmt.Errorf("получение категории: %w", err)
	}

	return category, nil
}

// Create сохраняет категорию.
func (r categoryRepository) Create(ctx context.Context, category *entity.Category) (*entity.Category, error) {
	if err := category.Validate(); err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`INSERT INTO %s (parent_id, slug, name_ru, name_kk, name_en, position, is_active,
		image_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING %s`,
		categoryTable, categoryColumns)

	row := querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		category.ParentID, category.Slug, category.NameRU, category.NameKK, category.NameEN,
		category.Position, category.IsActive, category.ImagePath,
		category.CreatedAt, category.UpdatedAt)

	created, err := scanCategory(row)
	if err != nil {
		return nil, categoryWriteError(err, "создание категории")
	}

	return created, nil
}

// Update обновляет редактируемые поля категории.
func (r categoryRepository) Update(ctx context.Context, category *entity.Category) error {
	if err := category.Validate(); err != nil {
		return err
	}

	q := fmt.Sprintf(`UPDATE %s SET parent_id = $1, slug = $2, name_ru = $3, name_kk = $4,
		name_en = $5, position = $6, is_active = $7, image_path = $8, updated_at = $9
		WHERE id = $10`, categoryTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q,
		category.ParentID, category.Slug, category.NameRU, category.NameKK, category.NameEN,
		category.Position, category.IsActive, category.ImagePath,
		category.UpdatedAt, category.ID)
	if err != nil {
		return categoryWriteError(err, "обновление категории")
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrCategoryNotFound
	}

	return nil
}

// Delete удаляет категорию.
func (r categoryRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, categoryTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		// На потомков и карточки стоит RESTRICT: отказ внешнего ключа здесь
		// означает «у категории есть содержимое», а не сбой.
		if isForeignKeyViolation(err) {
			return entity.ErrCategoryHasChildren
		}

		return fmt.Errorf("удаление категории: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrCategoryNotFound
	}

	return nil
}

// categoryWriteError приводит ошибки записи категории к доменным.
func categoryWriteError(err error, action string) error {
	if isUniqueViolation(err) {
		return entity.ErrSlugAlreadyUsed
	}

	// Цикл в дереве отвергает триггер categories_reject_cycle_trg,
	// поднимая ошибку с кодом нарушения CHECK.
	if isCheckViolation(err) {
		return entity.ErrCategoryCycle
	}

	return fmt.Errorf("%s: %w", action, err)
}

// scanCategory читает строку категории.
func scanCategory(row pgx.Row) (*entity.Category, error) {
	var category entity.Category

	if err := row.Scan(&category.ID, &category.ParentID, &category.Slug,
		&category.NameRU, &category.NameKK, &category.NameEN,
		&category.Position, &category.IsActive, &category.ImagePath,
		&category.CreatedAt, &category.UpdatedAt); err != nil {
		return nil, err
	}

	return &category, nil
}

// scanCategoryRow адаптирует scanCategory к pgx.CollectRows.
func scanCategoryRow(row pgx.CollectableRow) (*entity.Category, error) {
	return scanCategory(row)
}

// isCheckViolation распознаёт нарушение CHECK или отказ триггера.
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == checkViolationCode
}

// isForeignKeyViolation распознаёт нарушение внешнего ключа.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == foreignKeyViolationCode
}

// placeholders собирает список плейсхолдеров $from..$from+count-1.
// Используется пакетными вставками, где число строк известно только в рантайме.
func placeholders(from, count int) string {
	parts := make([]string, 0, count)

	for i := range count {
		parts = append(parts, fmt.Sprintf("$%d", from+i))
	}

	return strings.Join(parts, ", ")
}

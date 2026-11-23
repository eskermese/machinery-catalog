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

// attributeColumns список колонок характеристики в порядке выборки.
const attributeColumns = `id, category_id, code, name_ru, name_kk, name_en,
	data_type, unit, is_filterable, position, created_at, updated_at`

// attributeColumnsQualified тот же список с префиксом таблицы.
//
// Нужен там, где к выборке подмешан CTE ancestors: у него тоже есть колонка id,
// и неквалифицированное имя стало бы неоднозначным. Порядок обязан совпадать
// с attributeColumns — он контракт со сканером.
const attributeColumnsQualified = `attr.id, attr.category_id, attr.code,
	attr.name_ru, attr.name_kk, attr.name_en, attr.data_type, attr.unit,
	attr.is_filterable, attr.position, attr.created_at, attr.updated_at`

// attributeOptionColumns список колонок варианта в порядке выборки.
const attributeOptionColumns = `id, attribute_id, code, name_ru, name_kk, name_en, position, created_at`

// attributeRepository репозиторий характеристик категории.
type attributeRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewAttributeRepository создаёт новый объект attributeRepository.
func NewAttributeRepository(pool *pgxpool.Pool) repository.AttributeRepository {
	return &attributeRepository{pool: pool}
}

// categoryAncestorsCTE обход дерева категорий ВВЕРХ, от узла к корню.
//
// Зеркало categorySubtreeCTE из поиска: там обход вниз, здесь вверх.
// UNION и предел глубины по той же причине — цикл, попавший в данные в обход
// триггера categories_reject_cycle_trg, не должен превращать чтение панели
// фильтров в бесконечную рекурсию.
//
// depth — расстояние до узла: 0 у самой категории, 1 у её родителя и так
// далее. По нему решается, чьё объявление характеристики побеждает.
const categoryAncestorsCTE = `WITH RECURSIVE ancestors(id, depth) AS (
		SELECT id, 0 FROM %s WHERE id = %s
		UNION
		SELECT c.parent_id, a.depth + 1 FROM %s c JOIN ancestors a ON c.id = a.id
		WHERE c.parent_id IS NOT NULL AND a.depth < %s
	)`

// ListByCategory возвращает характеристики категории вместе с унаследованными
// от предков и вариантами enum.
//
// Характеристика, объявленная у родителя, действует и в дочерних категориях:
// «мощность двигателя» у экскаваторов — та же характеристика, что у гусеничных
// экскаваторов, и объявлять её заново значило бы завести второй идентификатор
// для того же понятия. Именно это ломало фильтр по поддереву: карточки дочерних
// категорий ссылались на свой идентификатор и выпадали из отбора по
// родительскому.
//
// DISTINCT ON (code) с сортировкой по depth даёт затенение: если категория
// объявила характеристику с тем же кодом, что и предок, побеждает ближайшее
// объявление. UNIQUE (category_id, code) в схеме при этом сохраняет прежний
// смысл — запрет коллизии у соседей.
func (r attributeRepository) ListByCategory(ctx context.Context, categoryID string) (entity.Attributes, error) {
	prefix := fmt.Sprintf(categoryAncestorsCTE, categoryTable, "$1", categoryTable, "$2")

	q := fmt.Sprintf(`%s
		SELECT %s FROM (
			SELECT DISTINCT ON (attr.code) %s
			FROM %s attr JOIN ancestors a ON attr.category_id = a.id
			ORDER BY attr.code, a.depth
		) inherited
		ORDER BY position, code`,
		prefix, attributeColumns, attributeColumnsQualified, attributeTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, categoryID, maxCategoryDepth)
	if err != nil {
		return nil, fmt.Errorf("получение характеристик категории: %w", err)
	}

	attributes, err := pgx.CollectRows(rows, scanAttributeRow)
	if err != nil {
		return nil, fmt.Errorf("чтение характеристик категории: %w", err)
	}

	if err := r.attachOptions(ctx, attributes); err != nil {
		return nil, err
	}

	return attributes, nil
}

// GetByID возвращает характеристику по идентификатору вместе с вариантами.
func (r attributeRepository) GetByID(ctx context.Context, id string) (*entity.Attribute, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, attributeColumns, attributeTable)

	attribute, err := scanAttribute(querierFromContext(ctx, r.pool).QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAttributeNotFound
		}

		return nil, fmt.Errorf("получение характеристики: %w", err)
	}

	if err := r.attachOptions(ctx, entity.Attributes{attribute}); err != nil {
		return nil, err
	}

	return attribute, nil
}

// attachOptions дочитывает варианты enum ко всему набору характеристик одним запросом.
//
// Один запрос на набор, а не на характеристику: панель фильтров категории
// читается на каждой странице каталога, и N+1 здесь означал бы десяток
// обращений вместо двух.
func (r attributeRepository) attachOptions(ctx context.Context, attributes entity.Attributes) error {
	enumIDs := make([]string, 0, len(attributes))
	byID := make(map[string]*entity.Attribute, len(attributes))

	for _, attribute := range attributes {
		byID[attribute.ID] = attribute

		if attribute.DataType == entity.AttributeTypeEnum {
			enumIDs = append(enumIDs, attribute.ID)
		}
	}

	if len(enumIDs) == 0 {
		return nil
	}

	q := fmt.Sprintf(`SELECT %s FROM %s WHERE attribute_id = ANY($1) ORDER BY position, code`,
		attributeOptionColumns, attributeOptionTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, enumIDs)
	if err != nil {
		return fmt.Errorf("получение вариантов характеристик: %w", err)
	}

	options, err := pgx.CollectRows(rows, scanAttributeOptionRow)
	if err != nil {
		return fmt.Errorf("чтение вариантов характеристик: %w", err)
	}

	for _, option := range options {
		if attribute, ok := byID[option.AttributeID]; ok {
			attribute.Options = append(attribute.Options, option)
		}
	}

	return nil
}

// Create сохраняет характеристику.
func (r attributeRepository) Create(ctx context.Context, attribute *entity.Attribute) (*entity.Attribute, error) {
	if err := attribute.Validate(); err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`INSERT INTO %s (category_id, code, name_ru, name_kk, name_en, data_type,
		unit, is_filterable, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING %s`,
		attributeTable, attributeColumns)

	created, err := scanAttribute(querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		attribute.CategoryID, attribute.Code, attribute.NameRU, attribute.NameKK, attribute.NameEN,
		attribute.DataType, attribute.Unit, attribute.IsFilterable, attribute.Position,
		attribute.CreatedAt, attribute.UpdatedAt))
	if err != nil {
		// Единственный уникальный ключ, на который может наткнуться вставка, —
		// attributes_category_code_unique: код уникален внутри категории.
		if isUniqueViolation(err) {
			return nil, entity.ErrAttributeCodeAlreadyUsed
		}

		return nil, fmt.Errorf("создание характеристики: %w", err)
	}

	return created, nil
}

// Update обновляет характеристику.
func (r attributeRepository) Update(ctx context.Context, attribute *entity.Attribute) error {
	if err := attribute.Validate(); err != nil {
		return err
	}

	q := fmt.Sprintf(`UPDATE %s SET code = $1, name_ru = $2, name_kk = $3, name_en = $4,
		data_type = $5, unit = $6, is_filterable = $7, position = $8, updated_at = $9
		WHERE id = $10`, attributeTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q,
		attribute.Code, attribute.NameRU, attribute.NameKK, attribute.NameEN,
		attribute.DataType, attribute.Unit, attribute.IsFilterable, attribute.Position,
		attribute.UpdatedAt, attribute.ID)
	if err != nil {
		// Составной внешний ключ (id, data_type) не даёт сменить тип
		// у характеристики, для которой уже заполнены значения. Это
		// не ошибка заполнения — это защита уже заполненного, и код у неё
		// свой.
		if isForeignKeyViolation(err) {
			return entity.ErrAttributeTypeLocked
		}

		if isUniqueViolation(err) {
			return entity.ErrAttributeCodeAlreadyUsed
		}

		return fmt.Errorf("обновление характеристики: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrAttributeNotFound
	}

	return nil
}

// Delete удаляет характеристику вместе с вариантами и значениями (каскадом схемы).
func (r attributeRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, attributeTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("удаление характеристики: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrAttributeNotFound
	}

	return nil
}

// ReplaceOptions заменяет набор вариантов характеристики целиком.
func (r attributeRepository) ReplaceOptions(
	ctx context.Context,
	attributeID string,
	options entity.AttributeOptions,
) error {
	q := querierFromContext(ctx, r.pool)

	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE attribute_id = $1`, attributeOptionTable)
	if _, err := q.Exec(ctx, deleteQuery, attributeID); err != nil {
		return fmt.Errorf("очистка вариантов характеристики: %w", err)
	}

	if len(options) == 0 {
		return nil
	}

	const columnsPerOption = 7

	args := make([]any, 0, len(options)*columnsPerOption)
	values := make([]string, 0, len(options))

	for i, option := range options {
		values = append(values, "("+placeholders(i*columnsPerOption+1, columnsPerOption)+")")
		args = append(args, attributeID, option.Code, option.NameRU, option.NameKK,
			option.NameEN, option.Position, option.CreatedAt)
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %s (attribute_id, code, name_ru, name_kk, name_en,
		position, created_at) VALUES %s`, attributeOptionTable, joinComma(values))

	if _, err := q.Exec(ctx, insertQuery, args...); err != nil {
		return fmt.Errorf("сохранение вариантов характеристики: %w", err)
	}

	return nil
}

// scanAttribute читает строку характеристики.
func scanAttribute(row pgx.Row) (*entity.Attribute, error) {
	var attribute entity.Attribute

	if err := row.Scan(&attribute.ID, &attribute.CategoryID, &attribute.Code,
		&attribute.NameRU, &attribute.NameKK, &attribute.NameEN,
		&attribute.DataType, &attribute.Unit, &attribute.IsFilterable, &attribute.Position,
		&attribute.CreatedAt, &attribute.UpdatedAt); err != nil {
		return nil, err
	}

	return &attribute, nil
}

// scanAttributeRow адаптирует scanAttribute к pgx.CollectRows.
func scanAttributeRow(row pgx.CollectableRow) (*entity.Attribute, error) {
	return scanAttribute(row)
}

// scanAttributeOptionRow читает строку варианта характеристики.
func scanAttributeOptionRow(row pgx.CollectableRow) (*entity.AttributeOption, error) {
	var option entity.AttributeOption

	if err := row.Scan(&option.ID, &option.AttributeID, &option.Code,
		&option.NameRU, &option.NameKK, &option.NameEN,
		&option.Position, &option.CreatedAt); err != nil {
		return nil, err
	}

	return &option, nil
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// machineImageColumns список колонок фотографии в порядке выборки.
const machineImageColumns = `id, machine_id, path, position, is_primary,
	alt_ru, alt_kk, alt_en, created_at`

// machineAttributeValueColumns список колонок значения характеристики.
const machineAttributeValueColumns = `id, machine_id, attribute_id, data_type,
	num_value, text_value, bool_value`

// machineRepository репозиторий карточек техники.
type machineRepository struct {
	pool *pgxpool.Pool // Пул соединений

	values repository.MachineAttributeValueRepository // Значения характеристик
	images repository.MachineImageRepository          // Фотографии
}

// NewMachineRepository создаёт новый объект machineRepository.
func NewMachineRepository(
	pool *pgxpool.Pool,
	values repository.MachineAttributeValueRepository,
	images repository.MachineImageRepository,
) repository.MachineRepository {
	return &machineRepository{pool: pool, values: values, images: images}
}

// Count возвращает число карточек, подходящих под фильтр.
//
// Отдельный запрос, а не побочный результат поиска: точное число требует
// просмотра всех подходящих строк, тогда как страница читает ровно лимит.
// Именно поэтому подсчёт делается только по явной просьбе клиента.
func (r machineRepository) Count(ctx context.Context, filter *entity.MachineFilter) (int, error) {
	query, args, err := buildMachineCountQuery(filter)
	if err != nil {
		return 0, err
	}

	var total int

	if scanErr := querierFromContext(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&total); scanErr != nil {
		return 0, fmt.Errorf("подсчёт карточек: %w", scanErr)
	}

	return total, nil
}

// Search возвращает страницу выдачи по фильтру.
func (r machineRepository) Search(ctx context.Context, filter *entity.MachineFilter) (entity.Machines, error) {
	query, args, err := buildMachineSearchQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("поиск карточек: %w", err)
	}

	machines, err := pgx.CollectRows(rows, scanMachineSearchRow)
	if err != nil {
		return nil, fmt.Errorf("чтение карточек: %w", err)
	}

	return machines, nil
}

// GetBySlug возвращает карточку по слагу вместе с характеристиками и фото.
func (r machineRepository) GetBySlug(ctx context.Context, slug string) (*entity.Machine, error) {
	return r.getBy(ctx, "m.slug", slug)
}

// GetByID возвращает карточку по идентификатору вместе с характеристиками и фото.
func (r machineRepository) GetByID(ctx context.Context, id string) (*entity.Machine, error) {
	return r.getBy(ctx, "m.id", id)
}

// getBy читает карточку по значению одной из колонок и дочитывает её состав.
// Имя колонки приходит константой из кода этого файла, а не снаружи.
func (r machineRepository) getBy(ctx context.Context, column, value string) (*entity.Machine, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s m WHERE %s = $1`, machineColumns, machineTable, column)

	machine, err := scanMachine(querierFromContext(ctx, r.pool).QueryRow(ctx, q, value))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrMachineNotFound
		}

		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	if machine.AttributeValues, err = r.values.ListByMachine(ctx, machine.ID); err != nil {
		return nil, err
	}

	if machine.Images, err = r.images.ListByMachine(ctx, machine.ID); err != nil {
		return nil, err
	}

	return machine, nil
}

// Create сохраняет карточку.
func (r machineRepository) Create(ctx context.Context, machine *entity.Machine) (*entity.Machine, error) {
	if err := machine.Validate(); err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`INSERT INTO %s (category_id, brand, model, slug, name_ru, name_kk, name_en,
		description_ru, description_kk, description_en, price_cny_cents, is_price_public,
		availability, delivery_weeks, production_year, condition,
		fuel_consumption_lph, service_interval_hours, service_cost_cny_cents,
		is_active, published_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23)
		RETURNING %s`, machineTable, strings.ReplaceAll(machineColumns, "m.", ""))

	created, err := scanMachine(querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		machine.CategoryID, machine.Brand, machine.Model, machine.Slug,
		machine.NameRU, machine.NameKK, machine.NameEN,
		machine.DescriptionRU, machine.DescriptionKK, machine.DescriptionEN,
		machine.PriceCNYCents, machine.IsPricePublic, machine.Availability, machine.DeliveryWeeks,
		machine.ProductionYear, machine.Condition,
		machine.FuelConsumptionLPH, machine.ServiceIntervalHours, machine.ServiceCostCNYCents,
		machine.IsActive, machine.PublishedAt,
		machine.CreatedAt, machine.UpdatedAt))
	if err != nil {
		// Слаг проверяется конфликтом от БД, а не предварительным SELECT:
		// между проверкой и вставкой его может занять кто-то другой.
		if isUniqueViolation(err) {
			return nil, entity.ErrSlugAlreadyUsed
		}

		return nil, fmt.Errorf("создание карточки: %w", err)
	}

	return created, nil
}

// Update обновляет редактируемые поля карточки.
//
// is_active и published_at СЮДА НЕ ВХОДЯТ, и это главное в этом запросе.
// Форма сохранения их не описывает: публикация — состояние с собственными
// переходами, а не поле карточки. Пока они писались здесь, правка опечатки
// приходила с нулевым признаком и молча снимала карточку с витрины.
// Менять их умеет только SetPublication.
func (r machineRepository) Update(ctx context.Context, machine *entity.Machine) error {
	if err := machine.Validate(); err != nil {
		return err
	}

	q := fmt.Sprintf(`UPDATE %s SET category_id = $1, brand = $2, model = $3, slug = $4,
		name_ru = $5, name_kk = $6, name_en = $7,
		description_ru = $8, description_kk = $9, description_en = $10,
		price_cny_cents = $11, is_price_public = $12, availability = $13, delivery_weeks = $14,
		production_year = $15, condition = $16,
		fuel_consumption_lph = $17, service_interval_hours = $18, service_cost_cny_cents = $19,
		updated_at = $20 WHERE id = $21`, machineTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q,
		machine.CategoryID, machine.Brand, machine.Model, machine.Slug,
		machine.NameRU, machine.NameKK, machine.NameEN,
		machine.DescriptionRU, machine.DescriptionKK, machine.DescriptionEN,
		machine.PriceCNYCents, machine.IsPricePublic, machine.Availability, machine.DeliveryWeeks,
		machine.ProductionYear, machine.Condition,
		machine.FuelConsumptionLPH, machine.ServiceIntervalHours, machine.ServiceCostCNYCents,
		machine.UpdatedAt, machine.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return entity.ErrSlugAlreadyUsed
		}

		return fmt.Errorf("обновление карточки: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrMachineNotFound
	}

	return nil
}

// SetPublication записывает состояние публикации карточки.
//
// Пишет ровно два поля и дату изменения. Отдельный запрос, а не Update
// с полной карточкой: публикация не должна зависеть от того, что лежит
// в остальных полях переданной сущности.
func (r machineRepository) SetPublication(ctx context.Context, machine *entity.Machine) error {
	if machine == nil {
		return entity.ErrNilPointer
	}

	q := fmt.Sprintf(`UPDATE %s SET is_active = $1, published_at = $2, updated_at = $3
		WHERE id = $4`, machineTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q,
		machine.IsActive, machine.PublishedAt, machine.UpdatedAt, machine.ID)
	if err != nil {
		return fmt.Errorf("смена публикации карточки: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrMachineNotFound
	}

	return nil
}

// Delete удаляет карточку вместе с характеристиками и фото (каскадом схемы).
func (r machineRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, machineTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		// Единственная ссылка на карточку с политикой RESTRICT — выпущенное
		// КП (quotes.machine_id). Значения характеристик и фотографии уходят
		// каскадом и отказать не могут, заявка обнуляет ссылку (SET NULL).
		if isForeignKeyViolation(err) {
			return entity.ErrMachineHasQuotes
		}

		return fmt.Errorf("удаление карточки: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrMachineNotFound
	}

	return nil
}

// scanMachine читает строку карточки.
func scanMachine(row pgx.Row) (*entity.Machine, error) {
	var machine entity.Machine

	if err := row.Scan(&machine.ID, &machine.CategoryID, &machine.Brand, &machine.Model, &machine.Slug,
		&machine.NameRU, &machine.NameKK, &machine.NameEN,
		&machine.DescriptionRU, &machine.DescriptionKK, &machine.DescriptionEN,
		&machine.PriceCNYCents, &machine.IsPricePublic, &machine.Availability, &machine.DeliveryWeeks,
		&machine.ProductionYear, &machine.Condition,
		&machine.FuelConsumptionLPH, &machine.ServiceIntervalHours, &machine.ServiceCostCNYCents,
		&machine.IsActive, &machine.PublishedAt,
		&machine.CreatedAt, &machine.UpdatedAt); err != nil {
		return nil, err
	}

	return &machine, nil
}

// scanMachineSearchRow читает строку выдачи поиска: карточку и её главное фото.
//
// Фото приходит соединением, а не отдельным запросом, поэтому его колонки
// читаются здесь же. Все они необязательные: карточка без фотографий даёт
// NULL во всех, и такая строка — не ошибка, а обычный случай.
func scanMachineSearchRow(row pgx.CollectableRow) (*entity.Machine, error) {
	var machine entity.Machine

	// Колонки фото читаются через указатели, включая объявленные в схеме
	// NOT NULL: необязательность здесь даёт не таблица, а левое соединение —
	// у карточки без фотографий NULL приходит во всех колонках сразу.
	var (
		imageID       *string
		imagePath     *string
		imagePosition *int
		altRU         *string
		altKK         *string
		altEN         *string
	)

	if err := row.Scan(&machine.ID, &machine.CategoryID, &machine.Brand, &machine.Model, &machine.Slug,
		&machine.NameRU, &machine.NameKK, &machine.NameEN,
		&machine.DescriptionRU, &machine.DescriptionKK, &machine.DescriptionEN,
		&machine.PriceCNYCents, &machine.IsPricePublic, &machine.Availability, &machine.DeliveryWeeks,
		&machine.ProductionYear, &machine.Condition,
		&machine.FuelConsumptionLPH, &machine.ServiceIntervalHours, &machine.ServiceCostCNYCents,
		&machine.IsActive, &machine.PublishedAt,
		&machine.CreatedAt, &machine.UpdatedAt,
		&machine.Relevance,
		&imageID, &imagePath, &imagePosition,
		&altRU, &altKK, &altEN); err != nil {
		return nil, err
	}

	// По идентификатору, а не по пути: путь у фотографии непустой всегда,
	// но отличить «фото нет» от «путь пуст» должен признак самого соединения.
	if imageID != nil {
		machine.PrimaryImage = &entity.MachineImage{
			ID:        *imageID,
			MachineID: machine.ID,
			Path:      stringOrEmpty(imagePath),
			Position:  intOrZero(imagePosition),
			// Признак не читается из базы: соединение отбирает строки
			// условием is_primary, поэтому иных здесь быть не может.
			IsPrimary: true,
			AltRU:     altRU,
			AltKK:     altKK,
			AltEN:     altEN,
		}
	}

	return &machine, nil
}

// machineAttributeValueRepository репозиторий значений характеристик машины.
type machineAttributeValueRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewMachineAttributeValueRepository создаёт новый объект machineAttributeValueRepository.
func NewMachineAttributeValueRepository(pool *pgxpool.Pool) repository.MachineAttributeValueRepository {
	return &machineAttributeValueRepository{pool: pool}
}

// ListByMachine возвращает значения характеристик машины.
func (r machineAttributeValueRepository) ListByMachine(
	ctx context.Context,
	machineID string,
) (entity.MachineAttributeValues, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE machine_id = $1`,
		machineAttributeValueColumns, machineAttributeValueTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, machineID)
	if err != nil {
		return nil, fmt.Errorf("получение значений характеристик: %w", err)
	}

	values, err := pgx.CollectRows(rows, scanMachineAttributeValueRow)
	if err != nil {
		return nil, fmt.Errorf("чтение значений характеристик: %w", err)
	}

	return values, nil
}

// Replace заменяет набор значений машины целиком.
//
// Удаление и вставка идут двумя запросами без промежуточного чтения: набор
// заменяется целиком, поэтому сверять старое с новым незачем. Вызывающий код
// обязан выполнять это внутри транзакции вместе с самой карточкой.
func (r machineAttributeValueRepository) Replace(
	ctx context.Context,
	machineID string,
	values entity.MachineAttributeValues,
) error {
	if err := values.Validate(); err != nil {
		return err
	}

	q := querierFromContext(ctx, r.pool)

	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE machine_id = $1`, machineAttributeValueTable)
	if _, err := q.Exec(ctx, deleteQuery, machineID); err != nil {
		return fmt.Errorf("очистка значений характеристик: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	const columnsPerValue = 6

	args := make([]any, 0, len(values)*columnsPerValue)
	rows := make([]string, 0, len(values))

	for i, value := range values {
		num, text, flag := splitAttributeValue(value.Value)

		rows = append(rows, "("+placeholders(i*columnsPerValue+1, columnsPerValue)+")")
		args = append(args, machineID, value.AttributeID, value.Value.DataType(), num, text, flag)
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %s (machine_id, attribute_id, data_type,
		num_value, text_value, bool_value) VALUES %s`, machineAttributeValueTable, joinComma(rows))

	if _, err := q.Exec(ctx, insertQuery, args...); err != nil {
		// Составной внешний ключ (attribute_id, data_type) отвергает значение,
		// тип которого не совпадает с объявленным у характеристики.
		if isForeignKeyViolation(err) {
			return entity.ErrAttributeValueTypeMismatch
		}

		return fmt.Errorf("сохранение значений характеристик: %w", err)
	}

	return nil
}

// splitAttributeValue раскладывает типизированное значение по колонкам таблицы.
//
// Переключатель по замкнутому набору реализаций: заполняется ровно одна
// колонка, поэтому CHECK-и machine_attribute_values_exactly_one_value
// и _type_matches_column не могут быть нарушены этим кодом в принципе.
func splitAttributeValue(value entity.AttributeValue) (num, text *string, flag *bool) {
	switch v := value.(type) {
	case entity.NumberValue:
		// Строка фиксированной точности, а не decimal.String(): точность
		// записи задаёт домен, а не то, сколько знаков оказалось у значения.
		return entity.Pointer(v.String()), nil, nil
	case entity.StringValue:
		return nil, entity.Pointer(v.Text), nil
	case entity.EnumValue:
		return nil, entity.Pointer(v.Code), nil
	case entity.BoolValue:
		return nil, nil, entity.Pointer(v.Flag)
	default:
		return nil, nil, nil
	}
}

// scanMachineAttributeValueRow читает строку значения характеристики и собирает
// типизированное значение по объявленному типу.
func scanMachineAttributeValueRow(row pgx.CollectableRow) (*entity.MachineAttributeValue, error) {
	var (
		value    entity.MachineAttributeValue
		dataType entity.AttributeDataType
		num      *string
		text     *string
		flag     *bool
	)

	if err := row.Scan(&value.ID, &value.MachineID, &value.AttributeID,
		&dataType, &num, &text, &flag); err != nil {
		return nil, err
	}

	typed, err := buildAttributeValue(dataType, num, text, flag)
	if err != nil {
		return nil, err
	}

	value.Value = typed

	return &value, nil
}

// buildAttributeValue собирает типизированное значение из колонок.
//
// Пустая колонка при объявленном типе означает рассогласование данных,
// которое CHECK machine_attribute_values_type_matches_column не пропустил бы, —
// поэтому это отказ чтения, а не значение по умолчанию.
func buildAttributeValue(
	dataType entity.AttributeDataType,
	num, text *string,
	flag *bool,
) (entity.AttributeValue, error) {
	switch dataType {
	case entity.AttributeTypeNumber:
		return buildNumberValue(num)
	case entity.AttributeTypeString:
		if text == nil {
			return nil, entity.ErrAttributeValueInvalid
		}

		return entity.NewStringValue(*text), nil
	case entity.AttributeTypeEnum:
		if text == nil {
			return nil, entity.ErrAttributeValueInvalid
		}

		return entity.NewEnumValue(*text), nil
	case entity.AttributeTypeBool:
		if flag == nil {
			return nil, entity.ErrAttributeValueInvalid
		}

		return entity.NewBoolValue(*flag), nil
	default:
		return nil, entity.ErrAttributeTypeInvalid
	}
}

// buildNumberValue разбирает числовое значение из numeric.
func buildNumberValue(num *string) (entity.AttributeValue, error) {
	if num == nil {
		return nil, entity.ErrAttributeValueInvalid
	}

	parsed, err := decimal.NewFromString(*num)
	if err != nil {
		return nil, fmt.Errorf("разбор числового значения %q: %w", *num, err)
	}

	return entity.NewNumberValue(parsed), nil
}

// machineImageRepository репозиторий фотографий машины.
type machineImageRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewMachineImageRepository создаёт новый объект machineImageRepository.
func NewMachineImageRepository(pool *pgxpool.Pool) repository.MachineImageRepository {
	return &machineImageRepository{pool: pool}
}

// ListByMachine возвращает фотографии машины в порядке показа.
func (r machineImageRepository) ListByMachine(ctx context.Context, machineID string) (entity.MachineImages, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE machine_id = $1 ORDER BY position, id`,
		machineImageColumns, machineImageTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, machineID)
	if err != nil {
		return nil, fmt.Errorf("получение фотографий: %w", err)
	}

	images, err := pgx.CollectRows(rows, scanMachineImageRow)
	if err != nil {
		return nil, fmt.Errorf("чтение фотографий: %w", err)
	}

	return images, nil
}

// UsedPaths возвращает те из переданных путей, на которые ещё ссылается
// хотя бы одна фотография.
//
// DISTINCT, а не просто выборка: один путь может стоять у нескольких
// карточек, и вызывающему коду важен сам факт ссылки, а не их число.
func (r machineImageRepository) UsedPaths(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	q := fmt.Sprintf(`SELECT DISTINCT path FROM %s WHERE path = ANY($1)`, machineImageTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, paths)
	if err != nil {
		return nil, fmt.Errorf("поиск используемых файлов: %w", err)
	}

	used, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("чтение используемых файлов: %w", err)
	}

	return used, nil
}

// Replace заменяет набор фотографий машины целиком.
func (r machineImageRepository) Replace(ctx context.Context, machineID string, images entity.MachineImages) error {
	if err := images.Validate(); err != nil {
		return err
	}

	q := querierFromContext(ctx, r.pool)

	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE machine_id = $1`, machineImageTable)
	if _, err := q.Exec(ctx, deleteQuery, machineID); err != nil {
		return fmt.Errorf("очистка фотографий: %w", err)
	}

	if len(images) == 0 {
		return nil
	}

	const columnsPerImage = 8

	args := make([]any, 0, len(images)*columnsPerImage)
	rows := make([]string, 0, len(images))

	for i, image := range images {
		rows = append(rows, "("+placeholders(i*columnsPerImage+1, columnsPerImage)+")")
		args = append(args, machineID, image.Path, image.Position, image.IsPrimary,
			image.AltRU, image.AltKK, image.AltEN, image.CreatedAt)
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %s (machine_id, path, position, is_primary,
		alt_ru, alt_kk, alt_en, created_at) VALUES %s`, machineImageTable, joinComma(rows))

	if _, err := q.Exec(ctx, insertQuery, args...); err != nil {
		if isUniqueViolation(err) {
			return entity.ErrMachineImagePrimaryConflict
		}

		return fmt.Errorf("сохранение фотографий: %w", err)
	}

	return nil
}

// SetPrimary назначает главное фото, снимая признак с остальных.
//
// Один UPDATE на все строки машины: частичный уникальный индекс
// machine_images_primary_uidx не допускает состояния с двумя главными,
// поэтому снятие и установка обязаны произойти в одном операторе.
func (r machineImageRepository) SetPrimary(ctx context.Context, machineID, imageID string) error {
	q := fmt.Sprintf(`UPDATE %s SET is_primary = (id = $1) WHERE machine_id = $2
		AND is_primary IS DISTINCT FROM (id = $1)`, machineImageTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, imageID, machineID)
	if err != nil {
		return fmt.Errorf("назначение главного фото: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrMachineNotFound
	}

	return nil
}

// scanMachineImageRow читает строку фотографии.
func scanMachineImageRow(row pgx.CollectableRow) (*entity.MachineImage, error) {
	var image entity.MachineImage

	if err := row.Scan(&image.ID, &image.MachineID, &image.Path, &image.Position, &image.IsPrimary,
		&image.AltRU, &image.AltKK, &image.AltEN, &image.CreatedAt); err != nil {
		return nil, err
	}

	return &image, nil
}

// joinComma соединяет части запятой. Части приходят только из кода этого пакета.
func joinComma(parts []string) string {
	return strings.Join(parts, ", ")
}

// stringOrEmpty разворачивает необязательную строку.
func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// intOrZero разворачивает необязательное целое.
func intOrZero(value *int) int {
	if value == nil {
		return 0
	}

	return *value
}

// attributeNumberBoundsQuery собирает запрос границ числовых характеристик.
//
// Отдельной функцией, чтобы правило читалось и проверялось без базы: границы
// считаются по ОПУБЛИКОВАННЫМ карточкам поддела и только по числовым
// значениям. Черновик в подсказку попадать не должен — его не видно в выдаче,
// и диапазон, которого никто не найдёт, хуже отсутствия подсказки.
func attributeNumberBoundsQuery() string {
	prefix := fmt.Sprintf(categorySubtreeCTE, categoryTable, "$1", categoryTable, "$2")

	return fmt.Sprintf(`%s
		SELECT v.attribute_id, min(v.num_value), max(v.num_value)
		FROM %s v JOIN %s m ON m.id = v.machine_id
		WHERE m.category_id IN (SELECT id FROM subtree)
			AND m.is_active AND m.published_at IS NOT NULL
			AND v.num_value IS NOT NULL
		GROUP BY v.attribute_id`,
		prefix, machineAttributeValueTable, machineTable)
}

// NumberBounds возвращает границы значений числовых характеристик раздела.
//
// Одним запросом на весь раздел, а не по запросу на характеристику: их у
// раздела бывает два десятка, и двадцать обращений ради подписей под полями —
// цена, которой подсказка не стоит.
func (r machineAttributeValueRepository) NumberBounds(
	ctx context.Context,
	categoryID string,
) (map[string]entity.AttributeBounds, error) {
	rows, err := querierFromContext(ctx, r.pool).
		Query(ctx, attributeNumberBoundsQuery(), categoryID, maxCategoryDepth)
	if err != nil {
		return nil, fmt.Errorf("границы характеристик: %w", err)
	}
	defer rows.Close()

	bounds := make(map[string]entity.AttributeBounds)

	for rows.Next() {
		var (
			attributeID string
			minValue    decimal.Decimal
			maxValue    decimal.Decimal
		)

		if scanErr := rows.Scan(&attributeID, &minValue, &maxValue); scanErr != nil {
			return nil, fmt.Errorf("чтение границ: %w", scanErr)
		}

		bounds[attributeID] = entity.AttributeBounds{Min: minValue, Max: maxValue}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("обход границ: %w", err)
	}

	return bounds, nil
}

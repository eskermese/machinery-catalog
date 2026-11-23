package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// leadColumns список колонок заявки в порядке выборки.
const leadColumns = `id, name, phone, city, machine_id, quote_id, source,
	status, message, comment, manager_id, created_at, updated_at`

// leadReadColumns список колонок заявки вместе с тем, что подтягивается
// соединениями: телефоном и именем ответственного, названием машины и слагами,
// из которых собирается адрес её карточки.
//
// Порядок обязан совпадать со сканером scanLead — это контракт с ним.
const leadReadColumns = `l.id, l.name, l.phone, l.city, l.machine_id, l.quote_id, l.source,
	l.status, l.message, l.comment, l.manager_id, l.created_at, l.updated_at,
	u.phone, u.name, m.name_ru, m.slug, c.slug`

// leadReadJoins соединения, дочитывающие показываемые значения.
//
// LEFT JOIN, а не INNER: заявка живёт и без назначенного менеджера, и без
// привязки к карточке — а INNER молча выбросил бы такие заявки из списка.
//
// Соединения, а не запрос на строку: список заявок открывается пачкой,
// и разрешение идентификаторов по одному дало бы N запросов на страницу.
const leadReadJoins = `LEFT JOIN users u ON u.id = l.manager_id
	LEFT JOIN machines m ON m.id = l.machine_id
	LEFT JOIN categories c ON c.id = m.category_id`

// leadRepository репозиторий заявок.
type leadRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewLeadRepository создаёт новый объект leadRepository.
func NewLeadRepository(pool *pgxpool.Pool) repository.LeadRepository {
	return &leadRepository{pool: pool}
}

// Create сохраняет заявку.
func (r leadRepository) Create(ctx context.Context, lead *entity.Lead) (*entity.Lead, error) {
	if err := lead.Validate(); err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`INSERT INTO %s (name, phone, city, machine_id, quote_id, source,
		status, message, comment, manager_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING %s`,
		leadTable, leadColumns)

	created, err := scanLead(querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		lead.Name, lead.Phone, lead.City, lead.MachineID, lead.QuoteID, lead.Source,
		lead.Status, lead.Message, lead.Comment, lead.ManagerID,
		lead.CreatedAt, lead.UpdatedAt), false)
	if err != nil {
		return nil, fmt.Errorf("сохранение заявки: %w", err)
	}

	return created, nil
}

// GetByID возвращает заявку по идентификатору.
func (r leadRepository) GetByID(ctx context.Context, id string) (*entity.Lead, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s l %s WHERE l.id = $1`,
		leadReadColumns, leadTable, leadReadJoins)

	lead, err := scanLead(querierFromContext(ctx, r.pool).QueryRow(ctx, q, id), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrLeadNotFound
		}

		return nil, fmt.Errorf("получение заявки: %w", err)
	}

	return lead, nil
}

// Count возвращает число заявок, подходящих под фильтр.
//
// Курсор не учитывается: итог считается по всей выдаче, а не по её остатку —
// на второй странице клиент ждёт того же числа, что и на первой.
func (r leadRepository) Count(ctx context.Context, filter *entity.LeadFilter) (int, error) {
	builder := newConditionBuilder()

	if filter.Status != nil {
		builder.addf("status = %s", filter.Status.String())
	}

	if filter.ManagerID != nil {
		builder.addf("manager_id = %s", *filter.ManagerID)
	}

	q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s`, leadTable, builder.where())

	var total int

	if err := querierFromContext(ctx, r.pool).QueryRow(ctx, q, builder.args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("подсчёт заявок: %w", err)
	}

	return total, nil
}

// List возвращает заявки по фильтру, свежие сверху.
//
// Условия собираются тем же построителем, что и фильтры каталога: значения
// уходят аргументами, а в текст запроса попадают только номера плейсхолдеров.
func (r leadRepository) List(ctx context.Context, filter *entity.LeadFilter) (entity.Leads, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}

	builder := newConditionBuilder()

	if filter.Status != nil {
		builder.addf("l.status = %s", filter.Status.String())
	}

	if filter.ManagerID != nil {
		builder.addf("l.manager_id = %s", *filter.ManagerID)
	}

	if filter.Cursor != nil {
		// Кортежем, а не «a < x OR (a = x AND b < y)»: форма атомарна
		// и позволяет планировщику пройти по составному индексу с нужной
		// позиции вместо разбора двух ветвей OR.
		builder.add(fmt.Sprintf("(l.created_at, l.id) < (%s, %s)",
			builder.placeholder(filter.Cursor.CreatedAt), builder.placeholder(filter.Cursor.ID)))
	}

	q := fmt.Sprintf(`SELECT %s FROM %s l %s WHERE %s ORDER BY l.created_at DESC, l.id DESC LIMIT %s`,
		leadReadColumns, leadTable, leadReadJoins, builder.where(), builder.placeholder(filter.Limit))

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, builder.args...)
	if err != nil {
		return nil, fmt.Errorf("получение заявок: %w", err)
	}

	leads, err := pgx.CollectRows(rows, scanLeadWithRefsRow)
	if err != nil {
		return nil, fmt.Errorf("чтение заявок: %w", err)
	}

	return leads, nil
}

// Update обновляет статус, ответственного и комментарий заявки.
//
// Обновляются только те поля, которые меняет менеджер: имя, телефон и источник
// заявки — то, что прислал клиент, и правке из админки не подлежат.
func (r leadRepository) Update(ctx context.Context, lead *entity.Lead) error {
	if err := lead.Validate(); err != nil {
		return err
	}

	q := fmt.Sprintf(`UPDATE %s SET status = $1, manager_id = $2, comment = $3, updated_at = $4
		WHERE id = $5`, leadTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q,
		lead.Status, lead.ManagerID, lead.Comment, lead.UpdatedAt, lead.ID)
	if err != nil {
		return fmt.Errorf("обновление заявки: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrLeadNotFound
	}

	return nil
}

// Delete удаляет заявку без следа.
//
// Строка уходит целиком, не помечается: удаление существует ради требования
// человека убрать его персональные данные, и пометка такое требование
// не исполняет — номер остался бы лежать в базе.
//
// Выпущенное по заявке предложение переживает её удаление: quotes на leads
// не ссылается вовсе, связь идёт в обратную сторону. Обязательство перед
// клиентом не должно исчезать оттого, что кто-то прибрал заявку.
func (r leadRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, leadTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("удаление заявки: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrLeadNotFound
	}

	return nil
}

// CountRecentByPhone считает заявки с номера за последний период.
func (r leadRepository) CountRecentByPhone(
	ctx context.Context,
	phone string,
	since entity.Timestamp,
) (int, error) {
	q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE phone = $1 AND created_at >= $2`, leadTable)

	var count int

	if err := querierFromContext(ctx, r.pool).QueryRow(ctx, q, phone, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("подсчёт недавних заявок: %w", err)
	}

	return count, nil
}

// scanLead читает строку заявки; withRefs сообщает, что в конце выборки
// есть дочитанные соединениями телефон ответственного, название машины
// и слаги для адреса её карточки.
//
// Один сканер на оба случая: разные наборы колонок отличаются только хвостом,
// и два почти одинаковых сканера разошлись бы при первой же новой колонке.
func scanLead(row pgx.Row, withRefs bool) (*entity.Lead, error) {
	var lead entity.Lead

	targets := []any{
		&lead.ID, &lead.Name, &lead.Phone, &lead.City,
		&lead.MachineID, &lead.QuoteID, &lead.Source,
		&lead.Status, &lead.Message, &lead.Comment, &lead.ManagerID,
		&lead.CreatedAt, &lead.UpdatedAt,
	}

	if withRefs {
		targets = append(targets,
			&lead.ManagerPhone, &lead.ManagerName,
			&lead.MachineName, &lead.MachineSlug, &lead.MachineCategorySlug)
	}

	if err := row.Scan(targets...); err != nil {
		return nil, err
	}

	return &lead, nil
}

// scanLeadWithRefsRow адаптирует scanLead к pgx.CollectRows.
func scanLeadWithRefsRow(row pgx.CollectableRow) (*entity.Lead, error) {
	return scanLead(row, true)
}

// outboxColumns список колонок очереди уведомлений в порядке выборки.
const outboxColumns = `id, event_type, entity_type, entity_id,
	recipient_role, recipient_id, payload, created_at, processed_at`

// outboxRepository репозиторий очереди уведомлений.
type outboxRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewOutboxRepository создаёт новый объект outboxRepository.
func NewOutboxRepository(pool *pgxpool.Pool) repository.OutboxRepository {
	return &outboxRepository{pool: pool}
}

// ListUnprocessed возвращает недоставленные события, старые сверху.
//
// Порядок и предикат повторяют частичный индекс
// notification_outbox_unprocessed_idx (created_at) WHERE processed_at IS NULL:
// он покрывает выборку целиком, и отправщику не приходится читать таблицу.
//
// FOR UPDATE SKIP LOCKED здесь не стоит: реплика одна, и блокировка на чтении
// только удлинила бы транзакцию. Что произойдёт при появлении второй реплики —
// разобрано в отчёте.
func (r outboxRepository) ListUnprocessed(ctx context.Context, limit int) (entity.OutboxMessages, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE processed_at IS NULL
		ORDER BY created_at LIMIT $1`, outboxColumns, notificationOutboxTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("получение недоставленных уведомлений: %w", err)
	}

	messages, err := pgx.CollectRows(rows, scanOutboxRow)
	if err != nil {
		return nil, fmt.Errorf("чтение недоставленных уведомлений: %w", err)
	}

	return messages, nil
}

// MarkProcessed помечает событие доставленным.
//
// Условие processed_at IS NULL делает отметку однократной: повторная доставка
// того же события не сдвинет момент, записанный при первой.
func (r outboxRepository) MarkProcessed(ctx context.Context, id string, processedAt entity.Timestamp) error {
	q := fmt.Sprintf(`UPDATE %s SET processed_at = $1 WHERE id = $2 AND processed_at IS NULL`,
		notificationOutboxTable)

	if _, err := querierFromContext(ctx, r.pool).Exec(ctx, q, processedAt, id); err != nil {
		return fmt.Errorf("отметка доставки уведомления: %w", err)
	}

	return nil
}

// scanOutboxRow читает строку очереди уведомлений.
func scanOutboxRow(row pgx.CollectableRow) (*entity.OutboxMessage, error) {
	var (
		message entity.OutboxMessage
		payload []byte
	)

	if err := row.Scan(&message.ID, &message.EventType, &message.EntityType, &message.EntityID,
		&message.Recipient.Role, &message.Recipient.ID, &payload,
		&message.CreatedAt, &message.ProcessedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payload, &message.Payload); err != nil {
		return nil, fmt.Errorf("разбор полей уведомления: %w", err)
	}

	return &message, nil
}

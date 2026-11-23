package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// auditColumns список колонок записи истории в порядке выборки.
const auditColumns = `id, entity_type, entity_id, action, actor_id, actor_type,
	previous_state, new_state, metadata, created_at`

// auditLogRepository репозиторий истории изменений.
type auditLogRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewAuditLogRepository создает новый объект auditLogRepository.
func NewAuditLogRepository(pool *pgxpool.Pool) repository.AuditLogRepository {
	return &auditLogRepository{pool: pool}
}

// Create сохраняет запись истории.
func (r auditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	if log == nil {
		return entity.ErrNilPointer
	}

	q := fmt.Sprintf(`INSERT INTO %s (entity_type, entity_id, action, actor_id, actor_type,
		previous_state, new_state, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, auditTable)

	var id string

	err := querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		log.EntityType, log.EntityID, log.Action, log.ActorID, log.ActorType,
		log.PreviousState, log.NewState, log.Metadata, log.CreatedAt,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("создание записи истории: %w", err)
	}

	log.ID = id

	return nil
}

// ListByEntity возвращает историю изменений сущности от новых к старым.
func (r auditLogRepository) ListByEntity(ctx context.Context, entityType entity.AuditEntityType, entityID string) (entity.AuditLogs, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC`, auditColumns, auditTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("получение истории изменений: %w", err)
	}

	logs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[entity.AuditLog])
	if err != nil {
		return nil, fmt.Errorf("чтение истории изменений: %w", err)
	}

	return logs, nil
}

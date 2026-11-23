package service

import (
	"context"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// AuditRecorder пишет историю изменений и гасит ошибки записи.
// История — вспомогательный след: ее недоступность не должна валить бизнес-операцию,
// поэтому ошибка логируется, а не возвращается наверх.
//
// Тип и его методы экспортированы, потому что собственных потребителей внутри
// пакета пока нет: доменные сервисы каталога появятся на следующем этапе.
type AuditRecorder struct {
	auditRepo repository.AuditLogRepository // Репозиторий истории изменений
}

// NewAuditRecorder создает регистратор истории изменений.
func NewAuditRecorder(auditRepo repository.AuditLogRepository) AuditRecorder {
	return AuditRecorder{auditRepo: auditRepo}
}

// Record сохраняет запись истории. Ошибка записи только логируется.
func (a AuditRecorder) Record(ctx context.Context, log *entity.AuditLog) {
	if a.auditRepo == nil || log == nil {
		return
	}

	if err := a.auditRepo.Create(ctx, log); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error":       err,
			"entity_type": log.EntityType.String(),
			"entity_id":   log.EntityID,
			"action":      log.Action.String(),
		}).Warn("запись истории изменений")
	}
}

// RecordStatusChange сохраняет смену статуса сущности.
func (a AuditRecorder) RecordStatusChange(
	ctx context.Context,
	entityType entity.AuditEntityType,
	entityID string,
	action entity.AuditAction,
	actor entity.Actor,
	previous, current map[string]any,
	currentTime time.Time,
) {
	a.Record(ctx, entity.
		NewAuditLog(entityType, entityID, action, actor, currentTime).
		WithStates(previous, current))
}

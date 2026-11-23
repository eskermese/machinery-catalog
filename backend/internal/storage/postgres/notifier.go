package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// outboxNotifier складывает доменные события в таблицу исходящих уведомлений.
//
// Отправки здесь нет: канал доставки еще не определен, поэтому событие только
// фиксируется. Когда появится адаптер, он будет читать необработанные строки по
// частичному индексу и проставлять processed_at — сервисы при этом не изменятся.
type outboxNotifier struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewNotifier создает новый объект outboxNotifier.
func NewNotifier(pool *pgxpool.Pool) repository.Notifier {
	return &outboxNotifier{pool: pool}
}

// Notify записывает событие в исходящие уведомления.
//
// Запись идет вне бизнес-транзакции: querierFromContext вернет пул, если
// транзакции в контексте нет, — а вызывающий код обязан звать Notify уже после
// коммита. Смысл тот же, что у истории изменений: уведомление не должно
// откатывать бизнес-операцию.
func (r outboxNotifier) Notify(ctx context.Context, event *entity.DomainEvent) error {
	if event == nil {
		return entity.ErrNilPointer
	}

	q := fmt.Sprintf(`INSERT INTO %s (event_type, entity_type, entity_id,
		recipient_role, recipient_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, notificationOutboxTable)

	var id string

	err := querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		event.Type, event.EntityType, event.EntityID,
		event.Recipient.Role, event.Recipient.ID, event.Payload, event.CreatedAt,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("запись уведомления: %w", err)
	}

	// Строка на каждое событие: пока отправщика нет, лог — единственный способ
	// увидеть поток уведомлений в реальном времени.
	logger.FromContext(ctx).WithFields(logger.Fields{
		"notification_id": id,
		"event_type":      event.Type.String(),
		"entity_type":     event.EntityType.String(),
		"entity_id":       event.EntityID,
		"recipient_role":  event.Recipient.Role.String(),
		"recipient_id":    event.Recipient.ID,
	}).Info("уведомление принято к доставке")

	return nil
}

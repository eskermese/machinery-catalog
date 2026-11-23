package worker

import (
	"context"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/service"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// OutboxSync отправщик уведомлений из очереди.
type OutboxSync struct {
	notifications service.NotificationService // Сервис доставки уведомлений
	interval      time.Duration               // Интервал разбора очереди
	batchSize     int                         // Сколько событий забирать за проход
}

// NewOutboxSync создаёт отправщик уведомлений.
func NewOutboxSync(
	notifications service.NotificationService,
	interval time.Duration,
	batchSize int,
) *OutboxSync {
	return &OutboxSync{notifications: notifications, interval: interval, batchSize: batchSize}
}

// Run разбирает очередь уведомлений до отмены контекста.
//
// Устроен так же, как синхронизация курса: один проход при старте, затем
// по тикеру, выход по контексту. Ошибка доставки логируется и не поднимается
// наверх — недоступный канал уведомлений не повод останавливать приложение
// вместе с HTTP, а событие останется в очереди и уйдёт на следующем проходе.
func (w *OutboxSync) Run(ctx context.Context) error {
	log := logger.FromContext(ctx).WithFields(logger.Fields{
		"interval":   w.interval.String(),
		"batch_size": w.batchSize,
	})

	log.Info("Отправка уведомлений запущена")

	w.deliverOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Отправка уведомлений остановлена")

			return nil
		case <-ticker.C:
			w.deliverOnce(ctx)
		}
	}
}

// deliverOnce выполняет один проход по очереди, гася ошибку в лог.
func (w *OutboxSync) deliverOnce(ctx context.Context) {
	delivered, err := w.notifications.DeliverPending(ctx, w.batchSize, time.Now().UTC())
	if err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
		}).Warn("разбор очереди уведомлений")

		return
	}

	if delivered > 0 {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"delivered": delivered,
		}).Info("уведомления доставлены")
	}
}

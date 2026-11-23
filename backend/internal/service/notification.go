package service

import (
	"context"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// EventNotifier рассылает доменные события и гасит ошибки доставки.
//
// Уведомление — вспомогательный эффект сценария, как и история изменений:
// его недоступность не должна откатывать бизнес-операцию, поэтому ошибка
// логируется, а не возвращается наверх. По той же причине события испускаются
// после коммита транзакции.
//
// Тип и его методы экспортированы, потому что собственных потребителей внутри
// пакета пока нет: доменные сервисы каталога появятся на следующем этапе.
type EventNotifier struct {
	notifier repository.Notifier // Порт исходящих уведомлений
}

// NewEventNotifier создает рассыльщик доменных событий.
func NewEventNotifier(notifier repository.Notifier) EventNotifier {
	return EventNotifier{notifier: notifier}
}

// Notify отправляет одно событие. Ошибка только логируется.
func (n EventNotifier) Notify(ctx context.Context, event *entity.DomainEvent) {
	if n.notifier == nil || event == nil {
		return
	}

	if err := n.notifier.Notify(ctx, event); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error":          err,
			"event_type":     event.Type.String(),
			"entity_id":      event.EntityID,
			"recipient_id":   event.Recipient.ID,
			"recipient_role": event.Recipient.Role.String(),
		}).Warn("отправка уведомления")
	}
}

// NotifyAll отправляет события пачкой. Неудача по одному получателю не отменяет
// доставку остальным: адресаты независимы.
func (n EventNotifier) NotifyAll(ctx context.Context, events entity.DomainEvents) {
	for _, event := range events {
		n.Notify(ctx, event)
	}
}

// MessageSender канал доставки уведомлений.
//
// Интерфейс объявлен здесь, где потребляется: сервису важен только факт
// доставки, а подключение конкретного канала (SMS, мессенджер, почта)
// не должно затрагивать ни очередь, ни сценарии, которые в неё пишут.
type MessageSender interface {
	// Send доставляет одно уведомление адресату.
	Send(ctx context.Context, message *entity.OutboxMessage) error
}

// NotificationService представляет интерфейс сервиса доставки уведомлений.
type NotificationService interface {
	// DeliverPending разбирает очередь и возвращает число доставленных событий.
	DeliverPending(ctx context.Context, batchSize int, currentTime time.Time) (int, error)
}

// notificationService реализация сервиса доставки уведомлений.
type notificationService struct {
	outbox repository.OutboxRepository // Очередь уведомлений
	sender MessageSender               // Канал доставки
}

// NewNotificationService создаёт сервис доставки уведомлений.
func NewNotificationService(outbox repository.OutboxRepository, sender MessageSender) NotificationService {
	return &notificationService{outbox: outbox, sender: sender}
}

// DeliverPending разбирает очередь уведомлений.
//
// processed_at проставляется ТОЛЬКО после успешной доставки: пометить раньше
// значило бы потерять событие при первом же сбое канала, а повторная доставка
// того же уведомления заметно безобиднее молчания.
//
// Неудача по одному событию не прекращает проход: адресаты независимы, и одно
// недоставленное сообщение не должно задерживать остальные — иначе очередь
// встанет на первом же событии с испорченным адресом.
func (s *notificationService) DeliverPending(
	ctx context.Context,
	batchSize int,
	currentTime time.Time,
) (int, error) {
	if s.sender == nil {
		return 0, entity.ErrNotificationSenderNotConfigured
	}

	if batchSize <= 0 {
		batchSize = defaultOutboxBatchSize
	}

	messages, err := s.outbox.ListUnprocessed(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("чтение очереди уведомлений: %w", err)
	}

	delivered := 0

	for _, message := range messages {
		if err = s.sender.Send(ctx, message); err != nil {
			logger.FromContext(ctx).WithFields(logger.Fields{
				"error":           err,
				"notification_id": message.ID,
				"event_type":      message.EventType.String(),
			}).Warn("доставка уведомления")

			continue
		}

		if err = s.outbox.MarkProcessed(ctx, message.ID, currentTime); err != nil {
			// Доставлено, но не помечено: событие уйдёт повторно на следующем
			// проходе. Это осознанный перекос в сторону дубля, а не потери.
			logger.FromContext(ctx).WithFields(logger.Fields{
				"error":           err,
				"notification_id": message.ID,
			}).Warn("отметка доставки уведомления")

			continue
		}

		delivered++
	}

	return delivered, nil
}

// defaultOutboxBatchSize размер пачки уведомлений за один проход.
//
// Полсотни: достаточно, чтобы разобрать накопившееся за интервал, и мало,
// чтобы один проход не держал соединение с каналом доставки минутами.
const defaultOutboxBatchSize = 50

// notConfiguredSender заглушка канала доставки.
//
// Подключение канала — отдельная задача, но воркер уже собирается и работает,
// поэтому отправитель нужен сейчас. Заглушка отказывает явной доменной
// ошибкой, а не делает вид, что доставила: события остаются в очереди
// с пустым processed_at, и незаконченная настройка видна и в логах, и в базе.
type notConfiguredSender struct{}

// NewNotConfiguredSender создаёт заглушку канала доставки.
func NewNotConfiguredSender() MessageSender {
	return notConfiguredSender{}
}

// Send всегда отказывает: канал доставки не подключён.
func (notConfiguredSender) Send(_ context.Context, _ *entity.OutboxMessage) error {
	return entity.ErrNotificationSenderNotConfigured
}

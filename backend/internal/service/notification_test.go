package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// testNotifyTime момент наступления событий в тестах уведомлений.
var testNotifyTime = time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

// testEvent собирает событие для тестов рассылки.
func testEvent(entityID string) *entity.DomainEvent {
	return entity.NewDomainEvent(
		"test.event", "test", entityID,
		entity.NewEventRecipient("4a2f0b0e-6f0a-4b6e-9a4e-0c1d2e3f4a5b", entity.ActorTypeManager),
		nil, testNotifyTime,
	)
}

// TestEventNotifier_NotifyFailureIsSwallowed фиксирует, что недоставленное
// уведомление не возвращается наверх: рассылка — вспомогательный эффект сценария,
// и её отказ не должен откатывать бизнес-операцию.
func TestEventNotifier_NotifyFailureIsSwallowed(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	notifier := mock_repo.NewMockNotifier(ctrl)

	notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Return(errors.New("канал недоступен"))

	// Отсутствие паники и возврата ошибки — весь наблюдаемый контракт метода.
	assert.NotPanics(t, func() {
		NewEventNotifier(notifier).Notify(context.Background(), testEvent("entity-1"))
	})
}

// TestEventNotifier_NotifyIgnoresMissingParts фиксирует, что ни пустой порт,
// ни пустое событие не приводят к обращению к адаптеру.
func TestEventNotifier_NotifyIgnoresMissingParts(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	notifier := mock_repo.NewMockNotifier(ctrl)

	notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Times(0)

	NewEventNotifier(nil).Notify(context.Background(), testEvent("entity-1"))
	NewEventNotifier(notifier).Notify(context.Background(), nil)
}

// TestEventNotifier_NotifyAllReachesEveryRecipient фиксирует, что неудача по
// одному адресату не отменяет доставку остальным: адресаты независимы.
func TestEventNotifier_NotifyAllReachesEveryRecipient(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	notifier := mock_repo.NewMockNotifier(ctrl)

	var delivered []string

	notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *entity.DomainEvent) error {
			delivered = append(delivered, event.EntityID)

			if event.EntityID == "entity-2" {
				return errors.New("канал недоступен")
			}

			return nil
		}).Times(3)

	events := entity.DomainEvents{testEvent("entity-1"), testEvent("entity-2"), testEvent("entity-3")}

	NewEventNotifier(notifier).NotifyAll(context.Background(), events)

	require.Len(t, delivered, 3)
	assert.Equal(t, []string{"entity-1", "entity-2", "entity-3"}, delivered)
}

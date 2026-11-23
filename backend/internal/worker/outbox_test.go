package worker_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	"github.com/eskermese/machinery-catalog/backend/internal/worker"
)

// stubNotificationService считает проходы по очереди.
type stubNotificationService struct {
	service.NotificationService // Остальные методы воркеру не нужны

	calls     atomic.Int64
	delivered int
	err       error
	done      chan struct{}
}

// newStubNotificationService создаёт заглушку сервиса доставки.
func newStubNotificationService(delivered int, err error) *stubNotificationService {
	return &stubNotificationService{delivered: delivered, err: err, done: make(chan struct{}, 1)}
}

// DeliverPending считает вызов и сигналит о нём.
func (s *stubNotificationService) DeliverPending(context.Context, int, time.Time) (int, error) {
	s.calls.Add(1)

	select {
	case s.done <- struct{}{}:
	default:
	}

	return s.delivered, s.err
}

// TestOutboxSync_RunsOnceBeforeFirstTick фиксирует проход при старте:
// накопившиеся за простой уведомления не должны ждать целый интервал.
func TestOutboxSync_RunsOnceBeforeFirstTick(t *testing.T) {
	t.Parallel()

	stub := newStubNotificationService(2, nil)
	w := worker.NewOutboxSync(stub, time.Hour, 50)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	select {
	case <-stub.done:
	case <-time.After(2 * time.Second):
		t.Fatal("стартовый проход не выполнен")
	}

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("воркер не завершился по отмене контекста")
	}

	assert.Equal(t, int64(1), stub.calls.Load(), "до первого тика ровно один проход")
}

// TestOutboxSync_StopsOnContextCancel фиксирует корректное завершение.
func TestOutboxSync_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	stub := newStubNotificationService(0, nil)
	w := worker.NewOutboxSync(stub, 10*time.Millisecond, 50)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	select {
	case <-stub.done:
	case <-time.After(2 * time.Second):
		t.Fatal("разбор очереди не запускался")
	}

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err, "остановка по контексту — не ошибка")
	case <-time.After(2 * time.Second):
		t.Fatal("воркер не завершился по отмене контекста")
	}
}

// TestOutboxSync_ErrorDoesNotStopWorker фиксирует, что отказ доставки
// не роняет приложение: недоступный канал уведомлений не повод останавливать
// сервис вместе с HTTP.
func TestOutboxSync_ErrorDoesNotStopWorker(t *testing.T) {
	t.Parallel()

	stub := newStubNotificationService(0, entity.ErrNotificationSenderNotConfigured)
	w := worker.NewOutboxSync(stub, 10*time.Millisecond, 50)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	require.Eventually(t, func() bool {
		return stub.calls.Load() >= 2
	}, 2*time.Second, 5*time.Millisecond, "воркер остановился после первой ошибки")

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err, "ошибка доставки не должна подниматься наверх")
	case <-time.After(2 * time.Second):
		t.Fatal("воркер не завершился по отмене контекста")
	}
}

// TestOutboxSync_RepeatsOnTicker фиксирует, что повтор обеспечивает тикер:
// событие, не ушедшее сейчас, уйдёт на следующем проходе.
func TestOutboxSync_RepeatsOnTicker(t *testing.T) {
	t.Parallel()

	stub := newStubNotificationService(0, errors.New("канал недоступен"))
	w := worker.NewOutboxSync(stub, 5*time.Millisecond, 50)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	require.Eventually(t, func() bool {
		return stub.calls.Load() >= 3
	}, 2*time.Second, 5*time.Millisecond)

	cancel()
	<-errCh
}

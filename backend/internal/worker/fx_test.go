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

// stubFXService считает вызовы синхронизации и отдаёт заданную ошибку.
// Рукописный, а не сгенерированный: gomock здесь неудобен — воркер вызывает
// сервис из своей горутины в неизвестный момент, и ожидания по числу вызовов
// пришлось бы синхронизировать вручную теми же средствами.
type stubFXService struct {
	service.FXService // Остальные методы воркеру не нужны и не вызываются

	calls atomic.Int64 // Число прогонов синхронизации
	err   error        // Ошибка, которую вернуть
	done  chan struct{}
}

// newStubFXService создаёт заглушку сервиса курсов.
func newStubFXService(err error) *stubFXService {
	return &stubFXService{err: err, done: make(chan struct{}, 1)}
}

// SyncRate считает вызов и сигналит о нём.
func (s *stubFXService) SyncRate(_ context.Context, _, _ string, _ time.Time) error {
	s.calls.Add(1)

	select {
	case s.done <- struct{}{}:
	default:
	}

	return s.err
}

// TestFXSync_RunsOnceBeforeFirstTick фиксирует прогон при старте: иначе после
// перезапуска сервис ждал бы целый интервал, ничего не зная о курсе.
func TestFXSync_RunsOnceBeforeFirstTick(t *testing.T) {
	t.Parallel()

	stub := newStubFXService(nil)

	// Интервал заведомо больше времени теста: сработать успеет только
	// стартовый прогон, а не тик.
	w := worker.NewFXSync(stub, time.Hour, "USD", "KZT")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	select {
	case <-stub.done:
	case <-time.After(2 * time.Second):
		t.Fatal("стартовый прогон не выполнен")
	}

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("воркер не завершился по отмене контекста")
	}

	assert.Equal(t, int64(1), stub.calls.Load(), "до первого тика ровно один прогон")
}

// TestFXSync_StopsOnContextCancel фиксирует корректное завершение: воркер
// обязан выйти по контексту, а не держать приложение на shutdown.
func TestFXSync_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	stub := newStubFXService(nil)
	w := worker.NewFXSync(stub, 10*time.Millisecond, "USD", "KZT")

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	// Даём тикеру сработать хотя бы раз, чтобы выход проверялся из ожидания тика.
	select {
	case <-stub.done:
	case <-time.After(2 * time.Second):
		t.Fatal("синхронизация не запускалась")
	}

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err, "остановка по контексту — не ошибка")
	case <-time.After(2 * time.Second):
		t.Fatal("воркер не завершился по отмене контекста")
	}
}

// TestFXSync_ErrorDoesNotStopWorker фиксирует, что отказ синхронизации не
// роняет воркер и не возвращается наверх: курс не критичен для работы сервиса,
// а errgroup остановил бы вместе с ним и HTTP.
func TestFXSync_ErrorDoesNotStopWorker(t *testing.T) {
	t.Parallel()

	stub := newStubFXService(entity.ErrFXProviderNotConfigured)
	w := worker.NewFXSync(stub, 10*time.Millisecond, "USD", "KZT")

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	// Ждём несколько прогонов: после первой ошибки воркер обязан продолжить.
	require.Eventually(t, func() bool {
		return stub.calls.Load() >= 2
	}, 2*time.Second, 5*time.Millisecond, "воркер остановился после первой ошибки")

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err, "ошибка синхронизации не должна подниматься наверх")
	case <-time.After(2 * time.Second):
		t.Fatal("воркер не завершился по отмене контекста")
	}
}

// TestFXSync_RepeatsOnTicker фиксирует, что повтор обеспечивает сам тикер:
// отдельного механизма ретраев внутри прогона нет намеренно.
func TestFXSync_RepeatsOnTicker(t *testing.T) {
	t.Parallel()

	stub := newStubFXService(errors.New("источник недоступен"))
	w := worker.NewFXSync(stub, 5*time.Millisecond, "USD", "KZT")

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- w.Run(ctx) }()

	require.Eventually(t, func() bool {
		return stub.calls.Load() >= 3
	}, 2*time.Second, 5*time.Millisecond)

	cancel()
	<-errCh
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// txKey тип ключа, под которым транзакция кладется в контекст.
type txKey struct{}

// querier общий для пула и транзакции набор методов. Позволяет репозиториям
// не знать, выполняются ли они внутри транзакции.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// querierFromContext возвращает транзакцию из контекста, а при её отсутствии — пул соединений.
func querierFromContext(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return pool
}

// txManager менеджер транзакций поверх пула соединений.
type txManager struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewTxManager создает менеджер транзакций.
func NewTxManager(pool *pgxpool.Pool) repository.TxManager {
	return &txManager{pool: pool}
}

// RunTransaction выполняет fn внутри транзакции. Транзакция кладется в контекст,
// поэтому вызванные внутри fn репозитории работают в ней автоматически.
func (m *txManager) RunTransaction(ctx context.Context, fn func(context.Context) error, _ bool) (err error) {
	// Вложенную транзакцию не начинаем — переиспользуем текущую.
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}

	// При ошибке откатываем транзакцию, сохраняя исходную ошибку.
	defer func() {
		if err == nil {
			return
		}

		if rErr := tx.Rollback(ctx); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("откат транзакции: %w", rErr))
		}
	}()

	err = fn(context.WithValue(ctx, txKey{}, tx))
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("коммит транзакции: %w", err)
	}

	return nil
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// appSettingsTable имя таблицы настроек.
const appSettingsTable = "app_settings"

// appSettingsRepository доступ к настройкам приложения.
type appSettingsRepository struct {
	pool *pgxpool.Pool
}

// NewAppSettingsRepository создаёт репозиторий настроек.
func NewAppSettingsRepository(pool *pgxpool.Pool) repository.AppSettingsRepository {
	return appSettingsRepository{pool: pool}
}

// Get возвращает настройку по ключу.
func (r appSettingsRepository) Get(ctx context.Context, key string) (*entity.AppSetting, error) {
	q := fmt.Sprintf(`SELECT key, value FROM %s WHERE key = $1`, appSettingsTable)

	var setting entity.AppSetting

	err := querierFromContext(ctx, r.pool).QueryRow(ctx, q, key).Scan(&setting.Key, &setting.Value)
	if err != nil {
		// Незаданная настройка — не ошибка: значение по умолчанию
		// выбирает читающий код, и отличать «не задана» от «задана пустой»
		// ему не требуется.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("получение настройки: %w", err)
	}

	return &setting, nil
}

// Set сохраняет настройку, перезаписывая прежнее значение.
func (r appSettingsRepository) Set(ctx context.Context, setting *entity.AppSetting) error {
	q := fmt.Sprintf(`INSERT INTO %s (key, value, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, appSettingsTable)

	if _, err := querierFromContext(ctx, r.pool).Exec(ctx, q, setting.Key, setting.Value); err != nil {
		return fmt.Errorf("сохранение настройки: %w", err)
	}

	return nil
}

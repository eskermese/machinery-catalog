package repository

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// AppSettingsRepository исходящий порт настроек приложения.
//
// В отличие от курсов, настройка перезаписывается: она описывает состояние
// «как сейчас», а не событие, которое уже показали клиенту. История здесь
// не нужна — предыдущее значение флага ничего не объясняет о выданных ценах,
// потому что цену объясняет курс и наценка, а они свою историю хранят.
type AppSettingsRepository interface {
	// Get возвращает настройку по ключу. Ненайденная — nil без ошибки:
	// незаданная настройка это штатное состояние, и значение по умолчанию
	// выбирает читающий код.
	Get(ctx context.Context, key string) (*entity.AppSetting, error)
	// Set сохраняет настройку, перезаписывая прежнее значение.
	Set(ctx context.Context, setting *entity.AppSetting) error
}

package repository

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

type DataStore interface {
	// Base базовый интерфейс для работы с DataStore
	Base

	// TxManager возвращает интерфейс для начала транзакции.
	TxManager() TxManager
	// AuditLogRepository возвращает репозиторий истории изменений.
	AuditLogRepository() AuditLogRepository
	// Notifier возвращает порт исходящих уведомлений.
	Notifier() Notifier
	// UserRepository возвращает репозиторий учётных записей.
	UserRepository() UserRepository
	// FXRateRepository возвращает репозиторий курсов валют.
	FXRateRepository() FXRateRepository
	// FXMarkupRepository возвращает репозиторий наценки к курсу.
	FXMarkupRepository() FXMarkupRepository
	// AppSettingsRepository возвращает репозиторий настроек приложения.
	AppSettingsRepository() AppSettingsRepository
	// CategoryRepository возвращает репозиторий категорий каталога.
	CategoryRepository() CategoryRepository
	// AttributeRepository возвращает репозиторий характеристик.
	AttributeRepository() AttributeRepository
	// MachineRepository возвращает репозиторий карточек техники.
	MachineRepository() MachineRepository
	// MachineAttributeValueRepository возвращает репозиторий значений характеристик.
	MachineAttributeValueRepository() MachineAttributeValueRepository
	// MachineImageRepository возвращает репозиторий фотографий.
	MachineImageRepository() MachineImageRepository
	// ResidualValueRepository ставки остаточной стоимости по разделам.
	ResidualValueRepository() ResidualValueRepository
	// QuoteRepository возвращает репозиторий коммерческих предложений.
	QuoteRepository() QuoteRepository
	// LeasingProgramRepository возвращает репозиторий программ лизинга.
	LeasingProgramRepository() LeasingProgramRepository
	// LeadRepository возвращает репозиторий заявок.
	LeadRepository() LeadRepository
	// OutboxRepository возвращает репозиторий очереди уведомлений.
	OutboxRepository() OutboxRepository
}

// Base представляет базовый интерфейс для работы с DataStore.
type Base interface {
	// Name возвращает название DataStore
	Name() string

	// Close закрывает соединение с DataStore
	Close(ctx context.Context) error

	// Connect устанавливает соединение с DataStore
	Connect() error
}

// AuditLogRepository представляет интерфейс для работы с историей изменений.
type AuditLogRepository interface {
	// Create сохраняет запись истории.
	Create(ctx context.Context, log *entity.AuditLog) error
	// ListByEntity возвращает историю изменений сущности.
	ListByEntity(ctx context.Context, entityType entity.AuditEntityType, entityID string) (entity.AuditLogs, error)
}

// TxManager определяет интерфейс для запуска транзакций.
type TxManager interface {
	// RunTransaction выполняет функцию в транзакции.
	RunTransaction(ctx context.Context, fn func(context.Context) error, retryable bool) (err error)
}

// TxStub — тестовая реализация TxManager.
// Просто выполняет функцию fn без какой-либо транзакции.
type TxStub struct{}

// RunTransaction просто вызывает fn(ctx) без начала транзакции.
func (TxStub) RunTransaction(ctx context.Context, fn func(context.Context) error, _ bool) error {
	return fn(ctx)
}

// NewTxStub возвращает новый экземпляр TxStub.
func NewTxStub() *TxStub {
	return &TxStub{}
}

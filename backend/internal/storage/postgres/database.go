package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/config"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

const (
	// auditTable название таблицы истории изменений.
	auditTable = "audit_logs"
	// notificationOutboxTable название таблицы исходящих уведомлений.
	notificationOutboxTable = "notification_outbox"
	// userTable название таблицы учётных записей.
	userTable = "users"
	// fxRateTable название таблицы курсов валют.
	fxRateTable = "fx_rates"
	// fxMarkupTable название таблицы наценки к курсу.
	fxMarkupTable = "fx_markups"
	// categoryTable название таблицы категорий каталога.
	categoryTable = "categories"
	// attributeTable название таблицы характеристик.
	attributeTable = "attributes"
	// attributeOptionTable название таблицы вариантов характеристик.
	attributeOptionTable = "attribute_options"
	// machineTable название таблицы карточек техники.
	machineTable = "machines"
	// machineAttributeValueTable название таблицы значений характеристик.
	machineAttributeValueTable = "machine_attribute_values"
	// machineImageTable название таблицы фотографий.
	machineImageTable = "machine_images"
	// quoteTable название таблицы коммерческих предложений.
	quoteTable = "quotes"
	// leasingProgramTable название таблицы программ лизинга.
	leasingProgramTable = "leasing_programs"
	// leadTable название таблицы заявок.
	leadTable = "leads"
	// residualValueCurveTable название таблицы кривых остаточной стоимости.
	residualValueCurveTable = "residual_value_curves"
)

const (
	// uniqueViolationCode код ошибки PostgreSQL при нарушении уникальности.
	uniqueViolationCode = "23505"
	// checkViolationCode код ошибки PostgreSQL при нарушении CHECK или отказе триггера.
	checkViolationCode = "23514"
)

// Postgres представляет собой хранилище данных PostgreSQL.
type Postgres struct {
	conf *config.Database // Конфигурация подключения

	pool *pgxpool.Pool // Пул соединений. Создается при подключении к БД

	txManager    repository.TxManager             // Менеджер транзакций
	auditRepo    repository.AuditLogRepository    // Репозиторий истории изменений
	notifier     repository.Notifier              // Исходящие уведомления
	userRepo     repository.UserRepository        // Репозиторий учётных записей
	fxRateRepo   repository.FXRateRepository      // Репозиторий курсов валют
	fxMarkupRepo repository.FXMarkupRepository    // Репозиторий наценки к курсу
	settingsRepo repository.AppSettingsRepository // Репозиторий настроек приложения

	categoryRepo  repository.CategoryRepository              // Репозиторий категорий
	attributeRepo repository.AttributeRepository             // Репозиторий характеристик
	machineRepo   repository.MachineRepository               // Репозиторий карточек техники
	valueRepo     repository.MachineAttributeValueRepository // Репозиторий значений характеристик
	imageRepo     repository.MachineImageRepository          // Репозиторий фотографий
	residualRepo  repository.ResidualValueRepository         // Репозиторий остаточной стоимости

	quoteRepo   repository.QuoteRepository          // Репозиторий коммерческих предложений
	programRepo repository.LeasingProgramRepository // Репозиторий программ лизинга
	leadRepo    repository.LeadRepository           // Репозиторий заявок
	outboxRepo  repository.OutboxRepository         // Репозиторий очереди уведомлений
}

// New создает новый экземпляр Postgres.
func New(conf *config.Database) repository.DataStore {
	return &Postgres{conf: conf}
}

// Name возвращает название БД.
func (p *Postgres) Name() string { return "postgres" }

// Connect подключается к БД и проверяет соединение.
func (p *Postgres) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.conf.ConnectTimout)
	defer cancel()

	poolConf, err := pgxpool.ParseConfig(p.conf.DSURL)
	if err != nil {
		return fmt.Errorf("разбор DSN: %w", err)
	}

	poolConf.MaxConns = p.conf.MaxConns
	poolConf.MinConns = p.conf.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConf)
	if err != nil {
		return fmt.Errorf("создание пула соединений: %w", err)
	}

	p.pool = pool

	if err = p.pool.Ping(ctx); err != nil {
		p.pool.Close()

		return fmt.Errorf("пинг: %w", err)
	}

	return nil
}

// Close закрывает пул соединений.
func (p *Postgres) Close(_ context.Context) error {
	if p.pool == nil {
		return nil
	}

	p.pool.Close()

	return nil
}

// TxManager возвращает менеджер транзакций.
func (p *Postgres) TxManager() repository.TxManager {
	if p.txManager == nil {
		p.txManager = NewTxManager(p.pool)
	}

	return p.txManager
}

// AuditLogRepository репозиторий истории изменений.
func (p *Postgres) AuditLogRepository() repository.AuditLogRepository {
	if p.auditRepo == nil {
		p.auditRepo = NewAuditLogRepository(p.pool)
	}

	return p.auditRepo
}

// Notifier возвращает порт исходящих уведомлений.
func (p *Postgres) Notifier() repository.Notifier {
	if p.notifier == nil {
		p.notifier = NewNotifier(p.pool)
	}

	return p.notifier
}

// UserRepository репозиторий учётных записей.
func (p *Postgres) UserRepository() repository.UserRepository {
	if p.userRepo == nil {
		p.userRepo = NewUserRepository(p.pool)
	}

	return p.userRepo
}

// FXRateRepository репозиторий курсов валют.
func (p *Postgres) FXRateRepository() repository.FXRateRepository {
	if p.fxRateRepo == nil {
		p.fxRateRepo = NewFXRateRepository(p.pool)
	}

	return p.fxRateRepo
}

// FXMarkupRepository репозиторий наценки к курсу.
func (p *Postgres) FXMarkupRepository() repository.FXMarkupRepository {
	if p.fxMarkupRepo == nil {
		p.fxMarkupRepo = NewFXMarkupRepository(p.pool)
	}

	return p.fxMarkupRepo
}

// AppSettingsRepository возвращает репозиторий настроек приложения.
func (p *Postgres) AppSettingsRepository() repository.AppSettingsRepository {
	if p.settingsRepo == nil {
		p.settingsRepo = NewAppSettingsRepository(p.pool)
	}

	return p.settingsRepo
}

// CategoryRepository репозиторий категорий каталога.
func (p *Postgres) CategoryRepository() repository.CategoryRepository {
	if p.categoryRepo == nil {
		p.categoryRepo = NewCategoryRepository(p.pool)
	}

	return p.categoryRepo
}

// AttributeRepository репозиторий характеристик.
func (p *Postgres) AttributeRepository() repository.AttributeRepository {
	if p.attributeRepo == nil {
		p.attributeRepo = NewAttributeRepository(p.pool)
	}

	return p.attributeRepo
}

// MachineAttributeValueRepository репозиторий значений характеристик.
func (p *Postgres) MachineAttributeValueRepository() repository.MachineAttributeValueRepository {
	if p.valueRepo == nil {
		p.valueRepo = NewMachineAttributeValueRepository(p.pool)
	}

	return p.valueRepo
}

// MachineImageRepository репозиторий фотографий.
func (p *Postgres) MachineImageRepository() repository.MachineImageRepository {
	if p.imageRepo == nil {
		p.imageRepo = NewMachineImageRepository(p.pool)
	}

	return p.imageRepo
}

// MachineRepository репозиторий карточек техники.
func (p *Postgres) MachineRepository() repository.MachineRepository {
	if p.machineRepo == nil {
		p.machineRepo = NewMachineRepository(p.pool,
			p.MachineAttributeValueRepository(), p.MachineImageRepository())
	}

	return p.machineRepo
}

// ResidualValueRepository репозиторий ставок остаточной стоимости.
func (p *Postgres) ResidualValueRepository() repository.ResidualValueRepository {
	if p.residualRepo == nil {
		p.residualRepo = NewResidualValueRepository(p.pool)
	}

	return p.residualRepo
}

// QuoteRepository репозиторий коммерческих предложений.
func (p *Postgres) QuoteRepository() repository.QuoteRepository {
	if p.quoteRepo == nil {
		p.quoteRepo = NewQuoteRepository(p.pool, p.MachineRepository())
	}

	return p.quoteRepo
}

// LeasingProgramRepository репозиторий программ лизинга.
func (p *Postgres) LeasingProgramRepository() repository.LeasingProgramRepository {
	if p.programRepo == nil {
		p.programRepo = NewLeasingProgramRepository(p.pool)
	}

	return p.programRepo
}

// LeadRepository репозиторий заявок.
func (p *Postgres) LeadRepository() repository.LeadRepository {
	if p.leadRepo == nil {
		p.leadRepo = NewLeadRepository(p.pool)
	}

	return p.leadRepo
}

// OutboxRepository репозиторий очереди уведомлений.
func (p *Postgres) OutboxRepository() repository.OutboxRepository {
	if p.outboxRepo == nil {
		p.outboxRepo = NewOutboxRepository(p.pool)
	}

	return p.outboxRepo
}

// queryOneRow читает ровно одну строку в структуру T, приводя отсутствие строки
// к переданной доменной ошибке. subject подставляется в текст ошибки.
func queryOneRow[T any](ctx context.Context, q querier, query, subject string, notFound error, args ...any) (*T, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("получение %s: %w", subject, err)
	}

	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[T])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound
		}

		return nil, fmt.Errorf("чтение %s: %w", subject, err)
	}

	return item, nil
}

// isUniqueViolation распознает нарушение уникального индекса,
// чтобы адаптер мог вернуть доменную ошибку вместо сырой ошибки драйвера.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == uniqueViolationCode
}

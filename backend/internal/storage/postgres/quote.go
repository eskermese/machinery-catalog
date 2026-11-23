package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// quoteColumns список колонок коммерческого предложения в порядке выборки.
const quoteColumns = `id, public_token, machine_id, configuration,
	price_cny_cents, fx_rate, fx_rate_date, fx_markup_bps, asset_cost_kzt_cents,
	leasing_program_id, schedule_type, down_payment_bps, term_months, rate_bps,
	residual_value_kzt_cents, commission_kzt_cents,
	financed_amount_kzt_cents, monthly_payment_kzt_cents, last_payment_kzt_cents,
	total_paid_kzt_cents, overpayment_kzt_cents, payment_schedule,
	author_id, expires_at, first_opened_at, created_at`

// quoteReadColumns те же колонки с алиасом плюс название программы,
// дочитанное соединением. Порядок обязан совпадать с scanQuoteWithProgram.
const quoteReadColumns = `q.id, q.public_token, q.machine_id, q.configuration,
	q.price_cny_cents, q.fx_rate, q.fx_rate_date, q.fx_markup_bps, q.asset_cost_kzt_cents,
	q.leasing_program_id, q.schedule_type, q.down_payment_bps, q.term_months, q.rate_bps,
	q.residual_value_kzt_cents, q.commission_kzt_cents,
	q.financed_amount_kzt_cents, q.monthly_payment_kzt_cents, q.last_payment_kzt_cents,
	q.total_paid_kzt_cents, q.overpayment_kzt_cents, q.payment_schedule,
	q.author_id, q.expires_at, q.first_opened_at, q.created_at,
	p.name_ru`

// quoteProgramJoin дочитывает название программы лизинга.
//
// LEFT JOIN: предложение считается и без программы, по переданным вручную
// параметрам, и INNER выбросил бы такие предложения из выдачи.
const quoteProgramJoin = `LEFT JOIN leasing_programs p ON p.id = q.leasing_program_id`

// quoteListColumns колонки списка: те же, что при чтении одного предложения,
// плюс название машины и слаги, из которых собирается адрес её карточки.
// Порядок обязан совпадать со сканером.
const quoteListColumns = quoteReadColumns + `, m.name_ru, m.slug, c.slug`

// quoteListJoins соединения списка.
//
// Машина дочитывается соединением, а не запросом на строку: список открывается
// пачкой, и разрешение идентификаторов по одному дало бы запрос на каждую
// строку. Полной карточки здесь и не нужно — в списке стоит название и ссылка.
//
// LEFT JOIN: снятая с публикации или удалённая карточка не должна выбрасывать
// выпущенное по ней предложение из списка менеджера.
const quoteListJoins = quoteProgramJoin + `
	LEFT JOIN machines m ON m.id = q.machine_id
	LEFT JOIN categories c ON c.id = m.category_id`

// quoteRepository репозиторий коммерческих предложений.
type quoteRepository struct {
	pool *pgxpool.Pool // Пул соединений

	machines repository.MachineRepository // Репозиторий карточек
}

// NewQuoteRepository создаёт новый объект quoteRepository.
func NewQuoteRepository(pool *pgxpool.Pool, machines repository.MachineRepository) repository.QuoteRepository {
	return &quoteRepository{pool: pool, machines: machines}
}

// Create сохраняет коммерческое предложение.
func (r quoteRepository) Create(ctx context.Context, quote *entity.Quote) (*entity.Quote, error) {
	if err := quote.Validate(); err != nil {
		return nil, err
	}

	schedule, err := json.Marshal(quote.PaymentSchedule)
	if err != nil {
		return nil, fmt.Errorf("сериализация графика платежей: %w", err)
	}

	configuration, err := json.Marshal(configurationOrEmpty(quote.Configuration))
	if err != nil {
		return nil, fmt.Errorf("сериализация конфигурации: %w", err)
	}

	q := fmt.Sprintf(`INSERT INTO %s (public_token, machine_id, configuration,
		price_cny_cents, fx_rate, fx_rate_date, fx_markup_bps, asset_cost_kzt_cents,
		leasing_program_id, schedule_type, down_payment_bps, term_months, rate_bps,
		residual_value_kzt_cents, commission_kzt_cents,
		financed_amount_kzt_cents, monthly_payment_kzt_cents, last_payment_kzt_cents,
		total_paid_kzt_cents, overpayment_kzt_cents, payment_schedule,
		author_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING %s`, quoteTable, quoteColumns)

	created, err := scanQuote(querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		quote.PublicToken, quote.MachineID, configuration,
		quote.PriceCNYCents, quote.FXRate.StringFixed(entity.FXRateScale),
		quote.FXRateDate, quote.FXMarkupBps, quote.AssetCostKZTCents,
		quote.LeasingProgramID, quote.ScheduleType, quote.DownPaymentBps,
		quote.TermMonths, quote.RateBps,
		quote.ResidualValueKZTCents, quote.CommissionKZTCents,
		quote.FinancedAmountKZTCents, quote.MonthlyPaymentKZTCents, quote.LastPaymentKZTCents,
		quote.TotalPaidKZTCents, quote.OverpaymentKZTCents, schedule,
		quote.AuthorID, quote.ExpiresAt, quote.CreatedAt))
	if err != nil {
		// Токен генерируется криптостойким источником, поэтому конфликт здесь
		// означает не занятое имя, а сбой источника случайности.
		if isUniqueViolation(err) {
			return nil, entity.ErrQuoteTokenGeneration
		}

		return nil, fmt.Errorf("сохранение предложения: %w", err)
	}

	return created, nil
}

// GetByToken возвращает предложение по публичному токену вместе с машиной.
func (r quoteRepository) GetByToken(ctx context.Context, token string) (*entity.Quote, error) {
	return r.getBy(ctx, "public_token", token)
}

// GetByID возвращает предложение по идентификатору вместе с машиной.
func (r quoteRepository) GetByID(ctx context.Context, id string) (*entity.Quote, error) {
	return r.getBy(ctx, "id", id)
}

// getBy читает предложение по значению одной из колонок и дочитывает машину.
// Имя колонки приходит константой из кода этого файла, а не снаружи.
func (r quoteRepository) getBy(ctx context.Context, column, value string) (*entity.Quote, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s q %s WHERE q.%s = $1`,
		quoteReadColumns, quoteTable, quoteProgramJoin, column)

	quote, err := scanQuoteWithProgram(querierFromContext(ctx, r.pool).QueryRow(ctx, q, value))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrQuoteNotFound
		}

		return nil, fmt.Errorf("получение предложения: %w", err)
	}

	// Машина дочитывается отдельным запросом, а не join-ом: она приходит
	// вместе со своими характеристиками и фотографиями, и собрать это одной
	// выборкой означало бы разбирать декартово произведение вручную.
	machine, err := r.machines.GetByID(ctx, quote.MachineID)
	if err != nil {
		return nil, err
	}

	quote.Machine = machine

	return quote, nil
}

// ListByAuthor возвращает предложения менеджера, свежие сверху.
func (r quoteRepository) ListByAuthor(ctx context.Context, authorID string, limit int) (entity.Quotes, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s q %s WHERE q.author_id = $1
		ORDER BY q.created_at DESC LIMIT $2`, quoteListColumns, quoteTable, quoteListJoins)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, authorID, limit)
	if err != nil {
		return nil, fmt.Errorf("получение предложений менеджера: %w", err)
	}

	quotes, err := pgx.CollectRows(rows, scanQuoteListRow)
	if err != nil {
		return nil, fmt.Errorf("чтение предложений менеджера: %w", err)
	}

	return quotes, nil
}

// MarkOpened проставляет момент первого открытия.
//
// Условие first_opened_at IS NULL делает запись однократной на уровне базы:
// два одновременных открытия ссылки иначе записали бы разное время.
// Ноль изменённых строк — не ошибка: отметка уже стоит.
func (r quoteRepository) MarkOpened(ctx context.Context, id string, openedAt entity.Timestamp) error {
	q := fmt.Sprintf(`UPDATE %s SET first_opened_at = $1
		WHERE id = $2 AND first_opened_at IS NULL`, quoteTable)

	if _, err := querierFromContext(ctx, r.pool).Exec(ctx, q, openedAt, id); err != nil {
		return fmt.Errorf("отметка открытия предложения: %w", err)
	}

	return nil
}

// quoteScanExtras сообщает сканеру, что дочитано соединениями в конце выборки.
// Набор колонок задаёт запрос, и сканер обязан ждать ровно их.
type quoteScanExtras struct {
	program    bool // Название программы лизинга
	machineRef bool // Название машины и слаги для адреса карточки
}

// scanQuoteWithProgram читает строку предложения вместе с названием программы.
func scanQuoteWithProgram(row pgx.Row) (*entity.Quote, error) {
	return scanQuoteFields(row, quoteScanExtras{program: true})
}

// scanQuote читает строку коммерческого предложения.
func scanQuote(row pgx.Row) (*entity.Quote, error) {
	return scanQuoteFields(row, quoteScanExtras{})
}

// scanQuoteFields читает строку предложения; extras сообщает, что дочитано
// соединениями в конце выборки.
func scanQuoteFields(row pgx.Row, extras quoteScanExtras) (*entity.Quote, error) {
	var (
		quote         entity.Quote
		rateRaw       string
		scheduleRaw   []byte
		configuration []byte
	)

	targets := []any{
		&quote.ID, &quote.PublicToken, &quote.MachineID, &configuration,
		&quote.PriceCNYCents, &rateRaw, &quote.FXRateDate, &quote.FXMarkupBps, &quote.AssetCostKZTCents,
		&quote.LeasingProgramID, &quote.ScheduleType, &quote.DownPaymentBps,
		&quote.TermMonths, &quote.RateBps,
		&quote.ResidualValueKZTCents, &quote.CommissionKZTCents,
		&quote.FinancedAmountKZTCents, &quote.MonthlyPaymentKZTCents, &quote.LastPaymentKZTCents,
		&quote.TotalPaidKZTCents, &quote.OverpaymentKZTCents, &scheduleRaw,
		&quote.AuthorID, &quote.ExpiresAt, &quote.FirstOpenedAt, &quote.CreatedAt,
	}

	if extras.program {
		targets = append(targets, &quote.LeasingProgramName)
	}

	if extras.machineRef {
		targets = append(targets,
			&quote.MachineName, &quote.MachineSlug, &quote.MachineCategorySlug)
	}

	if err := row.Scan(targets...); err != nil {
		return nil, err
	}

	rate, err := decimal.NewFromString(rateRaw)
	if err != nil {
		return nil, fmt.Errorf("разбор курса %q: %w", rateRaw, err)
	}

	quote.FXRate = rate

	if err = json.Unmarshal(scheduleRaw, &quote.PaymentSchedule); err != nil {
		return nil, fmt.Errorf("разбор графика платежей: %w", err)
	}

	if err = json.Unmarshal(configuration, &quote.Configuration); err != nil {
		return nil, fmt.Errorf("разбор конфигурации: %w", err)
	}

	return &quote, nil
}

// scanQuoteListRow читает строку списка: с программой и ссылкой на машину.
func scanQuoteListRow(row pgx.CollectableRow) (*entity.Quote, error) {
	return scanQuoteFields(row, quoteScanExtras{program: true, machineRef: true})
}

// configurationOrEmpty приводит отсутствующую конфигурацию к пустой карте:
// колонка объявлена NOT NULL с умолчанием '{}', и null туда не запишется.
func configurationOrEmpty(configuration map[string]any) map[string]any {
	if configuration == nil {
		return map[string]any{}
	}

	return configuration
}

// leasingProgramColumns список колонок программы лизинга в порядке выборки.
const leasingProgramColumns = `id, name_ru, name_kk, name_en, logo_path,
	min_down_payment_bps, term_months_min, term_months_max, rate_min_bps, rate_max_bps,
	max_amount_cny_cents, is_active, position, created_at, updated_at`

// leasingProgramRepository репозиторий программ лизинга.
type leasingProgramRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewLeasingProgramRepository создаёт новый объект leasingProgramRepository.
func NewLeasingProgramRepository(pool *pgxpool.Pool) repository.LeasingProgramRepository {
	return &leasingProgramRepository{pool: pool}
}

// GetByID возвращает программу по идентификатору.
func (r leasingProgramRepository) GetByID(ctx context.Context, id string) (*entity.LeasingProgram, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, leasingProgramColumns, leasingProgramTable)

	program, err := scanLeasingProgram(querierFromContext(ctx, r.pool).QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrLeasingProgramNotFound
		}

		return nil, fmt.Errorf("получение программы лизинга: %w", err)
	}

	return program, nil
}

// ListActive возвращает действующие программы в порядке вывода.
func (r leasingProgramRepository) ListActive(ctx context.Context) (entity.LeasingPrograms, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE is_active ORDER BY position, id`,
		leasingProgramColumns, leasingProgramTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("получение программ лизинга: %w", err)
	}

	programs, err := pgx.CollectRows(rows, scanLeasingProgramRow)
	if err != nil {
		return nil, fmt.Errorf("чтение программ лизинга: %w", err)
	}

	return programs, nil
}

// ListAll возвращает все программы, включая недействующие.
func (r leasingProgramRepository) ListAll(ctx context.Context) (entity.LeasingPrograms, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s ORDER BY position, id`,
		leasingProgramColumns, leasingProgramTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("получение программ лизинга: %w", err)
	}

	programs, err := pgx.CollectRows(rows, scanLeasingProgramRow)
	if err != nil {
		return nil, fmt.Errorf("чтение программ лизинга: %w", err)
	}

	return programs, nil
}

// Create сохраняет программу лизинга.
func (r leasingProgramRepository) Create(
	ctx context.Context,
	program *entity.LeasingProgram,
) (*entity.LeasingProgram, error) {
	q := fmt.Sprintf(`INSERT INTO %s (name_ru, name_kk, name_en, logo_path,
			min_down_payment_bps, term_months_min, term_months_max,
			rate_min_bps, rate_max_bps, max_amount_cny_cents, is_active, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING %s`, leasingProgramTable, leasingProgramColumns)

	created, err := scanLeasingProgram(querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		program.NameRU, program.NameKK, program.NameEN, program.LogoPath,
		program.MinDownPaymentBps, program.TermMonthsMin, program.TermMonthsMax,
		program.RateMinBps, program.RateMaxBps, program.MaxAmountCNYCents,
		program.IsActive, program.Position))
	if err != nil {
		return nil, fmt.Errorf("сохранение программы лизинга: %w", err)
	}

	return created, nil
}

// Update обновляет редактируемые поля программы.
func (r leasingProgramRepository) Update(ctx context.Context, program *entity.LeasingProgram) error {
	q := fmt.Sprintf(`UPDATE %s SET name_ru = $2, name_kk = $3, name_en = $4, logo_path = $5,
			min_down_payment_bps = $6, term_months_min = $7, term_months_max = $8,
			rate_min_bps = $9, rate_max_bps = $10, max_amount_cny_cents = $11,
			is_active = $12, position = $13, updated_at = now()
		WHERE id = $1`, leasingProgramTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, program.ID,
		program.NameRU, program.NameKK, program.NameEN, program.LogoPath,
		program.MinDownPaymentBps, program.TermMonthsMin, program.TermMonthsMax,
		program.RateMinBps, program.RateMaxBps, program.MaxAmountCNYCents,
		program.IsActive, program.Position)
	if err != nil {
		return fmt.Errorf("обновление программы лизинга: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrLeasingProgramNotFound
	}

	return nil
}

// Delete удаляет программу лизинга.
func (r leasingProgramRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, leasingProgramTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		// ON DELETE RESTRICT на quotes.leasing_program_id: по программе есть
		// выпущенные предложения, и стереть условия, на которых они выданы,
		// нельзя.
		if isForeignKeyViolation(err) {
			return entity.ErrLeasingProgramInUse
		}

		return fmt.Errorf("удаление программы лизинга: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrLeasingProgramNotFound
	}

	return nil
}

// scanLeasingProgram читает строку программы лизинга.
func scanLeasingProgram(row pgx.Row) (*entity.LeasingProgram, error) {
	var program entity.LeasingProgram

	if err := row.Scan(&program.ID, &program.NameRU, &program.NameKK, &program.NameEN,
		&program.LogoPath, &program.MinDownPaymentBps,
		&program.TermMonthsMin, &program.TermMonthsMax,
		&program.RateMinBps, &program.RateMaxBps, &program.MaxAmountCNYCents,
		&program.IsActive, &program.Position, &program.CreatedAt, &program.UpdatedAt); err != nil {
		return nil, err
	}

	return &program, nil
}

// scanLeasingProgramRow адаптирует scanLeasingProgram к pgx.CollectRows.
func scanLeasingProgramRow(row pgx.CollectableRow) (*entity.LeasingProgram, error) {
	return scanLeasingProgram(row)
}

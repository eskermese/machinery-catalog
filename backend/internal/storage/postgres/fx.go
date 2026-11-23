package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// fxRateColumns список колонок курса в порядке выборки.
const fxRateColumns = `id, base_currency, quote_currency, source, rate, rate_date, fetched_at`

// fxMarkupColumns список колонок наценки в порядке выборки.
const fxMarkupColumns = `id, base_currency, quote_currency, markup_bps, effective_from, changed_by, created_at`

// fxRateRepository репозиторий курсов валют.
//
// Методов UPDATE и DELETE в реализации нет вовсе, а не «есть, но не вызываются»:
// таблица append-only, и отсутствие кода — самая надёжная гарантия того,
// что уже показанный клиенту курс не перепишут.
type fxRateRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewFXRateRepository создаёт новый объект fxRateRepository.
func NewFXRateRepository(pool *pgxpool.Pool) repository.FXRateRepository {
	return &fxRateRepository{pool: pool}
}

// Create сохраняет курс источника за дату.
// Нарушение уникальности (источник, пара, дата) приводится к доменному конфликту.
func (r fxRateRepository) Create(ctx context.Context, rate *entity.FXRate) (*entity.FXRate, error) {
	if err := rate.Validate(); err != nil {
		return nil, fmt.Errorf("проверка курса: %w", err)
	}

	q := fmt.Sprintf(`INSERT INTO %s (base_currency, quote_currency, source, rate, rate_date, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING %s`, fxRateTable, fxRateColumns)

	// Курс уходит строкой: numeric принимает её без потерь и без предположений
	// о том, как драйвер отобразит произвольную точность на свои типы.
	created, err := scanFXRate(querierFromContext(ctx, r.pool).QueryRow(ctx, q,
		rate.BaseCurrency, rate.QuoteCurrency, rate.Source,
		rate.Rate.StringFixed(entity.FXRateScale), rate.RateDate, rate.FetchedAt,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, entity.ErrFXRateAlreadyExists
		}

		return nil, fmt.Errorf("сохранение курса: %w", err)
	}

	return created, nil
}

// GetLatest возвращает последний известный курс пары.
//
// Здесь применяется приоритет источников, объявленный в домене
// (entity.FXSourceManual): сначала самая поздняя дата, внутри неё —
// ручной курс, и лишь затем загруженный воркером. Ручной ввод работает
// как замена: администратор поправил курс за сегодня — берётся его,
// не поправил — берётся то, что принёс воркер.
//
// fetched_at остаётся последним разрывом ничьей: два курса одного вида
// за одну дату означают повторную загрузку, и свежее тот, что позже.
func (r fxRateRepository) GetLatest(ctx context.Context, base, quote string) (*entity.FXRate, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE base_currency = $1 AND quote_currency = $2
		ORDER BY rate_date DESC, (source = $3) DESC, fetched_at DESC LIMIT 1`,
		fxRateColumns, fxRateTable)

	rate, err := scanFXRate(
		querierFromContext(ctx, r.pool).QueryRow(ctx, q, base, quote, entity.FXSourceManual),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrFXRateNotFound
		}

		return nil, fmt.Errorf("получение последнего курса: %w", err)
	}

	return rate, nil
}

// GetLatestBySource возвращает последний курс пары от конкретного источника.
//
// Нужен админке: рядом показываются то, что предложил внешний источник,
// и то, что ввёл человек. Без этого «действующий курс» не с чем сравнить,
// и админ не видит, поправил он что-то или согласился.
func (r fxRateRepository) GetLatestBySource(
	ctx context.Context,
	base, quote, source string,
) (*entity.FXRate, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s
		WHERE base_currency = $1 AND quote_currency = $2 AND source = $3
		ORDER BY rate_date DESC, fetched_at DESC LIMIT 1`, fxRateColumns, fxRateTable)

	rate, err := scanFXRate(querierFromContext(ctx, r.pool).QueryRow(ctx, q, base, quote, source))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrFXRateNotFound
		}

		return nil, fmt.Errorf("получение курса источника: %w", err)
	}

	return rate, nil
}

// GetLatestExceptSource возвращает последний курс пары от любого источника,
// кроме указанного.
//
// Зеркало GetLatestBySource и нужно ровно там же — в админке. Своим методом,
// а не переиспользованным GetLatest: тот применяет приоритет источников
// и при наличии ручного курса вернул бы именно его, то есть скрыл бы то,
// с чем администратор и должен сравнивать.
func (r fxRateRepository) GetLatestExceptSource(
	ctx context.Context,
	base, quote, source string,
) (*entity.FXRate, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s
		WHERE base_currency = $1 AND quote_currency = $2 AND source <> $3
		ORDER BY rate_date DESC, fetched_at DESC LIMIT 1`, fxRateColumns, fxRateTable)

	rate, err := scanFXRate(querierFromContext(ctx, r.pool).QueryRow(ctx, q, base, quote, source))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrFXRateNotFound
		}

		return nil, fmt.Errorf("получение курса внешнего источника: %w", err)
	}

	return rate, nil
}

// ListBySource возвращает историю курсов пары от источника, свежие первыми.
func (r fxRateRepository) ListBySource(
	ctx context.Context,
	base, quote, source string,
	limit int,
) (entity.FXRates, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s
		WHERE base_currency = $1 AND quote_currency = $2 AND source = $3
		ORDER BY rate_date DESC, fetched_at DESC LIMIT $4`, fxRateColumns, fxRateTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, base, quote, source, limit)
	if err != nil {
		return nil, fmt.Errorf("получение истории курсов: %w", err)
	}
	defer rows.Close()

	rates := make(entity.FXRates, 0, limit)

	for rows.Next() {
		rate, scanErr := scanFXRate(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("чтение курса: %w", scanErr)
		}

		rates = append(rates, rate)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("обход курсов: %w", rows.Err())
	}

	return rates, nil
}

// scanFXRate читает строку курса.
//
// Ручной скан, а не RowToAddrOfStructByName: курс приходит из numeric и
// разбирается в decimal через строку, чтобы точность не зависела от того,
// в какой из своих типов драйвер решит его отобразить.
func scanFXRate(row pgx.Row) (*entity.FXRate, error) {
	var (
		rate    entity.FXRate
		rateRaw string
	)

	if err := row.Scan(&rate.ID, &rate.BaseCurrency, &rate.QuoteCurrency, &rate.Source,
		&rateRaw, &rate.RateDate, &rate.FetchedAt); err != nil {
		return nil, err
	}

	parsed, err := decimal.NewFromString(rateRaw)
	if err != nil {
		return nil, fmt.Errorf("разбор курса %q: %w", rateRaw, err)
	}

	rate.Rate = parsed

	return &rate, nil
}

// fxMarkupRepository репозиторий наценки к курсу.
type fxMarkupRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewFXMarkupRepository создаёт новый объект fxMarkupRepository.
func NewFXMarkupRepository(pool *pgxpool.Pool) repository.FXMarkupRepository {
	return &fxMarkupRepository{pool: pool}
}

// Create добавляет запись наценки.
func (r fxMarkupRepository) Create(ctx context.Context, markup *entity.FXMarkup) (*entity.FXMarkup, error) {
	if err := markup.Validate(); err != nil {
		return nil, fmt.Errorf("проверка наценки: %w", err)
	}

	q := fmt.Sprintf(`INSERT INTO %s (base_currency, quote_currency, markup_bps, effective_from, changed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING %s`, fxMarkupTable, fxMarkupColumns)

	created, err := queryOneRow[entity.FXMarkup](ctx, querierFromContext(ctx, r.pool), q, "наценки",
		entity.ErrFXRateNotFound,
		markup.BaseCurrency, markup.QuoteCurrency, markup.MarkupBps,
		markup.EffectiveFrom, markup.ChangedBy, markup.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("сохранение наценки: %w", err)
	}

	return created, nil
}

// GetEffective возвращает наценку, действующую на переданный момент.
//
// Записи с effective_from в будущем отсекаются условием, а не сортировкой:
// наценку задают заранее, и до своего момента она не должна применяться.
// Отсутствие записи возвращается как (nil, nil) — наценка может быть не задана,
// и это штатное «продаём по курсу источника», а не отказ.
func (r fxMarkupRepository) GetEffective(
	ctx context.Context,
	base, quote string,
	at time.Time,
) (*entity.FXMarkup, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s
		WHERE base_currency = $1 AND quote_currency = $2 AND effective_from <= $3
		ORDER BY effective_from DESC LIMIT 1`, fxMarkupColumns, fxMarkupTable)

	markup, err := queryOneRow[entity.FXMarkup](ctx, querierFromContext(ctx, r.pool), q, "наценки",
		errNoMarkup, base, quote, at)
	if err != nil {
		if errors.Is(err, errNoMarkup) {
			return nil, nil //nolint:nilnil // отсутствие наценки — штатное состояние, а не отказ
		}

		return nil, fmt.Errorf("получение действующей наценки: %w", err)
	}

	return markup, nil
}

// List возвращает историю наценок пары, свежие первыми.
//
// Нужна админке: без истории «действующая наценка 2 %» ничего не говорит
// о том, когда она такой стала и что было до неё, а именно это объясняет
// цену в выданном ранее предложении.
func (r fxMarkupRepository) List(
	ctx context.Context,
	base, quote string,
	limit int,
) (entity.FXMarkups, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s
		WHERE base_currency = $1 AND quote_currency = $2
		ORDER BY effective_from DESC, created_at DESC LIMIT $3`, fxMarkupColumns, fxMarkupTable)

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, base, quote, limit)
	if err != nil {
		return nil, fmt.Errorf("получение истории наценок: %w", err)
	}

	// Наценка целочисленная, decimal в ней нет — читается разбором по именам,
	// в отличие от курса, которому нужен ручной скан из-за numeric.
	markups, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[entity.FXMarkup])
	if err != nil {
		return nil, fmt.Errorf("чтение наценок: %w", err)
	}

	return markups, nil
}

// errNoMarkup внутренний признак отсутствия наценки.
// Наружу не выходит: queryOneRow требует ошибку для случая пустой выборки,
// а домену про «наценка не задана» нужно знать как про nil, а не как про отказ.
var errNoMarkup = errors.New("наценка не задана")

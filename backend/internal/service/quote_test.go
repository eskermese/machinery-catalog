package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// Идентификаторы в тестах предложений.
const (
	testQuoteID    = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testProgramID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	testAuthorID   = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	testQuoteToken = "0123456789abcdef0123456789abcdef0123456789"
)

// testQuoteNow момент операций в тестах предложений.
var testQuoteNow = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

// quoteMocks набор моков для сервиса предложений.
type quoteMocks struct {
	quotes   *mock_repo.MockQuoteRepository
	programs *mock_repo.MockLeasingProgramRepository
	catalog  *stubCatalogReader
	fx       *stubFXFixer
}

// stubCatalogReader отдаёт карточку для выпуска предложения.
type stubCatalogReader struct {
	CatalogService // Остальные методы сервису предложений не нужны

	machine *entity.Machine
	err     error
}

func (s *stubCatalogReader) PublicMachineBySlug(context.Context, string) (*entity.Machine, error) {
	return s.machine, s.err
}

// stubFXFixer фиксирует курс для предложения.
type stubFXFixer struct {
	FXService // Остальные методы сервису предложений не нужны

	applied entity.AppliedFXRate
	err     error
	calls   int // Сколько раз курс фиксировали
	reads   int // Сколько раз курс читали без фиксации
}

func (s *stubFXFixer) FixRateForQuote(
	context.Context, string, string, time.Time,
) (entity.AppliedFXRate, error) {
	s.calls++

	return s.applied, s.err
}

func (s *stubFXFixer) AppliedRate(
	context.Context, string, string, time.Time,
) (entity.AppliedFXRate, error) {
	s.reads++

	return s.applied, s.err
}

// newQuoteService собирает сервис предложений на моках.
func newQuoteService(t *testing.T, machine *entity.Machine, applied entity.AppliedFXRate) (QuoteService, quoteMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mocks := quoteMocks{
		quotes:   mock_repo.NewMockQuoteRepository(ctrl),
		programs: mock_repo.NewMockLeasingProgramRepository(ctrl),
		catalog:  &stubCatalogReader{machine: machine},
		fx:       &stubFXFixer{applied: applied},
	}

	return NewQuoteService(mocks.quotes, mocks.programs, mocks.catalog, mocks.fx), mocks
}

// testQuoteMachine собирает опубликованную карточку.
func testQuoteMachine() *entity.Machine {
	return &entity.Machine{
		ID: testMachineID, Slug: testMachineSlug, CategoryID: testCategoryID,
		Brand: "XCMG", Model: "XE215C",
		NameRU: "X", NameKK: "X", NameEN: "X",
		PriceCNYCents: 10_000_000, IsPricePublic: true,
		Availability: entity.AvailabilityInStock, Condition: entity.ConditionNew,
		IsActive: true, PublishedAt: entity.Pointer(testQuoteNow),
	}
}

// testAppliedRate собирает применённый курс.
func testAppliedRate(t *testing.T, rate string) entity.AppliedFXRate {
	t.Helper()

	value, err := decimal.NewFromString(rate)
	require.NoError(t, err)

	return entity.AppliedFXRate{Rate: value, RateDate: testQuoteNow, MarkupBps: 200}
}

// testIssue собирает валидные параметры выпуска.
func testIssue() *QuoteIssue {
	return &QuoteIssue{
		MachineSlug:    testMachineSlug,
		ScheduleType:   entity.LeasingScheduleAnnuity,
		DownPaymentBps: 2000,
		TermMonths:     36,
		RateBps:        1800,
		AuthorID:       testAuthorID,
	}
}

// testProgram собирает действующую программу лизинга.
func testProgram() *entity.LeasingProgram {
	return &entity.LeasingProgram{
		ID: testProgramID, IsActive: true,
		MinDownPaymentBps: 1500,
		TermMonthsMin:     12, TermMonthsMax: 60,
		RateMinBps: 1800, RateMaxBps: 2400,
		MaxAmountCNYCents: entity.Pointer(int64(50_000_000)),
	}
}

// TestQuoteService_Issue_FixesRate фиксирует, что курс берётся через
// FixRateForQuote и попадает в предложение вместе с датой и наценкой.
func TestQuoteService_Issue_FixesRate(t *testing.T) {
	t.Parallel()

	applied := testAppliedRate(t, "500")
	srv, mocks := newQuoteService(t, testQuoteMachine(), applied)

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		})

	quote, err := srv.Issue(context.Background(), testIssue(), testQuoteNow)
	require.NoError(t, err)

	assert.Equal(t, 1, mocks.fx.calls, "курс фиксируется ровно один раз")
	assert.Equal(t, "500", quote.FXRate.String())
	assert.Equal(t, testQuoteNow, quote.FXRateDate)
	assert.Equal(t, 200, quote.FXMarkupBps, "наценка сохраняется вместе с курсом")

	// Цена 10 000 000 центов USD по курсу 500 — 5 000 000 000 тиынов.
	assert.Equal(t, int64(5_000_000_000), quote.AssetCostKZTCents,
		"стоимость в тиынах хранится, а не выводится на лету")
	assert.Equal(t, int64(10_000_000), quote.PriceCNYCents, "долларовая цена тоже сохраняется")
}

// TestQuoteService_Issue_CalculationLandsInQuote фиксирует, что весь результат
// калькулятора попадает в строку предложения.
func TestQuoteService_Issue_CalculationLandsInQuote(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		})

	issue := testIssue()
	issue.CommissionKZTCents = 50_000

	quote, err := srv.Issue(context.Background(), issue, testQuoteNow)
	require.NoError(t, err)

	// Вход сохранён целиком.
	assert.Equal(t, entity.LeasingScheduleAnnuity, quote.ScheduleType)
	assert.Equal(t, 2000, quote.DownPaymentBps)
	assert.Equal(t, 36, quote.TermMonths)
	assert.Equal(t, 1800, quote.RateBps)
	assert.Equal(t, int64(50_000), quote.CommissionKZTCents)

	// Результат сохранён целиком.
	assert.Equal(t, int64(4_000_000_000), quote.FinancedAmountKZTCents,
		"финансирование = 5 000 000 000 минус взнос 20%")
	assert.Positive(t, quote.MonthlyPaymentKZTCents)
	assert.Positive(t, quote.LastPaymentKZTCents)
	assert.Positive(t, quote.TotalPaidKZTCents)
	assert.Positive(t, quote.OverpaymentKZTCents)
	assert.Len(t, quote.PaymentSchedule, 36, "график сохраняется целиком")

	// Инвариант калькулятора переживает перенос в предложение.
	var sumAmount int64
	for _, payment := range quote.PaymentSchedule {
		sumAmount += payment.AmountCents
	}

	assert.Equal(t, quote.TotalPaidKZTCents-quote.CommissionKZTCents, sumAmount,
		"итого выплат = сумма графика плюс комиссия")
}

// TestQuoteService_Issue_ChecksProgramBounds — главная проверка части 2:
// параметры сверяются с границами программы, и каждое нарушение имеет
// собственную ошибку.
func TestQuoteService_Issue_ChecksProgramBounds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		mutate  func(*QuoteIssue)
		program func(*entity.LeasingProgram)
		expErr  error
	}{
		{
			name:   "взнос меньше минимального",
			mutate: func(i *QuoteIssue) { i.DownPaymentBps = 1000 },
			expErr: entity.ErrLeasingDownPaymentTooLow,
		},
		{
			name:   "срок меньше нижней границы",
			mutate: func(i *QuoteIssue) { i.TermMonths = 6 },
			expErr: entity.ErrLeasingTermOutOfRange,
		},
		{
			name:   "срок больше верхней границы",
			mutate: func(i *QuoteIssue) { i.TermMonths = 72 },
			expErr: entity.ErrLeasingTermOutOfRange,
		},
		{
			name:   "ставка ниже диапазона",
			mutate: func(i *QuoteIssue) { i.RateBps = 1000 },
			expErr: entity.ErrLeasingRateOutOfRange,
		},
		{
			name:   "ставка выше диапазона",
			mutate: func(i *QuoteIssue) { i.RateBps = 3000 },
			expErr: entity.ErrLeasingRateOutOfRange,
		},
		{
			name:    "сумма выше максимума программы",
			program: func(p *entity.LeasingProgram) { p.MaxAmountCNYCents = entity.Pointer(int64(5_000_000)) },
			expErr:  entity.ErrLeasingAmountAboveMaximum,
		},
		{
			name:    "программа не действует",
			program: func(p *entity.LeasingProgram) { p.IsActive = false },
			expErr:  entity.ErrLeasingProgramInactive,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

			program := testProgram()
			if c.program != nil {
				c.program(program)
			}

			mocks.programs.EXPECT().GetByID(gomock.Any(), testProgramID).Return(program, nil)
			// До сохранения дело не доходит.
			mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			issue := testIssue()
			issue.LeasingProgramID = testProgramID

			if c.mutate != nil {
				c.mutate(issue)
			}

			_, err := srv.Issue(context.Background(), issue, testQuoteNow)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestQuoteService_Issue_WithinProgramBounds фиксирует, что параметры внутри
// границ программы проходят и программа попадает в предложение.
func TestQuoteService_Issue_WithinProgramBounds(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	mocks.programs.EXPECT().GetByID(gomock.Any(), testProgramID).Return(testProgram(), nil)
	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		})

	issue := testIssue()
	issue.LeasingProgramID = testProgramID

	quote, err := srv.Issue(context.Background(), issue, testQuoteNow)
	require.NoError(t, err)

	require.NotNil(t, quote.LeasingProgramID)
	assert.Equal(t, testProgramID, *quote.LeasingProgramID)
}

// TestQuoteService_Issue_WithoutProgram фиксирует расчёт по переданным вручную
// параметрам: программа опциональна.
func TestQuoteService_Issue_WithoutProgram(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	// Программа не читается вовсе.
	mocks.programs.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(0)
	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		})

	// Взнос ниже, чем у любой программы, — без программы это допустимо.
	issue := testIssue()
	issue.DownPaymentBps = 500

	quote, err := srv.Issue(context.Background(), issue, testQuoteNow)
	require.NoError(t, err)

	assert.Nil(t, quote.LeasingProgramID)
	assert.Equal(t, 500, quote.DownPaymentBps)
}

// TestQuoteService_Issue_DraftMachineRejected фиксирует, что предложение
// по неопубликованной карточке не выпускается.
func TestQuoteService_Issue_DraftMachineRejected(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, testAppliedRate(t, "500"))
	mocks.catalog.err = entity.ErrMachineNotFound

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.Issue(context.Background(), testIssue(), testQuoteNow)

	require.ErrorIs(t, err, entity.ErrMachineNotFound)
	assert.Zero(t, mocks.fx.calls, "курс не фиксируется, если карточки нет")
}

// TestQuoteService_Issue_FXFailureAborts фиксирует, что без курса предложение
// не выпускается: считать по подставленной единице нельзя.
func TestQuoteService_Issue_FXFailureAborts(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{})
	mocks.fx.err = entity.ErrFXRateNotFound

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.Issue(context.Background(), testIssue(), testQuoteNow)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
}

// TestQuoteService_Issue_TokenIsRandomAndLongEnough фиксирует свойства токена:
// длина не меньше границы схемы и разные значения при каждом выпуске.
func TestQuoteService_Issue_TokenIsRandomAndLongEnough(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		}).Times(2)

	first, err := srv.Issue(context.Background(), testIssue(), testQuoteNow)
	require.NoError(t, err)

	second, err := srv.Issue(context.Background(), testIssue(), testQuoteNow)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(first.PublicToken), entity.MinQuoteTokenLength)
	assert.NotEqual(t, first.PublicToken, second.PublicToken, "токены не повторяются")
}

// TestQuoteService_Issue_ExpiresAt фиксирует срок годности: переданный и по умолчанию.
func TestQuoteService_Issue_ExpiresAt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		validFor time.Duration
		expDays  int
	}{
		{name: "срок по умолчанию", validFor: 0, expDays: 14},
		{name: "переданный срок", validFor: 30 * 24 * time.Hour, expDays: 30},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

			mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
					return q, nil
				})

			issue := testIssue()
			issue.ValidFor = c.validFor

			quote, err := srv.Issue(context.Background(), issue, testQuoteNow)
			require.NoError(t, err)

			assert.Equal(t, testQuoteNow.AddDate(0, 0, c.expDays), quote.ExpiresAt)
			assert.False(t, quote.IsExpired(testQuoteNow), "свежее предложение не истекло")
		})
	}
}

// TestQuoteService_ByPublicToken_MarksOpenedOnce — проверка однократности
// отметки первого открытия.
func TestQuoteService_ByPublicToken_MarksOpenedOnce(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	stored := &entity.Quote{
		ID: testQuoteID, PublicToken: testQuoteToken,
		ExpiresAt: testQuoteNow.AddDate(0, 0, 14), CreatedAt: testQuoteNow,
	}

	// Первое открытие: отметка ставится.
	mocks.quotes.EXPECT().GetByToken(gomock.Any(), testQuoteToken).Return(stored, nil)
	mocks.quotes.EXPECT().MarkOpened(gomock.Any(), testQuoteID, testQuoteNow).Return(nil)

	first, err := srv.ByPublicToken(context.Background(), testQuoteToken, testQuoteNow)
	require.NoError(t, err)
	require.NotNil(t, first.FirstOpenedAt)
	assert.Equal(t, testQuoteNow, *first.FirstOpenedAt)

	// Второе открытие часом позже: отметка НЕ перезаписывается и в базу
	// повторно не идёт.
	later := testQuoteNow.Add(time.Hour)
	opened := &entity.Quote{
		ID: testQuoteID, PublicToken: testQuoteToken,
		FirstOpenedAt: entity.Pointer(testQuoteNow),
		ExpiresAt:     testQuoteNow.AddDate(0, 0, 14), CreatedAt: testQuoteNow,
	}

	mocks.quotes.EXPECT().GetByToken(gomock.Any(), testQuoteToken).Return(opened, nil)
	mocks.quotes.EXPECT().MarkOpened(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	second, err := srv.ByPublicToken(context.Background(), testQuoteToken, later)
	require.NoError(t, err)

	require.NotNil(t, second.FirstOpenedAt)
	assert.Equal(t, testQuoteNow, *second.FirstOpenedAt, "момент первого открытия не сдвигается")
}

// TestQuoteService_ByPublicToken_ExpiredIsReturned — проверка признака истечения:
// просроченное предложение читается, а не отвергается.
func TestQuoteService_ByPublicToken_ExpiredIsReturned(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	expired := &entity.Quote{
		ID: testQuoteID, PublicToken: testQuoteToken,
		FirstOpenedAt: entity.Pointer(testQuoteNow),
		ExpiresAt:     testQuoteNow, CreatedAt: testQuoteNow.AddDate(0, 0, -14),
	}

	mocks.quotes.EXPECT().GetByToken(gomock.Any(), testQuoteToken).Return(expired, nil)

	later := testQuoteNow.Add(time.Hour)

	quote, err := srv.ByPublicToken(context.Background(), testQuoteToken, later)

	require.NoError(t, err, "истёкшее предложение читается, а не отвергается")
	assert.True(t, quote.IsExpired(later), "но помечено как истёкшее")
	assert.False(t, quote.IsExpired(testQuoteNow.Add(-time.Hour)), "до срока — действующее")
}

// TestQuoteService_ByPublicToken_ShortTokenRejected фиксирует, что короткий
// токен отвергается до похода в базу: он не мог быть выдан.
func TestQuoteService_ByPublicToken_ShortTokenRejected(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	mocks.quotes.EXPECT().GetByToken(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.ByPublicToken(context.Background(), "short", testQuoteNow)

	require.ErrorIs(t, err, entity.ErrQuoteNotFound)
}

// TestQuoteService_ByPublicToken_MarkFailureDoesNotBreakRead фиксирует, что
// неудачная отметка не мешает клиенту увидеть предложение.
func TestQuoteService_ByPublicToken_MarkFailureDoesNotBreakRead(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	stored := &entity.Quote{
		ID: testQuoteID, PublicToken: testQuoteToken,
		ExpiresAt: testQuoteNow.AddDate(0, 0, 14), CreatedAt: testQuoteNow,
	}

	mocks.quotes.EXPECT().GetByToken(gomock.Any(), testQuoteToken).Return(stored, nil)
	mocks.quotes.EXPECT().MarkOpened(gomock.Any(), testQuoteID, testQuoteNow).
		Return(errors.New("база недоступна"))

	quote, err := srv.ByPublicToken(context.Background(), testQuoteToken, testQuoteNow)

	require.NoError(t, err, "отметка — вспомогательный эффект просмотра")
	assert.Equal(t, testQuoteID, quote.ID)
}

// TestQuoteService_ByPublicToken_NotFound фиксирует проброс отсутствия.
func TestQuoteService_ByPublicToken_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	mocks.quotes.EXPECT().GetByToken(gomock.Any(), testQuoteToken).Return(nil, entity.ErrQuoteNotFound)

	_, err := srv.ByPublicToken(context.Background(), testQuoteToken, testQuoteNow)

	require.ErrorIs(t, err, entity.ErrQuoteNotFound)
}

// TestQuoteService_ByID фиксирует чтение для админки: отметка открытия не ставится.
func TestQuoteService_ByID(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	stored := &entity.Quote{ID: testQuoteID, PublicToken: testQuoteToken}

	mocks.quotes.EXPECT().GetByID(gomock.Any(), testQuoteID).Return(stored, nil)
	mocks.quotes.EXPECT().MarkOpened(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	quote, err := srv.ByID(context.Background(), testQuoteID)
	require.NoError(t, err)

	assert.Nil(t, quote.FirstOpenedAt, "менеджер, смотрящий свой список, — не клиент")
}

// TestQuoteService_ListByAuthor фиксирует урезание размера страницы.
func TestQuoteService_ListByAuthor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		limit    int
		expLimit int
	}{
		{name: "лимит не задан", limit: 0, expLimit: entity.DefaultLeadLimit},
		{name: "лимит в пределах", limit: 10, expLimit: 10},
		{name: "лимит сверх предела", limit: 10_000, expLimit: entity.DefaultLeadLimit},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

			mocks.quotes.EXPECT().ListByAuthor(gomock.Any(), testAuthorID, c.expLimit).
				Return(entity.Quotes{}, nil)

			_, err := srv.ListByAuthor(context.Background(), testAuthorID, c.limit)
			require.NoError(t, err)
		})
	}
}

// TestQuoteService_LeasingPrograms фиксирует чтение действующих программ.
func TestQuoteService_LeasingPrograms(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	stored := entity.LeasingPrograms{testProgram()}

	mocks.programs.EXPECT().ListActive(gomock.Any()).Return(stored, nil)

	programs, err := srv.LeasingPrograms(context.Background())
	require.NoError(t, err)

	assert.Equal(t, stored, programs)
}

// TestQuoteService_LeasingPrograms_Failure фиксирует проброс отказа.
func TestQuoteService_LeasingPrograms_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	repoErr := errors.New("база недоступна")

	mocks.programs.EXPECT().ListActive(gomock.Any()).Return(nil, repoErr)

	_, err := srv.LeasingPrograms(context.Background())

	require.ErrorIs(t, err, repoErr)
}

// TestQuoteService_Issue_SaveFailure фиксирует проброс отказа сохранения.
func TestQuoteService_Issue_SaveFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	saveErr := errors.New("база недоступна")

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, saveErr)

	_, err := srv.Issue(context.Background(), testIssue(), testQuoteNow)

	require.ErrorIs(t, err, saveErr)
}

// TestQuoteService_Issue_InvalidCalculationParams фиксирует, что отказ
// калькулятора поднимается наверх.
func TestQuoteService_Issue_InvalidCalculationParams(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	issue := testIssue()
	issue.TermMonths = 0

	_, err := srv.Issue(context.Background(), issue, testQuoteNow)

	require.Error(t, err)
}

// TestQuoteService_Issue_ProgramNotFound фиксирует проброс отсутствия программы.
func TestQuoteService_Issue_ProgramNotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))

	mocks.programs.EXPECT().GetByID(gomock.Any(), testProgramID).
		Return(nil, entity.ErrLeasingProgramNotFound)
	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	issue := testIssue()
	issue.LeasingProgramID = testProgramID

	_, err := srv.Issue(context.Background(), issue, testQuoteNow)

	require.ErrorIs(t, err, entity.ErrLeasingProgramNotFound)
}

// TestQuoteService_SaveLeasingProgram фиксирует создание и обновление одной
// точкой входа: различает их заполненный идентификатор.
func TestQuoteService_SaveLeasingProgram(t *testing.T) {
	t.Parallel()

	t.Run("создание", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

		program := testLeasingProgram()
		program.ID = ""

		mocks.programs.EXPECT().Create(gomock.Any(), program).Return(program, nil)

		saved, err := srv.SaveLeasingProgram(context.Background(), program)

		require.NoError(t, err)
		assert.Equal(t, program, saved)
	})

	t.Run("обновление перечитывает сохранённое", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

		program := testLeasingProgram()

		mocks.programs.EXPECT().Update(gomock.Any(), program).Return(nil)
		mocks.programs.EXPECT().GetByID(gomock.Any(), program.ID).Return(program, nil)

		saved, err := srv.SaveLeasingProgram(context.Background(), program)

		require.NoError(t, err)
		assert.Equal(t, program, saved)
	})

	t.Run("пустая программа", func(t *testing.T) {
		t.Parallel()

		srv, _ := newQuoteService(t, nil, entity.AppliedFXRate{})

		_, err := srv.SaveLeasingProgram(context.Background(), nil)

		require.ErrorIs(t, err, entity.ErrNilPointer)
	})
}

// TestQuoteService_SaveLeasingProgram_InvalidBounds фиксирует, что несогласованные
// границы отвергаются до похода в базу: программа с нижней границей выше верхней
// оставила бы калькулятору пустое множество допустимых значений.
func TestQuoteService_SaveLeasingProgram_InvalidBounds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*entity.LeasingProgram)
		expErr error
	}{
		{
			name:   "срок наоборот",
			mutate: func(p *entity.LeasingProgram) { p.TermMonthsMin, p.TermMonthsMax = 60, 12 },
			expErr: entity.ErrLeasingTermOutOfRange,
		},
		{
			name:   "ставка наоборот",
			mutate: func(p *entity.LeasingProgram) { p.RateMinBps, p.RateMaxBps = 2400, 1800 },
			expErr: entity.ErrLeasingRateOutOfRange,
		},
		{
			name:   "взнос больше ста процентов",
			mutate: func(p *entity.LeasingProgram) { p.MinDownPaymentBps = 10_001 },
			expErr: entity.ErrQuoteDownPaymentInvalid,
		},
		{
			name:   "нет русского названия",
			mutate: func(p *entity.LeasingProgram) { p.NameRU = "" },
			expErr: entity.ErrLeasingProgramNameRequired,
		},
		{
			name:   "отрицательный порядок",
			mutate: func(p *entity.LeasingProgram) { p.Position = -1 },
			expErr: entity.ErrLeasingProgramPositionInvalid,
		},
		{
			name:   "нулевой потолок суммы",
			mutate: func(p *entity.LeasingProgram) { p.MaxAmountCNYCents = entity.Pointer(int64(0)) },
			expErr: entity.ErrLeasingAmountAboveMaximum,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

			mocks.programs.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			mocks.programs.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			program := testLeasingProgram()
			c.mutate(program)

			_, err := srv.SaveLeasingProgram(context.Background(), program)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestQuoteService_DeleteLeasingProgram фиксирует удаление и отказ по программе
// с выпущенными предложениями.
func TestQuoteService_DeleteLeasingProgram(t *testing.T) {
	t.Parallel()

	t.Run("удаление", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

		mocks.programs.EXPECT().Delete(gomock.Any(), "program-1").Return(nil)

		require.NoError(t, srv.DeleteLeasingProgram(context.Background(), "program-1"))
	})

	t.Run("программа в использовании", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

		mocks.programs.EXPECT().Delete(gomock.Any(), "program-1").
			Return(entity.ErrLeasingProgramInUse)

		err := srv.DeleteLeasingProgram(context.Background(), "program-1")

		require.ErrorIs(t, err, entity.ErrLeasingProgramInUse)
	})
}

// TestQuoteService_AllLeasingPrograms фиксирует, что админке отдаются
// и выключенные программы.
func TestQuoteService_AllLeasingPrograms(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	programs := entity.LeasingPrograms{
		testLeasingProgram(),
		{ID: "program-2", IsActive: false},
	}

	mocks.programs.EXPECT().ListAll(gomock.Any()).Return(programs, nil)

	all, err := srv.AllLeasingPrograms(context.Background())

	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.False(t, all[1].IsActive, "выключенную программу нужно видеть, чтобы включить обратно")
}

// TestQuoteService_AllLeasingPrograms_Failure фиксирует отказ хранилища.
func TestQuoteService_AllLeasingPrograms_Failure(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

	mocks.programs.EXPECT().ListAll(gomock.Any()).Return(nil, errors.New("база недоступна"))

	_, err := srv.AllLeasingPrograms(context.Background())

	require.Error(t, err)
}

// testLeasingProgram собирает корректную программу лизинга.
func testLeasingProgram() *entity.LeasingProgram {
	return &entity.LeasingProgram{
		ID:                "program-1",
		NameRU:            "Halyk Leasing",
		NameKK:            "Halyk Leasing",
		NameEN:            "Halyk Leasing",
		MinDownPaymentBps: 1500,
		TermMonthsMin:     12,
		TermMonthsMax:     60,
		RateMinBps:        1800,
		RateMaxBps:        2400,
		IsActive:          true,
		Position:          1,
	}
}

// TestQuoteService_SaveLeasingProgram_StorageFailure фиксирует, что отказ
// хранилища доходит до вызывающего кода на обоих путях.
func TestQuoteService_SaveLeasingProgram_StorageFailure(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("база недоступна")

	t.Run("отказ создания", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

		program := testLeasingProgram()
		program.ID = ""

		mocks.programs.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, storageErr)

		_, err := srv.SaveLeasingProgram(context.Background(), program)

		require.Error(t, err)
	})

	t.Run("отказ обновления", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newQuoteService(t, nil, entity.AppliedFXRate{})

		mocks.programs.EXPECT().Update(gomock.Any(), gomock.Any()).Return(storageErr)
		mocks.programs.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(0)

		_, err := srv.SaveLeasingProgram(context.Background(), testLeasingProgram())

		require.Error(t, err)
	})
}

// TestQuoteService_Calculate_TakesPriceAndRateItself — главная проверка
// публичного расчёта: цену и курс сервер берёт сам.
//
// Приди цена снаружи, любой посетитель получил бы «официальный» график
// от произвольной суммы.
func TestQuoteService_Calculate_TakesPriceAndRateItself(t *testing.T) {
	t.Parallel()

	machine := testQuoteMachine()
	applied := entity.AppliedFXRate{
		Rate:      decimal.RequireFromString("500"),
		RateDate:  testQuoteNow.AddDate(0, 0, -1),
		MarkupBps: 200,
	}

	srv, _ := newQuoteService(t, machine, applied)

	quote, err := srv.Calculate(context.Background(), &QuoteCalculate{
		MachineSlug:    "xcmg-xe215c",
		ScheduleType:   entity.LeasingScheduleAnnuity,
		DownPaymentBps: 2000,
		TermMonths:     36,
		RateBps:        1800,
	}, testQuoteNow)

	require.NoError(t, err)

	assert.Equal(t, machine.PriceCNYCents, quote.PriceCNYCents, "цена взята по слагу")
	assert.Equal(t, applied.Rate, quote.FXRate, "курс взят сервисом курсов")
	assert.Equal(t, applied.RateDate, quote.FXRateDate)
	assert.Equal(t, applied.ConvertPriceCents(machine.PriceCNYCents), quote.AssetCostKZTCents)

	// Ничего не выпущено: ни ссылки, ни срока годности.
	assert.Empty(t, quote.ID)
	assert.Empty(t, quote.PublicToken, "расчёт не выпускает ссылку")
	assert.True(t, quote.ExpiresAt.IsZero(), "у расчёта нет срока годности")
	assert.Len(t, quote.PaymentSchedule, 36)
}

// TestQuoteService_Calculate_HiddenPriceRejected фиксирует отказ по машине
// со скрытой ценой: посчитать по ней график значило бы отдать цифру,
// которую форма «цена по запросу» и прячет.
func TestQuoteService_Calculate_HiddenPriceRejected(t *testing.T) {
	t.Parallel()

	machine := testQuoteMachine()
	machine.IsPricePublic = false

	srv, _ := newQuoteService(t, machine, entity.AppliedFXRate{
		Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow,
	})

	_, err := srv.Calculate(context.Background(), &QuoteCalculate{
		MachineSlug:  "xcmg-xe215c",
		ScheduleType: entity.LeasingScheduleAnnuity,
		TermMonths:   36,
	}, testQuoteNow)

	require.ErrorIs(t, err, entity.ErrMachinePriceHidden)
}

// TestQuoteService_Calculate_ChecksProgramBounds фиксирует, что публичный
// расчёт проверяет границы программы теми же кодами, что и выпуск.
func TestQuoteService_Calculate_ChecksProgramBounds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*QuoteCalculate)
		expErr error
	}{
		{
			name:   "взнос ниже минимального",
			mutate: func(c *QuoteCalculate) { c.DownPaymentBps = 500 },
			expErr: entity.ErrLeasingDownPaymentTooLow,
		},
		{
			name:   "срок вне диапазона",
			mutate: func(c *QuoteCalculate) { c.TermMonths = 120 },
			expErr: entity.ErrLeasingTermOutOfRange,
		},
		{
			name:   "ставка вне диапазона",
			mutate: func(c *QuoteCalculate) { c.RateBps = 100 },
			expErr: entity.ErrLeasingRateOutOfRange,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{
				Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow,
			})

			mocks.programs.EXPECT().GetByID(gomock.Any(), "program-1").Return(&entity.LeasingProgram{
				ID: "program-1", MinDownPaymentBps: 1500,
				TermMonthsMin: 12, TermMonthsMax: 60,
				RateMinBps: 1800, RateMaxBps: 2400, IsActive: true,
			}, nil)

			calc := &QuoteCalculate{
				MachineSlug:      "xcmg-xe215c",
				LeasingProgramID: "program-1",
				ScheduleType:     entity.LeasingScheduleAnnuity,
				DownPaymentBps:   2000,
				TermMonths:       36,
				RateBps:          1800,
			}
			c.mutate(calc)

			_, err := srv.Calculate(context.Background(), calc, testQuoteNow)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestQuoteService_Calculate_NoRateAborts фиксирует, что без курса расчёт
// отказывает, а не подставляет единицу.
func TestQuoteService_Calculate_NoRateAborts(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{})
	mocks.fx.err = entity.ErrFXRateNotFound

	_, err := srv.Calculate(context.Background(), &QuoteCalculate{
		MachineSlug:  "xcmg-xe215c",
		ScheduleType: entity.LeasingScheduleAnnuity,
		TermMonths:   36,
	}, testQuoteNow)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
}

// TestQuoteService_Calculate_Nil фиксирует отказ на пустых параметрах.
func TestQuoteService_Calculate_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newQuoteService(t, nil, entity.AppliedFXRate{})

	_, err := srv.Calculate(context.Background(), nil, testQuoteNow)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestQuoteService_Calculate_DoesNotFixRate фиксирует, что публичный расчёт
// читает курс, но НЕ фиксирует его: фиксация — операция выпуска, после которой
// цифра перестаёт меняться, а справочный расчёт обязан следовать за курсом.
func TestQuoteService_Calculate_DoesNotFixRate(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{
		Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow,
	})

	_, err := srv.Calculate(context.Background(), &QuoteCalculate{
		MachineSlug:  "xcmg-xe215c",
		ScheduleType: entity.LeasingScheduleAnnuity,
		TermMonths:   36,
	}, testQuoteNow)

	require.NoError(t, err)
	assert.Equal(t, 1, mocks.fx.reads, "курс прочитан один раз")
	assert.Zero(t, mocks.fx.calls, "FixRateForQuote не вызывается: это операция выпуска")
}

// TestQuoteService_Calculate_WritesNothing фиксирует, что расчёт ничего
// не сохраняет: ни предложения, ни чего-либо ещё.
func TestQuoteService_Calculate_WritesNothing(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{
		Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow,
	})

	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.Calculate(context.Background(), &QuoteCalculate{
		MachineSlug:  "xcmg-xe215c",
		ScheduleType: entity.LeasingScheduleAnnuity,
		TermMonths:   36,
	}, testQuoteNow)

	require.NoError(t, err)
}

// TestQuoteService_Calculate_MatchesIssue фиксирует, что справочный расчёт
// и выпуск по тем же параметрам и тому же курсу дают одинаковые цифры.
//
// Иначе клиент увидел бы на калькуляторе один платёж, а в присланном
// предложении другой.
func TestQuoteService_Calculate_MatchesIssue(t *testing.T) {
	t.Parallel()

	applied := entity.AppliedFXRate{
		Rate: decimal.RequireFromString("522.34"), RateDate: testQuoteNow,
	}

	params := func() *QuoteCalculate {
		return &QuoteCalculate{
			MachineSlug:    "xcmg-xe215c",
			ScheduleType:   entity.LeasingScheduleAnnuity,
			DownPaymentBps: 2000,
			TermMonths:     36,
			RateBps:        1800,
		}
	}

	calcSrv, _ := newQuoteService(t, testQuoteMachine(), applied)

	calculated, err := calcSrv.Calculate(context.Background(), params(), testQuoteNow)
	require.NoError(t, err)

	issueSrv, issueMocks := newQuoteService(t, testQuoteMachine(), applied)
	issueMocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		})

	p := params()
	issued, err := issueSrv.Issue(context.Background(), &QuoteIssue{
		MachineSlug:    p.MachineSlug,
		ScheduleType:   p.ScheduleType,
		DownPaymentBps: p.DownPaymentBps,
		TermMonths:     p.TermMonths,
		RateBps:        p.RateBps,
	}, testQuoteNow)
	require.NoError(t, err)

	assert.Equal(t, issued.AssetCostKZTCents, calculated.AssetCostKZTCents)
	assert.Equal(t, issued.FinancedAmountKZTCents, calculated.FinancedAmountKZTCents)
	assert.Equal(t, issued.MonthlyPaymentKZTCents, calculated.MonthlyPaymentKZTCents)
	assert.Equal(t, issued.LastPaymentKZTCents, calculated.LastPaymentKZTCents)
	assert.Equal(t, issued.TotalPaidKZTCents, calculated.TotalPaidKZTCents)
	assert.Equal(t, issued.PaymentSchedule, calculated.PaymentSchedule)
}

// TestQuoteService_Issue_CarriesProgramName фиксирует, что название программы
// уходит вместе с выпущенным предложением.
//
// Проставляется из уже прочитанной программы, а не перечитывается: второй
// поход в базу ради строки, которая лежит рядом, был бы лишним.
func TestQuoteService_Issue_CarriesProgramName(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{
		Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow,
	})

	mocks.programs.EXPECT().GetByID(gomock.Any(), "program-1").Return(&entity.LeasingProgram{
		ID: "program-1", NameRU: "Halyk Leasing", MinDownPaymentBps: 1500,
		TermMonthsMin: 12, TermMonthsMax: 60,
		RateMinBps: 1800, RateMaxBps: 2400, IsActive: true,
	}, nil)
	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, q *entity.Quote) (*entity.Quote, error) {
			return q, nil
		})

	quote, err := srv.Issue(context.Background(), &QuoteIssue{
		MachineSlug:      "xcmg-xe215c",
		LeasingProgramID: "program-1",
		ScheduleType:     entity.LeasingScheduleAnnuity,
		DownPaymentBps:   2000,
		TermMonths:       36,
		RateBps:          1800,
	}, testQuoteNow)

	require.NoError(t, err)
	require.NotNil(t, quote.LeasingProgramName)
	assert.Equal(t, "Halyk Leasing", *quote.LeasingProgramName)
}

// TestQuoteService_Calculate_CarriesProgramName фиксирует то же для публичного
// расчёта: клиент видит, по какой программе посчитано.
func TestQuoteService_Calculate_CarriesProgramName(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), entity.AppliedFXRate{
		Rate: decimal.RequireFromString("500"), RateDate: testQuoteNow,
	})

	mocks.programs.EXPECT().GetByID(gomock.Any(), "program-1").Return(&entity.LeasingProgram{
		ID: "program-1", NameRU: "Halyk Leasing", MinDownPaymentBps: 1500,
		TermMonthsMin: 12, TermMonthsMax: 60,
		RateMinBps: 1800, RateMaxBps: 2400, IsActive: true,
	}, nil)

	quote, err := srv.Calculate(context.Background(), &QuoteCalculate{
		MachineSlug:      "xcmg-xe215c",
		LeasingProgramID: "program-1",
		ScheduleType:     entity.LeasingScheduleAnnuity,
		DownPaymentBps:   2000,
		TermMonths:       36,
		RateBps:          1800,
	}, testQuoteNow)

	require.NoError(t, err)
	require.NotNil(t, quote.LeasingProgramName)
	assert.Equal(t, "Halyk Leasing", *quote.LeasingProgramName)
}

// TestQuoteService_RecalculateByToken фиксирует главное правило пересчёта:
// зафиксированное предложение не меняется, а курс берётся его собственный.
//
// Проверяется не только результат, но и то, что курс НЕ читался заново:
// пересчёт по текущему курсу менял бы вместе со сроком ещё и стоимость
// машины, и клиент прочитал бы это как «предложение изменилось».
func TestQuoteService_RecalculateByToken(t *testing.T) {
	t.Parallel()

	now := testQuoteNow
	stored := testStoredQuote(now)

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))
	mocks.quotes.EXPECT().GetByToken(gomock.Any(), stored.PublicToken).Return(stored, nil)
	// Ни сохранения, ни отметки открытия: пересчёт ничего не меняет.
	mocks.quotes.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
	mocks.quotes.EXPECT().MarkOpened(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	recalculated, err := srv.RecalculateByToken(context.Background(), stored.PublicToken,
		QuoteRecalculate{
			ScheduleType:   entity.LeasingScheduleAnnuity,
			DownPaymentBps: 3000,
			TermMonths:     48,
			RateBps:        1800,
		})
	require.NoError(t, err)

	assert.Zero(t, mocks.fx.calls, "курс не фиксируется заново")
	assert.Zero(t, mocks.fx.reads, "текущий курс не читается: считаем по зафиксированному")

	assert.Empty(t, recalculated.ID, "пересчёт не является сохранённым предложением")
	assert.Empty(t, recalculated.PublicToken, "у пересчёта нет своей ссылки")

	// Курс, цена и стоимость в тенге перенесены из предложения без изменений.
	assert.True(t, stored.FXRate.Equal(recalculated.FXRate), "курс тот же")
	assert.Equal(t, stored.FXRateDate, recalculated.FXRateDate)
	assert.Equal(t, stored.AssetCostKZTCents, recalculated.AssetCostKZTCents)
	assert.Equal(t, stored.PriceCNYCents, recalculated.PriceCNYCents)

	// Новые условия применились, и график пересчитан под них.
	assert.Equal(t, 48, recalculated.TermMonths)
	assert.Equal(t, 3000, recalculated.DownPaymentBps)
	assert.Len(t, recalculated.PaymentSchedule, 48)
	assert.NotEqual(t, stored.MonthlyPaymentKZTCents, recalculated.MonthlyPaymentKZTCents)

	// Само предложение осталось нетронутым.
	assert.Equal(t, 36, stored.TermMonths, "исходное предложение не изменилось")
}

// TestQuoteService_RecalculateByToken_ShortToken фиксирует, что короткий токен
// отвергается до похода в базу: он не мог быть выдан, и запрос по нему — перебор.
func TestQuoteService_RecalculateByToken_ShortToken(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))
	mocks.quotes.EXPECT().GetByToken(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.RecalculateByToken(context.Background(), "short", QuoteRecalculate{
		ScheduleType: entity.LeasingScheduleAnnuity, TermMonths: 12,
	})

	require.ErrorIs(t, err, entity.ErrQuoteNotFound)
}

// TestQuoteService_RecalculateByToken_NotFound фиксирует, что отказ хранилища
// доходит наружу.
func TestQuoteService_RecalculateByToken_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))
	mocks.quotes.EXPECT().GetByToken(gomock.Any(), gomock.Any()).
		Return(nil, entity.ErrQuoteNotFound)

	_, err := srv.RecalculateByToken(context.Background(), testStoredQuote(testQuoteNow).PublicToken,
		QuoteRecalculate{ScheduleType: entity.LeasingScheduleAnnuity, TermMonths: 12})

	require.ErrorIs(t, err, entity.ErrQuoteNotFound)
}

// TestQuoteService_RecalculateByToken_ProgramBounds фиксирует, что границы
// программы проверяются те же, что и при выпуске: пересчёт, который клиент
// не сможет получить предложением, ему не нужен.
func TestQuoteService_RecalculateByToken_ProgramBounds(t *testing.T) {
	t.Parallel()

	stored := testStoredQuote(testQuoteNow)
	stored.LeasingProgramID = entity.Pointer(testProgramID)

	srv, mocks := newQuoteService(t, testQuoteMachine(), testAppliedRate(t, "500"))
	mocks.quotes.EXPECT().GetByToken(gomock.Any(), stored.PublicToken).Return(stored, nil)
	mocks.programs.EXPECT().GetByID(gomock.Any(), testProgramID).Return(&entity.LeasingProgram{
		ID: testProgramID, NameRU: "Стандарт", IsActive: true,
		TermMonthsMin: 12, TermMonthsMax: 36,
		RateMinBps: 1500, RateMaxBps: 2200, MinDownPaymentBps: 2000,
	}, nil)

	// 48 месяцев выходят за верхнюю границу программы.
	_, err := srv.RecalculateByToken(context.Background(), stored.PublicToken,
		QuoteRecalculate{
			ScheduleType:   entity.LeasingScheduleAnnuity,
			DownPaymentBps: 2000,
			TermMonths:     48,
			RateBps:        1800,
		})

	require.Error(t, err)
}

// testStoredQuote собирает сохранённое предложение: 36 месяцев, курс 500.
func testStoredQuote(now time.Time) *entity.Quote {
	return &entity.Quote{
		ID:          "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
		PublicToken: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		MachineID:   testMachineID,

		PriceCNYCents: 10_000_000,
		FXRate:        decimal.RequireFromString("500"),
		FXRateDate:    now,
		FXMarkupBps:   200,

		AssetCostKZTCents: 50_000_000_00,

		ScheduleType:   entity.LeasingScheduleAnnuity,
		DownPaymentBps: 2000,
		TermMonths:     36,
		RateBps:        1800,

		MonthlyPaymentKZTCents: 1_000_000_00,

		ExpiresAt: now.Add(14 * 24 * time.Hour),
		CreatedAt: now,
	}
}

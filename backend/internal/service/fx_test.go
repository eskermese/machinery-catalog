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

const (
	testBase   = "USD"
	testQuote  = "KZT"
	testSource = "nationalbank"
)

// testFXNow момент расчёта в тестах курса.
var testFXNow = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

// testFXRateDate дата курса в тестах.
var testFXRateDate = time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)

// fxMocks набор моков для сервиса курсов.
type fxMocks struct {
	rateRepo   *mock_repo.MockFXRateRepository
	markupRepo *mock_repo.MockFXMarkupRepository
	cache      *mock_repo.MockFXCache
}

// newFXService собирает сервис курсов на моках.
// testPivotCurrency валюта-посредник в тестах: та же, что в бою, — тенге.
const testPivotCurrency = "KZT"

// testStaleAfterDays порог устаревания курса в тестах: как в бою.
const testStaleAfterDays = 3

func newFXService(t *testing.T, provider RateProvider) (FXService, fxMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mocks := fxMocks{
		rateRepo:   mock_repo.NewMockFXRateRepository(ctrl),
		markupRepo: mock_repo.NewMockFXMarkupRepository(ctrl),
		cache:      mock_repo.NewMockFXCache(ctrl),
	}

	return NewFXService(mocks.rateRepo, mocks.markupRepo, mocks.cache, provider, testPivotCurrency, testStaleAfterDays), mocks
}

// testRate собирает курс источника с заданной датой.
func testRate(t *testing.T, rate string, rateDate time.Time) *entity.FXRate {
	t.Helper()

	value, err := decimal.NewFromString(rate)
	require.NoError(t, err)

	return entity.NewFXRate(testBase, testQuote, testSource, value, rateDate, rateDate)
}

// TestFXService_AppliedRate_MarkupApplied фиксирует применение наценки при
// чтении из БД: нулевая наценка отдаёт курс источника, ненулевая — накручивает.
func TestFXService_AppliedRate_MarkupApplied(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		markup    *entity.FXMarkup
		expRate   string
		expMarkup int
	}{
		{
			// Наценка не задана вовсе — продаём по курсу источника.
			name: "наценки нет", markup: nil,
			expRate: "522.340000", expMarkup: 0,
		},
		{
			name:      "наценка ноль",
			markup:    entity.NewFXMarkup(testBase, testQuote, 0, testFXNow, nil, testFXNow),
			expRate:   "522.340000",
			expMarkup: 0,
		},
		{
			// 522.34 * 1.02 = 532.7868
			name:      "наценка 200 bps",
			markup:    entity.NewFXMarkup(testBase, testQuote, 200, testFXNow, nil, testFXNow),
			expRate:   "532.786800",
			expMarkup: 200,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newFXService(t, nil)

			mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
				Return(entity.AppliedFXRate{}, false)
			mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
				Return(testRate(t, "522.34", testFXRateDate), nil)
			mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), testBase, testQuote, testFXNow).
				Return(c.markup, nil)
			mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), testBase, testQuote, gomock.Any())

			applied, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)
			require.NoError(t, err)

			assert.Equal(t, c.expRate, applied.String())
			assert.Equal(t, c.expMarkup, applied.MarkupBps)
			assert.Equal(t, testFXRateDate, applied.RateDate, "дата курса уходит наружу вместе с курсом")
		})
	}
}

// TestFXService_AppliedRate_CacheHit фиксирует, что попадание в кэш не идёт в БД.
func TestFXService_AppliedRate_CacheHit(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	cached := entity.NewAppliedFXRate(testRate(t, "522.34", testFXRateDate), 200)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).Return(cached, true)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	applied, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)
	require.NoError(t, err)

	assert.Equal(t, "532.786800", applied.String())
}

// TestFXService_AppliedRate_CacheFailureDoesNotBreakRead фиксирует главное
// свойство кэша: он ускоритель, а не источник правды. Отказ кэша на чтении
// и на записи одинаково приводит к чтению из БД и не мешает вернуть курс.
func TestFXService_AppliedRate_CacheFailureDoesNotBreakRead(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	// Кэш отвечает промахом — адаптер гасит собственную ошибку внутри себя
	// и наружу отдаёт ровно то же, что при обычном промахе.
	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
		Return(testRate(t, "522.34", testFXRateDate), nil)
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), testBase, testQuote, testFXNow).
		Return(nil, nil)
	// Запись в кэш тоже не удалась — метод ничего не возвращает, отказать некому.
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), testBase, testQuote, gomock.Any())

	applied, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)

	require.NoError(t, err, "отказ кэша не должен ломать чтение курса")
	assert.Equal(t, "522.340000", applied.String())
}

// TestFXService_AppliedRate_WorksWithoutCache фиксирует, что сервис собирается
// и работает вообще без кэша.
func TestFXService_AppliedRate_WorksWithoutCache(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	rateRepo := mock_repo.NewMockFXRateRepository(ctrl)
	markupRepo := mock_repo.NewMockFXMarkupRepository(ctrl)

	rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
		Return(testRate(t, "522.34", testFXRateDate), nil)
	markupRepo.EXPECT().GetEffective(gomock.Any(), testBase, testQuote, testFXNow).Return(nil, nil)

	srv := NewFXService(rateRepo, markupRepo, nil, nil, testPivotCurrency, testStaleAfterDays)

	applied, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)
	require.NoError(t, err)

	assert.Equal(t, "522.340000", applied.String())
}

// TestFXService_AppliedRate_StaleRateIsReturned фиксирует поведение при
// отсутствии курса за сегодня: отдаётся последний известный вместе с его датой.
//
// Подставлять единицу нельзя — это тихо продаст технику по цене в долларах
// как за тенге. Падать тоже нельзя — вчерашний курс вполне пригоден для показа
// цены. Поэтому дата возвращается наружу: решение о предупреждении принимает
// потребитель, а не сервис.
func TestFXService_AppliedRate_StaleRateIsReturned(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	staleDate := testFXRateDate.AddDate(0, 0, -3)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
		Return(testRate(t, "500.00", staleDate), nil)
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), testBase, testQuote, testFXNow).Return(nil, nil)
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), testBase, testQuote, gomock.Any())

	applied, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)

	require.NoError(t, err, "несвежий курс — не ошибка")
	assert.Equal(t, "500.000000", applied.String())
	assert.Equal(t, staleDate, applied.RateDate, "дата позволяет потребителю решить, предупреждать ли о несвежести")
}

// TestFXService_AppliedRate_NoRateAtAll фиксирует, что полное отсутствие курса —
// это отказ, а не подстановка единицы.
func TestFXService_AppliedRate_NoRateAtAll(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
		Return(nil, entity.ErrFXRateNotFound)

	applied, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
	assert.True(t, applied.IsZero())
}

// TestFXService_FixRateForQuote фиксирует, что для КП отдаются все три
// величины сразу — курс, его дата и наценка.
func TestFXService_FixRateForQuote(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
		Return(testRate(t, "522.34", testFXRateDate), nil)
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), testBase, testQuote, testFXNow).
		Return(entity.NewFXMarkup(testBase, testQuote, 200, testFXNow, nil, testFXNow), nil)
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), testBase, testQuote, gomock.Any())

	fixed, err := srv.FixRateForQuote(context.Background(), testBase, testQuote, testFXNow)
	require.NoError(t, err)

	// Ровно то, что уходит в quotes.fx_rate, fx_rate_date и fx_markup_bps.
	assert.Equal(t, "532.786800", fixed.String())
	assert.Equal(t, testFXRateDate, fixed.RateDate)
	assert.Equal(t, 200, fixed.MarkupBps)
}

// TestFXService_SaveRate_InvalidatesCache фиксирует сброс кэша после загрузки курса.
func TestFXService_SaveRate_InvalidatesCache(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	rate := testRate(t, "522.34", testFXRateDate)

	mocks.rateRepo.EXPECT().Create(gomock.Any(), rate).Return(rate, nil)
	mocks.cache.EXPECT().DeleteAppliedRate(gomock.Any(), testBase, testQuote)

	saved, err := srv.SaveRate(context.Background(), rate)
	require.NoError(t, err)

	assert.Equal(t, rate, saved)
}

// TestFXService_SaveRate_Nil фиксирует отказ на пустом курсе.
func TestFXService_SaveRate_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newFXService(t, nil)

	_, err := srv.SaveRate(context.Background(), nil)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// TestFXService_SetMarkup_InvalidatesCache фиксирует сброс кэша после смены наценки.
// Без сброса менеджер, поменявший наценку, до конца TTL видел бы старую цену.
func TestFXService_SetMarkup_InvalidatesCache(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	markup := entity.NewFXMarkup(testBase, testQuote, 250, testFXNow, nil, testFXNow)

	mocks.markupRepo.EXPECT().Create(gomock.Any(), markup).Return(markup, nil)
	mocks.cache.EXPECT().DeleteAppliedRate(gomock.Any(), testBase, testQuote)

	saved, err := srv.SetMarkup(context.Background(), markup)
	require.NoError(t, err)

	assert.Equal(t, markup, saved)
}

// TestFXService_SetMarkup_Nil фиксирует отказ на пустой наценке.
func TestFXService_SetMarkup_Nil(t *testing.T) {
	t.Parallel()

	srv, _ := newFXService(t, nil)

	_, err := srv.SetMarkup(context.Background(), nil)

	require.ErrorIs(t, err, entity.ErrNilPointer)
}

// stubProvider источник курса для тестов синхронизации.
type stubProvider struct {
	rate *entity.FXRate
	err  error
}

// FetchRate возвращает заданный курс или ошибку.
func (p stubProvider) FetchRate(_ context.Context, _, _ string, _ time.Time) (*entity.FXRate, error) {
	return p.rate, p.err
}

// TestFXService_SyncRate фиксирует, что синхронизация сохраняет курс и сбрасывает кэш.
func TestFXService_SyncRate(t *testing.T) {
	t.Parallel()

	rate := testRate(t, "522.34", testFXRateDate)
	srv, mocks := newFXService(t, stubProvider{rate: rate})

	mocks.rateRepo.EXPECT().Create(gomock.Any(), rate).Return(rate, nil)
	mocks.cache.EXPECT().DeleteAppliedRate(gomock.Any(), testBase, testQuote)

	require.NoError(t, srv.SyncRate(context.Background(), testBase, testQuote, testFXNow))
}

// TestFXService_SyncRate_DuplicateIsNotAnError фиксирует, что повторная
// загрузка курса того же дня от того же источника не создаёт вторую строку
// и не считается сбоем синхронизации: расписание могло сработать дважды.
func TestFXService_SyncRate_DuplicateIsNotAnError(t *testing.T) {
	t.Parallel()

	rate := testRate(t, "522.34", testFXRateDate)
	srv, mocks := newFXService(t, stubProvider{rate: rate})

	// Репозиторий отвергает вторую строку конфликтом по UNIQUE.
	mocks.rateRepo.EXPECT().Create(gomock.Any(), rate).Return(nil, entity.ErrFXRateAlreadyExists)

	err := srv.SyncRate(context.Background(), testBase, testQuote, testFXNow)

	require.NoError(t, err, "повторная загрузка того же дня — штатный повтор расписания")
}

// TestFXService_SyncRate_ProviderNotConfigured фиксирует отказ заглушки источника.
func TestFXService_SyncRate_ProviderNotConfigured(t *testing.T) {
	t.Parallel()

	srv, _ := newFXService(t, NewNotConfiguredRateProvider())

	err := srv.SyncRate(context.Background(), testBase, testQuote, testFXNow)

	require.ErrorIs(t, err, entity.ErrFXProviderNotConfigured)
}

// TestFXService_SyncRate_NoProvider фиксирует отказ при вовсе не заданном источнике.
func TestFXService_SyncRate_NoProvider(t *testing.T) {
	t.Parallel()

	srv, _ := newFXService(t, nil)

	err := srv.SyncRate(context.Background(), testBase, testQuote, testFXNow)

	require.ErrorIs(t, err, entity.ErrFXProviderNotConfigured)
}

// TestFXService_SyncRate_ProviderFailure фиксирует, что отказ источника
// поднимается наверх: воркер решит, что с ним делать.
func TestFXService_SyncRate_ProviderFailure(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("источник недоступен")
	srv, _ := newFXService(t, stubProvider{err: providerErr})

	err := srv.SyncRate(context.Background(), testBase, testQuote, testFXNow)

	require.ErrorIs(t, err, providerErr)
}

// TestFXService_SyncRate_SaveFailure фиксирует, что ошибка сохранения,
// не являющаяся конфликтом, поднимается наверх.
func TestFXService_SyncRate_SaveFailure(t *testing.T) {
	t.Parallel()

	rate := testRate(t, "522.34", testFXRateDate)
	srv, mocks := newFXService(t, stubProvider{rate: rate})

	saveErr := errors.New("база недоступна")
	mocks.rateRepo.EXPECT().Create(gomock.Any(), rate).Return(nil, saveErr)

	err := srv.SyncRate(context.Background(), testBase, testQuote, testFXNow)

	require.ErrorIs(t, err, saveErr)
}

// TestFXService_AppliedRate_MarkupFailure фиксирует, что отказ чтения наценки
// не подменяется нулём: наценка «не задана» и наценка «не прочиталась» — разные
// состояния, и второе обязано быть отказом.
func TestFXService_AppliedRate_MarkupFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)

	markupErr := errors.New("база недоступна")

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).
		Return(testRate(t, "522.34", testFXRateDate), nil)
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), testBase, testQuote, testFXNow).
		Return(nil, markupErr)

	_, err := srv.AppliedRate(context.Background(), testBase, testQuote, testFXNow)

	require.ErrorIs(t, err, markupErr)
}

// TestFXService_AppliedRate_CrossRate фиксирует вывод пары, которой нет
// у источника напрямую.
//
// Нацбанк публикует курсы к тенге: CNY→KZT и USD→KZT есть, CNY→USD нет.
// Пара выводится делением, и это единственный способ показать цену
// в долларах, не заставляя человека считать деление руками каждый день.
func TestFXService_AppliedRate_CrossRate(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)
	now := time.Now().UTC()
	rateDate := now.AddDate(0, 0, -1)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "USD").Return(entity.AppliedFXRate{}, false)
	// Прямого курса пары нет.
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").Return(nil, entity.ErrFXRateNotFound)
	// Зато есть обе стороны через посредника: 72.5 / 500 = 0.145.
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").
		Return(testRate(t, "72.5", rateDate), nil)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "USD", "KZT").
		Return(testRate(t, "500", now), nil)
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), "CNY", "USD", gomock.Any()).Return(nil, nil)
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), "CNY", "USD", gomock.Any())

	applied, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

	require.NoError(t, err)
	assert.Equal(t, "0.145000", applied.String())
	// Дата — более ранняя из двух: курс не свежее самой старой из величин,
	// из которых он получен.
	assert.Equal(t, rateDate.Format(time.DateOnly), applied.RateDate.Format(time.DateOnly))
}

// TestFXService_AppliedRate_CrossRateMarkupOnce — наценка на выведенную пару
// накручивается РОВНО ОДИН РАЗ.
//
// Тест существует ради ошибки, которую легко сделать и невозможно заметить
// глазами: если делить курсы С НАЦЕНКАМИ, множители сокращаются —
// (raw1 * 1.02) / (raw2 * 1.02) равно raw1/raw2, — и наценки не остаётся
// вовсе. При разных наценках получилась бы разность, которую никто
// не назначал. Поэтому делятся ЧИСТЫЕ курсы, а наценка берётся из строки
// самой пары CNY→USD.
//
// Здесь у пар-слагаемых наценки заведомо разные и крупные: сократись они
// или примени́сь дважды — результат отличался бы от ожидаемого заметно.
func TestFXService_AppliedRate_CrossRateMarkupOnce(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)
	now := time.Now().UTC()

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "USD").Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").Return(nil, entity.ErrFXRateNotFound)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").Return(testRate(t, "72.5", now), nil)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "USD", "KZT").Return(testRate(t, "500", now), nil)

	// Наценка пары CNY→USD: 10 %.
	mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), "CNY", "USD", gomock.Any()).
		Return(&entity.FXMarkup{MarkupBps: 1000}, nil)
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), "CNY", "USD", gomock.Any())

	applied, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

	require.NoError(t, err)
	// 0.145 * 1.10 = 0.1595. Ни 0.145 (наценка потерялась),
	// ни 0.17545 (наценка применена дважды).
	assert.Equal(t, "0.159500", applied.String())
}

// TestFXService_AppliedRate_CrossRateUnavailable — не хватает одной стороны,
// и пара не выводится.
//
// Отказ, а не выдуманный курс: подставить сюда что угодно значило бы
// показать цену в долларах, взятую из воздуха.
func TestFXService_AppliedRate_CrossRateUnavailable(t *testing.T) {
	t.Parallel()

	srv, mocks := newFXService(t, nil)
	now := time.Now().UTC()

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "USD").Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").Return(nil, entity.ErrFXRateNotFound)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").Return(testRate(t, "72.5", now), nil)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "USD", "KZT").Return(nil, entity.ErrFXRateNotFound)

	_, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

	require.ErrorIs(t, err, entity.ErrFXRateNotFound)
}

// TestFXService_AppliedRate_CrossRateGuards — случаи, в которых выводить
// нечего, и сервис не пытается.
//
// Посредник не может быть стороной собственной пары: деление свелось бы
// к «курс KZT→KZT», то есть к единице, а единица в курсе — это молчаливая
// продажа юаней по цене тенге. Нулевой курс делителя — то же самое с другой
// стороны: деление на ноль либо паника, либо бесконечность, и обе хуже отказа.
func TestFXService_AppliedRate_CrossRateGuards(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("посредник — сторона пары", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newFXService(t, nil)
		mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "KZT").
			Return(entity.AppliedFXRate{}, false)
		// Прямого курса нет, и выводить его через тот же KZT нельзя:
		// второй запрос к хранилищу не делается вовсе.
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").
			Return(nil, entity.ErrFXRateNotFound)

		_, err := srv.AppliedRate(context.Background(), "CNY", "KZT", now)

		require.ErrorIs(t, err, entity.ErrFXRateNotFound)
	})

	t.Run("посредник не настроен", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		rateRepo := mock_repo.NewMockFXRateRepository(ctrl)
		// Пустой посредник — вывод невозможен, и второго запроса нет.
		srv := NewFXService(rateRepo, mock_repo.NewMockFXMarkupRepository(ctrl), nil, nil, "", testStaleAfterDays)

		rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").
			Return(nil, entity.ErrFXRateNotFound)

		_, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

		require.ErrorIs(t, err, entity.ErrFXRateNotFound)
	})

	t.Run("курс делителя нулевой", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newFXService(t, nil)
		mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "USD").
			Return(entity.AppliedFXRate{}, false)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").
			Return(nil, entity.ErrFXRateNotFound)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").
			Return(testRate(t, "72.5", now), nil)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "USD", "KZT").
			Return(testRate(t, "0", now), nil)

		_, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

		require.ErrorIs(t, err, entity.ErrFXRateNotFound)
	})
}

// TestFXService_AppliedRate_CrossRateDateAndBase закрывает две оставшиеся
// ветки вывода: отсутствие первой стороны и выбор более ранней даты,
// когда старее оказалась вторая.
func TestFXService_AppliedRate_CrossRateDateAndBase(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("нет первой стороны", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newFXService(t, nil)
		mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "USD").
			Return(entity.AppliedFXRate{}, false)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").
			Return(nil, entity.ErrFXRateNotFound)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").
			Return(nil, entity.ErrFXRateNotFound)

		_, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

		require.ErrorIs(t, err, entity.ErrFXRateNotFound)
	})

	t.Run("вторая сторона старее", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newFXService(t, nil)
		older := now.AddDate(0, 0, -3)

		mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), "CNY", "USD").
			Return(entity.AppliedFXRate{}, false)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "USD").
			Return(nil, entity.ErrFXRateNotFound)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "CNY", "KZT").
			Return(testRate(t, "72.5", now), nil)
		mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), "USD", "KZT").
			Return(testRate(t, "500", older), nil)
		mocks.markupRepo.EXPECT().GetEffective(gomock.Any(), "CNY", "USD", gomock.Any()).
			Return(nil, nil)
		mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), "CNY", "USD", gomock.Any())

		applied, err := srv.AppliedRate(context.Background(), "CNY", "USD", now)

		require.NoError(t, err)
		// Дата берётся у той стороны, что старее: курс не может считаться
		// свежее самой старой из величин, из которых он получен.
		assert.Equal(t, older.Format(time.DateOnly), applied.RateDate.Format(time.DateOnly))
	})
}

// TestFXService_PairStatus_ShowsBothSources фиксирует главное требование
// ручки состояния: рядом с действующим курсом видно И курс внешнего
// источника, И ручной ввод.
//
// Без этого «действующий курс» не с чем сравнить, и администратор не видит,
// поправил он что-то или согласился с Нацбанком.
func TestFXService_PairStatus_ShowsBothSources(t *testing.T) {
	t.Parallel()

	svc, mocks := newFXService(t, nil)
	ctx := t.Context()

	manual := entity.NewFXRate(
		testBase, testQuote, entity.FXSourceManual,
		decimal.RequireFromString("500"), testFXNow, testFXNow,
	)
	provider := testRate(t, "480", testFXNow)

	// Действует ручной: приоритет источников разрешается в GetLatest.
	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).Return(manual, nil)
	mocks.markupRepo.EXPECT().
		GetEffective(gomock.Any(), testBase, testQuote, gomock.Any()).
		Return(entity.NewFXMarkup(testBase, testQuote, 200, testFXNow, nil, testFXNow), nil).
		Times(2)
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), testBase, testQuote, gomock.Any())

	mocks.rateRepo.EXPECT().
		GetLatestBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
		Return(manual, nil)
	mocks.rateRepo.EXPECT().
		GetLatestExceptSource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
		Return(provider, nil)

	status, err := svc.PairStatus(ctx, testBase, testQuote, testFXNow)
	require.NoError(t, err)

	require.True(t, status.HasRate())
	assert.Equal(t, entity.FXSourceManual, status.EffectiveSource, "действует ручной")
	assert.Equal(t, "510.000000", status.Effective.String(), "500 + 2 % = 510")
	assert.Equal(t, 200, status.MarkupBps)
	assert.False(t, status.Stale, "курс за сегодня не устарел")

	require.NotNil(t, status.Manual)
	assert.Equal(t, "500", status.Manual.Rate.String())

	require.NotNil(t, status.Provider,
		"курс Нацбанка обязан быть виден даже когда действует ручной")
	assert.Equal(t, "480", status.Provider.Rate.String())
}

// TestFXService_PairStatus_WithoutRate фиксирует, что отсутствие курса —
// штатное состояние ручки, а не её отказ: именно его админка и показывает.
func TestFXService_PairStatus_WithoutRate(t *testing.T) {
	t.Parallel()

	svc, mocks := newFXService(t, nil)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().
		GetLatest(gomock.Any(), testBase, testQuote).
		Return(nil, entity.ErrFXRateNotFound)
	mocks.rateRepo.EXPECT().
		GetLatestBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
		Return(nil, entity.ErrFXRateNotFound)
	mocks.rateRepo.EXPECT().
		GetLatestExceptSource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
		Return(nil, entity.ErrFXRateNotFound)
	mocks.markupRepo.EXPECT().
		GetEffective(gomock.Any(), testBase, testQuote, gomock.Any()).
		Return(nil, nil)

	status, err := svc.PairStatus(t.Context(), testBase, testQuote, testFXNow)
	require.NoError(t, err)

	assert.False(t, status.HasRate())
	assert.Nil(t, status.Manual)
	assert.Nil(t, status.Provider)
	assert.Equal(t, 0, status.MarkupBps)
}

// TestFXService_PairStatus_StaleRate фиксирует признак устаревания: курс есть,
// но он старше допустимого возраста, и админка обязана это показать.
func TestFXService_PairStatus_StaleRate(t *testing.T) {
	t.Parallel()

	svc, mocks := newFXService(t, nil)

	old := testFXNow.AddDate(0, 0, -(testStaleAfterDays + 1))
	rate := testRate(t, "480", old)

	mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
		Return(entity.AppliedFXRate{}, false)
	mocks.rateRepo.EXPECT().GetLatest(gomock.Any(), testBase, testQuote).Return(rate, nil)
	mocks.markupRepo.EXPECT().
		GetEffective(gomock.Any(), testBase, testQuote, gomock.Any()).
		Return(nil, nil).
		Times(2)
	mocks.cache.EXPECT().SetAppliedRate(gomock.Any(), testBase, testQuote, gomock.Any())
	mocks.rateRepo.EXPECT().
		GetLatestBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
		Return(nil, entity.ErrFXRateNotFound)
	mocks.rateRepo.EXPECT().
		GetLatestExceptSource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
		Return(rate, nil)

	status, err := svc.PairStatus(t.Context(), testBase, testQuote, testFXNow)
	require.NoError(t, err)

	require.True(t, status.HasRate(), "устаревший курс всё ещё действует")
	assert.True(t, status.Stale, "и обязан быть помечен устаревшим")
}

// TestFXService_PairStatus_Errors фиксирует, что отказ хранилища не выдаётся
// за «курса нет»: неотличимые снаружи, эти состояния означают разное.
func TestFXService_PairStatus_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		setup func(mocks fxMocks)
	}{
		{
			name: "отказ на действующем курсе",
			setup: func(mocks fxMocks) {
				mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
					Return(entity.AppliedFXRate{}, false)
				mocks.rateRepo.EXPECT().
					GetLatest(gomock.Any(), testBase, testQuote).
					Return(nil, entity.ErrDatabaseCommon)
			},
		},
		{
			name: "отказ на ручном курсе",
			setup: func(mocks fxMocks) {
				mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
					Return(entity.AppliedFXRate{}, false)
				mocks.rateRepo.EXPECT().
					GetLatest(gomock.Any(), testBase, testQuote).
					Return(nil, entity.ErrFXRateNotFound)
				mocks.rateRepo.EXPECT().
					GetLatestBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
					Return(nil, entity.ErrDatabaseCommon)
			},
		},
		{
			name: "отказ на курсе источника",
			setup: func(mocks fxMocks) {
				mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
					Return(entity.AppliedFXRate{}, false)
				mocks.rateRepo.EXPECT().
					GetLatest(gomock.Any(), testBase, testQuote).
					Return(nil, entity.ErrFXRateNotFound)
				mocks.rateRepo.EXPECT().
					GetLatestBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
					Return(nil, entity.ErrFXRateNotFound)
				mocks.rateRepo.EXPECT().
					GetLatestExceptSource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
					Return(nil, entity.ErrDatabaseCommon)
			},
		},
		{
			name: "отказ на наценке",
			setup: func(mocks fxMocks) {
				mocks.cache.EXPECT().GetAppliedRate(gomock.Any(), testBase, testQuote).
					Return(entity.AppliedFXRate{}, false)
				mocks.rateRepo.EXPECT().
					GetLatest(gomock.Any(), testBase, testQuote).
					Return(nil, entity.ErrFXRateNotFound)
				mocks.rateRepo.EXPECT().
					GetLatestBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
					Return(nil, entity.ErrFXRateNotFound)
				mocks.rateRepo.EXPECT().
					GetLatestExceptSource(gomock.Any(), testBase, testQuote, entity.FXSourceManual).
					Return(nil, entity.ErrFXRateNotFound)
				mocks.markupRepo.EXPECT().
					GetEffective(gomock.Any(), testBase, testQuote, gomock.Any()).
					Return(nil, entity.ErrDatabaseCommon)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			svc, mocks := newFXService(t, nil)
			c.setup(mocks)

			status, err := svc.PairStatus(t.Context(), testBase, testQuote, testFXNow)
			require.Error(t, err)
			assert.Nil(t, status)
		})
	}
}

// TestFXService_History фиксирует, что история курсов отдаётся ТОЛЬКО ручная.
//
// Курсы Нацбанка в этот список не идут: история нужна, чтобы объяснить, кто
// и когда поменял цену, а выдача источника меняется сама и объясняет только
// сама себя.
func TestFXService_History(t *testing.T) {
	t.Parallel()

	t.Run("курсы", func(t *testing.T) {
		t.Parallel()

		svc, mocks := newFXService(t, nil)
		want := entity.FXRates{testRate(t, "480", testFXNow)}

		mocks.rateRepo.EXPECT().
			ListBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual, 10).
			Return(want, nil)

		got, err := svc.RateHistory(t.Context(), testBase, testQuote, 10)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("отказ на курсах", func(t *testing.T) {
		t.Parallel()

		svc, mocks := newFXService(t, nil)
		mocks.rateRepo.EXPECT().
			ListBySource(gomock.Any(), testBase, testQuote, entity.FXSourceManual, 10).
			Return(nil, entity.ErrDatabaseCommon)

		_, err := svc.RateHistory(t.Context(), testBase, testQuote, 10)
		require.Error(t, err)
	})

	t.Run("наценки", func(t *testing.T) {
		t.Parallel()

		svc, mocks := newFXService(t, nil)
		want := entity.FXMarkups{entity.NewFXMarkup(testBase, testQuote, 200, testFXNow, nil, testFXNow)}

		mocks.markupRepo.EXPECT().List(gomock.Any(), testBase, testQuote, 10).Return(want, nil)

		got, err := svc.MarkupHistory(t.Context(), testBase, testQuote, 10)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("отказ на наценках", func(t *testing.T) {
		t.Parallel()

		svc, mocks := newFXService(t, nil)
		mocks.markupRepo.EXPECT().
			List(gomock.Any(), testBase, testQuote, 10).
			Return(nil, entity.ErrDatabaseCommon)

		_, err := svc.MarkupHistory(t.Context(), testBase, testQuote, 10)
		require.Error(t, err)
	})
}

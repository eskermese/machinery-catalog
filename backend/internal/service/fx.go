package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// RateProvider внешний источник курса.
//
// Интерфейс объявлен здесь, где потребляется: сервису важен только факт
// получения курса, а замена источника (Нацбанк, биржа, ручной ввод) не должна
// затрагивать ни сценарий синхронизации, ни хранилище.
type RateProvider interface {
	// FetchRate возвращает курс пары на дату, ближайшую к переданному моменту.
	// Дату определяет сам источник: он публикует курс на свои сутки.
	FetchRate(ctx context.Context, base, quote string, at time.Time) (*entity.FXRate, error)
}

// FXService представляет интерфейс сервиса курсов валют.
type FXService interface {
	// AppliedRate возвращает применённый курс пары на текущий момент.
	AppliedRate(ctx context.Context, base, quote string, currentTime time.Time) (entity.AppliedFXRate, error)
	// FixRateForQuote фиксирует курс для коммерческого предложения.
	FixRateForQuote(ctx context.Context, base, quote string, currentTime time.Time) (entity.AppliedFXRate, error)
	// SaveRate сохраняет курс от источника и сбрасывает кэш пары.
	SaveRate(ctx context.Context, rate *entity.FXRate) (*entity.FXRate, error)
	// SetMarkup меняет наценку и сбрасывает кэш пары.
	SetMarkup(ctx context.Context, markup *entity.FXMarkup) (*entity.FXMarkup, error)
	// SyncRate забирает курс у источника и сохраняет его.
	SyncRate(ctx context.Context, base, quote string, currentTime time.Time) error
	// PairStatus описывает состояние курса пары для админки: действующий
	// курс, что предложил внешний источник, что ввёл человек, признак
	// устаревания. Отвечает на вопрос «почему на витрине нет цены в тенге»
	// до того, как его задаст клиент.
	PairStatus(
		ctx context.Context,
		base, quote string,
		currentTime time.Time,
	) (*entity.FXPairStatus, error)
	// RateHistory возвращает историю ручных курсов пары, свежие первыми.
	RateHistory(ctx context.Context, base, quote string, limit int) (entity.FXRates, error)
	// MarkupHistory возвращает историю наценок пары, свежие первыми.
	MarkupHistory(ctx context.Context, base, quote string, limit int) (entity.FXMarkups, error)
}

// fxService реализация сервиса курсов валют.
type fxService struct {
	rateRepo   repository.FXRateRepository   // Репозиторий курсов
	markupRepo repository.FXMarkupRepository // Репозиторий наценки
	cache      repository.FXCache            // Кэш применённых курсов
	provider   RateProvider                  // Внешний источник курса

	// staleAfterDays через сколько суток курс считается устаревшим.
	// Задаётся настройкой: праздники сдвигают публикацию источника.
	staleAfterDays int

	// pivotCurrency валюта, против которой внешний источник публикует курсы.
	// Через неё выводится пара, которой у источника нет напрямую, — см.
	// deriveCrossRate.
	pivotCurrency string
}

// NewFXService создаёт сервис курсов валют.
func NewFXService(
	rateRepo repository.FXRateRepository,
	markupRepo repository.FXMarkupRepository,
	fxCache repository.FXCache,
	provider RateProvider,
	pivotCurrency string,
	staleAfterDays int,
) FXService {
	return &fxService{
		rateRepo:       rateRepo,
		markupRepo:     markupRepo,
		cache:          fxCache,
		provider:       provider,
		pivotCurrency:  pivotCurrency,
		staleAfterDays: staleAfterDays,
	}
}

// PairStatus описывает состояние курса пары для админки.
//
// Собирается из трёх независимых источников — действующий курс, курс
// внешнего источника, ручной ввод, — и каждый из них может отсутствовать.
// Отсутствие любого НЕ является ошибкой: «курса нет» — это ровно то
// состояние, ради показа которого ручка и заведена.
func (s *fxService) PairStatus(
	ctx context.Context,
	base, quote string,
	currentTime time.Time,
) (*entity.FXPairStatus, error) {
	status := &entity.FXPairStatus{BaseCurrency: base, QuoteCurrency: quote}

	if err := s.fillEffective(ctx, status, currentTime); err != nil {
		return nil, err
	}

	if manual, manualErr := s.rateRepo.GetLatestBySource(
		ctx, base, quote, entity.FXSourceManual,
	); manualErr == nil {
		status.Manual = manual
	} else if !errors.Is(manualErr, entity.ErrFXRateNotFound) {
		return nil, fmt.Errorf("получение ручного курса: %w", manualErr)
	}

	if provider, providerErr := s.latestProviderRate(ctx, base, quote); providerErr != nil {
		return nil, providerErr
	} else if provider != nil {
		status.Provider = provider
	}

	if markup, markupErr := s.markupRepo.GetEffective(ctx, base, quote, currentTime); markupErr != nil {
		return nil, fmt.Errorf("получение наценки: %w", markupErr)
	} else if markup != nil {
		status.MarkupBps = markup.MarkupBps
	}

	return status, nil
}

// fillEffective проставляет действующий курс пары и признак его устаревания.
//
// Курс берётся тем же методом, что и на витрине: админка обязана показывать
// ровно то, по чему считается цена, а не своё представление о ней.
//
// Отсутствие курса — штатное состояние, а не ошибка: именно его и надо
// показать администратору, поэтому оно оставляет статус пустым и не всплывает
// наверх.
func (s *fxService) fillEffective(
	ctx context.Context,
	status *entity.FXPairStatus,
	currentTime time.Time,
) error {
	applied, err := s.AppliedRate(ctx, status.BaseCurrency, status.QuoteCurrency, currentTime)

	switch {
	case err == nil:
		status.Effective = &applied
		status.EffectiveSource = applied.Source
		status.Stale = entity.IsRateStale(applied.RateDate, currentTime, s.staleAfterDays)
	case errors.Is(err, entity.ErrFXRateNotFound):
		return nil
	default:
		return fmt.Errorf("получение действующего курса: %w", err)
	}

	return nil
}

// latestProviderRate возвращает последний курс пары НЕ от ручного ввода.
//
// Спрашивается от противного, потому что имя внешнего источника сервису
// неизвестно: его знает клиент провайдера, а сервис работает с портом.
//
// GetLatest здесь не годится: он применяет приоритет источников и при
// заведённом ручном курсе вернул бы именно его — то есть скрыл бы то, ради
// чего курс источника и показывается.
func (s *fxService) latestProviderRate(
	ctx context.Context,
	base, quote string,
) (*entity.FXRate, error) {
	rate, err := s.rateRepo.GetLatestExceptSource(ctx, base, quote, entity.FXSourceManual)

	switch {
	case errors.Is(err, entity.ErrFXRateNotFound):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("получение курса источника: %w", err)
	}

	return rate, nil
}

// RateHistory возвращает историю ручных курсов пары.
func (s *fxService) RateHistory(
	ctx context.Context,
	base, quote string,
	limit int,
) (entity.FXRates, error) {
	rates, err := s.rateRepo.ListBySource(ctx, base, quote, entity.FXSourceManual, limit)
	if err != nil {
		return nil, fmt.Errorf("история курсов: %w", err)
	}

	return rates, nil
}

// MarkupHistory возвращает историю наценок пары.
func (s *fxService) MarkupHistory(
	ctx context.Context,
	base, quote string,
	limit int,
) (entity.FXMarkups, error) {
	markups, err := s.markupRepo.List(ctx, base, quote, limit)
	if err != nil {
		return nil, fmt.Errorf("история наценок: %w", err)
	}

	return markups, nil
}

// AppliedRate возвращает применённый курс пары: кэш, затем база.
//
// Курса нет вовсе — ErrFXRateNotFound: подставить единицу значило бы продать
// технику по цене в долларах как за тенге, а это тихая потеря денег вместо
// заметного отказа.
//
// Курса за сегодня нет, но есть вчерашний — отдаётся вчерашний вместе с его
// датой, и это НЕ ошибка. Дата возвращается наружу именно затем, чтобы
// потребитель сам решил, показывать ли предупреждение о несвежести: для
// показа цены в каталоге вчерашний курс приемлем, для выпуска КП на крупную
// сумму менеджер, возможно, захочет дождаться свежего.
func (s *fxService) AppliedRate(
	ctx context.Context,
	base, quote string,
	currentTime time.Time,
) (entity.AppliedFXRate, error) {
	if cached, ok := s.cachedRate(ctx, base, quote); ok {
		return cached, nil
	}

	applied, err := s.buildAppliedRate(ctx, base, quote, currentTime)
	if err != nil {
		return entity.AppliedFXRate{}, err
	}

	if s.cache != nil {
		s.cache.SetAppliedRate(ctx, base, quote, applied)
	}

	return applied, nil
}

// FixRateForQuote фиксирует курс для коммерческого предложения.
//
// Отдельный метод, а не вызов AppliedRate напрямую: фиксация в КП — момент,
// после которого курс уже нельзя пересчитать, и точка входа для этого сценария
// должна быть названа так, чтобы её нельзя было применить случайно.
// Возвращает курс, дату курса и наценку одним значением — ровно то, что уходит
// в quotes.fx_rate, quotes.fx_rate_date и quotes.fx_markup_bps.
func (s *fxService) FixRateForQuote(
	ctx context.Context,
	base, quote string,
	currentTime time.Time,
) (entity.AppliedFXRate, error) {
	return s.AppliedRate(ctx, base, quote, currentTime)
}

// SaveRate сохраняет курс от источника и сбрасывает кэш пары.
func (s *fxService) SaveRate(ctx context.Context, rate *entity.FXRate) (*entity.FXRate, error) {
	if rate == nil {
		return nil, entity.ErrNilPointer
	}

	created, err := s.rateRepo.Create(ctx, rate)
	if err != nil {
		return nil, fmt.Errorf("сохранение курса: %w", err)
	}

	s.invalidate(ctx, rate.BaseCurrency, rate.QuoteCurrency)

	return created, nil
}

// SetMarkup меняет наценку и сбрасывает кэш пары.
func (s *fxService) SetMarkup(ctx context.Context, markup *entity.FXMarkup) (*entity.FXMarkup, error) {
	if markup == nil {
		return nil, entity.ErrNilPointer
	}

	created, err := s.markupRepo.Create(ctx, markup)
	if err != nil {
		return nil, fmt.Errorf("сохранение наценки: %w", err)
	}

	s.invalidate(ctx, markup.BaseCurrency, markup.QuoteCurrency)

	return created, nil
}

// SyncRate забирает курс у источника и сохраняет его.
//
// Повторная загрузка того же дня — не ошибка сценария: расписание могло
// сработать дважды, и это штатно. Конфликт гасится здесь, а не в репозитории,
// потому что различить безобидный повтор и попытку подменить показанный
// клиенту курс может только вызывающий код, и для синхронизации по расписанию
// ответ однозначен — повтор безобиден.
func (s *fxService) SyncRate(ctx context.Context, base, quote string, currentTime time.Time) error {
	if s.provider == nil {
		return entity.ErrFXProviderNotConfigured
	}

	rate, err := s.provider.FetchRate(ctx, base, quote, currentTime)
	if err != nil {
		return fmt.Errorf("получение курса от источника: %w", err)
	}

	if _, err = s.SaveRate(ctx, rate); err != nil {
		if errors.Is(err, entity.ErrFXRateAlreadyExists) {
			logger.FromContext(ctx).WithFields(logger.Fields{
				"base":      base,
				"quote":     quote,
				"rate_date": rate.RateDate,
			}).Info("курс за эту дату уже загружен")

			return nil
		}

		return err
	}

	return nil
}

// cachedRate читает применённый курс из кэша.
// Отсутствие кэша в сборке равнозначно промаху: сервис обязан работать и без него.
func (s *fxService) cachedRate(ctx context.Context, base, quote string) (entity.AppliedFXRate, bool) {
	if s.cache == nil {
		return entity.AppliedFXRate{}, false
	}

	return s.cache.GetAppliedRate(ctx, base, quote)
}

// buildAppliedRate собирает применённый курс из последнего курса и действующей наценки.
func (s *fxService) buildAppliedRate(
	ctx context.Context,
	base, quote string,
	currentTime time.Time,
) (entity.AppliedFXRate, error) {
	rate, err := s.rateRepo.GetLatest(ctx, base, quote)
	if errors.Is(err, entity.ErrFXRateNotFound) {
		// Прямого курса пары нет — пробуем вывести его через валюту-посредник.
		rate, err = s.deriveCrossRate(ctx, base, quote)
	}

	if err != nil {
		return entity.AppliedFXRate{}, fmt.Errorf("получение курса: %w", err)
	}

	markup, err := s.markupRepo.GetEffective(ctx, base, quote, currentTime)
	if err != nil {
		return entity.AppliedFXRate{}, fmt.Errorf("получение наценки: %w", err)
	}

	// Наценка не задана — продаём по курсу источника. Это штатное состояние,
	// а не повод отказать: сервис поднимается раньше, чем бизнес назначит наценку.
	markupBps := 0
	if markup != nil {
		markupBps = markup.MarkupBps
	}

	return entity.NewAppliedFXRate(rate, markupBps), nil
}

// deriveCrossRate выводит курс пары через валюту-посредник.
//
// Зачем. Нацбанк публикует курсы К ТЕНГЕ: CNY→KZT и USD→KZT есть, а CNY→USD
// напрямую нет. Требовать за неё ручной ввод значило бы заставлять человека
// ежедневно считать деление, которое сервис умеет сделать сам, — и получать
// пустое место на витрине в тот день, когда он забыл.
//
// КУРСЫ БЕРУТСЯ ЧИСТЫЕ, БЕЗ НАЦЕНОК. Это не мелочь. Наценки на паре
// CNY→KZT и на паре USD→KZT входили бы в деление и там СОКРАЩАЛИСЬ:
//
//	(raw_cny_kzt * (1+m1)) / (raw_usd_kzt * (1+m2))  =  raw_cross * (1+m1)/(1+m2)
//
// При равных наценках множитель обращается в единицу, и наценки не остаётся
// вовсе; при разных получается разность, которую никто не назначал. Поэтому
// делятся чистые курсы, а наценка пары CNY→USD накручивается РОВНО ОДИН РАЗ
// и берётся из своей строки fx_markups — это делает вызывающий код,
// buildAppliedRate, ровно так же, как для прямого курса.
//
// Наценка дважды на одну цену — не «страховка», а тихое завышение; здесь
// её нет ни разу лишней и ни разу недостающей.
//
// Дата у выведенного курса — более РАННЯЯ из двух: курс не может считаться
// свежее самой старой из величин, из которых он получен.
func (s *fxService) deriveCrossRate(
	ctx context.Context,
	base, quote string,
) (*entity.FXRate, error) {
	// Посредник не задан, либо он сам и есть одна из сторон пары —
	// выводить не из чего.
	if s.pivotCurrency == "" || base == s.pivotCurrency || quote == s.pivotCurrency {
		return nil, entity.ErrFXRateNotFound
	}

	basePivot, err := s.rateRepo.GetLatest(ctx, base, s.pivotCurrency)
	if err != nil {
		return nil, err
	}

	quotePivot, err := s.rateRepo.GetLatest(ctx, quote, s.pivotCurrency)
	if err != nil {
		return nil, err
	}

	if quotePivot.Rate.IsZero() {
		return nil, entity.ErrFXRateNotFound
	}

	rateDate := basePivot.RateDate
	if quotePivot.RateDate.Before(rateDate) {
		rateDate = quotePivot.RateDate
	}

	return entity.NewFXRate(
		base,
		quote,
		entity.FXSourceDerived,
		basePivot.Rate.Div(quotePivot.Rate),
		rateDate,
		basePivot.FetchedAt,
	), nil
}

// invalidate сбрасывает кэш пары. Ошибку сброса гасит сам кэш: строка в БД
// уже сохранена и она источник правды, а разошедшийся кэш дотухнет по TTL.
func (s *fxService) invalidate(ctx context.Context, base, quote string) {
	if s.cache == nil {
		return
	}

	s.cache.DeleteAppliedRate(ctx, base, quote)
}

// notConfiguredRateProvider заглушка источника курса.
//
// Подключение реального источника — отдельная задача, но сервис уже собирается
// и запускается, поэтому провайдер нужен сейчас. Заглушка отказывает явной
// доменной ошибкой, а не возвращает выдуманный курс: воркер запишет отказ
// в лог, и незаконченная настройка будет видна, а не спрятана за правдоподобным
// числом.
type notConfiguredRateProvider struct{}

// NewNotConfiguredRateProvider создаёт заглушку источника курса.
func NewNotConfiguredRateProvider() RateProvider {
	return notConfiguredRateProvider{}
}

// FetchRate всегда отказывает: источник не подключён.
func (notConfiguredRateProvider) FetchRate(
	_ context.Context,
	_, _ string,
	_ time.Time,
) (*entity.FXRate, error) {
	return nil, entity.ErrFXProviderNotConfigured
}

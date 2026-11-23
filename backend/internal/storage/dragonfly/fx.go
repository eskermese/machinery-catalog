package dragonfly

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	"github.com/eskermese/machinery-catalog/backend/pkg/cache"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// fxRateCacheTTL время жизни применённого курса в кэше.
//
// Пять минут — не про частоту обновления курса (источник публикует его раз
// в сутки), а про то, как долго сервис может отдавать устаревшее значение,
// если явная инвалидация не сработала. Инвалидация здесь основной механизм:
// кэш сбрасывается сразу после загрузки нового курса и после смены наценки.
// TTL — страховка на случай, когда сброс не дошёл: перезапуск в момент записи,
// недоступность кэша ровно в этот миг, вторая реплика сервиса.
//
// Пять минут выбраны как верхняя граница расхождения, которую бизнес готов
// принять при смене наценки. Меньше — кэш перестаёт снимать нагрузку с БД
// на горячем пути показа цены; больше — менеджер, поменявший наценку, слишком
// долго видит в интерфейсе старую цифру и меняет её второй раз.
const fxRateCacheTTL = 5 * time.Minute

// fxCache кэш применённых курсов поверх Redis-совместимого хранилища.
type fxCache struct {
	cacheData cache.Cache // Кэш данных
}

// NewFXCache создаёт кэш применённых курсов.
func NewFXCache(cacheData cache.Cache) repository.FXCache {
	return &fxCache{cacheData: cacheData}
}

// cachedAppliedRate представление применённого курса в кэше.
//
// Курс хранится строкой, а не числом: JSON-число прошло бы через float64
// и потеряло бы младшие разряды, ради точного хранения которых курс
// и объявлен numeric(18,6).
type cachedAppliedRate struct {
	Rate      string    `json:"rate"`      // Применённый курс
	RateDate  time.Time `json:"rateDate"`  // Дата курса источника
	MarkupBps int       `json:"markupBps"` // Наценка, зашитая в курс
	// Source кто поставил курс. Хранится наравне с остальным: админка
	// показывает действующий источник, и курс, пришедший из кэша, не должен
	// отвечать на этот вопрос иначе, чем тот же курс из базы.
	Source string `json:"source"`
}

// GetAppliedRate возвращает применённый курс из кэша.
//
// Промах, отказ кэша и битое значение неразличимы для вызывающего кода:
// во всех трёх случаях возвращается false, и вызывающий читает из БД.
// Ошибка логируется на warn и наверх не поднимается — недоступность кэша
// не повод отказывать в показе цены.
func (c *fxCache) GetAppliedRate(ctx context.Context, base, quote string) (entity.AppliedFXRate, bool) {
	key := appliedRateKey(base, quote)

	raw, err := c.cacheData.Get(ctx, key)
	if err != nil {
		// Промах — обычное состояние кэша, поэтому отдельного разбора «это
		// промах или сбой» здесь нет: логируется всё на warn, решение одно.
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("чтение курса из кэша")

		return entity.AppliedFXRate{}, false
	}

	var cached cachedAppliedRate
	if err = json.Unmarshal([]byte(raw), &cached); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("разбор курса из кэша")

		return entity.AppliedFXRate{}, false
	}

	rate, err := decimal.NewFromString(cached.Rate)
	if err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
			"rate":  cached.Rate,
		}).Warn("разбор значения курса из кэша")

		return entity.AppliedFXRate{}, false
	}

	return entity.AppliedFXRate{
		Rate:      rate,
		RateDate:  cached.RateDate,
		MarkupBps: cached.MarkupBps,
		Source:    cached.Source,
	}, true
}

// SetAppliedRate сохраняет применённый курс в кэш. Ошибка только логируется.
func (c *fxCache) SetAppliedRate(ctx context.Context, base, quote string, rate entity.AppliedFXRate) {
	key := appliedRateKey(base, quote)

	payload, err := json.Marshal(cachedAppliedRate{
		Rate:      rate.String(),
		RateDate:  rate.RateDate,
		MarkupBps: rate.MarkupBps,
		Source:    rate.Source,
	})
	if err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("сериализация курса для кэша")

		return
	}

	if err = c.cacheData.Set(ctx, key, payload, fxRateCacheTTL); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("сохранение курса в кэш")
	}
}

// DeleteAppliedRate удаляет применённый курс из кэша. Ошибка только логируется.
//
// Неудачное удаление не откатывает загрузку курса: строка в БД уже сохранена
// и она источник правды, а разошедшийся кэш дотухнет по TTL.
func (c *fxCache) DeleteAppliedRate(ctx context.Context, base, quote string) {
	key := appliedRateKey(base, quote)

	if err := c.cacheData.Delete(ctx, key); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("сброс курса в кэше")
	}
}

// appliedRateKey собирает ключ кэша применённого курса.
// Пара в ключе, а не в значении: курсы разных пар вытесняются независимо.
func appliedRateKey(base, quote string) string {
	return fmt.Sprintf("fx:applied:%s:%s", base, quote)
}

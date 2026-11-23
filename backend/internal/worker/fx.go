// Package worker содержит фоновые задачи, запускаемые по расписанию.
//
// Воркер — такой же входящий адаптер, как HTTP-обработчик: он инициирует
// сценарий сервиса, только поводом служит время, а не запрос. Поэтому он лежит
// рядом с транспортом по смыслу, но отдельным пакетом: транспортом он не
// является и ничего не знает ни о протоколах, ни о сериализации.
package worker

import (
	"context"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/service"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// FXSync фоновая синхронизация курса валют.
type FXSync struct {
	fxService     service.FXService // Сервис курсов валют
	interval      time.Duration     // Интервал синхронизации
	baseCurrency  string            // Валюта прайса
	quoteCurrency string            // Валюта, в которую пересчитывается цена
}

// NewFXSync создаёт воркер синхронизации курса.
func NewFXSync(
	fxService service.FXService,
	interval time.Duration,
	base, quote string,
) *FXSync {
	return &FXSync{
		fxService:     fxService,
		interval:      interval,
		baseCurrency:  base,
		quoteCurrency: quote,
	}
}

// Run запускает синхронизацию до отмены контекста.
//
// Один прогон выполняется сразу, до первого тика: иначе после перезапуска
// сервис ждал бы целый интервал, ничего не зная о курсе, — а перезапуск
// как раз и случается тогда, когда курс мог измениться.
//
// Ошибка синхронизации логируется и НЕ возвращается наверх: курс не критичен
// для работы приложения. Отдать ошибку в errgroup значило бы уронить весь
// сервис из-за недоступного внешнего источника — вместе с HTTP, которому
// последний известный курс из базы вполне достаточен.
//
// Отдельного механизма повторов внутри тика нет намеренно, и pkg/repeatable
// здесь не используется: его DoWithTries не принимает контекст, поэтому
// после отмены он продолжил бы спать между попытками и задержал бы остановку
// приложения. Повтор при этом уже обеспечен самим тикером — следующий тик
// и есть следующая попытка, а интервал синхронизации курса измеряется
// часами, и внутренние ретраи в его масштабе ничего не добавляют.
func (w *FXSync) Run(ctx context.Context) error {
	log := logger.FromContext(ctx).WithFields(logger.Fields{
		"base":     w.baseCurrency,
		"quote":    w.quoteCurrency,
		"interval": w.interval.String(),
	})

	log.Info("Синхронизация курса запущена")

	w.syncOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Синхронизация курса остановлена")

			return nil
		case <-ticker.C:
			w.syncOnce(ctx)
		}
	}
}

// syncOnce выполняет один прогон синхронизации, гася ошибку в лог.
func (w *FXSync) syncOnce(ctx context.Context) {
	if err := w.fxService.SyncRate(ctx, w.baseCurrency, w.quoteCurrency, time.Now().UTC()); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"base":  w.baseCurrency,
			"quote": w.quoteCurrency,
		}).Warn("синхронизация курса")
	}
}

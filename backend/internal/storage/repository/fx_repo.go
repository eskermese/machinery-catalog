package repository

import (
	"context"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// FXRateRepository исходящий порт курсов валют.
//
// Таблица append-only: методов изменения и удаления здесь нет намеренно.
// Курс, по которому уже выпущено коммерческое предложение, обязан оставаться
// воспроизводимым, поэтому «поправить вчерашний курс» — не операция домена.
type FXRateRepository interface {
	// Create сохраняет курс источника за дату.
	//
	// Повторная загрузка того же источника за ту же дату — конфликт:
	// возвращается ErrFXRateAlreadyExists, а вторая строка не создаётся.
	// Ошибка, а не тихое ON CONFLICT DO NOTHING, потому что повторная загрузка
	// бывает двух разных видов: безобидный повтор расписания и попытка
	// подменить уже показанный клиенту курс. Различить их может только
	// вызывающий код, и отдавать ему для этого нечего, кроме отказа.
	Create(ctx context.Context, rate *entity.FXRate) (*entity.FXRate, error)
	// GetLatest возвращает последний известный курс пары независимо от даты.
	// Отсутствие курса — ErrFXRateNotFound.
	GetLatest(ctx context.Context, base, quote string) (*entity.FXRate, error)
	// GetLatestBySource возвращает последний курс пары от конкретного
	// источника. Нужен админке: она показывает рядом то, что предложил
	// внешний источник, и то, что поправил человек, — иначе «действующий
	// курс» не с чем сравнить. Отсутствие — ErrFXRateNotFound.
	GetLatestBySource(ctx context.Context, base, quote, source string) (*entity.FXRate, error)
	// GetLatestExceptSource возвращает последний курс пары от ЛЮБОГО источника,
	// кроме указанного.
	//
	// Нужен потому, что имя внешнего источника коду сервиса неизвестно: его
	// знает клиент провайдера, а сервис работает с портом. Спросить «что
	// предложил Нацбанк» можно только от противного — «всё, кроме ручного
	// ввода». Отсутствие — ErrFXRateNotFound.
	GetLatestExceptSource(ctx context.Context, base, quote, source string) (*entity.FXRate, error)
	// ListBySource возвращает историю курсов пары от источника, свежие
	// первыми. Ограничение сверху обязательно: история растёт ежедневно,
	// и отдавать её целиком незачем.
	ListBySource(ctx context.Context, base, quote, source string, limit int) (entity.FXRates, error)
}

// FXMarkupRepository исходящий порт наценки к курсу.
type FXMarkupRepository interface {
	// Create добавляет запись наценки.
	Create(ctx context.Context, markup *entity.FXMarkup) (*entity.FXMarkup, error)
	// GetEffective возвращает наценку, действующую на переданный момент:
	// запись с наибольшим effective_from, не превышающим этот момент.
	//
	// Отсутствие записи — не ошибка: наценка может быть просто не задана,
	// и это штатное состояние «продаём по курсу источника». Возвращается nil.
	GetEffective(ctx context.Context, base, quote string, at time.Time) (*entity.FXMarkup, error)
	// List возвращает историю наценок пары, свежие первыми.
	List(ctx context.Context, base, quote string, limit int) (entity.FXMarkups, error)
}

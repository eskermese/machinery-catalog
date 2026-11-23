package repository

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// CacheStore представляет интерфейс для работы с кэшем.
type CacheStore interface {
	// Ping проверяет, что соединение с кэшем установлено.
	Ping() error
	// FXCache возвращает кэш применённых курсов.
	FXCache() FXCache
	// CatalogCache возвращает кэш справочных данных каталога.
	CatalogCache() CatalogCache
}

// FXCache кэш применённого курса валютной пары.
//
// Кэшируется применённый курс целиком — вместе с датой курса и наценкой, —
// а не одно число: потребителю нужны все три значения сразу (они уходят в КП
// одним набором), и собирать их из разных источников означало бы поймать
// момент, когда курс уже из кэша, а наценка ещё из базы.
//
// Ни один метод не возвращает ошибку наверх как повод отказать: кэш —
// ускоритель, а не источник правды. Промах и отказ кэша неразличимы для
// вызывающего кода и одинаково ведут к чтению из БД.
type FXCache interface {
	// GetAppliedRate возвращает применённый курс из кэша.
	// Второе значение — признак попадания: false и при промахе, и при отказе кэша.
	GetAppliedRate(ctx context.Context, base, quote string) (entity.AppliedFXRate, bool)
	// SetAppliedRate сохраняет применённый курс в кэш.
	SetAppliedRate(ctx context.Context, base, quote string, rate entity.AppliedFXRate)
	// DeleteAppliedRate удаляет применённый курс из кэша.
	// Вызывается после загрузки нового курса и после смены наценки.
	DeleteAppliedRate(ctx context.Context, base, quote string)
}

// CatalogCache кэш справочных данных каталога.
//
// Кэшируются дерево категорий и набор характеристик категории: они читаются
// на каждой странице каталога, а меняются раз в месяцы. Результаты поиска
// машин не кэшируются — комбинаций фильтров столько, что попаданий по одному
// и тому же ключу практически не будет.
//
// Контракт тот же, что у FXCache: ни один метод не возвращает ошибку. Промах
// и отказ кэша неразличимы для вызывающего кода и одинаково ведут к чтению
// из БД.
type CatalogCache interface {
	// GetCategoryTree возвращает дерево категорий из кэша.
	// Второе значение — признак попадания: false и при промахе, и при отказе.
	GetCategoryTree(ctx context.Context) (entity.Categories, bool)
	// SetCategoryTree сохраняет дерево категорий в кэш.
	SetCategoryTree(ctx context.Context, categories entity.Categories)
	// DeleteCategoryTree сбрасывает дерево категорий.
	// Вызывается после любого изменения категории.
	DeleteCategoryTree(ctx context.Context)

	// GetCategoryAttributes возвращает характеристики категории из кэша.
	GetCategoryAttributes(ctx context.Context, categoryID string) (entity.Attributes, bool)
	// SetCategoryAttributes сохраняет характеристики категории в кэш.
	SetCategoryAttributes(ctx context.Context, categoryID string, attributes entity.Attributes)
	// DeleteCategoryAttributes сбрасывает характеристики категории.
	// Вызывается после любого изменения характеристики или её вариантов.
	DeleteCategoryAttributes(ctx context.Context, categoryID string)
}

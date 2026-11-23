package dragonfly

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	"github.com/eskermese/machinery-catalog/backend/pkg/cache"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// catalogCacheTTL время жизни справочных данных каталога в кэше.
//
// Дерево категорий и набор характеристик читаются на каждой странице каталога,
// а меняются раз в месяцы — соотношение чтений к записям здесь тысячи к одной,
// и кэш снимает с базы почти весь этот трафик.
//
// Час, а не пять минут как у курса: цена устаревания здесь несравнимо ниже.
// Устаревший курс — неверная цена в коммерческом предложении; устаревшее
// дерево категорий — новая категория, которая появится в меню на несколько
// минут позже. При этом инвалидация всё равно явная (при любом изменении
// категории или характеристики), поэтому час — это срок жизни только для
// случая, когда сброс не дошёл.
//
// Верхнюю границу задаёт терпимость редактора: правка названия категории,
// не проявившаяся в течение рабочего дня, читается как поломка админки.
const catalogCacheTTL = time.Hour

// catalogCache кэш справочных данных каталога.
//
// Результаты поиска машин здесь не кэшируются намеренно: комбинаций фильтров
// (категория × цена × наличие × состояние × год × произвольный набор
// характеристик) столько, что попаданий по одному и тому же ключу практически
// не будет, а кэш превратится в свалку записей, вытесняющих друг друга.
type catalogCache struct {
	cacheData cache.Cache // Кэш данных
}

// NewCatalogCache создаёт кэш справочных данных каталога.
func NewCatalogCache(cacheData cache.Cache) repository.CatalogCache {
	return &catalogCache{cacheData: cacheData}
}

// categoryTreeKey ключ дерева категорий.
const categoryTreeKey = "catalog:categories:tree"

// categoryAttributesKey собирает ключ характеристик категории.
func categoryAttributesKey(categoryID string) string {
	return fmt.Sprintf("catalog:attributes:%s", categoryID)
}

// GetCategoryTree возвращает дерево категорий из кэша.
// Промах, отказ кэша и битое значение неразличимы: во всех случаях false.
func (c *catalogCache) GetCategoryTree(ctx context.Context) (entity.Categories, bool) {
	var categories entity.Categories

	if !c.get(ctx, categoryTreeKey, &categories) {
		return nil, false
	}

	return categories, true
}

// SetCategoryTree сохраняет дерево категорий в кэш.
func (c *catalogCache) SetCategoryTree(ctx context.Context, categories entity.Categories) {
	c.set(ctx, categoryTreeKey, categories)
}

// DeleteCategoryTree сбрасывает дерево категорий в кэше.
func (c *catalogCache) DeleteCategoryTree(ctx context.Context) {
	c.delete(ctx, categoryTreeKey)
}

// GetCategoryAttributes возвращает характеристики категории из кэша.
func (c *catalogCache) GetCategoryAttributes(ctx context.Context, categoryID string) (entity.Attributes, bool) {
	var attributes entity.Attributes

	if !c.get(ctx, categoryAttributesKey(categoryID), &attributes) {
		return nil, false
	}

	return attributes, true
}

// SetCategoryAttributes сохраняет характеристики категории в кэш.
func (c *catalogCache) SetCategoryAttributes(ctx context.Context, categoryID string, attributes entity.Attributes) {
	c.set(ctx, categoryAttributesKey(categoryID), attributes)
}

// DeleteCategoryAttributes сбрасывает характеристики категории в кэше.
func (c *catalogCache) DeleteCategoryAttributes(ctx context.Context, categoryID string) {
	c.delete(ctx, categoryAttributesKey(categoryID))
}

// get читает и разбирает значение по ключу.
// Возвращает false при любой неудаче: кэш — ускоритель, а не источник правды,
// и отказать в выдаче каталога из-за него нельзя.
func (c *catalogCache) get(ctx context.Context, key string, target any) bool {
	raw, err := c.cacheData.Get(ctx, key)
	if err != nil {
		// Промах — обычное состояние кэша, поэтому разбора «промах или сбой»
		// здесь нет: логируется всё на warn, решение одно.
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("чтение каталога из кэша")

		return false
	}

	if err = json.Unmarshal([]byte(raw), target); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("разбор каталога из кэша")

		return false
	}

	return true
}

// set сохраняет значение по ключу. Ошибка только логируется.
func (c *catalogCache) set(ctx context.Context, key string, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("сериализация каталога для кэша")

		return
	}

	if err = c.cacheData.Set(ctx, key, payload, catalogCacheTTL); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("сохранение каталога в кэш")
	}
}

// delete удаляет значение по ключу. Ошибка только логируется.
//
// Неудачный сброс не откатывает изменение: строка в БД уже сохранена и она
// источник правды, а разошедшийся кэш дотухнет по TTL.
func (c *catalogCache) delete(ctx context.Context, key string) {
	if err := c.cacheData.Delete(ctx, key); err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err,
			"key":   key,
		}).Warn("сброс каталога в кэше")
	}
}

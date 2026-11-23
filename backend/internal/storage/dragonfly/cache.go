package dragonfly

import (
	"context"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/config"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	"github.com/eskermese/machinery-catalog/backend/pkg/cache"
	"github.com/eskermese/machinery-catalog/backend/pkg/cache/redis"
)

// connectionTimeout время ожидания подключения к кэшу.
const connectionTimeout = 3 * time.Second

// Dragonfly реализация кэша.
type Dragonfly struct {
	cacheData cache.Cache // Кэш данных.

	connectionTimeout time.Duration // Время ожидания подключения к кэшу.

	fxCache      repository.FXCache      // Кэш применённых курсов.
	catalogCache repository.CatalogCache // Кэш справочных данных каталога.
}

// New создает кэш поверх Redis-совместимого хранилища.
func New(conf config.Cache) repository.CacheStore {
	return &Dragonfly{
		cacheData: redis.New(&redis.Config{
			Addr:     conf.CacheAddr,
			Username: conf.CacheUsername,
			Password: conf.CachePassword,
		}),
		connectionTimeout: connectionTimeout,
	}
}

// Ping проверяет что соединение с кэшем установлено.
//
// Проверяется командой PING, а не чтением служебного ключа: чтение
// отсутствующего ключа возвращает ошибку «не найдено», и живой пустой кэш
// не отличался бы от недоступного.
func (m *Dragonfly) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), m.connectionTimeout)
	defer cancel()

	if err := m.cacheData.Ping(ctx); err != nil {
		return fmt.Errorf("пинг кэша: %w", err)
	}

	return nil
}

// FXCache возвращает кэш применённых курсов.
func (m *Dragonfly) FXCache() repository.FXCache {
	if m.fxCache == nil {
		m.fxCache = NewFXCache(m.cacheData)
	}

	return m.fxCache
}

// CatalogCache возвращает кэш справочных данных каталога.
func (m *Dragonfly) CatalogCache() repository.CatalogCache {
	if m.catalogCache == nil {
		m.catalogCache = NewCatalogCache(m.cacheData)
	}

	return m.catalogCache
}

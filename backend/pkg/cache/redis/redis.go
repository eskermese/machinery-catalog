package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	cache "github.com/eskermese/machinery-catalog/backend/pkg/cache"
)

const (
	valueExist    = 1 // Значение существует. Используется в Exists.
	valueNotExist = 0 // Значение не существует.
)

// ErrNotFound сообщает, что значение не найдено в Redis.
var ErrNotFound = errors.New("значение не найдено")

// Config представляет собой конфигурацию кэша.
type Config struct {
	Addr         string        // Адрес кэша.
	Username     string        // Имя пользователя для кэша.
	Password     string        // Пароль для кэша.
	DB           int           // Номер базы данных для кэша. Необязательный параметр. 0 - используется база данных по умолчанию.
	ReadTimeout  time.Duration // Тайм-аут для чтения из сокета
	WriteTimeout time.Duration // Тайм-аут для записи в сокет
}

// Cache представляет собой реализацию кэша.
type Cache struct {
	redis cache.UniversalClient // redis.
}

// Проверяем, что Cache реализует интерфейс cache.Cache.
var _ cache.Cache = (*Cache)(nil)

// New возвращает новый экземпляр кэша.
func New(cfg *Config) Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	return Cache{redis: rdb}
}

// GetClient возвращает чистый клиент.
func (c Cache) GetClient() cache.UniversalClient {
	return c.redis
}

// Incr увеличивает значение по ключу.
func (c Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.redis.Incr(ctx, key).Result()
}

// Decr уменьшает значение по ключу.
func (c Cache) Decr(ctx context.Context, key string) (int64, error) {
	return c.redis.Decr(ctx, key).Result()
}

// MGet возвращает значения по ключам.
func (c Cache) MGet(ctx context.Context, keys []string) ([]interface{}, error) {
	return c.redis.MGet(ctx, keys...).Result()
}

// Get возвращает значение по ключу.
func (c Cache) Get(ctx context.Context, key string) (string, error) {
	result, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNotFound
		}

		return "", fmt.Errorf("получение значения из кэша: %w", err)
	}

	return result, nil
}

// Set сохраняет значение по ключу.
// key - ключ, по которому необходимо сохранить значение.
// value - значение, которое необходимо сохранить. Хранится в виде json.
// expiration - время жизни значения, если 0, то значение не истекает.
func (c Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.redis.Set(ctx, key, value, expiration).Err()
}

// SetArgs сохраняет значение по ключу с определенными аргументами.
func (c Cache) SetArgs(ctx context.Context, key string, value interface{}, args cache.SetArgs) error {
	return c.redis.SetArgs(ctx, key, value, redis.SetArgs{
		TTL:      args.TTL,
		ExpireAt: args.ExpireAt,
		KeepTTL:  args.KeepTTL,
	}).Err()
}

// SetArgsAndReturn сохраняет значение по ключу с определенными аргументами и возвращает результат,
// если запись по ключу уже существует.
func (c Cache) SetArgsAndReturn(ctx context.Context, key string, value interface{}, args cache.SetArgs) (string, error) {
	result, err := c.redis.SetArgs(ctx, key, value, redis.SetArgs{
		Mode:     args.Mode,
		TTL:      args.TTL,
		ExpireAt: args.ExpireAt,
		KeepTTL:  args.KeepTTL,
		Get:      args.Get,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return result, ErrNotFound
	}

	return result, err
}

// MSet сохраняет значения по ключам.
// values - значения, которые необходимо сохранить. Хранятся в виде json.
func (c Cache) MSet(ctx context.Context, values map[string]interface{}) error {
	pairs := make([]interface{}, 0, len(values)+len(values))
	for key, value := range values {
		pairs = append(pairs, key, value)
	}

	return c.redis.MSet(ctx, pairs...).Err()
}

// ZAdd добавляет значение по ключу.
func (c Cache) ZAdd(ctx context.Context, key string, members ...cache.ZMember) error {
	z := make([]redis.Z, 0, len(members))
	for _, member := range members {
		z = append(z, redis.Z{Score: member.Score, Member: member.Member})
	}

	if len(z) == 0 {
		return nil
	}

	return c.redis.ZAdd(ctx, key, z...).Err()
}

// ZRem удаляет значения по ключу.
func (c Cache) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.redis.ZRem(ctx, key, members...).Err()
}

// ZRemRangeByScore удаляет значения по ключу в заданном диапазоне.
func (c Cache) ZRemRangeByScore(ctx context.Context, key, minVal, maxVal string) error {
	return c.redis.ZRemRangeByScore(ctx, key, minVal, maxVal).Err()
}

// ZRangeByScore возвращает значения по ключу в заданном диапазоне.
func (c Cache) ZRangeByScore(ctx context.Context, key, minVal, maxVal string) ([]string, error) {
	return c.redis.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: minVal,
		Max: maxVal,
	}).Result()
}

// Keys возвращает все ключи.
func (c Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.redis.Keys(ctx, pattern).Result()
}

// Delete удаляет значение по ключу.
func (c Cache) Delete(ctx context.Context, keys ...string) error {
	return c.redis.Del(ctx, keys...).Err()
}

// Exists проверяет существование значения по ключу.
func (c Cache) Exists(ctx context.Context, key string) (bool, error) {
	r, err := c.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return r == valueExist, nil
}

// GeoSearch возвращает геолокации по поисковому запросу.
func (c Cache) GeoSearch(ctx context.Context, key string, query *cache.GeoSearchQuery) (cache.GeoLocations, error) {
	if query == nil {
		query = &cache.GeoSearchQuery{}
	}

	// Преобразуем GeoSearchQuery в redis.GeoSearchQuery.
	rq := &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Member:     query.Member,
			Longitude:  query.Longitude,
			Latitude:   query.Latitude,
			Radius:     query.Radius,
			RadiusUnit: query.Unit,
			Sort:       query.Sort,
			Count:      query.Count,
		},
		WithCoord: query.WithCoord,
		WithDist:  query.WithDist,
	}

	result, err := c.redis.GeoSearchLocation(ctx, key, rq).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	// Преобразуем результат в cache.GeoLocations.
	geoLocations := make(cache.GeoLocations, 0, len(result))
	for _, loc := range result {
		geoLocations = append(geoLocations, cache.GeoLocation{
			Member:    loc.Name,
			Locations: cache.Position{Longitude: loc.Longitude, Latitude: loc.Latitude},
			Distance:  loc.Dist,
		})
	}

	return geoLocations, nil
}

// GeoAdd добавляет геолокации по ключу.
func (c Cache) GeoAdd(ctx context.Context, key string, geoLocation ...cache.GeoLocation) error {
	// Преобразуем cache.GeoLocation в redis.GeoLocation.
	locations := make([]*redis.GeoLocation, 0, len(geoLocation))
	for _, loc := range geoLocation {
		locations = append(locations, &redis.GeoLocation{
			Name:      loc.Member,
			Longitude: loc.Locations.Longitude,
			Latitude:  loc.Locations.Latitude,
		})
	}

	if len(locations) == 0 {
		return nil
	}

	return c.redis.GeoAdd(ctx, key, locations...).Err()
}

// Scan позволяет получать несколько значений по паттерну ключа.
func (c Cache) Scan(ctx context.Context, cursor uint64, match string, count int64) cache.ScanIterator {
	return c.redis.Scan(ctx, cursor, match, count).Iterator()
}

// Eval выполняет скрипт Lua на стороне Redis.
func (c Cache) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return c.redis.Eval(ctx, script, keys, args...).Result()
}

// Clear очищает кэш.
func (c Cache) Clear(ctx context.Context) error {
	return c.redis.FlushAll(ctx).Err()
}

// Ping проверяет что соединение с Redis установлено.
func (c Cache) Ping(ctx context.Context) error {
	return c.redis.Ping(ctx).Err()
}

// Close закрывает соединение с Redis.
func (c Cache) Close(_ context.Context) error {
	return c.redis.Close()
}

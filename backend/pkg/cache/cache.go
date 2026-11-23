package cache

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// NotExistsArgument аргумент SetArgs для режима записи только при отсутствии ключа.
const NotExistsArgument = "NX"

// UniversalClient универсальный клиент к Redis.
type UniversalClient interface {
	// UniversalClient встраивает клиент redis-go.
	redis.UniversalClient
}

// SetArgs представляет собой аргументы для сохранения значения по ключу.
type SetArgs struct {
	KeepTTL  bool          // Оставить предыдущее время жизни который привязан к ключу.
	TTL      time.Duration // От текущего времени установить время жизни ключа.
	ExpireAt time.Time     // Время когда ключ должен истечь.
	Mode     string        // Режим работы.
	Get      bool          // Получить значение.
}

// ZMember представляет собой элемент множества.
type ZMember struct {
	Score  float64     // Количество очков.
	Member interface{} // Элемент множества.
}

// GeoSearchQuery предназначен для поиска геолокаций.
type GeoSearchQuery struct {
	Member      string  // Значение элемента, для которого выполняется поиск.
	Longitude   float64 // Долгота
	Latitude    float64 // Широта
	Radius      float64 // Радиус поиска.
	Unit        string  // Может быть m, km, ft или mi. По умолчанию km.
	WithCoord   bool    // Возвращать ли координаты.
	WithDist    bool    // Возвращать ли расстояние.
	WithGeoHash bool    // Возвращать ли геохеш.
	Count       int     // Максимальное количество результатов для возврата. По умолчанию 10.
	Sort        string  // Может быть ASC или DESC. По умолчанию без сортировки.
	Store       string  // Ключ для сохранения результатов в сортированное множество.
	StoreDist   string  // Ключ для сохранения расстояний в сортированное множество.
}

// GeoLocation представляет собой срез геолокаций, которые возвращаются из GeoRadius.
type GeoLocation struct {
	Member    string   // Значение элемента.
	Locations Position // Список геолокаций.
	Distance  float64  // Расстояние от центра радиуса.
}

// GeoLocations представляет собой срез геолокаций, которые возвращаются из GeoRadius.
type GeoLocations []GeoLocation

// Position представляет собой географическую позицию с координатами.
type Position struct {
	Longitude float64 // Долгота.
	Latitude  float64 // Широта.
}

// Cache представляет собой интерфейс для работы с кэшем.
type Cache interface {
	// Incr увеличивает значение по ключу на 1.
	Incr(ctx context.Context, key string) (int64, error)
	// Decr уменьшает значение по ключу на 1.
	Decr(ctx context.Context, key string) (int64, error)

	// MGet возвращает значения по ключам.
	MGet(ctx context.Context, keys []string) ([]interface{}, error)
	// MSet сохраняет значения по ключам.
	MSet(ctx context.Context, values map[string]interface{}) error

	// Get возвращает значение по ключу.
	Get(ctx context.Context, key string) (string, error)
	// Set сохраняет значение по ключу.
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	// SetArgs сохраняет значение по ключу с определенными аргументами.
	SetArgs(ctx context.Context, key string, value interface{}, args SetArgs) error
	// SetArgsAndReturn сохраняет значение по ключу с определенными аргументами и возвращает результат,
	// если запись по ключу уже существует.
	SetArgsAndReturn(ctx context.Context, key string, value interface{}, args SetArgs) (string, error)
	// Delete удаляет значения по ключам.
	Delete(ctx context.Context, keys ...string) error
	// Exists проверяет существование значения по ключу.
	Exists(ctx context.Context, key string) (bool, error)

	// ZAdd добавляет значения в сортированное множество по ключу.
	ZAdd(ctx context.Context, key string, members ...ZMember) error
	// ZRem удаляет значения из сортированного множества по ключу.
	ZRem(ctx context.Context, key string, members ...interface{}) error
	// ZRemRangeByScore удаляет значения из сортированного множества по ключу в заданном диапазоне.
	ZRemRangeByScore(ctx context.Context, key, minVal, maxVal string) error
	// ZRangeByScore возвращает значения из сортированного множества по ключу в заданном диапазоне.
	ZRangeByScore(ctx context.Context, key, minVal, maxVal string) ([]string, error)

	// GeoSearch возвращает геолокации по поиску.
	GeoSearch(ctx context.Context, key string, query *GeoSearchQuery) (GeoLocations, error)
	// GeoAdd добавляет геолокации в кэш по ключу.
	GeoAdd(ctx context.Context, key string, geoLocation ...GeoLocation) error

	// Keys возвращает все ключи по шаблону.
	Keys(ctx context.Context, pattern string) ([]string, error)
	// Scan позволяет получать несколько значений по паттерну ключа.
	Scan(ctx context.Context, cursor uint64, match string, count int64) ScanIterator
	// Eval выполняет скрипт на стороне кэша.
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)

	// Clear очищает кэш.
	Clear(ctx context.Context) error
	// Ping проверяет что соединение с Redis установлено.
	Ping(ctx context.Context) error
	// Close закрывает соединение с Redis.
	Close(ctx context.Context) error
}

// ScanIterator является итератором по результатам при получении значений из кэша.
type ScanIterator interface {
	// Err возвращает ошибку, которая могла произойти во время итерации.
	// Вызывается после окончания итерации, то есть когда Next() вернет false.
	Err() error
	// Next позволяет проходить по ключам кэша.
	Next(ctx context.Context) bool
	// Val возвращает ключ в рамках текущей итерации.
	Val() string
}

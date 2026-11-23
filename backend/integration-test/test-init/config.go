//go:build integration

package test_init

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/eskermese/machinery-catalog/backend/internal/config"
)

// Config конфигурация интеграционных тестов.
type Config struct {
	config.Server   `yaml:"server"`
	config.Database `yaml:"database"`
	config.Cache    `yaml:"cache"`
	config.Log      `yaml:"logger"`
	Hostnames       `yaml:"hostnames"`
	Postgres        `yaml:"postgres"`
	ServiceName     string `env:"SERVICE_NAME" yaml:"service_name" env-default:"techline" env-description:"Название сервиса"`
	NetworkName     string `env:"NETWORK_NAME" yaml:"network_name" env-default:"techline-network" env-description:"Название сети"`
	BranchName      string `env:"BRANCH_NAME" yaml:"branch_name" env-default:"main" env-description:"Название git ветки"`
	Version         string `env:"APP_VERSION" yaml:"version" env-default:"unknown" env-description:"Версия приложения"`
	GatewayToken    string `env:"GATEWAY_TOKEN" yaml:"gateway_token" env-default:"integration-gateway-token" env-description:"Секрет доверенного шлюза"`
	JWTSecret       string `env:"JWT_SECRET" yaml:"jwt_secret" env-default:"integration-jwt-secret" env-description:"Секрет подписи токенов"`
	AdminPhones     string `env:"ADMIN_PHONES" yaml:"admin_phones" env-default:"+77000000001" env-description:"Телефоны администраторов набора"`
}

// Hostnames хосты сервисов.
type Hostnames struct {
	DSHostname     string `env:"DATASTORE_HOSTNAME" yaml:"database_hostname" env-default:"postgres" env-description:"Хост PostgreSQL"`
	CacheHostname  string `env:"CACHE_HOSTNAME" yaml:"cache_hostname" env-default:"cache" env-description:"Хост кэша"`
	ServerHostname string `env:"SERVER_HOSTNAME" yaml:"server_hostname" env-default:"service" env-description:"Хост HTTP сервиса"`
}

// Postgres параметры контейнера PostgreSQL.
type Postgres struct {
	PostgresUser     string `env:"POSTGRES_USER" yaml:"user" env-default:"postgres" env-description:"Пользователь PostgreSQL"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" yaml:"password" env-default:"postgres" env-description:"Пароль PostgreSQL"`
	PostgresDB       string `env:"POSTGRES_DB" yaml:"db" env-default:"techline" env-description:"База данных PostgreSQL"`
}

// generateBranchName генерирует имя ветки для названий контейнеров.
func (c *Config) generateBranchName() {
	c.BranchName = strings.ToLower(c.BranchName)

	if c.BranchName == "" {
		c.BranchName = "main"
	}

	randomNum := uint8(rand.Uint32())
	c.BranchName = fmt.Sprintf("%d-%s", randomNum, c.BranchName)
}

// scopeHostnames делает сетевые имена контейнеров набора уникальными.
//
// Без этого набор и стенд делили имена: контейнеры набора поднимались
// с хостами postgres, redis и service в той же сети techline-network, где
// эти имена уже заняты сервисами стенда. Кто ответит на DNS — вопрос
// удачи, и часть запросов набора уходила в базу стенда: после прогона там
// оставались 36 карточек, 22 категории и лишние пользователи.
//
// Имена привязаны к BranchName, в котором уже есть случайный префикс,
// поэтому два одновременных прогона тоже не столкнутся.
//
// Адреса подключения пересобираются из этих же имён, а не читаются из yml:
// иначе имя контейнера и адрес, по которому к нему стучатся, разъехались бы
// при первой же правке.
func (c *Config) scopeHostnames() {
	c.DSHostname = "postgres-" + c.BranchName
	c.CacheHostname = "cache-" + c.BranchName
	c.ServerHostname = "service-" + c.BranchName

	c.DSURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword, c.DSHostname, postgresPort, c.PostgresDB)
	c.CacheAddr = fmt.Sprintf("%s:%d", c.CacheHostname, cachePort)
}

// Порты внутри контейнеров: наружу набор не публикует ничего, вся связь
// идёт по сети докера.
const (
	postgresPort = 5432
	cachePort    = 6379
)

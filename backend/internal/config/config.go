package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	productionEnvironment = "production" // Production окружение
	localEnvironment      = "local"      // Local окружение
)

// ErrJWTSecretRequired сообщает, что сервис запущен без секрета подписи токенов.
var ErrJWTSecretRequired = errors.New("JWT_SECRET обязателен вне окружения local")

type (
	// Config конфигурация приложения.
	Config struct {
		Server      `yaml:"server"`
		Log         `yaml:"logger"`
		Database    `yaml:"database"`
		Cache       `yaml:"cache"`
		Pyroscope   `yaml:"pyroscope"`
		Clients     `yaml:"clients"`
		Auth        `yaml:"auth"`
		FX          `yaml:"fx"`
		Outbox      `yaml:"outbox"`
		Environment `yaml:"environment"`
		ServiceName string `env:"SERVICE_NAME" yaml:"service_name" env-default:"techline" env-description:"Название сервиса"`
		Version     string `env:"APP_VERSION" yaml:"version" env-default:"unknown" env-description:"Версия приложения"`
	}

	// Auth конфигурация доверия к шлюзу.
	// Идентификация пользователя живет за шлюзом, сервис доверяет его заголовкам
	// только при совпадении общего секрета.
	Auth struct {
		GatewayToken string `env:"GATEWAY_TOKEN" yaml:"gateway_token" env-description:"Общий секрет доверенного шлюза (только отладочный режим)"`
		JWTSecret    string `env:"JWT_SECRET" yaml:"jwt_secret" env-description:"Секрет подписи токенов доступа"`
		AdminPhones  string `env:"ADMIN_PHONES" yaml:"admin_phones" env-description:"Телефоны администраторов через запятую"`
		//nolint:lll // описание тега не переносится
		DebugHeaderAuth bool `env:"DEBUG_HEADER_AUTH" yaml:"debug_header_auth" env-description:"Разрешить вход по заголовкам вместо токена. Работает только в окружении local"`

		// SelfRegistration разрешает менеджеру завести учётную запись самим фактом
		// входа: ручки «создать сотрудника» в API нет, и до сих пор это был
		// единственный способ появиться в системе.
		//
		// Обратная сторона: пускает ЛЮБОЙ номер. Пока сервис доступен только
		// изнутри, это допустимо; как только он открыт наружу — нет, потому что
		// менеджер видит заявки с именами и телефонами клиентов.
		//
		// По умолчанию включено: выключение здесь не должно молча запереть
		// сотрудников там, где сервис и так закрыт.
		SelfRegistration bool `env:"AUTH_SELF_REGISTRATION" yaml:"self_registration" env-default:"true" env-description:"Разрешить менеджеру завести учётную запись первым входом. Выключать, когда сервис доступен снаружи"`
	}

	// FX конфигурация курсов валют.
	FX struct {
		//nolint:lll // описание тега не переносится
		FXSyncEnabled  bool          `env:"FX_SYNC_ENABLED" yaml:"sync_enabled" env-default:"false" env-description:"Включить фоновую синхронизацию курса"`
		FXSyncInterval time.Duration `env:"FX_SYNC_INTERVAL" yaml:"sync_interval" env-default:"1h" env-description:"Интервал фоновой синхронизации курса"`
		// ВАЛЮТ ЗДЕСЬ НЕТ, И ЭТО РЕШЕНИЕ.
		//
		// Раньше пара воркера задавалась переменными окружения. Настройкой
		// она могла разойтись с парой, в которой сервис считает цену:
		// воркер тянул бы курс EUR→KZT, а витрина спрашивала бы CNY→KZT
		// и молчала про тенге при исправно работающей синхронизации.
		// Пара объявлена в домене (entity.CurrencyCNY и CurrencyKZT) — там,
		// где ей и место: сменить валюту прайса без миграции цен всё равно
		// нельзя, а значит это не настройка развёртывания.
		//nolint:lll // описание тега не переносится
		FXProviderURL string `env:"FX_PROVIDER_URL" yaml:"provider_url" env-default:"https://nationalbank.kz" env-description:"Адрес источника курса"`
		//nolint:lll // описание тега не переносится
		FXProviderTimeout time.Duration `env:"FX_PROVIDER_TIMEOUT" yaml:"provider_timeout" env-default:"10s" env-description:"Таймаут запроса к источнику курса"`

		// FXRateStaleAfter через сколько суток курс считается устаревшим.
		//
		// Трое суток по календарю источника: Нацбанк публикует курс
		// по рабочим дням, и пятничный курс законно действует в субботу
		// и воскресенье — к утру понедельника ему ровно три дня. Меньше —
		// и админка кричала бы каждый понедельник; больше — и забытый
		// на неделю курс выглядел бы нормальным.
		//
		// Праздничный понедельник добавляет сутки, поэтому это настройка,
		// а не константа.
		FXRateStaleAfterDays int `env:"FX_RATE_STALE_AFTER_DAYS" yaml:"rate_stale_after_days" env-default:"3" env-description:"Через сколько суток курс считается устаревшим"`
	}

	// Outbox конфигурация доставки уведомлений.
	Outbox struct {
		//nolint:lll // описание тега не переносится
		OutboxSyncEnabled  bool          `env:"OUTBOX_SYNC_ENABLED" yaml:"sync_enabled" env-default:"false" env-description:"Включить отправку уведомлений из очереди"`
		OutboxSyncInterval time.Duration `env:"OUTBOX_SYNC_INTERVAL" yaml:"sync_interval" env-default:"1m" env-description:"Интервал разбора очереди уведомлений"`
		OutboxBatchSize    int           `env:"OUTBOX_BATCH_SIZE" yaml:"batch_size" env-default:"50" env-description:"Сколько уведомлений забирать за проход"`
	}

	// Environment окружение.
	Environment struct {
		Name string `env:"ENVIRONMENT_NAME" yaml:"name" env-default:"production" env-description:"Название окружения"`
	}

	// Server сервер.
	Server struct {
		Host           string `env:"SERVER_HOST" yaml:"host" env-default:"0.0.0.0" env-description:"Хост HTTP сервиса"`
		HTTPListenAddr int    `env:"SERVER_PORT" yaml:"http_listen_addr" env-default:"8000" env-description:"Адрес HTTP сервера"`
		BasePath       string `env:"BASE_PATH" yaml:"base_path" env-default:"/" env-description:"Базовый путь сервиса"`
		FilesDir       string `env:"FILES_DIR" yaml:"files_dir" env-default:"/swagger" env-description:"Директория с файлами"`
		// MediaDir каталог с загруженными изображениями каталога техники.
		//
		// Отдельно от FilesDir: тот указывает на каталог со сгенерированной
		// спецификацией и раздаётся только вне production. Смешивать их
		// значило бы отдавать наружу служебные файлы вместе с фотографиями.
		MediaDir string `env:"MEDIA_DIR" yaml:"media_dir" env-default:"./media" env-description:"Каталог с изображениями каталога техники"`
	}

	// Log логирование.
	Log struct {
		LogLevel string `env:"LOG_LEVEL" yaml:"level" env-default:"info" env-description:"Уровень логирования"`
	}

	// Database конфигурация PostgreSQL.
	Database struct {
		DSURL         string        `env:"DATABASE_URL" yaml:"url" env-required:"true" env-description:"DSN PostgreSQL (формат: postgres://user:pass@host:5432/db?sslmode=disable)"`
		MaxConns      int32         `env:"DATABASE_MAX_CONNS" yaml:"max_conns" env-default:"10" env-description:"Максимальный размер пула соединений"`
		MinConns      int32         `env:"DATABASE_MIN_CONNS" yaml:"min_conns" env-default:"0" env-description:"Минимальный размер пула соединений"`
		ConnectTimout time.Duration `env:"DATABASE_CONNECT_TIMEOUT" yaml:"connect_timeout" env-default:"5s" env-description:"Таймаут подключения к БД"`
	}

	// Cache конфигурация кэша.
	Cache struct {
		CacheAddr     string `env:"CACHE_ADDR" yaml:"addr" env-description:"Адрес кэша"`
		CacheUsername string `env:"CACHE_USERNAME" yaml:"username" env-description:"Имя пользователя кэша"`
		CachePassword string `env:"CACHE_PASSWORD" yaml:"password" env-description:"Пароль кэша"`
	}

	// Pyroscope конфигурация профилирования.
	Pyroscope struct {
		PyroscopeAddr string `env:"PYROSCOPE_SERVER_ADDRESS" yaml:"addr" env-description:"Адрес сервера Pyroscope"`
	}

	// Clients конфигурация клиентов.
	Clients struct {
		SettingsBaseURL     string        `env:"SETTINGS_BASE_URL" yaml:"settingsBaseUrl" env-description:"Базовый адрес сервиса настроек"`
		GetSettingsURI      string        `env:"GET_SETTINGS_URI" yaml:"getSettingsUri" env-description:"URI получения настроек"`
		SettingsHTTPTimeout time.Duration `env:"SETTINGS_CLIENT_TIMEOUT" yaml:"settings_http_timeout" env-default:"5s" env-description:"Таймаут при получении ответа от сервиса настроек"`
	}
)

// NewConfig - конструктор конфига.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("./config/config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("парсинг config.yml: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, fmt.Errorf("парсинг переменных окружения: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("проверка конфигурации: %w", err)
	}

	return cfg, nil
}

// GetHTTPDomain возвращает домен для HTTP сервера.
func (s Server) GetHTTPDomain() string {
	return fmt.Sprintf("%s:%d", s.Host, s.HTTPListenAddr)
}

// AdminPhoneList возвращает телефоны администраторов из конфигурации.
func (a Auth) AdminPhoneList() []string {
	phones := make([]string, 0)

	for _, phone := range strings.Split(a.AdminPhones, ",") {
		if trimmed := strings.TrimSpace(phone); trimmed != "" {
			phones = append(phones, trimmed)
		}
	}

	return phones
}

// IsHeaderAuthAllowed сообщает, что разрешён отладочный вход по заголовкам.
// Двойное условие намеренно: одного флага в конфигурации мало, вне local
// заголовочная схема недоступна ни при какой настройке.
func (c *Config) IsHeaderAuthAllowed() bool {
	return c.IsLocal() && c.DebugHeaderAuth
}

// Validate проверяет пригодность конфигурации к запуску.
// Пустой секрет подписи вне local — фатальная ошибка: сервис, который не может
// проверить ни один токен, обязан не стартовать, а не отвечать 401 на всё подряд.
func (c *Config) Validate() error {
	if c.JWTSecret == "" && !c.IsLocal() {
		return ErrJWTSecretRequired
	}

	return nil
}

// IsProduction является ли прод окружением.
func (e *Environment) IsProduction() bool {
	return e.Name == productionEnvironment
}

// IsLocal является ли локальным окружением.
func (e *Environment) IsLocal() bool {
	return e.Name == localEnvironment
}

// HasPyroscopeAddr проверяет задан ли адрес pyroscope.
func (p Pyroscope) HasPyroscopeAddr() bool {
	return p.PyroscopeAddr != ""
}

// IsFXSyncEnabled включена ли фоновая синхронизация курса.
//
// Двойное условие по образцу IsPyroscopeEnabled: одного флага мало — нулевой
// или отрицательный интервал означал бы тикер, срабатывающий непрерывно,
// поэтому такая конфигурация считается выключенной, а не бесконечным циклом.
func (c *Config) IsFXSyncEnabled() bool {
	if c == nil {
		return false
	}

	return c.FXSyncEnabled && c.FXSyncInterval > 0
}

// IsOutboxSyncEnabled включена ли отправка уведомлений из очереди.
//
// Двойное условие по образцу IsFXSyncEnabled: нулевой интервал означал бы
// тикер, срабатывающий непрерывно.
func (c *Config) IsOutboxSyncEnabled() bool {
	if c == nil {
		return false
	}

	return c.OutboxSyncEnabled && c.OutboxSyncInterval > 0
}

// IsPyroscopeEnabled включен ли pyroscope.
func (c *Config) IsPyroscopeEnabled() bool {
	if c == nil {
		return false
	}

	return c.IsLocal() && c.HasPyroscopeAddr()
}

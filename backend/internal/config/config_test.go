package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment_IsProduction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  string
		exp  bool
	}{
		{
			name: "production",
			env:  productionEnvironment,
			exp:  true,
		},
		{
			name: "empty",
			env:  "",
			exp:  false,
		},
		{
			name: "unknown",
			env:  "unknown",
			exp:  false,
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			e := Environment{Name: s.env}
			assert.Equal(t, s.exp, e.IsProduction())
		})
	}
}

func TestEnvironment_IsLocal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  string
		exp  bool
	}{
		{
			name: "local",
			env:  localEnvironment,
			exp:  true,
		},
		{
			name: "empty",
			env:  "",
			exp:  false,
		},
		{
			name: "unknown",
			env:  "unknown",
			exp:  false,
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			e := Environment{Name: s.env}
			assert.Equal(t, s.exp, e.IsLocal())
		})
	}
}

func TestServer_GetHTTPDomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		host string
		port int
		exp  string
	}{
		{
			name: "default",
			host: "localhost",
			port: 8080,
			exp:  "localhost:8080",
		},
		{
			name: "empty",
			host: "",
			port: 0,
			exp:  ":0",
		},
		{
			name: "empty host",
			host: "",
			port: 8080,
			exp:  ":8080",
		},
		{
			name: "empty port",
			host: "localhost",
			port: 0,
			exp:  "localhost:0",
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			srv := Server{Host: s.host, HTTPListenAddr: s.port}
			assert.Equal(t, s.exp, srv.GetHTTPDomain())
		})
	}
}

func TestPyroscope_HasPyroscopeAddr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		addr string
		exp  bool
	}{
		{
			name: "Пустой адрес",
			addr: "",
			exp:  false,
		},
		{
			name: "Заполненный адрес",
			addr: "localhost:4040",
			exp:  true,
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			p := Pyroscope{PyroscopeAddr: s.addr}
			assert.Equal(t, s.exp, p.HasPyroscopeAddr())
		})
	}
}

func TestConfig_IsPyroscopeEnabled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  string
		addr string
		exp  bool
	}{
		{
			name: "Local окружение с адресом",
			env:  localEnvironment,
			addr: "localhost:4040",
			exp:  true,
		},
		{
			name: "Local окружение без адреса",
			env:  localEnvironment,
			addr: "",
			exp:  false,
		},
		{
			name: "Development окружение с адресом",
			env:  "development",
			addr: "localhost:4040",
			exp:  false,
		},
		{
			name: "Development окружение без адреса",
			env:  "development",
			addr: "",
			exp:  false,
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			c := Config{
				Environment: Environment{Name: s.env},
				Pyroscope:   Pyroscope{PyroscopeAddr: s.addr},
			}

			assert.Equal(t, s.exp, c.IsPyroscopeEnabled())
		})
	}
}

// TestConfig_Validate фиксирует fail-fast: вне local сервис не должен стартовать
// без секрета подписи. Сервис, который не может проверить ни один токен, обязан
// не подняться, а не отвечать 401 на всё прикладное API при зелёном health-check.
func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		env    string
		secret string
		expErr error
	}{
		{name: "прод без секрета не стартует", env: "production", expErr: ErrJWTSecretRequired},
		{name: "прод с секретом стартует", env: "production", secret: "s3cret"},
		{name: "неизвестное окружение считается не-local", env: "staging", expErr: ErrJWTSecretRequired},
		{name: "local без секрета стартует", env: "local"},
		{name: "local с секретом стартует", env: "local", secret: "s3cret"},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}
			cfg.Name = s.env
			cfg.JWTSecret = s.secret

			if s.expErr != nil {
				require.ErrorIs(t, cfg.Validate(), s.expErr)

				return
			}

			require.NoError(t, cfg.Validate())
		})
	}
}

// TestConfig_IsHeaderAuthAllowed фиксирует, что отладочный вход по заголовкам
// невозможен вне local ни при какой конфигурации: одного флага недостаточно.
func TestConfig_IsHeaderAuthAllowed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		env   string
		debug bool
		exp   bool
	}{
		{name: "local и флаг включён", env: "local", debug: true, exp: true},
		{name: "local без флага", env: "local"},
		{name: "прод с флагом", env: "production", debug: true},
		{name: "прод без флага", env: "production"},
		{name: "staging с флагом", env: "staging", debug: true},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}
			cfg.Name = s.env
			cfg.DebugHeaderAuth = s.debug

			assert.Equal(t, s.exp, cfg.IsHeaderAuthAllowed())
		})
	}
}

// TestAuth_AdminPhoneList проверяет разбор списка телефонов из переменной окружения.
func TestAuth_AdminPhoneList(t *testing.T) {
	t.Parallel()

	auth := Auth{AdminPhones: " +77001112233 , 87002223344 ,, "}

	assert.Equal(t, []string{"+77001112233", "87002223344"}, auth.AdminPhoneList())
	assert.Empty(t, Auth{}.AdminPhoneList())
}

// TestConfig_IsFXSyncEnabled фиксирует двойное условие включения воркера курса.
//
// Одного флага мало: нулевой или отрицательный интервал означал бы тикер,
// срабатывающий непрерывно, поэтому такая конфигурация считается выключенной,
// а не бесконечным циклом.
func TestConfig_IsFXSyncEnabled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		enabled  bool
		interval time.Duration
		exp      bool
	}{
		{name: "включён с положительным интервалом", enabled: true, interval: time.Hour, exp: true},
		{name: "выключен флагом", enabled: false, interval: time.Hour, exp: false},
		{name: "включён, но интервал нулевой", enabled: true, interval: 0, exp: false},
		{name: "включён, но интервал отрицательный", enabled: true, interval: -time.Second, exp: false},
		{name: "выключен и без интервала", enabled: false, interval: 0, exp: false},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			c := &Config{}
			c.FXSyncEnabled = s.enabled
			c.FXSyncInterval = s.interval

			assert.Equal(t, s.exp, c.IsFXSyncEnabled())
		})
	}
}

// TestConfig_IsFXSyncEnabled_Nil фиксирует, что пустая конфигурация не паникует.
func TestConfig_IsFXSyncEnabled_Nil(t *testing.T) {
	t.Parallel()

	var c *Config

	assert.False(t, c.IsFXSyncEnabled())
}

// TestConfig_IsOutboxSyncEnabled фиксирует двойное условие включения отправщика
// уведомлений: по образцу IsFXSyncEnabled нулевой или отрицательный интервал
// считается выключенным воркером, а не тикером без паузы.
func TestConfig_IsOutboxSyncEnabled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		enabled  bool
		interval time.Duration
		exp      bool
	}{
		{name: "включён с положительным интервалом", enabled: true, interval: time.Minute, exp: true},
		{name: "выключен флагом", enabled: false, interval: time.Minute, exp: false},
		{name: "включён, но интервал нулевой", enabled: true, interval: 0, exp: false},
		{name: "включён, но интервал отрицательный", enabled: true, interval: -time.Second, exp: false},
		{name: "выключен и без интервала", enabled: false, interval: 0, exp: false},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			c := &Config{}
			c.OutboxSyncEnabled = s.enabled
			c.OutboxSyncInterval = s.interval

			assert.Equal(t, s.exp, c.IsOutboxSyncEnabled())
		})
	}
}

// TestConfig_IsOutboxSyncEnabled_Nil фиксирует, что пустая конфигурация не паникует.
func TestConfig_IsOutboxSyncEnabled_Nil(t *testing.T) {
	t.Parallel()

	var c *Config

	assert.False(t, c.IsOutboxSyncEnabled())
}

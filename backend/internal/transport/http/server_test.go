package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/auth"
	"github.com/eskermese/machinery-catalog/backend/internal/config"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/form"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
)

// testJWTSecret секрет подписи токенов в тестах роутера.
const testJWTSecret = "s3cret-token"

// testTokens выпускает токены для тестов роутера.
var testTokens = auth.NewTokenService(testJWTSecret, auth.TokenTTL)

// testRouter собирает роутер один раз на весь пакет.
// Повторная сборка невозможна: metrics.NewMiddlewareMetric регистрирует
// коллекторы в глобальном реестре Prometheus и падает на дублирующей регистрации.
// Роутер только читается, поэтому параллельным подтестам его можно делить.
var testRouter = sync.OnceValue(func() chi.Router {
	return newTestServer().setupRouter()
})

// testMediaDir создаёт каталог с изображением один раз на пакет.
//
// Не t.TempDir(): роутер тоже строится один раз — setupRouter регистрирует
// коллекторы Prometheus, и второй вызов уронил бы процесс на повторной
// регистрации. Значит, и каталог обязан пережить отдельный тест.
var testMediaDir = sync.OnceValue(func() string {
	dir, err := os.MkdirTemp("", "techline-media")
	if err != nil {
		panic(err)
	}

	if err = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("картинка"), 0o600); err != nil {
		panic(err)
	}

	return dir
})

// newTestServer собирает сервер с включенной проверкой шлюза.
// Сервисы не заданы: тесты проверяют только то, что происходит до обработчиков.
func newTestServer() *Server {
	return &Server{
		Environment:     config.Environment{Name: "local"},
		MediaDir:        testMediaDir(),
		version:         "test",
		serviceName:     "techline",
		tokens:          testTokens,
		idleConnsClosed: make(chan struct{}),
	}
}

// TestRouter_ServiceRoutesStayOpen фиксирует, что инфраструктурные маршруты не
// закрыты проверкой шлюза и роли: их дергают Prometheus и деплой, а не пользователь.
func TestRouter_ServiceRoutesStayOpen(t *testing.T) {
	t.Parallel()

	router := testRouter()

	for _, path := range []string{"/metrics", "/version"} {
		path := path

		t.Run(path, func(t *testing.T) {
			t.Parallel()

			// Ни секрета шлюза, ни роли — заголовков нет вовсе.
			r := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestRouter_AuthRoutesOpen фиксирует, что выпуск токена доступен без токена:
// иначе войти было бы невозможно.
func TestRouter_AuthRoutesOpen(t *testing.T) {
	t.Parallel()

	router := testRouter()

	for _, path := range []string{
		"/api/v1/auth/manager/token",
		"/api/v1/auth/admin/token",
	} {
		path := path

		t.Run(path, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"phone":"+77001112233"}`))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			// Сервис входа в тестовом сервере не задан, поэтому до него доходить
			// нельзя — важно лишь то, что маршрут не отсекается аутентификацией.
			assert.NotEqual(t, http.StatusUnauthorized, w.Code, "выпуск токена не должен требовать токен")
		})
	}
}

// TestServer_Validate фиксирует отказ на старте при неподключённых зависимостях.
//
// Регрессия: опции WithAuthService и WithTokenParser забыли передать в wiring,
// и сервер поднимался, а падал уже на первом запросе к защищённому маршруту.
// Отказ при старте честнее и заметнее.
func TestServer_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		server *Server
		expErr error
	}{
		{
			name:   "сервис выпуска токенов не задан",
			server: &Server{tokens: testTokens},
			expErr: ErrAuthServiceMissing,
		},
		{
			name:   "проверка токенов не задана",
			server: &Server{authService: stubAuthService{}},
			expErr: ErrTokenParserMissing,
		},
		{
			// В отладочном режиме токены не разбираются, парсер не нужен.
			name:   "отладочный режим без парсера допустим",
			server: &Server{authService: stubAuthService{}, headerAuth: true},
		},
		{
			name:   "всё подключено",
			server: &Server{authService: stubAuthService{}, tokens: testTokens},
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			if s.expErr != nil {
				require.ErrorIs(t, s.server.validate(), s.expErr)

				return
			}

			require.NoError(t, s.server.validate())
		})
	}
}

// stubAuthService — минимальная реализация сервиса выпуска токенов для проверки сборки.
type stubAuthService struct{}

// IssueToken не используется: тест проверяет только сборку сервера.
func (stubAuthService) IssueToken(
	_ context.Context, _ form.AuthTokenIssue, _ time.Time,
) (presenter.AuthToken, error) {
	return presenter.AuthToken{}, nil
}

// EnsureAdmins не используется: тест проверяет только сборку сервера.
func (stubAuthService) EnsureAdmins(_ context.Context, _ []string, _ time.Time) error {
	return nil
}

// CurrentUser не используется: тест проверяет только сборку сервера.
func (stubAuthService) CurrentUser(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}

// ListStaff не используется: тест проверяет только сборку сервера.
func (stubAuthService) ListStaff(_ context.Context, _ entity.ActorType) (entity.Users, error) {
	return nil, nil
}

// RenameSelf не используется: тест проверяет только сборку сервера.
func (stubAuthService) RenameSelf(_ context.Context, _, _ string) (*entity.User, error) {
	return nil, nil
}

// TestRouter_MediaRouteNeedsNoAuth фиксирует, что изображения раздаёт сам
// сервис и без аутентификации: фотографии техники — часть публичной витрины,
// а путь внутри хранилища клиенту ничего не даёт.
func TestRouter_MediaRouteNeedsNoAuth(t *testing.T) {
	t.Parallel()

	// Ни токена, ни секрета шлюза.
	r := httptest.NewRequest(http.MethodGet, presenter.MediaRoutePrefix+"/a.txt", nil)
	w := httptest.NewRecorder()

	testRouter().ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "картинка", w.Body.String())
}

// TestRouter_MediaRouteRejectsEscape фиксирует, что из каталога изображений
// нельзя выйти вверх: путь приходит из базы, но в запросе его подменяет клиент.
func TestRouter_MediaRouteRejectsEscape(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, presenter.MediaRoutePrefix+"/../../etc/passwd", nil)
	w := httptest.NewRecorder()

	testRouter().ServeHTTP(w, r)

	assert.NotEqual(t, http.StatusOK, w.Code, "выход за пределы каталога не отдаёт файл")
}

// TestOptions фиксирует, что каждая опция кладёт свою зависимость в сервер.
//
// Сервер собирается из необязательных частей — витрина без сервиса курсов,
// админка без сервиса заявок, — поэтому опции независимы и применяются
// по одной.
func TestOptions(t *testing.T) {
	t.Parallel()

	srv := &Server{}

	catalogService := service.CatalogService(&stubCatalog{})
	fxService := service.FXService(&stubFX{})
	quoteService := service.QuoteService(&stubQuote{})
	leadService := service.LeadService(&stubLead{})
	authService := service.AuthService(&stubAuth{})
	tcoService := service.TCOService(&stubTCO{})

	WithCatalogService(catalogService)(srv)
	WithFXService(fxService)(srv)
	WithQuoteService(quoteService)(srv)
	WithLeadService(leadService)(srv)
	WithTCOService(tcoService)(srv)
	WithAuthService(authService)(srv)
	WithTokenParser(testTokens)(srv)

	assert.Equal(t, catalogService, srv.catalogService)
	assert.Equal(t, fxService, srv.fxService)
	assert.Equal(t, quoteService, srv.quoteService)
	assert.Equal(t, leadService, srv.leadService)
	assert.Equal(t, tcoService, srv.tcoService)
	assert.Equal(t, authService, srv.authService)
	assert.Equal(t, testTokens, srv.tokens)
}

// Пустые заглушки сервисов: опциям важно лишь то, что значение доехало
// до поля, а не то, что оно умеет. По одной на интерфейс — общая заглушка
// не собирается, потому что у сервисов есть одноимённые методы.
type (
	stubCatalog struct{ service.CatalogService }
	stubFX      struct{ service.FXService }
	stubQuote   struct{ service.QuoteService }
	stubLead    struct{ service.LeadService }
	stubAuth    struct{ service.AuthService }
)

// stubTCO заглушка сервиса стоимости владения: сервер его только хранит.
type stubTCO struct{ service.TCOService }

package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/auth"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// testActorUUID идентификатор действующего пользователя в тестах middleware.
const testActorUUID = "4a2f0b0e-6f0a-4b6e-9a4e-0c1d2e3f4a5b"

// passedHandler отмечает, что запрос дошел до обработчика за middleware.
func passedHandler(passed *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*passed = true

		w.WriteHeader(http.StatusOK)
	})
}

// actorCapturingHandler запоминает актора, которого middleware положил в контекст.
func actorCapturingHandler(captured *entity.Actor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = actorFromContext(r.Context())

		w.WriteHeader(http.StatusOK)
	})
}

// TestAuthMiddleware фиксирует аутентификацию по токену: кто пришёл, определяется
// подписью, а не заголовками с идентификатором.
func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	tokens := auth.NewTokenService("test-secret", auth.TokenTTL)

	valid, err := tokens.Issue(testActorUUID, entity.ActorTypeManager, time.Now().UTC())
	require.NoError(t, err)

	expired, err := auth.NewTokenService("test-secret", time.Hour).
		Issue(testActorUUID, entity.ActorTypeManager, time.Now().UTC().Add(-2*time.Hour))
	require.NoError(t, err)

	foreign, err := auth.NewTokenService("другой-секрет", auth.TokenTTL).
		Issue(testActorUUID, entity.ActorTypeManager, time.Now().UTC())
	require.NoError(t, err)

	cases := []struct {
		name       string
		authHeader string
		expStatus  int
		expCode    string
	}{
		{name: "валидный токен", authHeader: "Bearer " + valid, expStatus: http.StatusOK},
		{name: "регистр схемы не важен", authHeader: "bearer " + valid, expStatus: http.StatusOK},
		{
			name: "заголовка нет", expStatus: http.StatusUnauthorized, expCode: "AUTH_REQUIRED",
		},
		{
			name: "схема не Bearer", authHeader: "Basic " + valid,
			expStatus: http.StatusUnauthorized, expCode: "AUTH_REQUIRED",
		},
		{
			name: "чужая подпись", authHeader: "Bearer " + foreign,
			expStatus: http.StatusUnauthorized, expCode: "AUTH_REQUIRED",
		},
		{
			// Отдельный код: клиенту нужно понять, что достаточно перелогиниться.
			name: "истёкший токен", authHeader: "Bearer " + expired,
			expStatus: http.StatusUnauthorized, expCode: "TOKEN_EXPIRED",
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			var captured entity.Actor

			handler := AuthMiddleware(tokens)(actorCapturingHandler(&captured))

			r := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
			if s.authHeader != "" {
				r.Header.Set("Authorization", s.authHeader)
			}

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			require.Equal(t, s.expStatus, w.Code)

			if s.expStatus != http.StatusOK {
				assert.Contains(t, w.Body.String(), entity.CodePrefix+s.expCode)

				return
			}

			require.NotNil(t, captured.ID, "актор кладётся в контекст")
			assert.Equal(t, testActorUUID, *captured.ID)
			assert.Equal(t, entity.ActorTypeManager, captured.Type, "роль берётся из токена")
		})
	}
}

// TestAuthMiddleware_IgnoresIdentityHeaders фиксирует, что продовая схема не верит
// заголовкам X-User-Id и X-User-Role: подделать актора ими нельзя.
func TestAuthMiddleware_IgnoresIdentityHeaders(t *testing.T) {
	t.Parallel()

	tokens := auth.NewTokenService("test-secret", auth.TokenTTL)

	token, err := tokens.Issue(testActorUUID, entity.ActorTypeManager, time.Now().UTC())
	require.NoError(t, err)

	var captured entity.Actor

	handler := AuthMiddleware(tokens)(actorCapturingHandler(&captured))

	r := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set(HeaderXUserRole, "admin")
	r.Header.Set(HeaderXUserID, "00000000-0000-4000-8000-000000000000")

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, entity.ActorTypeManager, captured.Type, "роль только из токена")
	assert.Equal(t, testActorUUID, *captured.ID, "идентификатор только из токена")
}

// TestDebugHeaderAuthMiddleware фиксирует отладочную схему прежнего образца.
// К роутеру она подключается только в local при явном DEBUG_HEADER_AUTH — см.
// TestConfig_IsHeaderAuthAllowed и Server.authMiddleware.
func TestDebugHeaderAuthMiddleware(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		role      string
		userID    string
		expStatus int
	}{
		{name: "роль и идентификатор переданы", role: "admin", userID: testActorUUID, expStatus: http.StatusOK},
		{name: "роли нет", userID: testActorUUID, expStatus: http.StatusUnauthorized},
		{name: "неизвестная роль", role: "superuser", userID: testActorUUID, expStatus: http.StatusUnauthorized},
		{name: "идентификатор не UUID", role: "admin", userID: "not-a-uuid", expStatus: http.StatusUnauthorized},
		{name: "идентификатора нет", role: "admin", expStatus: http.StatusUnauthorized},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			var captured entity.Actor

			handler := DebugHeaderAuthMiddleware()(actorCapturingHandler(&captured))

			r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/catalog", nil)
			if s.role != "" {
				r.Header.Set(HeaderXUserRole, s.role)
			}

			if s.userID != "" {
				r.Header.Set(HeaderXUserID, s.userID)
			}

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			require.Equal(t, s.expStatus, w.Code)

			if s.expStatus == http.StatusOK {
				require.NotNil(t, captured.ID)
				assert.Equal(t, entity.ActorTypeAdmin, captured.Type)
			}
		})
	}
}

// TestGatewayMiddleware фиксирует, что отладочная схема пропускает только запросы
// с общим секретом шлюза, а вне local отсутствие секрета закрывает вход, а не открывает.
func TestGatewayMiddleware(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		token     string
		local     bool
		header    string
		expPassed bool
	}{
		{name: "пустой секрет в local выключает проверку", local: true, expPassed: true},
		{name: "пустой секрет вне local закрывает вход", local: false, expPassed: false},
		{name: "верный секрет", token: "s3cret", header: "s3cret", expPassed: true},
		{name: "неверный секрет", token: "s3cret", header: "другой", expPassed: false},
		{name: "секрет не передан", token: "s3cret", expPassed: false},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			var passed bool

			handler := GatewayMiddleware(s.token, s.local)(passedHandler(&passed))

			r := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
			if s.header != "" {
				r.Header.Set(HeaderXGatewayToken, s.header)
			}

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			assert.Equal(t, s.expPassed, passed)

			if !s.expPassed {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			}
		})
	}
}

// TestRoleMiddleware фиксирует разделение 401 и 403: не аутентифицирован — одно,
// аутентифицирован не в свой раздел — другое.
func TestRoleMiddleware(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		actor     entity.Actor
		expStatus int
		expCode   string
	}{
		{
			name: "роль совпадает", actor: entity.NewActor(testActorUUID, entity.ActorTypeAdmin),
			expStatus: http.StatusOK,
		},
		{
			name: "чужой раздел", actor: entity.NewActor(testActorUUID, entity.ActorTypeManager),
			expStatus: http.StatusForbidden, expCode: "ROLE_FORBIDDEN",
		},
		{
			name: "актора нет", expStatus: http.StatusUnauthorized, expCode: "AUTH_REQUIRED",
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			var passed bool

			handler := RoleMiddleware(entity.ActorTypeAdmin)(passedHandler(&passed))

			r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/catalog", nil)
			r = r.WithContext(withActor(r.Context(), s.actor))

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			require.Equal(t, s.expStatus, w.Code)

			if s.expStatus != http.StatusOK {
				assert.Contains(t, w.Body.String(), entity.CodePrefix+s.expCode)

				return
			}

			assert.True(t, passed)
		})
	}
}

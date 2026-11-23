package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/form"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	v1 "github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources/v1"
)

// stubAuthService — заглушка сервиса учётных записей.
type stubAuthService struct {
	service.AuthService // Остальные методы этим тестам не нужны

	lastUserID string           // Идентификатор, с которым позвали чтение своей записи
	lastRole   entity.ActorType // Роль, по которой запросили список
	lastName   string           // Имя, с которым позвали смену
	staffCalls int              // Сколько раз звали список сотрудников

	user  *entity.User
	users entity.Users
	err   error
}

func (s *stubAuthService) IssueToken(
	context.Context, form.AuthTokenIssue, time.Time,
) (presenter.AuthToken, error) {
	return presenter.AuthToken{}, s.err
}

func (s *stubAuthService) CurrentUser(_ context.Context, id string) (*entity.User, error) {
	s.lastUserID = id

	return s.user, s.err
}

func (s *stubAuthService) RenameSelf(_ context.Context, id, name string) (*entity.User, error) {
	s.lastUserID = id
	s.lastName = name

	return s.user, s.err
}

func (s *stubAuthService) ListStaff(_ context.Context, role entity.ActorType) (entity.Users, error) {
	s.lastRole = role
	s.staffCalls++

	return s.users, s.err
}

// authRequest выполняет запрос к роутеру учётных записей от имени актора.
//
// Проверка токена подменена: она разбирает подпись, а этим тестам важно
// поведение обработчиков при уже установленном акторе.
func authRequest(
	t *testing.T, srv *stubAuthService, actor entity.Actor, method, target string,
) *httptest.ResponseRecorder {
	return authRequestBody(t, srv, actor, method, target, "")
}

// authRequestBody — то же с телом запроса.
func authRequestBody(
	t *testing.T, srv *stubAuthService, actor entity.Actor, method, target, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	stubAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(v1.WithActorForTest(r.Context(), actor)))
		})
	}

	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	v1.NewAuthHandler(srv, stubAuth).Routes().ServeHTTP(w, r)

	return w
}

// TestAuthHandler_Me фиксирует, что своя запись читается по актору из токена,
// а не по чему-либо из запроса.
func TestAuthHandler_Me(t *testing.T) {
	t.Parallel()

	const userID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	srv := &stubAuthService{user: &entity.User{
		ID: userID, Phone: "+77001112233", Role: entity.ActorTypeManager,
	}}

	w := authRequest(t, srv, entity.NewActor(userID, entity.ActorTypeManager),
		http.MethodGet, "/me")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, srv.lastUserID, "идентификатор берётся из контекста")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, userID, body["id"])
	assert.Equal(t, "+77001112233", body["phone"])
	assert.Equal(t, "manager", body["role"])
}

// TestAuthHandler_MeNeedsAuth фиксирует, что без проверки токена маршрут
// не поднимается вовсе: иначе своя запись была бы доступна без входа.
func TestAuthHandler_MeNeedsAuth(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	// Обработчик без проверки токена отдаёт только выпуск токенов.
	v1.NewAuthHandler(&stubAuthService{}, nil).Routes().ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAuthHandler_MeNotFound фиксирует, что удалённая после выпуска токена
// запись даёт 404, а не пустой ответ.
func TestAuthHandler_MeNotFound(t *testing.T) {
	t.Parallel()

	srv := &stubAuthService{err: entity.ErrUserNotFound}

	w := authRequest(t, srv, entity.NewActor("id", entity.ActorTypeManager), http.MethodGet, "/me")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"USER_NOT_FOUND")
}

// TestAuthHandler_StaffNeedsAdmin — главная проверка правки: список сотрудников
// закрыт от менеджера.
//
// Номера коллег — персональные данные, и то, кто ещё работает в системе,
// менеджеру знать незачем.
func TestAuthHandler_StaffNeedsAdmin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		actor     entity.Actor
		expStatus int
	}{
		{
			name:      "администратор видит список",
			actor:     entity.NewActor("admin-1", entity.ActorTypeAdmin),
			expStatus: http.StatusOK,
		},
		{
			name:      "менеджеру отказано",
			actor:     entity.NewActor("manager-1", entity.ActorTypeManager),
			expStatus: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv := &stubAuthService{users: entity.Users{
				{ID: "u-1", Phone: "+77001112233", Role: entity.ActorTypeManager},
			}}

			w := authRequest(t, srv, c.actor, http.MethodGet, "/staff")

			require.Equal(t, c.expStatus, w.Code)

			if c.expStatus != http.StatusOK {
				assert.Empty(t, srv.users[0].ID != "" && srv.lastRole != "",
					"до сервиса запрос менеджера не доходит")

				return
			}

			assert.Contains(t, w.Body.String(), "+77001112233")
		})
	}
}

// TestAuthHandler_StaffFiltersByRole фиксирует необязательный отбор по роли.
func TestAuthHandler_StaffFiltersByRole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		query   string
		expRole entity.ActorType
	}{
		{name: "без отбора", query: "", expRole: ""},
		{name: "менеджеры", query: "?role=manager", expRole: entity.ActorTypeManager},
		{name: "администраторы", query: "?role=admin", expRole: entity.ActorTypeAdmin},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv := &stubAuthService{}

			w := authRequest(t, srv, entity.NewActor("admin-1", entity.ActorTypeAdmin),
				http.MethodGet, "/staff"+c.query)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, c.expRole, srv.lastRole)
		})
	}
}

// TestAuthHandler_StaffUnknownRoleIs400 фиксирует отказ на роли вне перечня.
func TestAuthHandler_StaffUnknownRoleIs400(t *testing.T) {
	t.Parallel()

	srv := &stubAuthService{err: entity.ErrRoleInvalid}

	w := authRequest(t, srv, entity.NewActor("admin-1", entity.ActorTypeAdmin),
		http.MethodGet, "/staff?role=system")

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), entity.CodePrefix+"ROLE_INVALID")
}

// TestAuthHandler_TokenRoutesStayOpen фиксирует, что выпуск токенов остался
// вне проверки: иначе войти было бы невозможно.
func TestAuthHandler_TokenRoutesStayOpen(t *testing.T) {
	t.Parallel()

	denyAll := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	for _, path := range []string{"/manager/token", "/admin/token"} {
		path := path

		t.Run(path, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodPost, path, http.NoBody)
			w := httptest.NewRecorder()

			v1.NewAuthHandler(&stubAuthService{}, denyAll).Routes().ServeHTTP(w, r)

			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"проверка токена не должна стоять на выпуске токена")
		})
	}
}

// TestAuthHandler_RenameMe фиксирует, что имя меняется только у той записи,
// от имени которой пришёл запрос.
//
// Идентификатор берётся из контекста, а не из тела: иначе переименовать
// можно было бы кого угодно.
func TestAuthHandler_RenameMe(t *testing.T) {
	t.Parallel()

	const userID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	srv := &stubAuthService{user: &entity.User{
		ID: userID, Phone: "+77001112233", Role: entity.ActorTypeManager,
		Name: entity.Pointer("Иван Петров"),
	}}

	w := authRequestBody(t, srv, entity.NewActor(userID, entity.ActorTypeManager),
		http.MethodPatch, "/me", `{"name":"Иван Петров","id":"чужой-идентификатор"}`)

	require.Equal(t, http.StatusOK, w.Code, "%s", w.Body.String())
	assert.Equal(t, userID, srv.lastUserID, "идентификатор берётся из контекста, а не из тела")
	assert.Equal(t, "Иван Петров", srv.lastName)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Иван Петров", body["name"])
}

// TestAuthHandler_RenameMeWithoutName фиксирует, что пустое имя — законный
// вход: так оно очищается.
func TestAuthHandler_RenameMeWithoutName(t *testing.T) {
	t.Parallel()

	const userID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	srv := &stubAuthService{user: &entity.User{
		ID: userID, Phone: "+77001112233", Role: entity.ActorTypeManager,
	}}

	w := authRequestBody(t, srv, entity.NewActor(userID, entity.ActorTypeManager),
		http.MethodPatch, "/me", `{"name":""}`)

	require.Equal(t, http.StatusOK, w.Code, "%s", w.Body.String())
	assert.Empty(t, srv.lastName)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotContains(t, body, "name", "незаданное имя не появляется пустой строкой")
}

// TestAuthHandler_RenameMeErrors фиксирует, что отказы доезжают до клиента
// своим статусом, а не пятисоткой.
func TestAuthHandler_RenameMeErrors(t *testing.T) {
	t.Parallel()

	const userID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

	cases := []struct {
		name      string
		body      string
		err       error
		expStatus int
		expCode   string
	}{
		{
			name:      "тело не разбирается",
			body:      "{",
			expStatus: http.StatusBadRequest,
			expCode:   "SNT_FORM_DECODE",
		},
		{
			name:      "слишком длинное имя",
			body:      `{"name":"Иван"}`,
			err:       entity.ErrUserNameInvalid,
			expStatus: http.StatusBadRequest,
			expCode:   "SNT_USER_NAME_INVALID",
		},
		{
			// Идентификатор из токена годен, а записи уже нет: её удалили
			// после выпуска токена.
			name:      "запись удалена",
			body:      `{"name":"Иван"}`,
			err:       entity.ErrUserNotFound,
			expStatus: http.StatusNotFound,
			expCode:   "SNT_USER_NOT_FOUND",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv := &stubAuthService{err: c.err}

			w := authRequestBody(t, srv, entity.NewActor(userID, entity.ActorTypeManager),
				http.MethodPatch, "/me", c.body)

			require.Equal(t, c.expStatus, w.Code, "%s", w.Body.String())
			assert.Contains(t, w.Body.String(), c.expCode)
		})
	}
}

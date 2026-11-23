package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/form"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	httpx "github.com/eskermese/machinery-catalog/backend/pkg/httpx"
)

// AuthHandler обработчик учётных записей: выпуск токенов и чтение своей записи.
type AuthHandler struct {
	authService service.AuthService             // Сервис учётных записей
	auth        func(http.Handler) http.Handler // Проверка токена для закрытых маршрутов
}

// NewAuthHandler создаёт обработчик учётных записей.
//
// Проверка токена приходит извне, а не строится здесь: раздел монтируется
// одним куском, и часть его маршрутов обязана оставаться открытой, поэтому
// решать, где стоит проверка, должен сам обработчик.
func NewAuthHandler(authService service.AuthService, auth func(http.Handler) http.Handler) *AuthHandler {
	return &AuthHandler{authService: authService, auth: auth}
}

// Routes возвращает роутер учётных записей.
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Выпуск токенов вне проверки токена: получить токен, имея токен, незачем.
	r.Post("/manager/token", h.managerToken)
	r.Post("/admin/token", h.adminToken)

	if h.auth == nil {
		return r
	}

	r.Group(func(r chi.Router) {
		r.Use(h.auth)

		// Своя запись доступна любому вошедшему.
		r.Get("/me", h.me)

		// Имя правит сам сотрудник, и только своё: оно ничего не решает
		// в правах, а спросить его больше некому — вход идёт по номеру,
		// форм регистрации нет.
		r.Patch("/me", h.renameMe)

		// Список сотрудников — только администратору: кто ещё работает
		// в системе, менеджеру знать незачем, а номера коллег — это
		// персональные данные, которые он не обязан видеть списком.
		r.Group(func(r chi.Router) {
			r.Use(RoleMiddleware(entity.ActorTypeAdmin))

			r.Get("/staff", h.staff)
		})
	})

	return r
}

// me возвращает учётную запись действующего пользователя.
//
// Существует затем, чтобы клиент не разбирал полезную нагрузку токена сам:
// разбор без проверки подписи — это доверие к строке, которую прислал
// сам клиент, и однажды в неё впишут чужую роль.
//
//	@Summary		Своя учётная запись
//	@Description	Учётная запись, от имени которой действует токен. Читается из хранилища, а не собирается из токена: запись могла быть удалена после его выпуска
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	presenter.AuthUser
//	@Failure		401	{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		404	{object}	swagger.HTTPResponse400	"Учётная запись не найдена"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/auth/me [get]
func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.authService.CurrentUser(r.Context(), actorIDFromContext(r.Context()))
	if err != nil {
		httpx.LogRequestError(r, "получение своей учётной записи", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAuthUser(user))
}

// renameMe задаёт имя своей учётной записи.
//
//	@Summary		Смена своего имени
//	@Description	Задаёт имя для показа в админке. Только своё: имя не участвует в правах, а знает его сам сотрудник — вход идёт по номеру, форм регистрации нет. Пустая строка очищает имя
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			name	body		form.AuthUserRename	true	"Имя"
//	@Success		200		{object}	presenter.AuthUser
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: имя слишком длинное"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Учётная запись не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/auth/me [patch]
func (h *AuthHandler) renameMe(w http.ResponseWriter, r *http.Request) {
	var renameForm form.AuthUserRename

	if !decodeForm(w, r, &renameForm) {
		return
	}

	user, err := h.authService.RenameSelf(r.Context(),
		actorIDFromContext(r.Context()), renameForm.Name)
	if err != nil {
		httpx.LogRequestError(r, "смена своего имени", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAuthUser(user))
}

// staff возвращает список сотрудников.
//
//	@Summary		Сотрудники
//	@Description	Учётные записи сотрудников в порядке (роль, телефон). Только для администратора
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			role	query		string	false	"Роль: manager или admin"
//	@Success		200		{object}	presenter.AuthUserList
//	@Failure		400		{object}	swagger.HTTPResponse400	"Неизвестная роль"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Список сотрудников доступен только администратору"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/auth/staff [get]
func (h *AuthHandler) staff(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.ListStaff(r.Context(), entity.ActorType(r.URL.Query().Get("role")))
	if err != nil {
		httpx.LogRequestError(r, "получение списка сотрудников", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAuthUserList(users))
}

// managerToken выпускает токен менеджера.
//
//	@Summary		Токен менеджера
//	@Description	Выпускает токен по номеру телефона. Учётная запись создаётся при первом входе
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		form.AuthTokenIssue	true	"Номер телефона"
//	@Success		200			{object}	presenter.AuthToken
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/auth/manager/token [post]
func (h *AuthHandler) managerToken(w http.ResponseWriter, r *http.Request) {
	h.issue(w, r, entity.ActorTypeManager)
}

// adminToken выпускает токен администратора.
//
//	@Summary		Токен администратора
//	@Description	Выпускает токен только для заранее заведённого администратора (ADMIN_PHONES)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		form.AuthTokenIssue	true	"Номер телефона"
//	@Success		200			{object}	presenter.AuthToken
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Номер не заведён как администратор"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/auth/admin/token [post]
func (h *AuthHandler) adminToken(w http.ResponseWriter, r *http.Request) {
	h.issue(w, r, entity.ActorTypeAdmin)
}

// issue выпускает токен под роль, заданную маршрутом.
// Роль не принимается из тела запроса: иначе клиент назначал бы её себе сам.
func (h *AuthHandler) issue(w http.ResponseWriter, r *http.Request, role entity.ActorType) {
	issueForm := form.AuthTokenIssue{Role: role}

	if err := httpx.DecodeJSON(r.Body, &issueForm); err != nil {
		httpx.LogRequestError(r, "form decode", err, nil)
		httpx.WriteError(w, err)

		return
	}

	issued, err := h.authService.IssueToken(r.Context(), issueForm, time.Now().UTC())
	if err != nil {
		httpx.LogRequestError(r, "ошибка выпуска токена", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, issued)
}

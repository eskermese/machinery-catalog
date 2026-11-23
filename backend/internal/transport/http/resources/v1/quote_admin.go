package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/form"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	httpx "github.com/eskermese/machinery-catalog/backend/pkg/httpx"
)

// QuoteAdminHandler обработчик административной части предложений и заявок.
type QuoteAdminHandler struct {
	quoteService service.QuoteService // Сервис предложений
	leadService  service.LeadService  // Сервис заявок
}

// NewQuoteAdminHandler создаёт обработчик административной части.
func NewQuoteAdminHandler(quoteService service.QuoteService, leadService service.LeadService) *QuoteAdminHandler {
	return &QuoteAdminHandler{quoteService: quoteService, leadService: leadService}
}

// Routes возвращает роутер административной части.
//
// Почти все операции доступны менеджеру: выпуск предложения и работа
// с заявками — его основная работа, и разделение по ролям здесь было бы
// разделением не по цене ошибки, а по должности. Статус заявки движется
// по таблице переходов, которая сама не пускает назад.
//
// УДАЛЕНИЯ ПРЕДЛОЖЕНИЙ ЗДЕСЬ НЕТ, И ЭТО РЕШЕНИЕ, А НЕ ПРОБЕЛ.
//
// Предложение — зафиксированное обязательство, ссылку на которое клиент уже
// держит в руках. Удалить его значит превратить присланный клиенту документ
// в 404 — ровно то «у меня ничего не открывается», от которого страницу
// и защищали пометкой об истечении. Жизненный цикл у предложения уже есть:
// оно истекает само и после этого открывается с пометкой, показывая
// зафиксированные цифры. Ошибочно выпущенное заменяется выпуском нового,
// а не стиранием старого: клиент мог увидеть его раньше, чем менеджер
// спохватился, и стирание этого уже не отменит.
//
// У заявки цикл другой: она не обязательство, а контакт человека, и у него
// есть право потребовать убрать свои данные. Это единственная причина, ради
// которой удаление здесь заведено, — статусом такое требование не исполняется.
func (h *QuoteAdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/quotes", h.issueQuote)
	r.Get("/quotes", h.listQuotes)
	r.Get("/quotes/{quoteID}", h.quoteByID)

	r.Get("/leads", h.listLeads)
	r.Get("/leads/{leadID}", h.leadByID)
	r.Post("/leads/{leadID}/status", h.changeLeadStatus)
	r.Post("/leads/{leadID}/assign", h.assignLead)
	r.Post("/leads/{leadID}/comment", h.setLeadComment)

	// Справочник программ лизинга. Заводить и править может менеджер:
	// условия партнёров меняются часто, и держать это за администратором
	// значило бы останавливать работу отдела на каждой смене ставки.
	r.Get("/leasing-programs", h.listLeasingPrograms)
	r.Post("/leasing-programs", h.saveLeasingProgram)

	// Удаление — только администратору, по образцу категорий: оно необратимо
	// и стирает условия, по которым уже могли посчитать сделку. Программу,
	// которая больше не действует, менеджеру достаточно выключить флагом.
	r.Group(func(r chi.Router) {
		r.Use(RoleMiddleware(entity.ActorTypeAdmin))

		r.Delete("/leasing-programs/{programID}", h.deleteLeasingProgram)

		// Удаление заявки существует ради требования человека убрать свои
		// персональные данные — статусом это не исполняется. Оно необратимо
		// и стирает контакт живого клиента, поэтому закрыто от менеджера.
		//
		// Предложения не удаляются вовсе: см. комментарий у Routes.
		r.Delete("/leads/{leadID}", h.deleteLead)
	})

	return r
}

// listLeasingPrograms возвращает все программы лизинга, включая выключенные.
//
//	@Summary		Программы лизинга (админка)
//	@Description	Все программы, включая недействующие: выключенную нужно видеть, чтобы включить обратно
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	presenter.LeasingProgramList
//	@Failure		401	{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403	{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leasing-programs [get]
func (h *QuoteAdminHandler) listLeasingPrograms(w http.ResponseWriter, r *http.Request) {
	programs, err := h.quoteService.AllLeasingPrograms(r.Context())
	if err != nil {
		httpx.LogRequestError(r, "получение программ лизинга", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLeasingProgramList(programs))
}

// saveLeasingProgram создаёт или обновляет программу лизинга.
//
//	@Summary		Сохранение программы лизинга
//	@Description	Создаёт программу при пустом id и обновляет при заполненном. Согласованность границ между собой проверяет домен
//	@Tags			quotes-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			program	body		form.LeasingProgramSave	true	"Программа лизинга"
//	@Success		200		{object}	presenter.LeasingProgram
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: границы программы несогласованы или невалидный вход"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Программа не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leasing-programs [post]
func (h *QuoteAdminHandler) saveLeasingProgram(w http.ResponseWriter, r *http.Request) {
	var programForm form.LeasingProgramSave

	if !decodeForm(w, r, &programForm) {
		return
	}

	program, err := h.quoteService.SaveLeasingProgram(r.Context(), programForm.ToEntity())
	if err != nil {
		httpx.LogRequestError(r, "сохранение программы лизинга", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLeasingProgram(program))
}

// deleteLeasingProgram удаляет программу лизинга.
//
//	@Summary		Удаление программы лизинга
//	@Description	Программу, по которой уже выпущены предложения, удалить нельзя — её выключают флагом isActive
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			programID	path	string	true	"Идентификатор программы"
//	@Success		204
//	@Failure		401	{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403	{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404	{object}	swagger.HTTPResponse400	"Программа не найдена"
//	@Failure		409	{object}	swagger.HTTPResponse400	"По программе выпущены предложения"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leasing-programs/{programID} [delete]
func (h *QuoteAdminHandler) deleteLeasingProgram(w http.ResponseWriter, r *http.Request) {
	if err := h.quoteService.DeleteLeasingProgram(r.Context(), chi.URLParam(r, "programID")); err != nil {
		httpx.LogRequestError(r, "удаление программы лизинга", err, nil)
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// issueQuote выпускает коммерческое предложение.
//
//	@Summary		Выпуск коммерческого предложения
//	@Description	Читает опубликованную карточку, фиксирует курс, проверяет параметры против границ программы лизинга и считает график. Возвращает предложение вместе с токеном публичной ссылки
//	@Tags			quotes-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			quote	body		form.QuoteIssue	true	"Параметры предложения"
//	@Success		201		{object}	presenter.QuoteIssued
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: параметры вне границ программы или невалидный вход"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Карточка или программа не найдены"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/quotes [post]
func (h *QuoteAdminHandler) issueQuote(w http.ResponseWriter, r *http.Request) {
	var issueForm form.QuoteIssue

	if !decodeForm(w, r, &issueForm) {
		return
	}

	currentTime := time.Now().UTC()

	issue := &service.QuoteIssue{
		MachineSlug:           issueForm.MachineSlug,
		Configuration:         issueForm.Configuration,
		LeasingProgramID:      issueForm.LeasingProgramID,
		ScheduleType:          entity.LeasingScheduleType(issueForm.ScheduleType),
		DownPaymentBps:        issueForm.DownPaymentBps,
		TermMonths:            issueForm.TermMonths,
		RateBps:               issueForm.RateBps,
		ResidualValueKZTCents: issueForm.ResidualValueKztCents,
		CommissionKZTCents:    issueForm.CommissionKztCents,
		ValidFor:              time.Duration(issueForm.ValidForDays) * 24 * time.Hour,
		AuthorID:              actorIDFromContext(r.Context()),
	}

	quote, err := h.quoteService.Issue(r.Context(), issue, currentTime)
	if err != nil {
		httpx.LogRequestError(r, "выпуск предложения", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, presenter.NewQuoteIssued(quote, currentTime))
}

// listQuotes возвращает предложения текущего менеджера.
//
//	@Summary		Мои коммерческие предложения
//	@Description	Предложения, выпущенные текущим пользователем, свежие сверху
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Размер страницы"
//	@Success		200		{object}	presenter.QuoteList
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/quotes [get]
func (h *QuoteAdminHandler) listQuotes(w http.ResponseWriter, r *http.Request) {
	limit, err := form.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		httpx.LogRequestError(r, "разбор размера страницы", err, nil)
		httpx.WriteError(w, err)

		return
	}

	quotes, err := h.quoteService.ListByAuthor(r.Context(), actorIDFromContext(r.Context()), limit)
	if err != nil {
		httpx.LogRequestError(r, "получение предложений", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewQuoteList(quotes, time.Now().UTC()))
}

// quoteByID возвращает предложение по идентификатору.
//
// Отметку первого открытия не ставит: она означает «клиент увидел
// предложение», а менеджер, просматривающий свой список, — не клиент.
//
//	@Summary		Коммерческое предложение (админка)
//	@Description	Предложение по внутреннему идентификатору. Отметку первого открытия не ставит
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			quoteID	path		string	true	"Идентификатор предложения"
//	@Success		200		{object}	presenter.Quote
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Предложение не найдено"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/quotes/{quoteID} [get]
func (h *QuoteAdminHandler) quoteByID(w http.ResponseWriter, r *http.Request) {
	quote, err := h.quoteService.ByID(r.Context(), chi.URLParam(r, "quoteID"))
	if err != nil {
		httpx.LogRequestError(r, "получение предложения", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewQuote(quote, time.Now().UTC()))
}

// listLeads возвращает заявки по фильтру.
//
//	@Summary		Заявки
//	@Description	Заявки с фильтром по статусу и ответственному менеджеру, свежие сверху. Keyset-пагинация: курсор следующей страницы приходит в ответе. Общее число считается только по запросу
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string	false	"Статус: new, in_progress, won, lost"
//	@Param			managerID	query		string	false	"Ответственный менеджер"
//	@Param			limit		query		int		false	"Размер страницы"
//	@Param			cursor		query		string	false	"Курсор следующей страницы из предыдущего ответа"
//	@Param			withTotal	query		bool	false	"Посчитать общее число найденных заявок"
//	@Success		200			{object}	presenter.LeadList
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leads [get]
func (h *QuoteAdminHandler) listLeads(w http.ResponseWriter, r *http.Request) {
	filter := &entity.LeadFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = entity.Pointer(entity.LeadStatus(status))
	}

	if managerID := r.URL.Query().Get("managerID"); managerID != "" {
		filter.ManagerID = entity.Pointer(managerID)
	}

	limit, err := form.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		httpx.LogRequestError(r, "разбор размера страницы", err, nil)
		httpx.WriteError(w, err)

		return
	}

	filter.Limit = limit
	filter.WithTotal = r.URL.Query().Get("withTotal") == "true"

	if filter.Cursor, err = entity.DecodeLeadCursor(r.URL.Query().Get("cursor")); err != nil {
		httpx.LogRequestError(r, "разбор курсора", err, nil)
		httpx.WriteError(w, err)

		return
	}

	page, err := h.leadService.List(r.Context(), filter)
	if err != nil {
		httpx.LogRequestError(r, "получение заявок", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLeadList(page))
}

// leadByID возвращает заявку по идентификатору.
//
//	@Summary		Заявка
//	@Description	Заявка по идентификатору
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			leadID	path		string	true	"Идентификатор заявки"
//	@Success		200		{object}	presenter.Lead
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Заявка не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leads/{leadID} [get]
func (h *QuoteAdminHandler) leadByID(w http.ResponseWriter, r *http.Request) {
	lead, err := h.leadService.ByID(r.Context(), chi.URLParam(r, "leadID"))
	if err != nil {
		httpx.LogRequestError(r, "получение заявки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLead(lead))
}

// changeLeadStatus переводит заявку в целевой статус.
//
//	@Summary		Смена статуса заявки
//	@Description	Переводит заявку по таблице переходов new → in_progress → won|lost. Недопустимый переход отвергается
//	@Tags			quotes-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			leadID	path		string					true	"Идентификатор заявки"
//	@Param			status	body		form.LeadStatusChange	true	"Целевой статус"
//	@Success		200		{object}	presenter.Lead
//	@Failure		400		{object}	swagger.HTTPResponse400	"Недопустимый переход статуса"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Заявка не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leads/{leadID}/status [post]
func (h *QuoteAdminHandler) changeLeadStatus(w http.ResponseWriter, r *http.Request) {
	var statusForm form.LeadStatusChange

	if !decodeForm(w, r, &statusForm) {
		return
	}

	h.writeLeadUpdate(w, r, "смена статуса заявки", func() (*entity.Lead, error) {
		return h.leadService.ChangeStatus(r.Context(), chi.URLParam(r, "leadID"),
			statusForm.ToStatus(), time.Now().UTC())
	})
}

// setLeadComment заменяет комментарий менеджера к заявке.
//
//	@Summary		Комментарий менеджера к заявке
//	@Description	Заменяет заметку менеджера целиком; пустая строка её стирает. Сообщение клиента (message) этой ручкой не меняется — это разные поля, и правка менеджера не должна затирать запрос клиента
//	@Tags			quotes-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			leadID	path		string				true	"Идентификатор заявки"
//	@Param			comment	body		form.LeadComment	true	"Комментарий менеджера"
//	@Success		200		{object}	presenter.Lead
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Заявка не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leads/{leadID}/comment [post]
func (h *QuoteAdminHandler) setLeadComment(w http.ResponseWriter, r *http.Request) {
	var commentForm form.LeadComment

	if !decodeForm(w, r, &commentForm) {
		return
	}

	h.writeLeadUpdate(w, r, "комментарий к заявке", func() (*entity.Lead, error) {
		return h.leadService.SetComment(r.Context(), chi.URLParam(r, "leadID"),
			commentForm.Comment, time.Now().UTC())
	})
}

// writeLeadUpdate применяет правку к заявке и отдаёт её представление.
//
// Хвост у всех правок заявки одинаковый — отказ в лог и наружу, успех
// представлением, — и различаются они только тем, что именно меняют.
// Держать этот хвост в трёх копиях значило бы чинить его трижды.
func (h *QuoteAdminHandler) writeLeadUpdate(
	w http.ResponseWriter,
	r *http.Request,
	action string,
	update func() (*entity.Lead, error),
) {
	lead, err := update()
	if err != nil {
		httpx.LogRequestError(r, action, err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLead(lead))
}

// assignLead назначает ответственного менеджера.
//
//	@Summary		Назначение ответственного
//	@Description	Назначает менеджера, ответственного за заявку. Не зависит от статуса: распределение — вопрос загрузки людей, а не стадии обработки
//	@Tags			quotes-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			leadID	path		string			true	"Идентификатор заявки"
//	@Param			assign	body		form.LeadAssign	true	"Ответственный менеджер"
//	@Success		200		{object}	presenter.Lead
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Заявка не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leads/{leadID}/assign [post]
func (h *QuoteAdminHandler) assignLead(w http.ResponseWriter, r *http.Request) {
	var assignForm form.LeadAssign

	if !decodeForm(w, r, &assignForm) {
		return
	}

	h.writeLeadUpdate(w, r, "назначение ответственного", func() (*entity.Lead, error) {
		return h.leadService.AssignManager(r.Context(), chi.URLParam(r, "leadID"),
			assignForm.ManagerID, time.Now().UTC())
	})
}

// actorIDFromContext возвращает идентификатор действующего пользователя.
// Пустая строка означает, что запрос пришёл мимо аутентификации, —
// такой сценарий отсекается middleware до обработчика.
func actorIDFromContext(ctx context.Context) string {
	actor := actorFromContext(ctx)
	if actor.ID == nil {
		return ""
	}

	return *actor.ID
}

// deleteLead удаляет заявку без следа.
//
//	@Summary		Удаление заявки
//	@Description	Удаляет заявку вместе с контактом клиента, без следа. Существует ради требования человека убрать свои персональные данные — статусом такое не исполняется. Выпущенное по заявке предложение удаление переживает: обязательство перед клиентом не исчезает оттого, что прибрали заявку. Предложения, в отличие от заявок, не удаляются вовсе — они истекают
//	@Tags			quotes-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			leadID	path	string	true	"Идентификатор заявки"
//	@Success		204
//	@Failure		401	{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403	{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404	{object}	swagger.HTTPResponse400	"Заявка не найдена"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/leads/{leadID} [delete]
func (h *QuoteAdminHandler) deleteLead(w http.ResponseWriter, r *http.Request) {
	if err := h.leadService.Delete(r.Context(), chi.URLParam(r, "leadID")); err != nil {
		httpx.LogRequestError(r, "удаление заявки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

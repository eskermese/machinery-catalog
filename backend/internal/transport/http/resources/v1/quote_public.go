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

// QuoteHandler обработчик публичной страницы коммерческого предложения
// и приёма заявок с сайта.
type QuoteHandler struct {
	quoteService   service.QuoteService   // Сервис предложений
	leadService    service.LeadService    // Сервис заявок
	tcoService     service.TCOService     // Сервис стоимости владения
	catalogService service.CatalogService // Сервис каталога: слаг раздела для ссылки
}

// NewQuoteHandler создаёт обработчик публичной части предложений и заявок.
func NewQuoteHandler(
	quoteService service.QuoteService,
	leadService service.LeadService,
	tcoService service.TCOService,
	catalogService service.CatalogService,
) *QuoteHandler {
	return &QuoteHandler{
		quoteService:   quoteService,
		leadService:    leadService,
		tcoService:     tcoService,
		catalogService: catalogService,
	}
}

// Routes возвращает роутер публичной части.
func (h *QuoteHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/quotes/{token}", h.quoteByToken)
	r.Get("/leasing-programs", h.leasingPrograms)
	r.Post("/leads", h.createLead)

	// Расчёт закрыт ограничителем частоты, остальные публичные ручки — нет:
	// они отдают готовые данные, а расчёт читает карточку, программу и курс
	// и строит график на весь срок. Предел стоит на маршруте, а не на роутере,
	// чтобы не мешать обычному обходу каталога.
	r.Group(func(r chi.Router) {
		r.Use(RateLimitMiddleware(calculateRateLimit, calculateRateWindow))

		r.Post("/leasing/calculate", h.calculateLeasing)
		// Стоимость владения и пересчёт предложения — такие же расчёты
		// по запросу, и предел у них общий с лизингом.
		r.Post("/machines/{slug}/tco", h.calculateTCO)
		r.Post("/quotes/{token}/recalculate", h.recalculateQuote)
	})

	return r
}

// calculateLeasing считает график по опубликованной карточке.
//
// Ничего не сохраняет и НЕ фиксирует курс: фиксация — операция выпуска
// предложения, после которой цифра перестаёт меняться. Здесь расчёт
// справочный и пересчитывается вместе с курсом.
//
//	@Summary		Расчёт лизинга
//	@Description	Справочный расчёт по опубликованной карточке. Цену и курс сервер берёт сам: цену по слагу, курс — действующий. Ничего не сохраняется и не фиксируется. Машина со скрытой ценой отвергается. Частота обращений с одного адреса ограничена
//	@Tags			catalog
//	@Accept			json
//	@Produce		json
//	@Param			params	body		form.LeasingCalculate	true	"Параметры расчёта"
//	@Success		200		{object}	presenter.LeasingCalculation
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: параметры вне границ программы, цена скрыта или невалидный вход"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Карточка, программа или курс не найдены"
//	@Failure		429		{object}	swagger.HTTPResponse400	"Превышена частота обращений с адреса"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/leasing/calculate [post]
func (h *QuoteHandler) calculateLeasing(w http.ResponseWriter, r *http.Request) {
	var calcForm form.LeasingCalculate

	if !decodeForm(w, r, &calcForm) {
		return
	}

	quote, err := h.quoteService.Calculate(r.Context(), &service.QuoteCalculate{
		MachineSlug:           calcForm.MachineSlug,
		LeasingProgramID:      calcForm.LeasingProgramID,
		ScheduleType:          entity.LeasingScheduleType(calcForm.ScheduleType),
		DownPaymentBps:        calcForm.DownPaymentBps,
		TermMonths:            calcForm.TermMonths,
		RateBps:               calcForm.RateBps,
		ResidualValueKZTCents: calcForm.ResidualValueKztCents,
		CommissionKZTCents:    calcForm.CommissionKztCents,
	}, time.Now().UTC())
	if err != nil {
		httpx.LogRequestError(r, "расчёт лизинга", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLeasingCalculation(quote))
}

// calculateTCO считает стоимость владения по опубликованной карточке.
//
// Ничего не сохраняет. Незаведённые у карточки величины отказом не являются:
// ответ приходит неполным, и в нём перечислено, чего не хватило. Клиент
// обязан показать это словами — «нет данных», а не «0 ₸»: расчёт с выдуманным
// расходом топлива выглядит как расчёт, а является догадкой.
//
//	@Summary		Стоимость владения
//	@Description	Разложенная стоимость владения за срок. Расход топлива, интервал и стоимость ТО, остаточная стоимость берутся по карточке; чего не хватило — в поле missing
//	@Tags			catalog
//	@Accept			json
//	@Produce		json
//	@Param			slug	path		string				true	"Идентификатор карточки в URL"
//	@Param			params	body		form.TCOCalculate	true	"Параметры расчёта"
//	@Success		200		{object}	presenter.TCOResult
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: цена скрыта или невалидный вход"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Карточка или курс не найдены"
//	@Failure		429		{object}	swagger.HTTPResponse400	"Превышена частота обращений с адреса"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/machines/{slug}/tco [post]
func (h *QuoteHandler) calculateTCO(w http.ResponseWriter, r *http.Request) {
	var calcForm form.TCOCalculate

	if !decodeForm(w, r, &calcForm) {
		return
	}

	calculated, err := h.tcoService.Calculate(r.Context(), service.TCORequest{
		MachineSlug:            chi.URLParam(r, "slug"),
		Years:                  calcForm.Years,
		HoursPerYear:           calcForm.HoursPerYear,
		FuelPriceKZTCents:      calcForm.FuelPriceKztCents,
		OperatorHourlyKZTCents: calcForm.OperatorHourlyKztCents,
	}, time.Now().UTC())
	if err != nil {
		httpx.LogRequestError(r, "расчёт стоимости владения", err, nil)
		httpx.WriteError(w, err)

		return
	}

	// Слаг раздела нужен для ссылки на карточку и читается отдельно: расчёт
	// его не знает, а собирать адрес из идентификатора категории клиенту
	// нечем — адрес карточки на витрине вложен в раздел.
	categorySlug := h.categorySlugOf(r, calculated.Machine.CategoryID)

	httpx.WriteJSON(w, http.StatusOK,
		presenter.NewTCOResult(calculated.Machine, categorySlug, calculated.FX, &calculated.Result))
}

// categorySlugOf возвращает слаг раздела карточки или пустую строку.
//
// Отказ дерева не роняет расчёт: без слага пропадёт только ссылка на карточку,
// а цифры, ради которых запрос и делался, останутся.
func (h *QuoteHandler) categorySlugOf(r *http.Request, categoryID string) string {
	categories, err := h.catalogService.CategoryTree(r.Context())
	if err != nil {
		return ""
	}

	for _, category := range categories {
		if category != nil && category.ID == categoryID {
			return category.Slug
		}
	}

	return ""
}

// recalculateQuote пересчитывает предложение на других условиях.
//
// Ничего не сохраняет и выпущенное предложение не меняет: КП — это
// зафиксированное обязательство, и клиентский ввод не может подменить
// его цифры. Ответ клиент видит РЯДОМ с предложением, а не вместо него.
//
// Курс берётся зафиксированный в предложении. Обоснование — в сервисе;
// коротко: пересчёт отвечает на вопрос об условиях сделки, а не о валюте,
// и менять вместе со сроком ещё и стоимость машины значило бы отвечать
// сразу на два вопроса, не ответив ни на один.
//
// Токен в пути, как и при чтении предложения, не попадает ни в логи,
// ни в метрики.
//
//	@Summary		Пересчёт предложения на других условиях
//	@Description	Считает график на новых условиях по зафиксированному в предложении курсу. Ничего не сохраняет и выпущенное предложение не меняет
//	@Tags			quotes
//	@Accept			json
//	@Produce		json
//	@Param			token	path		string					true	"Публичный токен предложения"
//	@Param			params	body		form.QuoteRecalculate	true	"Новые условия"
//	@Success		200		{object}	presenter.LeasingCalculation
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: параметры вне границ программы или невалидный вход"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Предложение не найдено"
//	@Failure		429		{object}	swagger.HTTPResponse400	"Превышена частота обращений с адреса"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/quotes/{token}/recalculate [post]
func (h *QuoteHandler) recalculateQuote(w http.ResponseWriter, r *http.Request) {
	var recalcForm form.QuoteRecalculate

	if !decodeForm(w, r, &recalcForm) {
		return
	}

	quote, err := h.quoteService.RecalculateByToken(r.Context(), chi.URLParam(r, "token"),
		service.QuoteRecalculate{
			ScheduleType:   entity.LeasingScheduleType(recalcForm.ScheduleType),
			DownPaymentBps: recalcForm.DownPaymentBps,
			TermMonths:     recalcForm.TermMonths,
			RateBps:        recalcForm.RateBps,
		})
	if err != nil {
		httpx.LogRequestError(r, "пересчёт предложения", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLeasingCalculation(quote))
}

// quoteByToken возвращает коммерческое предложение по публичной ссылке.
//
// Токен в пути — единственная защита предложения, поэтому он не должен
// попадать ни в логи, ни в метрики. Обеспечено тремя вещами:
//
//   - метрики маркируют запрос chi-паттерном («/quotes/{token}»), а не путём;
//   - общий лог запросов пишет тот же паттерн — см. pkg/logger/middleware;
//   - здесь ошибки логируются без пути и без query.
//
// Истёкшее предложение отдаётся с признаком isExpired, а не отказом: клиент
// должен увидеть, что предложение было, и когда оно истекло, — иначе ссылка
// из переписки выглядит сломанной.
//
//	@Summary		Коммерческое предложение по ссылке
//	@Description	Предложение вместе с машиной, зафиксированным курсом, параметрами расчёта и полным графиком платежей. Первое открытие фиксируется. Истёкшее предложение возвращается с признаком isExpired
//	@Tags			quotes
//	@Produce		json
//	@Param			token	path		string	true	"Токен публичной ссылки"
//	@Success		200		{object}	presenter.Quote
//	@Failure		404		{object}	swagger.HTTPResponse400	"Предложение не найдено"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/quotes/{token} [get]
func (h *QuoteHandler) quoteByToken(w http.ResponseWriter, r *http.Request) {
	currentTime := time.Now().UTC()

	quote, err := h.quoteService.ByPublicToken(r.Context(), chi.URLParam(r, "token"), currentTime)
	if err != nil {
		// Логируется без пути: путь содержит токен.
		logQuoteError(r, "получение предложения по ссылке", err)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewQuote(quote, currentTime))
}

// leasingPrograms возвращает действующие программы лизинга.
//
//	@Summary		Программы лизинга
//	@Description	Действующие программы с границами, внутри которых считается платёж
//	@Tags			quotes
//	@Produce		json
//	@Success		200	{object}	presenter.LeasingProgramList
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/leasing-programs [get]
func (h *QuoteHandler) leasingPrograms(w http.ResponseWriter, r *http.Request) {
	programs, err := h.quoteService.LeasingPrograms(r.Context())
	if err != nil {
		httpx.LogRequestError(r, "получение программ лизинга", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewLeasingProgramList(programs))
}

// createLead принимает заявку с сайта.
//
//	@Summary		Заявка с сайта
//	@Description	Принимает обратный звонок или запрос цены. Номер нормализуется. Слишком частые заявки с одного номера отклоняются с кодом 409
//	@Tags			quotes
//	@Accept			json
//	@Produce		json
//	@Param			lead	body		form.LeadCreate	true	"Заявка"
//	@Success		201		{object}	presenter.LeadAccepted
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		409		{object}	swagger.HTTPResponse400	"Заявка с этого номера уже принята"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/leads [post]
func (h *QuoteHandler) createLead(w http.ResponseWriter, r *http.Request) {
	var leadForm form.LeadCreate

	if !decodeForm(w, r, &leadForm) {
		return
	}

	lead, err := h.leadService.Create(r.Context(), &service.LeadCreate{
		Name:      leadForm.Name,
		Phone:     leadForm.Phone,
		City:      leadForm.City,
		MachineID: leadForm.MachineID,
		QuoteID:   leadForm.QuoteID,
		Source:    leadForm.Source,
		Message:   leadForm.Message,
	}, time.Now().UTC())
	if err != nil {
		// Телефон в заявке — персональные данные, поэтому тело запроса
		// в лог не идёт: httpx.LogRequestError пишет только query и путь.
		httpx.LogRequestError(r, "приём заявки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, presenter.NewLeadAccepted(lead))
}

// logQuoteError логирует ошибку работы с предложением, не раскрывая токен.
//
// Собственная функция вместо httpx.LogRequestError: та пишет полный путь
// и строку запроса, а путь предложения целиком состоит из секрета.
func logQuoteError(r *http.Request, message string, err error) {
	httpx.LogError(r.Context(), message, err, map[string]any{
		"method": r.Method,
		"route":  chi.RouteContext(r.Context()).RoutePattern(),
	})
}

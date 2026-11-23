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

// historyLimit сколько записей истории отдаётся за раз.
//
// Тридцать — примерно месяц ежедневных курсов: столько нужно, чтобы
// объяснить цену в недавно выданном предложении, и не столько, чтобы
// страница админки превратилась в бесконечную ленту.
const historyLimit = 30

// FXAdminHandler обработчик курсов, наценок и настроек показа цены.
type FXAdminHandler struct {
	fxService       service.FXService       // Сервис курсов валют
	settingsService service.SettingsService // Сервис настроек
	baseCurrency    string                  // Валюта прайса
	quoteCurrency   string                  // Валюта, в которую считаем обязательно
	usdCurrency     string                  // Валюта второго отображения
}

// NewFXAdminHandler создаёт обработчик курсов и наценок.
func NewFXAdminHandler(
	fxService service.FXService,
	settingsService service.SettingsService,
	baseCurrency, quoteCurrency, usdCurrency string,
) *FXAdminHandler {
	return &FXAdminHandler{
		fxService:       fxService,
		settingsService: settingsService,
		baseCurrency:    baseCurrency,
		quoteCurrency:   quoteCurrency,
		usdCurrency:     usdCurrency,
	}
}

// Routes возвращает роутер курсов, наценок и настроек.
//
// РАЗДЕЛЕНИЕ ПРАВ ЗДЕСЬ НЕ ПО ДОЛЖНОСТИ, А ПО ЦЕНЕ ОШИБКИ.
//
// Ввод курса доступен менеджеру. Это ежедневная работа: курс обновляется
// каждый рабочий день, и запереть её за одним человеком значит остановить
// показ цен в тот день, когда он в отпуске. Ошибка при этом видна и дёшева:
// рядом показывается курс Нацбанка, с которым введённое сверяется глазом,
// а опечатка исправляется вводом нового курса за ту же дату.
//
// Наценка и включатель доллара — только администратору. Наценка множит цену
// ВСЕГО каталога разом и мгновенно: ошибка в ней не «видна в одном расчёте»,
// как у программы лизинга, а уже показана каждому, кто открыл витрину.
// Меняется она при этом редко — это решение о марже, а не операция, — так что
// администраторский доступ ничего не тормозит. Включатель доллара меняет то,
// что видит клиент, на всех страницах сразу и по той же причине закрыт.
func (h *FXAdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/status", h.status)
	r.Get("/rates", h.rateHistory)
	r.Post("/rates", h.saveRate)

	r.Group(func(r chi.Router) {
		r.Use(RoleMiddleware(entity.ActorTypeAdmin))

		r.Get("/markups", h.markupHistory)
		r.Post("/markups", h.saveMarkup)
		r.Get("/settings", h.settings)
		r.Put("/settings", h.saveSettings)
	})

	return r
}

// status отдаёт состояние курсов по всем парам, которые нужны витрине.
//
//	@Summary		Состояние курсов
//	@Description	Действующий курс каждой пары, что предложил внешний источник, что ввёл человек, признак устаревания. Пара второго отображения приходит всегда — по ней видно, готова ли витрина показывать цену в долларах
//	@Tags			admin-fx
//	@Produce		json
//	@Success		200	{object}	presenter.FXStatusList
//	@Failure		401	{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Security		BearerAuth
//	@Router			/v1/admin/fx/status [get]
func (h *FXAdminHandler) status(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	pairs := [][2]string{
		{h.baseCurrency, h.quoteCurrency},
		{h.baseCurrency, h.usdCurrency},
	}

	statuses := make([]*entity.FXPairStatus, 0, len(pairs))

	for _, pair := range pairs {
		status, err := h.fxService.PairStatus(r.Context(), pair[0], pair[1], now)
		if err != nil {
			httpx.LogRequestError(r, "состояние курса", err, nil)
			httpx.WriteError(w, err)

			return
		}

		// Пустое состояние всё равно называет пару: строка без валют
		// не сообщает администратору, курса ЧЕГО не хватает, а именно
		// за этим ручка и открывается.
		if status == nil {
			status = &entity.FXPairStatus{BaseCurrency: pair[0], QuoteCurrency: pair[1]}
		}

		statuses = append(statuses, status)
	}

	showUSD := h.settingsService.ShowUSDPrice(r.Context())

	httpx.WriteJSON(w, http.StatusOK, presenter.NewFXStatusList(statuses, showUSD))
}

// rateHistory отдаёт историю ручных курсов пары.
//
//	@Summary		История ручных курсов
//	@Description	Курсы, введённые человеком, свежие первыми. Загруженные воркером сюда не попадают: их история — это история источника, а не решений администратора
//	@Tags			admin-fx
//	@Produce		json
//	@Param			base	query		string	true	"Валюта, из которой считаем"
//	@Param			quote	query		string	true	"Валюта, в которую считаем"
//	@Success		200		{object}	presenter.FXRateList
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Security		BearerAuth
//	@Router			/v1/admin/fx/rates [get]
func (h *FXAdminHandler) rateHistory(w http.ResponseWriter, r *http.Request) {
	base, quote, ok := h.pairFromQuery(w, r)
	if !ok {
		return
	}

	rates, err := h.fxService.RateHistory(r.Context(), base, quote, historyLimit)
	if err != nil {
		httpx.LogRequestError(r, "история курсов", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewFXRateList(rates))
}

// saveRate сохраняет курс, введённый человеком.
//
//	@Summary		Ввести курс вручную
//	@Description	Ручной курс заменяет курс внешнего источника за ту же дату. Повторный ввод за ту же дату — исправление опечатки: создаётся новая запись, прежняя остаётся в истории
//	@Tags			admin-fx
//	@Accept			json
//	@Produce		json
//	@Param			rate	body		form.FXRateSave	true	"Курс"
//	@Success		201		{object}	presenter.FXRate
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Security		BearerAuth
//	@Router			/v1/admin/fx/rates [post]
func (h *FXAdminHandler) saveRate(w http.ResponseWriter, r *http.Request) {
	var rateForm form.FXRateSave

	if err := httpx.DecodeJSON(r.Body, &rateForm); err != nil {
		httpx.LogRequestError(r, "form decode", err, nil)
		httpx.WriteError(w, err)

		return
	}

	rate, err := rateForm.ToEntity(time.Now().UTC())
	if err != nil {
		httpx.LogRequestError(r, "разбор курса", err, nil)
		httpx.WriteError(w, err)

		return
	}

	saved, err := h.fxService.SaveRate(r.Context(), rate)
	if err != nil {
		httpx.LogRequestError(r, "сохранение курса", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, presenter.NewFXRate(saved))
}

// markupHistory отдаёт историю наценок пары.
//
//	@Summary		История наценок
//	@Description	Наценки пары, свежие первыми. Действующая — первая, у которой момент вступления не в будущем
//	@Tags			admin-fx
//	@Produce		json
//	@Param			base	query		string	true	"Валюта, из которой считаем"
//	@Param			quote	query		string	true	"Валюта, в которую считаем"
//	@Success		200		{object}	presenter.FXMarkupList
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Недостаточно прав"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Security		BearerAuth
//	@Router			/v1/admin/fx/markups [get]
func (h *FXAdminHandler) markupHistory(w http.ResponseWriter, r *http.Request) {
	base, quote, ok := h.pairFromQuery(w, r)
	if !ok {
		return
	}

	markups, err := h.fxService.MarkupHistory(r.Context(), base, quote, historyLimit)
	if err != nil {
		httpx.LogRequestError(r, "история наценок", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewFXMarkupList(markups))
}

// saveMarkup задаёт наценку на пару.
//
//	@Summary		Задать наценку
//	@Description	Наценка своя у каждой пары. История сохраняется: прежняя запись остаётся и объясняет цену в ранее выданных предложениях
//	@Tags			admin-fx
//	@Accept			json
//	@Produce		json
//	@Param			markup	body		form.FXMarkupSave	true	"Наценка"
//	@Success		201		{object}	presenter.FXMarkup
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Недостаточно прав"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Security		BearerAuth
//	@Router			/v1/admin/fx/markups [post]
func (h *FXAdminHandler) saveMarkup(w http.ResponseWriter, r *http.Request) {
	var markupForm form.FXMarkupSave

	if err := httpx.DecodeJSON(r.Body, &markupForm); err != nil {
		httpx.LogRequestError(r, "form decode", err, nil)
		httpx.WriteError(w, err)

		return
	}

	actor := actorFromContext(r.Context())

	markup, err := markupForm.ToEntity(actor.ID, time.Now().UTC())
	if err != nil {
		httpx.LogRequestError(r, "разбор наценки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	saved, err := h.fxService.SetMarkup(r.Context(), markup)
	if err != nil {
		httpx.LogRequestError(r, "сохранение наценки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, presenter.NewFXMarkup(saved))
}

// settings отдаёт настройки показа цены.
//
//	@Summary	Настройки показа цены
//	@Tags		admin-fx
//	@Produce	json
//	@Success	200	{object}	presenter.PriceSettings
//	@Failure	401	{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure	403	{object}	swagger.HTTPResponse400	"Недостаточно прав"
//	@Security	BearerAuth
//	@Router		/v1/admin/fx/settings [get]
func (h *FXAdminHandler) settings(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, presenter.PriceSettings{
		ShowUSDPrice: h.settingsService.ShowUSDPrice(r.Context()),
	})
}

// saveSettings меняет настройки показа цены.
//
//	@Summary		Изменить настройки показа цены
//	@Description	Доллар считается из юаня по паре CNY→USD со своей наценкой, а не хранится отдельно. Включение без заданного курса этой пары оставит цену в долларах пустой — состояние курса видно в ручке состояния
//	@Tags			admin-fx
//	@Accept			json
//	@Produce		json
//	@Param			settings	body		form.PriceSettingsSave	true	"Настройки"
//	@Success		200			{object}	presenter.PriceSettings
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Не авторизован"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Недостаточно прав"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Security		BearerAuth
//	@Router			/v1/admin/fx/settings [put]
func (h *FXAdminHandler) saveSettings(w http.ResponseWriter, r *http.Request) {
	var settingsForm form.PriceSettingsSave

	if err := httpx.DecodeJSON(r.Body, &settingsForm); err != nil {
		httpx.LogRequestError(r, "form decode", err, nil)
		httpx.WriteError(w, err)

		return
	}

	if err := h.settingsService.SetShowUSDPrice(r.Context(), settingsForm.ShowUSDPrice); err != nil {
		httpx.LogRequestError(r, "сохранение настроек", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.PriceSettings{ShowUSDPrice: settingsForm.ShowUSDPrice})
}

// pairFromQuery читает валютную пару из параметров запроса.
//
// Пара обязательна и не имеет умолчания: ручка обслуживает и CNY→KZT,
// и CNY→USD, и «забыл указать» здесь означало бы молча показать историю
// не той пары.
func (h *FXAdminHandler) pairFromQuery(
	w http.ResponseWriter, r *http.Request,
) (base, quote string, ok bool) {
	base = r.URL.Query().Get("base")
	quote = r.URL.Query().Get("quote")

	if base == "" || quote == "" {
		httpx.WriteError(w, entity.ErrFXPairInvalid)

		return "", "", false
	}

	return base, quote, true
}

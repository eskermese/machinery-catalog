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

// CatalogHandler обработчик публичной витрины каталога.
//
// Держит только публичные методы сервиса. Админские сюда не приходят
// физически: обработчик их не вызывает, поэтому «забыть фильтр черновиков»
// на публичном маршруте нельзя — такого вызова просто нет в коде.
type CatalogHandler struct {
	catalogService  service.CatalogService  // Сервис каталога
	fxService       service.FXService       // Курсы: пересчёт цены в тенге и доллары
	settingsService service.SettingsService // Настройки: включён ли показ доллара

	baseCurrency  string // Валюта прайса
	quoteCurrency string // Валюта обязательного пересчёта
	usdCurrency   string // Валюта второго отображения
}

// NewCatalogHandler создаёт обработчик публичной витрины.
func NewCatalogHandler(
	catalogService service.CatalogService,
	fxService service.FXService,
	settingsService service.SettingsService,
	baseCurrency, quoteCurrency, usdCurrency string,
) *CatalogHandler {
	return &CatalogHandler{
		catalogService:  catalogService,
		fxService:       fxService,
		settingsService: settingsService,
		baseCurrency:    baseCurrency,
		quoteCurrency:   quoteCurrency,
		usdCurrency:     usdCurrency,
	}
}

// showcaseRates возвращает курсы для пересчёта цен выдачи.
//
// Запрашиваются ОДИН раз на ответ и передаются в представление целиком:
// на карточку курс не запрашивается — иначе страница выдачи стоила бы
// столько же обращений к кэшу, сколько на ней карточек, а часть карточек
// могла бы оказаться пересчитанной по другому курсу.
//
// Обе валюты считаются ОТ ЮАНЯ, каждая по своей паре: юань — единственная
// хранимая цена, и второй прайс не заводится ни в базе, ни здесь.
//
// Отсутствие курса не ошибка: витрина обязана открыться и показать юани.
// Возвращает пустую структуру — представление в этом случае опускает цену
// в этой валюте вместе с датой курса.
//
// Курс НЕ фиксируется: FixRateForQuote здесь не зовётся, это операция выпуска
// предложения. Пересчёт витрины справочный и меняется вместе с курсом.
func (h *CatalogHandler) showcaseRates(r *http.Request) presenter.PriceRates {
	if h.fxService == nil {
		return presenter.PriceRates{}
	}

	now := time.Now().UTC()
	rates := presenter.PriceRates{KZT: h.showcaseRate(r, h.quoteCurrency, now)}

	// Доллар спрашивается, только когда его показ включён: иначе каждый
	// ответ витрины ходил бы за курсом, который никому не показывается.
	if h.settingsService != nil && h.settingsService.ShowUSDPrice(r.Context()) {
		rates.USD = h.showcaseRate(r, h.usdCurrency, now)
	}

	return rates
}

// showcaseRate возвращает действующий курс одной пары или nil.
func (h *CatalogHandler) showcaseRate(
	r *http.Request,
	quote string,
	currentTime time.Time,
) *entity.AppliedFXRate {
	applied, err := h.fxService.AppliedRate(r.Context(), h.baseCurrency, quote, currentTime)
	if err != nil {
		// Логируется, но не возвращается: отсутствие курса роняет пересчёт,
		// а не каталог.
		httpx.LogRequestError(r, "получение курса для витрины", err, nil)

		return nil
	}

	return &applied
}

// Routes возвращает роутер публичной витрины.
func (h *CatalogHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/categories", h.categories)
	r.Get("/categories/{categoryID}/attributes", h.categoryAttributes)
	r.Post("/machines/search", h.searchMachines)
	r.Get("/machines/{slug}", h.machineBySlug)

	return r
}

// categories возвращает дерево категорий.
//
//	@Summary		Дерево категорий
//	@Description	Плоский список категорий с parent_id в порядке (родитель, position). Дерево клиент собирает сам одним проходом
//	@Tags			catalog
//	@Produce		json
//	@Success		200	{object}	presenter.CategoryList
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/catalog/categories [get]
func (h *CatalogHandler) categories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catalogService.CategoryTree(r.Context())
	if err != nil {
		httpx.LogRequestError(r, "получение дерева категорий", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewCategoryList(categories))
}

// categoryAttributes возвращает характеристики категории с вариантами enum.
//
// Набор включает унаследованные от предков: характеристика, объявленная
// у родителя, действует и в дочерних категориях. Без этого панель фильтров
// дочерней категории не совпадала бы с родительской, а фильтр по поддереву
// терял бы карточки потомков — у них был бы другой идентификатор для того же
// понятия.
//
//	@Summary		Характеристики категории
//	@Description	Набор характеристик для панели фильтров вместе с вариантами для типа enum. Включает характеристики, объявленные у родительских категорий: они действуют и в дочерних. Объявление в самой категории затеняет одноимённое у предка. У числовых приходят границы значений в разделе: из них строится подсказка под полем фильтра
//	@Tags			catalog
//	@Produce		json
//	@Param			categoryID	path		string	true	"Идентификатор категории"	example(7c9e6679-7425-40de-944b-e07fc1f90ae7)
//	@Success		200			{object}	presenter.AttributeList
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/catalog/categories/{categoryID}/attributes [get]
func (h *CatalogHandler) categoryAttributes(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryID")

	attributes, err := h.catalogService.PublicCategoryAttributes(r.Context(), categoryID)
	if err != nil {
		httpx.LogRequestError(r, "получение характеристик категории", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAttributeList(attributes))
}

// searchMachines ищет карточки по витрине.
//
// POST, а не GET, потому что набор условий по характеристикам переменный
// и вложенный: в строке запроса он превратился бы в разбор повторяющихся
// параметров вида attr[0][id]=…&attr[0][min]=…, где ошибку кодирования
// не увидит ни валидатор, ни компилятор. Ссылочность выдачи от этого не
// страдает: фронт хранит состояние фильтров в query собственной страницы
// и отправляет его телом — ссылка остаётся шарящейся, а контракт API
// остаётся типизированным.
//
//	@Summary		Поиск по каталогу
//	@Description	Поиск опубликованных карточек с фильтрами, сортировкой и keyset-пагинацией. Черновики не возвращаются ни при каких параметрах, включая isActive — на витрине он игнорируется. Поиск query ищет подстроку в названии, бренде и модели. Цена в тенге приходит вместе с датой курса; при отсутствии курса обоих полей нет, а выдача работает как обычно
//	@Tags			catalog
//	@Accept			json
//	@Produce		json
//	@Param			filter	body		form.MachineSearch	true	"Условия поиска"
//	@Success		200		{object}	presenter.MachinePage
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: невалидный фильтр или испорченный курсор"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/catalog/machines/search [post]
func (h *CatalogHandler) searchMachines(w http.ResponseWriter, r *http.Request) {
	filter, ok := decodeSearchFilter(w, r)
	if !ok {
		return
	}

	page, err := h.catalogService.SearchPublicMachines(r.Context(), filter)
	if err != nil {
		httpx.LogRequestError(r, "поиск по каталогу", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewMachinePage(page, h.showcaseRates(r)))
}

// machineBySlug возвращает опубликованную карточку по слагу.
//
//	@Summary		Карточка техники
//	@Description	Опубликованная карточка вместе с характеристиками и фотографиями. Неопубликованная отдаётся как несуществующая. Цена в тенге приходит вместе с датой курса; при отсутствии курса обоих полей нет
//	@Tags			catalog
//	@Produce		json
//	@Param			slug	path		string	true	"Идентификатор карточки в URL"	example(xcmg-xe215c)
//	@Success		200		{object}	presenter.Machine
//	@Failure		404		{object}	swagger.HTTPResponse400	"Карточка не найдена или не опубликована"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/catalog/machines/{slug} [get]
func (h *CatalogHandler) machineBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	machine, err := h.catalogService.PublicMachineBySlug(r.Context(), slug)
	if err != nil {
		httpx.LogRequestError(r, "получение карточки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewMachine(machine, h.showcaseRates(r)))
}

// validatable форма, умеющая себя проверить.
type validatable interface {
	Validate() error
}

// decodeForm разбирает тело запроса в форму и проверяет её.
//
// Общая часть всех обработчиков сохранения: разбор и проверка формы устроены
// одинаково, и повторять их в каждом значило бы копировать ещё и обработку
// ошибок — то есть место, где расхождение заметят позже всего.
// Возвращает false, если ответ клиенту уже записан.
func decodeForm[T validatable](w http.ResponseWriter, r *http.Request, target T) bool {
	if err := httpx.DecodeJSON(r.Body, target); err != nil {
		httpx.LogRequestError(r, "form decode", err, nil)
		httpx.WriteError(w, err)

		return false
	}

	if err := target.Validate(); err != nil {
		httpx.LogRequestError(r, "валидация формы", err, nil)
		httpx.WriteError(w, err)

		return false
	}

	return true
}

// decodeSearchFilter разбирает тело поиска и приводит его к доменному фильтру.
//
// Общая часть публичного и админского поиска: разбор формы, урезание лимита
// и разбор курсора одинаковы, а решает про видимость черновиков сервис.
// Возвращает false, если ответ клиенту уже записан.
func decodeSearchFilter(w http.ResponseWriter, r *http.Request) (*entity.MachineFilter, bool) {
	var searchForm form.MachineSearch

	if !decodeForm(w, r, &searchForm) {
		return nil, false
	}

	// Лимит урезается до максимума, а не отвергается: клиент, попросивший
	// десять тысяч карточек, ошибся в масштабе, а не в синтаксисе.
	searchForm.Limit = clampLimit(searchForm.Limit)

	filter, err := searchForm.ToFilter()
	if err != nil {
		// Испорченный курсор — это 400, а не 500: его прислал клиент.
		httpx.LogRequestError(r, "разбор фильтра", err, nil)
		httpx.WriteError(w, err)

		return nil, false
	}

	return filter, true
}

// clampLimit приводит размер страницы к допустимым границам.
func clampLimit(limit int) int {
	if limit <= 0 {
		return entity.DefaultMachineLimit
	}

	return min(limit, entity.MaxMachineLimit)
}

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

// CatalogAdminHandler обработчик административной части каталога.
type CatalogAdminHandler struct {
	catalogService service.CatalogService // Сервис каталога
}

// NewCatalogAdminHandler создаёт обработчик административной части каталога.
func NewCatalogAdminHandler(catalogService service.CatalogService) *CatalogAdminHandler {
	return &CatalogAdminHandler{catalogService: catalogService}
}

// Routes возвращает роутер административной части каталога.
//
// Права разведены по цене ошибки, а не по «важности» раздела.
//
// Менеджеру доступно всё, что можно откатить тем же действием: завести карточку,
// поправить её, снять с публикации, переставить фотографии. Ошибся — исправил,
// потерянного состояния нет.
//
// Администратору оставлены две операции, которые исправить обратной правкой
// нельзя.
//
// Первая — удаление категории и характеристики. Схема каскадит характеристики
// и значения вслед за категорией, поэтому удаление уносит вместе с узлом
// заполненные значения всех его карточек, и «создать заново» их не вернёт.
//
// Вторая — публикация карточки. Она делает позицию видимой снаружи вместе
// с ценой; отозвать уже показанную клиенту цену нельзя, а published_at
// проставляется один раз и задаёт место в ленте навсегда.
//
// Изменение цены при этом остаётся у менеджера намеренно: цена меняется
// в рабочем порядке и на уже выпущенные КП не влияет — там она зафиксирована
// копией.
func (h *CatalogAdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Доступно менеджеру и администратору.
	r.Group(func(r chi.Router) {
		r.Get("/categories", h.categories)
		r.Post("/categories", h.createCategory)
		r.Put("/categories/{categoryID}", h.updateCategory)

		r.Get("/categories/{categoryID}/attributes", h.categoryAttributes)
		r.Post("/attributes", h.createAttribute)
		r.Put("/attributes/{attributeID}", h.updateAttribute)

		r.Post("/machines/search", h.searchMachines)
		r.Get("/machines/{slug}", h.machineBySlug)
		r.Post("/machines", h.createMachine)
		r.Put("/machines/{machineID}", h.updateMachine)
		r.Post("/machines/{machineID}/unpublish", h.unpublishMachine)

		r.Get("/machines/{machineID}/images", h.machineImages)
		r.Put("/machines/{machineID}/images", h.replaceImages)
		r.Post("/machines/{machineID}/images/{imageID}/primary", h.setPrimaryImage)
	})

	// Только администратору: необратимые операции.
	r.Group(func(r chi.Router) {
		r.Use(RoleMiddleware(entity.ActorTypeAdmin))

		r.Delete("/categories/{categoryID}", h.deleteCategory)
		r.Delete("/attributes/{attributeID}", h.deleteAttribute)
		r.Delete("/machines/{machineID}", h.deleteMachine)
		r.Post("/machines/{machineID}/publish", h.publishMachine)

		// Кривые остаточной стоимости целиком под администратором,
		// в отличие от программ лизинга, где заводить и править может
		// менеджер. Разница в цене ошибки: выключенная программа ничего
		// не считает и молчит, а кривая участвует в расчёте всегда —
		// занижение остаточной стоимости завышает стоимость владения сразу
		// на всех карточках раздела, и по одной карточке этого не заметить.
		r.Get("/residual-curves", h.residualCurves)
		r.Get("/residual-curves/{categoryID}", h.residualCurveByCategory)
		r.Put("/residual-curves/{categoryID}", h.saveResidualCurve)
		r.Delete("/residual-curves/{categoryID}", h.deleteResidualCurve)
	})

	return r
}

// categories возвращает все категории, включая неактивные.
//
//	@Summary		Категории (админка)
//	@Description	Полное дерево категорий, включая скрытые с сайта
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	presenter.CategoryList
//	@Failure		401	{object}	swagger.HTTPResponse400	"Запрос не аутентифицирован"
//	@Failure		403	{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/categories [get]
func (h *CatalogAdminHandler) categories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catalogService.CategoryTree(r.Context())
	if err != nil {
		httpx.LogRequestError(r, "получение категорий", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewCategoryList(categories))
}

// createCategory заводит новую категорию.
//
//	@Summary		Создание категории
//	@Description	Занятый слаг возвращает 409
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			category	body		form.CategorySave	true	"Категория"
//	@Success		200			{object}	presenter.Category
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		409			{object}	swagger.HTTPResponse400	"Слаг уже занят"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/categories [post]
func (h *CatalogAdminHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	h.saveCategory(w, r)
}

// updateCategory изменяет существующую категорию.
//
//	@Summary		Изменение категории
//	@Description	Занятый слаг возвращает 409, попытка сделать категорию собственным предком — 400
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			categoryID	path		string				true	"Идентификатор категории"
//	@Param			category	body		form.CategorySave	true	"Категория"
//	@Success		200			{object}	presenter.Category
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки: например, цикл в дереве"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Категория не найдена"
//	@Failure		409			{object}	swagger.HTTPResponse400	"Слаг уже занят"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/categories/{categoryID} [put]
func (h *CatalogAdminHandler) updateCategory(w http.ResponseWriter, r *http.Request) {
	h.saveCategory(w, r)
}

// saveCategory создаёт или изменяет категорию: идентификатор берётся из пути.
func (h *CatalogAdminHandler) saveCategory(w http.ResponseWriter, r *http.Request) {
	var categoryForm form.CategorySave

	if !decodeForm(w, r, &categoryForm) {
		return
	}

	saved, err := h.catalogService.SaveCategory(r.Context(), categoryForm.ToEntity(chi.URLParam(r, "categoryID")))
	if err != nil {
		httpx.LogRequestError(r, "сохранение категории", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewCategory(saved))
}

// deleteCategory удаляет категорию. Только администратор.
//
//	@Summary		Удаление категории
//	@Description	Необратимая операция: каскад схемы уносит характеристики категории и заполненные значения всех её карточек. Категория с потомками или карточками не удаляется
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			categoryID	path	string	true	"Идентификатор категории"
//	@Success		204			"Категория удалена"
//	@Failure		400			{object}	swagger.HTTPResponse400	"У категории есть потомки или карточки"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Категория не найдена"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/categories/{categoryID} [delete]
func (h *CatalogAdminHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.catalogService.DeleteCategory(r.Context(), chi.URLParam(r, "categoryID")); err != nil {
		httpx.LogRequestError(r, "удаление категории", err, nil)
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteMachine удаляет карточку. Только администратор.
//
//	@Summary		Удаление карточки
//	@Description	Необратимая операция: каскад схемы уносит значения характеристик и фотографии карточки, а файлы снимков удаляются, если на них не ссылается другая карточка. Карточка с выпущенным КП не удаляется — её снимают с публикации
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path	string	true	"Идентификатор карточки"
//	@Success		204			"Карточка удалена"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Карточка не найдена"
//	@Failure		409			{object}	swagger.HTTPResponse400	"По карточке выпущены коммерческие предложения"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID} [delete]
func (h *CatalogAdminHandler) deleteMachine(w http.ResponseWriter, r *http.Request) {
	if err := h.catalogService.DeleteMachine(r.Context(), chi.URLParam(r, "machineID")); err != nil {
		httpx.LogRequestError(r, "удаление карточки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// categoryAttributes возвращает характеристики категории.
//
//	@Summary		Характеристики категории (админка)
//	@Description	Характеристики вместе с вариантами для типа enum
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			categoryID	path		string	true	"Идентификатор категории"
//	@Success		200			{object}	presenter.AttributeList
//	@Failure		401			{object}	swagger.HTTPResponse400	"Запрос не аутентифицирован"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/categories/{categoryID}/attributes [get]
func (h *CatalogAdminHandler) categoryAttributes(w http.ResponseWriter, r *http.Request) {
	attributes, err := h.catalogService.CategoryAttributes(r.Context(), chi.URLParam(r, "categoryID"))
	if err != nil {
		httpx.LogRequestError(r, "получение характеристик", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAttributeList(attributes))
}

// createAttribute заводит новую характеристику вместе с вариантами.
//
//	@Summary		Создание характеристики
//	@Description	Создаёт характеристику вместе с набором вариантов enum
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			attribute	body		form.AttributeSave	true	"Характеристика"
//	@Success		200			{object}	presenter.Attribute
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/attributes [post]
func (h *CatalogAdminHandler) createAttribute(w http.ResponseWriter, r *http.Request) {
	h.saveAttribute(w, r)
}

// updateAttribute изменяет существующую характеристику вместе с вариантами.
//
//	@Summary		Изменение характеристики
//	@Description	Заменяет характеристику вместе с набором вариантов enum. Смена типа у характеристики с заполненными значениями отвергается
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			attributeID	path		string				true	"Идентификатор характеристики"
//	@Param			attribute	body		form.AttributeSave	true	"Характеристика"
//	@Success		200			{object}	presenter.Attribute
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки: например, смена типа у заполненной характеристики"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Характеристика не найдена"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/attributes/{attributeID} [put]
func (h *CatalogAdminHandler) updateAttribute(w http.ResponseWriter, r *http.Request) {
	h.saveAttribute(w, r)
}

// saveAttribute создаёт или изменяет характеристику: идентификатор из пути.
func (h *CatalogAdminHandler) saveAttribute(w http.ResponseWriter, r *http.Request) {
	var attributeForm form.AttributeSave

	if !decodeForm(w, r, &attributeForm) {
		return
	}

	saved, err := h.catalogService.SaveAttribute(r.Context(),
		attributeForm.ToEntity(chi.URLParam(r, "attributeID")))
	if err != nil {
		httpx.LogRequestError(r, "сохранение характеристики", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAttribute(saved))
}

// deleteAttribute удаляет характеристику. Только администратор.
//
//	@Summary		Удаление характеристики
//	@Description	Необратимая операция: каскад схемы уносит варианты и заполненные значения всех карточек категории
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			attributeID	path	string	true	"Идентификатор характеристики"
//	@Success		204			"Характеристика удалена"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Характеристика не найдена"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/attributes/{attributeID} [delete]
func (h *CatalogAdminHandler) deleteAttribute(w http.ResponseWriter, r *http.Request) {
	if err := h.catalogService.DeleteAttribute(r.Context(), chi.URLParam(r, "attributeID")); err != nil {
		httpx.LogRequestError(r, "удаление характеристики", err, nil)
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// searchMachines ищет карточки по всему каталогу, включая черновики.
//
//	@Summary		Поиск по каталогу (админка)
//	@Description	Поиск по всему каталогу, включая неопубликованные карточки. Цена отдаётся всегда, даже когда скрыта на витрине. Отбор isActive показывает только опубликованные или только черновики; query ищет подстроку в названии, бренде и модели
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			filter	body		form.MachineSearch	true	"Условия поиска"
//	@Success		200		{object}	presenter.MachinePage
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: невалидный фильтр или испорченный курсор"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/search [post]
func (h *CatalogAdminHandler) searchMachines(w http.ResponseWriter, r *http.Request) {
	filter, ok := decodeSearchFilter(w, r)
	if !ok {
		return
	}

	page, err := h.catalogService.SearchAllMachines(r.Context(), filter)
	if err != nil {
		httpx.LogRequestError(r, "поиск по каталогу", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAdminMachinePage(page))
}

// machineBySlug возвращает карточку по слагу, включая черновик.
//
//	@Summary		Карточка техники (админка)
//	@Description	Любая карточка, включая неопубликованную, вместе с характеристиками и фотографиями
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Идентификатор карточки в URL"
//	@Success		200		{object}	presenter.Machine
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404		{object}	swagger.HTTPResponse400	"Карточка не найдена"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{slug} [get]
func (h *CatalogAdminHandler) machineBySlug(w http.ResponseWriter, r *http.Request) {
	machine, err := h.catalogService.MachineBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		httpx.LogRequestError(r, "получение карточки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAdminMachine(machine))
}

// createMachine заводит новую карточку.
//
// Отдельная точка входа при общем теле обработчика: у создания и изменения
// разные маршруты и разные наборы параметров, а один блок аннотаций описал бы
// оба одинаково — и приписал бы POST несуществующий параметр пути.
//
//	@Summary		Создание карточки
//	@Description	Карточка, её характеристики и фотографии пишутся одной транзакцией. Занятый слаг возвращает 409
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machine	body		form.MachineSave	true	"Карточка"
//	@Success		200		{object}	presenter.Machine
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		409		{object}	swagger.HTTPResponse400	"Слаг уже занят"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines [post]
func (h *CatalogAdminHandler) createMachine(w http.ResponseWriter, r *http.Request) {
	h.saveMachine(w, r)
}

// updateMachine изменяет существующую карточку.
//
//	@Summary		Изменение карточки
//	@Description	Заменяет карточку целиком вместе с характеристиками и фотографиями — одной транзакцией. Занятый другим слаг возвращает 409
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path		string				true	"Идентификатор карточки"
//	@Param			machine		body		form.MachineSave	true	"Карточка"
//	@Success		200			{object}	presenter.Machine
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Карточка не найдена"
//	@Failure		409			{object}	swagger.HTTPResponse400	"Слаг уже занят"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID} [put]
func (h *CatalogAdminHandler) updateMachine(w http.ResponseWriter, r *http.Request) {
	h.saveMachine(w, r)
}

// saveMachine создаёт или изменяет карточку: идентификатор берётся из пути.
func (h *CatalogAdminHandler) saveMachine(w http.ResponseWriter, r *http.Request) {
	var machineForm form.MachineSave

	if !decodeForm(w, r, &machineForm) {
		return
	}

	machine, err := machineForm.ToEntity(chi.URLParam(r, "machineID"))
	if err != nil {
		httpx.LogRequestError(r, "разбор карточки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	saved, err := h.catalogService.SaveMachine(r.Context(), machine)
	if err != nil {
		httpx.LogRequestError(r, "сохранение карточки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAdminMachine(saved))
}

// publishMachine публикует карточку. Только администратор.
//
//	@Summary		Публикация карточки
//	@Description	Делает карточку видимой на витрине и проставляет дату первой публикации. Повторная публикация дату не сбрасывает: она задаёт место в ленте
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path		string	true	"Идентификатор карточки"
//	@Success		200			{object}	presenter.Machine
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Карточка не найдена"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID}/publish [post]
func (h *CatalogAdminHandler) publishMachine(w http.ResponseWriter, r *http.Request) {
	machine, err := h.catalogService.PublishMachine(r.Context(),
		chi.URLParam(r, "machineID"), time.Now().UTC())
	if err != nil {
		httpx.LogRequestError(r, "публикация карточки", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAdminMachine(machine))
}

// unpublishMachine снимает карточку с публикации.
//
//	@Summary		Снятие карточки с публикации
//	@Description	Скрывает карточку с витрины, сохраняя дату первой публикации
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path		string	true	"Идентификатор карточки"
//	@Success		200			{object}	presenter.Machine
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Карточка не найдена"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID}/unpublish [post]
func (h *CatalogAdminHandler) unpublishMachine(w http.ResponseWriter, r *http.Request) {
	machine, err := h.catalogService.UnpublishMachine(r.Context(), chi.URLParam(r, "machineID"))
	if err != nil {
		httpx.LogRequestError(r, "снятие карточки с публикации", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewAdminMachine(machine))
}

// machineImages возвращает фотографии карточки.
//
//	@Summary		Фотографии карточки
//	@Description	Фотографии в порядке показа
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path		string	true	"Идентификатор карточки"
//	@Success		200			{object}	presenter.MachineImageList
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID}/images [get]
func (h *CatalogAdminHandler) machineImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.catalogService.MachineImages(r.Context(), chi.URLParam(r, "machineID"))
	if err != nil {
		httpx.LogRequestError(r, "получение фотографий", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewMachineImageList(images))
}

// replaceImages заменяет набор фотографий карточки целиком.
//
//	@Summary		Замена фотографий
//	@Description	Заменяет набор целиком: пустой набор очищает галерею. Главной может быть не более одной фотографии
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path		string						true	"Идентификатор карточки"
//	@Param			images		body		form.MachineImagesReplace	true	"Набор фотографий"
//	@Success		200			{object}	presenter.MachineImageList
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки: например, две главные фотографии"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID}/images [put]
func (h *CatalogAdminHandler) replaceImages(w http.ResponseWriter, r *http.Request) {
	var imagesForm form.MachineImagesReplace

	if !decodeForm(w, r, &imagesForm) {
		return
	}

	machineID := chi.URLParam(r, "machineID")

	images, err := h.catalogService.ReplaceMachineImages(r.Context(), machineID, imagesForm.ToEntity())
	if err != nil {
		httpx.LogRequestError(r, "замена фотографий", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewMachineImageList(images))
}

// setPrimaryImage назначает главное фото карточки.
//
//	@Summary		Назначение главного фото
//	@Description	Снимает признак с остальных фотографий одним запросом: состояния с двумя главными не существует
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			machineID	path		string	true	"Идентификатор карточки"
//	@Param			imageID		path		string	true	"Идентификатор фотографии"
//	@Success		200			{object}	presenter.MachineImageList
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Карточка или фотография не найдены"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/machines/{machineID}/images/{imageID}/primary [post]
func (h *CatalogAdminHandler) setPrimaryImage(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")

	images, err := h.catalogService.SetPrimaryImage(r.Context(), machineID, chi.URLParam(r, "imageID"))
	if err != nil {
		httpx.LogRequestError(r, "назначение главного фото", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewMachineImageList(images))
}

// residualCurves возвращает все заведённые кривые остаточной стоимости.
//
//	@Summary		Кривые остаточной стоимости
//	@Description	Все заведённые кривые вместе с посчитанными по ним долями по годам. Разделы без своей кривой сюда не попадают — они наследуют кривую предка
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	presenter.ResidualValueCurveList
//	@Failure		401	{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403	{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/residual-curves [get]
func (h *CatalogAdminHandler) residualCurves(w http.ResponseWriter, r *http.Request) {
	curves, err := h.catalogService.ResidualCurves(r.Context())
	if err != nil {
		httpx.LogRequestError(r, "получение кривых остаточной стоимости", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewResidualValueCurveList(curves))
}

// residualCurveByCategory возвращает кривую, заведённую именно для раздела.
//
//	@Summary		Кривая остаточной стоимости раздела
//	@Description	Кривая, заведённая ИМЕННО для этого раздела. Вверх по дереву не поднимается: справочник показывает своё, а не унаследованное, иначе редактор принял бы кривую предка за собственную и завёл бы её копию
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			categoryID	path		string	true	"Идентификатор раздела"
//	@Success		200			{object}	presenter.ResidualValueCurve
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404			{object}	swagger.HTTPResponse400	"У раздела нет своей кривой"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/residual-curves/{categoryID} [get]
func (h *CatalogAdminHandler) residualCurveByCategory(w http.ResponseWriter, r *http.Request) {
	curve, err := h.catalogService.ResidualCurveByCategory(r.Context(), chi.URLParam(r, "categoryID"))
	if err != nil {
		httpx.LogRequestError(r, "получение кривой остаточной стоимости", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewResidualValueCurve(curve))
}

// saveResidualCurve заводит кривую раздела или переписывает её.
//
//	@Summary		Сохранение кривой остаточной стоимости
//	@Description	Кривая задаётся тремя числами целиком, а не строкой на каждый год: тридцать чисел на раздел руками не заводят. Заводит кривую раздела или переписывает её — у раздела она ровно одна. Возвращает кривую вместе с посчитанными долями по годам
//	@Tags			catalog-admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			categoryID	path		string						true	"Идентификатор раздела"
//	@Param			curve		body		form.ResidualValueCurveSave	true	"Кривая остаточной стоимости"
//	@Success		200			{object}	presenter.ResidualValueCurve
//	@Failure		400			{object}	swagger.HTTPResponse400	"Код ошибки: доли вне границ или порог выше остатка первого года"
//	@Failure		401			{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403			{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404			{object}	swagger.HTTPResponse400	"Раздел не найден"
//	@Failure		500			{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/residual-curves/{categoryID} [put]
func (h *CatalogAdminHandler) saveResidualCurve(w http.ResponseWriter, r *http.Request) {
	var curveForm form.ResidualValueCurveSave

	if !decodeForm(w, r, &curveForm) {
		return
	}

	// Раздел берётся из адреса: в теле его нет и быть не должно — иначе
	// вышло бы два места, где написано, какому разделу правят кривую.
	curve, err := h.catalogService.SaveResidualCurve(r.Context(),
		curveForm.ToEntity(chi.URLParam(r, "categoryID")))
	if err != nil {
		httpx.LogRequestError(r, "сохранение кривой остаточной стоимости", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, presenter.NewResidualValueCurve(curve))
}

// deleteResidualCurve убирает кривую раздела.
//
//	@Summary		Удаление кривой остаточной стоимости
//	@Description	После удаления раздел наследует кривую предка, а если её нет ни у кого — расчёт владения снова честно скажет, что остаточной стоимости не знает
//	@Tags			catalog-admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			categoryID	path	string	true	"Идентификатор раздела"
//	@Success		204
//	@Failure		401	{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403	{object}	swagger.HTTPResponse400	"Операция доступна только администратору"
//	@Failure		404	{object}	swagger.HTTPResponse400	"У раздела нет своей кривой"
//	@Failure		500	{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/catalog/residual-curves/{categoryID} [delete]
func (h *CatalogAdminHandler) deleteResidualCurve(w http.ResponseWriter, r *http.Request) {
	if err := h.catalogService.DeleteResidualCurve(r.Context(), chi.URLParam(r, "categoryID")); err != nil {
		httpx.LogRequestError(r, "удаление кривой остаточной стоимости", err, nil)
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

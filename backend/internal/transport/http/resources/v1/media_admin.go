package v1

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	httpx "github.com/eskermese/machinery-catalog/backend/pkg/httpx"
)

const (
	// uploadFieldName имя поля с файлом в multipart-форме.
	uploadFieldName = "file"

	// multipartOverhead запас на служебную часть multipart-запроса:
	// границы, заголовки частей и имя поля. Предел самого файла проверяет
	// домен, здесь ограничивается всё тело целиком.
	multipartOverhead = 1 << 16

	// multipartMemory сколько разрешено держать в памяти при разборе формы;
	// остальное уходит во временный файл.
	multipartMemory = 1 << 20
)

// MediaAdminHandler обработчик загрузки изображений.
//
// Раздел хранилища задаётся маршрутом, а не полем формы. Разница
// принципиальная: маршрут перечислен в коде, а поле формы пришло бы
// от клиента, и «раздел» превратился бы в произвольный путь на диске.
type MediaAdminHandler struct {
	mediaService service.MediaService // Сервис изображений
}

// NewMediaAdminHandler создаёт обработчик загрузки изображений.
func NewMediaAdminHandler(mediaService service.MediaService) *MediaAdminHandler {
	return &MediaAdminHandler{mediaService: mediaService}
}

// Routes возвращает роутер загрузки изображений.
//
// Все ручки доступны менеджеру и администратору: загрузка обратима удалением
// файла и ничего не публикует сама по себе — карточка остаётся черновиком,
// пока её не опубликует администратор.
func (h *MediaAdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/machines", h.uploadMachineImage)
	r.Post("/programs", h.uploadProgramLogo)
	r.Post("/categories", h.uploadCategoryCover)

	return r
}

// uploadMachineImage принимает фотографию карточки техники.
//
//	@Summary		Загрузка фотографии карточки
//	@Description	Принимает изображение JPEG, PNG или WebP не больше 5 МиБ. Тип определяется по содержимому файла, имя генерирует сервер. Возвращает путь и адрес в том же виде, что и фотография карточки
//	@Tags			media-admin
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Изображение"
//	@Success		201		{object}	presenter.UploadedFile
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: файл не передан, больше предела или неподдерживаемого типа"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/media/machines [post]
func (h *MediaAdminHandler) uploadMachineImage(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, entity.MediaKindMachine)
}

// uploadProgramLogo принимает логотип программы лизинга.
//
//	@Summary		Загрузка логотипа программы лизинга
//	@Description	Принимает изображение JPEG, PNG или WebP не больше 5 МиБ. Тип определяется по содержимому файла, имя генерирует сервер
//	@Tags			media-admin
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Изображение"
//	@Success		201		{object}	presenter.UploadedFile
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: файл не передан, больше предела или неподдерживаемого типа"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/media/programs [post]
func (h *MediaAdminHandler) uploadProgramLogo(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, entity.MediaKindProgram)
}

// uploadCategoryCover принимает обложку раздела каталога.
//
//	@Summary		Загрузка обложки раздела каталога
//	@Description	Принимает изображение JPEG, PNG или WebP не больше 5 МиБ. Тип определяется по содержимому файла, имя генерирует сервер. Путь из ответа кладётся в imagePath категории; без него обложку подставляет сервис по первой опубликованной машине поддерева
//	@Tags			media-admin
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Изображение"
//	@Success		201		{object}	presenter.UploadedFile
//	@Failure		400		{object}	swagger.HTTPResponse400	"Код ошибки: файл не передан, больше предела или неподдерживаемого типа"
//	@Failure		401		{object}	swagger.HTTPResponse400	"Токен не передан, испорчен или истёк"
//	@Failure		403		{object}	swagger.HTTPResponse400	"Роль не имеет доступа к разделу"
//	@Failure		500		{object}	swagger.HTTPResponse500	"Внутренняя ошибка сервера"
//	@Router			/v1/admin/media/categories [post]
func (h *MediaAdminHandler) uploadCategoryCover(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, entity.MediaKindCategory)
}

// upload разбирает multipart-запрос и отдаёт файл сервису.
func (h *MediaAdminHandler) upload(w http.ResponseWriter, r *http.Request, kind entity.MediaKind) {
	// Тело ограничивается ДО разбора. Без этого запрос на гигабайт был бы
	// сначала целиком прочитан во временный файл и только потом отвергнут.
	r.Body = http.MaxBytesReader(w, r.Body, entity.MaxImageBytes+multipartOverhead)

	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		httpx.WriteError(w, uploadError(err))

		return
	}

	// Временные файлы, в которые multipart сложил части, убираются здесь:
	// без этого каждая загрузка оставляла бы мусор в /tmp.
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile(uploadFieldName)
	if err != nil {
		httpx.WriteError(w, uploadError(err))

		return
	}

	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		httpx.WriteError(w, uploadError(err))

		return
	}

	stored, err := h.mediaService.SaveImage(r.Context(), kind, content)
	if err != nil {
		httpx.LogRequestError(r, "загрузка изображения", err, nil)
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, presenter.NewUploadedFile(stored))
}

// uploadError переводит отказы разбора запроса в доменные ошибки.
//
// Без этого перевода превышение предела вернулось бы клиенту пятисоткой:
// http.MaxBytesReader отдаёт собственный тип ошибки, а «файл слишком большой» —
// это ошибка запроса, а не сервера.
func uploadError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return entity.ErrMediaTooLarge
	}

	if errors.Is(err, http.ErrMissingFile) {
		return entity.ErrMediaFileMissing
	}

	return entity.ErrFormDecode
}

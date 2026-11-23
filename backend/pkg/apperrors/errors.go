package apperrors

import (
	"net/http"

	"github.com/go-chi/render"
)

// Response - структура ответа.
type Response struct {
	Err            error  `json:"-"`    // Низкоуровневая ошибка исполнения.
	HTTPStatusCode int    `json:"-"`    // HTTP статус код.
	Code           string `json:"code"` // Код ошибки.
}

// Render - рендерит ответ.
func (e *Response) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)

	return nil
}

// PreconditionFailed - возвращает ошибку 412.
func PreconditionFailed(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusPreconditionFailed,
		Code:           code,
	}
}

// UnprocessableEntity - возвращает ошибку 422.
func UnprocessableEntity(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusUnprocessableEntity,
		Code:           code,
	}
}

// InvalidRequest - возвращает ошибку 400.
// Неправильный запрос.
// Возникает тогда, когда к запросу переданы неверные параметры.
func InvalidRequest(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		Code:           code,
	}
}

// ErrRender - возвращает ошибку 422.
// Неправильный ответ от Renderer.
// Возникает тогда, когда рендереру не удается отрисовать ответ.
func ErrRender(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusUnprocessableEntity,
		Code:           code,
	}
}

// ResourceNotFound - возвращает ошибку 404.
// Не найден какой-то ресурс.
func ResourceNotFound(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusNotFound,
		Code:           code,
	}
}

// TooManyRequests - возвращает ошибку 429.
func TooManyRequests(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusTooManyRequests,
		Code:           code,
	}
}

// Internal - возвращает ошибку 500.
// Внутренняя ошибка сервера.
func Internal(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           code,
	}
}

// InvalidCredentials - возвращает ошибку 401.
// Неправильный логин и пароль.
func InvalidCredentials(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           code,
	}
}

// AccessDenied - возвращает ошибку 403.
// Недостаточно прав.
func AccessDenied(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusForbidden,
		Code:           code,
	}
}

// TokenNotFound - возвращает ошибку 404.
// Нет такого токена.
func TokenNotFound(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusNotFound,
		Code:           code,
	}
}

// Conflict - возвращает ошибку 409.
// Конфликт данных.
func Conflict(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusConflict,
		Code:           code,
	}
}

// ConflictPhone - возвращает ошибку 409.
// Конфликт номера телефона.
func ConflictPhone(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusConflict,
		Code:           code,
	}
}

// ConflictEmail - возвращает ошибку 409.
// Конфликт email.
func ConflictEmail(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusConflict,
		Code:           code,
	}
}

// Unauthorized - возвращает ошибку 401.
// Не авторизован.
func Unauthorized(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           code,
	}
}

// BadRequest - возвращает ошибку 400.
// Неправильный запрос.
func BadRequest(err error, code string) render.Renderer {
	return &Response{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		Code:           code,
	}
}

// Custom - возвращает ошибку с заданным кодом.
// Кастомная ошибка.
func Custom(err error, httpCode int, code string) render.Renderer {
	renderer := &Response{
		Err:            err,
		HTTPStatusCode: httpCode,
		Code:           code,
	}

	return renderer
}

package apperrors

import (
	"net/http"
)

// ErrorType описывает доменную категорию ошибки.
// Используется для единообразного маппинга в HTTP-коды и принятия решений в адаптерах.
type ErrorType string

const (
	// InvalidArgumentErrorType — невалидный аргумент/данные запроса:
	// значение вне допустимого диапазона, не входит в список, неверный формат и т.п.
	InvalidArgumentErrorType ErrorType = "INVALID_ARGUMENT"

	// NotFoundErrorType — сущность/ресурс не найден(а).
	NotFoundErrorType ErrorType = "NOT_FOUND"

	// StateMismatchErrorType — операция не допускается для текущего состояния сущности
	// (например, действие над заказом в неподходящем статусе).
	StateMismatchErrorType ErrorType = "STATE_MISMATCH"

	// StateTransitionDeniedErrorType — запрет на переход состояния (нарушение графа переходов),
	// например попытка перейти из A в B, когда такой переход недопустим.
	StateTransitionDeniedErrorType ErrorType = "STATE_TRANSITION_DENIED"

	// UnauthorizedErrorType — отсутствует аутентификация или недействительные учётные данные.
	UnauthorizedErrorType ErrorType = "UNAUTHORIZED"

	// AccessDeniedErrorType — аутентификация есть, но недостаточно прав для доступа.
	AccessDeniedErrorType ErrorType = "ACCESS_DENIED"

	// BlockedErrorType — доступ запрещён по бизнес-причине (бан, блокировка сессии и т.п.).
	BlockedErrorType ErrorType = "BLOCKED"

	// ConflictErrorType — ошибка конкурентности.
	ConflictErrorType ErrorType = "CONFLICT"

	// TooManyRequestsErrorType — превышена допустимая частота обращений.
	// Отдельно от ConflictErrorType: конфликт означает несовместимое состояние
	// данных и не проходит при повторе, а превышение частоты проходит — нужно
	// лишь подождать, и клиент обязан различать эти два случая.
	TooManyRequestsErrorType ErrorType = "TOO_MANY_REQUESTS"

	// InternalErrorType — внутренняя ошибка системы (неожиданная, инфраструктурная).
	InternalErrorType ErrorType = "INTERNAL"
)

// HTTPStatusCode возвращает HTTP-статус, соответствующий категории ошибки.
func (e ErrorType) HTTPStatusCode() int {
	switch e {
	case InvalidArgumentErrorType, StateMismatchErrorType, StateTransitionDeniedErrorType, BlockedErrorType:
		return http.StatusBadRequest // 400
	case UnauthorizedErrorType:
		return http.StatusUnauthorized // 401
	case AccessDeniedErrorType:
		return http.StatusForbidden // 403
	case NotFoundErrorType:
		return http.StatusNotFound // 404
	case ConflictErrorType:
		return http.StatusConflict // 409
	case TooManyRequestsErrorType:
		return http.StatusTooManyRequests // 429
	case InternalErrorType:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError // 500
	}
}

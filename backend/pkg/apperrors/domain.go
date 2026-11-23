package apperrors

import (
	"errors"
	"fmt"
	"strings"
)

// DomainError — единый доменный тип ошибки.
// Она не привязана к транспортам, но содержит достаточно метаданных.
type DomainError struct {
	ErrorType   ErrorType // Доменная категория (например: INVALID_ARGUMENT / NOT_FOUND / STATE_MISMATCH ...).
	Code        string    // Стабильный публичный код в формате <CodePrefix>_<SNAKE_CASE>, например: "CDL_ORDER_NOT_FOUND".
	Description string    // Краткое безопасное сообщение для логов.
	Retryable   bool      // Подсказка для брокер-адаптера: true => повтор не поможет; false => временная проблема.
}

// Error реализует интерфейс error и возвращает человекочитаемое описание.
func (e *DomainError) Error() string {
	return e.Description
}

// NewDomainError создаёт доменную ошибку с единым форматом кода.
func NewDomainError(errorType ErrorType, prefix, code, description string, retryable bool) error {
	if !strings.HasPrefix(code, prefix) {
		code = prefix + code
	}

	return &DomainError{
		ErrorType:   errorType,
		Code:        code,
		Description: description,
		Retryable:   retryable,
	}
}

// UnwrapCodeError возвращает код ошибки из ошибки, если это возможно.
func UnwrapCodeError(err error) string {
	if err == nil {
		return ""
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return ""
	}

	return domainErr.Code
}

// ExternalServiceError описывает ошибку, возвращенную внешним HTTP сервисом.
type ExternalServiceError struct {
	ServiceCode    string `json:"code"` // Код ошибки внешнего сервиса.
	HTTPStatusCode int    `json:"-"`    // HTTP статус ответа внешнего сервиса.
}

// Error возвращает текст ошибки внешнего HTTP сервиса.
func (e *ExternalServiceError) Error() string {
	return fmt.Sprintf("http-статус: %d, код ошибки: '%s'", e.HTTPStatusCode, e.ServiceCode)
}

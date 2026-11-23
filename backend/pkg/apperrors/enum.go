package apperrors

import (
	"strings"
	"sync"
)

// once защищает однократную установку глобального префикса кодов ошибок.
var once sync.Once

var (
	// CodePrefix описывает префикс кода ошибки, который будет добавлен к коду ошибки при форматировании.
	CodePrefix = ""

	// DefaultExternalEmptyCode описывает код ошибки, возвращаемый внешним HTTP сервисом по умолчанию.
	DefaultExternalEmptyCode = "DEFAULT_EXTERNAL_EMPTY_CODE"

	// InternalCode описывает код ошибки, возвращаемый внутренним сервисом по умолчанию.
	InternalCode = "INTERNAL"

	// FormDecodeCode описывает код ошибки, возвращаемый при ошибке декодирования формы.
	FormDecodeCode = "FORM_DECODE"

	// ErrInternal ошибка внутреннего сервера.
	ErrInternal = internalError()

	// ErrFormDecode ошибка декодирования формы.
	ErrFormDecode = formDecodeError()
)

// SetGlobalCodePrefixOnce устанавливает глобальный префикс кода ошибки и обновляет код ошибки по умолчанию для внешнего HTTP сервиса.
func SetGlobalCodePrefixOnce(prefix string) {
	once.Do(func() {
		CodePrefix = prefix

		DefaultExternalEmptyCode = addCodePrefix(prefix, DefaultExternalEmptyCode)
		InternalCode = addCodePrefix(prefix, InternalCode)
		FormDecodeCode = addCodePrefix(prefix, FormDecodeCode)

		ErrInternal = internalError()
		ErrFormDecode = formDecodeError()
	})
}

// addCodePrefix добавляет префикс к коду ошибки, если он еще не добавлен.
func addCodePrefix(prefix, code string) string {
	if strings.HasPrefix(code, prefix) {
		return code
	}

	return prefix + code
}

// internalError ошибка внутреннего сервера.
func internalError() error {
	return NewDomainError(
		InternalErrorType,
		CodePrefix,
		InternalCode,
		"внутренняя ошибка сервиса",
		true,
	)
}

// formDecodeError ошибка декодирования формы.
func formDecodeError() error {
	return NewDomainError(
		InvalidArgumentErrorType,
		CodePrefix,
		FormDecodeCode,
		"не удалось декодировать входной запрос",
		false,
	)
}

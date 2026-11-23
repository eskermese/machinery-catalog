package validate

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// Rules содержит правила валидации.
type Rules struct {
	CommandErrorCode string // Общий код ошибки для всех полей
}

// HasCommandErrorCode проверяет что CommandErrorCode не пустой.
func (r Rules) HasCommandErrorCode() bool {
	return r.CommandErrorCode != ""
}

// Validator реализует интерфейс Validate.
type Validator struct {
	shortServiceName string              // Короткое название сервиса
	validate         *validator.Validate // Валидатор на основе библиотеки go-playground/validator
	codes            Codes               // Коды ошибок валидации
	mu               sync.RWMutex        // Защита от одновременного изменения кодов ошибок
	rules            Rules               // Правила валидации
}

// Проверка реализации интерфейса.
var _ Validate = (*Validator)(nil)

// New создает новый экземпляр валидатора.
func New(shortServiceName string) *Validator {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		n := 2

		name := strings.SplitN(fld.Tag.Get("json"), ",", n)
		if len(name) == 0 || name[0] == "-" {
			return ""
		}

		return name[0]
	})

	return &Validator{
		shortServiceName: shortServiceName,
		validate:         validate,
		codes:            defaultCodes(),
		mu:               sync.RWMutex{},
	}
}

// Validate выполняет валидацию структуры.
// st - структура для валидации. errCode - код ошибки, который будет возвращен в случае ошибки валидации, опциональный параметр.
// Если валидация прошла успешно, то возвращается nil.
// Если валидация не прошла, то возвращается ошибка типа ValidationError.
func (v *Validator) Validate(st interface{}) error {
	err := v.validate.Struct(st)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) || len(validationErrors) == 0 {
		return err
	}

	// Получаем первую ошибку, так как мы отдаем клиенту только один код ошибки
	e := validationErrors[0]
	fieldName := e.Field()

	if v.rules.HasCommandErrorCode() {
		return v.getValidationError(e, fieldName, v.rules.CommandErrorCode)
	}

	validationTag := e.Tag()

	code, ok := v.codes[validationTag]
	if !ok {
		code = strings.ToUpper(validationTag)
	}

	return v.getValidationError(e, fieldName, v.generateCode(fieldName, code))
}

// getValidationError создает ошибку валидации с заданным кодом.
func (v *Validator) getValidationError(e error, fieldName, code string) error {
	if e == nil {
		return nil
	}

	return ValidationError{
		Err:         e,
		Field:       fieldName,
		Code:        code,
		Description: describeFieldError(e, fieldName),
	}
}

// describeFieldError собирает описание по конкретной ошибке поля.
//
// Правило и его параметр известны только типу validator.FieldError, поэтому
// они читаются здесь, а не передаются через всю цепочку вызовов.
func describeFieldError(e error, fieldName string) string {
	var fieldErr validator.FieldError
	if !errors.As(e, &fieldErr) {
		return describeViolation(fieldName, "", "")
	}

	return describeViolation(fieldName, fieldErr.Tag(), fieldErr.Param())
}

// AddCodes добавляет коды в валидатор.
func (v *Validator) AddCodes(m Codes) {
	v.mu.Lock()
	defer v.mu.Unlock()

	for code, msg := range m {
		v.codes[code] = msg
	}
}

// SetCommonRuleErrorCode устанавливает общий код ошибки для всех полей.
func (v *Validator) SetCommonRuleErrorCode(errorCode string) Validate {
	v.rules.CommandErrorCode = errorCode

	return v
}

// Codes возвращает коды валидатора.
func (v *Validator) Codes() Codes {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.codes
}

// RegisterValidation добавляет функцию валидации.
func (v *Validator) RegisterValidation(tag string, fn Func, callValidationEvenIfNull ...bool) error {
	if fn == nil {
		return ErrFunctionEmpty
	}

	return v.validate.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return fn(fl)
	}, callValidationEvenIfNull...)
}

// generateCode генерирует код ошибки.
// Пример: SNT_NAME_REQUIRED
// TMP - краткое название сервиса
// NAME - название поля
// REQUIRED - описание ошибки.
func (v *Validator) generateCode(fieldName, errorCode string) string {
	b := strings.Builder{}
	b.WriteString(v.shortServiceName)

	if fieldName != "" {
		b.WriteString("_")
		b.WriteString(strings.ToUpper(fieldName))
	}

	if errorCode != "" {
		b.WriteString("_")
		b.WriteString(strings.ToUpper(errorCode))
	}

	return b.String()
}

package validate

import (
	"reflect"
)

// Validate интерфейс валидатора.
type Validate interface {
	// Validate выполняет валидацию структуры.
	Validate(st interface{}) error
	// AddCodes добавляет коды сообщений в валидатор.
	AddCodes(m Codes)
	// Codes возвращает коды валидатора.
	Codes() Codes
	// RegisterValidation добавляет функцию валидации.
	RegisterValidation(tag string, fn Func, callValidationEvenIfNull ...bool) error
	// SetCommonRuleErrorCode устанавливает общий код ошибки для всех полей.
	SetCommonRuleErrorCode(errorCode string) Validate
}

// Func тип функции валидатора.
type Func func(fl FieldLevel) bool

// FieldLevel интерфейс уровня поля структуры.
type FieldLevel interface {
	// Top возвращает верхний уровень структуры.
	Top() reflect.Value
	// Parent возвращает родительский уровень структуры.
	Parent() reflect.Value
	// Field возвращает текущее поле структуры.
	Field() reflect.Value
	// FieldName возвращает имя текущего поля структуры.
	FieldName() string
	// StructFieldName возвращает имя текущего поля структуры.
	StructFieldName() string
	// Param возвращает параметр валидации.
	Param() string
	// GetTag возвращает тег валидации.
	GetTag() string
	// ExtractType возвращает тип поля.
	ExtractType(field reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool)
	// GetStructFieldOK возвращает поле структуры.
	GetStructFieldOK() (reflect.Value, reflect.Kind, bool)
	// GetStructFieldOKAdvanced возвращает поле структуры.
	GetStructFieldOKAdvanced(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool)
	// GetStructFieldOK2 возвращает поле структуры.
	GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool)
	// GetStructFieldOKAdvanced2 возвращает поле структуры.
	GetStructFieldOKAdvanced2(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool, bool)
}

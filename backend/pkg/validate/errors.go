package validate

import (
	"errors"
	"strings"
)

// ErrFunctionEmpty сообщает, что функция валидации не передана.
var ErrFunctionEmpty = errors.New("функция не может быть пустой")

// ValidationError ошибка валидации.
type ValidationError struct {
	Err   error  `json:"-"`    // Исходная ошибка валидации.
	Field string `json:"-"`    // Имя поля с ошибкой.
	Code  string `json:"code"` // Код ошибки для клиента.
	// Description человекочитаемое описание для показа пользователю.
	//
	// Собирается из имени поля и нарушенного правила, а не берётся
	// из Err: текст go-playground технический, на английском и содержит
	// путь к полю структуры («Key: 'QuoteIssue.MachineSlug' Error:Field
	// validation for 'MachineSlug' failed on the 'required' tag»).
	Description string `json:"-"`
}

// Error возвращает текст ошибки.
func (e ValidationError) Error() string {
	return e.Err.Error()
}

// describeViolation собирает описание нарушенного правила для человека.
//
// Имя поля берётся в том виде, в каком оно уходит в JSON: валидатор
// зарегистрирован с RegisterTagNameFunc по тегу json, поэтому клиент видит
// в описании то же имя, что и в теле запроса.
func describeViolation(field, tag, param string) string {
	phrase, ok := defaultMessages()[tag]
	if !ok {
		return "поле " + field + ": значение не прошло проверку"
	}

	if param != "" && strings.Contains(phrase, "%s") {
		phrase = strings.ReplaceAll(phrase, "%s", param)
	}

	return "поле " + field + ": " + phrase
}

// defaultMessages описывает нарушенные правила по-русски.
//
// Ключи совпадают с defaultCodes: код сообщает машине, ЧТО нарушено,
// описание — человеку. Держать их рядом обязательно, иначе одно
// разойдётся со вторым при следующем добавлении правила.
//
// «%s» подставляется параметром правила: gte=0 даёт «должно быть больше
// или равно 0», oneof=annuity differentiated — перечень допустимых значений.
func defaultMessages() map[string]string {
	return map[string]string{
		"required":    "обязательно",
		"min":         "должно быть не меньше %s",
		"max":         "должно быть не больше %s",
		"email":       "должно быть адресом электронной почты",
		"eqfield":     "должно совпадать с полем %s",
		"oneof":       "должно быть одним из: %s",
		"unique":      "значения должны быть уникальными",
		"len":         "длина должна быть равна %s",
		"gte":         "должно быть больше или равно %s",
		"lte":         "должно быть меньше или равно %s",
		"gt":          "должно быть больше %s",
		"lt":          "должно быть меньше %s",
		"alpha":       "должно содержать только буквы",
		"alphanum":    "должно содержать только буквы и цифры",
		"numeric":     "должно быть числом",
		"hexadecimal": "должно быть шестнадцатеричным числом",
		"hexcolor":    "должно быть цветом в формате HEX",
		"rgb":         "должно быть цветом RGB",
		"json":        "должно быть корректным JSON",
		"jwt":         "должно быть токеном JWT",
		"longitude":   "должно быть долготой",
		"latitude":    "должно быть широтой",
		"uuid":        "должно быть UUID",
		"uuid4":       "должно быть UUID версии 4",
		"datetime":    "должно быть датой в формате %s",
		"required_if": "обязательно при заданном %s",
		"base64":      "должно быть строкой base64",
		"base64url":   "должно быть строкой base64url",
		"cron":        "должно быть выражением cron",
		"file":        "должно быть файлом",
		"image":       "должно быть изображением",
		"dir":         "должно быть каталогом",
		"dirpath":     "должно быть путём к каталогу",
		"dive":        "значение внутри списка не прошло проверку",
	}
}

type (
	// Codes связывает теги валидации с кодами ошибок для клиента.
	Codes map[string]string
)

// defaultCodes возвращает сообщения об ошибках по умолчанию.
func defaultCodes() Codes {
	return Codes{
		"required":    "REQUIRED",      // обязательное поле
		"min":         "MIN",           // должно быть больше или равно
		"max":         "MAX",           // должно быть меньше или равно
		"email":       "INVALID_EMAIL", // должно быть email
		"eqfield":     "EQFIELD",       // должно быть равно полю
		"oneof":       "ONEOF",         // должно быть одним из
		"unique":      "UNIQUE",        // должно быть уникальным
		"len":         "LEN",           // должно быть равно
		"gte":         "GTE",           // должно быть больше или равно
		"lte":         "LTE",           // должно быть меньше или равно
		"gt":          "GT",            // должно быть больше
		"lt":          "LT",            // должно быть меньше
		"alpha":       "ALPHA",         // должно содержать только буквы
		"alphanum":    "ALPHANUM",      // должно содержать только буквы и цифры
		"numeric":     "NUMERIC",       // должно быть числом
		"hexadecimal": "HEXADECIMAL",   // должно быть шестнадцатеричным числом
		"hexcolor":    "HEXCOLOR",      // должно быть цветом в HEX формате
		"rgb":         "RGB",           // должно быть RGB цветом
		"json":        "JSON",          // должно быть JSON
		"jwt":         "JWT",           // должно быть JWT
		"longitude":   "LONGITUDE",     // должно быть долготой
		"latitude":    "LATITUDE",      // должно быть широтой
		"uuid":        "UUID",          // должно быть UUID
		"datetime":    "DATETIME",      // должно быть датой
		"required_if": "REQUIRED_IF",   // обязательное поле, если другое поле равно
		"base64":      "BASE64",        // должно быть base64
		"base64url":   "BASE64URL",     // должно быть base64url
		"cron":        "CRON",          // должно быть cron выражением
		"file":        "FILE",          // должно быть файлом
		"image":       "IMAGE",         // должно быть изображением
		"dir":         "DIR",           // должно быть директорией
		"dirpath":     "DIRPATH",       // должно быть путем к директории
	}
}

// Package swagger содержит метаданные тел ответов, на которые ссылаются
// swagger-аннотации обработчиков: сгенерировать их из httpx нельзя, потому что
// errorResponse там неэкспортируемый.
package swagger

// HTTPResponse400 структура, которая отображается как тело ответа при 400 коде возврата от HTTP.
type HTTPResponse400 struct {
	Code    string `json:"code" validate:"required" example:"SNT_SLUG_ALREADY_USED"` // Стабильный код ошибки.
	Message string `json:"message" validate:"required" example:"слаг уже занят"`     // Описание ошибки для человека.
}

// HTTPResponse500 структура, которая отображается как тело ответа при 500 коде возврата от HTTP.
//
// Описание здесь всегда общее: подробности внутреннего сбоя уходят в лог,
// а не клиенту (см. httpx.clientMessage).
type HTTPResponse500 struct {
	Code    string `json:"code" validate:"required" example:"SNT_INTERNAL"`                 // Стабильный код ошибки.
	Message string `json:"message" validate:"required" example:"внутренняя ошибка сервиса"` // Описание ошибки для человека.
}

package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/schema"
	jsoniter "github.com/json-iterator/go"

	"github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
	"github.com/eskermese/machinery-catalog/backend/pkg/logger"
	"github.com/eskermese/machinery-catalog/backend/pkg/validate"
)

// Decoder декодирует query/form-параметры в DTO.
var Decoder = newDecoder()

// newDecoder создает decoder для form/query-параметров.
func newDecoder() *schema.Decoder {
	decoder := schema.NewDecoder()
	decoder.SetAliasTag("form")
	decoder.IgnoreUnknownKeys(true)

	return decoder
}

// Тексты ответов, которые нельзя собрать из доменной ошибки.
//
// Константы, а не строки из err.Error(): единственный способ поместить
// в ответ переменный текст — Description доменной ошибки, всё остальное
// приходит отсюда. Формулировки общие намеренно — подробности сбоя
// диагностируют по логу, а не по ответу клиенту.
const (
	// messageInternal уходит клиенту при любом сбое на стороне сервиса.
	messageInternal = "внутренняя ошибка сервиса"
	// messageExternal уходит, когда ошибку вернул внешний сервис: его
	// формулировки — не наш контракт и клиенту ничего не объясняют.
	messageExternal = "внешний сервис вернул ошибку"
)

// errorResponse описывает HTTP-ответ с ошибкой.
type errorResponse struct {
	// Code содержит стабильный код ошибки для клиента.
	Code string `json:"code" example:"SNT_MACHINE_NOT_FOUND"`
	// Message содержит человекочитаемое описание ошибки.
	//
	// Заполняется из DomainError.Description — текста, написанного
	// в internal/domain/entity/errors.go специально для показа человеку.
	// Берётся именно Description, а не err.Error(): сервисы оборачивают
	// ошибки контекстом («получение карточки: …»), и Error() вынес бы
	// наружу внутренние имена операций.
	Message string `json:"message" example:"карточка техники не найдена"`
}

// DecodeJSON декодирует JSON из io.Reader.
func DecodeJSON(r io.Reader, value any) error {
	decoder := jsoniter.NewDecoder(r)
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("%w: %w", apperrors.ErrFormDecode, err)
	}

	return nil
}

// DecodeForm декодирует query/form-параметры в DTO.
func DecodeForm(query url.Values, value any) error {
	if err := Decoder.Decode(value, query); err != nil {
		return fmt.Errorf("%w: %w", apperrors.ErrFormDecode, err)
	}

	return nil
}

// WriteJSON записывает JSON-ответ в HTTP-ответ.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := jsoniter.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

// WriteError преобразует ошибку приложения в HTTP-ответ.
func WriteError(w http.ResponseWriter, err error) {
	status, payload := mapError(err)
	WriteJSON(w, status, payload)
}

// LogRequestError записывает ошибку обработчика с HTTP-контекстом.
func LogRequestError(r *http.Request, message string, err error, fields logger.Fields) {
	status, payload := mapError(err)
	logFields := logger.Fields{
		"status_code": status,
		"code":        payload.Code,
		"method":      r.Method,
		"path":        r.URL.Path,
	}

	for key, value := range requestQueryLogFields(r) {
		logFields[key] = value
	}

	for key, value := range fields {
		logFields[key] = value
	}

	logFields["error"] = err

	log := logger.FromContext(r.Context()).WithFields(logFields)
	if status >= http.StatusInternalServerError || status == http.StatusConflict {
		log.Error(message)

		return
	}

	log.Warn(message)
}

// LogError записывает ошибку без привязки к пути запроса.
//
// Нужна там, где сам путь секретен: у публичной ссылки на коммерческое
// предложение весь путь состоит из токена, и LogRequestError записал бы его
// в лог вместе с ошибкой. Поля вызывающий код задаёт сам — ровно те,
// которые безопасно хранить.
func LogError(ctx context.Context, message string, err error, fields map[string]any) {
	status, payload := mapError(err)

	logFields := logger.Fields{
		"status_code": status,
		"code":        payload.Code,
		"error":       err,
	}

	for key, value := range fields {
		logFields[key] = value
	}

	log := logger.FromContext(ctx).WithFields(logFields)
	if status >= http.StatusInternalServerError {
		log.Error(message)

		return
	}

	log.Warn(message)
}

// requestQueryLogFields собирает query-параметры запроса для диагностики ошибок парсинга.
func requestQueryLogFields(r *http.Request) logger.Fields {
	fields := logger.Fields{
		"raw_query": "",
	}
	if r == nil || r.URL == nil {
		return fields
	}

	fields["raw_query"] = r.URL.RawQuery
	if r.URL.RawQuery != "" {
		fields["query_params"] = r.URL.Query()
	}

	return fields
}

// mapError преобразует ошибку в структуру для ответа клиенту.
func mapError(err error) (int, errorResponse) {
	var validationErrPtr *validate.ValidationError
	if errors.As(err, &validationErrPtr) && validationErrPtr != nil {
		return http.StatusBadRequest, errorResponse{
			Code:    validationErrPtr.Code,
			Message: clientMessage(validationErrPtr.Description, http.StatusBadRequest),
		}
	}

	var validationErr validate.ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest, errorResponse{
			Code:    validationErr.Code,
			Message: clientMessage(validationErr.Description, http.StatusBadRequest),
		}
	}

	var externalErr *apperrors.ExternalServiceError
	if errors.As(err, &externalErr) && isErrorHTTPStatus(externalErr.HTTPStatusCode) {
		code := externalErr.ServiceCode
		if code == "" {
			code = apperrors.DefaultExternalEmptyCode
		}

		return externalErr.HTTPStatusCode, errorResponse{Code: code, Message: messageExternal}
	}

	status := http.StatusInternalServerError
	code := apperrors.InternalCode
	message := messageInternal

	var appErr *apperrors.DomainError
	if errors.As(err, &appErr) {
		code = appErr.Code
		status = appErr.ErrorType.HTTPStatusCode()
		message = clientMessage(appErr.Description, status)
	}

	return status, errorResponse{Code: code, Message: message}
}

// clientMessage выбирает текст, который безопасно показать клиенту.
//
// Описание ошибки уходит наружу только у ответов со статусом ниже 500.
// Отбор по статусу, а не по списку исключений: описания внутренних ошибок
// пишутся для того, кто читает лог, — «database common», «значение не может
// быть nil», «канал доставки уведомлений не настроен». Клиенту они ничего
// не объясняют, зато рассказывают об устройстве сервиса, и держать их
// в стороне поштучно значило бы полагаться на внимательность автора
// следующей ошибки.
func clientMessage(description string, status int) string {
	if status >= http.StatusInternalServerError || description == "" {
		return messageInternal
	}

	return description
}

// isErrorHTTPStatus проверяет, что статус можно вернуть как HTTP-ошибку.
func isErrorHTTPStatus(status int) bool {
	const (
		// maxErrorHTTPStatusExclusive ограничивает валидный диапазон HTTP-ошибок. Нужно для линтера
		maxErrorHTTPStatusExclusive = 600
	)

	return status >= http.StatusBadRequest && status < maxErrorHTTPStatusExclusive
}

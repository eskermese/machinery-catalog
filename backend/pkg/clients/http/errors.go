package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"resty.dev/v3"

	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// serviceInternalErrorCode возвращает код ошибки для неразобранного ответа
// внешнего сервиса.
//
// Читается на месте вызова, а не константой пакета: префикс кодов задаётся
// один раз при инициализации домена (apperrors.SetGlobalCodePrefixOnce),
// и значение, снятое на этапе инициализации переменных пакета, оказалось бы
// без префикса. Раньше здесь стоял литерал с чужим префиксом — и он не просто
// украшал спеку, а уходил клиенту кодом ответа при сбое внешнего сервиса.
func serviceInternalErrorCode() string {
	return apperrors.InternalCode
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var ErrEmptyResponse = errors.New("пустой ответ сервиса")

// NewExternalServiceErrorFromHTTPResponse возвращает новый экземпляр ExternalServiceError из http.Response.
func NewExternalServiceErrorFromHTTPResponse(resp *http.Response) (*apperrors.ExternalServiceError, error) {
	se := &apperrors.ExternalServiceError{
		HTTPStatusCode: http.StatusInternalServerError,
		ServiceCode:    serviceInternalErrorCode(),
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return se, fmt.Errorf("чтение тела ответа: %w", err)
	}

	if len(body) == 0 {
		return se, ErrEmptyResponse
	}

	if err = json.Unmarshal(body, se); err != nil {
		return se, fmt.Errorf("не удалось распарсить тело ответа: %w; body: %s", err, string(body))
	}

	se.HTTPStatusCode = resp.StatusCode

	return se, nil
}

// requestMiddleware логгирует все исходящие запросы.
func requestMiddleware(_ *resty.Client, r *resty.Request) error {
	ctx := r.Context()

	// Логируем метод и URL
	log := logger.FromContext(ctx)
	url := r.URL

	if r.RawRequest != nil && r.RawRequest.URL != nil {
		url = r.RawRequest.URL.String()
	}

	log.WithFields(logger.Fields{
		"method":       r.Method,
		"url":          url,
		"request_body": r.Body,
		"headers":      r.Header,
	}).Info("outgoing http request")

	return nil
}

// responseMiddleware логгирует все ответы.
func responseMiddleware(_ *resty.Client, resp *resty.Response) error {
	ctx := resp.Request.Context()
	log := logger.FromContext(ctx)

	url := resp.Request.URL
	if resp.Request.RawRequest != nil && resp.Request.RawRequest.URL != nil {
		url = resp.Request.RawRequest.URL.String()
	}

	log.WithFields(logger.Fields{
		"method": resp.Request.Method,
		"url":    url,
		"status": resp.StatusCode(),
	}).Info("incoming http response")

	return nil
}

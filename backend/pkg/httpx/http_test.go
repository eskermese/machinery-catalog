package v1

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
	"github.com/eskermese/machinery-catalog/backend/pkg/validate"
)

// TestMapError_DomainErrorCarriesDescription фиксирует, что человеческий текст
// доменной ошибки доходит до клиента: ради этого поле message и заведено —
// иначе потребителю пришлось бы держать у себя копию словаря описаний.
func TestMapError_DomainErrorCarriesDescription(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		errorType apperrors.ErrorType
		expStatus int
	}{
		{name: "не найдено", errorType: apperrors.NotFoundErrorType, expStatus: http.StatusNotFound},
		{name: "невалидный аргумент", errorType: apperrors.InvalidArgumentErrorType, expStatus: http.StatusBadRequest},
		{name: "несоответствие состояния", errorType: apperrors.StateMismatchErrorType, expStatus: http.StatusBadRequest},
		{name: "запрет перехода", errorType: apperrors.StateTransitionDeniedErrorType, expStatus: http.StatusBadRequest},
		{name: "блокировка", errorType: apperrors.BlockedErrorType, expStatus: http.StatusBadRequest},
		{name: "не аутентифицирован", errorType: apperrors.UnauthorizedErrorType, expStatus: http.StatusUnauthorized},
		{name: "нет прав", errorType: apperrors.AccessDeniedErrorType, expStatus: http.StatusForbidden},
		{name: "конфликт", errorType: apperrors.ConflictErrorType, expStatus: http.StatusConflict},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := apperrors.NewDomainError(c.errorType, "SNT_", "SOMETHING", "карточка техники не найдена", false)

			status, payload := mapError(err)

			assert.Equal(t, c.expStatus, status)
			assert.Equal(t, "SNT_SOMETHING", payload.Code)
			assert.Equal(t, "карточка техники не найдена", payload.Message)
		})
	}
}

// TestMapError_InternalDescriptionNeverLeaks — главная проверка правки.
//
// Описания внутренних ошибок пишутся для того, кто читает лог, и рассказывают
// об устройстве сервиса. Наружу вместо них обязан уходить общий текст,
// а отбор идёт по статусу, а не по списку исключений: иначе безопасность
// ответа зависела бы от внимательности автора следующей ошибки.
func TestMapError_InternalDescriptionNeverLeaks(t *testing.T) {
	t.Parallel()

	// Формулировки взяты из реальных внутренних ошибок сервиса.
	descriptions := []string{
		"database common",
		"значение не может быть nil",
		"канал доставки уведомлений не настроен",
		"источник курса не настроен",
	}

	for _, description := range descriptions {
		description := description

		t.Run(description, func(t *testing.T) {
			t.Parallel()

			err := apperrors.NewDomainError(apperrors.InternalErrorType, "SNT_", "INTERNAL", description, false)

			status, payload := mapError(err)

			require.Equal(t, http.StatusInternalServerError, status)
			assert.Equal(t, messageInternal, payload.Message)
			assert.NotContains(t, payload.Message, description, "внутреннее описание наружу не уходит")
		})
	}
}

// TestMapError_UnknownErrorIsInternal фиксирует, что ошибка, не являющаяся
// доменной, отдаётся общим текстом: её Error() — это стек чужих обёрток.
func TestMapError_UnknownErrorIsInternal(t *testing.T) {
	t.Parallel()

	err := errors.New("pq: relation \"machines\" does not exist")

	status, payload := mapError(err)

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, apperrors.InternalCode, payload.Code)
	assert.Equal(t, messageInternal, payload.Message)
	assert.NotContains(t, payload.Message, "machines", "текст исходной ошибки наружу не уходит")
}

// TestMapError_WrappingContextStaysInside фиксирует, что в ответ уходит
// Description доменной ошибки, а не err.Error().
//
// Сервисы оборачивают ошибки контекстом операции — «получение карточки: …».
// Error() вынес бы наружу внутренние имена шагов сценария, поэтому берётся
// поле распакованной ошибки.
func TestMapError_WrappingContextStaysInside(t *testing.T) {
	t.Parallel()

	domainErr := apperrors.NewDomainError(
		apperrors.NotFoundErrorType, "SNT_", "QUOTE_NOT_FOUND", "коммерческое предложение не найдено", false)
	wrapped := fmt.Errorf("получение предложения: %w", fmt.Errorf("чтение из хранилища: %w", domainErr))

	_, payload := mapError(wrapped)

	assert.Equal(t, "коммерческое предложение не найдено", payload.Message)
	assert.NotContains(t, payload.Message, "получение предложения")
	assert.NotContains(t, payload.Message, "чтение из хранилища")
}

// TestMapError_EmptyDescriptionFallsBack фиксирует, что доменная ошибка без
// описания не превращается в ответ с пустым message.
func TestMapError_EmptyDescriptionFallsBack(t *testing.T) {
	t.Parallel()

	err := apperrors.NewDomainError(apperrors.NotFoundErrorType, "SNT_", "NO_TEXT", "", false)

	status, payload := mapError(err)

	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, messageInternal, payload.Message)
}

// TestMapError_ValidationError фиксирует ответ на отказ валидатора: наружу
// уходит собранное описание нарушенного правила, а технический текст
// go-playground с путём к полю структуры — нет.
func TestMapError_ValidationError(t *testing.T) {
	t.Parallel()

	inner := errors.New("Key: 'QuoteIssue.MachineSlug' Error:Field validation failed on the 'required' tag")

	t.Run("значением", func(t *testing.T) {
		t.Parallel()

		status, payload := mapError(validate.ValidationError{
			Err: inner, Field: "machineSlug", Code: "REQUIRED",
			Description: "поле machineSlug: обязательно",
		})

		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "REQUIRED", payload.Code)
		assert.Equal(t, "поле machineSlug: обязательно", payload.Message)
		assert.NotContains(t, payload.Message, "QuoteIssue", "путь к полю структуры наружу не уходит")
	})

	t.Run("указателем", func(t *testing.T) {
		t.Parallel()

		status, payload := mapError(&validate.ValidationError{
			Err: inner, Field: "scheduleType", Code: "ONEOF",
			Description: "поле scheduleType: должно быть одним из: annuity differentiated",
		})

		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "ONEOF", payload.Code)
		assert.Equal(t, "поле scheduleType: должно быть одним из: annuity differentiated", payload.Message)
	})

	t.Run("без описания", func(t *testing.T) {
		t.Parallel()

		_, payload := mapError(validate.ValidationError{Err: inner, Field: "x", Code: "REQUIRED"})

		assert.Equal(t, messageInternal, payload.Message, "пустое описание не даёт пустой message")
	})
}

// TestMapError_ExternalServiceError фиксирует, что формулировки внешнего
// сервиса не выдаются за свои.
func TestMapError_ExternalServiceError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		code    string
		status  int
		expCode string
	}{
		{name: "с кодом", code: "RATE_LIMIT", status: http.StatusTooManyRequests, expCode: "RATE_LIMIT"},
		{name: "без кода", code: "", status: http.StatusBadGateway, expCode: apperrors.DefaultExternalEmptyCode},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := &apperrors.ExternalServiceError{ServiceCode: c.code, HTTPStatusCode: c.status}

			status, payload := mapError(err)

			assert.Equal(t, c.status, status)
			assert.Equal(t, c.expCode, payload.Code)
			assert.Equal(t, messageExternal, payload.Message)
		})
	}
}

// TestMapError_ExternalErrorWithNonHTTPStatus фиксирует, что ошибка внешнего
// сервиса с бессмысленным статусом не превращает его в статус нашего ответа.
func TestMapError_ExternalErrorWithNonHTTPStatus(t *testing.T) {
	t.Parallel()

	err := &apperrors.ExternalServiceError{ServiceCode: "OK", HTTPStatusCode: http.StatusOK}

	status, payload := mapError(err)

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, messageInternal, payload.Message)
}

// TestWriteError фиксирует форму тела ответа целиком: оба поля на месте
// и названы так, как их ждёт потребитель спеки.
func TestWriteError(t *testing.T) {
	t.Parallel()

	err := apperrors.NewDomainError(
		apperrors.NotFoundErrorType, "SNT_", "MACHINE_NOT_FOUND", "карточка техники не найдена", false)

	w := httptest.NewRecorder()
	WriteError(w, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t,
		`{"code":"SNT_MACHINE_NOT_FOUND","message":"карточка техники не найдена"}`,
		w.Body.String())
}

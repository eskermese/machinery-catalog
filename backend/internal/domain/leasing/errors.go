package leasing

import (
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
)

// Ошибки валидации параметров расчёта.
//
// Объявлены здесь, а не в internal/domain/entity: калькулятор самодостаточен
// и не имеет других потребителей, кроме будущего сервиса лизинга, — держать его
// ошибки в общем перечне значило бы разносить одно понятие по двум пакетам.
// Конструктор и префикс кода при этом общие с остальным доменом, поэтому
// транспорт разворачивает их тем же mapError, что и все прочие доменные ошибки.
var (
	// ErrAssetCostInvalid сообщает о некорректной стоимости предмета лизинга.
	ErrAssetCostInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_ASSET_COST_INVALID", "стоимость предмета лизинга не может быть отрицательной", false)

	// ErrDownPaymentInvalid сообщает о первоначальном взносе вне диапазона 0..100%.
	ErrDownPaymentInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_DOWN_PAYMENT_INVALID", "первоначальный взнос должен быть в диапазоне от 0 до 100 процентов", false)

	// ErrTermInvalid сообщает о недопустимом сроке лизинга.
	ErrTermInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_TERM_INVALID", "срок лизинга должен быть положительным и не превышать допустимый предел", false)

	// ErrRateInvalid сообщает об отрицательной процентной ставке.
	ErrRateInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_RATE_INVALID", "годовая ставка не может быть отрицательной", false)

	// ErrResidualValueInvalid сообщает о недопустимой выкупной стоимости.
	ErrResidualValueInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_RESIDUAL_VALUE_INVALID", "выкупная стоимость отрицательна или превышает сумму финансирования", false)

	// ErrCommissionInvalid сообщает об отрицательной комиссии за организацию.
	ErrCommissionInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_COMMISSION_INVALID", "комиссия за организацию не может быть отрицательной", false)

	// ErrScheduleTypeInvalid сообщает о неизвестном типе графика платежей.
	ErrScheduleTypeInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"LEASING_SCHEDULE_TYPE_INVALID", "неизвестный тип графика платежей", false)
)

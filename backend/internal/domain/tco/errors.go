package tco

import (
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
)

// Ошибки валидации параметров расчёта стоимости владения.
//
// Объявлены здесь по тем же основаниям, что и ошибки лизинга: калькулятор
// самодостаточен, и держать его ошибки в общем перечне значило бы разносить
// одно понятие по двум пакетам. Конструктор и префикс кода общие с остальным
// доменом, поэтому транспорт разворачивает их тем же mapError.
//
// Здесь только ошибки ВВОДА. Незаведённые величины карточки ошибкой не
// считаются: расчёт по ним неполон, но не отказан, — см. Result.Missing.
var (
	// ErrPurchasePriceInvalid сообщает о неположительной цене покупки.
	ErrPurchasePriceInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_PURCHASE_PRICE_INVALID", "цена покупки должна быть положительной", false)

	// ErrYearsInvalid сообщает о недопустимом сроке владения.
	ErrYearsInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_YEARS_INVALID", "срок владения должен быть положительным и не превышать 30 лет", false)

	// ErrHoursPerYearInvalid сообщает о недопустимой годовой наработке.
	ErrHoursPerYearInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_HOURS_PER_YEAR_INVALID", "наработка за год должна быть положительной и не превышать число часов в году", false)

	// ErrFuelPriceInvalid сообщает об отрицательной цене топлива.
	ErrFuelPriceInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_FUEL_PRICE_INVALID", "цена топлива не может быть отрицательной", false)

	// ErrOperatorRateInvalid сообщает об отрицательной ставке оператора.
	ErrOperatorRateInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_OPERATOR_RATE_INVALID", "оплата оператора не может быть отрицательной", false)

	// ErrFuelConsumptionInvalid сообщает о неположительном расходе топлива.
	ErrFuelConsumptionInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_FUEL_CONSUMPTION_INVALID", "расход топлива должен быть положительным", false)

	// ErrServiceIntervalInvalid сообщает о неположительном интервале ТО.
	ErrServiceIntervalInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_SERVICE_INTERVAL_INVALID", "интервал технического обслуживания должен быть положительным", false)

	// ErrResidualRateInvalid сообщает о доле остаточной стоимости вне 0..100%.
	ErrResidualRateInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, entity.CodePrefix,
		"TCO_RESIDUAL_RATE_INVALID", "доля остаточной стоимости должна быть в диапазоне от 0 до 100 процентов", false)
)

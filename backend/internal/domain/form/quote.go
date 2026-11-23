package form

import (
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// QuoteIssue форма выпуска коммерческого предложения.
type QuoteIssue struct {
	// MachineSlug карточка, по которой выпускается предложение.
	MachineSlug string `json:"machineSlug" validate:"required" example:"xcmg-xe215c"`
	// Configuration выбранная комплектация и опции.
	Configuration map[string]any `json:"configuration"`

	// LeasingProgramID выбранная программа; пусто — расчёт по переданным параметрам.
	LeasingProgramID string `json:"leasingProgramID" validate:"omitempty,uuid4" example:""`
	// ScheduleType тип графика: annuity или differentiated.
	ScheduleType string `json:"scheduleType" validate:"required,oneof=annuity differentiated" example:"annuity"`
	// DownPaymentBps первоначальный взнос в базисных пунктах: 20.00% это 2000.
	DownPaymentBps int `json:"downPaymentBps" validate:"gte=0,lte=10000" example:"2000"`
	// TermMonths срок в месяцах.
	TermMonths int `json:"termMonths" validate:"required,gt=0" example:"36"`
	// RateBps годовая ставка в базисных пунктах: 18% это 1800.
	RateBps int `json:"rateBps" validate:"gte=0" example:"1800"`
	// ResidualValueKztCents выкупная стоимость в тиынах.
	ResidualValueKztCents int64 `json:"residualValueKztCents" validate:"gte=0" example:"0"`
	// CommissionKztCents разовая комиссия в тиынах.
	CommissionKztCents int64 `json:"commissionKztCents" validate:"gte=0" example:"0"`
	// ValidForDays срок годности предложения в днях; 0 — значение по умолчанию.
	ValidForDays int `json:"validForDays" validate:"gte=0,lte=365" example:"14"`
}

// Validate проверяет форму выпуска предложения.
func (f *QuoteIssue) Validate() error {
	return validatorDefault.Validate(f)
}

// LeadCreate форма заявки с сайта.
type LeadCreate struct {
	// Name имя клиента.
	Name string `json:"name" validate:"required" example:"Иван"`
	// Phone номер телефона в любом написании: нормализуется сервисом.
	Phone string `json:"phone" validate:"required" example:"+7 (700) 111-22-33"`
	// City город.
	City string `json:"city" example:"Алматы"`
	// MachineID машина, из карточки которой пришла заявка.
	MachineID string `json:"machineID" validate:"omitempty,uuid4" example:""`
	// QuoteID коммерческое предложение, по которому обратились.
	QuoteID string `json:"quoteID" validate:"omitempty,uuid4" example:""`
	// Source откуда пришла заявка: страница, форма или рекламная метка.
	Source string `json:"source" validate:"required" example:"machine_card"`
	// Message что написал клиент: какая деталь нужна, к какой технике,
	// что случилось. Необязательное.
	//
	// Отдельно от комментария менеджера: то поле принадлежит ему, и одно
	// на двоих означало бы, что его правка затирает запрос клиента.
	Message string `json:"message" validate:"omitempty,max=2000" example:"Нужен масляный фильтр на XE215C"`
}

// Validate проверяет форму заявки.
//
// Формат номера здесь не проверяется: его нормализует и проверяет домен —
// иначе правило написания пришлось бы держать в двух местах, и одно из них
// рано или поздно разошлось бы со вторым.
func (f *LeadCreate) Validate() error {
	return validatorDefault.Validate(f)
}

// LeadComment форма комментария менеджера к заявке.
//
// Комментарий менеджера, а не сообщение клиента: их поля разные, и эта форма
// до message не дотягивается. Пустая строка — законное значение: она стирает
// комментарий, а не оставляет прежний.
type LeadComment struct {
	// Comment заметка менеджера по заявке.
	Comment string `json:"comment" validate:"omitempty,max=2000" example:"Перезвонить после обеда"`
}

// Validate проверяет форму комментария.
func (f *LeadComment) Validate() error {
	return validatorDefault.Validate(f)
}

// LeadStatusChange форма смены статуса заявки.
type LeadStatusChange struct {
	// Status целевой статус.
	Status string `json:"status" validate:"required,oneof=new in_progress won lost" example:"in_progress"`
}

// Validate проверяет форму смены статуса.
func (f *LeadStatusChange) Validate() error {
	return validatorDefault.Validate(f)
}

// ToStatus возвращает целевой статус заявки.
func (f *LeadStatusChange) ToStatus() entity.LeadStatus {
	return entity.LeadStatus(f.Status)
}

// LeadAssign форма назначения ответственного менеджера.
type LeadAssign struct {
	// ManagerID ответственный менеджер.
	ManagerID string `json:"managerID" validate:"required,uuid4" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
}

// Validate проверяет форму назначения ответственного.
func (f *LeadAssign) Validate() error {
	return validatorDefault.Validate(f)
}

// LeasingProgramSave форма создания и изменения программы лизинга.
//
// Пустой идентификатор означает создание: разводить две почти одинаковые
// формы значило бы дважды описывать одни и те же границы.
type LeasingProgramSave struct {
	// ID программы; пусто — создание новой.
	ID string `json:"id" validate:"omitempty,uuid4" example:""`

	// NameRU название на русском.
	NameRU string `json:"nameRU" validate:"required" example:"Halyk Leasing"`
	// NameKK название на казахском.
	NameKK string `json:"nameKK" validate:"required" example:"Halyk Leasing"`
	// NameEN название на английском.
	NameEN string `json:"nameEN" validate:"required" example:"Halyk Leasing"`
	// LogoPath путь к логотипу партнёра внутри хранилища.
	LogoPath string `json:"logoPath" example:""`

	// MinDownPaymentBps минимальный первоначальный взнос в базисных пунктах.
	MinDownPaymentBps int `json:"minDownPaymentBps" validate:"gte=0,lte=10000" example:"1500"`
	// TermMonthsMin нижняя граница срока в месяцах.
	TermMonthsMin int `json:"termMonthsMin" validate:"required,gt=0" example:"12"`
	// TermMonthsMax верхняя граница срока в месяцах.
	TermMonthsMax int `json:"termMonthsMax" validate:"required,gt=0" example:"60"`
	// RateMinBps нижняя граница ставки в базисных пунктах.
	RateMinBps int `json:"rateMinBps" validate:"gte=0" example:"1800"`
	// RateMaxBps верхняя граница ставки в базисных пунктах.
	RateMaxBps int `json:"rateMaxBps" validate:"gte=0" example:"2400"`
	// MaxAmountCnyCents потолок суммы сделки в фынях (1/100 юаня); пусто — без потолка.
	MaxAmountCnyCents *int64 `json:"maxAmountCnyCents" validate:"omitempty,gt=0" example:"50000000"`

	// IsActive действует ли программа.
	IsActive bool `json:"isActive" example:"true"`
	// Position порядок вывода.
	Position int `json:"position" validate:"gte=0" example:"1"`
}

// Validate проверяет форму программы лизинга.
//
// Согласованность границ между собой здесь не проверяется: это правило самой
// программы, оно живёт в LeasingProgram.Validate — иначе одно и то же условие
// пришлось бы держать в двух местах.
func (f *LeasingProgramSave) Validate() error {
	return validatorDefault.Validate(f)
}

// ToEntity собирает программу лизинга из формы.
func (f *LeasingProgramSave) ToEntity() *entity.LeasingProgram {
	program := &entity.LeasingProgram{
		ID:                f.ID,
		NameRU:            f.NameRU,
		NameKK:            f.NameKK,
		NameEN:            f.NameEN,
		MinDownPaymentBps: f.MinDownPaymentBps,
		TermMonthsMin:     f.TermMonthsMin,
		TermMonthsMax:     f.TermMonthsMax,
		RateMinBps:        f.RateMinBps,
		RateMaxBps:        f.RateMaxBps,
		MaxAmountCNYCents: f.MaxAmountCnyCents,
		IsActive:          f.IsActive,
		Position:          f.Position,
	}

	if f.LogoPath != "" {
		program.LogoPath = entity.Pointer(f.LogoPath)
	}

	return program
}

// LeasingCalculate форма публичного расчёта лизинга.
//
// Цены здесь нет намеренно: её сервер берёт сам по слагу карточки. Приди она
// снаружи, любой посетитель получил бы «официальный» график от произвольной
// суммы и мог бы выдать его за расчёт сервиса.
type LeasingCalculate struct {
	// MachineSlug карточка, по цене которой считается платёж.
	MachineSlug string `json:"machineSlug" validate:"required" example:"xcmg-xe215c"`
	// LeasingProgramID выбранная программа; пусто — расчёт по переданным параметрам.
	LeasingProgramID string `json:"leasingProgramID" validate:"omitempty,uuid4" example:""`
	// ScheduleType тип графика: annuity или differentiated.
	ScheduleType string `json:"scheduleType" validate:"required,oneof=annuity differentiated" example:"annuity"`
	// DownPaymentBps первоначальный взнос в базисных пунктах: 20.00% это 2000.
	DownPaymentBps int `json:"downPaymentBps" validate:"gte=0,lte=10000" example:"2000"`
	// TermMonths срок в месяцах.
	TermMonths int `json:"termMonths" validate:"required,gt=0,lte=360" example:"36"`
	// RateBps годовая ставка в базисных пунктах: 18% это 1800.
	RateBps int `json:"rateBps" validate:"gte=0,lte=10000" example:"1800"`
	// ResidualValueKztCents выкупная стоимость в тиынах.
	ResidualValueKztCents int64 `json:"residualValueKztCents" validate:"gte=0" example:"0"`
	// CommissionKztCents разовая комиссия в тиынах.
	CommissionKztCents int64 `json:"commissionKztCents" validate:"gte=0" example:"0"`
}

// Validate проверяет форму расчёта.
func (f *LeasingCalculate) Validate() error {
	return validatorDefault.Validate(f)
}

// TCOCalculate форма расчёта стоимости владения.
//
// Здесь только то, чего сервер знать не может: сколько лет клиент собирается
// владеть машиной, сколько нарабатывать, почём заправляться и сколько платить
// оператору. Расход топлива, интервал и стоимость ТО, остаточная стоимость
// берутся по карточке и разделу — их клиент не задаёт и подменить не может.
type TCOCalculate struct {
	// Years срок владения в годах.
	Years int `json:"years" validate:"required,gt=0,lte=30" example:"5"`
	// HoursPerYear наработка в моточасах за год.
	HoursPerYear int `json:"hoursPerYear" validate:"required,gt=0,lte=8760" example:"1500"`
	// FuelPriceKztCents цена литра топлива в тиынах.
	FuelPriceKztCents int64 `json:"fuelPriceKztCents" validate:"gte=0" example:"29000"`
	// OperatorHourlyKztCents оплата оператора за моточас в тиынах.
	OperatorHourlyKztCents int64 `json:"operatorHourlyKztCents" validate:"gte=0" example:"250000"`
}

// Validate проверяет форму расчёта стоимости владения.
func (f *TCOCalculate) Validate() error {
	return validatorDefault.Validate(f)
}

// QuoteRecalculate форма пересчёта предложения на других условиях.
//
// Машины, цены и курса здесь нет намеренно: они приходят из самого
// предложения. Предложение — зафиксированное обязательство, и подменять
// его цифры клиентским вводом нельзя.
type QuoteRecalculate struct {
	// ScheduleType тип графика: annuity или differentiated.
	ScheduleType string `json:"scheduleType" validate:"required,oneof=annuity differentiated" example:"annuity"`
	// DownPaymentBps первоначальный взнос в базисных пунктах.
	DownPaymentBps int `json:"downPaymentBps" validate:"gte=0,lte=10000" example:"2000"`
	// TermMonths срок в месяцах.
	TermMonths int `json:"termMonths" validate:"required,gt=0,lte=360" example:"48"`
	// RateBps годовая ставка в базисных пунктах.
	RateBps int `json:"rateBps" validate:"gte=0,lte=10000" example:"1800"`
}

// Validate проверяет форму пересчёта.
func (f *QuoteRecalculate) Validate() error {
	return validatorDefault.Validate(f)
}

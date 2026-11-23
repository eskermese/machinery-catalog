package presenter

import (
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// QuotePayment строка графика платежей в ответе.
type QuotePayment struct {
	Number         int   `json:"number" validate:"required" example:"1"`          // Номер платежа
	PrincipalCents int64 `json:"principal" validate:"required" example:"7871440"` // Часть основного долга
	InterestCents  int64 `json:"interest" validate:"required" example:"6450000"`  // Часть процентов
	AmountCents    int64 `json:"amount" validate:"required" example:"14321440"`   // Сумма платежа
	BalanceCents   int64 `json:"balance" validate:"required" example:"366057040"` // Остаток долга после платежа
}

// QuoteCalculation параметры и результат расчёта лизинга.
type QuoteCalculation struct {
	ScheduleType     string `json:"scheduleType" validate:"required" example:"annuity"` // Тип графика
	LeasingProgramID string `json:"leasingProgramID,omitempty" example:""`              // Программа лизинга
	// LeasingProgramName название программы, по которой считали.
	//
	// Одного идентификатора мало: публичный справочник отдаёт только
	// действующие программы, и у предложения, выпущенного по с тех пор
	// выключенной, название взять было бы негде.
	LeasingProgramName string `json:"leasingProgramName,omitempty" example:"Halyk Leasing"`
	DownPaymentBps     int    `json:"downPaymentBps" validate:"required" example:"2000"` // Первоначальный взнос в bps
	TermMonths         int    `json:"termMonths" validate:"required" example:"36"`       // Срок в месяцах
	RateBps            int    `json:"rateBps" validate:"required" example:"1800"`        // Ставка в bps

	AssetCostKztCents      int64 `json:"assetCostKztCents" validate:"required" example:"4675943000"`      // Стоимость машины в тиынах
	DownPaymentKztCents    int64 `json:"downPaymentKztCents" validate:"required" example:"935188600"`     // Первоначальный взнос в тиынах
	FinancedAmountKztCents int64 `json:"financedAmountKztCents" validate:"required" example:"3740754400"` // Сумма финансирования
	ResidualValueKztCents  int64 `json:"residualValueKztCents" validate:"required" example:"0"`           // Выкупная стоимость
	CommissionKztCents     int64 `json:"commissionKztCents" validate:"required" example:"0"`              // Комиссия за организацию

	MonthlyPaymentKztCents int64 `json:"monthlyPaymentKztCents" validate:"required" example:"143214400"` // Регулярный платёж
	LastPaymentKztCents    int64 `json:"lastPaymentKztCents" validate:"required" example:"143214500"`    // Последний платёж
	TotalPaidKztCents      int64 `json:"totalPaidKztCents" validate:"required" example:"5155718500"`     // Итого выплат
	OverpaymentKztCents    int64 `json:"overpaymentKztCents" validate:"required" example:"1414964100"`   // Переплата
	TotalInterestKztCents  int64 `json:"totalInterestKztCents" validate:"required" example:"1414964100"` // Проценты за срок

	// Schedule полный график платежей.
	//
	// Приходит там, где предложение читают по одному: на публичной странице
	// и в карточке предложения. В списке его нет — см. NewQuoteList.
	Schedule []QuotePayment `json:"schedule,omitempty"`
}

// QuoteFX зафиксированный курс предложения.
type QuoteFX struct {
	// Rate применённый курс — уже с наценкой. Строкой, чтобы не потерять
	// младшие разряды на float64.
	Rate string `json:"rate" validate:"required" example:"532.786800"`
	// RateDate дата курса источника.
	RateDate time.Time `json:"rateDate" validate:"required" example:"2026-08-27T00:00:00Z"`
	// MarkupBps наценка, зашитая в курс.
	MarkupBps int `json:"markupBps" validate:"required" example:"200"`
}

// Quote представление коммерческого предложения.
type Quote struct {
	ID string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор

	// Machine машина предложения целиком: с фотографиями, характеристиками
	// и ценой по зафиксированному курсу. Приходит там, где предложение читают
	// по одному, — на публичной странице и в карточке предложения.
	Machine *Machine `json:"machine,omitempty"`
	// MachineRef краткая ссылка на карточку: название и то, по чему перейти.
	//
	// Приходит в списке вместо Machine. Читать полную карточку на каждую
	// строку списка значило бы делать запрос на строку ради одного названия,
	// а отдавать пустой объект «машины» — врать про её отсутствие.
	MachineRef    *MachineRef    `json:"machineRef,omitempty"`
	Configuration map[string]any `json:"configuration"` // Выбранная комплектация

	PriceCnyCents int64   `json:"priceCnyCents" validate:"required" example:"8950000"` // Цена машины в фынях (1/100 юаня)
	FX            QuoteFX `json:"fx" validate:"required"`                              // Зафиксированный курс

	Calculation QuoteCalculation `json:"calculation" validate:"required"` // Параметры и результат расчёта

	ExpiresAt     time.Time  `json:"expiresAt" validate:"required" example:"2026-09-10T10:00:00Z"` // Срок годности
	IsExpired     bool       `json:"isExpired" validate:"required" example:"false"`                // Истекло ли предложение
	FirstOpenedAt *time.Time `json:"firstOpenedAt,omitempty"`                                      // Момент первого открытия
	CreatedAt     time.Time  `json:"createdAt" validate:"required" example:"2026-08-27T10:00:00Z"` // Момент выпуска
}

// QuoteIssued ответ на выпуск предложения.
//
// Содержит публичную ссылку: это единственный момент, когда токен уходит
// наружу — менеджеру, который его отправит клиенту.
type QuoteIssued struct {
	Quote Quote `json:"quote" validate:"required"` // Выпущенное предложение
	// PublicToken токен публичной ссылки.
	PublicToken string `json:"publicToken" validate:"required" example:"IEkYYm4kQ0lLZ1hnR2dnUlZUR0FBQUFBQUFBQUFBQQ"`
}

// NewQuote создаёт представление предложения.
//
// Публичного токена здесь нет намеренно: представление уходит и на публичную
// страницу, и в список менеджера, а токен — право доступа, и повторять его
// в теле ответа тому, кто по нему уже пришёл, незачем.
func NewQuote(quote *entity.Quote, currentTime time.Time) Quote {
	if quote == nil {
		return Quote{}
	}

	view := Quote{
		ID:            quote.ID,
		Configuration: quote.Configuration,
		PriceCnyCents: quote.PriceCNYCents,
		FX: QuoteFX{
			Rate:      quote.FXRate.StringFixed(entity.FXRateScale),
			RateDate:  quote.FXRateDate,
			MarkupBps: quote.FXMarkupBps,
		},
		Calculation:   newQuoteCalculation(quote),
		ExpiresAt:     quote.ExpiresAt,
		IsExpired:     quote.IsExpired(currentTime),
		FirstOpenedAt: quote.FirstOpenedAt,
		CreatedAt:     quote.CreatedAt,
	}

	if quote.Machine != nil {
		// Машина показывается по ЗАФИКСИРОВАННОМУ курсу предложения, а не
		// по действующему: иначе цена в шапке страницы разошлась бы с суммой
		// в расчёте под ней, хотя предложение выпущено на одном курсе.
		// Доллар в предложении не участвует вовсе: он — второе отображение
		// витрины, а выпущенное КП обязано остаться неизменным, и курс,
		// который никто не фиксировал, менял бы его задним числом.
		machine := NewMachine(quote.Machine, PriceRates{KZT: &entity.AppliedFXRate{
			Rate:      quote.FXRate,
			RateDate:  quote.FXRateDate,
			MarkupBps: quote.FXMarkupBps,
		}})
		view.Machine = &machine
	}

	view.MachineRef = quoteMachineRef(quote)

	return view
}

// quoteMachineRef собирает ссылку на карточку, если её дочитали вместе с КП.
//
// Слаг категории проверяется наравне с остальным: адрес карточки на витрине
// вложен в категорию, и половина ссылки хуже, чем её отсутствие, — клиент
// собрал бы из неё несуществующий адрес. Тот же разбор, что у заявки.
func quoteMachineRef(quote *entity.Quote) *MachineRef {
	if quote.MachineName == nil || quote.MachineSlug == nil ||
		quote.MachineCategorySlug == nil {
		return nil
	}

	return &MachineRef{
		ID:           quote.MachineID,
		Slug:         *quote.MachineSlug,
		CategorySlug: *quote.MachineCategorySlug,
		NameRU:       *quote.MachineName,
	}
}

// newQuoteCalculation собирает блок расчёта.
//
// Первоначальный взнос и проценты выводятся из хранимых величин, а не читаются
// из отдельных колонок: обе выводятся однозначно (взнос — как стоимость минус
// финансирование, проценты — как переплата минус комиссия), и хранить их
// значило бы завести второй источник правды.
func newQuoteCalculation(quote *entity.Quote) QuoteCalculation {
	calculation := QuoteCalculation{
		ScheduleType:   quote.ScheduleType.String(),
		DownPaymentBps: quote.DownPaymentBps,
		TermMonths:     quote.TermMonths,
		RateBps:        quote.RateBps,

		AssetCostKztCents:      quote.AssetCostKZTCents,
		DownPaymentKztCents:    quote.AssetCostKZTCents - quote.FinancedAmountKZTCents,
		FinancedAmountKztCents: quote.FinancedAmountKZTCents,
		ResidualValueKztCents:  quote.ResidualValueKZTCents,
		CommissionKztCents:     quote.CommissionKZTCents,

		MonthlyPaymentKztCents: quote.MonthlyPaymentKZTCents,
		LastPaymentKztCents:    quote.LastPaymentKZTCents,
		TotalPaidKztCents:      quote.TotalPaidKZTCents,
		OverpaymentKztCents:    quote.OverpaymentKZTCents,
		TotalInterestKztCents:  quote.OverpaymentKZTCents - quote.CommissionKZTCents,

		Schedule: make([]QuotePayment, 0, len(quote.PaymentSchedule)),
	}

	if quote.LeasingProgramID != nil {
		calculation.LeasingProgramID = *quote.LeasingProgramID
		calculation.LeasingProgramName = stringValue(quote.LeasingProgramName)
	}

	for _, payment := range quote.PaymentSchedule {
		calculation.Schedule = append(calculation.Schedule, QuotePayment{
			Number:         payment.Number,
			PrincipalCents: payment.PrincipalCents,
			InterestCents:  payment.InterestCents,
			AmountCents:    payment.AmountCents,
			BalanceCents:   payment.BalanceCents,
		})
	}

	return calculation
}

// NewQuoteIssued создаёт ответ на выпуск предложения.
func NewQuoteIssued(quote *entity.Quote, currentTime time.Time) QuoteIssued {
	if quote == nil {
		return QuoteIssued{}
	}

	return QuoteIssued{Quote: NewQuote(quote, currentTime), PublicToken: quote.PublicToken}
}

// QuoteList ответ со списком предложений.
type QuoteList struct {
	Items []Quote `json:"items" validate:"required"` // Предложения, свежие сверху
}

// NewQuoteList создаёт представление списка предложений.
//
// График платежей в списке не отдаётся. Он длиной в срок сделки — 36–48
// строк на предложение, — и на полусотне предложений это сотни килобайт
// ради таблицы из пяти колонок, ни одна из которых графика не показывает.
// Смотрят его, когда открывают предложение целиком, и там он есть.
func NewQuoteList(quotes entity.Quotes, currentTime time.Time) QuoteList {
	items := make([]Quote, 0, len(quotes))

	for _, quote := range quotes {
		if quote == nil {
			continue
		}

		item := NewQuote(quote, currentTime)
		item.Calculation.Schedule = nil

		items = append(items, item)
	}

	return QuoteList{Items: items}
}

// LeasingProgram представление программы лизинга.
type LeasingProgram struct {
	ID       string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	NameRU   string `json:"nameRU" validate:"required" example:"Halyk Leasing"`                    // Название на русском
	NameKK   string `json:"nameKK" validate:"required" example:"Halyk Leasing"`                    // Название на казахском; пусто, пока нет перевода
	NameEN   string `json:"nameEN" validate:"required" example:"Halyk Leasing"`                    // Название на английском; пусто, пока нет перевода
	LogoPath string `json:"logoPath" example:""`                                                   // Путь к логотипу внутри хранилища
	// LogoURL адрес логотипа относительно корня API; пусто — логотипа нет.
	LogoURL           string `json:"logoURL" example:"/media/programs/halyk.png"`
	MinDownPaymentBps int    `json:"minDownPaymentBps" validate:"required" example:"1500"` // Минимальный взнос в bps
	TermMonthsMin     int    `json:"termMonthsMin" validate:"required" example:"12"`       // Нижняя граница срока
	TermMonthsMax     int    `json:"termMonthsMax" validate:"required" example:"60"`       // Верхняя граница срока
	RateMinBps        int    `json:"rateMinBps" validate:"required" example:"1800"`        // Нижняя граница ставки
	RateMaxBps        int    `json:"rateMaxBps" validate:"required" example:"2400"`        // Верхняя граница ставки
	MaxAmountCnyCents *int64 `json:"maxAmountCnyCents,omitempty" example:"50000000"`       // Потолок суммы сделки
	Position          int    `json:"position" validate:"required" example:"1"`             // Порядок вывода
	// IsActive действует ли программа.
	//
	// В публичной выдаче всегда true — она отдаёт только действующие;
	// в админской показывает выключенные, которые иначе выглядели бы
	// потерянными.
	IsActive bool `json:"isActive" validate:"required" example:"true"`
}

// LeasingProgramList ответ со списком программ лизинга.
type LeasingProgramList struct {
	Items []LeasingProgram `json:"items" validate:"required"` // Действующие программы
}

// NewLeasingProgramList создаёт представление списка программ.
func NewLeasingProgramList(programs entity.LeasingPrograms) LeasingProgramList {
	items := make([]LeasingProgram, 0, len(programs))

	for _, program := range programs {
		if program == nil {
			continue
		}

		items = append(items, NewLeasingProgram(program))
	}

	return LeasingProgramList{Items: items}
}

// NewLeasingProgram создаёт представление программы лизинга.
//
// Логотип адресуется так же, как фотографии карточек: путь остаётся
// для редактора, адрес — для показа.
func NewLeasingProgram(program *entity.LeasingProgram) LeasingProgram {
	if program == nil {
		return LeasingProgram{}
	}

	logoPath := stringValue(program.LogoPath)

	return LeasingProgram{
		ID:                program.ID,
		NameRU:            program.NameRU,
		NameKK:            program.NameKK,
		NameEN:            program.NameEN,
		LogoPath:          logoPath,
		LogoURL:           mediaURL(logoPath),
		MinDownPaymentBps: program.MinDownPaymentBps,
		TermMonthsMin:     program.TermMonthsMin,
		TermMonthsMax:     program.TermMonthsMax,
		RateMinBps:        program.RateMinBps,
		RateMaxBps:        program.RateMaxBps,
		MaxAmountCnyCents: program.MaxAmountCNYCents,
		Position:          program.Position,
		IsActive:          program.IsActive,
	}
}

// Lead представление заявки.
type Lead struct {
	ID        string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор
	Name      string `json:"name" validate:"required" example:"Иван"`                               // Имя клиента
	Phone     string `json:"phone" validate:"required" example:"+77001112233"`                      // Номер в нормализованном виде
	City      string `json:"city" example:"Алматы"`                                                 // Город
	MachineID string `json:"machineID,omitempty" example:""`                                        // Машина
	// Machine краткая ссылка на карточку: то, что нужно показать в списке
	// и по чему перейти. Приходит вместе с machineID, когда машина указана.
	//
	// Одного идентификатора мало: чтобы показать название, клиенту пришлось
	// бы тянуть карточку на каждую строку списка или весь каталог целиком.
	Machine *MachineRef `json:"machine,omitempty"`
	QuoteID string      `json:"quoteID,omitempty" example:""`                      // Коммерческое предложение
	Source  string      `json:"source" validate:"required" example:"machine_card"` // Источник заявки
	Status  string      `json:"status" validate:"required" example:"new"`          // Стадия обработки
	// AllowedTransitions статусы, в которые можно перевести заявку отсюда.
	//
	// Приходит вместе с самой заявкой, а не отдельной справочной ручкой:
	// перечень зависит от текущего статуса, и справочник заставил бы клиента
	// держать у себя таблицу переходов и применять её самому — то есть
	// хранить копию правила, расхождение с которой обнаружилось бы только
	// отказом бэкенда.
	//
	// Пустой список означает, что заявка завершена: won и lost терминальны.
	AllowedTransitions []string `json:"allowedTransitions" validate:"required" example:"in_progress"`
	// Message то, что написал КЛИЕНТ в форме заявки.
	//
	// Отдельно от comment намеренно: тот принадлежит менеджеру. В админке
	// их нельзя показывать одним полем — правка менеджера затёрла бы
	// единственную запись того, зачем человек обратился.
	Message   string `json:"message,omitempty" example:"Нужен масляный фильтр на XE215C"`
	Comment   string `json:"comment" example:""`             // Комментарий менеджера
	ManagerID string `json:"managerID,omitempty" example:""` // Ответственный менеджер
	// Manager краткая ссылка на учётную запись ответственного.
	//
	// Одного идентификатора мало: список сотрудников доступен только
	// администратору, и менеджер, глядя на свою заявку, не смог бы
	// разрешить его вовсе.
	Manager   *StaffRef `json:"manager,omitempty"`
	CreatedAt time.Time `json:"createdAt" validate:"required" example:"2026-08-27T10:00:00Z"` // Момент поступления
	UpdatedAt time.Time `json:"updatedAt" validate:"required" example:"2026-08-27T10:00:00Z"` // Момент изменения
}

// LeadAccepted ответ на приём заявки с сайта.
//
// Отдаёт только идентификатор и статус: клиенту незачем видеть внутреннюю
// кухню обработки, а лишние поля в публичном ответе — лишняя поверхность.
type LeadAccepted struct {
	ID     string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор заявки
	Status string `json:"status" validate:"required" example:"new"`                              // Стадия обработки
}

// LeadList ответ со списком заявок.
type LeadList struct {
	Items []Lead `json:"items" validate:"required"` // Заявки, свежие сверху
	// NextCursor курсор следующей страницы; пусто — страница последняя.
	//
	// Непрозрачная строка: её содержимое — деталь реализации пагинации,
	// и клиент обязан возвращать её как есть, а не собирать сам.
	NextCursor string `json:"nextCursor" validate:"required" example:""`
	// HasMore есть ли ещё заявки за этой страницей.
	HasMore bool `json:"hasMore" validate:"required" example:"true"`
	// Total общее число найденных заявок.
	//
	// Приходит только если клиент попросил его флагом withTotal: подсчёт —
	// отдельный запрос по тому же предикату.
	Total *int `json:"total,omitempty" example:"137"`
}

// NewLead создаёт представление заявки.
func NewLead(lead *entity.Lead) Lead {
	if lead == nil {
		return Lead{}
	}

	return Lead{
		ID:                 lead.ID,
		Name:               lead.Name,
		Phone:              lead.Phone,
		City:               stringValue(lead.City),
		MachineID:          stringValue(lead.MachineID),
		QuoteID:            stringValue(lead.QuoteID),
		Source:             lead.Source,
		Status:             lead.Status.String(),
		AllowedTransitions: allowedTransitions(lead.Status),
		Message:            stringValue(lead.Message),
		Comment:            stringValue(lead.Comment),
		ManagerID:          stringValue(lead.ManagerID),
		CreatedAt:          lead.CreatedAt,
		UpdatedAt:          lead.UpdatedAt,
		Machine:            machineRef(lead),
		Manager:            staffRef(lead),
	}
}

// machineRef собирает ссылку на карточку, если её дочитали вместе с заявкой.
//
// Слаг категории проверяется наравне с остальными: ссылка без него неполна,
// а отдать половину ссылки хуже, чем не отдать её вовсе — клиент собрал бы
// из неё несуществующий адрес.
func machineRef(lead *entity.Lead) *MachineRef {
	if lead.MachineID == nil || lead.MachineName == nil ||
		lead.MachineSlug == nil || lead.MachineCategorySlug == nil {
		return nil
	}

	return &MachineRef{
		ID:           *lead.MachineID,
		Slug:         *lead.MachineSlug,
		CategorySlug: *lead.MachineCategorySlug,
		NameRU:       *lead.MachineName,
	}
}

// staffRef собирает ссылку на ответственного, если его дочитали вместе с заявкой.
func staffRef(lead *entity.Lead) *StaffRef {
	if lead.ManagerID == nil || lead.ManagerPhone == nil {
		return nil
	}

	ref := &StaffRef{ID: *lead.ManagerID, Phone: *lead.ManagerPhone}
	if lead.ManagerName != nil {
		ref.Name = *lead.ManagerName
	}

	return ref
}

// allowedTransitions переводит перечень доменных статусов в строки ответа.
func allowedTransitions(status entity.LeadStatus) []string {
	allowed := status.AllowedTransitions()
	items := make([]string, 0, len(allowed))

	for _, target := range allowed {
		items = append(items, target.String())
	}

	return items
}

// MachineRef краткая ссылка на карточку техники.
//
// Отдельный тип, а не полное представление машины: в списке заявок нужны
// название и переход, а тащить туда цену, характеристики и фотографии
// значило бы платить за них на каждой строке.
type MachineRef struct {
	ID   string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Slug string `json:"slug" validate:"required" example:"xcmg-xe215c"`
	// CategorySlug слаг категории карточки.
	//
	// Нужен для перехода: адрес карточки на витрине вложен в категорию
	// (/catalog/{категория}/{машина}), и одного слага машины для ссылки мало.
	CategorySlug string `json:"categorySlug" validate:"required" example:"excavators"`
	NameRU       string `json:"nameRU" validate:"required" example:"Экскаватор XCMG XE215C"`
}

// StaffRef краткая ссылка на учётную запись сотрудника.
//
// Телефон обязателен, имя — нет: телефон есть у каждой записи по построению,
// а имя сотрудник задаёт себе сам и может не задать. Клиент показывает имя,
// когда оно есть, и телефон, когда его нет: подставлять вместо имени заглушку
// значило бы выдумывать её за человека.
type StaffRef struct {
	ID    string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Phone string `json:"phone" validate:"required" example:"+77001112233"`
	Name  string `json:"name,omitempty" example:"Иван Петров"`
}

// NewLeadAccepted создаёт ответ на приём заявки.
func NewLeadAccepted(lead *entity.Lead) LeadAccepted {
	if lead == nil {
		return LeadAccepted{}
	}

	return LeadAccepted{ID: lead.ID, Status: lead.Status.String()}
}

// NewLeadList создаёт представление списка заявок.
func NewLeadList(page entity.LeadPage) LeadList {
	items := make([]Lead, 0, len(page.Leads))

	for _, lead := range page.Leads {
		if lead == nil {
			continue
		}

		items = append(items, NewLead(lead))
	}

	return LeadList{
		Items:      items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
		Total:      page.Total,
	}
}

// LeasingCalculationFX курс, по которому выполнен справочный расчёт.
//
// Отдельно от QuoteFX и без наценки: наценка уже внутри курса, а её размер —
// внутреннее устройство ценообразования, и посетителю витрины его знать
// незачем. В выпущенном предложении она есть — там его читает тот, кому
// предложение адресовано.
type LeasingCalculationFX struct {
	// Rate применённый курс — уже с наценкой. Строкой, чтобы не потерять
	// младшие разряды на float64.
	Rate string `json:"rate" validate:"required" example:"532.786800"`
	// RateDate дата курса источника.
	//
	// Приходит всегда: при отсутствии свежего курса расчёт выполнен
	// по последнему известному, и без даты это неразличимо.
	RateDate time.Time `json:"rateDate" validate:"required" example:"2026-08-27T00:00:00Z"`
}

// LeasingCalculation ответ публичного расчёта лизинга.
//
// Ничего не сохранено: ни идентификатора, ни ссылки здесь нет и быть не может.
// Расчёт справочный и меняется вместе с курсом — зафиксированную цифру даёт
// только выпущенное коммерческое предложение.
type LeasingCalculation struct {
	// Calculation параметры и результат расчёта — те же, что и в предложении.
	Calculation QuoteCalculation `json:"calculation" validate:"required"`
	// FX курс, по которому выполнен расчёт.
	FX LeasingCalculationFX `json:"fx" validate:"required"`
	// PriceCnyCents цена машины в фынях (1/100 юаня), взятая сервером по слагу.
	PriceCnyCents int64 `json:"priceCnyCents" validate:"required" example:"8950000"`
}

// NewLeasingCalculation создаёт ответ публичного расчёта.
//
// Принимает несохранённое предложение: расчёт — ровно то, что попало бы в КП,
// и собирать его вторым способом значило бы завести второе место, где живёт
// формула, — с гарантией, что рано или поздно они разойдутся.
func NewLeasingCalculation(quote *entity.Quote) LeasingCalculation {
	if quote == nil {
		return LeasingCalculation{}
	}

	return LeasingCalculation{
		Calculation:   newQuoteCalculation(quote),
		PriceCnyCents: quote.PriceCNYCents,
		FX: LeasingCalculationFX{
			Rate:     quote.FXRate.StringFixed(entity.FXRateScale),
			RateDate: quote.FXRateDate,
		},
	}
}

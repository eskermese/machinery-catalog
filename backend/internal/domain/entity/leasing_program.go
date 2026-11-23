package entity

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// LeasingProgram программа лизингового партнёра.
//
// Задаёт границы, внутри которых считается платёж. Сам расчёт про программы
// не знает: калькулятору приходят уже выбранные значения, а соответствие
// границам проверяет сервис — см. CheckBounds.
type LeasingProgram struct {
	ID                string    // Идентификатор
	NameRU            string    // Название на русском
	NameKK            string    // Название на казахском
	NameEN            string    // Название на английском
	LogoPath          *string   // Логотип партнёра
	MinDownPaymentBps int       // Минимальный первоначальный взнос в базисных пунктах
	TermMonthsMin     int       // Нижняя граница срока
	TermMonthsMax     int       // Верхняя граница срока
	RateMinBps        int       // Нижняя граница ставки в базисных пунктах
	RateMaxBps        int       // Верхняя граница ставки в базисных пунктах
	MaxAmountCNYCents *int64    // Потолок суммы сделки; nil — без ограничения
	IsActive          bool      // Действует ли программа
	Position          int       // Порядок вывода
	CreatedAt         time.Time // Дата создания
	UpdatedAt         time.Time // Дата изменения
}

// LeasingPrograms список программ.
type LeasingPrograms []*LeasingProgram

// LeasingBounds параметры сделки, проверяемые против границ программы.
type LeasingBounds struct {
	DownPaymentBps int   // Первоначальный взнос в базисных пунктах
	TermMonths     int   // Срок в месяцах
	RateBps        int   // Ставка в базисных пунктах
	AmountCNYCents int64 // Сумма сделки в фынях (1/100 юаня)
}

// Validate проверяет пригодность программы к сохранению.
//
// Диапазоны проверяются парой, а не по отдельности: программа с нижней
// границей выше верхней проходит поштучную проверку и оставляет калькулятору
// пустое множество допустимых значений — отказ на каждом расчёте вместо
// отказа при заведении.
func (p *LeasingProgram) Validate() error {
	if p == nil {
		return ErrNilPointer
	}

	// Обязателен только русский — см. Category.Validate.
	if p.NameRU == "" {
		return ErrLeasingProgramNameRequired
	}

	if p.Position < 0 {
		return ErrLeasingProgramPositionInvalid
	}

	return p.validateBounds()
}

// validateBounds проверяет согласованность границ программы между собой.
func (p *LeasingProgram) validateBounds() error {
	if p.MinDownPaymentBps < 0 || p.MinDownPaymentBps > bpsInWhole {
		return ErrQuoteDownPaymentInvalid
	}

	if p.TermMonthsMin <= 0 || p.TermMonthsMax < p.TermMonthsMin {
		return ErrLeasingTermOutOfRange
	}

	if p.RateMinBps < 0 || p.RateMaxBps < p.RateMinBps {
		return ErrLeasingRateOutOfRange
	}

	if p.MaxAmountCNYCents != nil && *p.MaxAmountCNYCents <= 0 {
		return ErrLeasingAmountAboveMaximum
	}

	return nil
}

// CheckBounds проверяет параметры сделки против границ программы.
//
// Живёт в домене, а не в сервисе: это правило самой программы, и сервис лишь
// решает, когда его применить. Каждое нарушение — своя ошибка: клиенту нужно
// понять, что именно поправить, а «параметры не подходят» этого не говорит.
func (p *LeasingProgram) CheckBounds(bounds LeasingBounds) error {
	if p == nil {
		return ErrLeasingProgramNotFound
	}

	// Снятая программа не считает новые сделки, но остаётся в выпущенных КП:
	// её удаление запрещено внешним ключом с политикой RESTRICT.
	if !p.IsActive {
		return ErrLeasingProgramInactive
	}

	if bounds.DownPaymentBps < p.MinDownPaymentBps {
		return ErrLeasingDownPaymentTooLow
	}

	if bounds.TermMonths < p.TermMonthsMin || bounds.TermMonths > p.TermMonthsMax {
		return ErrLeasingTermOutOfRange
	}

	if bounds.RateBps < p.RateMinBps || bounds.RateBps > p.RateMaxBps {
		return ErrLeasingRateOutOfRange
	}

	// Потолок сравнивается в валюте прайса: обе величины в фынях (1/100 юаня),
	// и приводить их курсом не требуется.
	if p.MaxAmountCNYCents != nil && bounds.AmountCNYCents > *p.MaxAmountCNYCents {
		return ErrLeasingAmountAboveMaximum
	}

	return nil
}

// OutboxMessage событие, ожидающее доставки.
//
// Читающее представление строки notification_outbox: сценарии пишут туда
// DomainEvent, отправщик разбирает уже сохранённые строки.
type OutboxMessage struct {
	ID          string          // Идентификатор записи
	EventType   DomainEventType // Тип события
	EntityType  AuditEntityType // Тип сущности
	EntityID    string          // Идентификатор сущности
	Recipient   EventRecipient  // Адресат
	Payload     map[string]any  // Поля для рендера сообщения
	CreatedAt   time.Time       // Момент наступления события
	ProcessedAt *time.Time      // Момент успешной доставки; nil — не доставлено
}

// OutboxMessages список событий очереди.
type OutboxMessages []*OutboxMessage

// Timestamp момент времени, передаваемый в порт хранилища.
//
// Собственный тип поверх time.Time: он делает подпись метода однозначной там,
// где иначе рядом стояли бы два time.Time с разным смыслом.
type Timestamp = time.Time

// LeadFilter условия отбора заявок.
type LeadFilter struct {
	// Status ограничивает выдачу одним статусом.
	Status *LeadStatus
	// ManagerID ограничивает выдачу заявками одного менеджера.
	ManagerID *string
	// Cursor позиция, с которой продолжается выдача; nil — первая страница.
	Cursor *LeadCursor
	// Limit размер страницы.
	Limit int
	// WithTotal просить ли общее число найденных заявок.
	//
	// По запросу клиента, а не всегда: подсчёт — второй запрос по тому же
	// предикату, и он обязан просмотреть все подходящие строки.
	WithTotal bool
}

// LeadCursor позиция, с которой продолжается список заявок.
//
// Keyset вместо OFFSET по той же причине, что и в каталоге: OFFSET заставляет
// базу прочитать и отбросить пропускаемые строки, а новая заявка, пришедшая
// во время листания, сдвигает окно и показывает одну и ту же строку дважды.
// У заявок это заметнее, чем у карточек: они и появляются чаще.
//
// Сортировки в курсоре нет: список заявок всегда идёт свежими сверху,
// выбора порядка у клиента нет, и хранить его было бы обещанием, которого
// контракт не даёт.
type LeadCursor struct {
	// CreatedAt момент поступления последней строки страницы.
	CreatedAt time.Time `json:"createdAt"`
	// ID последней строки страницы. Тай-брейкер: две заявки могут прийти
	// в одну микросекунду, и без него порядок на границе страниц неустойчив.
	ID string `json:"id"`
}

// Encode кодирует курсор в непрозрачную строку.
func (c LeadCursor) Encode() (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", ErrCatalogCursorInvalid
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecodeLeadCursor разбирает курсор списка заявок.
func DecodeLeadCursor(encoded string) (*LeadCursor, error) {
	if encoded == "" {
		return nil, nil //nolint:nilnil // пустой курсор — первая страница, а не отказ
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrCatalogCursorInvalid
	}

	var cursor LeadCursor
	if err = json.Unmarshal(payload, &cursor); err != nil {
		return nil, ErrCatalogCursorInvalid
	}

	// Курсор без идентификатора или без момента продолжил бы выдачу
	// с нулевого времени, то есть с начала списка — молча и на каждой странице.
	if cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return nil, ErrCatalogCursorInvalid
	}

	return &cursor, nil
}

// LeadPage страница списка заявок.
type LeadPage struct {
	// Leads заявки страницы.
	Leads Leads
	// NextCursor курсор следующей страницы; пуст, если страница последняя.
	NextCursor string
	// HasMore сообщает, что за этой страницей есть ещё заявки.
	HasMore bool
	// Total общее число найденных заявок; nil — не считали.
	Total *int
}

// NewLeadPage собирает страницу из выдачи, запрошенной с запасом в одну строку.
//
// Лишняя строка отвечает на вопрос «есть ли следующая страница» без второго
// запроса с count(*) — тем же приёмом, что и в каталоге.
func NewLeadPage(leads Leads, limit int) (LeadPage, error) {
	page := LeadPage{Leads: leads}

	if len(leads) > limit {
		page.Leads = leads[:limit]
		page.HasMore = true
	}

	if !page.HasMore || len(page.Leads) == 0 {
		return page, nil
	}

	last := page.Leads[len(page.Leads)-1]

	encoded, err := LeadCursor{CreatedAt: last.CreatedAt, ID: last.ID}.Encode()
	if err != nil {
		return LeadPage{}, err
	}

	page.NextCursor = encoded

	return page, nil
}

// Validate проверяет параметры отбора и подставляет умолчания.
func (f *LeadFilter) Validate() error {
	if f == nil {
		return ErrNilPointer
	}

	if f.Status != nil && !f.Status.IsValid() {
		return ErrLeadStatusInvalid
	}

	if f.Limit == 0 {
		f.Limit = DefaultLeadLimit
	}

	if f.Limit < 0 || f.Limit > MaxLeadLimit {
		return ErrCatalogLimitInvalid
	}

	return nil
}

// Границы размера страницы списка заявок.
const (
	// DefaultLeadLimit размер страницы по умолчанию.
	DefaultLeadLimit = 50
	// MaxLeadLimit верхняя граница размера страницы.
	MaxLeadLimit = 200
)

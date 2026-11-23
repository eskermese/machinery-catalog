package entity

import (
	"time"
	"unicode/utf8"
)

// LeadStatus стадия обработки заявки менеджером.
type LeadStatus string

// Статусы заявки. Значения совпадают с CHECK leads_status_allowed.
const (
	LeadStatusNew        LeadStatus = "new"         // Поступила, не разобрана
	LeadStatusInProgress LeadStatus = "in_progress" // Менеджер работает с клиентом
	LeadStatusWon        LeadStatus = "won"         // Сделка состоялась
	LeadStatusLost       LeadStatus = "lost"        // Клиент отказался
)

// String возвращает строковое представление статуса.
func (s LeadStatus) String() string { return string(s) }

// IsValid сообщает, что статус известен домену.
func (s LeadStatus) IsValid() bool {
	_, ok := leadStatusTransitions[s]

	return ok
}

// IsTerminal сообщает, что из статуса нет исходящих переходов.
func (s LeadStatus) IsTerminal() bool {
	return len(leadStatusTransitions[s]) == 0
}

// leadStatusTransitions таблица допустимых переходов статуса заявки.
//
// Таблица, а не набор условий в коде: она читается как одна картина,
// и добавление стадии видно целиком, а не разбросано по ветвям if.
// Присваиванием поля статус не меняется — только через TransitionTo,
// иначе первый же забытый вызов превратит перечень в украшение.
//
// Отсутствие перехода из выигранной и проигранной заявки намеренно: обе
// терминальны. Клиент, вернувшийся спустя месяцы, — это новая заявка,
// а не воскрешение старой; иначе история обращений схлопнется в одну строку
// и по ней нельзя будет посчитать ни повторные обращения, ни конверсию.
var leadStatusTransitions = map[LeadStatus][]LeadStatus{
	LeadStatusNew:        {LeadStatusInProgress, LeadStatusWon, LeadStatusLost},
	LeadStatusInProgress: {LeadStatusWon, LeadStatusLost},
	LeadStatusWon:        {},
	LeadStatusLost:       {},
}

// AllowedTransitions возвращает статусы, в которые можно перейти отсюда.
//
// Копия, а не сам срез таблицы: вызывающий код получил бы доступ к внутреннему
// состоянию перечня и мог бы его переписать — не злонамеренно, а обычным
// append в чужую память.
//
// Пустой результат означает терминальный статус и читается снаружи именно так:
// отдельного признака «дальше некуда» не нужно.
func (s LeadStatus) AllowedTransitions() []LeadStatus {
	allowed := leadStatusTransitions[s]

	return append(make([]LeadStatus, 0, len(allowed)), allowed...)
}

// CanTransitionTo сообщает, допустим ли переход в целевой статус.
func (s LeadStatus) CanTransitionTo(target LeadStatus) bool {
	for _, allowed := range leadStatusTransitions[s] {
		if allowed == target {
			return true
		}
	}

	return false
}

// MaxLeadMessageLength предел длины сообщения клиента в символах.
//
// Совпадает с CHECK leads_message_length. Две тысячи — это несколько абзацев:
// запрос запчастей со списком позиций укладывается с запасом, а вставленный
// в форму документ отсекается.
const MaxLeadMessageLength = 2000

// MaxLeadCommentLength предел длины комментария менеджера в символах.
//
// Тот же, что у сообщения клиента: это две заметки об одном разговоре,
// и разные пределы у них выглядели бы решением, которого никто не принимал.
const MaxLeadCommentLength = MaxLeadMessageLength

// Lead заявка с сайта.
type Lead struct {
	ID        string     // Идентификатор
	Name      string     // Имя клиента
	Phone     string     // Номер телефона в нормализованном виде
	City      *string    // Город
	MachineID *string    // Машина, из карточки которой пришла заявка
	QuoteID   *string    // Коммерческое предложение, по которому обратились
	Source    string     // Откуда пришла заявка
	Status    LeadStatus // Стадия обработки
	// Message то, что написал КЛИЕНТ в форме заявки.
	//
	// Отдельно от Comment намеренно: тот принадлежит менеджеру. Одно поле
	// на двоих означало бы, что правка менеджера затирает запрос клиента —
	// а это единственная запись того, зачем человек обратился.
	Message   *string
	Comment   *string   // Комментарий менеджера
	ManagerID *string   // Ответственный менеджер
	CreatedAt time.Time // Момент поступления
	UpdatedAt time.Time // Момент последнего изменения

	// ManagerPhone и ManagerName — телефон и имя ответственного, если запись
	// дочитана вместе с заявкой.
	//
	// Телефон есть всегда: по нему заводится учётная запись. Имя сотрудник
	// задаёт себе сам и может не задать вовсе — тогда его называет телефон.
	ManagerPhone *string
	ManagerName  *string
	// MachineName и MachineSlug — название и слаг машины, если их дочитали
	// вместе с заявкой. Название для колонки списка, слаг для перехода
	// на карточку.
	MachineName *string
	MachineSlug *string
	// MachineCategorySlug слаг категории машины.
	//
	// Одного слага машины для перехода мало: адрес карточки на витрине вложен
	// в категорию, и без её слага ссылку не собрать — клиенту пришлось бы
	// дочитывать категорию на каждую строку списка.
	MachineCategorySlug *string
}

// Leads список заявок.
type Leads []*Lead

// NewLead создаёт заявку, нормализуя номер телефона.
//
// Нормализация здесь, а не в форме: один и тот же человек приходит и заявкой,
// и учётной записью, и склеить его по номеру можно только при одинаковом
// написании — значит правило написания одно на весь домен.
func NewLead(name, phone, source string, currentTime time.Time) *Lead {
	return &Lead{
		Name:      name,
		Phone:     NormalizePhone(phone),
		Source:    source,
		Status:    LeadStatusNew,
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

// Validate проверяет пригодность заявки к сохранению.
func (l *Lead) Validate() error {
	if l == nil {
		return ErrNilPointer
	}

	if l.Name == "" {
		return ErrLeadNameRequired
	}

	if !IsValidPhone(l.Phone) {
		return ErrPhoneInvalid
	}

	if l.Source == "" {
		return ErrLeadSourceRequired
	}

	if err := l.validateTexts(); err != nil {
		return err
	}

	if !l.Status.IsValid() {
		return ErrLeadStatusInvalid
	}

	// Заявка «по этому КП» без самой машины — противоречие: предложение
	// всегда о машине. То же ограничение стоит CHECK-ом leads_quote_requires_machine.
	if l.QuoteID != nil && l.MachineID == nil {
		return ErrLeadQuoteRequiresMachine
	}

	return nil
}

// validateTexts проверяет длину свободных полей заявки.
//
// Их два, и они принадлежат разным людям: message написал клиент, comment —
// менеджер. Пределы одинаковые, потому что это две заметки об одном
// разговоре, но поля разные, и путать их нельзя.
func (l *Lead) validateTexts() error {
	if l.Message != nil && utf8.RuneCountInString(*l.Message) > MaxLeadMessageLength {
		return ErrLeadMessageTooLong
	}

	if l.Comment != nil && utf8.RuneCountInString(*l.Comment) > MaxLeadCommentLength {
		return ErrLeadCommentTooLong
	}

	return nil
}

// TransitionTo переводит заявку в целевой статус.
//
// Единственный способ сменить статус: присваивание поля обошло бы таблицу
// переходов, и «выигранная» заявка могла бы вернуться в работу молча.
func (l *Lead) TransitionTo(target LeadStatus, currentTime time.Time) error {
	if l == nil {
		return ErrNilPointer
	}

	if !target.IsValid() {
		return ErrLeadStatusInvalid
	}

	if l.Status == target {
		return ErrLeadStatusTransitionDenied
	}

	if !l.Status.CanTransitionTo(target) {
		return ErrLeadStatusTransitionDenied
	}

	l.Status = target
	l.UpdatedAt = currentTime

	return nil
}

// AssignManager назначает ответственного менеджера.
//
// Назначение не завязано на статус: заявку распределяют и до того, как
// с ней начали работать, и переназначают в любой момент — это не переход
// стадии, а вопрос загрузки людей.
func (l *Lead) AssignManager(managerID string, currentTime time.Time) error {
	if l == nil {
		return ErrNilPointer
	}

	if managerID == "" {
		return ErrLeadManagerRequired
	}

	l.ManagerID = Pointer(managerID)
	l.UpdatedAt = currentTime

	return nil
}

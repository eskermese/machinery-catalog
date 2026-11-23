package entity

import "time"

// DomainEventType тип доменного события. Часть контракта с будущим адаптером
// канала доставки.
//
// Перечень наполняется по мере появления сценариев, которым есть о чём
// уведомить: событие описывает случившееся, а не способ доставки.
type DomainEventType string

// Типы событий.
const (
	// EventLeadCreated заявка с сайта принята.
	EventLeadCreated DomainEventType = "lead.created"
)

// String возвращает строковое представление типа события.
func (t DomainEventType) String() string { return string(t) }

// EventRecipient адресат события: кому и в какой роли доставлять сообщение.
// Канал доставки домену неизвестен — его выберет адаптер по роли и идентификатору.
type EventRecipient struct {
	Role ActorType // Роль получателя
	ID   string    // Идентификатор получателя
}

// NewEventRecipient создает адресата события.
func NewEventRecipient(id string, role ActorType) EventRecipient {
	return EventRecipient{Role: role, ID: id}
}

// DomainEvent факт, о котором нужно уведомить участника сценария.
//
// Событие описывает случившееся, а не способ доставки: ни текста сообщения, ни
// канала здесь нет. Payload несет минимум полей для рендера сообщения на стороне
// адаптера, чтобы тому не приходилось ходить в базу за контекстом.
//
// EntityType переиспользует тип истории изменений: обе подсистемы говорят об
// одних и тех же сущностях домена, и второй параллельный перечень разошелся бы
// с первым при добавлении сущности.
type DomainEvent struct {
	Type       DomainEventType // Тип события
	EntityType AuditEntityType // Тип сущности, о которой событие
	EntityID   string          // Идентификатор сущности
	Recipient  EventRecipient  // Адресат события
	Payload    map[string]any  // Минимум полей для рендера сообщения
	CreatedAt  time.Time       // Момент наступления события
}

// DomainEvents список доменных событий.
type DomainEvents []*DomainEvent

// NewDomainEvent создает доменное событие.
// Пустой payload заменяется на пустую карту: адаптеру не нужно различать
// отсутствие полей и nil.
func NewDomainEvent(
	eventType DomainEventType,
	entityType AuditEntityType,
	entityID string,
	recipient EventRecipient,
	payload map[string]any,
	currentTime time.Time,
) *DomainEvent {
	if payload == nil {
		payload = map[string]any{}
	}

	return &DomainEvent{
		Type:       eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Recipient:  recipient,
		Payload:    payload,
		CreatedAt:  currentTime,
	}
}

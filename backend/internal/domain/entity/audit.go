package entity

import "time"

// AuditEntityType тип сущности, изменение которой фиксируется в истории.
//
// Тип объявлен заранее, потому что история изменений и доменные события уже
// оперируют им и не должны переезжать на другой тип вместе с первой сущностью.
type AuditEntityType string

// Типы сущностей аудита и событий.
const (
	// AuditEntityLead заявка с сайта.
	AuditEntityLead AuditEntityType = "lead"
	// AuditEntityQuote коммерческое предложение.
	AuditEntityQuote AuditEntityType = "quote"
	// AuditEntityMachine карточка техники.
	AuditEntityMachine AuditEntityType = "machine"
)

// String возвращает строковое представление типа сущности.
func (t AuditEntityType) String() string { return string(t) }

// AuditAction действие, зафиксированное в истории.
type AuditAction string

// Действия аудита. Перечень общий для любых сущностей: действия, специфичные
// для домена каталога, добавятся вместе с ним.
const (
	AuditActionCreated       AuditAction = "created"        // Сущность создана
	AuditActionUpdated       AuditAction = "updated"        // Сущность изменена
	AuditActionStatusChanged AuditAction = "status_changed" // Изменен статус
)

// String возвращает строковое представление действия.
func (a AuditAction) String() string { return string(a) }

// ActorType тип инициатора изменения.
type ActorType string

// Типы инициаторов.
const (
	ActorTypeManager ActorType = "manager" // Менеджер
	ActorTypeAdmin   ActorType = "admin"   // Администратор
	ActorTypeSystem  ActorType = "system"  // Системное действие
)

// String возвращает строковое представление типа инициатора.
func (a ActorType) String() string { return string(a) }

// AuditLog запись истории изменений ключевых сущностей.
// Состояния и метаданные хранятся как JSON, чтобы история не зависела от схемы сущности.
type AuditLog struct {
	ID            string          `json:"id" db:"id"`                        // Идентификатор записи
	EntityType    AuditEntityType `json:"entityType" db:"entity_type"`       // Тип сущности
	EntityID      string          `json:"entityID" db:"entity_id"`           // Идентификатор сущности
	Action        AuditAction     `json:"action" db:"action"`                // Действие
	ActorID       *string         `json:"actorID" db:"actor_id"`             // Идентификатор инициатора
	ActorType     ActorType       `json:"actorType" db:"actor_type"`         // Тип инициатора
	PreviousState map[string]any  `json:"previousState" db:"previous_state"` // Состояние до изменения
	NewState      map[string]any  `json:"newState" db:"new_state"`           // Состояние после изменения
	Metadata      map[string]any  `json:"metadata" db:"metadata"`            // Дополнительный контекст
	CreatedAt     time.Time       `json:"createdAt" db:"created_at"`         // Дата записи
}

// AuditLogs список записей истории.
type AuditLogs []*AuditLog

// NewAuditLog создает запись истории.
func NewAuditLog(entityType AuditEntityType, entityID string, action AuditAction, actor Actor, currentTime time.Time) *AuditLog {
	return &AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ID,
		ActorType:  actor.Type,
		Metadata:   map[string]any{},
		CreatedAt:  currentTime,
	}
}

// WithStates добавляет к записи состояния до и после изменения.
func (l *AuditLog) WithStates(previous, current map[string]any) *AuditLog {
	if l == nil {
		return nil
	}

	l.PreviousState = previous
	l.NewState = current

	return l
}

// WithMetadata добавляет к записи дополнительный контекст.
func (l *AuditLog) WithMetadata(metadata map[string]any) *AuditLog {
	if l == nil {
		return nil
	}

	if metadata != nil {
		l.Metadata = metadata
	}

	return l
}

// Actor инициатор изменения. Идентификатор опционален: системные действия его не имеют.
type Actor struct {
	ID   *string   // Идентификатор инициатора
	Type ActorType // Тип инициатора
}

// NewActor создает инициатора изменения.
func NewActor(id string, actorType ActorType) Actor {
	if id == "" {
		return Actor{Type: actorType}
	}

	return Actor{ID: Pointer(id), Type: actorType}
}

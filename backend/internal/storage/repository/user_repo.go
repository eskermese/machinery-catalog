package repository

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// UserRepository исходящий порт учётных записей.
type UserRepository interface {
	// FindOrCreate возвращает существующего пользователя по паре (phone, role)
	// или создаёт нового. Конкурентный первый вход одного телефона не должен
	// приводить к ошибке: реализация обязана разрешать гонку по уникальному индексу.
	FindOrCreate(ctx context.Context, user *entity.User) (*entity.User, error)
	// GetByPhoneRole возвращает пользователя по паре (phone, role).
	// Отсутствие записи — ErrUserNotFound.
	GetByPhoneRole(ctx context.Context, phone string, role entity.ActorType) (*entity.User, error)
	// GetByID возвращает пользователя по идентификатору.
	// Отсутствие записи — ErrUserNotFound.
	GetByID(ctx context.Context, id string) (*entity.User, error)
	// List возвращает учётные записи сотрудников в порядке (роль, телефон).
	// Роль в фильтре пуста — возвращаются все.
	List(ctx context.Context, role entity.ActorType) (entity.Users, error)
	// UpdateName задаёт имя для показа. nil очищает имя.
	// Отсутствие записи — ErrUserNotFound.
	UpdateName(ctx context.Context, id string, name *string) error
	// EnsureExists создаёт пользователя, если его ещё нет, и ничего не делает иначе.
	// Нужен для идемпотентной посадки администраторов при старте приложения.
	EnsureExists(ctx context.Context, user *entity.User) error
}

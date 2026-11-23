package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// userColumns список колонок пользователя в порядке выборки.
const userColumns = `id, phone, role, name, created_at`

// userRepository репозиторий учётных записей.
type userRepository struct {
	pool *pgxpool.Pool // Пул соединений
}

// NewUserRepository создаёт новый объект userRepository.
func NewUserRepository(pool *pgxpool.Pool) repository.UserRepository {
	return &userRepository{pool: pool}
}

// GetByPhoneRole возвращает пользователя по паре (phone, role).
func (r userRepository) GetByPhoneRole(ctx context.Context, phone string, role entity.ActorType) (*entity.User, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE phone = $1 AND role = $2", userColumns, userTable)

	return queryOneRow[entity.User](ctx, querierFromContext(ctx, r.pool), q, "пользователя",
		entity.ErrUserNotFound, phone, role)
}

// GetByID возвращает пользователя по идентификатору.
func (r userRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", userColumns, userTable)

	return queryOneRow[entity.User](ctx, querierFromContext(ctx, r.pool), q, "пользователя",
		entity.ErrUserNotFound, id)
}

// UpdateName задаёт имя для показа.
func (r userRepository) UpdateName(ctx context.Context, id string, name *string) error {
	q := fmt.Sprintf("UPDATE %s SET name = $1 WHERE id = $2", userTable)

	tag, err := querierFromContext(ctx, r.pool).Exec(ctx, q, name, id)
	if err != nil {
		return fmt.Errorf("обновление имени: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrUserNotFound
	}

	return nil
}

// List возвращает учётные записи сотрудников.
//
// Порядок (роль, телефон), а не по дате создания: список читает человек,
// и ему нужен предсказуемый порядок, а не хронология регистраций.
func (r userRepository) List(ctx context.Context, role entity.ActorType) (entity.Users, error) {
	builder := newConditionBuilder()

	if role != "" {
		builder.addf("role = %s", role.String())
	}

	q := fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY role, phone",
		userColumns, userTable, builder.where())

	rows, err := querierFromContext(ctx, r.pool).Query(ctx, q, builder.args...)
	if err != nil {
		return nil, fmt.Errorf("получение учётных записей: %w", err)
	}

	users, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[entity.User])
	if err != nil {
		return nil, fmt.Errorf("чтение учётных записей: %w", err)
	}

	return users, nil
}

// FindOrCreate возвращает существующего пользователя или создаёт нового.
//
// Гонка первого входа разрешается уникальным индексом, а не проверкой перед
// вставкой: два параллельных запроса с одним телефоном оба увидели бы «нет
// записи» и один из них получил бы 500. Поэтому при нарушении уникальности
// запись просто перечитывается — победитель гонки её уже создал.
func (r userRepository) FindOrCreate(ctx context.Context, user *entity.User) (*entity.User, error) {
	if user == nil {
		return nil, entity.ErrNilPointer
	}

	existing, err := r.GetByPhoneRole(ctx, user.Phone, user.Role)
	if err == nil {
		return existing, nil
	}

	if !errors.Is(err, entity.ErrUserNotFound) {
		return nil, err
	}

	created, err := r.create(ctx, user)
	if err == nil {
		return created, nil
	}

	if !errors.Is(err, entity.ErrUserAlreadyExists) {
		return nil, err
	}

	return r.GetByPhoneRole(ctx, user.Phone, user.Role)
}

// EnsureExists создаёт пользователя, если записи ещё нет.
// Повторный запуск ничего не меняет: посадка администраторов идемпотентна.
func (r userRepository) EnsureExists(ctx context.Context, user *entity.User) error {
	if user == nil {
		return entity.ErrNilPointer
	}

	if _, err := r.FindOrCreate(ctx, user); err != nil {
		return err
	}

	return nil
}

// create вставляет пользователя, приводя нарушение уникальности к доменной ошибке.
func (r userRepository) create(ctx context.Context, user *entity.User) (*entity.User, error) {
	q := fmt.Sprintf(`INSERT INTO %s (phone, role, created_at) VALUES ($1, $2, $3)
		RETURNING %s`, userTable, userColumns)

	created, err := queryOneRow[entity.User](ctx, querierFromContext(ctx, r.pool), q, "пользователя",
		entity.ErrUserNotFound, user.Phone, user.Role, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, entity.ErrUserAlreadyExists
		}

		return nil, fmt.Errorf("создание пользователя: %w", err)
	}

	return created, nil
}

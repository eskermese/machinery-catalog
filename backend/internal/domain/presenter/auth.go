// Package presenter переводит сущности домена в тела HTTP-ответов.
//
// Тег validate:"required" здесь не запускает валидацию: представления никто
// не проверяет, их собирает сам сервис. Тег читает swaggo и переносит поле
// в required спеки — без этого В СПЕКЕ все поля ответов необязательные,
// и генератор типов на стороне клиента объявляет каждое как id?, items?,
// token?, заставляя проверять на undefined то, что приходит всегда.
//
// Тег ставится по правилу: ключ всегда присутствует в ответе И его значение
// никогда не означает «значения нет». Поле, которое сервис опускает
// (omitempty) или которым он сообщает отсутствие величины — пустая строка
// в parentID, unit, description, nextCursor, — остаётся необязательным.
// Обещать потребителю осмысленную величину там, где приходит признак
// отсутствия, хуже, чем лишняя проверка.
package presenter

import (
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// AuthUser представление учётной записи в ответе на вход.
// Идентификатор отдаётся намеренно: он лежит в токене как sub, и клиенту
// иначе неоткуда узнать, от чьего имени он теперь действует.
type AuthUser struct {
	ID    string `json:"id" validate:"required" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"` // Идентификатор пользователя
	Phone string `json:"phone" validate:"required" example:"+77001234567"`                      // Номер телефона
	// Name имя для показа; отсутствует, пока сотрудник его не задал.
	//
	// Необязательное намеренно: подставлять вместо незаполненного имени
	// заглушку значило бы выдумывать её за человека, а клиенту в этом случае
	// есть что показать — телефон.
	Name string `json:"name,omitempty" example:"Иван Петров"`
	Role string `json:"role" validate:"required" example:"manager"`
}

// AuthToken ответ на выпуск токена.
type AuthToken struct {
	Token string   `json:"token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // Токен доступа
	User  AuthUser `json:"user" validate:"required"`                                                    // Учётная запись
}

// NewAuthToken создаёт ответ на выпуск токена.
func NewAuthToken(token string, user *entity.User) AuthToken {
	if user == nil {
		return AuthToken{Token: token}
	}

	return AuthToken{Token: token, User: NewAuthUser(user)}
}

// AuthUserList ответ со списком учётных записей сотрудников.
type AuthUserList struct {
	Items []AuthUser `json:"items" validate:"required"` // Сотрудники в порядке (роль, телефон)
}

// NewAuthUser создаёт представление учётной записи.
func NewAuthUser(user *entity.User) AuthUser {
	if user == nil {
		return AuthUser{}
	}

	view := AuthUser{ID: user.ID, Phone: user.Phone, Role: user.Role.String()}
	if user.Name != nil {
		view.Name = *user.Name
	}

	return view
}

// NewAuthUserList создаёт представление списка сотрудников.
func NewAuthUserList(users entity.Users) AuthUserList {
	items := make([]AuthUser, 0, len(users))

	for _, user := range users {
		if user == nil {
			continue
		}

		items = append(items, NewAuthUser(user))
	}

	return AuthUserList{Items: items}
}

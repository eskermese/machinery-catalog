package presenter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestNewAuthUser_NameIsOptional фиксирует, что имя приходит, только когда оно
// задано.
//
// Пустая строка вместо незаданного имени заставила бы клиента гадать, показать
// её или телефон, — а телефон есть всегда, и именно он служит запасным
// обозначением человека.
func TestNewAuthUser_NameIsOptional(t *testing.T) {
	t.Parallel()

	withName := NewAuthUser(&entity.User{
		ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager,
		Name: entity.Pointer("Иван Петров"),
	})
	assert.Equal(t, "Иван Петров", withName.Name)

	withoutName := NewAuthUser(&entity.User{
		ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager,
	})
	assert.Empty(t, withoutName.Name)

	// Идентификатор, телефон и роль есть всегда: по ним клиент понимает,
	// от чьего имени действует.
	assert.Equal(t, "user-1", withoutName.ID)
	assert.Equal(t, "+77001112233", withoutName.Phone)
	assert.Equal(t, "manager", withoutName.Role)
}

// TestNewAuthUser_Nil фиксирует, что пустая запись не роняет представление.
func TestNewAuthUser_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AuthUser{}, NewAuthUser(nil))
}

// TestNewAuthToken_CarriesUser фиксирует, что ответ на вход несёт ту же
// учётную запись, что и GET /me.
//
// Собирается одним и тем же NewAuthUser: два места сборки разошлись бы,
// и имя появлялось бы то в одном ответе, то в другом.
func TestNewAuthToken_CarriesUser(t *testing.T) {
	t.Parallel()

	view := NewAuthToken("token-1", &entity.User{
		ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeAdmin,
		Name: entity.Pointer("Пётр"),
	})

	assert.Equal(t, "token-1", view.Token)
	assert.Equal(t, "Пётр", view.User.Name)
	assert.Equal(t, "admin", view.User.Role)

	// Без учётной записи остаётся один токен, а не запись с пустыми полями.
	assert.Equal(t, AuthToken{Token: "token-1"}, NewAuthToken("token-1", nil))
}

// TestNewAuthUserList_SkipsNil фиксирует, что пустая строка набора пропускается.
func TestNewAuthUserList_SkipsNil(t *testing.T) {
	t.Parallel()

	list := NewAuthUserList(entity.Users{
		{ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager},
		nil,
	})

	require.Len(t, list.Items, 1)
	assert.Equal(t, "user-1", list.Items[0].ID)
}

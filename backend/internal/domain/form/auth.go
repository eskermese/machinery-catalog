package form

import (
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// AuthTokenIssue форма выпуска токена по номеру телефона.
//
// Роль в теле запроса не принимается: её задаёт маршрут. Иначе пассажир мог бы
// попросить токен администратора, и вся проверка свелась бы к доверию клиенту.
type AuthTokenIssue struct {
	Phone string           `json:"phone" validate:"required" example:"+77001234567"` // Номер телефона
	Role  entity.ActorType `json:"-"`                                                // Роль. Определяется маршрутом
}

// Validate нормализует номер и проверяет форму.
// Нормализация идёт до проверки: «+7 (700) 000-00-00» и «77000000000» — один
// человек, иначе повторный вход создал бы вторую учётную запись.
func (f *AuthTokenIssue) Validate() error {
	if err := validatorDefault.Validate(f); err != nil {
		return err
	}

	f.Phone = entity.NormalizePhone(f.Phone)

	if !entity.IsValidPhone(f.Phone) {
		return entity.ErrPhoneInvalid
	}

	if !entity.IsLoginRole(f.Role) {
		return entity.ErrRoleForbidden
	}

	return nil
}

// AuthUserRename форма смены собственного имени.
//
// Только имя: телефон и роль опознают учётную запись, и менять их запросом
// от её же имени означало бы позволить пользователю переехать в чужую запись
// или выдать себе другую роль.
type AuthUserRename struct {
	// Name имя для показа; пустая строка очищает имя.
	Name string `json:"name" example:"Иван Петров"`
}

// Validate проверяет форму.
//
// Границы длины и обрезка пробелов живут в домене (entity.User.SetName):
// форма отвечает за разбор входа, а что считается допустимым именем —
// правило домена, и держать его в двух местах значило бы разойтись.
func (f *AuthUserRename) Validate() error {
	return validatorDefault.Validate(f)
}

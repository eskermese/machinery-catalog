package entity

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Границы длины номера в ЦИФРАХ, без ведущего плюса.
//
// Нижняя граница мягкая: строгий E.164 на этом этапе не нужен — номера приходят
// из разных стран и форматов, а подтверждения владения номером всё равно нет.
// Верхняя жёсткая и равна пределу E.164: тот же предел стоит CHECK-ом в users
// и leads, и без него номер длиннее пятнадцати цифр проходил бы валидацию формы
// и падал уже на вставке — клиент получал бы 500 вместо внятного отказа.
const (
	minPhoneDigits = 10
	maxPhoneDigits = 15
)

// User учётная запись пользователя сервиса.
//
// Пара (phone, role) уникальна: один человек может входить в разных ролях,
// это разные записи с разными идентификаторами.
type User struct {
	ID    string    `json:"id" db:"id"`       // Идентификатор пользователя
	Phone string    `json:"phone" db:"phone"` // Номер телефона в нормализованном виде
	Role  ActorType `json:"role" db:"role"`   // Роль пользователя
	// Name имя для показа; nil — не задано, тогда человека называет телефон.
	//
	// Не участвует ни в опознании, ни в правах: учётную запись определяет
	// пара (phone, role), а имя нужно людям, чтобы называть друг друга
	// в списке заявок.
	Name      *string   `json:"name" db:"name"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"` // Дата создания
}

// maxUserNameLength предел длины имени. Тот же, что в CHECK-е таблицы:
// иначе слишком длинное имя проходило бы проверку и падало на вставке,
// а клиент получал бы 500 вместо внятного отказа.
const maxUserNameLength = 100

// SetName задаёт имя для показа.
//
// Пустая строка очищает имя, а не сохраняется как пустое значение:
// «имени нет» — одно состояние, и хранить его двумя способами значило бы
// заставить каждого читателя проверять оба.
func (u *User) SetName(raw string) error {
	if u == nil {
		return ErrNilPointer
	}

	name := strings.TrimSpace(raw)

	if name == "" {
		u.Name = nil

		return nil
	}

	if utf8.RuneCountInString(name) > maxUserNameLength {
		return ErrUserNameInvalid
	}

	u.Name = &name

	return nil
}

// Users список учётных записей.
type Users []*User

// NewUser создаёт учётную запись пользователя.
func NewUser(phone string, role ActorType, currentTime time.Time) *User {
	return &User{
		Phone:     phone,
		Role:      role,
		CreatedAt: currentTime,
	}
}

// Actor возвращает действующего пользователя для контекста запроса и истории изменений.
func (u *User) Actor() Actor {
	if u == nil {
		return Actor{}
	}

	return NewActor(u.ID, u.Role)
}

// localPhoneLength длина номера КЗ и РФ в цифрах: 8 7XX XXX XX XX.
const localPhoneLength = 11

// NormalizePhone приводит номер к цифрам с ведущим плюсом.
//
// Нормализация нужна, чтобы «+7 (700) 000-00-00» и «77000000000» считались одним
// человеком: иначе повторный вход создаст вторую учётную запись, и пользователь
// потеряет всё, что было привязано к первой.
//
// Одиннадцатизначный номер, начинающийся с восьмёрки, приводится к семёрке:
// в КЗ и РФ «8 700 111 22 33» и «+7 700 111 22 33» — один и тот же абонент, и
// без этого он получил бы две учётные записи в зависимости от того, как набрал.
// Номера других стран не трогаются: ведущая восьмёрка там значащая.
func NormalizePhone(phone string) string {
	digits := strings.Builder{}

	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}

	if digits.Len() == 0 {
		return ""
	}

	normalized := digits.String()
	if len(normalized) == localPhoneLength && normalized[0] == '8' {
		normalized = "7" + normalized[1:]
	}

	return "+" + normalized
}

// IsValidPhone проверяет, что число цифр в номере укладывается в допустимые границы.
//
// Считаются именно цифры: прежняя реализация сравнивала длину строки вместе
// с ведущим плюсом, из-за чего границу приходилось держать в уме смещённой
// на единицу — источник ошибки при любой правке.
func IsValidPhone(phone string) bool {
	digits := phoneDigitCount(NormalizePhone(phone))

	return digits >= minPhoneDigits && digits <= maxPhoneDigits
}

// phoneDigitCount возвращает число цифр в нормализованном номере.
func phoneDigitCount(normalized string) int {
	return len(strings.TrimPrefix(normalized, "+"))
}

// IsLoginRole сообщает, что под этой ролью можно выпустить токен.
// Системная роль исключена: она принадлежит самому сервису, а не человеку.
func IsLoginRole(role ActorType) bool {
	return role == ActorTypeManager || role == ActorTypeAdmin
}

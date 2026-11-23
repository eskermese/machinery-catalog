package entity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalizePhone фиксирует приведение номера к единому виду.
//
// Смысл проверки: любой способ записи одного и того же номера обязан давать одну
// строку, иначе повторный вход создаст вторую учётную запись и пассажир потеряет
// свои брони. Отдельно зафиксирована граница — номера других стран не трогаются.
func TestNormalizePhone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		phone string
		exp   string
	}{
		{name: "местный с восьмёркой", phone: "87001112233", exp: "+77001112233"},
		{name: "местный с восьмёркой и разделителями", phone: "8 700 111 22 33", exp: "+77001112233"},
		{name: "международный формат", phone: "+7 (700) 111-22-33", exp: "+77001112233"},
		{name: "только цифры с семёркой", phone: "77001112233", exp: "+77001112233"},
		{name: "уже нормализованный", phone: "+77001112233", exp: "+77001112233"},
		{
			// Контрпример: ведущая восьмёрка значащая, номер другой страны.
			name: "номер другой страны не меняется", phone: "+861234567890", exp: "+861234567890",
		},
		{
			// Длина не одиннадцать — правило приведения не применяется.
			name: "короткий номер с восьмёркой", phone: "8700111", exp: "+8700111",
		},
		{name: "без цифр", phone: "не телефон", exp: ""},
		{name: "пустая строка", phone: "", exp: ""},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, s.exp, NormalizePhone(s.phone))
		})
	}
}

// TestNormalizePhone_SameNumberSameResult фиксирует главное следствие правила:
// разные способы записи одного номера дают одну учётную запись.
func TestNormalizePhone_SameNumberSameResult(t *testing.T) {
	t.Parallel()

	variants := []string{"87001112233", "8 700 111 22 33", "+7 (700) 111-22-33", "77001112233"}

	for _, variant := range variants {
		assert.Equal(t, NormalizePhone(variants[0]), NormalizePhone(variant),
			"запись %q должна давать тот же номер", variant)
	}
}

// TestIsValidPhone фиксирует пригодность номера и обе границы длины.
//
// Границы проверяются вместе с переходом через них: обе стоят CHECK-ом в users
// и leads, и расхождение на одну цифру означает 500 вместо отказа формы.
func TestIsValidPhone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		phone string
		exp   bool
	}{
		{name: "обычный номер", phone: "+77001112233", exp: true},
		{name: "номер с восьмёркой нормализуется", phone: "87001112233", exp: true},
		{name: "номер с разделителями", phone: "+7 (700) 111-22-33", exp: true},

		// Нижняя граница и переход через неё.
		{name: "девять цифр — на цифру меньше нижней границы", phone: "+123456789", exp: false},
		{name: "десять цифр — ровно нижняя граница", phone: "+1234567890", exp: true},
		{name: "одиннадцать цифр — внутри границ", phone: "+12345678901", exp: true},

		// Верхняя граница и переход через неё.
		{name: "четырнадцать цифр — внутри границ", phone: "+12345678901234", exp: true},
		{name: "пятнадцать цифр — ровно верхняя граница E.164", phone: "+123456789012345", exp: true},
		{name: "шестнадцать цифр — на цифру больше верхней границы", phone: "+1234567890123456", exp: false},
		{name: "двадцать цифр", phone: "+12345678901234567890", exp: false},

		{name: "слишком короткий", phone: "+7700", exp: false},
		{name: "не телефон", phone: "не телефон", exp: false},
		{name: "пустая строка", phone: "", exp: false},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.exp, IsValidPhone(c.phone))
		})
	}
}

// TestUser_SetName фиксирует правила имени: обрезка пробелов, очистка пустой
// строкой и предел длины.
func TestUser_SetName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expName  *string
		expError error
	}{
		{name: "обычное имя", input: "Иван Петров", expName: Pointer("Иван Петров")},
		{name: "пробелы по краям срезаются", input: "  Иван  ", expName: Pointer("Иван")},
		{name: "пустая строка очищает имя", input: ""},
		{name: "строка из пробелов тоже очищает", input: "   "},
		{
			name:     "слишком длинное отвергается",
			input:    strings.Repeat("я", maxUserNameLength+1),
			expError: ErrUserNameInvalid,
		},
		{
			// Предел считается в символах, а не в байтах: кириллица занимает
			// по два байта, и байтовый предел резал бы имя вдвое раньше.
			name:    "ровно предел в символах допустим",
			input:   strings.Repeat("я", maxUserNameLength),
			expName: Pointer(strings.Repeat("я", maxUserNameLength)),
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			user := &User{Phone: "+77001112233", Role: ActorTypeManager, Name: Pointer("прежнее")}

			err := user.SetName(c.input)

			if c.expError != nil {
				require.ErrorIs(t, err, c.expError)
				assert.Equal(t, Pointer("прежнее"), user.Name, "отказ не меняет запись")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, c.expName, user.Name)
		})
	}
}

// TestUser_SetName_NilReceiver фиксирует отказ на пустой записи.
func TestUser_SetName_NilReceiver(t *testing.T) {
	t.Parallel()

	var user *User

	require.ErrorIs(t, user.SetName("Иван"), ErrNilPointer)
}

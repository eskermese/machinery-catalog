package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// probeForm форма для проверки описаний: правила подобраны так, чтобы
// сработали разные ветви — с параметром и без.
type probeForm struct {
	Slug     string `json:"slug"     validate:"required"`
	Schedule string `json:"schedule" validate:"omitempty,oneof=annuity differentiated"`
	Term     int    `json:"term"     validate:"gte=1"`
	Rate     int    `json:"rate"     validate:"lte=10000"`
	Program  string `json:"program"  validate:"omitempty,uuid4"`
}

// TestValidator_DescriptionIsHumanReadable фиксирует, что вместе с кодом
// собирается описание для человека и в нём нет технического текста
// go-playground с путём к полю структуры.
func TestValidator_DescriptionIsHumanReadable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		form    probeForm
		expCode string
		expDesc string
	}{
		{
			name:    "обязательное поле",
			form:    probeForm{Term: 1},
			expCode: "SNT_SLUG_REQUIRED",
			expDesc: "поле slug: обязательно",
		},
		{
			name:    "значение вне перечня",
			form:    probeForm{Slug: "x", Term: 1, Schedule: "bullet"},
			expCode: "SNT_SCHEDULE_ONEOF",
			expDesc: "поле schedule: должно быть одним из: annuity differentiated",
		},
		{
			name:    "число ниже границы",
			form:    probeForm{Slug: "x", Term: 0},
			expCode: "SNT_TERM_GTE",
			expDesc: "поле term: должно быть больше или равно 1",
		},
		{
			name:    "число выше границы",
			form:    probeForm{Slug: "x", Term: 1, Rate: 10001},
			expCode: "SNT_RATE_LTE",
			expDesc: "поле rate: должно быть меньше или равно 10000",
		},
		{
			name:    "не uuid",
			form:    probeForm{Slug: "x", Term: 1, Program: "nope"},
			expCode: "SNT_PROGRAM_UUID4",
			expDesc: "поле program: должно быть UUID версии 4",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := New("SNT").Validate(&c.form)
			require.Error(t, err)

			var validationErr ValidationError
			require.ErrorAs(t, err, &validationErr)

			assert.Equal(t, c.expCode, validationErr.Code)
			assert.Equal(t, c.expDesc, validationErr.Description)
			assert.NotContains(t, validationErr.Description, "probeForm",
				"имя структуры наружу не уходит")
			assert.NotContains(t, validationErr.Description, "Error:Field")
		})
	}
}

// TestValidator_ValidFormHasNoError фиксирует, что корректная форма проходит.
func TestValidator_ValidFormHasNoError(t *testing.T) {
	t.Parallel()

	form := probeForm{Slug: "x", Term: 12, Rate: 1800, Schedule: "annuity"}

	require.NoError(t, New("SNT").Validate(&form))
}

// TestDescribeViolation_UnknownTag фиксирует запасное описание: правило,
// которому не сопоставлено сообщение, не оставляет клиента с пустой строкой.
func TestDescribeViolation_UnknownTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "поле slug: значение не прошло проверку",
		describeViolation("slug", "не-существующее-правило", ""))
}

// TestDefaultMessages_CoversDefaultCodes фиксирует, что описания не отстают
// от кодов: код сообщает машине, ЧТО нарушено, описание — человеку,
// и добавление правила без описания оставило бы клиента с общей фразой.
func TestDefaultMessages_CoversDefaultCodes(t *testing.T) {
	t.Parallel()

	messages := defaultMessages()

	for tag := range defaultCodes() {
		assert.Contains(t, messages, tag, "у правила %q нет описания", tag)
	}
}

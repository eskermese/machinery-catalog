package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLeadTime момент создания заявки в тестах.
var testLeadTime = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

// TestLeadStatus_Transitions — таблица переходов статуса заявки.
//
// Проверяется целиком, включая запрещённые направления: смысл таблицы в том,
// что она закрыта, и обратный ход по воронке исказил бы отчётность.
func TestLeadStatus_Transitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		from    LeadStatus
		to      LeadStatus
		allowed bool
	}{
		{from: LeadStatusNew, to: LeadStatusInProgress, allowed: true},
		{from: LeadStatusNew, to: LeadStatusWon, allowed: true},
		{from: LeadStatusNew, to: LeadStatusLost, allowed: true},
		{from: LeadStatusInProgress, to: LeadStatusWon, allowed: true},
		{from: LeadStatusInProgress, to: LeadStatusLost, allowed: true},

		// Назад по воронке нельзя.
		{from: LeadStatusInProgress, to: LeadStatusNew},
		// Терминальные не воскрешаются: вернувшийся клиент — новая заявка,
		// иначе история обращений схлопнется в одну строку.
		{from: LeadStatusWon, to: LeadStatusNew},
		{from: LeadStatusWon, to: LeadStatusInProgress},
		{from: LeadStatusWon, to: LeadStatusLost},
		{from: LeadStatusLost, to: LeadStatusNew},
		{from: LeadStatusLost, to: LeadStatusInProgress},
		{from: LeadStatusLost, to: LeadStatusWon},
		// Переход в себя — не переход.
		{from: LeadStatusNew, to: LeadStatusNew},
		{from: LeadStatusInProgress, to: LeadStatusInProgress},
	}

	for _, c := range cases {
		c := c

		t.Run(c.from.String()+"_to_"+c.to.String(), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.allowed, c.from.CanTransitionTo(c.to))
		})
	}
}

// TestLeadStatus_IsValid фиксирует закрытый перечень статусов.
func TestLeadStatus_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, LeadStatusNew.IsValid())
	assert.True(t, LeadStatusInProgress.IsValid())
	assert.True(t, LeadStatusWon.IsValid())
	assert.True(t, LeadStatusLost.IsValid())
	assert.False(t, LeadStatus("archived").IsValid())
	assert.False(t, LeadStatus("").IsValid())
	assert.Equal(t, "in_progress", LeadStatusInProgress.String())
}

// TestLeadStatus_IsTerminal фиксирует, из каких статусов выхода нет.
func TestLeadStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	assert.False(t, LeadStatusNew.IsTerminal())
	assert.False(t, LeadStatusInProgress.IsTerminal())
	assert.True(t, LeadStatusWon.IsTerminal())
	assert.True(t, LeadStatusLost.IsTerminal())
}

// TestNewLead_NormalizesPhone фиксирует нормализацию номера при создании.
func TestNewLead_NormalizesPhone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		phone    string
		expPhone string
	}{
		{name: "с разделителями", phone: "+7 (700) 111-22-33", expPhone: "+77001112233"},
		{name: "с ведущей восьмёркой", phone: "87001112233", expPhone: "+77001112233"},
		{name: "без плюса", phone: "77001112233", expPhone: "+77001112233"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			lead := NewLead("Иван", c.phone, "catalog", testQuoteTime)

			assert.Equal(t, c.expPhone, lead.Phone)
			assert.Equal(t, LeadStatusNew, lead.Status, "новая заявка начинается со статуса new")
			assert.Equal(t, testQuoteTime, lead.CreatedAt)
		})
	}
}

// TestLead_TransitionTo фиксирует, что статус меняется только через переход
// и что момент изменения обновляется.
func TestLead_TransitionTo(t *testing.T) {
	t.Parallel()

	lead := NewLead("Иван", "+77001112233", "catalog", testQuoteTime)
	later := testQuoteTime.Add(time.Hour)

	require.NoError(t, lead.TransitionTo(LeadStatusInProgress, later))
	assert.Equal(t, LeadStatusInProgress, lead.Status)
	assert.Equal(t, later, lead.UpdatedAt)

	// Обратный ход запрещён.
	require.ErrorIs(t, lead.TransitionTo(LeadStatusNew, later), ErrLeadStatusTransitionDenied)
	assert.Equal(t, LeadStatusInProgress, lead.Status, "статус не изменился после отказа")

	// Вперёд — можно.
	require.NoError(t, lead.TransitionTo(LeadStatusWon, later))
	assert.Equal(t, LeadStatusWon, lead.Status)

	// Из терминального выхода нет.
	require.ErrorIs(t, lead.TransitionTo(LeadStatusLost, later), ErrLeadStatusTransitionDenied)
}

// TestLead_TransitionTo_UnknownStatus фиксирует отказ на статусе вне перечня.
func TestLead_TransitionTo_UnknownStatus(t *testing.T) {
	t.Parallel()

	lead := NewLead("Иван", "+77001112233", "catalog", testQuoteTime)

	require.ErrorIs(t, lead.TransitionTo("archived", testQuoteTime), ErrLeadStatusInvalid)
}

// TestLead_TransitionTo_Nil фиксирует, что пустая заявка не паникует.
func TestLead_TransitionTo_Nil(t *testing.T) {
	t.Parallel()

	var lead *Lead

	require.ErrorIs(t, lead.TransitionTo(LeadStatusWon, testQuoteTime), ErrNilPointer)
	require.ErrorIs(t, lead.AssignManager("manager", testQuoteTime), ErrNilPointer)
}

// TestLead_AssignManager фиксирует назначение ответственного.
func TestLead_AssignManager(t *testing.T) {
	t.Parallel()

	lead := NewLead("Иван", "+77001112233", "catalog", testQuoteTime)
	later := testQuoteTime.Add(time.Hour)

	require.NoError(t, lead.AssignManager("manager-1", later))
	require.NotNil(t, lead.ManagerID)
	assert.Equal(t, "manager-1", *lead.ManagerID)
	assert.Equal(t, later, lead.UpdatedAt)
	assert.Equal(t, LeadStatusNew, lead.Status, "назначение не двигает стадию")

	// Переназначение возможно в любой момент.
	require.NoError(t, lead.AssignManager("manager-2", later))
	assert.Equal(t, "manager-2", *lead.ManagerID)

	require.ErrorIs(t, lead.AssignManager("", later), ErrLeadManagerRequired)
}

// TestLead_Validate фиксирует проверки заявки.
func TestLead_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Lead)
		expErr error
	}{
		{name: "корректная заявка", mutate: func(*Lead) {}},
		{name: "нет имени", mutate: func(l *Lead) { l.Name = "" }, expErr: ErrLeadNameRequired},
		{name: "короткий номер", mutate: func(l *Lead) { l.Phone = "+7700" }, expErr: ErrPhoneInvalid},
		{name: "нет источника", mutate: func(l *Lead) { l.Source = "" }, expErr: ErrLeadSourceRequired},
		{
			name:   "неизвестный статус",
			mutate: func(l *Lead) { l.Status = "archived" },
			expErr: ErrLeadStatusInvalid,
		},
		{
			// Предложение всегда о машине.
			name:   "предложение без машины",
			mutate: func(l *Lead) { l.QuoteID = Pointer("quote-1") },
			expErr: ErrLeadQuoteRequiresMachine,
		},
		{
			name: "предложение вместе с машиной",
			mutate: func(l *Lead) {
				l.QuoteID = Pointer("quote-1")
				l.MachineID = Pointer("machine-1")
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			lead := NewLead("Иван", "+77001112233", "catalog", testQuoteTime)
			c.mutate(lead)

			err := lead.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestLead_Validate_Nil фиксирует отказ на пустой заявке.
func TestLead_Validate_Nil(t *testing.T) {
	t.Parallel()

	var lead *Lead

	require.ErrorIs(t, lead.Validate(), ErrNilPointer)
}

// TestLeadStatus_AllowedTransitions фиксирует перечень, который уходит клиенту.
//
// Он обязан совпадать с таблицей переходов: расхождение означало бы, что
// клиент показывает кнопку, которую бэкенд отвергнет.
func TestLeadStatus_AllowedTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status LeadStatus
		exp    []LeadStatus
	}{
		{status: LeadStatusNew, exp: []LeadStatus{LeadStatusInProgress, LeadStatusWon, LeadStatusLost}},
		{status: LeadStatusInProgress, exp: []LeadStatus{LeadStatusWon, LeadStatusLost}},
		{status: LeadStatusWon, exp: []LeadStatus{}},
		{status: LeadStatusLost, exp: []LeadStatus{}},
		{status: "archived", exp: []LeadStatus{}},
	}

	for _, c := range cases {
		c := c

		t.Run(c.status.String(), func(t *testing.T) {
			t.Parallel()

			allowed := c.status.AllowedTransitions()

			assert.Equal(t, c.exp, allowed)

			// Перечень и таблица — один источник: каждый заявленный переход
			// действительно разрешён, и разрешённых сверх заявленных нет.
			for _, target := range allowed {
				assert.True(t, c.status.CanTransitionTo(target),
					"заявленный переход %s обязан быть разрешён", target)
			}

			for _, target := range []LeadStatus{
				LeadStatusNew, LeadStatusInProgress, LeadStatusWon, LeadStatusLost,
			} {
				if c.status.CanTransitionTo(target) {
					assert.Contains(t, allowed, target, "разрешённый переход обязан попасть в перечень")
				}
			}
		})
	}
}

// TestLeadStatus_AllowedTransitions_IsCopy фиксирует, что наружу уходит копия:
// иначе обычный append в чужую память переписал бы таблицу переходов.
func TestLeadStatus_AllowedTransitions_IsCopy(t *testing.T) {
	t.Parallel()

	allowed := LeadStatusNew.AllowedTransitions()
	require.Len(t, allowed, 3)

	allowed[0] = "испорчено"
	_ = append(allowed, "лишнее") //nolint:staticcheck // намеренная порча копии

	assert.Equal(t,
		[]LeadStatus{LeadStatusInProgress, LeadStatusWon, LeadStatusLost},
		LeadStatusNew.AllowedTransitions(),
		"таблица переходов не изменилась")
}

// TestLeadCursor_RoundTrip фиксирует, что курсор переживает кодирование
// и разбор без потерь.
func TestLeadCursor_RoundTrip(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)

	encoded, err := LeadCursor{CreatedAt: created, ID: "lead-1"}.Encode()
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	decoded, err := DecodeLeadCursor(encoded)
	require.NoError(t, err)
	require.NotNil(t, decoded)

	assert.Equal(t, "lead-1", decoded.ID)
	assert.True(t, created.Equal(decoded.CreatedAt))
}

// TestDecodeLeadCursor_Invalid фиксирует, что испорченный курсор отвергается,
// а не продолжает выдачу с начала списка молча.
func TestDecodeLeadCursor_Invalid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		encoded string
	}{
		{name: "не base64", encoded: "!!!не-курсор!!!"},
		{name: "base64, но не json", encoded: "aGVsbG8"},
		// {"id":"lead-1"} — без момента продолжения.
		{name: "без момента", encoded: "eyJpZCI6ImxlYWQtMSJ9"},
		// {"createdAt":"2026-08-28T12:00:00Z"} — без идентификатора.
		{name: "без идентификатора", encoded: "eyJjcmVhdGVkQXQiOiIyMDI2LTA4LTI4VDEyOjAwOjAwWiJ9"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, err := DecodeLeadCursor(c.encoded)

			require.ErrorIs(t, err, ErrCatalogCursorInvalid)
		})
	}
}

// TestDecodeLeadCursor_Empty фиксирует, что пустой курсор — первая страница,
// а не отказ.
func TestDecodeLeadCursor_Empty(t *testing.T) {
	t.Parallel()

	cursor, err := DecodeLeadCursor("")

	require.NoError(t, err)
	assert.Nil(t, cursor)
}

// TestNewLeadPage фиксирует сборку страницы заявок.
func TestNewLeadPage(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

	t.Run("строк меньше страницы", func(t *testing.T) {
		t.Parallel()

		page, err := NewLeadPage(Leads{{ID: "lead-1", CreatedAt: created}}, 20)

		require.NoError(t, err)
		assert.Len(t, page.Leads, 1)
		assert.False(t, page.HasMore)
		assert.Empty(t, page.NextCursor, "курсора у последней страницы нет")
	})

	t.Run("есть продолжение", func(t *testing.T) {
		t.Parallel()

		page, err := NewLeadPage(Leads{
			{ID: "lead-1", CreatedAt: created},
			{ID: "lead-2", CreatedAt: created.Add(-time.Minute)},
		}, 1)

		require.NoError(t, err)
		require.Len(t, page.Leads, 1, "лишняя строка в выдачу не попадает")
		assert.True(t, page.HasMore)

		cursor, err := DecodeLeadCursor(page.NextCursor)
		require.NoError(t, err)
		assert.Equal(t, "lead-1", cursor.ID, "курсор указывает на последнюю отданную строку")
	})

	t.Run("пустая выдача", func(t *testing.T) {
		t.Parallel()

		page, err := NewLeadPage(nil, 20)

		require.NoError(t, err)
		assert.Empty(t, page.Leads)
		assert.False(t, page.HasMore)
	})
}

// TestLead_Validate_Message фиксирует границу длины сообщения клиента.
//
// Та же граница стоит CHECK-ом leads_message_length: поле приходит
// с публичной формы, и проверять его только в домене мало.
func TestLead_Validate_Message(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		message *string
		expErr  error
	}{
		"без сообщения":     {message: nil},
		"обычное сообщение": {message: Pointer("Нужен фильтр на XE215C")},
		"ровно по границе":  {message: Pointer(strings.Repeat("я", MaxLeadMessageLength))},
		"длиннее границы":   {message: Pointer(strings.Repeat("я", MaxLeadMessageLength+1)), expErr: ErrLeadMessageTooLong},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			lead := NewLead("Иван", "+77001112233", "spare_parts", testLeadTime)
			lead.Message = c.message

			err := lead.Validate()

			if c.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestLead_MessageIsNotComment фиксирует, что сообщение клиента и комментарий
// менеджера — разные поля.
//
// Проверка выглядит тавтологией ровно до того дня, когда кто-нибудь решит
// «сэкономить» и свести их в одно: тогда правка менеджера затрёт запрос
// клиента, а восстановить его будет неоткуда.
func TestLead_MessageIsNotComment(t *testing.T) {
	t.Parallel()

	lead := NewLead("Иван", "+77001112233", "spare_parts", testLeadTime)
	lead.Message = Pointer("Нужен фильтр на XE215C")
	lead.Comment = Pointer("Перезвонить после обеда")

	require.NoError(t, lead.Validate())
	assert.Equal(t, "Нужен фильтр на XE215C", *lead.Message)
	assert.Equal(t, "Перезвонить после обеда", *lead.Comment)
}

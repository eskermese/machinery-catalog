package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestSearchQueryVariants разбирает, когда кириллицу стоит заменять латиницей.
//
// Правило одно: замена применяется, только если после неё не осталось
// кириллицы вовсе. Иначе «Каток» — слово из одних омоглифов — превратилось
// бы в «Katok», латинскую бессмыслицу, которая ничего не найдёт и только
// разбавит выдачу.
func TestSearchQueryVariants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			// Ради этого случая всё и делается: обозначения техники латинские,
			// а раскладка у человека русская.
			name:     "обозначение русскими буквами",
			query:    "ХЕ370",
			expected: []string{"ХЕ370", "XE370"},
		},
		{
			name:     "строчные тоже",
			query:    "хе370",
			expected: []string{"хе370", "xe370"},
		},
		{
			name:     "смешанное: русская Х, латинская E",
			query:    "Хe370",
			expected: []string{"Хe370", "Xe370"},
		},
		{
			// Слово целиком из омоглифов. Замена дала бы «katok» — ничего
			// не находящую латиницу, поэтому варианта быть не должно.
			name:     "русское слово из одних омоглифов",
			query:    "каток",
			expected: []string{"каток"},
		},
		{
			// Здесь «э», «в», «а», «и» не все омоглифы, и кириллица остаётся:
			// замена только испортила бы запрос.
			name:     "обычное русское слово",
			query:    "экскаватор",
			expected: []string{"экскаватор"},
		},
		{
			name:     "чистая латиница — заменять нечего",
			query:    "XE370G",
			expected: []string{"XE370G"},
		},
		{
			name:     "цифры",
			query:    "370",
			expected: []string{"370"},
		},
		{
			name:     "пробелы по краям обрезаются",
			query:    "  XE370G  ",
			expected: []string{"XE370G"},
		},
		{
			name:     "пусто",
			query:    "   ",
			expected: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.expected, entity.SearchQueryVariants(c.query))
		})
	}
}

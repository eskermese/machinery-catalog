package entity

import "strings"

// cyrillicHomoglyphs кириллические буквы, неотличимые на вид от латинских.
//
// Ровно те, что совпадают начертанием, и только заглавные с их строчными
// парами. Транслитерации здесь нет и не нужно: «Экскаватор» латиницей никто
// не набирает, а вот «ХЕ370» вместо «XE370» получается само собой —
// у обозначений техники латинские буквы, а раскладка остаётся русской.
var cyrillicHomoglyphs = map[rune]rune{
	'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H',
	'О': 'O', 'Р': 'P', 'С': 'C', 'Т': 'T', 'У': 'Y', 'Х': 'X',
	'а': 'a', 'в': 'b', 'е': 'e', 'к': 'k', 'м': 'm', 'н': 'h',
	'о': 'o', 'р': 'p', 'с': 'c', 'т': 't', 'у': 'y', 'х': 'x',
}

// SearchQueryVariants возвращает варианты запроса, которые стоит попробовать.
//
// Первым идёт сам запрос, вторым — он же с кириллическими омоглифами,
// заменёнными на латинские. Нужно это ровно для одного случая: обозначения
// техники латинские, а раскладка у человека русская, и «ХЕ370» набирается
// вместо «XE370» само собой.
//
// Опасность здесь очевидная и её надо обойти: «каток» состоит из омоглифов
// ЦЕЛИКОМ и превратился бы в «katok» — латинскую бессмыслицу, которая ничего
// не найдёт и только разбавит выдачу. Значит, «после замены не осталось
// кириллицы» — недостаточное условие.
//
// Отличает обозначение от слова не набор букв, а ЦИФРА или смешанная
// раскладка: «ХЕ370», «Хe370» — обозначения, «каток», «кот» — слова.
// Поэтому вариант появляется, когда выполнено всё сразу:
//
//   - замена что-то изменила;
//   - после неё не осталось кириллицы, которую заменить нечем;
//   - в запросе есть цифра или латинская буква.
func SearchQueryVariants(query string) []string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	converted, changed := latinizeHomoglyphs(trimmed)
	if !changed || hasCyrillic(converted) || !looksLikeDesignation(trimmed) {
		return []string{trimmed}
	}

	return []string{trimmed, converted}
}

// looksLikeDesignation сообщает, похож ли запрос на обозначение модели,
// а не на слово: в нём есть цифра или латинская буква.
func looksLikeDesignation(value string) bool {
	for _, symbol := range value {
		if symbol >= '0' && symbol <= '9' {
			return true
		}

		if symbol >= 'a' && symbol <= 'z' || symbol >= 'A' && symbol <= 'Z' {
			return true
		}
	}

	return false
}

// latinizeHomoglyphs заменяет кириллические омоглифы латинскими.
func latinizeHomoglyphs(query string) (result string, changed bool) {
	var builder strings.Builder

	builder.Grow(len(query))

	for _, symbol := range query {
		if latin, ok := cyrillicHomoglyphs[symbol]; ok {
			builder.WriteRune(latin)

			changed = true

			continue
		}

		builder.WriteRune(symbol)
	}

	return builder.String(), changed
}

// hasCyrillic сообщает, осталась ли в строке кириллица.
func hasCyrillic(value string) bool {
	for _, symbol := range value {
		if symbol >= 'А' && symbol <= 'я' || symbol == 'ё' || symbol == 'Ё' {
			return true
		}
	}

	return false
}

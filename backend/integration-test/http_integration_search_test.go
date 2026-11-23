//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchTotal ищет по витрине и возвращает число найденного и первое название.
func searchTotal(t require.TestingT, httpBasePath, query string) (int, string) {
	body := fmt.Sprintf(`{"query":%q,"limit":5,"withTotal":true}`, query)

	status, payload := requestJSON(http.MethodPost,
		httpBasePath+"/v1/catalog/machines/search", "", body)
	require.Equal(t, http.StatusOK, status, "поиск «%s»: %s", query, payload)

	page := decodeJSONMap(t, payload)

	total, _ := page["total"].(float64)

	items, _ := page["items"].([]any)
	if len(items) == 0 {
		return int(total), ""
	}

	first, _ := items[0].(map[string]any)
	name, _ := first["nameRU"].(string)

	return int(total), name
}

// TestHTTPSearchFullText проходит по тем самым запросам, на которых ILIKE
// промахивался, и проверяет, что теперь они находят.
//
// Каталог здесь свой, заведённый тестом: набор в общей базе не фиксирован,
// а проверять надо поведение поиска, а не содержимое стенда.
func (s *HTTPTestSuite) TestHTTPSearchFullText() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "search-"+suffix, "Катки")

	// Одно название на все проверки: «Каток XE370G» разом даёт и русское
	// слово со склонением, и латинское обозначение модели.
	slug := "search-katok-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 1_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// Название задаёт createMachine («Э»), поэтому переименуем карточку
	// в то, на чём проверяется поиск.
	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":"XE370G","slug":%q,
		"nameRU":"Пневмошинный каток XE370G","nameKK":"К","nameEN":"Roller",
		"priceCnyCents":1000000,"isPricePublic":true,
		"availability":"in_stock","condition":"new",
		"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`, categoryID, slug)

	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, machineID),
		admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "переименование: %s", payload)

	cases := []struct {
		query string
		note  string
	}{
		{"каток", "точное слово"},
		{"катки", "множественное число: снежковый стеммер его не берёт, берут триграммы"},
		{"катка", "родительный падеж"},
		{"каток пневмошинный", "обратный порядок слов"},
		{"пневмошинный каток", "прямой порядок"},
		{"XE370G", "обозначение модели целиком"},
		{"xe370", "префикс обозначения"},
		{"370", "часть обозначения: не префикс, находится подстрокой"},
		{"ХЕ370", "обозначение русскими буквами: раскладка"},
		{"катог", "опечатка"},
	}

	for _, c := range cases {
		total, name := searchTotal(s.T(), s.httpBasePath, c.query)

		assert.Positive(s.T(), total, "«%s» (%s) должен находить", c.query, c.note)
		assert.Contains(s.T(), name, "каток", "«%s» (%s): нашлась не та карточка", c.query, c.note)
	}
}

// TestHTTPSearchRanksExactModelFirst фиксирует ранжирование: точное
// совпадение модели встаёт первым, даже если карточка опубликована раньше
// других и по прежнему порядку была бы в конце.
func (s *HTTPTestSuite) TestHTTPSearchRanksExactModelFirst() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "rank-"+suffix, "Экскаваторы")

	// Точная модель заводится и публикуется ПЕРВОЙ: по порядку «свежие
	// сверху» она оказалась бы последней.
	target := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: "rank-target-" + suffix, PriceCents: 1_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, target)

	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":%q,"slug":%q,
		"nameRU":"Экскаватор точная модель","nameKK":"Э","nameEN":"E",
		"priceCnyCents":1000000,"isPricePublic":true,
		"availability":"in_stock","condition":"new",
		"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`,
		categoryID, "RANK"+suffix, "rank-target-"+suffix)

	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, target),
		admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "переименование: %s", payload)

	// Две карточки, у которых обозначение лишь НАЧИНАЕТСЯ с искомого,
	// и опубликованы они позже.
	for index := range 2 {
		other := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("rank-other-%d-%s", index, suffix),
			PriceCents: 1_000_000,
		})
		publishMachine(s.T(), s.httpBasePath, admin, other)

		body = fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":%q,"slug":%q,
			"nameRU":"Экскаватор похожая модель","nameKK":"Э","nameEN":"E",
			"priceCnyCents":1000000,"isPricePublic":true,
			"availability":"in_stock","condition":"new",
			"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`,
			categoryID, fmt.Sprintf("RANK%s-%d", suffix, index),
			fmt.Sprintf("rank-other-%d-%s", index, suffix))

		status, payload = requestJSON(http.MethodPut,
			fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, other),
			admin.Token, body)
		require.Equal(s.T(), http.StatusOK, status, "переименование: %s", payload)
	}

	total, first := searchTotal(s.T(), s.httpBasePath, "RANK"+suffix)

	assert.Equal(s.T(), 3, total, "префикс находит все три")
	assert.Equal(s.T(), "Экскаватор точная модель", first,
		"точное совпадение модели встаёт выше префиксных, несмотря на дату публикации")
}

// TestHTTPSearchPaginatesRanked фиксирует, что ранжированная выдача листается
// без потерь и повторов: курсор несёт релевантность, и продолжение идёт
// по тому же порядку.
func (s *HTTPTestSuite) TestHTTPSearchPaginatesRanked() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "page-"+suffix, "Погрузчики")

	const total = 7

	for index := range total {
		id := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
			CategoryID: categoryID,
			Slug:       fmt.Sprintf("page-%d-%s", index, suffix),
			PriceCents: int64(1_000_000 + index),
		})
		publishMachine(s.T(), s.httpBasePath, admin, id)

		body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":%q,"slug":%q,
			"nameRU":"Погрузчик страничный","nameKK":"П","nameEN":"L",
			"priceCnyCents":1000000,"isPricePublic":true,
			"availability":"in_stock","condition":"new",
			"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`,
			categoryID, fmt.Sprintf("PAGE%s-%d", suffix, index),
			fmt.Sprintf("page-%d-%s", index, suffix))

		status, payload := requestJSON(http.MethodPut,
			fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, id),
			admin.Token, body)
		require.Equal(s.T(), http.StatusOK, status, "переименование: %s", payload)
	}

	seen := make(map[string]bool, total)
	cursor := ""

	for page := 0; page < total; page++ {
		body := fmt.Sprintf(`{"query":%q,"limit":3,"cursor":%q}`, "PAGE"+suffix, cursor)

		status, payload := requestJSON(http.MethodPost,
			s.httpBasePath+"/v1/catalog/machines/search", "", body)
		require.Equal(s.T(), http.StatusOK, status, "страница %d: %s", page, payload)

		decoded := decodeJSONMap(s.T(), payload)

		items, _ := decoded["items"].([]any)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			id, _ := item["id"].(string)

			assert.False(s.T(), seen[id], "карточка %s показана дважды", id)
			seen[id] = true
		}

		cursor, _ = decoded["nextCursor"].(string)
		if cursor == "" {
			break
		}
	}

	assert.Len(s.T(), seen, total, "по страницам прошли все карточки, ни одна не потерялась")
}

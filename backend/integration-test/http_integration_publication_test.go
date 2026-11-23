//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPMachineEditKeepsPublication — главный сценарий правки: менеджер
// заходит поправить опечатку в опубликованной карточке, и карточка остаётся
// на витрине.
//
// Раньше это ломалось молча. Форма сохранения признака публикации не описывает,
// а UPDATE писал is_active вместе с остальными полями — то есть нулевым.
// Менеджер видел «сохранено», а карточка исчезала с сайта, и узнавал он
// об этом от клиента.
//
// Проверяется всё состояние публикации, а не только видимость: published_at
// задаёт порядок ленты, и сброс даты поднял бы карточку наверх как новинку.
func (s *HTTPTestSuite) TestHTTPMachineEditKeepsPublication() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "edit-keeps-"+suffix, "Экскаваторы")

	slug := "edit-keeps-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID,
		Slug:       slug,
		PriceCents: 8_950_000,
	})

	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	status, published := adminMachineBySlug(s.httpBasePath, admin, slug)
	require.Equal(s.T(), http.StatusOK, status)
	require.Equal(s.T(), true, published["isActive"], "карточка опубликована")

	publishedAt, ok := published["publishedAt"].(string)
	require.True(s.T(), ok, "дата первой публикации проставлена")

	// Правка, какую делает менеджер: поменялось одно название. Признака
	// публикации в теле нет — его в форме и не существует.
	fixed := "Экскаватор XCMG XE215C"
	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":"XE215C","slug":%q,
		"nameRU":%q,"nameKK":"Э","nameEN":"E","priceCnyCents":8950000,"isPricePublic":true,
		"availability":"in_stock","condition":"new","productionYear":2024,
		"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`,
		categoryID, slug, fixed)

	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, machineID),
		admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "правка карточки: %s", payload)

	// Правка доехала.
	status, edited := adminMachineBySlug(s.httpBasePath, admin, slug)
	require.Equal(s.T(), http.StatusOK, status)
	assert.Equal(s.T(), fixed, edited["nameRU"], "название исправлено")

	// И публикация осталась ровно такой, какой была.
	assert.Equal(s.T(), true, edited["isActive"], "правка не снимает карточку с публикации")
	assert.Equal(s.T(), publishedAt, edited["publishedAt"],
		"дата первой публикации не сдвигается: она задаёт порядок ленты")

	// Главная проверка — витрина: именно её терял менеджер.
	status, public := publicMachineBySlug(s.httpBasePath, slug)
	require.Equal(s.T(), http.StatusOK, status, "карточка осталась на витрине")
	assert.Equal(s.T(), fixed, public["nameRU"], "витрина показывает исправленное название")
}

// TestHTTPMachineEditKeepsDraft — та же правка с другой стороны: черновик
// после правки не публикуется сам собой.
//
// Симметрия важна. Если бы сохранение просто не трогало колонку, а форма
// когда-нибудь начала присылать true, черновик уехал бы на витрину
// незамеченным. Публикацию меняет только POST /publish.
func (s *HTTPTestSuite) TestHTTPMachineEditKeepsDraft() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "edit-draft-"+suffix, "Экскаваторы")

	slug := "edit-draft-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID,
		Slug:       slug,
		PriceCents: 8_950_000,
	})

	body := fmt.Sprintf(`{"categoryID":%q,"brand":"XCMG","model":"XE215C","slug":%q,
		"nameRU":"Черновик после правки","nameKK":"Э","nameEN":"E",
		"priceCnyCents":8950000,"isPricePublic":true,
		"availability":"in_stock","condition":"new","productionYear":2024,
		"images":[{"path":"a.jpg","position":0,"isPrimary":true}]}`, categoryID, slug)

	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, machineID),
		admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "правка черновика: %s", payload)

	status, edited := adminMachineBySlug(s.httpBasePath, admin, slug)
	require.Equal(s.T(), http.StatusOK, status)
	assert.Equal(s.T(), false, edited["isActive"], "правка не публикует черновик")

	status, _ = publicMachineBySlug(s.httpBasePath, slug)
	assert.Equal(s.T(), http.StatusNotFound, status, "черновик на витрине не появился")
}

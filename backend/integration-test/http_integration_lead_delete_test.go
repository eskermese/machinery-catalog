//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPLeadDelete удаляет заявку и проверяет, что выпущенное по ней
// предложение это переживает.
//
// Заявка — контакт живого человека, и удаление заведено ради единственного
// требования, которое статусом не исполняется: человек вправе потребовать
// убрать свои персональные данные. Обязательство перед клиентом при этом
// исчезать не должно, поэтому предложение остаётся и по ссылке открывается.
func (s *HTTPTestSuite) TestHTTPLeadDelete() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "lead-del-"+suffix, "Экскаваторы")

	slug := "lead-del-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 3_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	// Заявка с сайта.
	body := fmt.Sprintf(`{"name":%q,"phone":%q,"machineID":%q,"source":"machine_card"}`,
		"Клиент "+suffix, uniquePhone(), machineID)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status, "приём заявки: %s", payload)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	require.NotEmpty(s.T(), leadID)

	// Предложение по этой же машине.
	issue := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800,"validForDays":14}`, slug)

	status, payload = requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/quotes",
		admin.Token, issue)
	require.Equal(s.T(), http.StatusCreated, status, "выпуск предложения: %s", payload)

	token, _ := decodeJSONMap(s.T(), payload)["publicToken"].(string)
	require.NotEmpty(s.T(), token)

	leadURL := fmt.Sprintf("%s/v1/admin/leads/%s", s.httpBasePath, leadID)

	// Менеджеру удаление закрыто: оно необратимо и стирает контакт клиента.
	manager := managerSession(s.T(), s.httpBasePath)

	status, payload = requestJSON(http.MethodDelete, leadURL, manager.Token, "")
	require.Equal(s.T(), http.StatusForbidden, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"ROLE_FORBIDDEN")

	// Администратору — открыто.
	status, payload = requestJSON(http.MethodDelete, leadURL, admin.Token, "")
	require.Equal(s.T(), http.StatusNoContent, status, "удаление заявки: %s", payload)

	// Заявки больше нет, и следа от неё тоже.
	status, _ = requestJSON(http.MethodGet, leadURL, admin.Token, "")
	assert.Equal(s.T(), http.StatusNotFound, status, "заявка удалена без следа")

	// Повторное удаление — 404, а не молчаливый успех.
	status, _ = requestJSON(http.MethodDelete, leadURL, admin.Token, "")
	assert.Equal(s.T(), http.StatusNotFound, status)

	// А предложение живо и открывается по своей ссылке: обязательство перед
	// клиентом не исчезает оттого, что кто-то прибрал заявку.
	status, payload = requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/quotes/%s", s.httpBasePath, token), "", "")
	require.Equal(s.T(), http.StatusOK, status, "предложение пережило удаление заявки: %s", payload)
}

// TestHTTPQuotesAreNotDeletable фиксирует РЕШЕНИЕ, а не пробел: ручки удаления
// предложения нет.
//
// Предложение — зафиксированное обязательство, ссылку на которое клиент уже
// держит в руках. Удалить его значит превратить присланный ему документ
// в 404 — ровно то «у меня ничего не открывается», от которого страницу
// и защищали пометкой об истечении. Цикл у предложения уже есть: оно истекает
// само и после этого открывается с зафиксированными цифрами.
func (s *HTTPTestSuite) TestHTTPQuotesAreNotDeletable() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "quote-del-"+suffix, "Экскаваторы")

	slug := "quote-del-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 3_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	issue := fmt.Sprintf(`{"machineSlug":%q,"scheduleType":"annuity",
		"downPaymentBps":2000,"termMonths":36,"rateBps":1800,"validForDays":14}`, slug)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/admin/quotes",
		admin.Token, issue)
	require.Equal(s.T(), http.StatusCreated, status, "выпуск предложения: %s", payload)

	// Идентификатор лежит внутри quote: наружу ответ отдаёт предложение
	// вместе с токеном публичной ссылки.
	issued, _ := decodeJSONMap(s.T(), payload)["quote"].(map[string]any)
	require.NotNil(s.T(), issued)

	quoteID, _ := issued["id"].(string)
	require.NotEmpty(s.T(), quoteID)

	// Даже администратору. Удалять предложения нельзя никому.
	status, _ = requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/quotes/%s", s.httpBasePath, quoteID), admin.Token, "")

	assert.Equal(s.T(), http.StatusMethodNotAllowed, status,
		"ручки удаления предложения нет и быть не должно")
}

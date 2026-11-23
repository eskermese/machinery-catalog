//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// onePixelPNG минимальный настоящий PNG.
//
// Настоящий, а не произвольные байты: загрузка проверяет содержимое, и мусор
// она отвергнет. Один пиксель — ровно столько, сколько нужно, чтобы файл
// оказался на диске.
var onePixelPNG = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89,
	0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01,
	0x0d, 0x0a, 0x2d, 0xb4,
	0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

// uploadMachineImage загружает снимок и возвращает его путь в хранилище
// и адрес, по которому сервис его раздаёт.
func uploadMachineImage(t require.TestingT, httpBasePath string, admin authSession) (path, url string) {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "shot.png")
	require.NoError(t, err)

	_, err = part.Write(onePixelPNG)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		httpBasePath+"/v1/admin/media/machines", &body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+admin.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "загрузка снимка: %s", payload)

	uploaded := decodeJSONMap(t, string(payload))
	path, _ = uploaded["path"].(string)
	url, _ = uploaded["url"].(string)
	require.NotEmpty(t, path)
	require.NotEmpty(t, url)

	return path, url
}

// fileServed сообщает, отдаёт ли сервис файл по его адресу.
func fileServed(t require.TestingT, httpBasePath, mediaURL string) bool {
	// Фотографии смонтированы в корне сервера, а не под /api: адрес снимка
	// обязан работать там же, где работает каталог, и префикса API у него нет.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		strings.TrimSuffix(httpBasePath, "/api")+mediaURL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode == http.StatusOK
}

// TestHTTPMachineDeleteRemovesValuesImagesAndOrphanFile проходит удаление
// целиком: строка карточки, её значения характеристик, её фотографии и файл
// снимка, на который больше никто не ссылается.
func (s *HTTPTestSuite) TestHTTPMachineDeleteRemovesValuesImagesAndOrphanFile() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "del-cat-"+suffix, "Раздел удаления")
	powerID := createNumberAttribute(s.T(), s.httpBasePath, admin, categoryID, "power_"+suffix)

	imagePath, imageURL := uploadMachineImage(s.T(), s.httpBasePath, admin)
	require.True(s.T(), fileServed(s.T(), s.httpBasePath, imageURL), "загруженный файл раздаётся")

	slug := "del-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 5_000_000,
		PowerID: powerID, Power: "120", ImagePath: imagePath,
	})

	// Значения и фото на месте до удаления.
	status, payload := requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, slug), admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status, "карточка читается: %s", payload)

	machine := decodeJSONMap(s.T(), payload)
	values, _ := machine["attributeValues"].([]any)
	images, _ := machine["images"].([]any)
	require.Len(s.T(), values, 1, "значение характеристики заполнено")
	require.Len(s.T(), images, 1, "фотография привязана")

	// Удаление.
	status, payload = requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, machineID), admin.Token, "")
	require.Equal(s.T(), http.StatusNoContent, status, "удаление карточки: %s", payload)

	// Карточки нет.
	status, _ = requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, slug), admin.Token, "")
	assert.Equal(s.T(), http.StatusNotFound, status, "карточка исчезла")

	// Значения и фото ушли каскадом: слаг снова свободен, а карточка с тем же
	// набором заводится заново без конфликтов по уникальным ключам.
	recreated := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 5_000_000,
		PowerID: powerID, Power: "130",
	})
	assert.NotEqual(s.T(), machineID, recreated, "это новая карточка, а не воскресшая старая")

	// Файл снимка исчез: на него больше никто не ссылался.
	assert.False(s.T(), fileServed(s.T(), s.httpBasePath, imageURL),
		"файл снимка удалённой карточки не должен оставаться на диске")
}

// TestHTTPMachineDeleteKeepsSharedFile фиксирует главное правило чистки
// файлов: один снимок может стоять у нескольких карточек, и удаление одной
// из них файл не трогает.
//
// В импортированном каталоге это не гипотеза: один файл стоит у семи
// тоннажей одной модели.
func (s *HTTPTestSuite) TestHTTPMachineDeleteKeepsSharedFile() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "shared-cat-"+suffix, "Общий снимок")

	imagePath, imageURL := uploadMachineImage(s.T(), s.httpBasePath, admin)

	first := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: "shared-one-" + suffix, PriceCents: 1_000_000,
		ImagePath: imagePath,
	})
	createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: "shared-two-" + suffix, PriceCents: 2_000_000,
		ImagePath: imagePath,
	})

	status, payload := requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, first), admin.Token, "")
	require.Equal(s.T(), http.StatusNoContent, status, "удаление первой карточки: %s", payload)

	assert.True(s.T(), fileServed(s.T(), s.httpBasePath, imageURL),
		"снимок нужен второй карточке и обязан уцелеть")
}

// TestHTTPMachineDeleteRejectedWithQuote фиксирует, что карточку с выпущенным
// предложением удалить нельзя, и что отказ приходит своим кодом, а не
// пересказом нарушения внешнего ключа.
func (s *HTTPTestSuite) TestHTTPMachineDeleteRejectedWithQuote() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "quoted-cat-"+suffix, "С предложением")

	slug := "quoted-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: slug, PriceCents: 9_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	issueQuote(s.T(), s.httpBasePath, admin, slug)

	status, payload := requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, machineID), admin.Token, "")

	require.Equal(s.T(), http.StatusConflict, status, "удаление отвергается: %s", payload)
	assert.Contains(s.T(), payload, entity.CodePrefix+"MACHINE_HAS_QUOTES")

	// Карточка на месте: отказ ничего не поломал на полпути.
	status, _ = requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, slug), admin.Token, "")
	assert.Equal(s.T(), http.StatusOK, status, "после отказа карточка цела")
}

// TestHTTPMachineDeleteKeepsLead фиксирует, что заявка переживает удаление
// карточки: ссылка обнуляется (SET NULL), а сам контакт клиента остаётся.
//
// Терять заявку из-за чистки каталога нельзя: карточка — это позиция
// в прайсе, а заявка — человек, который написал.
func (s *HTTPTestSuite) TestHTTPMachineDeleteKeepsLead() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "lead-cat-"+suffix, "С заявкой")

	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID, Slug: "lead-machine-" + suffix, PriceCents: 3_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	name := "Клиент " + suffix
	body := fmt.Sprintf(`{"name":%q,"phone":%q,"machineID":%q,"source":"machine_card"}`,
		name, uniquePhone(), machineID)

	status, payload := requestJSON(http.MethodPost, s.httpBasePath+"/v1/leads", "", body)
	require.Equal(s.T(), http.StatusCreated, status, "приём заявки: %s", payload)

	leadID, _ := decodeJSONMap(s.T(), payload)["id"].(string)
	require.NotEmpty(s.T(), leadID)

	status, payload = requestJSON(http.MethodDelete,
		fmt.Sprintf("%s/v1/admin/catalog/machines/%s", s.httpBasePath, machineID), admin.Token, "")
	require.Equal(s.T(), http.StatusNoContent, status, "удаление карточки: %s", payload)

	status, payload = requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/leads/%s", s.httpBasePath, leadID), admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status, "заявка пережила удаление карточки: %s", payload)

	lead := decodeJSONMap(s.T(), payload)
	assert.Equal(s.T(), name, lead["name"], "контакт клиента на месте")
	assert.Empty(s.T(), lead["machineID"], "ссылка на удалённую карточку обнулена")
	assert.Nil(s.T(), lead["machine"], "краткой ссылки на карточку больше нет")
}

// TestHTTPMachineDeleteNeedsAdmin фиксирует, что удаление закрыто от менеджера.
func (s *HTTPTestSuite) TestHTTPMachineDeleteNeedsAdmin() {
	manager := managerSession(s.T(), s.httpBasePath)

	status, payload := requestJSON(http.MethodDelete,
		s.httpBasePath+"/v1/admin/catalog/machines/66666666-6666-4666-8666-666666666666",
		manager.Token, "")

	assert.Equal(s.T(), http.StatusForbidden, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"ROLE_FORBIDDEN")
}

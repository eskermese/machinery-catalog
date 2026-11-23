//go:build integration

package integration_test

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPResidualCurveFlow проходит справочник остаточной стоимости целиком
// и доводит его до расчёта владения.
//
// Главное здесь — последняя часть: пока кривой нет, строка «остаточная
// стоимость» честно говорит «нет данных», а как только кривая заведена,
// расчёт становится полным. Это и есть смысл справочника: без него стоимость
// владения всегда считалась не до конца и ЗАНИЖАЛАСЬ.
func (s *HTTPTestSuite) TestHTTPResidualCurveFlow() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin, "residual-"+suffix, "Экскаваторы")

	curveURL := fmt.Sprintf("%s/v1/admin/catalog/residual-curves/%s", s.httpBasePath, categoryID)

	// Своей кривой у раздела нет.
	status, _ := requestJSON(http.MethodGet, curveURL, admin.Token, "")
	assert.Equal(s.T(), http.StatusNotFound, status, "у нового раздела своей кривой нет")

	// Заводим кривую. Раздел — в адресе, в теле его нет.
	body := `{"firstYearDropBps":2000,"annualDropBps":1200,"floorBps":2000}`

	status, payload := requestJSON(http.MethodPut, curveURL, admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "сохранение кривой: %s", payload)

	saved := decodeJSONMap(s.T(), payload)
	assert.Equal(s.T(), categoryID, saved["categoryID"])
	assert.Equal(s.T(), float64(2000), saved["firstYearDropBps"])

	// Вместе с параметрами приходит посчитанная таблица долей по годам:
	// три числа кривой не дают понять, сколько машина будет стоить через
	// пять лет, а заводят её именно ради этого.
	preview, ok := saved["preview"].([]any)
	require.True(s.T(), ok, "кривая отдаётся вместе с таблицей долей")
	require.Len(s.T(), preview, 10)

	first, _ := preview[0].(map[string]any)
	assert.Equal(s.T(), float64(8000), first["rateBps"], "после первого года остаётся 80%")

	// Повторное сохранение переписывает кривую, а не падает конфликтом:
	// у раздела она ровно одна.
	body = `{"firstYearDropBps":2500,"annualDropBps":1000,"floorBps":1500}`

	status, payload = requestJSON(http.MethodPut, curveURL, admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "повторное сохранение: %s", payload)
	assert.Equal(s.T(), float64(2500), decodeJSONMap(s.T(), payload)["firstYearDropBps"])

	// Кривая видна в списке.
	listURL := s.httpBasePath + "/v1/admin/catalog/residual-curves"

	status, payload = requestJSON(http.MethodGet, listURL, admin.Token, "")
	require.Equal(s.T(), http.StatusOK, status)
	assert.Contains(s.T(), payload, categoryID)

	// А теперь то, ради чего справочник и заводится: расчёт владения.
	slug := "residual-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: categoryID,
		Slug:       slug,
		PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	tcoURL := fmt.Sprintf("%s/v1/machines/%s/tco", s.httpBasePath, slug)
	tcoBody := `{"years":5,"hoursPerYear":1500,"fuelPriceKztCents":29000,` +
		`"operatorHourlyKztCents":250000}`

	status, payload = requestJSON(http.MethodPost, tcoURL, "", tcoBody)
	require.Equal(s.T(), http.StatusOK, status, "расчёт владения: %s", payload)

	tco := decodeJSONMap(s.T(), payload)

	residual, ok := tco["residualValue"].(map[string]any)
	require.True(s.T(), ok)
	assert.Contains(s.T(), residual, "amountKztCents",
		"с заведённой кривой остаточная стоимость посчитана, а не «нет данных»")
	assert.NotContains(s.T(), residual, "missing")

	// Убираем кривую — и расчёт снова честно говорит, что не знает.
	status, payload = requestJSON(http.MethodDelete, curveURL, admin.Token, "")
	require.Equal(s.T(), http.StatusNoContent, status, "удаление кривой: %s", payload)

	status, payload = requestJSON(http.MethodPost, tcoURL, "", tcoBody)
	require.Equal(s.T(), http.StatusOK, status)

	residual, ok = decodeJSONMap(s.T(), payload)["residualValue"].(map[string]any)
	require.True(s.T(), ok)
	assert.NotContains(s.T(), residual, "amountKztCents",
		"без кривой строка не посчитана")
	assert.Contains(s.T(), residual, "missing",
		"и говорит об этом словами, а не показывает ноль")
}

// TestHTTPResidualCurveInheritedByChild проверяет наследование кривой вниз
// по дереву разделов.
//
// Ради этого кривая и ищется по предкам: «экскаваторы теряют столько-то»
// верно и для гусеничных, и для колёсных, и повторять это по веткам значило
// бы завести 44 места, где одно и то же число расходится.
func (s *HTTPTestSuite) TestHTTPResidualCurveInheritedByChild() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	parentID := createCategory(s.T(), s.httpBasePath, admin, "residual-parent-"+suffix, "Экскаваторы")
	childID := createChildCategory(s.T(), s.httpBasePath, admin, parentID,
		"residual-child-"+suffix, "Гусеничные")

	// Кривая заводится ТОЛЬКО родителю.
	body := `{"firstYearDropBps":2000,"annualDropBps":1200,"floorBps":2000}`
	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/residual-curves/%s", s.httpBasePath, parentID),
		admin.Token, body)
	require.Equal(s.T(), http.StatusOK, status, "кривая родителя: %s", payload)

	// Своей у ребёнка по-прежнему нет: справочник показывает своё,
	// а не унаследованное, иначе редактор принял бы кривую предка
	// за собственную и завёл бы её копию.
	status, _ = requestJSON(http.MethodGet,
		fmt.Sprintf("%s/v1/admin/catalog/residual-curves/%s", s.httpBasePath, childID),
		admin.Token, "")
	assert.Equal(s.T(), http.StatusNotFound, status, "своей кривой у ребёнка нет")

	// Но в расчёте она действует.
	slug := "residual-child-machine-" + suffix
	machineID := createMachine(s.T(), s.httpBasePath, admin, machineRequest{
		CategoryID: childID,
		Slug:       slug,
		PriceCents: 10_000_000,
	})
	publishMachine(s.T(), s.httpBasePath, admin, machineID)

	status, payload = requestJSON(http.MethodPost,
		fmt.Sprintf("%s/v1/machines/%s/tco", s.httpBasePath, slug), "",
		`{"years":5,"hoursPerYear":1500,"fuelPriceKztCents":29000,"operatorHourlyKztCents":250000}`)
	require.Equal(s.T(), http.StatusOK, status, "расчёт владения: %s", payload)

	residual, ok := decodeJSONMap(s.T(), payload)["residualValue"].(map[string]any)
	require.True(s.T(), ok)
	assert.Contains(s.T(), residual, "amountKztCents",
		"ребёнок считает по кривой предка")
}

// TestHTTPResidualCurveRequiresAdmin фиксирует права: справочник закрыт
// от менеджера целиком.
//
// В отличие от программ лизинга, где менеджер заводит и правит. Разница
// в цене ошибки: выключенная программа ничего не считает и молчит, а кривая
// участвует в расчёте всегда — занижение задирает стоимость владения сразу
// на всех карточках раздела, и по одной карточке этого не заметить.
func (s *HTTPTestSuite) TestHTTPResidualCurveRequiresAdmin() {
	manager := managerSession(s.T(), s.httpBasePath)

	status, payload := requestJSON(http.MethodGet,
		s.httpBasePath+"/v1/admin/catalog/residual-curves", manager.Token, "")

	require.Equal(s.T(), http.StatusForbidden, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"ROLE_FORBIDDEN")
}

// TestHTTPResidualCurveValidation отвергает кривую, которая росла бы со временем.
func (s *HTTPTestSuite) TestHTTPResidualCurveValidation() {
	admin := adminSession(s.T(), s.httpBasePath)

	suffix := uniqueSuffix()
	categoryID := createCategory(s.T(), s.httpBasePath, admin,
		"residual-invalid-"+suffix, "Экскаваторы")

	// Порог 90% выше того, что остаётся после первого года (80%): такая
	// кривая сделала бы технику дорожающей со второго года.
	body := `{"firstYearDropBps":2000,"annualDropBps":1200,"floorBps":9000}`

	status, payload := requestJSON(http.MethodPut,
		fmt.Sprintf("%s/v1/admin/catalog/residual-curves/%s", s.httpBasePath, categoryID),
		admin.Token, body)

	require.Equal(s.T(), http.StatusBadRequest, status)
	assert.Contains(s.T(), payload, entity.CodePrefix+"RESIDUAL_CURVE_FLOOR_ABOVE_FIRST_YEAR")
}

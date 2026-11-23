package http

import (
	"encoding/json"
	nethttp "net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRouterMatchesSpec сверяет маршруты роутера со сгенерированной спекой
// в обе стороны.
//
// Расхождение в любую сторону — дефект контракта: описанная, но не поднятая
// ручка даёт клиенту 404 по спеке, а поднятая, но не описанная не существует
// для того, кто генерирует по спеке типы.
func TestRouterMatchesSpec(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../../swagger/swagger.json")
	require.NoError(t, err, "спека должна быть сгенерирована: make swag")

	var spec struct {
		BasePath string                                `json:"basePath"`
		Paths    map[string]map[string]json.RawMessage `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(raw, &spec))

	inSpec := make(map[string]struct{}, len(spec.Paths))

	for path, operations := range spec.Paths {
		for method := range operations {
			inSpec[strings.ToUpper(method)+" "+spec.BasePath+path] = struct{}{}
		}
	}

	inRouter := make(map[string]struct{})

	walk := func(method, route string, _ nethttp.Handler, _ ...func(nethttp.Handler) nethttp.Handler) error {
		route = strings.TrimSuffix(route, "/*")

		// Сервисные маршруты в спеку не входят: их дёргает инфраструктура.
		for _, service := range []string{"/metrics", "/version", "/files", "/swagger", "/media"} {
			if strings.HasPrefix(route, service) {
				return nil
			}
		}

		inRouter[method+" "+route] = struct{}{}

		return nil
	}

	require.NoError(t, chi.Walk(testRouter(), walk))

	// Обе стороны непусты: иначе сверка проходила бы вхолостую, сравнивая
	// пустое множество с пустым.
	require.NotEmpty(t, inSpec, "спека пуста")
	require.NotEmpty(t, inRouter, "роутер не отдал ни одного прикладного маршрута")
	require.Equal(t, len(inSpec), len(inRouter), "число операций совпадает")

	var missingInSpec, missingInRouter []string

	for route := range inRouter {
		if _, ok := inSpec[route]; !ok {
			missingInSpec = append(missingInSpec, route)
		}
	}

	for route := range inSpec {
		if _, ok := inRouter[route]; !ok {
			missingInRouter = append(missingInRouter, route)
		}
	}

	assert.Empty(t, missingInSpec, "маршрут поднят, но не описан в спеке")
	assert.Empty(t, missingInRouter, "маршрут описан в спеке, но не поднят")
}

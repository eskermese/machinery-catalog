//go:build integration

package integration_test

import (
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestHTTPManagerLoginCreatesAccountOnce проверяет на живой базе, что вход
// менеджера идемпотентен: учётная запись создаётся при первом входе, а повторный
// вход тем же номером возвращает тот же идентификатор.
func (s *HTTPTestSuite) TestHTTPManagerLoginCreatesAccountOnce() {
	phone := uniquePhone()

	first := login(s.httpBasePath, entity.ActorTypeManager, phone)
	require.NotEmpty(s.T(), first.Token, "вход должен выдавать токен")
	require.NotEmpty(s.T(), first.ID, "клиенту нужен идентификатор: иначе его негде взять")

	// Вход в обход кэша: проверяется поведение сервиса, а не памяти набора.
	second := loginFresh(s.httpBasePath, entity.ActorTypeManager, phone)

	assert.Equal(s.T(), first.ID, second.ID, "повторный вход не должен заводить вторую учётную запись")
}

// TestHTTPManagerLoginNormalizesPhone проверяет, что номер в разных форматах
// записи приводит к одной и той же учётной записи.
func (s *HTTPTestSuite) TestHTTPManagerLoginNormalizesPhone() {
	phone := uniquePhone()

	canonical := login(s.httpBasePath, entity.ActorTypeManager, phone)
	require.NotEmpty(s.T(), canonical.ID)

	// Тот же номер, записанный с разделителями.
	spaced := loginFresh(s.httpBasePath, entity.ActorTypeManager,
		phone[:2]+" "+phone[2:5]+" "+phone[5:8]+"-"+phone[8:])

	assert.Equal(s.T(), canonical.ID, spaced.ID, "формат записи номера не должен заводить вторую запись")
}

// TestHTTPAdminLoginRequiresSeededPhone проверяет, что администратором может
// войти только заранее заведённый номер: самостоятельная регистрация означала бы,
// что права раздаёт тот, кто их запрашивает.
func (s *HTTPTestSuite) TestHTTPAdminLoginRequiresSeededPhone() {
	assert.Equal(s.T(), http.StatusOK, adminLoginStatus(s.httpBasePath, adminPhone),
		"заведённый через ADMIN_PHONES номер должен входить")

	assert.Equal(s.T(), http.StatusForbidden, adminLoginStatus(s.httpBasePath, uniquePhone()),
		"незаведённый номер не должен получать права администратора")
}

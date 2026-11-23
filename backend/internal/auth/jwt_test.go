package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

const testSecret = "test-secret"

// testNow момент выпуска токенов в тестах.
//
// Считается от текущего времени, а не фиксированной датой: при фиксированной дате
// тест разбора зелёный ровно TokenTTL после её наступления, а дальше начинает
// падать на истёкшем токене — отказ, не связанный с проверяемым поведением.
var testNow = time.Now().UTC()

// TestTokenService_IssueAndParse фиксирует круг: выпущенный токен разбирается
// обратно в того же действующего пользователя.
func TestTokenService_IssueAndParse(t *testing.T) {
	t.Parallel()

	svc := NewTokenService(testSecret, TokenTTL)

	token, err := svc.Issue("8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b", entity.ActorTypeManager, testNow)
	require.NoError(t, err)

	actor, err := svc.Parse(token)
	require.NoError(t, err)
	require.NotNil(t, actor.ID)
	assert.Equal(t, "8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b", *actor.ID)
	assert.Equal(t, entity.ActorTypeManager, actor.Type)
}

// TestTokenService_Parse_Rejects фиксирует отказы разбора.
//
// Истёкший токен отделён от прочих причин: клиенту нужно отличать «перелогинься»
// от «ты не аутентифицирован». Остальные причины снаружи неотличимы намеренно.
func TestTokenService_Parse_Rejects(t *testing.T) {
	t.Parallel()

	svc := NewTokenService(testSecret, TokenTTL)

	valid, err := svc.Issue("8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b", entity.ActorTypeManager, testNow)
	require.NoError(t, err)

	expired, err := NewTokenService(testSecret, time.Hour).
		Issue("8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b", entity.ActorTypeManager, time.Now().UTC().Add(-2*time.Hour))
	require.NoError(t, err)

	foreign, err := NewTokenService("другой-секрет", TokenTTL).
		Issue("8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b", entity.ActorTypeManager, time.Now().UTC())
	require.NoError(t, err)

	// Токен без подписи: классическая атака alg=none.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		Role: entity.ActorTypeAdmin.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b",
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	// Токен без роли: подпись верна, но действующего пользователя из него не собрать.
	roleless, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b",
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
	}).SignedString([]byte(testSecret))
	require.NoError(t, err)

	cases := []struct {
		name   string
		token  string
		expErr error
	}{
		{name: "валидный проходит", token: valid},
		{name: "истёкший", token: expired, expErr: entity.ErrTokenExpired},
		{name: "чужая подпись", token: foreign, expErr: entity.ErrAuthRequired},
		{name: "без подписи (alg=none)", token: unsigned, expErr: entity.ErrAuthRequired},
		{name: "без роли", token: roleless, expErr: entity.ErrAuthRequired},
		{name: "мусор", token: "не-токен", expErr: entity.ErrAuthRequired},
		{name: "пустой", token: "", expErr: entity.ErrAuthRequired},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.Parse(s.token)

			if s.expErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, s.expErr)
		})
	}
}

// TestTokenService_SecretMissing фиксирует, что без секрета сервис отказывает,
// а не выпускает токены, которые никто не сможет проверить.
func TestTokenService_SecretMissing(t *testing.T) {
	t.Parallel()

	svc := NewTokenService("", TokenTTL)

	assert.False(t, svc.IsConfigured())

	_, err := svc.Issue("id", entity.ActorTypeManager, testNow)
	require.ErrorIs(t, err, ErrSecretMissing)

	_, err = svc.Parse("любой")
	require.ErrorIs(t, err, ErrSecretMissing)
}

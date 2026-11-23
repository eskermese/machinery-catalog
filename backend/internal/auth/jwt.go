// Package auth выпускает и проверяет токены доступа.
//
// Схема временная: собственного IdP у Заказчика ещё нет, поэтому сервис сам
// выдаёт JWT по номеру телефона. Пакет намеренно узкий — выпуск и разбор токена, —
// чтобы замена на внешний IdP свелась к другой реализации TokenService.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TokenTTL срок жизни токена. Refresh-токенов нет: по истечении клиент логинится заново.
const TokenTTL = 24 * time.Hour

// signingMethod алгоритм подписи. HMAC достаточно: и выпуск, и проверка живут
// в одном сервисе, распределять публичные ключи некому.
var signingMethod = jwt.SigningMethodHS256

var (
	// ErrSecretMissing сообщает, что сервис не сконфигурирован для работы с токенами.
	ErrSecretMissing = errors.New("секрет подписи токенов не задан")
	// ErrUnexpectedAlg сообщает о токене, подписанном другим алгоритмом.
	ErrUnexpectedAlg = errors.New("неожиданный алгоритм подписи токена")
)

// Claims полезная нагрузка токена. sub — идентификатор пользователя, role — его роль.
type Claims struct {
	Role string `json:"role"`

	jwt.RegisteredClaims
}

// TokenService выпускает и разбирает токены доступа.
type TokenService struct {
	secret []byte        // Секрет подписи
	ttl    time.Duration // Срок жизни токена
}

// NewTokenService создаёт сервис токенов.
// Пустой секрет допускается: решение о том, фатально это или нет, принимает
// вызывающий код по окружению, а сам сервис в таком виде просто отказывает.
func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), ttl: ttl}
}

// IsConfigured сообщает, что секрет задан и токены можно выпускать.
func (s *TokenService) IsConfigured() bool {
	return s != nil && len(s.secret) > 0
}

// Issue выпускает токен для пользователя.
func (s *TokenService) Issue(userID string, role entity.ActorType, currentTime time.Time) (string, error) {
	if !s.IsConfigured() {
		return "", ErrSecretMissing
	}

	claims := Claims{
		Role: role.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(currentTime),
			ExpiresAt: jwt.NewNumericDate(currentTime.Add(s.ttl)),
		},
	}

	token, err := jwt.NewWithClaims(signingMethod, claims).SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("подпись токена: %w", err)
	}

	return token, nil
}

// Parse проверяет подпись и срок действия токена и возвращает действующего пользователя.
//
// Истёкший токен отделён от остальных отказов доменной ошибкой: клиенту нужно
// понять, что достаточно перелогиниться. Все прочие причины — битая подпись,
// чужой алгоритм, отсутствующие поля — снаружи неотличимы намеренно.
func (s *TokenService) Parse(token string) (entity.Actor, error) {
	if !s.IsConfigured() {
		return entity.Actor{}, ErrSecretMissing
	}

	claims := &Claims{}

	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		// Алгоритм проверяется явно: иначе токен, подписанный «none» или RS256
		// с подставленным ключом, прошёл бы проверку.
		if t.Method.Alg() != signingMethod.Alg() {
			return nil, fmt.Errorf("%w: %s", ErrUnexpectedAlg, t.Method.Alg())
		}

		return s.secret, nil
	}, jwt.WithValidMethods([]string{signingMethod.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return entity.Actor{}, entity.ErrTokenExpired
		}

		return entity.Actor{}, entity.ErrAuthRequired
	}

	role := entity.ActorType(claims.Role)
	if claims.Subject == "" || !entity.IsLoginRole(role) {
		return entity.Actor{}, entity.ErrAuthRequired
	}

	return entity.NewActor(claims.Subject, role), nil
}

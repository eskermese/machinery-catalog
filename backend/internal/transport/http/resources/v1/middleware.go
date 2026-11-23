package v1

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	httpx "github.com/eskermese/machinery-catalog/backend/pkg/httpx"
)

// Заголовки, которые проставляет доверенный шлюз.
const (
	HeaderXUserRole     = "X-User-Role"     // Роль действующего пользователя
	HeaderXGatewayToken = "X-Gateway-Token" // Общий секрет доверенного шлюза
)

// tokenParser проверяет токен доступа и возвращает действующего пользователя.
// Интерфейс объявлен здесь, где потребляется: транспорту важен только результат
// проверки, а замена собственных JWT на внешний IdP не должна его затрагивать.
type tokenParser interface {
	Parse(token string) (entity.Actor, error)
}

// AuthMiddleware аутентифицирует запрос по токену доступа.
//
// Единственная продовая схема — Bearer-токен: обратный прокси Заказчика ничего
// не проверяет и передаёт заголовок как есть, поэтому подпись проверяет сервис.
//
// Аутентификация отделена от авторизации намеренно: здесь выясняется, КТО пришёл,
// а соответствие роли разделу проверяет RoleMiddleware. Актор кладётся в контекст,
// и дальше никто не читает заголовки.
func AuthMiddleware(tokens tokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeAuthError(w, r, entity.ErrAuthRequired, "токен не передан")

				return
			}

			actor, err := tokens.Parse(token)
			if err != nil {
				// Истёкший токен отделён от прочих отказов: клиенту нужно понять,
				// что достаточно перелогиниться. Остальные причины снаружи
				// неотличимы — по ним удобно подбирать доступ.
				if errors.Is(err, entity.ErrTokenExpired) {
					writeAuthError(w, r, entity.ErrTokenExpired, "срок действия токена истёк")

					return
				}

				writeAuthError(w, r, entity.ErrAuthRequired, "токен не прошёл проверку")

				return
			}

			next.ServeHTTP(w, r.WithContext(actorContext(r.Context(), actor)))
		})
	}
}

// bearerToken достаёт токен из заголовка Authorization.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "

	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}

	return strings.TrimSpace(header[len(prefix):]), true
}

// actorContext кладёт актора и логгер с его идентификатором в контекст запроса.
// Одна точка на оба режима аутентификации, чтобы они не разъезжались.
func actorContext(ctx context.Context, actor entity.Actor) context.Context {
	id := ""
	if actor.ID != nil {
		id = *actor.ID
	}

	return withActorLogger(withActor(ctx, actor), id, nil)
}

// GatewayMiddleware пропускает только запросы доверенного шлюза.
//
// Собственной аутентификации у сервиса нет: identity живет за шлюзом, а сервис
// доверяет заголовкам X-User-Id и X-User-Role ровно потому, что запрос пришел
// с общим секретом. Без этой проверки роль из заголовка не значит ничего.
//
// Секрет сверяется постоянным по времени сравнением: обычное сравнение строк
// прекращается на первом различии, и по времени ответа секрет можно подобрать
// побайтно.
//
// Пустой секрет в local выключает проверку — иначе локальный запуск требовал бы
// конфигурации. Непустой секрет включает ее в любом окружении. Вне local пустой
// секрет считается ошибкой конфигурации, и сервис закрывается, а не открывается:
// молча пускать всех в проде хуже, чем не отвечать вовсе.
func GatewayMiddleware(token string, local bool) func(http.Handler) http.Handler {
	var (
		expected = []byte(token)
		disabled = token == "" && local
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if disabled {
				next.ServeHTTP(w, r)

				return
			}

			if len(expected) == 0 {
				writeAuthError(w, r, entity.ErrAuthRequired, "секрет шлюза не сконфигурирован")

				return
			}

			if subtle.ConstantTimeCompare(expected, []byte(r.Header.Get(HeaderXGatewayToken))) != 1 {
				writeAuthError(w, r, entity.ErrAuthRequired, "неверный секрет шлюза")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RoleMiddleware пропускает только запросы с ролью, соответствующей группе маршрутов.
//
// Заголовков больше не читает: кто пришёл, уже выяснила аутентификация и положила
// актора в контекст. Здесь решается только вопрос прав — отсюда разделение
// 401 (не аутентифицирован) и 403 (аутентифицирован, но не в свой раздел).
func RoleMiddleware(expected entity.ActorType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := actorFromContext(r.Context())
			if actor.ID == nil || actor.Type == "" {
				// Обработчик смонтирован мимо аутентификации либо она не заполнила
				// актора: пускать дальше нельзя ни при каких условиях.
				writeAuthError(w, r, entity.ErrAuthRequired, "запрос не аутентифицирован")

				return
			}

			if actor.Type != expected {
				writeAuthError(w, r, entity.ErrRoleForbidden, "роль не соответствует разделу")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RoleAnyMiddleware пропускает запросы с любой из перечисленных ролей.
//
// Отдельная функция, а не RoleMiddleware со списком: разделы с единственной
// допустимой ролью — большинство, и для них список из одного элемента читался
// бы как недосмотр. Здесь же перечисление ролей — осознанное решение о том,
// что раздел общий.
func RoleAnyMiddleware(allowed ...entity.ActorType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := actorFromContext(r.Context())
			if actor.ID == nil || actor.Type == "" {
				writeAuthError(w, r, entity.ErrAuthRequired, "запрос не аутентифицирован")

				return
			}

			for _, role := range allowed {
				if actor.Type == role {
					next.ServeHTTP(w, r)

					return
				}
			}

			writeAuthError(w, r, entity.ErrRoleForbidden, "роль не соответствует разделу")
		})
	}
}

// DebugHeaderAuthMiddleware аутентифицирует запрос по заголовкам X-User-Id и X-User-Role.
//
// Отладочная схема прежнего образца: применяется ТОЛЬКО когда вызывающий код
// явно её включил (ENVIRONMENT_NAME=local и DEBUG_HEADER_AUTH=true). Вне local
// эта ветка не подключается к роутеру вовсе — заголовкам, пришедшим снаружи,
// доверять нельзя, а «режим» в конфигурации прода стал бы дырой ровно на один
// неверно выставленный флаг.
func DebugHeaderAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := entity.ActorType(r.Header.Get(HeaderXUserRole))
			if !entity.IsLoginRole(role) {
				writeAuthError(w, r, entity.ErrAuthRequired, "роль не передана")

				return
			}

			id := r.Header.Get(HeaderXUserID)
			if _, err := uuid.Parse(id); err != nil {
				writeAuthError(w, r, entity.ErrAuthRequired, "идентификатор пользователя не является UUID")

				return
			}

			next.ServeHTTP(w, r.WithContext(actorContext(r.Context(), entity.NewActor(id, role))))
		})
	}
}

// writeAuthError логирует причину отказа и отвечает доменной ошибкой.
// Клиенту уходит только код ошибки: конкретная причина отказа — внутренняя
// информация, по которой удобно подбирать доступ.
func writeAuthError(w http.ResponseWriter, r *http.Request, err error, reason string) {
	httpx.LogRequestError(r, reason, err, nil)
	httpx.WriteError(w, err)
}

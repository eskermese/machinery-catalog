package v1

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// HeaderXUserID заголовок с идентификатором действующего пользователя,
// который проставляет доверенный шлюз.
const HeaderXUserID = "X-User-Id"

// actorCtxKey типизированный ключ актора в контексте запроса.
// Собственный тип, а не строка: чужой пакет не сможет ни прочитать, ни подменить
// актора, случайно совпав ключом.
type actorCtxKey struct{}

// withActor кладёт действующего пользователя в контекст запроса.
// Единственная точка записи: обработчики читают актора отсюда, а не из заголовков.
func withActor(ctx context.Context, actor entity.Actor) context.Context {
	return context.WithValue(ctx, actorCtxKey{}, actor)
}

// actorFromContext возвращает действующего пользователя из контекста запроса.
// Пустой актор означает, что аутентификация не отработала: вызывающий код
// обязан считать это отказом, а не анонимным доступом.
func actorFromContext(ctx context.Context) entity.Actor {
	actor, _ := ctx.Value(actorCtxKey{}).(entity.Actor)

	return actor
}

// withActorLogger кладёт в контекст логгер, помеченный идентификатором
// действующего пользователя, чтобы записи всей обработки запроса связывались
// с тем, кто её инициировал.
func withActorLogger(ctx context.Context, userID string, fields logger.Fields) context.Context {
	log := logger.FromContext(ctx).WithFields(logger.Fields{"user_id": userID})

	if len(fields) > 0 {
		log = log.WithFields(fields)
	}

	return logger.WithLogger(ctx, log)
}

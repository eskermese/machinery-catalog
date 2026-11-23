package v1

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// WithActorForTest кладёт актора в контекст запроса из внешнего тестового пакета.
//
// Экспортируется только для тестов (файл с суффиксом _test.go в сборку
// не попадает): сами тесты живут во внешнем пакете v1_test, чтобы проверять
// публичный контракт обработчиков, но проставить актора без доступа
// к приватному ключу контекста иначе нельзя.
func WithActorForTest(ctx context.Context, actor entity.Actor) context.Context {
	return withActor(ctx, actor)
}

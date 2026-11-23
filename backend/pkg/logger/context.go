package logger

import "context"

// DefaultLoggerInContext используется, если логгер не найден в context.Context.
var DefaultLoggerInContext = NewStub()

// ctxKey — типизированный ключ для хранения логгера в контексте.
type ctxKey struct{}

// WithLogger добавляет логгер в context.Context.
func WithLogger(ctx context.Context, log Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext извлекает логгер из контекста.
// Если логгер не найден, возвращает Stub (пустой логгер).
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return DefaultLoggerInContext
	}

	if log, ok := ctx.Value(ctxKey{}).(Logger); ok && log != nil {
		return log
	}

	return DefaultLoggerInContext
}

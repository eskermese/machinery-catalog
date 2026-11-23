package logger

import "os"

// Stub стаб для пустого логгера.
type Stub struct{}

// NewStub создает новый Stub.
func NewStub() Logger {
	return &Stub{}
}

// Debug ничего не делает. Реализует интерфейс Logger для вызовов Debug(...).
func (s *Stub) Debug(_ ...interface{}) {}

// Debugf ничего не делает. Реализует форматированный вариант Debug.
func (s *Stub) Debugf(_ string, _ ...interface{}) {}

// Info ничего не делает. Реализует интерфейс Logger для вызовов Info(...).
func (s *Stub) Info(_ ...interface{}) {}

// Infof ничего не делает. Реализует форматированный вариант Info.
func (s *Stub) Infof(_ string, _ ...interface{}) {}

// Warn ничего не делает. Реализует интерфейс Logger для вызовов Warn(...).
func (s *Stub) Warn(_ ...interface{}) {}

// Warnf ничего не делает. Реализует форматированный вариант Warn.
func (s *Stub) Warnf(_ string, _ ...interface{}) {}

// Error ничего не делает. Реализует интерфейс Logger для вызовов Error(...).
func (s *Stub) Error(_ ...interface{}) {}

// Errorf ничего не делает. Реализует форматированный вариант Error.
func (s *Stub) Errorf(_ string, _ ...interface{}) {}

// Fatal ничего не делает и не завершает выполнение программы.
// В отличие от реальных логгеров, Stub не вызывает os.Exit.
func (s *Stub) Fatal(_ ...interface{}) {
	os.Exit(1)
}

// Fatalf ничего не делает и не завершает выполнение программы.
func (s *Stub) Fatalf(_ string, _ ...interface{}) {
	os.Exit(1)
}

// WithFields возвращает сам себя, не добавляя никаких полей.
// Метод нужен для совместимости с интерфейсом Logger.
func (s *Stub) WithFields(_ Fields) Logger {
	return s
}

// String всегда возвращает пустую строку и nil-ошибку.
// Может использоваться для проверки, что логгер не инициализирован.
func (s *Stub) String() (string, error) {
	return "", nil
}

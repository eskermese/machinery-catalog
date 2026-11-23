package logger

import (
	"github.com/sirupsen/logrus"
)

// Fields представляет собой поля для логгера.
type Fields logrus.Fields

const (
	Fatal = "fatal" // Fatal логгирует сообщение о фатальной ошибке.
	Error = "error" // Error логгирует сообщение об ошибке.
	Warn  = "warn"  // Warn логгирует предупреждение.
	Info  = "info"  // Info логгирует информационное сообщение.
	Debug = "debug" // Debug логгирует отладочное сообщение.
)

// Logger представляет собой логгер.
type Logger interface {
	// Debug логгирует отладочное сообщение.
	Debug(args ...interface{})
	// Debugf логгирует форматированное отладочное сообщение.
	Debugf(format string, args ...interface{})
	// Info логгирует информационное сообщение.
	Info(args ...interface{})
	// Infof логгирует форматированное информационное сообщение.
	Infof(format string, args ...interface{})
	// Warn логгирует предупреждение.
	Warn(args ...interface{})
	// Warnf логгирует форматированное предупреждение.
	Warnf(format string, args ...interface{})
	// Error логгирует сообщение об ошибке.
	Error(args ...interface{})
	// Errorf логгирует форматированное сообщение об ошибке.
	Errorf(format string, args ...interface{})
	// Fatal логгирует сообщение о фатальной ошибке.
	Fatal(args ...interface{})
	// Fatalf логгирует форматированное сообщение о фатальной ошибке.
	Fatalf(format string, args ...interface{})
	// WithFields возвращает новый логгер с указанными полями.
	WithFields(fields Fields) Logger
	// String возвращает строковое представление логгера.
	String() (string, error)
}

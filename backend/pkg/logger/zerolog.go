package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

// getLevel получает уровень.
func getLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case Debug:
		return zerolog.DebugLevel
	case Info:
		return zerolog.InfoLevel
	case Warn:
		return zerolog.WarnLevel
	case Error:
		return zerolog.ErrorLevel
	case Fatal:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// Zerologger реализует интерфейс Logger на базе rs/zerolog.
type Zerologger struct {
	l zerolog.Logger // Внутренний логгер zerolog.
}

// once гарантирует, что глобальные настройки zerolog устанавливаются только один раз.
var once sync.Once

// NewZeroLogger создает новый экземпляр Zerologger.
func NewZeroLogger(level, serviceName, version string) Logger {
	return newZerologger(level, serviceName, version, os.Stdout)
}

// newZerologger создает Zerologger с указанным объектом записи.
func newZerologger(level, serviceName, version string, writer io.Writer) *Zerologger {
	once.Do(func() {
		zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000Z07:00"
	})

	// Устанавливаем уровень логирования
	lvl := getLevel(level)

	skipFrameCount := 3

	zlCtx := zerolog.New(writer).
		Level(lvl).
		With().
		Timestamp().
		CallerWithSkipFrameCount(skipFrameCount)

	if serviceName != "" {
		zlCtx = zlCtx.Str("service", serviceName)
	}

	if version != "" {
		zlCtx = zlCtx.Str("v", version)
	}

	return &Zerologger{l: zlCtx.Logger()}
}

// Debug логгирует отладочное сообщение.
func (z *Zerologger) Debug(args ...interface{}) {
	z.l.Debug().Msg(fmt.Sprint(args...))
}

// Debugf логгирует форматированное отладочное сообщение.
func (z *Zerologger) Debugf(format string, args ...interface{}) {
	z.l.Debug().Msgf(format, args...)
}

// Info логгирует информационное сообщение.
func (z *Zerologger) Info(args ...interface{}) {
	z.l.Info().Msg(fmt.Sprint(args...))
}

// Infof логгирует форматированное информационное сообщение.
func (z *Zerologger) Infof(format string, args ...interface{}) {
	z.l.Info().Msgf(format, args...)
}

// Warn логгирует предупреждение.
func (z *Zerologger) Warn(args ...interface{}) {
	z.l.Warn().Msg(fmt.Sprint(args...))
}

// Warnf логгирует форматированное предупреждение.
func (z *Zerologger) Warnf(format string, args ...interface{}) {
	z.l.Warn().Msgf(format, args...)
}

// Error логгирует сообщение об ошибке.
func (z *Zerologger) Error(args ...interface{}) {
	z.l.Error().Msg(fmt.Sprint(args...))
}

// Errorf логгирует форматированное сообщение об ошибке.
func (z *Zerologger) Errorf(format string, args ...interface{}) {
	z.l.Error().Msgf(format, args...)
}

// Fatal логгирует сообщение о фатальной ошибке и завершает выполнение программы.
func (z *Zerologger) Fatal(args ...interface{}) {
	z.l.Fatal().Msg(fmt.Sprint(args...))
}

// Fatalf логгирует форматированное сообщение о фатальной ошибке и завершает выполнение программы.
func (z *Zerologger) Fatalf(format string, args ...interface{}) {
	z.l.Fatal().Msgf(format, args...)
}

// WithFields возвращает новый логгер с указанными полями.
func (z *Zerologger) WithFields(fields Fields) Logger {
	ctx := z.l.With()

	for k, v := range map[string]interface{}(logrus.Fields(fields)) {
		appendField(&ctx, k, v)
	}

	return &Zerologger{l: ctx.Logger()}
}

// appendField добавляет одно поле в контекст логгера, разбирая одноуровневый
// interface{} вручную для частых типов — иначе zerolog сериализует их через reflect.
// ctx передается указателем: zerolog.Context тяжелый для копирования.
//
//nolint:gocyclo,cyclop,gocognit // Плоская диспетчеризация по типу: ветвей много, но логика в каждой тривиальна.
func appendField(ctx *zerolog.Context, k string, v interface{}) {
	if v == nil {
		*ctx = ctx.Interface(k, nil)

		return
	}

	switch vv := v.(type) {
	case []byte:
		var buf bytes.Buffer
		if err := json.Compact(&buf, vv); err == nil {
			*ctx = ctx.RawJSON(k, buf.Bytes())
		} else {
			*ctx = ctx.Interface(k, vv)
		}
	case error:
		var errStr string
		if vv != nil {
			errStr = fmt.Sprint(vv)
		}

		*ctx = ctx.Str(k, errStr)
	case fmt.Stringer:
		*ctx = ctx.Str(k, vv.String())
	case *string:
		if vv == nil {
			*ctx = ctx.Interface(k, nil)
		} else {
			*ctx = ctx.Str(k, *vv)
		}
	case *int:
		if vv == nil {
			*ctx = ctx.Interface(k, nil)
		} else {
			*ctx = ctx.Int(k, *vv)
		}
	case *bool:
		if vv == nil {
			*ctx = ctx.Interface(k, nil)
		} else {
			*ctx = ctx.Bool(k, *vv)
		}
	case *float64:
		if vv == nil {
			*ctx = ctx.Interface(k, nil)
		} else {
			*ctx = ctx.Float64(k, *vv)
		}
	default:
		*ctx = ctx.Interface(k, vv)
	}
}

// String возвращает строковое представление текущего уровня логгера.
func (z *Zerologger) String() (string, error) {
	return z.l.GetLevel().String(), nil
}

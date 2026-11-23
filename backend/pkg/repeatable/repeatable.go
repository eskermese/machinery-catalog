package repeatable

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// DoWithTries - повторяет выполнение функции fn, пока не будет выполнена успешно.
func DoWithTries(fn func() error, attempts int, delay time.Duration) error {
	if attempts <= 0 {
		return nil
	}

	if attempts == 1 {
		return fn()
	}

	//nolint: gosec,gocritic // OK
	attemptsUINT := uint64(attempts - 1)

	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(delay), attemptsUINT)

	return backoff.Retry(fn, bo)
}

//go:build integration

package integration_test

import (
	"net/http"
	"time"

	. "github.com/Eun/go-hit"

	"github.com/eskermese/machinery-catalog/backend/pkg/repeatable"
)

// httpHealthCheck проверяет доступность сервера.
func (s *HTTPTestSuite) httpHealthCheck() error {
	attempts := 20

	return repeatable.DoWithTries(func() error {
		return Do(
			Get(s.httpHealthPath),
			Expect().Status().Equal(http.StatusOK),
		)
	}, attempts, time.Second)
}

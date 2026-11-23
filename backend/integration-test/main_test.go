//go:build integration

package integration_test

import (
	"os"
	"testing"

	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
	"github.com/ory/dockertest/v3"

	test_init "github.com/eskermese/machinery-catalog/backend/integration-test/test-init"
)

var testLog = logger.NewZeroLogger(logger.Info, "", "")

// TestMain точка входа в тесты.
func TestMain(m *testing.M) {
	testLog.Info("Запуск TestMain")

	os.Exit(testMain(m))
}

// testMain запускает docker сервисы и тесты.
func testMain(m *testing.M) int {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLog.Warnf("Ошибка создания docker пула: %s", err)
		return 1
	}

	testLog.Info("Инициализация docker сервисов...")

	err = test_init.InitServices(pool)
	if err != nil {
		testLog.Warnf("Ошибка инициализаций docker сервисов: %s", err)
		test_init.ShutdownServices(pool, false)

		return 1
	}

	code := m.Run()

	test_init.ShutdownServices(pool, code != 0)

	return code
}

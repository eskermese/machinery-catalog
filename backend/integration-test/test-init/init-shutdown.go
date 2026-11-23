//go:build integration

package test_init

import (
	"fmt"
	"os"
	"slices"
	"strings"

	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"golang.org/x/sync/errgroup"
)

// InitServices поднимает сервисы.
func InitServices(pool *dockertest.Pool) error {
	err := cleanenv.ReadConfig("../config/integrations.yml", &Cfg)
	if err != nil {
		return fmt.Errorf("парсинг integrations.yml: %w", err)
	}

	Cfg.generateBranchName()
	Cfg.scopeHostnames()

	Logrus = logger.NewZeroLogger(Cfg.LogLevel, "", "")
	Logrus.Infof("Конфигурация интеграционного теста: %+v", Cfg)

	if err = createNetwork(pool); err != nil {
		return fmt.Errorf("создание общей сети: %w", err)
	}

	g := errgroup.Group{}

	g.Go(func() error {
		if err = initDatabaseService(pool); err != nil {
			return fmt.Errorf("поднятие postgres: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		if err = initCacheService(pool); err != nil {
			return fmt.Errorf("поднятие кэша: %w", err)
		}

		return nil
	})

	if err = g.Wait(); err != nil {
		return fmt.Errorf("поднятие сервисов: %w", err)
	}

	if err = initAppService(pool); err != nil {
		return fmt.Errorf("поднятие приложения: %w", err)
	}

	return nil
}

// ShutdownServices останавливает сервисы.
func ShutdownServices(pool *dockertest.Pool, isPrintAppLogs bool) {
	Logrus.Info("Остановка сервисов...")

	// Копируем список ресурсов, чтобы не менять исходный.
	// Переворачиваем список, чтобы сначала закрыть контейнер с приложением.
	resourcesCopy := resources.getCopy()
	slices.Reverse(resourcesCopy)

	for _, resource := range resourcesCopy {
		if resource == nil {
			continue
		}

		Logrus.Infof("Удаление контейнера %s", resource.Container.Name)

		if strings.Contains(resource.Container.Name, "app-integration") && isPrintAppLogs {
			// Выводим логи приложения.
			Logrus.Info("==== Application logs ====")

			err := pool.Client.Logs(docker.LogsOptions{
				Container:    resource.Container.ID,
				OutputStream: os.Stdout,
				ErrorStream:  os.Stderr,
				Stdout:       true,
				Stderr:       true,
			})
			if err != nil {
				Logrus.Warnf("логи приложения: %s", err)
			}

			Logrus.Info("==== End logs ====")
		}

		if err := resource.Close(); err != nil {
			Logrus.Warnf("закрытие ресурса: %s", err)
		}
	}
}

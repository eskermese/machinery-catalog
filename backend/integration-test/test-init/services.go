//go:build integration

package test_init

import (
	"fmt"
	"sync"

	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	"github.com/eskermese/machinery-catalog/backend/pkg/cache"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	Cfg Config // Конфигурация интеграционных тестов.

	Cache     cache.Cache          // Кэш.
	DataStore repository.DataStore // Работа с базой данных.

	// Logrus логирует запросы и ошибки интеграционных тестов.
	Logrus = logger.NewZeroLogger(logger.Info, "", "")

	resources       = &safeResources{}  // Список ресурсов которые нужно закрыть после тестов.
	network         *dockertest.Network // Сеть где будут запускаться контейнеры.
	timeoutInSecond = uint(10 * 60)     // 10 минут
)

// createNetwork создает общую сеть для сервисов.
func createNetwork(pool *dockertest.Pool) error {
	var err error

	ns, err := pool.NetworksByName(Cfg.NetworkName)
	if err != nil {
		return fmt.Errorf("поиск сети %s: %w", Cfg.NetworkName, err)
	}

	if len(ns) > 0 {
		network = &ns[0]
		return nil
	}

	network, err = pool.CreateNetwork(Cfg.NetworkName)
	if err != nil {
		return fmt.Errorf("создание сети %s: %w", Cfg.NetworkName, err)
	}

	return nil
}

// getOrRunWithOptions получает контейнер с указанным именем или запускает новый, если не найден.
func getOrRunWithOptions(pool *dockertest.Pool, options *dockertest.RunOptions) (*dockertest.Resource, error) {
	// Проверяем наличие контейнера с указанным именем
	resource, exists := pool.ContainerByName(options.Name)
	if exists {
		Logrus.Infof("Контейнер %s уже запущен", options.Name)
		return resource, nil
	}

	// Если контейнер не найден, запускаем новый
	resource, err := pool.RunWithOptions(options, func(hostConfig *docker.HostConfig) {
		hostConfig.AutoRemove = true
	})
	if err != nil {
		return nil, err
	}

	_ = resource.Expire(timeoutInSecond)

	Logrus.Infof("Запущен контейнер %s", options.Name)

	return resource, nil
}

// safeResources хранит список ресурсов и обеспечивает безопасный доступ к ним.
type safeResources struct {
	resources []*dockertest.Resource
	mu        sync.Mutex
}

// add добавляет ресурс в список.
func (r *safeResources) add(resource *dockertest.Resource) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.resources = append(r.resources, resource)
}

// getCopy возвращает копию списка ресурсов.
func (r *safeResources) getCopy() []*dockertest.Resource {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]*dockertest.Resource{}, r.resources...)
}

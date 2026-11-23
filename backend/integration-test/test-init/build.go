//go:build integration

package test_init

import (
	"fmt"
	"net/http"

	"github.com/ory/dockertest/v3"
)

// initAppService запускает приложение в контейнере Docker.
func initAppService(pool *dockertest.Pool) error {
	appResource, err := getOrRunWithOptions(pool, &dockertest.RunOptions{
		Name:       "techline-app-integration-" + Cfg.BranchName,
		Repository: "techline",
		Tag:        "latest",
		NetworkID:  network.Network.ID,
		Hostname:   Cfg.ServerHostname,
		Env: []string{
			fmt.Sprintf("SERVER_PORT=%d", Cfg.HTTPListenAddr),
			fmt.Sprintf("DATABASE_URL=%s", Cfg.DSURL),
			fmt.Sprintf("CACHE_ADDR=%s", Cfg.CacheAddr),
			fmt.Sprintf("LOG_LEVEL=%s", Cfg.LogLevel),
			fmt.Sprintf("GATEWAY_TOKEN=%s", Cfg.GatewayToken),
			fmt.Sprintf("JWT_SECRET=%s", Cfg.JWTSecret),
			fmt.Sprintf("ADMIN_PHONES=%s", Cfg.AdminPhones),
		},
	})
	if err != nil {
		return fmt.Errorf("запуск приложения: %w", err)
	}

	resources.add(appResource)

	_ = appResource.Expire(timeoutInSecond)

	if err = pool.Retry(func() error {
		healthPath := fmt.Sprintf("http://%s:%d/version", Cfg.ServerHostname, Cfg.HTTPListenAddr)

		response, err := http.Get(healthPath)
		if err != nil {
			return fmt.Errorf("запрос к приложению: %w", err)
		}

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("статус ответа: %s", response.Status)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("запуск приложения: %w", err)
	}

	return nil
}

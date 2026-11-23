//go:build integration

package test_init

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Драйверы golang-migrate регистрируются через side-effect импорт.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"

	"github.com/eskermese/machinery-catalog/backend/internal/config"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/postgres"
	"github.com/eskermese/machinery-catalog/backend/pkg/cache/redis"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// migrationsSource путь до миграций относительно рабочей директории тестов.
// Локально это корень репозитория, в контейнере — каталог, куда их кладет Dockerfile.
const migrationsSource = "file://../migrations"

// dbPool отдельный пул для проверок состояния БД прямо из тестов
// (например, наличия таблиц). DataStore пул наружу не отдает.
var dbPool *pgxpool.Pool

// QueryRow выполняет запрос к тестовой БД. Используется тестами для проверки схемы и данных.
func QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return dbPool.QueryRow(ctx, query, args...)
}

// Exec выполняет команду в тестовой БД. Нужен для посева данных, которые
// нельзя завести через API: курс валют приложение только читает, а класть
// его пишущим эндпоинтом ради теста значило бы подгонять сервис под тест.
func Exec(ctx context.Context, query string, args ...any) error {
	_, err := dbPool.Exec(ctx, query, args...)

	return err
}

// initDatabaseService поднимает PostgreSQL и подключает к нему DataStore.
func initDatabaseService(pool *dockertest.Pool) error {
	dbResource, err := getOrRunWithOptions(pool, &dockertest.RunOptions{
		Name:       "techline-postgres-integration-" + Cfg.BranchName,
		Repository: "postgres",
		Tag:        "17-alpine",
		Hostname:   Cfg.DSHostname,
		NetworkID:  network.Network.ID,
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", Cfg.PostgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", Cfg.PostgresPassword),
			fmt.Sprintf("POSTGRES_DB=%s", Cfg.PostgresDB),
		},
	})
	if err != nil {
		return fmt.Errorf("запуск postgres: %w", err)
	}

	resources.add(dbResource)

	err = pool.Retry(func() error {
		Logrus.WithFields(logger.Fields{
			"database_url": Cfg.DSURL,
		}).Info("Подключение к postgres")

		DataStore = postgres.New(&config.Database{
			DSURL:         Cfg.DSURL,
			MaxConns:      Cfg.MaxConns,
			MinConns:      Cfg.MinConns,
			ConnectTimout: Cfg.ConnectTimout,
		})

		return DataStore.Connect()
	})
	if err != nil {
		return fmt.Errorf("подключение к postgres: %w", err)
	}

	if err = migrateUp(); err != nil {
		return fmt.Errorf("миграции postgres: %w", err)
	}

	dbPool, err = pgxpool.New(context.Background(), Cfg.DSURL)
	if err != nil {
		return fmt.Errorf("создание пула для проверок: %w", err)
	}

	return nil
}

// migrateUp накатывает схему на поднятый PostgreSQL. Без этого репозитории
// работают по несуществующим таблицам — Mongo схемы не требовал, PostgreSQL требует.
func migrateUp() error {
	m, err := migrate.New(migrationsSource, Cfg.DSURL)
	if err != nil {
		return fmt.Errorf("создание мигратора: %w", err)
	}

	defer func() {
		if sErr, dErr := m.Close(); sErr != nil || dErr != nil {
			Logrus.Warnf("закрытие мигратора: source err - %s, database err - %s", sErr, dErr)
		}
	}()

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("накат миграций: %w", err)
	}

	Logrus.Info("Миграции применены")

	return nil
}

// initCacheService поднимает сервис кэша.
func initCacheService(pool *dockertest.Pool) error {
	cacheResource, err := getOrRunWithOptions(pool, &dockertest.RunOptions{
		Name:       "techline-cache-integration-" + Cfg.BranchName,
		Repository: "docker.dragonflydb.io/dragonflydb/dragonfly",
		Tag:        "v1.33.0",
		Hostname:   Cfg.CacheHostname,
		NetworkID:  network.Network.ID,
		Env: []string{
			"DFLY_proactor_threads=4",
		},
	})
	if err != nil {
		return fmt.Errorf("запуск dragonfly: %w", err)
	}

	resources.add(cacheResource)

	err = pool.Retry(func() error {
		Cache = redis.New(&redis.Config{Addr: Cfg.CacheAddr})

		if _, cErr := Cache.Exists(context.Background(), "test-key"); cErr != nil {
			return cErr
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("подключение к dragonfly: %w", err)
	}

	return nil
}

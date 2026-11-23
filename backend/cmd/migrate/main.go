package main

import (
	"errors"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/eskermese/machinery-catalog/backend/internal/config"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

const (
	_defaultAttempts = 20          // количество попыток подключения к базе данных
	_defaultTimeout  = time.Second // таймаут между попытками подключения к базе данных
)

func main() {
	log := logger.NewZeroLogger(logger.Info, "", "")

	cfg, err := config.NewConfig()
	if err != nil {
		log.Warnf("Migrate: ошибка конфигурации: %s", err)

		return
	}

	var (
		attempts = _defaultAttempts
		m        *migrate.Migrate
	)

	for attempts > 0 {
		m, err = migrate.New("file://migrations", cfg.DSURL)
		if err == nil {
			break
		}

		log.Infof("Migrate: подключение к базе данных, осталось попыток: %d", attempts)
		time.Sleep(_defaultTimeout)

		attempts--
	}

	if err != nil {
		log.Warnf("Migrate: ошибка подключения к базе данных: %s", err)

		return
	}

	defer func() {
		if sErr, dErr := m.Close(); sErr != nil || dErr != nil {
			log.Warnf("Migrate: ошибка закрытия соединения с базой данных: source err - %s, database err - %s", sErr, dErr)
		}
	}()

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Warn("Migrate: миграция не требуется")

			return
		}

		log.Warnf("Migrate: ошибка миграции: %s", err)

		return
	}

	log.Info("Migrate: миграция выполнена успешно")
}

package main

import (
	"os"

	"github.com/eskermese/machinery-catalog/backend/internal/app"
	"github.com/eskermese/machinery-catalog/backend/internal/config"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
	docs "github.com/eskermese/machinery-catalog/backend/swagger"
)

//	@title			Techline API
//	@version		1.0
//	@description	Backend каталога спецтехники: карточки с ценами в USD, пересчёт в тенге по курсу с наценкой, калькулятор лизинга, коммерческие предложения и заявки

//	@host		localhost:8000
//	@BasePath	/api

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Токен доступа в формате "Bearer <token>". Выпускается ручками /v1/auth/*/token

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {
	log := logger.NewZeroLogger(logger.Info, "", "")

	// Конфигурация приложения
	cfg, err := config.NewConfig()
	if err != nil {
		log.Warnf("Ошибка инициализации конфигурации: %s", err)
		os.Exit(1)
	}

	// Документация Swagger
	docs.SwaggerInfo.Host = cfg.Host

	// Запуск приложения
	err = app.Run(cfg)
	if err != nil {
		// позволяет дать знать кубернетесу, что процесс завершился с ошибкой
		log.Warnf("ошибка при запуске: %s", err)
		os.Exit(1)
	}
}

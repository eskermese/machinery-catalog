.PHONY: tidy lint test swag bin-deps fmt migrate-up migrate-down migrate-down-all migrate-new compose compose-down integration mock all

LOG_LEVEL ?= debug
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/techline?sslmode=disable

lint:
	golangci-lint run -v

tidy:
	go mod tidy
	go mod vendor

test:
	go test -cover -race ./internal/... ./pkg/...

swag:
	swag fmt
	swag init -g cmd/app/main.go -o ./swagger/ --parseVendor --exclude ./vendor

bin-deps:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest

fmt:
	gofumpt -w .

# Запуск миграций
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

# Откатить последнюю миграцию
migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

# Откатить все миграции
migrate-down-all:
	migrate -path ./migrations -database "$(DATABASE_URL)" down -all

# Создать новую миграцию
migrate-new:
	migrate -path ./migrations create -dir ./migrations -ext sql $(name)

compose:
	LOG_LEVEL=$(LOG_LEVEL) docker compose up --build -d postgres dragonfly service pyroscope

compose-debug:
	docker compose up --build -d postgres dragonfly pyroscope

compose-down:
	docker compose down --remove-orphans

integration:
	@if [ -z "$$BRANCH_NAME" ]; then \
		export "BRANCH_NAME=main"; \
	fi; \
	echo "BRANCH_NAME="$$BRANCH_NAME; \
	test_status=0 ;\
	docker compose --profile integration build || { echo "Build failed, aborting."; exit 1; }; \
	docker compose run --rm integration || test_status=$$? ;\
	docker compose --profile integration down --remove-orphans ;\
	echo "status="$$test_status; exit $$test_status ;\

mock:
	mockgen -source="internal/storage/repository/db_repo.go" -destination="internal/storage/repository/mock_repo/mock_db_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/cache_repo.go" -destination="internal/storage/repository/mock_repo/mock_cache_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/notifier.go" -destination="internal/storage/repository/mock_repo/mock_notifier.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/user_repo.go" -destination="internal/storage/repository/mock_repo/mock_user_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/settings_repo.go" -destination="internal/storage/repository/mock_repo/mock_settings_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/fx_repo.go" -destination="internal/storage/repository/mock_repo/mock_fx_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/catalog_repo.go" -destination="internal/storage/repository/mock_repo/mock_catalog_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/quote_repo.go" -destination="internal/storage/repository/mock_repo/mock_quote_repo.go" -package="mock_repo"
	mockgen -source="internal/storage/repository/media_repo.go" -destination="internal/storage/repository/mock_repo/mock_media_repo.go" -package="mock_repo"
	mockgen -source="pkg/metrics/types/types.go" -destination="pkg/metrics/mock_metrics/mock_types.go" -package="mock_metrics"
	mockgen -source="pkg/clients/http/settings.go" -destination="pkg/clients/http/mock_http/mock_settings.go" -package="mock_http"

all: fmt tidy swag mock lint test

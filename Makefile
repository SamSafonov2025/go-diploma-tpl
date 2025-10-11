# автоопределяем точку входа
APP_ENTRY := $(if $(wildcard cmd/gophermart/main.go),./cmd/gophermart,.)

# ----------------------------
# Config (override via CLI)
# ----------------------------
APP_RUN_ADDRESS        ?= :8080
APP_ACCRUAL_ADDRESS    ?= http://localhost:8082
APP_TOKEN_SIGN_KEY     ?= change-me

DB_NAME                ?= gothermart
DB_USER                ?= postgres
DB_PASS                ?= postgres
DB_HOST                ?= localhost
DB_PORT                ?= 5432
# Полная строка подключения; можно переопределить DATABASE_URI напрямую
DATABASE_URI           ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Путь к миграциям goose (у тебя они лежат в internal/storage/postgres/sql/migrations)
MIGRATIONS_DIR         ?= ./internal/storage/postgres/sql/migrations

# ----------------------------
# Phony
# ----------------------------
.PHONY: help run build t test mocks db-create db-drop db-reset db-uri db-migrate-up db-migrate-down docker-db docker-stop

# ----------------------------
# Help
# ----------------------------
help:
	@echo ""
	@echo "Targets:"
	@echo "  make run                - запустить сервис с переменными окружения"
	@echo "  make build              - собрать бинарник ./bin/gophermart"
	@echo "  make t (test)           - прогнать все тесты"
	@echo "  make mocks              - сгенерировать моки для storage.Storage"
	@echo "  make db-uri             - показать итоговый DATABASE_URI"
	@echo "  make db-create          - создать БД $(DB_NAME)"
	@echo "  make db-drop            - удалить БД $(DB_NAME)"
	@echo "  make db-reset           - пересоздать БД (drop + create)"
	@echo "  make db-migrate-up      - выполнить миграции goose (если нужно вне кода)"
	@echo "  make db-migrate-down    - откатить миграции goose (если нужно)"
	@echo "  make docker-db          - поднять Postgres в Docker (dev)"
	@echo "  make docker-stop        - остановить контейнер Postgres"
	@echo ""
	@echo "Переменные (override):"
	@echo "  APP_RUN_ADDRESS, APP_ACCRUAL_ADDRESS, APP_TOKEN_SIGN_KEY"
	@echo "  DB_NAME, DB_USER, DB_PASS, DB_HOST, DB_PORT, DATABASE_URI, MIGRATIONS_DIR"
	@echo ""

# ----------------------------
# App
# ----------------------------
run:
	@echo "Starting app on $(APP_RUN_ADDRESS)"
	RUN_ADDRESS="$(APP_RUN_ADDRESS)" \
	ACCRUAL_SYSTEM_ADDRESS="$(APP_ACCRUAL_ADDRESS)" \
	DATABASE_URI="$(DATABASE_URI)" \
	TOKEN_SIGN_KEY="$(APP_TOKEN_SIGN_KEY)" \
	go run $(APP_ENTRY)

build:
	@mkdir -p bin
	GO111MODULE=on CGO_ENABLED=0 go build -o bin/gophermart .

t test:
	RUN_ADDRESS="$(APP_RUN_ADDRESS)" \
	ACCRUAL_SYSTEM_ADDRESS="$(APP_ACCRUAL_ADDRESS)" \
	DATABASE_URI="$(DATABASE_URI)" \
	go test ./... -v

# ----------------------------
# Mocks (gomock)
# ----------------------------
mocks:
	@mkdir -p internal/test/mocks
	mockgen -source="internal/storage/interface.go" -destination="internal/test/mocks/storage_mock.go" -package=mocks
	@echo "Mocks generated at internal/test/mocks/storage_mock.go"

# ----------------------------
# Database (psql/goose)
# ----------------------------
db-uri:
	@echo $(DATABASE_URI)

db-create:
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -c "CREATE DATABASE $(DB_NAME);" || true

db-drop:
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -c "DROP DATABASE IF EXISTS $(DB_NAME);"

db-reset: db-drop db-create

# Миграции через goose: обычно у тебя они запускаются в коде (embed + goose.Up),
# но оставим цели на случай ручного прогона.
db-migrate-up:
	goose -dir "$(MIGRATIONS_DIR)" postgres "$(DATABASE_URI)" up

db-migrate-down:
	goose -dir "$(MIGRATIONS_DIR)" postgres "$(DATABASE_URI)" down

# ----------------------------
# Docker helper (dev only)
# ----------------------------
docker-db:
	@docker run --name gophermart-pg -e POSTGRES_PASSWORD=$(DB_PASS) -e POSTGRES_USER=$(DB_USER) \
		-p $(DB_PORT):5432 -d postgres:14
	@echo "Waiting for Postgres..."
	@sleep 3
	@psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -c "CREATE DATABASE $(DB_NAME);" || true

docker-stop:
	-@docker rm -f gophermart-pg 2>/dev/null || true

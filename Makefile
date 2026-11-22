include .env
export

SERVER_DIR=cmd/shortener
BINARY_NAME=shortener.exe
TESTER_NAME=shortenertest.exe
num=1

# Данные для сборки (ldflags)
BUILD_VERSION ?= dev
BUILD_DATE    := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
BUILD_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -ldflags "\
	-X 'main.buildVersion=$(BUILD_VERSION)' \
	-X 'main.buildDate=$(BUILD_DATE)' \
	-X 'main.buildCommit=$(BUILD_COMMIT)'"

.PHONY: build build-info run-shortener test-app test stress-test clean \
        goose-create goose-status goose-up goose-up-by-one goose-down goose-down-to goose-up-to

# ==============================================================================
# СБОРКА И ЗАПУСК
# ==============================================================================

# Сборка бинарного файла
build:
	go build -o $(SERVER_DIR)/$(BINARY_NAME) $(SERVER_DIR)/main.go

# Сборка с внедрением версии (dev), даты и последнего хэша коммита
build-info:
	go build $(LDFLAGS) -o $(SERVER_DIR)/$(BINARY_NAME) $(SERVER_DIR)/main.go

# Локальный запуск сервера с параметрами
run-shortener:
	go run ./cmd/shortener \
		-a $$SERVER_ADDRESS \
		-d $$DATABASE_DSN \
		-f $$FILE_STORAGE_PATH \
		--audit-file $$AUDIT_FILE

# Удаление скомпилированного бинарника
clean:
	rm -f $(SERVER_DIR)/$(BINARY_NAME)
	@echo "Cleared!"

# ==============================================================================
# ТЕСТЫ
# ==============================================================================

# Запуск юнит-тестов проекта
test-app:
	go test -v ./...

# Сборка и запуск автотестов практикума
test:
	$(MAKE) build
	./$(TESTER_NAME) \
		-test.v \
		-test.run=^TestIteration$(num)$$ \
		-binary-path=$(SERVER_DIR)/$(BINARY_NAME) \
		-source-path=.

# Нагрузочное тестирование с помощью k6
stress-test:
	k6 run -e SERVER_ADDRESS=$$SERVER_ADDRESS shorten.js

# ==============================================================================
# МИГРАЦИИ GOOSE
# ==============================================================================

# Создать новый SQL-файл миграции
goose-create:
	@if [ -z "$(name)" ]; then \
		echo "There is no name param. Example: make migrate-create name=name_of_migrate"; \
		exit 1; \
	fi;	\

	goose \
	create $(name) sql

# Показать текущий статус миграции
goose-status:
	goose \
	status

# Накатить все доступные миграции
goose-up:
	goose \
	up

# Накатить только одну следующую миграцию
goose-up-by-one:
	goose \
	up-by-one

# Откатить последнюю применённую миграцию
goose-down:
	goose \
	down

# Накатить миграции до указанной версии
goose-up-to:
	@if [ -z "$(id)" ]; then \
		echo "There is no migrate id. Example: make goose-down-to id=20170614145246"; \
		exit 1; \
	fi;	\

	goose \
	up-to $(id)

# Откатить миграции до указанной версии
goose-down-to: 
	@if [ -z "$(id)" ]; then \
		echo "There is no migrate id. Example: make goose-down-to id=20170614145246"; \
		exit 1; \
	fi; \

	goose \
	down-to $(id)
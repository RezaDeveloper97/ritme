# Ritme Go API — developer entry points. See CLAUDE.md.
#
# Later tasks add behaviour by adding the scripts/commands the placeholder targets
# call (scripts/schema-diff.sh, cmd/contract) — not by editing this file.

SHELL := /bin/bash
.DEFAULT_GOAL := help

# Optional local overrides (never committed).
-include .env
export

COMPOSE_TEST := docker compose -f docker-compose.test.yml

# Host ports of docker-compose.test.yml (uncommon on purpose; other stacks own 3306/3307/6379/6380).
TEST_DB_PORT         ?= 13317
TEST_REDIS_PORT      ?= 16380

# `make run` defaults: the test compose stack (make test-db-up). Override in .env.
APP_ENV              ?= local
APP_DEBUG            ?= true
HTTP_ADDR            ?= :8020
DB_HOST              ?= 127.0.0.1
DB_PORT              ?= $(TEST_DB_PORT)
DB_DATABASE          ?= ritme_test
DB_USERNAME          ?= ritme
DB_PASSWORD          ?= ritme
REDIS_HOST           ?= 127.0.0.1
REDIS_PORT           ?= $(TEST_REDIS_PORT)
STORAGE_PATH         ?= $(CURDIR)/../backend/storage

TEST_DB_DSN          ?= ritme:ritme@tcp(127.0.0.1:$(TEST_DB_PORT))/ritme_test?parseTime=true&loc=Asia%2FTehran&charset=utf8mb4&collation=utf8mb4_unicode_ci
TEST_REDIS_ADDR      ?= 127.0.0.1:$(TEST_REDIS_PORT)
# testdb creates one database per test package (internal/platform/db/testdb): needs CREATE DATABASE.
TEST_DB_ADMIN_DSN    ?= root:root@tcp(127.0.0.1:$(TEST_DB_PORT))/?parseTime=true&loc=Asia%2FTehran&charset=utf8mb4&collation=utf8mb4_unicode_ci

PKG ?= ./...

# sqlc from PATH, else $(go env GOPATH)/bin, else built on the fly (pinned; cached after the first run).
SQLC_VERSION ?= v1.31.1
SQLC ?= $(shell command -v sqlc 2>/dev/null || { p="$$(go env GOPATH)/bin/sqlc"; [ -x "$$p" ] && echo "$$p"; } || echo "go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)")

.PHONY: help run test test-int test-db-up test-db-down lint fmt vet sqlc schema-diff contract contract-record docker

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

run: ## Run the API locally (needs `make test-db-up` or DB_* / REDIS_* in .env)
	go run ./cmd/api

test: ## Unit tests (integration tests skip without TEST_DB_DSN)
	TEST_DB_DSN= TEST_REDIS_ADDR= go test ./...

test-db-up: ## Start MariaDB 11.4 + Redis for integration tests
	$(COMPOSE_TEST) up -d --wait

test-db-down: ## Stop and wipe the test databases
	$(COMPOSE_TEST) down -v

test-int: test-db-up ## Integration tests: make test-int PKG=./internal/content/...
	TEST_DB_DSN='$(TEST_DB_DSN)' TEST_DB_ADMIN_DSN='$(TEST_DB_ADMIN_DSN)' TEST_REDIS_ADDR='$(TEST_REDIS_ADDR)' go test -count=1 -p 1 $(PKG)

vet: ## go vet
	go vet ./...

lint: ## golangci-lint (v2)
	golangci-lint run

fmt: ## gofmt + goimports via golangci-lint
	gofmt -w .
	golangci-lint fmt

sqlc: ## Regenerate internal/<domain>/store from db/queries/<domain>/*.sql (sqlc.yaml)
	$(SQLC) generate

schema-diff: ## Laravel-migrated DB vs goose baseline DB must be identical (needs php + backend/vendor)
	scripts/schema-diff.sh

contract: ## Diff Go responses against Laravel goldens: make contract ROUTES=<pattern> (T-M2-05)
	@if [ -d cmd/contract ]; then go run ./cmd/contract diff $(if $(ROUTES),--routes '$(ROUTES)'); else echo "contract: not implemented yet (cmd/contract arrives with T-M2-05)"; fi

contract-record: ## Record Laravel goldens: make contract-record ROUTES=<pattern> (T-M2-05)
	@if [ -d cmd/contract ]; then go run ./cmd/contract record $(if $(ROUTES),--routes '$(ROUTES)'); else echo "contract-record: not implemented yet (cmd/contract arrives with T-M2-05)"; fi

docker: ## Build the image
	docker build -t ritme-backend-go:dev .

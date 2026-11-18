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

# cgo fallback (T-M12-11). Some macOS Command Line Tools ship an SDK whose .tbd stubs the system linker
# cannot read ("tapi error: malformed file … unknown architecture arm64e.x1"), so every test binary that
# links cgo (net, os/user) fails. CGO_PROBE prints CGO_ENABLED=0 when a tiny cgo program does not link,
# and nothing otherwise; an explicit CGO_ENABLED in the environment always wins. The real fix is a
# Command Line Tools update (`softwareupdate --list`, or xcode-select --install after removing the old one).
# -race needs cgo: on such a Mac use `make test-race-docker`.
define CGO_PROBE
$$(if [ -n "$${CGO_ENABLED+x}" ]; then :; else \
  d="$$(mktemp -d)"; printf 'package main\nimport "C"\nfunc main() {}\n' > "$$d/main.go"; \
  if ! (cd "$$d" && GO111MODULE=off go build -o /dev/null . >/dev/null 2>&1); then \
    echo 'CGO_ENABLED=0'; echo 'cgo does not link here: CGO_ENABLED=0 (see Makefile, CGO_PROBE)' >&2; fi; \
  rm -rf "$$d"; fi)
endef

# Race suite (T-M12-11). RACE_P = packages built/tested at once: testdb gives every package its own database
# and the Redis tests use random key prefixes, so packages may run in parallel.
RACE_P               ?= 4
RACE_TIMEOUT         ?= 30m
# test-race-docker: a Linux Go container on the test stack's network (cgo works there). Its own scratch
# database $(RACE_DB) for the few tests that use TEST_DB_DSN directly — never ritme_test, which
# `make test-int` runs share.
RACE_GO_IMAGE        ?= golang:1.25-bookworm
RACE_DB              ?= ritme_race
RACE_NETWORK         ?= ritme-go-test_default

# Benchmarks (T-M9-08): bench/run.sh reads BENCH_PKGS / BENCH_COUNT / BENCH_TIME. benchstat from PATH, else pinned.
BENCHSTAT_VERSION ?= v0.0.0-20261009192801-be2c69fb417e
BENCHSTAT ?= $(shell command -v benchstat 2>/dev/null || { p="$$(go env GOPATH)/bin/benchstat"; [ -x "$$p" ] && echo "$$p"; } || echo "go run golang.org/x/perf/cmd/benchstat@$(BENCHSTAT_VERSION)")

# sqlc from PATH, else $(go env GOPATH)/bin, else built on the fly (pinned; cached after the first run).
SQLC_VERSION ?= v1.31.1
SQLC ?= $(shell command -v sqlc 2>/dev/null || { p="$$(go env GOPATH)/bin/sqlc"; [ -x "$$p" ] && echo "$$p"; } || echo "go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)")

.PHONY: help run test test-int test-race test-race-docker test-db-up test-db-down lint fmt vet sqlc schema-diff contract contract-record contract-v6 contract-v6-update docker \
	bench bench-baseline bench-compare bench-gate

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

run: ## Run the API locally (needs `make test-db-up` or DB_* / REDIS_* in .env)
	go run ./cmd/api

test: ## Unit tests (integration tests skip without TEST_DB_DSN; falls back to CGO_ENABLED=0 if cgo cannot link)
	env $(CGO_PROBE) CI= TEST_DB_DSN= TEST_REDIS_ADDR= go test ./...

test-db-up: ## Start MariaDB 11.4 + Redis for integration tests
	$(COMPOSE_TEST) up -d --wait

test-db-down: ## Stop and wipe the test databases
	$(COMPOSE_TEST) down -v

test-int: test-db-up ## Integration tests: make test-int PKG=./internal/content/...
	env $(CGO_PROBE) TEST_DB_DSN='$(TEST_DB_DSN)' TEST_DB_ADMIN_DSN='$(TEST_DB_ADMIN_DSN)' TEST_REDIS_ADDR='$(TEST_REDIS_ADDR)' go test -count=1 -p 1 $(PKG)

test-race: ## Full suite with -race, integration included (needs cgo + TEST_DB_DSN/TEST_REDIS_ADDR; CI runs this)
	CGO_ENABLED=1 go test -race -count=1 -p $(RACE_P) -timeout $(RACE_TIMEOUT) $(PKG)

test-race-docker: test-db-up ## -race in a Linux container on the test stack (Mac without a cgo linker): make test-race-docker [PKG=…]
	$(COMPOSE_TEST) exec -T mariadb mariadb -uroot -proot -e "CREATE DATABASE IF NOT EXISTS \`$(RACE_DB)\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; GRANT ALL ON \`$(RACE_DB)\`.* TO 'ritme'@'%';"
	docker run --rm --network '$(RACE_NETWORK)' \
	  -v '$(abspath $(CURDIR)/..)':/repo -w /repo/backend-go \
	  -v "$$(go env GOMODCACHE)":/go/pkg/mod \
	  -v ritme-go-race-buildcache:/root/.cache/go-build \
	  -e GOTOOLCHAIN=local -e GOFLAGS=-modcacherw -e CI= \
	  -e TEST_DB_DSN='ritme:ritme@tcp(mariadb:3306)/$(RACE_DB)?parseTime=true&loc=Asia%2FTehran&charset=utf8mb4&collation=utf8mb4_unicode_ci' \
	  -e TEST_DB_ADMIN_DSN='root:root@tcp(mariadb:3306)/?parseTime=true&loc=Asia%2FTehran&charset=utf8mb4&collation=utf8mb4_unicode_ci' \
	  -e TEST_REDIS_ADDR='redis:6379' \
	  '$(RACE_GO_IMAGE)' \
	  go test -race -count=1 -p $(RACE_P) -timeout $(RACE_TIMEOUT) $(PKG)

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

contract: ## Diff a fresh Go server against the Laravel goldens: make contract ROUTES=<group[,group…]|all> (exit 1 on diff)
	go run ./cmd/contract diff --routes '$(or $(ROUTES),all)'

contract-record: ## Record Laravel goldens (contract stack; CONTRACT_PORT, default 8090): make contract-record ROUTES=<group|all>
	go run ./cmd/contract record --routes '$(or $(ROUTES),all)'

contract-v6: ## Diff Go (CYCLE_ENGINE=v6) against contract/golden-v6 (cycle today/date cases): make contract-v6 [ROUTES=cycle]
	go run ./cmd/contract v6 --routes '$(or $(ROUTES),cycle)'

contract-v6-update: ## Rewrite contract/golden-v6 from Go (CYCLE_ENGINE=v6); review the diff by hand
	go run ./cmd/contract v6 --update --routes '$(or $(ROUTES),cycle)'

docker: ## Build the image
	docker build -t ritme-backend-go:dev .

bench: ## Benchmarks → bench/current.txt (BENCH_COUNT=6 BENCH_TIME=1s; header = machine + Go)
	bench/run.sh bench/current.txt

bench-baseline: ## Re-record bench/baseline.txt (quiet machine; commit it with the change that moved the numbers)
	bench/run.sh bench/baseline.txt

bench-compare: ## benchstat bench/baseline.txt vs bench/current.txt, then the CI gate
	$(BENCHSTAT) bench/baseline.txt bench/current.txt
	go run ./bench/gate -baseline bench/baseline.txt -current bench/current.txt

bench-gate: ## Only the CI gate: fail on an allocs/op regression, warn on sec/op > 20 % (X-09)
	go run ./bench/gate -baseline bench/baseline.txt -current bench/current.txt

# Load tests (T-M9-14). LOCAL ONLY: scripts/seed refuse non-local targets. Needs `make test-db-up`, `make loadtest-db`,
# `make loadtest-seed`, and the API running on :8020 against LOADTEST_DB (`make loadtest-api`, other terminal).
# SCENARIO = app_open | calendar_browse | log_burst | analysis_tab | fertility_tab | soak.  Recipe: header of loadtest/run.sh.
LOADTEST_DB        ?= ritme_loadtest
LOADTEST_USERS     ?= 2000
LOADTEST_DSN       ?= ritme:ritme@tcp(127.0.0.1:$(TEST_DB_PORT))/$(LOADTEST_DB)
SCENARIO           ?= app_open
K6_RUNNER          ?= docker
K6_IMAGE           ?= grafana/k6:1.3.0
.PHONY: loadtest loadtest-db loadtest-seed loadtest-api loadtest-drop loadtest-lint

loadtest-db: ## Create the scratch DB $(LOADTEST_DB) on the test stack (never ritme_test)
	$(COMPOSE_TEST) exec -T mariadb mariadb -uroot -proot -e "CREATE DATABASE IF NOT EXISTS \`$(LOADTEST_DB)\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; GRANT ALL ON \`$(LOADTEST_DB)\`.* TO 'ritme'@'%';"

loadtest-seed: ## Seed LOADTEST_USERS synthetic users + loadtest/.tokens.json (idempotent; RESET=1 starts over)
	go run ./loadtest/seed -dsn '$(LOADTEST_DSN)' -users $(LOADTEST_USERS) -tokens-out loadtest/.tokens.json $(if $(RESET),-reset)

loadtest-api: ## Run the API against the scratch DB (API on :8020, /metrics on 127.0.0.1:9090)
	DB_DATABASE=$(LOADTEST_DB) SMS_PROVIDER=log go run ./cmd/api

loadtest: ## Run a k6 scenario: make loadtest SCENARIO=app_open [RPS=200 DURATION=2m K6_RUNNER=local]
	K6_RUNNER='$(K6_RUNNER)' K6_IMAGE='$(K6_IMAGE)' loadtest/run.sh '$(SCENARIO)'

loadtest-lint: ## k6 inspect every script (pinned docker image, or K6_RUNNER=local)
	K6_RUNNER='$(K6_RUNNER)' K6_IMAGE='$(K6_IMAGE)' loadtest/run.sh --inspect

loadtest-drop: ## Drop the scratch DB $(LOADTEST_DB)
	$(COMPOSE_TEST) exec -T mariadb mariadb -uroot -proot -e "DROP DATABASE IF EXISTS \`$(LOADTEST_DB)\`"

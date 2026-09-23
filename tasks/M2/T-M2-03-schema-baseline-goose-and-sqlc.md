---
id: T-M2-03
title: Schema baseline with goose and sqlc setup
milestone: M2
type: backend
status: done
depends_on: [T-M2-02]
parallel_group: M2-B
touches: [backend-go/db, backend-go/scripts/schema-diff.sh, backend-go/sqlc.yaml, backend-go/internal/platform/db/migrate.go, backend-go/internal/platform/db/testdb, docs/go-migration/migrations.md]
skills: []
verify: cd backend-go && make sqlc && make test-int PKG=./internal/platform/db/... && make schema-diff
---

# T-M2-03 — Schema baseline with goose and sqlc setup

## Why
Go reuses the production MariaDB with zero data migration (README → DB). sqlc needs the exact schema, and goose must
take over migrations after cutover without re-running anything. Laravel keeps owning migrations until T-M2-27 (schema
frozen during M2). See domain-inventory §1.

## Scope
1. Dump the current schema from **staging** (`mysqldump --no-data --skip-comments`, read-only, no data). Commit it as
   `db/migrations/00001_baseline.sql` (goose `-- +goose Up`; Down documented as unsupported). Include the fa/en
   `languages` rows the Laravel migration inserts (`INSERT IGNORE`).
2. goose runner `internal/platform/db/migrate.go`: used **only** by tests and fresh environments for now; the prod/stage
   entrypoint must not run it until T-M2-27. Document how an existing DB gets version-stamped at cutover.
3. `sqlc.yaml`: MySQL engine, schema = `db/migrations`, **one package per domain declared up front** so parallel tasks
   only add `.sql` files under `db/queries/<domain>/`: auth, content, profile, healthlog, reminder, cycle, pregnancy,
   messages, home, admin, i18n (output `internal/<domain>/store`). Overrides: JSON columns → `json.RawMessage`,
   decimals → string, `date` → `civildate.Date` (coordinate with T-M2-04; fall back to `time.Time` + adapter if
   T-M2-04 isn't merged yet), tinyint(1) → bool.
4. `internal/platform/db/testdb`: creates a fresh schema per test package in the test MariaDB, applies the baseline,
   truncates between tests; optional loader for the contract fixture dump (T-M2-05).
5. `docs/go-migration/migrations.md`: ownership rule (Laravel until cutover; any schema change = Laravel migration +
   matching goose migration in the same commit), and `make schema-diff` (Laravel fresh DB vs goose baseline DB).

## Out of scope
Domain queries (each domain task adds its own). Running goose on stage/prod.

## Acceptance
- `make schema-diff` shows zero difference between a Laravel-migrated DB and a goose-baseline DB (ignoring the
  `migrations` / `goose_db_version` tables).
- `make sqlc` generates without errors; a sample query per package compiles.
- The integration test applies the baseline to a fresh MariaDB 11.4 container.

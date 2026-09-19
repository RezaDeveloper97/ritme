---
id: T-M1-12
title: Backend performance optimisation (queries, indexes, caching)
milestone: M1
type: backend
status: todo
depends_on: [T-M1-10]
parallel_group: M1-E
touches: [backend/app/Services, backend/app/Repositories, backend/app/Http/Resources, backend/database/migrations, backend/tests, backend/Dockerfile]
skills: [new-endpoint]
verify: cd backend && vendor/bin/pint --test && php artisan test
---

# T-M1-12 — Backend performance optimisation (queries, indexes, caching)

## Scope
Implement the backend items from `docs/investigations/perf-baseline.md`, typically: eager loading to kill N+1s,
indexes for the hot `WHERE`/`ORDER BY` columns (new migrations only — never edit old ones; must work on MariaDB
**and** SQLite), per-user caching of expensive cycle-engine results with correct invalidation on every write that
affects them, prod image with `config:cache`/`route:cache`/`event:cache` and opcache tuned.

## Acceptance
- Query count per hot endpoint asserted in a feature test (so N+1s can't come back).
- Before/after p95 appended to `docs/investigations/perf-baseline.md`.
- `pint --test` + `php artisan test` green (mind memory `ritme-backend-testing-gotchas`).

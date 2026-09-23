---
name: verify-all
description: Run the full Ritme verification suite — backend-go vet+tests+lint (the backend), frontend typecheck+lint+FSD rules+style gate+dark-mode gate+unit tests, and the legacy Laravel checks only when backend/ changed — and report a pass/fail table. Use before commits and always before deploy.
---

# Verify everything

Run these (parallelize backend-go and frontend):

## Backend (Go) — `backend-go/`, the backend for all new work
```bash
cd backend-go && go vet ./... && go test ./... && golangci-lint run
```

`golangci-lint` v2 is required (`brew install golangci-lint`). Integration tests skip without
`TEST_DB_DSN`; when the change touches SQL, a store or a service with DB access, also run
`make test-int PKG=./internal/<domain>/...` (starts the docker test stack). Also, when relevant:
- a route's behaviour changed → `make contract ROUTES=<group>` (must be green, see `backend-go/README.md`);
- a route or `api/openapi.yaml` changed → covered by `go test` (`internal/http` OpenAPI test);
- a migration changed (goose or Laravel) → `make schema-diff`;
- `db/queries/**` changed → `make sqlc` and check the generated diff is committed.

If this is pre-deploy, also run `make docker` in `backend-go` (builds `ritme-backend-go:dev`).

## Frontend
```bash
cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
```

`lint:dark` enforces CLAUDE.md §10.3: every colour token has a value in both
themes, dark never reads worse than light, and the pre-paint bootstrap agrees
with the theme store. Its `⚠` block lists pairs already under WCAG AA in light
mode — those are pre-existing palette decisions, not failures.

`lint:styles` enforces CLAUDE.md §10.1 (no static `style` props, no hex colour
literals, no `var(--x)` for an undeclared x). It is a ratchet against
`frontend/scripts/styles-baseline.json`, so it only fails on a *regression*. If
a file legitimately improved, run `npm run lint:styles:accept` to re-baseline —
never re-baseline to silence a genuine new violation.

If this is pre-deploy, also run `npm run build` in frontend — the production build catches errors dev mode doesn't, and a failed build wastes a full deploy cycle.

## Backend (Laravel) — legacy, production only until cutover
Run **only when the change touches `backend/`** (production fixes, or the mirror Laravel migration of a Go schema
change); otherwise report the row as "skipped (backend/ unchanged)".
```bash
cd backend && vendor/bin/pint --test && php artisan test
```
`pint --test` has been red on pre-existing files since M1 — report only regressions in files you changed
(`vendor/bin/pint --test <files>`).

## Reporting
Report a short pass/fail line per command. For failures, give the root cause with `file:line` and say whether it's from the current change or pre-existing. Fix failures caused by the current change; flag pre-existing ones to the user instead of silently fixing.

Known failure causes to check first:
- Go: `go.mod`/`go.sum` changed under you by a parallel agent ("missing go.sum entry") → `go get <module>@<version>`,
  never `go mod tidy` (see `backend-go/CLAUDE.md`)
- Go: integration tests silently skipped because `TEST_DB_DSN` is unset — use `make test-int`
- Missing fa/en message keys (next-intl) causing lint/test failures
- Laravel (legacy): SQLite date-cast + `updateOrCreate` mismatch; `Passport::actingAs` stale relation cache
  (needs `$user->refresh()`)

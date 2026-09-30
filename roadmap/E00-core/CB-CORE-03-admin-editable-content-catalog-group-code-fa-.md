---
id: CB-CORE-03
title: Admin-editable content catalog (group / code / fa+en / meta)
epic: CORE
type: backend
status: todo
depends_on: [CB-CORE-01]
parallel_group: CORE-B
touches: [backend-go/internal/catalog, backend-go/internal/http/routes_catalog.go, backend-go/db/queries/catalog, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md, backend-go/internal/http/routes_admin_catalog.go]
skills: [new-endpoint]
boards: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/catalog/... && make test-int PKG=./internal/catalog/... && golangci-lint run && make schema-diff
---

# CB-CORE-03 — Admin-editable content catalog (group / code / fa+en / meta)

## Why
Warning signs, missed-pill steps, FAQs, kit items, score items, layette templates… must be admin-editable (DECISIONS) without one table per list. bloom has no generic catalog (its log taxonomy covers only log items).

## Scope
1. `catalog_items` (group, code, sort, active, mode/audience filter, title/body fa+en, meta JSON) + a seed convention each epic appends to.
2. `GET /api/v1/catalog/{group}` (locale-aware, cached) and admin CRUD `/api/admin/v1/catalog/{group}`.
3. docs/canvas-build/catalog.md: when to use the catalog vs bloom's log taxonomy (B-N3-01) vs message_contents.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Groups readable fa/en; admin CRUD tested; doc written.
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.

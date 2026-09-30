---
id: CB-NAV-01
title: Global search API
epic: NAV
type: backend
status: todo
depends_on: [CB-CORE-01, B-N3-01]
parallel_group: NAV-A
touches: [backend-go/internal/search, backend-go/internal/http/routes_search.go, backend-go/db/queries/search, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations]
skills: [new-endpoint]
boards: [nbd_Nav_Search.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/search/... && make test-int PKG=./internal/search/... && golangci-lint run
---

# CB-NAV-01 — Global search API

## Why
bloom has no global search; the canvas puts one in the Today header.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Nav_Search.dc.html`

## Scope
1. `GET /api/v1/search?q=&scope=all|mine|education|programs|services` → grouped hits: my-log insights (e.g. 'N days with pain this cycle' → analysis), articles/courses (when N8 exists), care programs, city services/directory places, each with a route. Shop excluded (own search).
2. Persian normalisation (ی/ي، ک/ك، ZWNJ, digits); per-group limits; no raw health values in logs.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Grouped results; Persian normalisation tests.
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.

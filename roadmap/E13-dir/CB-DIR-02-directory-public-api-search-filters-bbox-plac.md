---
id: CB-DIR-02
title: Directory public API: search, filters, bbox, place, slots
epic: DIR
type: backend
status: todo
depends_on: [CB-DIR-01]
parallel_group: DIR-B
touches: [backend-go/internal/cityservices, backend-go/internal/http/routes_cityservices.go, backend-go/db/queries/cityservices, backend-go/api/openapi.yaml, backend-go/contract]
skills: [new-endpoint]
boards: [nbl_Dir_Home.dc.html, nbl_Dir_List.dc.html, nbl_Dir_Map.dc.html, nbl_Dir_Place.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/cityservices/... && make test-int PKG=./internal/cityservices/... && golangci-lint run
---

# CB-DIR-02 — Directory public API: search, filters, bbox, place, slots

## Why
Browse without login (web SEO) and in the app.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Home.dc.html`
- `nbl_Dir_List.dc.html`
- `nbl_Dir_Map.dc.html`
- `nbl_Dir_Place.dc.html`

## Scope
1. List with q, category, child age (months), open-now, amenities, lat/lng radius + distance sort, bbox; ranking = distance + age fit + rating, never paid. Place detail with services, next slots, reviews summary. Public, cached, rate-limited.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.

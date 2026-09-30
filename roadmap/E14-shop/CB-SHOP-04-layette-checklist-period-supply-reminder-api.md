---
id: CB-SHOP-04
title: Layette checklist + period-supply reminder API
epic: SHOP
type: backend
status: todo
depends_on: [CB-SHOP-01]
parallel_group: SHOP-C
touches: [backend-go/internal/shop/checklist, backend-go/internal/http/routes_shop.go, backend-go/db/queries/shop, backend-go/api/openapi.yaml]
skills: [new-endpoint]
boards: [nbd_Shop_Checklist.dc.html, nbd_Shop_Beauty.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/shop/... && make test-int PKG=./internal/shop/... && golangci-lint run
---

# CB-SHOP-04 — Layette checklist + period-supply reminder API

## Why
Her own list, not a sales funnel; supply reminder off by default.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Checklist.dc.html`
- `nbd_Shop_Beauty.dc.html`

## Scope
1. Checklist from template (groups), states have/in cart/in transit/not needed/later, custom items, family share (bloom companion, read-only), due date from pregnancy.
2. Supply reminder opt-in: uses only the predicted next-period date → care reminder 3 days before; nothing to sellers.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Reminder never created when off (test).
- `verify` green.

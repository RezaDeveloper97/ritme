---
id: CB-SHOP-03
title: Cart, checkout, COD orders, returns API
epic: SHOP
type: backend
status: todo
depends_on: [CB-SHOP-02]
parallel_group: SHOP-C
touches: [backend-go/internal/shop/order, backend-go/internal/http/routes_shop.go, backend-go/db/queries/shop, backend-go/api/openapi.yaml, backend-go/contract]
skills: [new-endpoint]
boards: [nbd_Shop_Cart.dc.html, nbd_Shop_Checkout.dc.html, nbd_Shop_Order.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/shop/... && make test-int PKG=./internal/shop/... && golangci-lint run
---

# CB-SHOP-03 — Cart, checkout, COD orders, returns API

## Why
Multi-seller cart and pay-on-delivery orders (bloom's payment adapter B-N2-05 NOT used — COD only).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Cart.dc.html`
- `nbd_Shop_Checkout.dc.html`
- `nbd_Shop_Order.dc.html`

## Scope
1. Server cart, discount codes (admin), addresses, per-seller slots, place order (stock reservation, per-seller shipments), order list/detail/timeline, cancel before confirm, return within 7 days of delivery; order → checklist 'in transit' → delivered 'have'; SMS via adapter.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Stock race test; COD-only enforced.
- `verify` green.

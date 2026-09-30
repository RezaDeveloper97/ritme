---
id: CB-SHOP-02
title: Shop catalog API
epic: SHOP
type: backend
status: todo
depends_on: [CB-SHOP-01]
parallel_group: SHOP-B
touches: [backend-go/internal/shop, backend-go/internal/http/routes_shop.go, backend-go/db/queries/shop, backend-go/api/openapi.yaml, backend-go/contract]
skills: [new-endpoint]
boards: [nbd_Shop_Home.dc.html, nbd_Shop_Beauty.dc.html, nbd_Shop_List.dc.html, nbd_Shop_Product.dc.html, nbd_Shop_ProductBeauty.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/shop/... && make test-int PKG=./internal/shop/... && golangci-lint run
---

# CB-SHOP-02 — Shop catalog API

## Why
Browse both shops.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Home.dc.html`
- `nbd_Shop_Beauty.dc.html`
- `nbd_Shop_List.dc.html`
- `nbd_Shop_Product.dc.html`
- `nbd_Shop_ProductBeauty.dc.html`

## Scope
1. Home sections, list filters (size, material, color, price, skin type) + sort, product detail with variants/stock/reviews, bought-together. Teen → 403.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.

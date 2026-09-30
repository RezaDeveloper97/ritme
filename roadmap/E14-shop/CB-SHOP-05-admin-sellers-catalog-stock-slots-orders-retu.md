---
id: CB-SHOP-05
title: Admin: sellers, catalog, stock, slots, orders, returns, reviews, layette template
epic: SHOP
type: fullstack
status: todo
depends_on: [CB-SHOP-03, CB-SHOP-04, CB-CORE-04]
parallel_group: SHOP-D
touches: [backend-go/internal/shop/admin, backend-go/internal/http/routes_admin_shop.go, backend-go/api/openapi.yaml, admin-web/src/screens/shop, admin-web/src/app/(panel)/shop, admin-web/src/widgets/shell/model/nav.ts, admin-web/messages]
skills: [new-endpoint]
boards: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/shop/... && make test-int PKG=./internal/shop/... && golangci-lint run && cd ../admin-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# CB-SHOP-05 — Admin: sellers, catalog, stock, slots, orders, returns, reviews, layette template

## Why
The team runs the shop in admin-web (extends bloom's shop admin).

## Scope
1. Sellers, categories, products + variants + images + IRC, stock, delivery-slot templates, orders queue per shipment (confirm → shipped → delivered / cancel), returns, reviews, discount codes, layette template.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Order lifecycle end to end from admin.
- `verify` green.

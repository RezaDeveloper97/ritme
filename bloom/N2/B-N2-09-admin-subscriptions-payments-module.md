---
id: B-N2-09
title: Admin — subscriptions & payments module
milestone: N2
type: fullstack
status: todo
depends_on: [B-N2-04]
parallel_group: N2-I
touches: [admin-web/src,backend-go/internal/admin,backend-go/internal/plus,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N2-09 — Admin — subscriptions & payments module

## Why
Admin sidebar «اشتراک‌ها و پرداخت».

## Scope
- Plans CRUD, discount codes CRUD, trial offer % and VAT config, subscriptions list/filter, payment log, manual refund/extend.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Admin endpoints role-protected; admin-web build green
- `verify` green

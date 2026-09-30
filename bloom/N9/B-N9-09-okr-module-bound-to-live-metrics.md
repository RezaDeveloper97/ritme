---
id: B-N9-09
title: OKR module bound to live metrics
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-08]
parallel_group: N9-I
touches: [backend-go/internal/okr,backend-go/db,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-09 — OKR module bound to live metrics

## Why
OKRs auto-update.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_OKR.dc.html` (+ `nbd_Admin_OKR`)

## Scope
- Objectives + KRs each bound to a metric key with baseline/target, auto progress + status (on track / at risk / done), owners, quarters.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Status computation tests
- `verify` green

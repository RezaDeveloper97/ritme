---
id: B-N9-03
title: Overview dashboard and metrics aggregation
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-02]
parallel_group: N9-C
touches: [backend-go/internal/metrics,backend-go/internal/admin,backend-go/db,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-03 — Overview dashboard and metrics aggregation

## Why
Admin home.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Overview.dc.html` (+ `nbd_Admin_Overview`)

## Scope
- Nightly/hourly aggregation: DAU, MAU, stickiness, signups, Plus subs, trial→paid, MRR, users by mode, engine health (mean period-start error, ±2-day accuracy, confidence distribution), active alerts, OKR summary.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Aggregation tests on seeded events
- `verify` green

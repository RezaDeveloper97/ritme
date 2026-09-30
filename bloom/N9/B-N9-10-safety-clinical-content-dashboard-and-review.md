---
id: B-N9-10
title: Safety & clinical content dashboard and review queue
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-05]
parallel_group: N9-J
touches: [backend-go/internal/safety,backend-go/internal/admin,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-10 — Safety & clinical content dashboard and review queue

## Why
Aggregated safety signals only, no user ids.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Safety.dc.html` (+ `nbd_Admin_Safety`)

## Scope
- Signal counts (BP crisis, glucose <54, EPDS Q10, reduced kicks, 5-1-1 contractions, heavy postpartum bleeding) 7d + delta; review queue (clinical messages, thresholds, lessons, legal texts) with approvals; urgent-path health checks (modal latency p95, 115 CTR, approved-text parity, nightly scenario tests).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Nightly scenario test job exists and runs in CI
- `verify` green

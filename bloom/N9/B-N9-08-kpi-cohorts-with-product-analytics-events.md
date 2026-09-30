---
id: B-N9-08
title: KPI & cohorts with product analytics events
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-03]
parallel_group: N9-H
touches: [backend-go/internal/metrics,backend-go/internal/analyticsevents,backend-go/db,backend-go/api,admin-web/src,frontend/src/shared/analytics]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-08 — KPI & cohorts with product analytics events

## Why
Growth/retention/engine/AI/telemed/learning KPIs.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_KPI.dc.html` (+ `nbd_Admin_KPI`)

## Scope
- First-party event tracking (no third-party SDK, privacy-safe), activation funnel, D0–D60 cohort retention, prediction accuracy, data-quality distribution, AI metrics (labs/day, OCR correction rate, cost, assistant msgs, referral rate), tabs.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Events carry no health payload (test)
- `verify` green

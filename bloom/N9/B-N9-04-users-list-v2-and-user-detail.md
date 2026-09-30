---
id: B-N9-04
title: Users list v2 and user detail
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-02,B-N9-03]
parallel_group: N9-D
touches: [backend-go/internal/admin,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-04 — Users list v2 and user detail

## Why
Better support tooling without exposing health data.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Users.dc.html` (+ `nbd_Admin_Users`)
- `docs/design/night-bloom/h-admin/nbl_Admin_UserDetail.dc.html` (+ `nbd_Admin_UserDetail`)

## Scope
- Filters (mode, plan, activity, data quality, confidence, engine warning, cohort, churn risk), saved segments, anonymised export, masked phones. Detail tabs: summary, engine output today, events, messages & challenges, subscription, consents, support; health data hidden unless temporary access granted.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- No health field in list/detail payloads without access (test)
- `verify` green

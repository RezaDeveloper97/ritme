---
id: B-N9-06
title: Message engine view and simulator
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-05]
parallel_group: N9-F
touches: [backend-go/internal/messages,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-06 — Message engine view and simulator

## Why
Explainability for the team.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Engine.dc.html` (+ `nbd_Admin_Engine`)

## Scope
- Pipeline explainer, slot capacity/caps config, simulator (user id + date → engine state + winner per slot with reason).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Simulator output matches live selection (test)
- `verify` green

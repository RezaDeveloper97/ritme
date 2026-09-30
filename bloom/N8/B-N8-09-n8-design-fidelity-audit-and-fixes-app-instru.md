---
id: B-N8-09
title: N8 design-fidelity audit and fixes (app + instructor-web)
milestone: N8
type: frontend
status: todo
depends_on: [B-N8-04,B-N8-07]
parallel_group: N8-I
touches: [docs/night-bloom/audit-n8.md,frontend/src,instructor-web/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build && cd .. && cd instructor-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# B-N8-09 — N8 design-fidelity audit and fixes (app + instructor-web)

## Why
Milestone audit (also check the dark-only `nbd_User_Library` / `nbd_User_Player` artboards).

## Scope
- As B-N1-16.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Audit table with Resolution
- `verify` green

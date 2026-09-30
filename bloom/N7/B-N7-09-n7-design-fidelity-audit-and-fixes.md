---
id: B-N7-09
title: N7 design-fidelity audit and fixes
milestone: N7
type: frontend
status: todo
depends_on: [B-N7-01,B-N7-05,B-N7-07]
parallel_group: N7-I
touches: [docs/night-bloom/audit-n7.md,frontend/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N7-09 — N7 design-fidelity audit and fixes

## Why
Milestone audit.

## Scope
- As B-N1-16.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Audit table with Resolution
- `verify` green

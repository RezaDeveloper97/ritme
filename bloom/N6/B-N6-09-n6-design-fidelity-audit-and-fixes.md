---
id: B-N6-09
title: N6 design-fidelity audit and fixes
milestone: N6
type: frontend
status: todo
depends_on: [B-N6-02,B-N6-04,B-N6-04b,B-N6-07,B-N6-08]
parallel_group: N6-I
touches: [docs/night-bloom/audit-n6.md,frontend/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N6-09 — N6 design-fidelity audit and fixes

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

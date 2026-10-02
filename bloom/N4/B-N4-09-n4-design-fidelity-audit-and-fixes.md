---
id: B-N4-09
title: N4 design-fidelity audit and fixes
milestone: N4
type: frontend
status: done
depends_on: [B-N4-08]
parallel_group: N4-I
touches: [docs/night-bloom/audit-n4.md,frontend/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N4-09 — N4 design-fidelity audit and fixes

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

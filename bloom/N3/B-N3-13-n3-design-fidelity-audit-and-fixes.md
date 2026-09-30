---
id: B-N3-13
title: N3 design-fidelity audit and fixes
milestone: N3
type: frontend
status: todo
depends_on: [B-N3-04,B-N3-05,B-N3-06,B-N3-09,B-N3-10,B-N3-11,B-N3-12]
parallel_group: N3-M
touches: [docs/night-bloom/audit-n3.md,frontend/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-13 — N3 design-fidelity audit and fixes

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

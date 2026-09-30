---
id: B-N2-10
title: N2 design-fidelity audit and fixes
milestone: N2
type: frontend
status: todo
depends_on: [B-N2-02,B-N2-03,B-N2-07,B-N2-08]
parallel_group: N2-J
touches: [docs/night-bloom/audit-n2.md,frontend/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N2-10 — N2 design-fidelity audit and fixes

## Why
Milestone audit.

## Scope
- As B-N1-16 for every N2 screen.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Audit table with Resolution; verify green
- `verify` green

---
id: B-N9-12
title: N9 design-fidelity audit and fixes (admin)
milestone: N9
type: frontend
status: todo
depends_on: [B-N9-04,B-N9-06,B-N9-07,B-N9-09,B-N9-10,B-N9-11]
parallel_group: N9-L
touches: [docs/night-bloom/audit-n9.md,admin-web/src]
skills: [verify-all]
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-12 — N9 design-fidelity audit and fixes (admin)

## Why
Milestone audit.

## Scope
- As B-N1-16 for the 12 admin artboards at 1440px.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Audit table with Resolution
- `verify` green

---
id: B-N5-10
title: N5 design-fidelity audit and fixes
milestone: N5
type: frontend
status: done
depends_on: [B-N5-04,B-N5-06,B-N5-07,B-N5-08]
parallel_group: N5-J
touches: [docs/night-bloom/audit-n5.md,frontend/src,frontend/messages]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N5-10 — N5 design-fidelity audit and fixes

## Why
Milestone audit.

## Scope
- As B-N1-16.

- From canvas QA (CB-LOSS-03): after a pregnancy loss `/pregnancy` shows a big «راه‌اندازی حالت بارداری» CTA — redirect to home or show one calm line without CTA. Companion home shows «بارداری · همین کارت بالا» when there is no card above — hide/fix the hint.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Audit table with Resolution
- `verify` green

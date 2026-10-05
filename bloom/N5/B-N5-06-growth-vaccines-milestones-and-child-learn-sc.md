---
id: B-N5-06
title: Growth, vaccines, milestones and child learn screens
milestone: N5
type: frontend
status: done
depends_on: [B-N5-05]
parallel_group: N5-F
touches: [frontend/src/screens/child-growth,frontend/src/screens/child-vaccines,frontend/src/screens/child-milestones,frontend/src/screens/child-learn]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N5-06 — Growth, vaccines, milestones and child learn screens

## Why
Child detail screens.

## Design
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v16_Growth.dc.html` (+ `nbd_v16_Growth`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v16_Vaccines.dc.html` (+ `nbd_v16_Vaccines`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v16_Milestones.dc.html` (+ `nbd_v16_Milestones`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v16_Learn.dc.html` (+ `nbd_v16_Learn`)

## Scope
- Growth chart with P3–P97 band and median, history; vaccines schedule/card/notes tabs, mark given, book → appointment form; milestones by month with non-judgemental copy; learn list by age.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

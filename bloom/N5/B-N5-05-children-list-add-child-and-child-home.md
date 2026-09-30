---
id: B-N5-05
title: Children list, add child and child home
milestone: N5
type: frontend
status: todo
depends_on: [B-N5-02]
parallel_group: N5-E
touches: [frontend/src/screens/children,frontend/src/screens/child-add,frontend/src/screens/child-home,frontend/src/entities/child]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N5-05 — Children list, add child and child home

## Why
v15/v16 child screens.

## Design
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v15_Children.dc.html` (+ `nbd_v15_Children`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v15_AddChild.dc.html` (+ `nbd_v15_AddChild`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v16_ChildHome.dc.html` (+ `nbd_v16_ChildHome`)

## Scope
- List with next vaccine and growth verdict, add child form (photo local only), child home (measurements, next vaccine, tiles, «این هفته آوا», today feed/sleep/diapers).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

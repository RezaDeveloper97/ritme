---
id: B-N5-04
title: Postpartum home, recovery and mood check screens
milestone: N5
type: frontend
status: todo
depends_on: [B-N5-01,B-N1-04]
parallel_group: N5-D
touches: [frontend/src/screens/postpartum,frontend/src/screens/postpartum-recovery,frontend/src/screens/postpartum-mood,frontend/src/entities/postpartum]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N5-04 — Postpartum home, recovery and mood check screens

## Why
v15 screens.

## Design
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v15_Main.dc.html` (+ `nbd_v15_Main`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v15_Recovery.dc.html` (+ `nbd_v15_Recovery`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_v15_MoodCheck.dc.html` (+ `nbd_v15_MoodCheck`)

## Scope
- Home (week ring, today's state chips, bleeding/feeds/sleep tiles, upcoming visits incl. child vaccines, «کی فوراً تماس بگیرم؟», weekly tip), Recovery form, Mood check (3 questions, non-diagnostic copy, safety path).

- From the N3 stage smoke (B-4): postpartum users currently see the cycle home with «تأخیر پریود ۱۰ روز» — the postpartum home must replace it.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

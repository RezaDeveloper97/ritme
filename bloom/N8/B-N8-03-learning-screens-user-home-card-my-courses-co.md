---
id: B-N8-03
title: Learning screens (user) — home card, my courses, course, players, unlocked
milestone: N8
type: frontend
status: todo
depends_on: [B-N8-01,B-N8-02]
parallel_group: N8-C
touches: [frontend/src/screens/learn*,frontend/src/entities/course,frontend/src/widgets/learn-home-card]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N8-03 — Learning screens (user) — home card, my courses, course, players, unlocked

## Why
Option A (home card) chosen by the user.

## Design
- `docs/design/night-bloom/e-learning-instructor/nbl_LearnEntry_HomeContinue.dc.html` (+ `nbd_LearnEntry_HomeContinue`)
- `docs/design/night-bloom/e-learning-instructor/nbl_LearnEntry_HomeNew.dc.html` (+ `nbd_LearnEntry_HomeNew`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Hub.dc.html` (+ `nbd_Learn_Hub`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Course.dc.html` (+ `nbd_Learn_Course`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Video.dc.html` (+ `nbd_Learn_Video`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Audio.dc.html` (+ `nbd_Learn_Audio`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_PDF.dc.html` (+ `nbd_Learn_PDF`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Unlocked.dc.html` (+ `nbd_Learn_Unlocked`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Profile.dc.html` (+ `nbd_Learn_Profile`)

## Scope
- Home card (continue / new from instructor), my courses (filters, access days, expired), course (chapters, progress, locked chapter note), video player (speed, audio-only, notes, files, next), audio player (background, speed, sleep timer, transcript), PDF viewer, unlocked notification screen, learning entry in Me.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

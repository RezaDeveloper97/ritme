---
id: B-N1-05
title: Splash, intro slides and welcome
milestone: N1
type: frontend
status: done
depends_on: [B-N1-03]
parallel_group: N1-E
touches: [frontend/src/screens/auth-splash,frontend/src/screens/welcome,frontend/src/widgets/intro-carousel]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-05 — Splash, intro slides and welcome

## Why
First impression screens of the new design.

## Design
- `docs/design/night-bloom/a-start-plus/nbl_Splash.dc.html` (+ `nbd_Splash`)
- `docs/design/night-bloom/a-start-plus/nbl_Intro_1.dc.html` (+ `nbd_Intro_1`)
- `docs/design/night-bloom/a-start-plus/nbl_Intro_2.dc.html` (+ `nbd_Intro_2`)
- `docs/design/night-bloom/a-start-plus/nbl_Intro_3.dc.html` (+ `nbd_Intro_3`)
- `docs/design/night-bloom/a-start-plus/nbl_Intro_4.dc.html` (+ `nbd_Intro_4`)
- `docs/design/night-bloom/a-start-plus/nbl_Intro_5.dc.html` (+ `nbd_Intro_5`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Welcome.dc.html` (+ `nbd_Onb_Welcome`)

## Scope
- Splash; 5 swipeable intro slides (skip, next, «حساب دارم · ورود»), slide 5 = social responsibility / always-free core; Welcome. Illustrations as inline SVG from the artboards.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

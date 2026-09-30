---
id: B-N8-04
title: Offline downloads for lessons
milestone: N8
type: frontend
status: todo
depends_on: [B-N8-03]
parallel_group: N8-D
touches: [frontend/src/screens/learn-downloads,frontend/src/shared/pwa,frontend/scripts]
skills: [pwa,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N8-04 — Offline downloads for lessons

## Why
«دانلود برای تماشای آفلاین».

## Design
- `docs/design/night-bloom/e-learning-instructor/nbl_Learn_Downloads.dc.html` (+ `nbd_Learn_Downloads`)

## Scope
- Store encrypted-at-rest-by-browser media in Cache Storage/IndexedDB via the SW, storage meter, delete, auto-purge when access expires; playable only inside the app. Follow the pwa skill; never edit `public/sw.js` directly.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

---
id: B-N7-07
title: Assistant screens — hub, department intro, chat, summary, handoff, history, profile
milestone: N7
type: frontend
status: todo
depends_on: [B-N7-06,B-N7-05]
parallel_group: N7-G
touches: [frontend/src/screens/assistant*,frontend/src/entities/assistant,frontend/src/features/assistant-chat]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N7-07 — Assistant screens — hub, department intro, chat, summary, handoff, history, profile

## Why
Section د assistant side.

## Design
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_AssistantHub.dc.html` (+ `nbd_v18_AssistantHub`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_DeptIntro.dc.html` (+ `nbd_v18_DeptIntro`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_AIChat.dc.html` (+ `nbd_v18_AIChat`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_Summary.dc.html` (+ `nbd_v18_Summary`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_Handoff.dc.html` (+ `nbd_v18_Handoff`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_History.dc.html` (+ `nbd_v18_History`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v18_Profile.dc.html` (+ `nbd_v18_Profile`)

## Scope
- Seven screens per artboards; streaming rendering; lab «ask assistant» deep link.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

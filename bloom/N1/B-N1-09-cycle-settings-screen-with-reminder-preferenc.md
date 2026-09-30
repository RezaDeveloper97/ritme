---
id: B-N1-09
title: Cycle settings screen with reminder preferences
milestone: N1
type: fullstack
status: todo
depends_on: [B-N1-06]
parallel_group: N1-I
touches: [frontend/src/screens/cycle-settings,backend-go/internal/profile,backend-go/internal/reminders,backend-go/db,backend-go/api]
skills: [new-endpoint,new-fsd-slice,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-09 — Cycle settings screen with reminder preferences

## Why
Settings are currently spread across profile sheets.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Settings.dc.html` (+ `nbd_Cycle_Settings`)

## Scope
- Cycle/period length with «خودکار از داده‌ها» toggle (engine medians vs manual).
- Reminders: before period (days + time), PMS start, fertile window, daily log (time), pill (time). Store as notification preferences; scheduler honours them.
- Mode section linking to the mode switcher (B-N2-03).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green

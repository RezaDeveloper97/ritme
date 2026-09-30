---
id: B-N1-11
title: Notification settings — categories, quiet hours, discreet copy
milestone: N1
type: fullstack
status: todo
depends_on: [B-N1-10]
parallel_group: N1-K
touches: [frontend/src/screens/profile-notifications,backend-go/internal/notifications,backend-go/internal/profile,backend-go/db,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-11 — Notification settings — categories, quiet hours, discreet copy

## Why
Users need control and privacy on the lock screen.

## Design
- `docs/design/night-bloom/g-me-settings/nbl_Me_Notifications.dc.html` (+ `nbd_Me_Notifications`)

## Scope
- Categories: cycle (before period, PMS, fertile, daily log), health (meds, appointments, checkups, vitals), other (learning, companion, articles ≤1/week). Quiet hours window. «متن خنثی» option: neutral push text.
- Backend stores prefs; every push/web-push sender checks category + quiet hours + neutral copy.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- Unit tests for quiet-hours and neutral-copy decisions
- `verify` green

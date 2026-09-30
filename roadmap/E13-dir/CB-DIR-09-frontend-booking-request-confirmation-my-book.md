---
id: CB-DIR-09
title: Frontend: booking request, confirmation, my bookings
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-08, CB-DIR-03]
parallel_group: DIR-G
touches: [frontend/src/screens/city-services-book, frontend/src/screens/city-services-booked, frontend/src/screens/my-bookings, frontend/src/app/[locale]/(app)/services/mother-child/book, frontend/src/app/[locale]/(app)/profile/bookings]
skills: [new-fsd-slice]
boards: [nbl_Dir_Book.dc.html, nbl_Dir_Booked.dc.html, nbl_Dir_MyBookings.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-09 — Frontend: booking request, confirmation, my bookings

## Why
Service, child, day, slot → request; track it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Book.dc.html`
- `nbl_Dir_Booked.dc.html`
- `nbl_Dir_MyBookings.dc.html`

## Scope
1. Book (services, child chips + add child, 7-day strip, slot grid, note, summary, data-sharing note, 'send request' — price 'pay at the place', no payment step). Booked/pending (status, what to bring, directions, reminder added, cancel window). My bookings (tabs, cancel/reschedule, review, book again), linked from Me.
2. (CB-CORE-01) Me is bloom's `/profile` hub (B-N1-10) → route `/profile/bookings` (not `/me/...`). It supersedes bloom B-N10-05's «my bookings» list and also lists bloom doctor bookings (B-N7-03, `/services/bookings/[id]`) when they exist.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- Board copy that says 'paid' becomes 'pay at the place' (DECISIONS).
- `verify` green.

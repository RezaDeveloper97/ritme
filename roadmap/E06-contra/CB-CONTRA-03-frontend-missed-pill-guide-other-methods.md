---
id: CB-CONTRA-03
title: Frontend: missed-pill guide + other methods
epic: CONTRA
type: frontend
status: in_progress
depends_on: [CB-CONTRA-02]
parallel_group: CONTRA-C
touches: [frontend/src/screens/contraception-missed, frontend/src/screens/contraception-other, frontend/src/app/[locale]/contraception/missed, frontend/src/app/[locale]/contraception/other, frontend/src/entities/contraception, frontend/messages/fa/contraception.json, frontend/messages/en/contraception.json, backend-go/resources/translations/fa/contraception.json, backend-go/resources/translations/en/contraception.json, backend-go/internal/i18n/testdata, frontend/src/app/message-scopes.ts, frontend/src/app/globals.css, docs/qa/canvas/contra.md, docs/qa/canvas/contra]
skills: [new-fsd-slice]
boards: [nbl_Contra_Missed.dc.html, nbl_Contra_Other.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-CONTRA-03 — Frontend: missed-pill guide + other methods

## Why
Guidance and long-acting reminders.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Contra_Missed.dc.html`
- `nbl_Contra_Other.dc.html`

## Scope
1. Missed: count chips → numbered steps (catalog), danger card, general-guidance note. Other: IUD / injection / implant cards with dates and done states.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.

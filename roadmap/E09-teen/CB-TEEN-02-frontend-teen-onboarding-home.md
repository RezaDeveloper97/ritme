---
id: CB-TEEN-02
title: Frontend: teen onboarding + home
epic: TEEN
type: frontend
status: done
depends_on: [CB-TEEN-01]
parallel_group: TEEN-B
touches: [frontend/src/screens/teen, frontend/src/screens/teen-onboarding, frontend/src/entities/teen, frontend/messages/fa/teen.json, frontend/messages/en/teen.json, frontend/src/app/[locale]/teen, frontend/src/app/message-scopes.ts, frontend/src/screens/home]
skills: [new-fsd-slice]
boards: [nbl_Teen_Onb.dc.html, nbl_Teen_Home.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-TEEN-02 — Frontend: teen onboarding + home

## Why
Replaces bloom's simplified teen home with the canvas home.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Teen_Onb.dc.html`
- `nbl_Teen_Home.dc.html`

## Scope
1. Onboarding (age chips, menarche cards, privacy note). Home: signs card + estimate, school-kit checklist, 'is it normal?' accordion, when-to-talk note, mother-sharing link. No shop/banners.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.

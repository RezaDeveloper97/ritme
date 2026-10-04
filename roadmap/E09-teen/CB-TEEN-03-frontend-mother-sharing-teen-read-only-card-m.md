---
id: CB-TEEN-03
title: Frontend: mother sharing (teen) + read-only card (mother)
epic: TEEN
type: frontend
status: done
depends_on: [CB-TEEN-02, B-N4-04]
parallel_group: TEEN-C
touches: [frontend/src/screens/teen-parent, frontend/src/app/[locale]/teen/parent, frontend/src/widgets/linked-teen-card, frontend/src/screens/home]
skills: [new-fsd-slice]
boards: [nbl_Teen_Parent.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-TEEN-03 — Frontend: mother sharing (teen) + read-only card (mother)

## Why
Teen chooses what mother sees.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Teen_Parent.dc.html`

## Scope
1. Grant switches, live preview card, invite via bloom invite flow; mother accepts and sees the read-only card on her Today.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.

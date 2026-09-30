---
id: CB-IVF-03
title: Frontend: injection schedule, site rotation, inventory
epic: IVF
type: frontend
status: todo
depends_on: [CB-IVF-02]
parallel_group: IVF-C
touches: [frontend/src/screens/ivf-meds, frontend/src/app/[locale]/(app)/ivf/meds]
skills: [new-fsd-slice]
boards: [nbl_IVF_Meds.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-IVF-03 — Frontend: injection schedule, site rotation, inventory

## Why
What, when, where to inject, how much is left.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_IVF_Meds.dc.html`

## Scope
1. Trigger card, today/tomorrow lists, 8-site picker with last/next, inventory with low badge, add-from-prescription form.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.

---
id: CB-MENO-08
title: Frontend: monthly score + patterns (stage tab 'علائم')
epic: MENO
type: frontend
status: done
depends_on: [CB-MENO-05, B-N1-04]
parallel_group: MENO-D
touches: [frontend/src/screens/menopause-score, frontend/src/app/[locale]/menopause/score, frontend/src/widgets/bottom-nav]
skills: [new-fsd-slice]
boards: [nbl_Meno_Score.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-08 — Frontend: monthly score + patterns (stage tab 'علائم')

## Why
Stage tab of menopause mode.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Score.dc.html`

## Scope
1. Score header with band bar, domain breakdown, 6-month chart (bloom chart primitives), HRT annotation, patterns + disclaimer, 11-question monthly flow (0–4).
2. (CB-CORE-01) Re-point the menopause mode tab «علائم» from bloom's default (`/analysis/symptoms`, docs/night-bloom/nav.md) to this screen — `nbl_Meno_Home`/`nbl_Meno_Score` nav: امروز → menopause home, علائم → score. Bloom's symptom analysis stays reachable from the score screen.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.

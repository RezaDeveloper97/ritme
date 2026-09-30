---
id: CB-CORE-02b
title: Canvas primitives into globals.css and ui-kit showcase
epic: CORE
type: frontend
status: todo
depends_on: [CB-CORE-02,B-N1-04,B-N1-05,B-N1-13]
parallel_group: CORE-C
touches: [frontend/src/app/globals.css,frontend/src/shared/ui/nb,frontend/src/screens/ui-kit,frontend/src/app/[locale]/dev/ui-kit]
skills: [new-fsd-slice]
boards: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-CORE-02b — Canvas primitives into globals.css and ui-kit showcase

## Why

## Boards

## Scope
1.

## Out of scope

## Acceptance
-
## Why
CB-CORE-02 put the primitives' styles in `frontend/src/shared/ui/nb/canvas-primitives.css` (globals.css was being edited by bloom B-N1-04/05/13), so `lint:styles` / `lint:dark` don't scan them, and the ui-kit showcase entry was out of its touches.

## Scope
1. Move `canvas-primitives.css` into `globals.css` (or wherever the gates scan — follow bloom B-N1-03's placement), delete the side file and its import.
2. Add SeverityScale, NumericScale (0–10 / 1–6), StepTimeline (numbered + dotted), CountdownRing, WeekDots, ProgressBar, RadioCardGroup, SearchField, Checkbox and UrgentCard `variant="note"` + hotlines to the `/dev/ui-kit` page next to bloom's primitives.
3. NumericScale 0–10 at 390 px: cells are ~26 px wide — keep ≥44 px touch area (e.g. two rows 0–5 / 6–10, or taller hit area); pick one, note it.
4. NumericScale `solid`: restrict to tones with contrast-safe `--on-*` pairs, or add the pair.

## Acceptance
- gates scan the primitives' CSS and stay green; ui-kit shows them in light + dark (screenshots in `docs/qa/canvas/core/CB-CORE-02b/`); `verify` green.

---
id: T-M5-05
title: Frontend — TTC quick tiles (LH / BBT / intercourse) on the cycle home
milestone: M5
type: frontend
status: todo
depends_on: [T-M5-04]
parallel_group: M5-B
touches: [frontend/src/widgets/fertility-tiles, frontend/src/screens/home/ui/HomePage.tsx]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M5-05 — Frontend — TTC quick tiles (LH / BBT / intercourse) on the cycle home

## Why
The section the user pasted from `v19_Main`.

## Scope
1. Widget: 3-column grid, card = 48 px icon disc (color @ 13 % fill / 33 % border), bold label, small value
   (LH label / BBT «۳۶٫۴۲°» / intercourse label, else «ثبت نشده») from `/fertility/today`.
2. Links: LH → `/fertility/log?focus=lh`, BBT → `/fertility/bbt`, intercourse → `/fertility/log?focus=intercourse`.
   Real `<a>` elements, ≥ 44 px, aria-label including the value.
3. Mount on `HomePage.tsx` in the design's spot (below the phase/chance block); visible when
   `pregnancyIntention === 'trying'`, hidden in pregnancy mode; skeleton while loading.

## Acceptance
- Light/dark × fa/en screenshots match the artboard (`v19_Main` / `nb2_Main`); gates green.

---
id: T-M5-10
title: Fertility UI polish from local e2e
milestone: M5
type: frontend
status: todo
depends_on: [T-M5-05,T-M5-06,T-M5-07,T-M5-08]
parallel_group: M5-D
touches: [frontend/src/widgets/fertility-tiles,frontend/src/widgets/bbt-chart,frontend/src/screens/fertility-log,frontend/src/screens/fertility-bbt,frontend/src/screens/fertility-insights,frontend/src/shared/lib, docs/fertility-ttc]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M5-10 — Fertility UI polish from local e2e

## Why
The local e2e for T-M5-09 (docs/fertility-ttc/README.md § Local e2e) found visual/formatting bugs on the fertility
screens that should be fixed before rollout.

## Scope
1. Tile colours swapped: LH must use the amber disc and BBT the turquoise one per `v19_Main` and the token comments
   (`frontend/src/widgets/fertility-tiles/ui/FertilityTiles.tsx` `DISC`). Check the design/tokens first and follow them.
2. Tiles row has no side gutter on the cycle home (FertilityTiles root) — match the 16px gutter of other home cards.
3. Cards on fertility-log / fertility-bbt / fertility-insights have no inner padding (global `.card` has none) — add
   padding in those screens (do not change the global `.card`).
4. Latin digits inside Persian copy («بر اساس 3 سیکل», «3 روز جاافتاده») — pass numbers through the locale number
   formatter (FertilityInsightsPage, FertilityBbtPage).
5. BBT chart Y-axis labels use `.` instead of the locale decimal separator — use the BBT formatter (BbtChart).
6. Insights calendar only draws the last month when the fertile window spans two months — render every month the
   window touches (or both months), marking all window days.

## Out of scope
- The `fertility_level` mismatch between `/cycle/today` top-level and `daily_card` (needs a product decision).
- Lalezar decimal-separator glyph look.

## Acceptance
- All 6 items fixed, with unit tests where logic changed (number formatting, month span).
- Frontend verify green; light and dark screenshots of home tiles, log, BBT and insights re-taken locally into
  `docs/fertility-ttc/screenshots/` (overwrite the old ones).

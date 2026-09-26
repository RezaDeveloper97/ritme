---
id: T-M7-11
title: Frontend — pregnancy «هفته‌به‌هفته» (Week) v2 screen
milestone: M7
type: frontend
status: done
depends_on: [T-M7-08, T-M7-02]
parallel_group: M7-C
touches: [frontend/src/screens/pregnancy-week, frontend/src/app/[locale]/pregnancy/weeks]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M7-11 — Frontend — pregnancy Week v2

## Scope
`/pregnancy/weeks/[n]` from `Week.dc.html`: header (week of 40, trimester · date range, bookmark toggle); chip
strip 1–42 auto-scrolled to current (past solid / current brand / future dashed) + legend; hero (FetusSize
illustration, size pill, headline, 3 stats, averages note); segmented tabs جنین / بدن تو / کارهای هفته (the pasted
section = tab جنین: highlights with tinted icon tiles); body tab chips + text + link to log; tasks checklist (shared
with Today); warning box; reviewer line with sources link. Swipe between weeks.

## Acceptance
- Light/dark × fa/en screenshots incl. the pasted tab; build green.

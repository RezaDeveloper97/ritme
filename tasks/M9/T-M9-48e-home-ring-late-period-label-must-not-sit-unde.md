---
id: T-M9-48e
title: Home ring: late-period label must not sit under the today marker
milestone: M9
type: frontend
status: done
depends_on: [T-M9-48]
parallel_group: M9-R
touches: [frontend/src/screens/home, frontend/src/shared/ui, frontend/src/app/globals.css]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run test -- --run src/screens/home
---

# T-M9-48e — Home ring: late-period label must not sit under the today marker

## Why
Stage visual QA of engine v6 (T-M9-48, `docs/engine-v6/qa/T-M9-48/2-late8/fa_home.*.png`): for a period 1–14 days
late the home ring shows the action buttons («پریودم شروع شد», «هنوز نه», «پریود نداشتم») and the label «تأخیر پریود»
is pushed up to the top of the ring, where the today marker dot sits on top of it (both themes). With ≥ 15 days late
(`3-late15`) the label sits clear of the marker. Engine data is correct; this is layout only.

## Scope
- In `frontend/src/screens/home/ui/HomePage.tsx` (late hero, around `hero.variant === 'late'`) keep the label, the
  number and the buttons inside the ring's inner circle so no text crosses the dot track or the today marker,
  at 360 px and 390 px widths, fa and en.
- Light and dark.

## Out of scope
- Engine or API changes; copy changes.

## Acceptance
- Late 8 persona: «تأخیر پریود» fully readable, no overlap with any ring dot, light and dark (screenshot at 360 and 390 px).
- Late 15 and regular rings unchanged.

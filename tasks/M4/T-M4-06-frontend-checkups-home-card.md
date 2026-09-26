---
id: T-M4-06
title: Frontend — «چکاپ‌های دوره‌ای» card on the cycle home
milestone: M4
type: frontend
status: done
depends_on: [T-M4-05]
parallel_group: M4-B
touches: [frontend/src/widgets/checkups-card, frontend/src/screens/home/ui/HomePage.tsx]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M4-06 — Frontend — «چکاپ‌های دوره‌ای» card on the cycle home

## Why
The card the user pasted (from `v14_Main`).

## Scope
1. Widget from `nbl_v14_Main` / `nbd_v14_Main`: 60 px progress ring (`up_to_date/total`, Lalezar label
   «۴/۶»), title + «همه» → `/checkups`, counts line «N مورد به‌روز · N موعدش رسیده · N عقب‌افتاده» (omit zero
   parts), up to two highlight rows from `/checkups/home` — cycle-timed/due in rose tint with «راهنما» (→ guide or
   detail), overdue in amber tint with «ثبت نوبت» (→ M3 AddAppointment with prefilled topic/title) — and the
   disclaimer line.
2. Placement on the cycle home near the design's position (after the cycle timeline, before «ثبت امروز»-style
   blocks); hidden in pregnancy mode and when `total = 0`; skeleton while loading; hidden on error.
3. Ring as an accessible SVG (`role="img"` + label «۴ از ۶ به‌روز»).

## Acceptance
- Light/dark × fa/en screenshots match the artboard; unit test for the counts line; gates green.

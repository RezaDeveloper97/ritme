---
id: T-M4-09
title: Frontend — Checkup history (+ PDF summary) and breast self-exam guide
milestone: M4
type: frontend
status: done
depends_on: [T-M4-08]
parallel_group: M4-C
touches: [frontend/src/screens/checkup-history, frontend/src/screens/checkup-self-exam, frontend/src/app/[locale]/checkups/history, frontend/src/app/[locale]/checkups/self-exam, frontend/src/shared/lib/pdf]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M4-09 — Frontend — Checkup history (+ PDF summary) and breast self-exam guide

## Why
Artboards `v14_History` and `v14_SelfExam`.

## Scope
1. History (`/checkups/history?type=`): header «سابقه چکاپ‌ها» + «N مورد ثبت‌شده», PDF export button; tabs همه /
   امسال / با پیوست (the last uses local-files presence); vertical timeline (dot + line, Jalali month-year, checkup
   title, result chip, «پیوست» chip opening the local file); infinite list; tap → edit record sheet.
2. «خلاصه برای پزشک»: client-side PDF (all records + results + notes, Vazirmatn embedded, RTL) via a lazily
   loaded generator in `shared/lib/pdf`; share/download; nothing sent to the server.
3. Self-exam (`/checkups/self-exam`): hero with «این ماه» chip, «روز N سیکل · X روز دیگر», best-time copy, cycle
   day ring (day/cycle length from the cycle entity); steps from `guide_steps`; findings chips from
   `finding_options` (multi-select, «چیزی متفاوت نبود» exclusive); «این ماه انجام دادم» → record with findings
   (result = follow_up if any finding, else normal) ; adherence line «N ماه از ۱۲ ماه اخیر انجام شده».
   A finding shows a gentle «با پزشک در میان بگذار» nudge with «ثبت نوبت».

## Acceptance
- Light/dark × fa/en screenshots; PDF opens with correct Persian shaping; build green.

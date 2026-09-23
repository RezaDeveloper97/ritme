---
id: T-M3-05
title: Frontend — «یادآورهای امروز» home card on both homes + AddChooser sheet
milestone: M3
type: frontend
status: done
depends_on: [T-M3-04]
parallel_group: M3-B
touches: [frontend/src/widgets/today-reminders, frontend/src/screens/home/ui/HomePage.tsx, frontend/src/screens/pregnancy/ui/PregnancyPage.tsx, frontend/src/screens/reminders-add, frontend/src/app/sheets/registry.tsx]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M3-05 — Frontend — «یادآورهای امروز» home card on both homes + AddChooser sheet

## Why
The card pasted by the user (= the section in `v13_Preg_Home`) replaces the hidden `DayTasks` block on the cycle
home and is added to the pregnancy home. Its «افزودن یادآور» button opens the AddChooser sheet.

## Scope
1. Widget `today-reminders` from `docs/design/reminders-v13/nbl_v13_Preg_Home.dc.html` (+ `nbd_` for dark):
   header (bell-ring tile + «یادآورهای امروز» + «همه» link → `/reminders`); one row per dose from `/care/today`
   (pill tile, title, localized time, round check: taken = success fill + check, not taken = outlined ring,
   taken title muted + line-through); tapping the check ticks/unticks via `log-intake` (optimistic); divider;
   next appointment row (amber date tile with Lalezar day + month name, title «topic · with», meta
   «N روز دیگر · ساعت … · place», chip «۱ روز قبل یادآوری») linking to the appointment detail;
   «+ افزودن یادآور» soft-brand button.
2. States: loading skeleton, empty (no doses, no appointment → short prompt + add button), error → hide card.
3. Sheet `reminders-add` (`v13_AddChooser`): handle, title «چه چیزی را یادآوری کنم؟», close button, three rows
   (medication → `/reminders/medication/new`, doctor visit → `/reminders/appointment/new?kind=in_person`,
   consultation → `?kind=phone`), footer disclaimer. Use the app's existing sheet host.
4. Mount on `HomePage.tsx` where `{/* <DayTasks date={base} /> */}` sits (remove that comment; leave DayTasks on the
   log page untouched) and on `PregnancyPage.tsx` in the design's position.

## Out of scope
The Reminders/Add/Detail screens (T-M3-06…08) — links may 404 until they land.

## Acceptance
- Pixel-close to the artboard in light **and** dark (screenshots of both themes, fa + en, via the headless
  verification recipe); RTL and LTR correct; all targets ≥ 44px; switches/checks are real buttons with labels.
- Unit test for dose-row state mapping; lint gates green.

## Note from T-M3-04
Register `care` in `frontend/src/app/message-scopes.ts` (`SHELL_NAMESPACES` if the AddChooser sheet lives in the
shell, and on the route lists of screens using it) — the scope test requires the lists to match real usage.

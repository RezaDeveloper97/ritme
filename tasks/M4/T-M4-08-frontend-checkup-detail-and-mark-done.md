---
id: T-M4-08
title: Frontend — Checkup detail screen and MarkDone sheet
milestone: M4
type: frontend
status: todo
depends_on: [T-M4-05]
parallel_group: M4-B
touches: [frontend/src/screens/checkup-detail, frontend/src/screens/checkup-mark-done, frontend/src/app/[locale]/checkups/[id], frontend/src/app/sheets/registry.tsx]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M4-08 — Frontend — Checkup detail screen and MarkDone sheet

## Why
Artboards `v14_CheckupDetail` and `v14_MarkDone`.

## Scope
1. Detail: header (title + interval, reminder bell → settings.remind toggle), gradient hero (category chip, status
   pill, «موعد بعدی» big Jalali date, relative line «حدود ۶ ماه گذشته · آخرین بار …», shield icon disc,
   «انجام دادم» → sheet, «ثبت نوبت» → M3 AddAppointment prefilled), «چرا مهم است؟», «قبل از رفتن» numbered
   steps + cycle-timing hint, «سابقه» (latest 2 records with result + «گزارش پیوست شده / بدون پیوست», «همه» →
   history filtered), disclaimer. Types with a guide (self-exam) link to the guide screen instead.
2. Sheet `checkup-mark-done`: title «X را انجام دادم», date picker (default today, not future), result tiles
   (نرمال / نیاز به پیگیری / منتظر جواب), optional attachment buttons (camera/photo and PDF → `local-files`,
   thumbnail + remove), note, next-due banner from `preview-next` with «تغییر» (date override), «ثبت».
   On success: close, toast, detail + home card refresh.
3. Editing an existing record reuses the sheet.

## Acceptance
- Light/dark × fa/en screenshots of both; attachment stays on device (network tab shows no upload); build green.

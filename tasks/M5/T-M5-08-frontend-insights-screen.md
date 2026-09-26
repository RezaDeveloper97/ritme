---
id: T-M5-08
title: Frontend — «پیش‌بینی‌ها» insights screen
milestone: M5
type: frontend
status: done
depends_on: [T-M5-04]
parallel_group: M5-B
touches: [frontend/src/screens/fertility-insights, frontend/src/app/[locale]/fertility/insights]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M5-08 — Frontend — «پیش‌بینی‌ها» insights screen

## Why
Artboard `v19_TTC_Insights` / `nb2_TTC_Insights`.

## Scope
1. Header «پیش‌بینی‌ها» + «بر اساس N سیکل ثبت‌شده».
2. Window card: title, confidence pill, summary line, month calendar (locale calendar, Saturday start) with window
   range and ovulation day highlighted, contraception disclaimer.
3. «این تخمین از کجا آمده؟» rows with strength chips (قوی / متوسط / ندارد).
4. «تخمک‌گذاری در سیکل‌های قبل» list (month, «روز N»), «چطور تخمین را دقیق‌تر کنم؟» tips with CTA to the Log.
5. Low-data state (0–1 cycles).

## Acceptance
- Light/dark × fa/en screenshots; build green.

---
id: CB-MENO-13b
title: Treatment adherence from the day an item was added
epic: MENO
type: backend
status: todo
depends_on: [CB-MENO-13]
parallel_group: MENO-E
touches: [backend-go/internal/menopause,frontend/src/screens/menopause-treatment]
skills: [new-endpoint]
boards: [nbl_Meno_Treatment.dc.html,nbl_Meno_Report.dc.html,nbl_Meno_Home.dc.html]
verify: cd backend-go && go vet ./... && go test ./internal/menopause/... && golangci-lint run ./internal/menopause/... && cd ../frontend && npm run typecheck && npm run test
---

# CB-MENO-13b — Treatment adherence from the day an item was added

## Why
Found in the MENO epic QA (CB-MENO-13, `docs/qa/canvas/meno.md`). «تاریخ شروع» is optional in the add sheet, and an
item without `started_on` is treated as running on every day of every window (`activeOn` / `overlaps` /
`adherenceOf` in `backend-go/internal/menopause/{treatment,report}.go`, and today.go's 7-day `Days`). A fresh user who
adds «استروژن ژل» today and taps «مصرف شد» sees:
- treatment screen: «این هفته · ۱ از ۵ روز» — Saturday–Tuesday (before she added it) count as missed;
- home card: «۱ از ۷ روز این هفته»;
- doctor report + PDF + shared link: «۱٪ مصرف منظم» over the 3-month window — misleading for the doctor.
Also, without a start date no review date is suggested, so the review card («بازبینی با پزشک») never appears.

## Boards
`nbl_Meno_Treatment` (week dots), `nbl_Meno_Home` (treatment card), `nbl_Meno_Report` («از مرداد · ۹۲٪ مصرف منظم»).

## Scope
1. Backend: an item with no `started_on` counts from its `created_at` date (Tehran wall clock) — one helper used by
   `activeOn`, `overlaps`, `adherenceOf`, the home's 7-day window and the treatment week; unit + int tests; check the
   `menopause` contract goldens (re-record only if a case changes, note it in the report).
2. Decide (ask user) whether the add sheet should prefill «تاریخ شروع» with today (she can change it) so the review
   date is suggested; if yes, do it in `screens/menopause-treatment` (`ItemSheet`).
3. Week dots: days before the item existed render as «not scheduled» (dashed), not as missed.

## Out of scope
- The night-sweat / symptom rates over tracked days (CB-MENO-03 open item: «۷ شب در هفته» from one logged day).
- Any other report wording.

## Acceptance
- A daily HRT item added today and taken today shows «۱ از ۱ روز» this week, 100% on the home card and in the report.
- An item with an explicit past `started_on` behaves as today.
- `verify` green.

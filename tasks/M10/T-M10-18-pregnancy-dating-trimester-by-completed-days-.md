---
id: T-M10-18
title: Pregnancy dating: trimester by completed days, Naegele and ACOG re-dating
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-16]
parallel_group: M10-C
touches: [backend-go/internal/pregnancy/calc, backend-go/internal/pregnancy/v2/dating.go, backend-go/internal/pregnancy/v2/dating_apply.go, backend-go/internal/pregnancy/v2/labels.go, backend-go/internal/pregnancy/profile.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist/pregnancy.yaml, backend-go/contract/allowlist/pregnancy-v2.yaml, backend-go/contract/allowlist/analysis-pregnancy.yaml, frontend/src/screens/pregnancy-onboarding, frontend/src/features/track-pregnancy, frontend/messages/fa/pregnancy-v2.json, frontend/messages/en/pregnancy-v2.json, backend-go/resources/translations/fa/pregnancy-v2.json, backend-go/resources/translations/en/pregnancy-v2.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/pregnancy/... && make test-int PKG=./internal/pregnancy/... && make contract ROUTES=pregnancy,pregnancy-v2,analysis-pregnancy && cd ../frontend && npm run typecheck && npm run lint && npm run test && npm run copy:check -- pregnancy-v2
---

# T-M10-18 — Pregnancy dating: trimester by completed days, Naegele and ACOG re-dating
**Model:** opus (clinical dating). The main session reviews the copy.

## Why
- Weeks 13+0 … 13+6 are labelled trimester 2 (`calc/calc.go:131-139` uses `weeks <= 12`; `v2/labels.go:107`; the test
  `calc_test.go:82` asserts the bug). SPEC §3.1: T1 ≤ 13+6, T2 14+0 … 27+6, T3 ≥ 28+0. Content, the trimester
  analysis and the lab Hb rule (T-M10-20) read it.
- An ultrasound always replaces LMP dating (`dating_apply.go:48`); SPEC §3.1 (ACOG CO 700) replaces it only when the
  discrepancy exceeds the band's limit (5/7/7/10/14/21 days) and says nothing is shown when it does not.
- EDD's 280 days and the optional cycle-length correction (default off) are not config.
D-169.

## Scope
1. One dating function (`internal/pregnancy/calc`) used by v1, v2, analysis, labs and message facts:
   - trimester from completed days and `pregnancy.trimester` (98 / 196);
   - EDD = LMP + `pregnancy.edd.naegele_days`; when `pregnancy.edd.cycle_correction.enabled = 1`, + (effective cycle
     length − 28) from the engine (off by default).
2. Re-dating: when a scan (date + GA at scan) is saved and an LMP exists, discrepancy = |GA by scan − GA by LMP| on the
   scan date; band from `pregnancy.redating` rows; replace the LMP dating only when the discrepancy is greater than
   the band's limit. Response `dating: {basis: lmp|ultrasound, replaced, discrepancy_days, band_until_days,
   limit_days}`. Without an LMP the scan dates the pregnancy. A user-chosen basis stays possible (existing
   `age_source`) but the default follows the rule.
3. Fix `v2/labels.go:107` and every other trimester computation found by grep (no second copy may remain; add a test
   that greps for `weeks <= 12`-style rules outside `calc`).
4. Contract: pregnancy, pregnancy-v2 and analysis-pregnancy payloads change only in the trimester of weeks 13+0…13+6
   and the new `dating` block — allow-list under D-169; update `calc_test.go:82`.
5. Frontend: after saving a scan, a short result line («تاریخ زایمانت بر اساس سونوگرافی به‌روز شد» / «اختلاف کمتر از حد
   لازم بود؛ تاریخ بر اساس آخرین پریود ماند»); trimester labels come from the API.

## Tests
- Trimester 13+6 → 1, 14+0 → 2, 27+6 → 2, 28+0 → 3.
- Re-dating: at 8+6 with 5 days → keep, 6 → replace; 9+0 with 7 → keep, 8 → replace; 15+6 with 8 → replace; 16+0 with
  10 → keep, 11 → replace; 27+6 with 14 → keep, 15 → replace; 28+0 with 21 → keep, 22 → replace.
- EDD for LMP 2026-01-01 = 2026-10-08; with correction on and a 35-day cycle = 2026-10-15.
- Frontend vitest for the result line mapping.

## Out of scope
- The pregnancy redesign (M7), labs (T-M10-20).

## Acceptance
- Verify passes; PROGRESS.md cites D-169.

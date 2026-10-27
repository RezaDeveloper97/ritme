---
id: T-M9-33
title: /cycle/month on the v6 Range: lateness, future cycles, assumed ranges
milestone: M9
type: backend
status: done
depends_on: [T-M9-32]
parallel_group: M9-C
touches: [backend-go/internal/cycle/service/month.go, backend-go/internal/cycle/service/month_test.go, backend-go/internal/cycle/service/month_int_test.go, backend-go/contract/golden-v6/cycle-month, backend-go/api/openapi.yaml, backend-go/contract/allowlist/cycle-sweep.yaml]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/service/... && make contract ROUTES=cycle,cycle-sweep
---

# T-M9-33 — /cycle/month on the v6 Range: lateness, future cycles, assumed ranges
**Model:** opus.

## Why
The calendar runs the legacy engine: its cycle day wraps modulo L, so it never shows lateness; ovulation is
`max(L−13, 7)`, the window O−5..O+1, PMS ≥ L−6, no contraception or pregnancy handling (gap 02 §5; D-30). In v6 mode the
month must come from the same engine as the Today screen.

## Scope
1. In `v6`/`shadow` mode `GET /cycle/month` (all views) is built from one `State.Range(first, last, LocalToday)`:
   per-day v6 fields (`main_phase`, `subphase`, `fertility_level`, `is_predicted`, `period_kind`: `logged` | `assumed`
   (open period's assumed days and the declared onboarding range, I-01) | `predicted` (future cycles) | null) plus the
   existing legacy day fields rebuilt from them (`is_period`, `is_fertile_window` = display window O−5..O,
   `is_pms`, phase name via I-11, cycle day without wrap).
2. Month-level `future_cycles` (k ≤ 6, E-05) and `limits`; late users: days from `predicted_next` to today are
   `period_expected`, tomorrow onward follow the rebased chain (E-04); nothing painted beyond `future_cycles[6]` (E-28);
   pregnancy: logged history only (E-25); hormonal: no fertile flags (E-13).
3. Completed months use the retrospective layout (E-22).
4. ETag helper; shadow classification for month responses; Go-native goldens for 6 personas × 3 months.
5. v1 mode unchanged (Laravel goldens).

## Tests
- Month spanning a cycle boundary; O+1 not fertile; 31-day July and 28/29-day February; a late user (8 days) shows
  period_expected through today and a predicted period from tomorrow; 15 days late → no predicted days; pill → no
  fertile days; onboarding-only → `assumed` days for the declared period; pregnancy → only logged history.
- Cross-consumer parity (8 personas × 120 days): month day == `cycle_view` for that date (phase, fertile flag, next
  start, lateness).
- Warm `/cycle/month?view=calendar` = 1 query, 304 on repeat.

## Budgets
- Warm p95 ≤ 20 ms, cold ≤ 80 ms; Range for the month ≤ 150 µs.

## Out of scope
- Frontend calendar (T-M9-45).

## Acceptance
- Tests green; PROGRESS.md cites D-121.

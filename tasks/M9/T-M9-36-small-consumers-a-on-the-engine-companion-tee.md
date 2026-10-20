---
id: T-M9-36
title: Small consumers A on the engine: companion, teen, checkups, cycle settings, challenges
milestone: M9
type: backend
status: todo
depends_on: [T-M9-32]
parallel_group: M9-C
touches: [backend-go/internal/companion/shared, backend-go/internal/teen, backend-go/internal/checkups, backend-go/internal/profile/cyclesettings.go, backend-go/internal/challenges, backend-go/internal/reminders]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/companion/... ./internal/teen/... ./internal/checkups/... ./internal/profile/... ./internal/challenges/... ./internal/reminders/... && make test-int PKG="./internal/companion/... ./internal/teen/... ./internal/checkups/... ./internal/challenges/..." && make contract ROUTES=companion,companion-home,teen,cycle-settings
---

# T-M9-36 — Small consumers A on the engine: companion, teen, checkups, cycle settings, challenges
**Model:** sonnet with main-session review (mechanical once the state provider exists; companion output is privacy-
sensitive — keep T-M9-02's rules).

## Why
These packages call the v1.1 resolver through private loaders and constants: companion hard-codes `MaxLateDays = 14`,
teen rolls its own week bucket, checkups build a profile without manual lengths, cycle settings skip the newest bleeding
period and take PMS from the reminders helper, challenges copy the loader (gap 02 §5).

## Scope
In v6/shadow mode (v1 unchanged):
1. Companion shared view: phase from the engine status; `fertility_suppressed` copied from `limits` (E-24); no lateness
   (T-M9-02 rule kept); cap from `Params`.
2. Teen: `WeekBucket` fed by v6 `predicted_next_period_start` (its deliberate lateness hiding stays).
3. Checkups: cycle start/length from the state (manual lengths honoured), projections from `FutureCycles`.
4. Cycle settings: metrics from the state; PMS window next − 3..next − 1.
5. Challenges: phase eligibility via the state provider; delete the duplicated loader.
6. `reminders/planner.go` (no caller): port it to `FutureCycles`/display window so a future sender cannot use the
   old math.
7. `LocalToday` in what you touch.

## Tests
- One fixture per package where today's code and the engine differ (manual lengths; newest bleeding-log period; 15 days
  late; pregnant user with no stored life_mode; pill).
- Existing package tests green; contract v1 unchanged.

## Out of scope
- Family model changes (M11).

## Acceptance
- Tests green; PROGRESS.md cites D-136.

---
id: T-M10-11
title: Message simulator v6 and conflict check v2 (admin API)
milestone: M10
type: backend
status: todo
depends_on: [T-M10-06]
parallel_group: M10-B
touches: [backend-go/internal/admin/messages/engine.go, backend-go/internal/admin/messages/simulate.go, backend-go/internal/admin/messages/conflicts.go, backend-go/internal/messages/rules/explain.go, backend-go/internal/messages/rules/conflicts.go, backend-go/internal/messages/rules/validate.go]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... ./internal/admin/messages/... && make test-int PKG=./internal/admin/messages/...
---

# T-M10-11 — Message simulator v6 and conflict check v2 (admin API)
**Model:** opus (reads a real user's health state; access must be audited).

## Why
SPEC §5.4.1: the simulator takes a user and a date and shows the engine output, the winner of each slot and why the
others were dropped; the editor shows conflicting messages and the winner. Today `simulate` takes synthetic facts only
and runs with no history, so caps and cooldowns never trigger; the real-user preview covers one rule; the conflict
check is a heuristic on shared fact names and dates (gap 03 row 7, §2.7).

## Scope
1. `POST /api/admin/v1/messages/simulate {user_id?, date?, time?, slots?, facts_override?}`:
   - with `user_id`: requires an active health-access grant for that user (the existing reason + ticket, 30-minute
     flow in `httpadmin/health.go`) and writes an audit row; builds real facts for that date and time in the user's
     zone (the engine resolves that date over the user's current inputs — say so in the response) and the user's real
     message history up to that moment; returns the engine summary (main_phase, subphase, cycle_day, days_late,
     confidence, data_quality, warnings, requires_user_input), winners per slot with variant, and every dropped
     candidate with its T-M10-04 reason;
   - without `user_id`: synthetic facts as today, empty history, labelled `synthetic`.
2. Conflict check v2 (`rules/conflicts.go`): two rules conflict when they share a slot, their schedules, hour windows
   and audiences overlap, and for every shared fact their value sets intersect (numeric intervals, enum and set
   intersections, booleans). Report each pair, the overlapping region and the winner by the selection order ("safety
   always wins"). Replace the heuristic in `validate.go:244-279`.

## Tests
- simulate(user, date, time) winners == feed winners for the same clock over 5 personas (golden).
- A cap reason appears when the seeded history fills the cap.
- No health-access grant → 403 `health_access_required`; with a grant → 200 and an audit row.
- Conflicts: `cycle_day` 1–5 vs 10–14 → none; 1–7 vs 5–10 → conflict with the overlap 5–7; safety vs product overlap
  → safety reported as the winner.

## Out of scope
- Admin-web screens (T-M10-13).

## Acceptance
- Verify passes.

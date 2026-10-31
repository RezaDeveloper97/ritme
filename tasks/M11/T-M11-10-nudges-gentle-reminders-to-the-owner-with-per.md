---
id: T-M11-10
title: Nudges: gentle reminders to the owner with per-dose, daily and quiet-hour limits
milestone: M11
type: backend
status: blocked
depends_on: [T-M11-06]
parallel_group: M11-C
touches: [backend-go/internal/family/nudges, backend-go/internal/care/today.go, backend-go/internal/http/routes_family.go, backend-go/db/queries/companion, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/family/... && make test-int PKG="./internal/family/... ./internal/care/..."
---

# T-M11-10 — Nudges: gentle reminders to the owner with per-dose, daily and quiet-hour limits
**Model:** opus.

## Why
SPEC §1.7–§1.8: at level nudge a member may send a kind reminder when a dose or appointment was not logged on time
(medication: 60 minutes after due), at most once per dose or appointment and 3 times a day per member, never 23:00–
08:00, with fixed kind copy and no drug name on the lock screen. Nothing exists (gap 05 §2.8). DECISIONS.md F-06,
F-07, F-08; D-198.

## Scope
1. `GET /api/v1/family/owners/{ownerId}/nudgeable` (member with meds or appointments ≥ nudge): today's dose slots
   ≥ 60 minutes past due in the owner's local day without an intake; appointments between her reminder time and the
   appointment time while still scheduled (F-07); each item says whether a nudge is possible and, if not, why.
2. `POST /api/v1/family/nudges {owner_id, target_type, target_id, slot}`: check() `nudge`, then one transaction with
   the link row locked: one per dose/appointment overall (UNIQUE), 3 per owner-local day per (member, owner) (F-06),
   hard quiet hours 23:00–08:00 owner-local and her own quiet hours → 409 `quiet_hours {available_from}` (F-08, never
   queued); before 60 minutes → 409 `not_due`; quota → 429 `nudge_quota`.
3. Delivery to her in-app notifications with her category preferences: title «یادآوری مهربان از همراهت» (no drug
   name), the in-app body may name her own medication; activity row `nudge_sent`.
4. Keep `sent_at` for the KPI "dose logged within 3 hours after a nudge" (T-M11-15).

## Tests
- `TestNudge_DailyCapAndPerDoseOnce` (4th refused, next local day allowed, second on the same dose refused),
  `TestNudge_QuietHoursOwnerLocal` (22:59 ok, 23:00 refused, 07:59 refused, 08:00 ok; an owner in Berlin),
  `TestNudge_60MinuteRule` (59 refused, 60 ok), `TestNudge_AppointmentWindow`, `TestNudge_TitleHasNoDrugName`,
  `TestNudge_ViewLevelDenied`, `TestNudge_IDOR`, `TestNudge_ConcurrentDoubleTapOneRow`.

## Out of scope
- Member UI (T-M11-20); member notifications (T-M11-13).

## Acceptance
- Verify passes; PROGRESS.md cites D-198.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.

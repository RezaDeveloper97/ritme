---
id: T-M12-21c
title: Medication doses, reminders and quiet hours on the user's local time zone
milestone: M12
type: fullstack
status: done
depends_on: [T-M12-11c,T-M12-21]
parallel_group: M12-TEST
touches: [backend-go/internal/care,backend-go/internal/notifications,backend-go/internal/reminders,backend-go/internal/platform/usertz,backend-go/api/openapi.yaml,frontend/src/features/manage-medication,frontend/src/entities/medication]
skills: []
verify: cd backend-go && CGO_ENABLED=0 go test ./internal/care/... ./internal/notifications/... ./internal/reminders/... && make test-int PKG="./internal/care/... ./internal/notifications/..."
---

# T-M12-21c — Medication doses, reminders and quiet hours on the user's local time zone

**Model:** opus. User decision 2026-11-18 (after T-M12-21): care follows the user's local time, like M-08 for messages.

## Why
Dose times, `/care/upcoming`, `reminders.Plan` and quiet hours (`notifications.InQuietHours/QuietEndsAt`) run on
Tehran time. A user in Berlin who sets «۸ صبح» gets the push at 05:30/06:30 local, and quiet hours 23–08 become
20:30–05:30 local. T-M12-21 tables pin the current behaviour (comments mark the question).

## Scope
1. Resolve the user's zone once per request/job via `platform/usertz` (stored zone; header wins in requests); Tehran
   stays the default when nothing is stored.
2. Dose occurrences, `/care/upcoming`, reminder planner and quiet hours computed in that zone; worker jobs read the
   stored zone per user.
3. Flip the T-M12-21 Berlin rows to local-time expectations (08:00 Berlin = 08:00 local, across both DST changes; no
   skipped or doubled dose on the 23 h / 25 h days); keep Tehran rows unchanged.
4. Existing stored schedules: times are wall-clock, so no data migration — document it.
5. OpenAPI: say times are in the user's zone.

## Out of scope
- Message engine (already M-08). Android.

## Acceptance
- Tables green with local-time rows; a Tehran user sees no change (existing tests untouched and green).

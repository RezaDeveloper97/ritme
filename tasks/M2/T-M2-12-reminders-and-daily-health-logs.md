---
id: T-M2-12
title: Reminders and daily health log endpoints
milestone: M2
type: backend
status: done
depends_on: [T-M2-05, T-M2-06, T-M2-07, T-M2-08]
parallel_group: M2-E
touches: [backend-go/internal/reminder, backend-go/internal/healthlog, backend-go/internal/http/routes_reminder.go, backend-go/internal/http/routes_healthlog.go, backend-go/db/queries/reminder, backend-go/db/queries/healthlog, backend-go/contract/allowlist/reminders.yaml, backend-go/contract/allowlist/healthlog.yaml]
skills: []
verify: cd backend-go && go test ./internal/reminder/... ./internal/healthlog/... && make test-int PKG=./internal/healthlog/... && make contract ROUTES=reminders,healthlog
---

# T-M2-12 — Reminders and daily health log endpoints

## Why
Daily logs feed the cycle engine, messages and home; their write path has side effects on cycle history. Read
api-inventory §1.5–§1.6, domain-inventory §1 (`daily_health_logs`, `reminders`), §2 (decimal casts) and
`DailyHealthLogController`, `StoreDailyHealthLogRequest`, `CycleHistoryService`.

## Scope
1. Reminders: `GET /reminders/enums` (types with icon; recurrences; D-01 3rd-locale behaviour — keep Laravel's 500
   unless D-01 is approved), `GET /reminders` (`?type`), `POST` (201), `PUT /{id}` (sometimes-rules, fresh model),
   `DELETE /{id}` (`{"success":true}`); controller 422 family with fa/en message; raw model JSON (plain `date` cast
   for starts_on/ends_on, `recurrence_time` `"16:00"` on create vs `"16:00:00"` on read); non-numeric id per D-02.
2. Health logs: `GET /health-logs/enums` (`DailyHealthLog::getEnumValues`), `GET /health-logs` (raw
   LengthAwarePaginator, 30/page, `from_date/to_date`), `POST` (~70 rules incl. `prepareForValidation` string→array
   for `exercise_type`; upsert on (user_id, log_date); **201 created / 200 updated**; created `data` = sent attrs +
   user_id, id, timestamps; updated = full row; decimal strings; optional top-level `warning` from
   `getSpottingWarning`; side effects `checkAndUpdatePeriodStart` + `markRecalculated`), `GET/DELETE /{date}` (404 text).
3. `internal/healthlog/model`: DailyHealthLog struct + serializer + the field accessor interface used by
   `RecommendationTrigger.Matches` (T-M2-07) and later by the engines/messages/home.
4. Confirm whether `CycleHistoryService` is live (domain-inventory §4.1 "dead code" note) and port only what is live;
   record the finding in the task's PROGRESS section.

## Out of scope
Period-log endpoints (T-M2-15).

## Acceptance
- `make contract ROUTES=reminders,healthlog` green (create vs update status codes, spotting warning, 422s fa/en,
  paginator URLs with `X-Forwarded-*`).
- Integration test: POST log with `spotting`/bleeding on a new date triggers the same cycle_histories change as Laravel
  (compare DB rows after the same sequence in both stacks).

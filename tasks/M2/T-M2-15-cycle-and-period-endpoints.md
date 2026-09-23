---
id: T-M2-15
title: Cycle and period-log endpoints with engine cache
milestone: M2
type: backend
status: done
depends_on: [T-M2-11, T-M2-12, T-M2-13, T-M2-14]
parallel_group: M2-G
touches: [backend-go/internal/cycle/periods, backend-go/internal/cycle/service, backend-go/internal/cycle/cache, backend-go/internal/http/routes_cycle.go, backend-go/db/queries/cycle, backend-go/contract/allowlist/cycle.yaml, backend-go/contract/allowlist/period.yaml]
skills: []
verify: cd backend-go && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/... && make contract ROUTES=cycle,period,cycle-sweep
---

# T-M2-15 — Cycle and period-log endpoints with engine cache

## Why
These are the hottest and most sensitive endpoints (calendar, today card, period logging). Read api-inventory §1.7,
domain-inventory §4.1 (cache, `loggedPeriodFor`), `CycleCalculationController`, `PeriodLogController` (783 lines).

## Scope
1. `cycle/service`: loads profile + histories + logs for the range (single queries, like the T-M1-12 slim version),
   calls legacy + v1.1 engines, "today" = Tehran civil date from the clock.
2. `cycle/cache`: Redis `ritme-go:cycle-engine:{uid}:{calc_version}:{locale}:{today}:{scope}:{hash(inputs)}`, TTL
   24h; correctness never depends on it (a test runs every contract case with cache on and off).
3. Endpoints: `GET /cycle/status`, `/cycle/today`, `/cycle/date/{date}` (lenient parse, 422), `/cycle/month/{y}/{m}`
   (`view=full|calendar`, 422 otherwise, UNESCAPED_UNICODE, raw bilingual `text_flags`/`daily_tips` in full view,
   `month_summary`), `POST /cycle/recalculate` (400 without profile).
4. `cycle/periods`: `GET /cycle/period/status`, `POST /cycle/period/start` (422 `code: previous_period_open` +
   `data.open_period_start`, `period_overlap`; warnings), `POST /cycle/period/end`, `GET /cycle/period/history`,
   `POST /cycle/period`, `PUT /cycle/period/{period}` (no `active` key), `DELETE /cycle/period/{period}`; every write
   in a transaction with the same `cycle_histories` reconciliation and `markRecalculated` as Laravel; validation
   `message = first error`.

## Out of scope
Home sections that show cycle data (T-M2-18).

## Acceptance
- `make contract ROUTES=cycle,period,cycle-sweep` green, including period write sequences (start → end → edit →
  delete → today) compared step by step.
- Integration test: the same period sequence run against Laravel and Go leaves identical `cycle_histories` rows.
- p95 latency of `/cycle/month` (calendar) on the fixture DB ≤ Laravel's (measured with `hey`, numbers in PROGRESS).

## Note from T-M2-14
- Write the sqlc adapter for `recommendation.Source` here (queries in `db/queries/cycle`): active rows ordered by
  `sort_order, id`, and an `EXISTS` over the whole table (inactive rows count for `HasContent`).
- Build `legacy.DailyLog.Source` as `DailyHealthLog::toArray()` output (reference casts: `dailyLogFromRow` in
  `internal/cycle/legacy/golden_sweep_test.go`). `legacy.Input.Today` = Tehran today; `Input.Tips` is required.

## Note from T-M2-11
`make contract ROUTES=profile` has 3 cases (create_profile.none/.en, update_cycle_fields) whose step 3 calls
`/cycle/period/history` and `/cycle/status` — they turn green once this task lands; include `profile` in the
contract run here. Reuse `internal/profile/model` (`Attributes` Eloquent serializer with CycleHistory and
DailyHealthLog cast tables) and `profile.MarkRecalculated`.

## Note from T-M2-12
3 cases in `make contract ROUTES=reminders,healthlog` fail only on later steps calling `/cycle/period/status|history`
— include `healthlog` in this task's contract run. For `legacy.DailyLog.Source` use
`healthlog/model.FromRow(store.DailyHealthLog(row)).ToArray()` (it also implements `enums.TriggerLog`).

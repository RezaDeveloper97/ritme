---
id: T-M2-15
title: Cycle and period-log endpoints with engine cache
milestone: M2
type: backend
status: todo
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

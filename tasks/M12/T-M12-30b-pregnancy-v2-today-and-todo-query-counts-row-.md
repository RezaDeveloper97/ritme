---
id: T-M12-30b
title: Pregnancy v2 today and todo query counts; row-growth budgets for home reads
milestone: M12
type: backend
status: todo
depends_on: [T-M12-30]
parallel_group: M12-PERF
touches: [backend-go/internal/http/routes_pregnancy_v2.go,backend-go/internal/pregnancy,backend-go/internal/todo,backend-go/db/queries/todo,backend-go/internal/querybudget]
skills: []
verify: cd backend-go && make test-int PKG="./internal/querybudget/... ./internal/pregnancy/... ./internal/todo/..."
---

# T-M12-30b — Pregnancy v2 today and todo query counts; row-growth budgets for home reads

**Model:** sonnet (opus reviews). Follow-up of T-M12-30 (`docs/quality-2026-11/perf.md`).

## Why
T-M12-30 measured `/pregnancy/v2/today` at 27 cold / 21 warm queries (service doc says 5; likely the
`al.EvaluateFirst` alerts middleware at `routes_pregnancy_v2.go:32`) and `/todo` at 6 warm on an empty fixture.
All new home-read budgets were measured on near-empty fixtures, so N+1 growth is not caught.

## Scope
1. Profile `/pregnancy/v2/today`; cut the alert evaluation from the read path (cache/skip when unchanged) or give it
   its own budget; target ≤ 8 warm. Same safety outcome (alerts still raised — test).
2. `/todo`: fold per-list reads; target ≤ 3 warm.
3. Row-growth tests: seed N=1 and N=20 rows for /care/today, /todo, /checkups/home, /home/notifications — count must
   not grow (like `TestMessagesSelectedConstantInRules`).
4. Tighten budgets to the new counts; show a tightened row fails when exceeded.

## Acceptance
- Budgets lowered and green; row-growth tests green; alert behaviour unchanged (tests).

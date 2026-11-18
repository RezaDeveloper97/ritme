---
id: T-M12-30
title: Query budgets for the real web-home fan-out and a web_home k6 scenario
milestone: M12
type: backend
status: todo
depends_on: [T-M12-11]
parallel_group: M12-PERF
touches: [backend-go/internal/querybudget, backend-go/loadtest/web_home.js, backend-go/loadtest/thresholds.js, docs/quality-2026-11/perf.md]
skills: []
verify: cd backend-go && make test-int PKG="./internal/querybudget/..." && make loadtest-lint
---

# T-M12-30 — Query budgets for the real web-home fan-out and a web_home k6 scenario
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-05: 14 endpoints the web home calls on every open have no query budget, and k6 measures the Android-only `/home`.

## Scope
1. Budget rows (queries per request, warm cache) for `/care/today`, `/checkups/home`, `/todo`, `/home/notifications`, `/modules`, `/profile`, `/fertility/today`, `/banners/ads`, `/learning/courses`, `/logs/taxonomy`, `/logs/days/:date`, `/pregnancy/v2/today`, `/companion/home`, `/teen/today`.
2. `web_home.js`: replays the real cold-home fan-out (derive the list from the frontend hooks, record it in perf.md) with p95 thresholds.
3. Any endpoint over a sensible budget → a fix in this task if small, or a follow-up task.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Budgets green; k6 run results recorded in perf.md.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).

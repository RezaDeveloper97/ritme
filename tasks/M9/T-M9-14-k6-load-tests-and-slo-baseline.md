---
id: T-M9-14
title: k6 load tests and SLO baseline
milestone: M9
type: backend
status: todo
depends_on: [T-M9-07, T-M9-08]
parallel_group: M9-B
touches: [backend-go/loadtest, backend-go/Makefile, docs/engine-v6/perf-baseline.md]
skills: [local-dev]
verify: cd backend-go && test -f loadtest/app_open.js && test -f loadtest/seed/main.go && go vet ./loadtest/...
---

# T-M9-14 — k6 load tests and SLO baseline
**Model:** opus.

## Why
SLOs in README §8 need an end-to-end measurement, before (baseline) and after (T-M9-49). There are no load scripts
today (gap 07 §7.7).

## Scope
1. `loadtest/seed`: Go program seeding N (default 2 000) synthetic users with 6–36 months of history, logs, BBT/LH for
   a TTC subset, pregnant/postpartum/pill users; idempotent; refuses to run unless the DB name contains `test` or
   `local` (never stage/beta/prod).
2. Token minting through the test-OTP path (local only); k6 scripts: `app_open` (today + month + home + messages +
   banners), `calendar_browse` (12 months back), `log_burst` (POST log → GET today: asserts the write is visible —
   zero stale reads), `analysis_tab`, `fertility_tab`, `soak` (30 min); `thresholds` = README §8.
3. `make loadtest SCENARIO=…` using a pinned k6 docker image; results summarised with server-side metrics
   (`/metrics` scrape before/after).
4. Run the baseline locally (docker MariaDB/Redis, `make run`) and record it in perf-baseline.md ("before").

## Tests
- Seed program unit test (deterministic counts per mode); the k6 scripts lint (`k6 inspect` in docker).

## Out of scope
- Load testing stage or beta (would hit real infrastructure and, for beta, real users).

## Acceptance
- Baseline table "before" recorded with p50/p95/p99, error rate and queries per request per scenario.

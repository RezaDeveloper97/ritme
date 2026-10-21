---
id: T-M9-14b
title: Record the k6 load-test baseline (before table)
milestone: M9
type: backend
status: todo
depends_on: [T-M9-14]
parallel_group: M9-B
touches: [docs/engine-v6/perf-baseline.md,backend-go/loadtest]
skills: []
verify: cd backend-go && go vet ./loadtest/... && grep -q 'Load-test baseline' ../docs/engine-v6/perf-baseline.md
---

# T-M9-14b — Record the k6 load-test baseline (before table)

## Why
T-M9-14 wrote the scripts, the seed tool and the make targets, but couldn't measure: the disk filled up and Docker Desktop hung partway through the session.

## Scope
1. Docker is healthy and the test stack is up (`make test-db-up`). Drop the leftover scratch DB if it exists: `make loadtest-drop LOADTEST_DB=ritme_loadtest_m914`.
2. Run `make loadtest-db && make loadtest-seed`, start `make loadtest-api`, then `make loadtest SCENARIO=<each>` for app_open, calendar_browse, log_burst, analysis_tab and fertility_tab. Soak is optional; it takes 30 min.
3. On the first runs, fix the unverified script assumptions:
   - the 200/201 status of `POST /health-logs`;
   - `data.notes` in `GET /health-logs/:date`;
   - the Idempotency-Key header;
   - the `/cycle/month/:y/:m?view=calendar` route;
   - the `/messages/selected?placement=status_banner` and `/analysis/*` paths;
   - the `grafana/k6:1.3.0` image pin.
4. Add a "Load-test baseline (before)" table to `docs/engine-v6/perf-baseline.md`: p50/p95/p99, error rate and queries per request, per scenario and endpoint, with the machine and load noted.

## Acceptance
- The table is in the doc, and every scenario has run once against a local API.
- Never run against stage, beta or production.

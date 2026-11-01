---
id: T-M10-31
title: Automatic metric alerts on the dashboard
milestone: M10
type: backend
status: done
depends_on: [T-M10-30, T-M10-24]
parallel_group: M10-D
touches: [backend-go/internal/metrics/alerts.go, backend-go/internal/metrics/metrics.go, backend-go/internal/admin/dashboard, backend-go/db/queries/metrics, docs/engine-v6/jobs.md, backend-go/internal/http/routes_admin_core.go, docs/go-migration/admin-api.md]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/metrics/... ./internal/admin/dashboard/... && make test-int PKG=./internal/metrics/...
---

# T-M10-31 — Automatic metric alerts on the dashboard
**Model:** opus.

## Why
SPEC §5.1: alerts when OCR correction > 12 %, message CTR drops > 30 % against its 7-day mean, users with
`requires_user_input` keep rising for more than 7 days, or engine MAE > 2.5 days. `Payload.Alerts` is always empty
today (`metrics.go:72-77,150`). D-180.

## Scope
1. After each snapshot (job chain), evaluate the enabled `metric_alert_rules` (T-M10-25 seed): open an alert when the
   condition holds, resolve it when it no longer does; one open alert per (rule, subject) — subject = message rule and
   version for CTR, '' otherwise; CTR only for versions above `min_volume` shown per day.
2. `safety_selftest_failed` opens an alert from the T-M10-24 run result.
3. The dashboard payload lists open alerts (newest first) and the last 10 resolved; admin API to change a rule's
   threshold/enabled (dashboard manage capability, audited).

## Tests
- Each rule at its boundary: OCR 12 % no / 12.01 % yes; CTR drop 29.9 % no / 30.1 % yes (and below min volume no);
  RUI rising 7 days no / 8 days yes; MAE 2.5 no / 2.51 yes.
- Re-evaluation with the same data does not open a second alert; recovery resolves it.

## Out of scope
- Notifications outside the dashboard.

## Acceptance
- Verify passes; PROGRESS.md cites D-180.

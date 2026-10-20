---
id: T-M10-30
title: KPI completion from engine snapshots and activity days; OKR status rule
milestone: M10
type: backend
status: todo
depends_on: [T-M10-25, T-M10-06, T-M9-39]
parallel_group: M10-D
touches: [backend-go/internal/metrics, backend-go/internal/okr, backend-go/internal/auth/guard.go, backend-go/internal/auth/activity, backend-go/internal/cycle/revision, backend-go/db/queries/metrics, docs/engine-v6/jobs.md, docs/engine-v6/kpi.md]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/metrics/... ./internal/okr/... && make test-int PKG="./internal/metrics/... ./internal/okr/..."
---

# T-M10-30 — KPI completion from engine snapshots and activity days; OKR status rule
**Model:** opus.

## Why
SPEC §5.6 defines each KPI. Today (gap 06 §2.4): DAU/MAU are a proxy (token refresh ∪ legacy log ∪ analytics event);
loggers count only the legacy daily sheet; the activation funnel reads opt-in analytics only (biased); MAE replays
stored lengths instead of the engine's real predictions and ignores pregnancy, contraception and postpartum;
confidence and data-quality distributions and "requires_user_input resolved in 7 days" have no data; AI cost is not
divided by Plus revenue; the KPI screen computes live on request; the OKR status rule is `progress ≥ 0.8 × elapsed`
instead of `progress ≥ elapsed − 10 points` (`okr.go:27-28,82-90`). T-M9-39 now stores daily engine snapshots and a
prediction log. DECISIONS.md A-05, A-06, A-07; D-179.

## Scope
1. Activity days (new `internal/auth/activity`): `Guard.RequireUser` (`internal/auth/guard.go`) marks `session = 1` on
   a user's first request of the user-local day (Redis `SET NX ritme-go:seen:{uid}:{day}` 26 h, then an async upsert
   through a small buffered writer that drops, never blocks, when full); the revision bump functions (T-M9-30,
   T-M10-28) mark `logged = 1` the same way. No extra query on later requests that day; Redis down → mark through the
   writer without the SET NX (idempotent upsert).
2. KPIs in the `metric_snapshots` job (one row per Tehran day, recomputed on its T-M9-11 schedule), never computed on
   the KPI request:
   DAU/MAU/stickiness from sessions; loggers per day; loggers ≥ 15 days in the month; activation funnel from server
   facts (OTP verified, onboarding completed, first log, logged on day 7 and day 30) with the install step labelled
   "from opt-in analytics"; retention Dn with a ±1 day window; engine MAE and ±2-day accuracy from
   `engine_prediction_log` (`kind = first`, `basis_cycles ≥ 2`, excluding pregnancy, hormonal contraception and
   postpartum; A-05); confidence and data_quality distributions over active users (A-06) from `engine_daily_snapshots`;
   share of `requires_user_input` episodes resolved within 7 days by a user input; trial conversion and MRR (kept); AI
   cost / Plus revenue (USD cost × an admin-set FX rate ÷ IRR revenue of the month); OCR correction rate; message CTR
   per version (T-M10-06 rollup); family KPIs as `null` until T-M11-15. Small-cohort suppression as in today's code.
3. The KPI endpoint reads the latest snapshot plus the previous period for deltas.
4. OKR: on-track iff `progress ≥ elapsed% − 10` (A-07), done at 100 %; the KR catalogue covers every §5.6 metric key
   (adds loggers_15d, low_confidence_share, ai_cost_share, ocr_correction_rate, message_ctr, engine_mae,
   accuracy_2d, rui_resolved_7d); evaluation runs after the snapshot job.
5. `docs/engine-v6/kpi.md`: one row per KPI — definition, source table, job, suppression rule.

## Tests
- Activity: two requests the same local day → one row, `session = 1`; a log write → `logged = 1`; a Berlin user's day
  boundary.
- MAE fixture: predictions 07-29 (first) vs actual 08-02 → error 4; a pregnancy cycle excluded; basis 1 excluded.
- RUI: an episode resolved on day 6 counts, on day 8 does not.
- Retention ±1 window; funnel counts on fixtures; AI cost share with an FX rate.
- OKR boundaries at 20 / 50 / 100 % elapsed (progress 10 vs 9.9 at 20 %).
- The KPI endpoint runs ≤ 2 queries (snapshot reads).

## Out of scope
- Alerts (T-M10-31), dashboards (T-M10-34), family KPIs (T-M11-15).

## Acceptance
- Verify passes; kpi.md complete; PROGRESS.md cites D-179.

# KPI definitions (SPEC v6 §5.6)

T-M10-30, D-179 (DECISIONS.md A-05, A-06, A-07). Code: `backend-go/internal/metrics` (job + definitions),
`backend-go/internal/auth/activity` (activity days), `backend-go/internal/okr` (status rule), queries in
`backend-go/db/queries/metrics/kpi.sql` and `activity.sql`.

Every KPI is computed by the **`admin_metrics:snapshot`** job (queue low, `15 * * * *` + at start, `Unique(55m)`;
[`jobs.md`](jobs.md)) and stored in **one `metric_snapshots` row per Tehran day** (payload `version: 2`; the `kpi`
block holds every funnel window and retention variant). `GET /api/admin/v1/kpi?days=&event=` reads the newest row and
the newest row on or before `day − days` for the deltas — **≤ 2 queries, nothing computed on the request**.
`GET /api/admin/v1/overview` reads the newest row.

**Suppression (k = 10, `MinCohort`)** — unchanged from B-N9-03/08: a group (users, cohort members, paying users) under
10 is hidden: the value is `null` and `suppressed: true`; counts of a smaller class inside a large enough group are
hidden but its share is kept. Only aggregates leave the package; no row carries a user id.

**Days.** "Yesterday" / "today" are Tehran days of the job's clock. Activity days are the **user-local** day (the
request's `X-Timezone`, else the stored `user_profiles.timezone`, else Asia/Tehran). Signup cohorts are keyed on the
Tehran signup day (`users.created_at`).

## Activity days (`user_activity_days`)

| Column | Written by | When |
|---|---|---|
| `session = 1` | `auth.Guard.RequireUser` → `activity.Marker` | first authenticated request of the user-local day |
| `logged = 1` | `revision.Bump` / `revision.BumpAnalysis` (every engine- or analysis-input write, T-M9-30 / T-M10-28) | first such write of the day on a user request (admin edits and jobs carry no marker and never count) |

The request only checks an in-process set and queues the mark (no query, no Redis call; a full 4 096-slot buffer drops
it, `activity_marks_dropped_total`). The writer claims `ritme-go:seen:{uid}:{day}` (`…:logged:…` for logs) with
`SET NX` for 26 h and upserts (`INSERT … ON DUPLICATE KEY UPDATE`); Redis down → upsert without the claim. A write in
a transaction that later rolls back still marks the day.

## Definitions

| KPI (payload key → OKR key) | Definition | Source tables | Window | Suppression |
|---|---|---|---|---|
| DAU (`dau` → `dau`) | users with `session = 1` yesterday; `prev` = the day before | `user_activity_days` | 1 day | none (a count) |
| MAU (`mau` → `mau`) | distinct users with `session = 1` in the 30 days ending yesterday; `prev` = the 30 days before | `user_activity_days` | 30 days | none |
| Stickiness (`stickiness_pct`) | DAU / MAU × 100 | — | — | none; null when MAU = 0 |
| Loggers (`loggers` → `loggers`) | users with `logged = 1` yesterday; daily `series[].loggers` | `user_activity_days` | 1 day | none |
| Loggers ≥ 15 days (`loggers_15d` → `loggers_15d`) | users with `logged = 1` on ≥ 15 of the 30 days ending yesterday, % of MAU | `user_activity_days` | 30 days | MAU < 10 |
| Signups (`signups_today`) | users created today (Tehran) | `users` | 1 day | none |
| Activation funnel (`kpi.funnels.{30,60,90}` → `activation_first_log_pct`, `activation_log_d7_pct` from the 90-day window) | users who signed up (= verified the OTP) in the window: `install` = sent an opt-in `app_open` (**from opt-in analytics**, `source: opt_in_analytics`), `otp_verified` = all of them, `onboarding_done` = `user_life_profiles.onboarding_completed_at` set, `first_log` = any `logged` day, `log_d7` / `log_d30` = a `logged` day within ±1 of day 7 / 30 (only users ≥ 8 / ≥ 31 days old count). % of the users each step considers | `users`, `user_life_profiles`, `user_activity_days`, `analytics_events` | 30 / 60 / 90 days | step group < 10 |
| Retention Dn (`kpi.retention.{any,log_saved}` → `retention_d7`, `retention_d30` from `log_saved`) | weekly signup cohorts (Saturday weeks, 12 weeks): share of the members who reached day n+1 with activity (`any` = any activity day, `log_saved` = `logged = 1`) on a day in [n−1, n+1]; n ∈ 0, 1, 7, 14, 30, 60 | `users`, `user_activity_days` | 12 cohorts | cohort < 10 (row), mature members < 10 (cell) |
| Engine MAE (`engine.mean_error_days` → `engine_mae`, also `engine_mean_error_days`) | mean \|`error_days`\| of `kind = first` predictions with `basis_cycles ≥ 2` whose next real start (`actual_start`) fell in the last 180 days, excluding cycles with a daily snapshot in [cycle_start, actual_start] in `pregnancy` / `postpartum` mode or with the `hormonal_contraception_active` warning (A-05) | `engine_prediction_log`, `engine_daily_snapshots` | 180 days | users < 10 |
| ±2-day accuracy (`engine.within_2_days_pct` → `accuracy_2d`, also `engine_within_2_days_pct`) | share of the same predictions with \|error\| ≤ 2 | same | 180 days | users < 10 |
| Engine status (`engine.status`) | `healthy` when MAE ≤ 2.5 and accuracy ≥ 70 %, else `watch` | — | — | null when suppressed |
| Confidence distribution (`kpi.distributions.confidence`, `engine.confidence` → `low_confidence_share`) | confidence classes (high, medium, low, unknown) of the active users (A-06: those the nightly job snapshotted) on the newest snapshot day ≤ today | `engine_daily_snapshots` | 1 day | total < 10 (all); class < 10 (count hidden, share kept) |
| Data-quality distribution (`kpi.distributions.data_quality` → `low_data_quality_share` = poor + insufficient) | data_quality classes (good, partial, poor, insufficient), same users and day | `engine_daily_snapshots` | 1 day | as above |
| requires_user_input resolved in 7 days (`kpi.rui_resolved_7d` → `rui_resolved_7d`) | episodes = runs of snapshot days with a user-input warning (§2.33) that started 7–36 days ago; resolved = the first following snapshot day without one has a newer `engine_rev` (an engine-input write by the user) and is ≤ 7 days after the start | `engine_daily_snapshots` | 30 days of starts | users < 10 |
| Trial conversion (`trial_to_paid` → `trial_to_paid_pct`) | trials that ended in the last 30 days followed by a purchase after the trial began (kept) | `plus_trials`, `plus_subscriptions` | 30 days | trials < 10 |
| MRR (`mrr_toman` → `mrr_toman`) | active purchased subscriptions: (subtotal − discount) / duration months, excluding VAT, in toman (kept) | `plus_subscriptions`, `plus_invoices` | now | none |
| AI cost share (`kpi.ai_cost` → `ai_cost_share`) | (lab_analysis + assistant `cost_micros` / 10⁶ USD) × the admin-set rate (`plus_settings.ai_cost_usd_irr_rate`, IRR per USD) ÷ IRR revenue (subtotal − discount of invoices paid) in the last 30 days, × 100 | `ai_usage_logs`, `plus_invoices`, `plus_settings` | 30 days | cost users < 10 or payers < 10; null without a rate |
| OCR correction rate (`kpi.ai.ocr_correction_pct` → `ocr_correction_rate`) | share of extracted lab values the user edited on the confirmation screen | `lab_markers`, `lab_reports` | 30 days | users < 10 |
| Message CTR (`kpi.message_ctr` → `message_ctr` = overall) | clicked / shown per rule version (50 most shown) and overall, from the daily rollup (T-M10-06; events `impression`, `click`) | `message_event_daily` | 30 days ending yesterday | version with < 10 shown; overall shown < 10 |
| Other AI indicators (`kpi.ai`) | labs/day, cost per analysis, assistant messages/day, referral % (kept from B-N9-08) | `lab_reports`, `ai_usage_logs`, `assistant_*` | 30 days | users / conversations < 10 |
| Family (`kpi.family`) | `null` until T-M11-15 | — | — | — |

## OKR (SPEC §5.7, A-07)

- Progress = (current − baseline) / (target − baseline), clamped 0–1.
- Status: **done** at 100 %; **on track** iff progress ≥ elapsed % of the period − 10 points; otherwise **at risk**;
  **no data** while the bound metric is null.
- Bound key results read `metrics.MetricValues` of the newest snapshot; the catalogue (`okr.Metrics`) covers every key
  above. The evaluation runs right after the KPIs in the same job and is stored as the snapshot's `okr` summary.

# Gap analysis 06 — Analysis tab, consent, Plus quotas, admin KPI/OKR/roles (SPEC §4.1–4.3, §5.1–5.8)

Scope: `docs/engine-v6/SPEC.md` lines ~700–901 vs `backend-go/` (+ admin-web where relevant). Read-only; all paths relative to repo root.

Test run (`cd backend-go && go test ./internal/analysis/... ./internal/okr/... ./internal/metrics/... ./internal/consent/...`):

```
ok  	github.com/ritme/backend-go/internal/analysis	(cached)
ok  	github.com/ritme/backend-go/internal/okr	(cached)
ok  	github.com/ritme/backend-go/internal/metrics	(cached)
ok  	github.com/ritme/backend-go/internal/consent	(cached)
?   	github.com/ritme/backend-go/internal/consent/store	[no test files]
```
(`*_int_test.go` need the DB test stack/tag; green here = unit tests only.)

---

## 1. Summary table

| # | Spec item | Status | Evidence |
|---|---|---|---|
| 4.1a | Cycle analyses (length/regularity/flags) | ⚠️ own math | `internal/analysis/cycle.go:95-160` — own FIGO median + shortest-to-longest regularity; enum `not_enough_data` (`cycle.go:18`); not the engine's §2.9/§2.27 output |
| 4.1b | Period intensity (≥2 periods) | ✅ | `period.go:53-54` `Ready()` = `BasedOn >= MinTrendPoints(2)` |
| 4.1c | Symptom pattern (≥3 cycles) | ✅ | `symptoms.go:69-70,147-156`, `analysis.go:49` |
| 4.1d | Mood by phase (≥10 days per phase) | ⚠️ | Cramér's V (`correlations.go:14,153-184`), min group 5 days (`stats.go:36`) — spec: % days per mood per phase, ≥10 days/phase |
| 4.1e | Correlations (group-rate ratio; strong = ≥30 days & ≥2×; min 30 paired days) | ⚠️ different method | φ / Cohen strengths (`stats.go:19-37`: strong ≥0.5, min 20 days, 5/group, 3 outcomes); ratio exists (`correlations.go:51-53`) but not used for strength |
| 4.1f | Weight 7-day MA, Δ7/Δ30, min 3 logs in 14 d | ⚠️ | MA7 + Δ7/Δ30 ✅ (`body.go:9-27,79-83`); readiness = ≥2 points (`body.go:37`), not "3 in 14 days" |
| 4.1g | BP/glucose mean/min–max/category distribution, 2 logs in 2 days | ⚠️ | `vitals.go` exists; readiness = generic `MinTrendPoints` |
| 4.1h | Lab trend (2 labs) | ⚠️ | only a hub card via `LabsHub` (`handlers.go:175-181`); no dedicated per-marker trend analysis endpoint in `internal/analysis` |
| 4.1i | BBT (≥10 temps/cycle, result feeds engine) | ⚠️ | 6+3=9 readings (`ttc.go:723-725`); chart ready at 2 points (`ttc.go:449`); feeds engine via `/fertility/bbt` rule (D-45) |
| 4.1j | Pregnancy weight vs IOM (pre-preg BMI + 2 logs) | ✅/⚠️ | `pregnancy.go`, `pregnancy_thresholds.go:28-33` (D-46); baseline fallback to `user_profiles.weight` |
| 4.1k | EPDS score+trend; **An_Hub_Post** (postpartum hub: EPDS trend, bleeding decline, breastfeeding, sleep, child growth, weight) | ❌ | no postpartum analysis in `internal/analysis`; only `/postpartum/epds` history (`http/routes_postpartum.go:26-27`) |
| 4.1l | Monthly report (10 logged days) | ⚠️ | `littleDataDays = 7` (`monthly.go:25`) |
| 4.1m | "co-occurrence, not cause" text | ✅ | `not_causal: true` (`correlations.go:283,297`, `handlers.go:247`) — verify copy string says «هم‌زمانی در داده‌های تو، نه علت» |
| 4.1n | Top finding = strength × recency | ❌ | fixed priority list (`finding.go:30-104`): FIGO verdict → pattern; no scoring, no recency |
| 4.1o | "log **N more days**" when insufficient | ⚠️ | nudge counts **cycles** (`finding.go:98-99` `needed = MinPatternCycles − BasedOn`) |
| 4.1p | Server-side, nightly job or on tab-open if data changed; past edits invalidate cache | ❌ | computed per request (`handlers.go:66-110`), no analysis cache/job; only the cycle day/month payload is cached (`cycle/service/service.go:189-278`) |
| 4.1q | Free vs Plus split of analyses | ⚠️ | `plus.DeepAnalysis` gate (`handlers.go:78-82`); 1-year range not gated (`range.go:23`) — spec: «بازه ۱ ساله» is Plus |
| 4.2a | `consent(user_id,type,version,granted_at,revoked_at,scope_json)` | ⚠️ | `user_consents` (`db/migrations/00012_privacy_support.sql:20-31`) + `version` (`00032_ai_platform.sql:16`); **no scope_json, one mutable row per (user,code)** (UNIQUE) → no history |
| 4.2b | Terms + health-data processing at signup | ⚠️ | `users.terms_accepted_at` only (`auth/store/models.go:480`); no version, not in consent model |
| 4.2c | AI lab analysis consent | ✅ | `consent.AILabAnalysis` (`consent/catalog.go:17`); enforced `labs/extract.go`, `ai/access/access.go` (403 `consent_required`, version-aware) |
| 4.2d | Assistant consent, per-item switches | ✅/⚠️ | `AssistantProfile`, `AIAssistant` (`catalog.go:18,20`), enforced `assistant/service.go`; per-item scope lives in assistant tables, not `scope_json` |
| 4.2e | Doctor share per booking/visit | ✅ separate model | `telemed_booking_consents` (`00050_telemed_bookings.sql:83-95`), gate `telemed/consent.go:14-16` `AuthorizeShare` |
| 4.2f | Family/companion per scope & level | ✅ separate model | `companion/record_scope.go`, `companion_audit_logs` (`00026_companions.sql:118`) |
| 4.2g | PDF link 7 days, revocable | ✅ | `sharelinks/sharelinks.go:33` `LinkTTL = 7*24h`; (doctor summary code 24 h `summary.go:35`) |
| 4.3a | Central entitlement/quota service | ✅ | `plus/entitlements.go` single `definitions` table + `Resolve`; `plus/service.go:218-310` `Entitlement/Consume/Refund`; callers listed in §2.3 |
| 4.3b | AI lab analysis: 1/month free, 10 Plus | ⚠️ | `LabAI: Plus 10, Free: locked` (`plus/entitlements.go` definitions) — free gets 0 |
| 4.3c | Assistant 5/month free; Plus "unlimited with fair cap" | ⚠️ | free 5 ✅; Plus `unlimited` (no fair-use cap; only global AI cost caps in `ai/access`) |
| 4.3d | Never gate emergency / med reminders / EPDS | ✅ | no `plus.` reference in `emergency/`, `reminders/`, `postpartum/`, `safety/`, `care/` |
| 5.1a | Dashboard cards DAU/MAU/stickiness/signups/Plus/trial conv/MRR + prev | ✅ | `metrics/metrics.go:87-103,147-275` snapshot (`00064_metric_snapshots.sql`) |
| 5.1b | Users by mode, engine health, OKR summary | ✅/⚠️ | modes ✅ (`metrics.go` Compute); engine health is a **backtest**, not real predictions (`metrics.go:298-360`) |
| 5.1c | Automatic alerts (OCR >12 %, CTR drop >30 % vs 7-d mean, RUI >7 d rising, MAE >2.5 d) | ❌ | `Alerts: []Alert{}` always (`metrics.go:72-77,150`) |
| 5.6 | KPI definitions | ⚠️ partial | see §2.4 table |
| 5.6-snap | Daily engine-output snapshot per active user | ❌ | `kpi.go:111` note "data-quality class is not stored per user"; `metrics.go:66-67` confidence "not stored anywhere" |
| 5.6-pred | Prediction log for MAE/±2 d | ❌ | none; MAE = median-of-prior-lengths replay over `cycle_histories` (`metrics.go:298-360`) |
| 5.2a | RT-xxxx pseudonym, masked mobile | ⚠️ | `RefCode = fmt.Sprintf("RT-%04X", id)` (`admin/users/v2.go:30-31`) = reversible hex of DB id; `MaskMobile` ✅ (`v2.go:33-41`) |
| 5.2b | Columns mode, data_quality, main engine warning | ❌ | list has plan/last activity/d30/churn only (`v2.go:180-207`) |
| 5.2c | Churn rule (high ≤50 % or 7 d inactive; medium ≤75 %) | ⚠️ wrong thresholds | `churnOf` high <30 %, medium <70 %, no 7-day-inactivity rule (`v2.go:119-140`) |
| 5.2d | Saved segments | ❌ | no table/endpoint |
| 5.2e | User detail: today's engine output, cycle-length history, messages with triggering rule | ⚠️ | overview has subscription/consents/support/messages (`v2.go:299-370`); no engine output / cycle history; message rows lack rule reason |
| 5.2f | Health data hidden; clinical-support 30-min with reason+ticket; audited | ✅ | `admin_health_access_requests` (`00060_admin_rbac_audit.sql`), `httpadmin/health.go:14-50` (real clock), callers `users/users.go:170`, `users/v2.go:268`, `rbac/rbac.go:636` |
| 5.2g | Exports anonymised only | ✅ | `v2.go:212-235` export: ref/plan/month/days/churn only |
| 5.7 | OKR progress clamp 0–100 % | ✅ | `okr/okr.go:57-66` |
| 5.7 | On-track if progress ≥ elapsed − 10 pts | ⚠️ different | `progress >= elapsed*0.8` (`okr.go:27-28,82-90`) |
| 5.7 | KR bound to any §5.6 metric | ⚠️ | catalogue of 11 keys (`okr.go:37-42`); missing loggers ≥15 d/month, low-confidence share, AI cost % revenue, OCR, CTR |
| 5.8a | Roles list | ✅ | `httpadmin/permissions.go:21-31,84-88` (all 7 + super/editor/doctor/instructor_moderator) |
| 5.8b | Role matrices match spec | ⚠️ | `permissions.go:106-140`: product_manager/marketing have `messages: manage` without type restriction except approval; engineering `config: manage` (spec: view thresholds); analyst `users: view` (spec KPI + anonymous export only) |
| 5.8c | Safety messages edited/published only by medical advisor | ⚠️ | approval gated by `CapClinical` (`admin/messages/rules.go:297-313`) — verify create/edit of `type=safety` also requires it |
| 5.8d | Threshold changes need 2 medical-advisor approvals | ✅ | `admin/clinicalcfg/clinicalcfg.go:7-8,42,200` |
| 5.8e | Immutable audit log | ⚠️ | app-level append-only (`httpadmin/audit.go:25-68`, no DELETE query found); no DB-level guard (no trigger/revoked grants/hash chain); row loss only logged (`audit.go:64-66`) |

---

## 2. Findings

### 2.1 §4.1 Analysis tab
- **Routes** (all Go-only, per request): `http/routes_analysis.go:41-47` (summary/cycle/period/symptoms/correlations/body/monthly), `routes_analysis_ttc.go:34-35`, `routes_analysis_pregnancy.go:35`. No postpartum hub.
- **Request path cost**: every call runs `plus.Entitlement` + `cycle.Load(today,today,today)` (all histories) + `healthlog.Range(from−40d … today)` + vitals (`handlers.go:66-110`). `range=all` reads the whole log history per tab open; the hub (`Summary`) recomputes cycle+period+symptoms+correlations (`summary.go:120-183`). No Redis, no memo, no job.
- **Shared engine vs own math**: medians/plausibility come from `internal/cycle/insights` (`cycle.go:109,153-154`), but the regularity verdict, FIGO status, phase split (`analysis.go:52-67`: ovulation = next start − 14, 6-day window) are re-implemented in `internal/analysis`. Spec says length/regularity/flags come "from the engine (§2.9, §2.27)" — so the analysis can disagree with `cycle_view.cycle_regularity` / `data_quality` shown on Today. Enum drift: analysis `not_enough_data` vs engine vocabulary (see gap report on §2.9/§2.27).
- **Thresholds vs spec** (spec value → code):
  - correlations min paired days 30 → 20 (`stats.go:35`); strength rule "≥30 days & ratio ≥2×" → Cohen |φ| ≥ 0.5.
  - mood-in-phase ≥10 days per phase → 5 days per group.
  - weight 3 logs in 14 days → ≥2 points.
  - BBT ≥10 temps in the cycle → 9 (6 baseline + 3 high).
  - monthly 10 logged days → 7 (`monthly.go:25`).
  - regularity "3 cycles in 12 months" → 3 counted cycles in the selected *range* (`cycle.go:155`), not a 12-month window.
- **Top finding**: deterministic priority, not `strength × recency`; nudge in cycles not days.
- **Missing**: An_Hub_Post (EPDS trend, bleeding decline, breastfeeding, mother/baby sleep, child growth, weight); dedicated labs trend analysis.

### 2.2 §4.2 Consent
- `user_consents` = one row per (user, code), mutated in place (`UNIQUE user_id,consent`). Granting v2 overwrites v1's timestamps → no proof of what was accepted when. No `scope_json`.
- Gate is solid for AI: `consent.Service.Require` → 403 `consent_required` with `outdated` reason, version-checked; used by `labs/extract.go`, `assistant/service.go`, `healthrecord/extract/service.go`, `ai/access/access.go`.
- Doctor share, family, share links each have their own consent-like tables (telemed_booking_consents, companions scopes, share links with TTL) — functionally compliant but not in the unified model; admin user detail shows only `user_consents` rows (`v2.go:319`).
- Signup consent: only `users.terms_accepted_at`, no version, no separate "health-data processing" consent.

### 2.3 §4.3 Entitlements
- Central: `plus/entitlements.go` (`definitions`, `ResolveOne`, `Resolve`) + `Service.Entitlement/Consume/Refund` (`plus/service.go:218-310`). Callers: `ai/access/access.go`, `voicelog/handlers.go`, `analysis/handlers.go`, `analysis/pregnancy_handlers.go`, `http/routes_{labs,assistant,sharelinks,telemed,healthrecord_extract,ai,plus,voicelog}.go`. Good — no ad-hoc tier checks found.
- Gaps: LabAI free = locked (spec 1/month); Plus assistant has no fair-use cap; 1-year range of analysis free; Plus "BBT with confirmation / LH-based prediction / prediction source+confidence" are not keys (check whether engine output hides `confidence` for free users — spec lists "source and confidence of prediction" as Plus).
- Safety: no gating found on emergency, med reminders, EPDS. Add a regression test that asserts this (see tasks).

### 2.4 §5.1 / §5.6 KPIs

| KPI | Status | Source |
|---|---|---|
| DAU/MAU | ⚠️ proxy | `activeUnion` = token refresh ∪ health log ∪ analytics event (`metrics.go:135-137`); spec = ≥1 *session* |
| Loggers per day | ✅ | `daily_health_logs` per day (`metrics.go` series) — but only the legacy sheet, not `health_log_entries`/vitals/etc. |
| Activation funnel install→OTP→onboarding→first log→D7→D30 | ⚠️ | `kpi.go:146-198` from `analytics_events` (opt-in `anonymous_stats` only → biased, `00065_analytics_events.sql:9`) |
| Retention Dn (±1) | ⚠️ | `kpi.go:200-303` exact offsets 0/1/7/14/30/60; verify ±1 window |
| MAE / ±2 d | ⚠️ wrong source | replay `median(prev ≤3 lengths)` vs real length on `cycle_histories` (`metrics.go:298-360`); not `predicted_next_period_start` as the engine emitted it; ignores pregnancy/contraception/postpartum, roll-forward, onboarding anchor |
| confidence / data_quality distribution | ❌ | not stored (`metrics.go:66-67`, `kpi.go:111`) |
| requires_user_input resolved ≤7 d | ❌ | no history of the flag |
| Trial conversion | ✅ | `metrics.go` (trials ended in last 30 d with a later purchase) |
| MRR | ✅ | `metrics.go` subtotal−discount / duration_months |
| AI cost | ⚠️ | cost per analysis USD (`kpi.go:305-348`); spec = AI cost / Plus revenue (needs FX/IRR) |
| OCR correction | ✅ | `lab_markers.source='edited'` share (`kpi.go`) |
| Message CTR | ✅ per rule in messages admin (`admin/messages/rules.go:386-406`), not in KPI/alerts |
| Family KPIs | ❌ | only companion count in snapshot |
| Alerts | ❌ | always empty |

Storage: `metric_snapshots` = one JSON doc per Tehran day (`00064`), refreshed **hourly by a goroutine started in every API process** (`http/routes_admin_core.go:56-66`, `metrics.go:402-415`) — duplicated work per replica, no lock; KPI screen is **computed live on request** (`kpi.go:99`).

### 2.5 §5.2 User monitoring
- RT code = hex of autoincrement id → trivially reversible/enumerable; acceptable as a support reference but not "pseudonymous" in the analytics sense.
- Directory query (`v2.go:75-86`) aggregates all users' 30-day `daily_health_logs` in a derived table on every page/count request — O(all users) per page, and `daily_health_logs` has only `UNIQUE(user_id, log_date)` (`00001_baseline.sql:214`), so `WHERE log_date >= ?` scans.
- Missing: mode, data_quality, engine main warning columns + filters; saved segments; engine output today + cycle history in detail; churn thresholds wrong; last-activity = `users.updated_at` proxy.

### 2.6 §5.7 OKR
- Progress formula ✅ incl. decreasing targets. Status rule differs: code `progress ≥ 0.8 × elapsed`; spec `progress% ≥ elapsed% − 10`. At 50 % elapsed: code needs 40 %, spec needs 40 % (equal); at 100 %: code 80 %, spec 90 %; at 20 %: code 16 %, spec 10 %. Small but explicit spec mismatch.
- Live values only from the latest snapshot (+ retention) (`okr.go:158-214`).

### 2.7 §5.8 Roles & audit
- Roles present. Default matrices broader than the spec in places (see table). `CapMessages` is a single capability — the spec splits product/marketing vs clinical/safety. Need a per-type check on create/edit/publish (approval is gated; authoring of `type=safety` must be too).
- Audit: append-only by convention. Recommend DB triggers `BEFORE UPDATE/DELETE ON admin_audit_logs → SIGNAL` (MariaDB supports), plus `prev_hash/row_hash` chain, and failing closed for health-access reads when the audit insert fails (currently logged and continues, `audit.go:64-66`).

### 2.8 Performance
- Heavy work on the request path: analysis tab (every screen), KPI screen (`kpi.go` live), user directory (full 30-day aggregate per page), hourly snapshot per replica.
- Missing indexes: `daily_health_logs(log_date, user_id)`; `cycle_histories(period_start_date)` (engine-health scan filters by date across all users); `users(created_at)` (signups/cohorts); `plus_subscriptions(status, ends_at)` check; `message_events(event, created_at)` exists only with `rule_id` prefix (`00061_messages_v2.sql:68`) — fine for per-rule CTR, not for global alert scans.
- Reusable pieces: cycle engine Redis cache with input digest (`cycle/service/service.go:189-215`, `cycle/cache/cache.go` TTL 24 h, key `cycle-engine:{uid}:{ver}:{locale}:{today}:{scope}:{md5(inputs)}`), asynq queue (`platform/queue/queue.go`, in-process worker).

---

## 3. Ambiguities needing a user decision (recommended default)

1. **Analysis regularity source** — keep analysis' FIGO shortest–longest rule or consume the engine's `cycle_regularity`? *Default: consume engine output for the verdict; keep FIGO ranges only as the displayed reference band.*
2. **Correlation method** — spec ratio rule (≥30 days, ≥2×) vs current φ/Cohen. *Default: spec rule for `strength` and readiness; keep φ as an extra field.*
3. **Free lab-AI quota** — spec 1/month vs canvas "locked". *Default: follow spec (Free `{Enabled, Limit:1}`), admin-tunable later.*
4. **Plus assistant fair cap** value. *Default: 300 messages/month, config-driven, soft message at cap.*
5. **Consent history** — migrate `user_consents` to append-only rows or add a `consent_events` ledger beside it? *Default: add `consent_events` ledger (append-only, `scope_json`), keep `user_consents` as current-state projection (no API break).*
6. **Prediction log semantics** — which prediction counts: the first `predicted_next_period_start` emitted after the previous start, or the last one before the actual start? *Default: log both (`horizon` = first emitted and day −1); MAE KPI uses the "first emitted" one per spec wording, only when `basis_cycles ≥ 2`.*
7. **Active user** for engine snapshot — *Default: any activity (session/log/event) in last 30 days.*
8. **OKR on-track rule** — switch to spec `elapsed − 10 pts`. *Default: yes.*
9. **Churn**: apply spec thresholds incl. "7 days no activity → high". *Default: yes.*
10. **RT code**: keep id-hex or keyed HMAC? *Default: `RT-` + first 6 base32 chars of HMAC(user_id, server secret), stored in a column for search.*
11. **Postpartum analysis hub** — in M9 or a later milestone? *Default: in M9 (backend only), behind module kill-switch.*

---

## 4. Architecture & performance

### 4.1 Daily engine-output snapshot (`engine_daily_snapshots`)
Purpose: confidence/data_quality distribution, requires_user_input aging/resolution, admin user list columns, alert "RUI > 7 d rising".

```sql
CREATE TABLE engine_daily_snapshots (
  id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  user_id bigint unsigned NOT NULL,
  day date NOT NULL,                       -- Tehran target_date
  engine_version smallint unsigned NOT NULL,
  life_mode varchar(24) NOT NULL,
  main_phase varchar(24) NULL,
  confidence varchar(8) NULL,              -- high|medium|low
  data_quality varchar(16) NULL,
  requires_user_input tinyint(1) NOT NULL DEFAULT 0,
  rui_since date NULL,                     -- first consecutive day the flag was true
  main_warning varchar(48) NULL,
  warnings_json json NULL,                 -- codes only
  basis_cycles tinyint unsigned NULL,
  input_digest char(32) NOT NULL,          -- same md5 as cycle cache key inputs
  created_at timestamp NULL,
  UNIQUE KEY (user_id, day),
  KEY (day, confidence), KEY (day, data_quality), KEY (day, requires_user_input, rui_since)
);
```
- **No dates of predicted periods** stored here → nothing resembling period logs or `future_cycles` (§2.40 compliant: codes/enums only).
- Writer: asynq periodic job `engine.snapshot.daily` at 03:30 Tehran, enqueues per-user chunk tasks (1 000 users/task) for active users; each calls `cycleservice.Load(...).Day(today)` (hits the Redis engine cache if warm). Single-run guarded by asynq unique task / Redis lock (fixes the per-replica goroutine pattern too).
- Retention: 400 days rolling purge; aggregates beyond that live in `metric_snapshots`.
- Health-data classification: confidence/data_quality/warnings are engine meta, not raw health values, but still per-user → admin reads only aggregated (k ≥ 10) except the user-list columns (allowed by §5.2: data_quality + main warning are listed columns).

### 4.2 Prediction log (`cycle_prediction_log`)
Purpose: MAE / ±2-day accuracy as defined in §5.6.

```sql
CREATE TABLE cycle_prediction_log (
  id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  user_id bigint unsigned NOT NULL,
  cycle_start date NOT NULL,              -- the confirmed start the prediction was made from
  predicted_start date NOT NULL,          -- predicted_next_period_start as emitted
  kind varchar(8) NOT NULL,               -- 'first' | 'last'
  emitted_on date NOT NULL,
  basis_cycles tinyint unsigned NOT NULL, -- real cycles used (spec: count only if >= 2)
  confidence varchar(8) NULL,
  engine_version smallint unsigned NOT NULL,
  actual_start date NULL,                 -- filled when the next real start is logged
  error_days smallint NULL,               -- actual - predicted (signed)
  excluded_reason varchar(24) NULL,       -- pregnancy|hormonal_contraception|postpartum|edited_away
  resolved_at timestamp NULL,
  UNIQUE KEY (user_id, cycle_start, kind),
  KEY (resolved_at), KEY (actual_start)
);
```
- **§2.40 compliance**: lives in an analytics-only table owned by `internal/metrics` (or new `internal/enginelog`); never read by the engine, never joined into `period_logs`/`cycle_histories`, never used as an effective value; one row per *past* cycle start (not a list of future cycles); only the *next* start, which the engine already emits. Document in a header comment + a guard test that `internal/cycle/...` does not import the package.
- Writers: (a) `first` row upserted (INSERT IGNORE) by the daily snapshot job / on first day computation after a new start; (b) `last` row overwritten on each daily run until resolved; (c) resolver hook on period-start create/edit/delete: set `actual_start`/`error_days`; if the anchor start is deleted/edited → mark `excluded_reason='edited_away'`.
- Retention: 24 months; aggregated nightly into `metric_snapshots.payload.engine`.
- MAE = mean |error_days| where `kind='first' AND basis_cycles >= 2 AND excluded_reason IS NULL AND actual_start` in window; replace the backtest (`metrics.go:298-360`) but keep it as a fallback label "backtest" until ≥ k samples.

### 4.3 Alerts
`metric_alert_rules(key, threshold, comparator, window, enabled)` seeded with the 4 spec alerts + `metric_alerts(key, day, value, threshold, state, opened_at, resolved_at)`; evaluated after each snapshot refresh; shown in `Payload.Alerts`. CTR drop needs per-rule daily CTR: nightly rollup `message_daily_stats(rule_id, variant, day, shown, clicked, dismissed, goal)` instead of scanning `message_events`.

### 4.4 Analysis precompute + cache
- Key: `analysis:{uid}:{analysis_version}:{locale}:{today}:{endpoint}:{range}:{plus}:{md5(inputs)}` where inputs = cycle snapshot digest (reuse `Snapshot.cacheKey` inputs) + `max(updated_at), count(*)` of health_log_entries/vitals/labs in the read window (cheap index-only query) + clinical-config version for FIGO/IOM.
- Past edits invalidate automatically via the digest; additionally bump a per-user `analysis:gen:{uid}` counter on any log write (cheap, makes the digest query optional).
- Nightly job warms `summary` + default range for users active in last 7 days; on-tab-open is cache-aside (spec "or on tab open if data changed"). TTL 24 h.
- Locale: cache the *data* document pre-localization if feasible; otherwise include locale in key (current cycle cache does).

### 4.5 Indexes / jobs hygiene
- Add `daily_health_logs (log_date, user_id)`, `cycle_histories (period_start_date)`, `users (created_at)`, `analytics_events` OK.
- Move `metrics.Loop` from per-process goroutine to asynq periodic task with uniqueness; same for OKR evaluation.
- User directory: precompute `admin_user_activity(user_id, last_activity, d30, a14, b14, churn, mode, plan, data_quality, main_warning, updated_on)` nightly (+ incremental on log write optional); directory queries this table with indexes on `(churn)`, `(plan)`, `(mode)`, `(last_activity)`.

---

## 5. Proposed task breakdown (backend first; admin-web separate; no Android, no prod cutover)

**M9-AK-01 — Analysis thresholds & methods aligned to §4.1** · touches `internal/analysis/{stats,correlations,body,monthly,ttc,cycle,finding}.go`, `analysis` translation namespace, goldens · deps: decision 1/2 · tests: table tests per threshold edge (29/30 days, ratio 1.9/2.0, 9/10 BBT temps, 9/10 monthly days, 2 vs 3 weights in 14 d, regularity 3 cycles inside 12 months only); re-record contract goldens.
- correlations: ratio-based strength + 30 paired days; mood-by-phase % per phase with ≥10 days/phase; weight 3-in-14; monthly 10 days; BBT 10 temps; regularity window = last 12 months.

**M9-AK-02 — Analysis consumes engine verdicts** · touches `internal/analysis/cycle.go`, `internal/cycle/service` (expose resolved regularity/data_quality/warnings) · deps: engine-v6 §2.9/§2.27 task from the engine gap report · tests: analysis verdict == `cycle_view` for fixture users; enum vocabulary identical.

**M9-AK-03 — Top finding = strength × recency + "log N more days"** · touches `internal/analysis/finding.go`, `summary.go`, copy · deps: AK-01 · tests: scoring unit tests (older strong vs recent medium), nudge days computed per analysis minimum, golden update.

**M9-AK-04 — Postpartum analysis hub (An_Hub_Post)** · touches new `internal/analysis/postpartum*.go`, `http/routes_analysis_postpartum.go`, openapi, deviations · deps: AK-01 · tests: EPDS 1 score/2 for trend, bleeding decline, sleep, child growth reuse WHO module; 409 outside postpartum; module kill-switch; EPDS never Plus-gated.

**M9-AK-05 — Analysis cache + nightly warm job** · touches `internal/analysis/handlers.go`, new `internal/analysis/cache.go`, `platform/queue` periodic registration, health-log write paths (gen bump) · deps: AK-01..03 · tests: miniredis hit/miss; editing a past log invalidates; Plus change invalidates; job idempotent; benchmark (p95 before/after).

**M9-AK-06 — Plus quota alignment** · touches `plus/entitlements.go` (LabAI free 1, assistant fair cap, 1-year range key), `analysis/range.go`, `plus` lang · deps: decisions 3/4 · tests: Resolve table, `Consume` at limit → `plus_quota_exceeded`, **regression test asserting emergency/reminder/EPDS routes never call plus** (route walk + 200 for a free user).

**M9-AK-07 — Consent ledger (`consent_events`, scope_json, versioned signup consents)** · touches new migration, `db/queries/consent`, `internal/consent/service.go`, `profile` consents, signup flow (terms + health-data processing as versioned codes) · deps: decision 5 · tests: grant v1→v2 keeps both rows; revoke sets revoked_at; Require reads projection; backfill from `user_consents`/`terms_accepted_at`.

**M9-AK-08 — Engine daily snapshot table + writer job** · touches new migration `engine_daily_snapshots`, new `internal/metrics/engine_snapshot.go` (or `internal/enginelog`), queue periodic task, retention purge · deps: engine-v6 output fields (confidence, data_quality, RUI, warnings) · tests: one row per active user/day, idempotent rerun, rui_since carry-over, purge >400 d, no date-of-prediction columns (schema guard test).

**M9-AK-09 — Prediction log + MAE/±2 d from real predictions** · touches new migration `cycle_prediction_log`, writer in AK-08 job, hook on period create/edit/delete, `metrics.go` engineHealth · deps: AK-08 · tests: first/last rows, resolution on new start, edited-away exclusion, pregnancy/contraception exclusion, basis_cycles<2 excluded, k-suppression; import guard: `internal/cycle/...` never imports it (§2.40).

**M9-AK-10 — KPI completion** · touches `internal/metrics/{kpi,metrics}.go`, `okr/okr.go` catalogue · deps: AK-08, AK-09 · tests: confidence/data_quality distribution, RUI resolved ≤7 d, AI cost / Plus revenue, family KPIs (invite acceptance, members per circle), loggers ≥15 days/month, DAU from sessions; retention ±1 window.

**M9-AK-11 — Automatic alerts** · touches new migration (`metric_alert_rules`, `metric_alerts`, `message_daily_stats`), `internal/metrics/alerts.go`, messages rollup job · deps: AK-08/09/10 · tests: each of the 4 rules fires/clears at boundary (12 %, 30 % vs 7-d mean, RUI>7 d rising, MAE 2.5).

**M9-AK-12 — Jobs & indexes hygiene** · touches migration (indexes on `daily_health_logs(log_date,user_id)`, `cycle_histories(period_start_date)`, `users(created_at)`), move `metrics.Loop`/OKR to asynq unique periodic tasks (`http/routes_admin_core.go:56-66`) · deps: none · tests: EXPLAIN-based int test; job runs once with 2 workers.

**M9-AK-13 — Admin user monitoring backend** · touches `internal/admin/users/v2.go`, new `admin_user_activity` projection + nightly job, `admin_segments` table + CRUD, HMAC ref code column · deps: AK-08, decisions 9/10 · tests: churn spec thresholds incl. 7-day inactivity; filters by mode/data_quality/warning; segment save/apply; overview shows today's engine output + cycle-length history, no health values without grant; RT code not derivable from id.

**M9-AK-14 — OKR status rule** · touches `internal/okr/okr.go:27-90`, tests · deps: decision 8 · tests: boundaries at 20/50/100 % elapsed.

**M9-AK-15 — Roles matrix + message-type authority + immutable audit** · touches `httpadmin/permissions.go:106-140`, `admin/messages` (per-type author/publish checks), migration with `admin_audit_logs` BEFORE UPDATE/DELETE triggers + hash chain, `audit.go` fail-closed for health reads · deps: none · tests: marketing cannot create/edit safety/clinical; product manager only non-clinical; engineering view-only thresholds; UPDATE/DELETE on audit raises; chain verification.

Admin-web (separate, after backend):
- **M9-AW-01** Dashboard alerts panel + engine health (confidence/data_quality distribution, MAE source label) — deps AK-09..11.
- **M9-AW-02** KPI screen new tiles (RUI resolution, AI cost %, family) — deps AK-10.
- **M9-AW-03** Users list columns (mode, data_quality, main warning), new churn, saved segments UI, user detail engine card + cycle history — deps AK-13.
- **M9-AW-04** OKR: status copy + new bindable metrics — deps AK-10, AK-14.
- **M9-AW-05** Roles screen reflects new matrix; message editor hides safety/clinical authoring for non-authorized roles — deps AK-15.
Test plan for AW: vitest for view-models, typecheck/lint, headless screenshots light+dark per memory note.

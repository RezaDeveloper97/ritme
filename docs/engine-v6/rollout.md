# Engine v6 rollout (T-M9-48, DECISIONS.md X-03)

Stage first (`shadow` → review → `v6` + visual QA), then beta (T-M9-50, user OK + DB backup). Production stays on
Laravel (T-M2-26/27 untouched).

## Stage shadow

### Setup

| | |
|---|---|
| Host / project | `root@89.251.8.115`, compose project `ritme-stage` (`https://stage.ritmeapp.ir`, Basic auth) — only `ritme-stage-*` containers touched |
| Shipped | `stage` branch, head `7d7798b8` at review time (T-M9-48 commit: shadow + sampled cache verify) |
| Env (`ritme-stage-backend-go-1`) | `CYCLE_ENGINE=shadow`, `ENGINE_CACHE_VERIFY=true`, `ENGINE_CACHE_VERIFY_SAMPLE=0.1`; worker container `ritme-stage-backend-go-worker-1` up (X-10) |
| Shadow started | 2026-11-02 13:16 UTC (backend start, goose 94…114 applied) |
| Traffic run | 2026-11-02, ended ≈ 13:43 UTC, `traffic.py` (all personas) → **5 981** comparisons |
| Breakdown run | 2026-11-02 ≈ 13:45–14:15 UTC, `probe.py` (one persona at a time, sequential, 0 dropped jobs) → **3 168** more (396 per persona) |
| Totals after both | 9 149 day comparisons, 545 dropped (all from the 4-thread traffic run), 0 failed; 388 month comparisons, 12 dropped |

How the comparison works (`backend-go/internal/cycle/shadow`): the request is served by v1.1; after the response a
2-slot pool computes the v6 status of the same day and `Classify` names the difference classes (several per day). Only
`/cycle/today` and `/cycle/date/{d}` are sampled into Redis (`ritme-go:eng:shadow:samples`, newest 1 000, 7 days);
`/cycle/month` days are classified by `service.ClassifyMonthDay` into the Prometheus counter
`engine_shadow_month_diff_total` only (no samples, not in the report). Samples carry an HMAC of the user id under a
per-process random key, so a hash can only be mapped to a persona by driving that persona alone and reading the newest
samples — which is what `probe.py` does.

### Personas (stage only, mobiles `09977000001…09`)

Seeded from the engine golden personas (`backend-go/internal/cycle/engine/testdata/personas/*.json`), every date
shifted by +12 days so the persona's `today` (2026-10-21) lands on 2026-11-02.

| id | Persona | Fixture | Shape | Sample hash (this process) |
|---:|---|---|---|---|
| 50 | regular | `regular_28` | 6 logged 28-day cycles, today cycle day 13 | `8aecce87a117c148` |
| 51 | late 8 | `late_8` | 28-day history, 8 days late | `a331b843632e4778` |
| 52 | late 15 | `late_15` | 28-day history, 15 days late (> max_late 14) | `a1f65cdacb307770` |
| 53 | pill | `pill` | combined pill (`contraception_methods`, pack started on the last bleed) + 6 withdrawal bleeds | `f89332a5cffbe448` |
| 54 | postpartum | `postpartum_breastfeeding` | delivery 120 days ago, breastfeeding, 1 period since, 4 pre-pregnancy periods | `7735a6e5ddb3931b` |
| 55 | onboarding-only | `onboarding_fresh` | declared LMP 10 days ago, declared 30/5, no logs | `1507ad72ca50250c` |
| 56 | stale anchor | `onboarding_stale` | declared LMP 90 days ago, default lengths, no logs (§2.41) | `961cb9099a7a2d71` |
| 57 | skip | `skip` | 28-day history, the predicted start 17 days ago marked «پریود نداشتم» (E-15) | `e40f7eb9b8cbbc71` |
| 58 | QA partner | — | partner of 50–57 with a `cycle` view grant | — (companion routes are not shadowed) |

### Re-running

Scripts in [`qa/stage-shadow/`](qa/stage-shadow/):

- [`gen_seed.py`](qa/stage-shadow/gen_seed.py) + [`seed_companion.sql`](qa/stage-shadow/seed_companion.sql) — the seed
  (idempotent inserts; bumps `user_revisions`). Change `TODAY` before re-seeding on another day.
- [`traffic.py`](qa/stage-shadow/traffic.py) — every persona through status/today/home/overview, fertility,
  analysis, history, messages, phase content, 14 months of `/cycle/month`, `/cycle/date` −365…+30 (two passes, 4
  threads), then the partner's companion home and per-link cycle section:
  `ssh root@89.251.8.115 python3 - [--days N] [--only 1,2] < docs/engine-v6/qa/stage-shadow/traffic.py`
- [`probe.py`](qa/stage-shadow/probe.py) — the per-persona breakdown run (sequential `/cycle/date` −365…+30 and
  15 months, then the report after each persona):
  `scp docs/engine-v6/qa/stage-shadow/probe.py root@89.251.8.115:/tmp/ && ssh root@89.251.8.115 python3 /tmp/probe.py 1,2,3,4,5,6,7,8`
- [`breakdown.py`](qa/stage-shadow/breakdown.py) — per persona: classes with the date offsets they cover, class
  values, and the field names inside `other`:
  `scp 'root@89.251.8.115:/tmp/shadowprobe/*' ./probe/ && python3 docs/engine-v6/qa/stage-shadow/breakdown.py ./probe report0.txt`

Report: `ssh root@89.251.8.115 'docker exec ritme-stage-backend-go-1 /app/api engine-shadow-report [-n 0]'`.
Month counters and cache verify: `docker exec ritme-stage-backend-go-1 wget -qO- http://127.0.0.1:9090/metrics | grep -E '^engine_'`.

v6 values for `other` (samples hold field names only) were reproduced offline by resolving the same fixtures with
`engine.State.Range(today−365, today+30, today)` in a scratch copy of the module; for every one of the 1 322 `other`
samples of the breakdown run the reproduction differs from the served v1 value on exactly the reported fields.

## Diff classes

Day classes (`engine-shadow-report`). Classes overlap except `new_fields_only` (= no other class). Shares of the
traffic run; the breakdown run reproduces them within 2 points (57.1 / 41.7 / 41.1 / 21.8 / 9.7 / 7.3 / 1.5 / 1.2 /
1.0 %).

| Class | Count (traffic) | Share | Breakdown run, by persona | Verdict | Decision / explanation |
|---|---:|---:|---|---|---|
| `new_fields_only` | 3 320 | 55.5 % | all but postpartum; pre-history days, PMS days of past cycles, onboarding-only almost everywhere (385/396) | expected | D-120: v6 only adds `today_ui`, `limits`, `clinical_flags`, `future_cycles`, … |
| `other` | 2 534 | 42.4 % | regular 161, late 8 154, late 15 161, pill 181, postpartum 396, onboarding 10, stale 93, skip 166 | expected (2 bugs inside, see below) | see the `other` breakdown below |
| `warnings` | 2 499 | 41.8 % | regular 161, late 8 124, late 15 161, pill 181, postpartum 396, stale 93, skip 186 | expected | **E-01**: v1.1 emits `prediction_based_output` on every prediction day, v6 only for `predicted_reference` anchors and future projections (599 of 1 302 rows remove only that). Plus **E-13** `hormonal_contraception_active` (pill), **E-14** `postpartum_cycles_not_returned` replacing v1's `predicted_period_overdue`/`cycle_unresolved…` (postpartum), **E-15** `skipped_cycle_recorded` (skip), **§2.41** `stale_onboarding_anchor` (stale). The postpartum rows before the delivery date are bug **T-M9-48b**. |
| `confidence` | 1 341 | 22.4 % | regular 15, late 8 30, late 15 30, pill 181, postpartum 396, skip 38 | expected | **E-03**: future projection takes today's confidence (regular +16…+30 medium→high; late 8 future medium→low because today is > 7 days late). **E-04**: late 15 future → `unknown` (no future cycles past max_late). **E-13** pill high/medium→low. **E-14** postpartum low→unknown before the first postpartum period, →low after. **E-15** skip medium→low (`predicted_reference`). |
| `phase` | 571 | 9.5 % | late 8 30, late 15 30, pill 147, postpartum 45, stale 38, skip 18 | expected | **E-04** late future days rebased to today+1 (v1 rolls the stale anchor forward) or `unknown` past max_late; **E-13** pill non-bleed days `unknown`; **E-14** postpartum predicted days `unknown`; **§2.41 / D-123** stale anchor rolled forward (v1 says `unknown`, > 14 days late); **E-15** skip days −17…0 restart at the skipped date instead of `period_expected`. |
| `suppression` | 433 | 7.2 % | pill 181, postpartum 51 | expected | **E-24 / E-13 / E-14**: `fertility_level` → `unknown` while `limits.fertility_suppressed`. |
| `onboarding_anchor` | 91 | 1.5 % | stale 48 (−47…0) | expected | **§2.41 + I-01 / D-123**: anchor > L+14 old → `predicted_reference` instead of `onboarding_declared`. |
| `lateness` | 70 | 1.2 % | late 8 8, late 15 15, stale 14 | expected | **§2.22 + E-04 / D-121**: `days_late` agrees, the legacy `calculation.cycle_day` wraps into a new cycle (v6 37/44 vs v1 9/16). |
| `o_plus_one` | 64 | 1.1 % | 1 per cycle per persona (32) | expected | **§2.18** (spec change table; classifier D-30): fertile window O−5…O, v1.1's window ends O+1. |

### What `other` hides (breakdown run, 1 322 samples)

| Field | Samples | Personas | v1 → v6 | Verdict / decision |
|---|---:|---|---|---|
| `confidence_reasons` | 1 266 | all but onboarding-only | `prediction_based_output` dropped (645 rows differ only by that), `hormonal_contraception_active` added (pill), overdue/unresolved reasons dropped (postpartum, stale, skip) | expected — mirrors the `warnings` row (E-01, E-13, E-14, E-15, §2.41). 554 `other` samples have nothing else. |
| `anchors.estimated_ovulation_date` | 703 | pill 181, postpartum 396, late, stale, skip | pill/postpartum → null; others moved with the anchor | expected — E-13/E-14 (no ovulation while suppressed); E-04/E-15/§2.41 anchor moves |
| `data_quality` | 601 | pill 144, postpartum 368, stale 48, late 15 30, skip 11 | pill good→partial; postpartum poor→partial, good→partial; stale poor→partial; late 15 future partial→insufficient; skip good/poor→partial | pill and postpartum good→partial: **bug T-M9-48c**; the rest expected (§2.28: `predicted_reference` start is not real; > 14 late no longer poor after §2.41 / E-15 re-anchoring; E-04 future without a chain) |
| `resolution_source` | 524 | postpartum 391, stale 93, late 15 30, onboarding 10 | postpartum prediction→unknown; stale prediction→default_based; late 15 future →unknown; onboarding future prediction→onboarding_based | expected — E-14, **E-10** (onboarding/default override incl. projections), E-04 |
| `days_to_period`, `anchors.predicted_next_period_start` | 522 | postpartum 396, late, stale, skip | postpartum → null; others moved | expected — E-14 (no `predicted_next`), E-04, E-15, §2.41 |
| `cycle_day` | 517 | postpartum 391, late 8/15 30 each, stale 48, skip 18 | postpartum → null off bleeding days; late future 11→2 (rebased) / →null; stale 91→7; skip 46→18 | expected — E-14 + T-M9-53, E-04, §2.41, E-15 |
| `days_late` | 491 | postpartum 396, stale 48, late 15 30, skip 17 | postpartum 0…401 → null; stale 62→0; skip 17→0; late 15 future 0→null | expected — E-14, §2.41, E-15, E-04 |
| `requires_user_input` | 378 | postpartum 345, stale 30, skip 3 | postpartum true→false; stale future false→true; skip true→false | expected — E-14 (`false`), §2.41 (`true`), E-15 |
| `days_to_ovulation` | 351 | pill 175, postpartum 50, late, stale, skip | → null under suppression; moved otherwise | expected — E-13, E-14, E-04, E-15, §2.41 |
| `anchors.current_period_start/_end/_source/_is_confirmed` | 41…159 | late 8/15, postpartum 23, stale 48, skip 18–28 | late 8 future start = today+1; late 15 future → null; postpartum future rolled v1 start → the real postpartum period; skip → skipped date, no end | expected — E-04, E-14, §2.41, E-15 |
| `fertility_level` (outside suppression) | 104 | late 8 18, late 15 30, stale 38, skip 18 | follows the phase change | expected — same rows as `phase` |
| `metrics.*` | 0 | — | — | no diff anywhere |

### Month days (`engine_shadow_month_diff_total`, no samples)

After both runs: 388 months; `phase` 8 226 and `other` 8 343 day-hits, `new_fields_only` 2 685, `suppression` 624,
`o_plus_one` 417, `lateness` 111. Reproduced offline per persona (same classifier, 396 days each):

| Month class | Main cause | Verdict / decision |
|---|---|---|
| `phase` (~70 % of days) | ≈ 80 % are days **before the first logged/declared period**: the legacy month engine paints phases backwards from the anchor, v6 has no cycle there (`unknown`, no content phase). Then `follicular`→`ovulation` on `high_fertility` days, pill/postpartum suppressed days, late/skip/stale rows as in the day table | expected — D-121 (month from the v6 range: no wrap, no invented cycles; E-22 lays out only real cycles), **I-11** content-phase map, E-04, E-13, E-14, E-15 |
| `other` | the same pre-history days (`cycle_day` present only in v1), legacy PMS window wider than `pms_possible` (next−3…next−1), period flag on rolled-forward/late days | expected — X-06 / I-11 (PMS = next−3…next−1), D-121, E-04, E-15 |
| `suppression`, `o_plus_one`, `lateness` | as the day classes | expected (E-24, §2.18, §2.22/D-121) |

### Classifier — open items (no code change in T-M9-48)

`other` is a catch-all that is mostly explained noise; it should be split before beta (T-M9-50) so a real regression
there is visible. Proposed (for T-M9-51 or a small follow-up):

1. Count a `confidence_reasons` diff that equals the `warnings` diff under `warnings`, not `other` (554 / 1 322
   `other` samples are only that).
2. New day classes: `future_projection` (target > today and either side resolves from a projection — E-03/E-04),
   `skip_anchor` (E-15), `declared_source` (`resolution_source` onboarding/default — E-10), `data_quality`; and let
   `suppression` own the nulls it causes (`days_to_ovulation`, ovulation, `predicted_next` under E-13/E-14).
3. Month: a `v6_no_cycle` class for days where v6 has no cycle and the legacy month paints one (pre-history), and a
   `pms_window` class; and print the month counters in `engine-shadow-report`.
4. Samples of `other` could carry the v1/v6 values of the date-free fields (`resolution_source`, `data_quality`,
   `requires_user_input`) — they are not health values beyond what the classes already store.

## Bugs found

None blocks the classifier review; each is small and lives in `backend-go/internal/cycle/engine`. Proposed tasks
(not created yet):

- **T-M9-48b — Postpartum/breastfeeding rule applies to targets before the delivery date.**
  `State.postpartumPeriods` (limits.go) turns E-14 on for every target once the mode is postpartum or breastfeeding
  is set; it never checks `target < delivery_date`. Persona 54 (postpartum): on stage all 396 days carry
  `postpartum_cycles_not_returned` and `limits.fertility_suppressed`, including offsets −365…−121, which are before the
  birth. Offline the same rule turns the completed **pre-pregnancy** cycles into `main_phase = unknown`,
  `cycle_day = null` (fixture 2025-06-10 / 2025-06-20 = cycle days 10 and 20 of the closed 29-day cycle
  2025-06-01 → 06-30; only the logged bleed days stay `menstrual`), so the Calendar history before the pregnancy goes blank. Expected: E-14 only for
  `target ≥ delivery_date`; earlier targets resolve normally (E-22 retrospective layout for closed cycles, `unknown`
  without the postpartum warning for the pregnancy gap).
- **T-M9-48c — `data_quality` drops good→partial on suppressed `unknown` days.**
  quality.go requires `MainPhase != unknown` for `good`, but under E-13/E-14 `unknown` is a deliberate suppression
  («"unknown" here never means "insufficient data"», E-13). Persona 53 (pill): 144 days flip `good` → `partial` and
  back to `good` on each withdrawal bleed (offsets −146…−123, −118…−95, … , −6…17); persona 54: 23 days good→partial
  right after the postpartum period (offsets −15…7). Expected: suppression-unknown counts as resolved for §2.28 `good` (confidence stays
  `low` through §2.29, so the user-visible confidence does not change; the `data_quality` badge and analytics do).
- **T-M9-48d (needs a product decision first) — Hormonal suppression has no start date.**
  `Input.Hormonal` is one bool, so E-13 applies to every target. Persona 53's pill pack started on the last bleed
  (2026-10-23 on stage), yet every day back to the first logged period (offset −150) is suppressed: `main_phase`,
  fertile window and ovulation disappear from cycles that were before the pill. E-13 / I-03 only say "active method";
  decide whether the engine should honour `contraception_methods` start (`pack_started_on` / created date) and lay
  out earlier cycles normally. Recommended: yes (same reasoning as T-M9-48b).

Not engine bugs, noted from the same run: `GET /cycle/phase-content/:phase` 404 ×116 (stage `phase_contents` is
empty) and 422 ×12 (a phase key from `traffic.py` the route rejects); `/analysis/fertility` 404
×4 (TTC-only); `/companion/home` and `/companions/links/:id/sections/:section` 403 ×7 for the QA partner — not
investigated (no shadow on those routes; family M11 is suspended).

## Metrics

Taken 2026-11-02 ≈ 14:20 UTC from the private listener (`127.0.0.1:9090`) and the backend log, process up since
13:16 UTC.

| Check | Value | Target |
|---|---|---|
| `engine_cache_verify_mismatch_total` | **0** (sample 10 % of 10 900 L1 hits + 4 L2 hits ≈ 1 090 verifications) | 0 ✔ |
| 5xx (Prometheus `http_request_duration_seconds_count{status=~"5.."}`) | **0** | no increase ✔ |
| 5xx in `docker logs ritme-stage-backend-go-1` | **0** (9 332 × 200, 120 × 404, 12 × 422, 7 × 403) | ✔ |
| `engine_shadow_jobs_total` | compared 9 149, dropped 545, failed 0 | failed 0 ✔ |
| `engine_shadow_month_jobs_total` | compared 388, dropped 12, failed 0 | ✔ |
| Engine caches | `engine_l1` 10 900 hit / 29 miss; `engine_l2` 4 / 12; v1 `cycle_engine` 6 806 / 3 288 | — |
| Errors/warnings in the log | no engine/shadow warnings; slow-request WARNs only on shop catalog and language messages (cold cache, ≤ 1.5 s); `ADMIN_REF_SECRET` unset on stage (admin ref codes disabled) | — |

Query counts per route (task step 5) are not collected yet — do it with the switch (`db_queries_per_request` by route).

## Stage: v6 since 2026-11-04

**Stage: v6 since 2026-11-04** (commit `db1a3214` sets `CYCLE_ENGINE` default to `v6` in `docker-compose.stage.yml`;
rollback = `CYCLE_ENGINE=shadow` in `/opt/ritme-stage/.env` and restart `backend-go`). Shadow ran 2026-11-02 13:16 UTC →
2026-11-04 (> 24 h). The T-M9-48b/c/d fixes shipped with it.

### Checks (2026-11-04 ≈ 09:50 UTC, process up since 08:15 UTC)

| Check | Value | Target |
|---|---|---|
| `docker exec ritme-stage-backend-go-1 env` | `CYCLE_ENGINE=v6`, `ENGINE_CACHE_VERIFY=true`, `ENGINE_CACHE_VERIFY_SAMPLE=0.1` | ✔ |
| Containers | `backend-go`, `backend-go-worker` healthy; `/ready` → `{"status":"ok"}` | ✔ |
| `engine_cache_verify_mismatch_total` | **0** | 0 ✔ |
| 5xx (Prometheus + backend log, last 26 h) | **0** (log: 600 × 200, 84 × 304, 2 × 409) | no increase ✔ |
| 409s | 2 × `GET /api/v1/messages/mode` | not engine; left as is |
| Log errors | only `admin ref codes disabled` (`ADMIN_REF_SECRET` unset on stage, known); one slow query `ListCheckupPlanRows` 202 ms | — |

Query counts per route (`db_queries_per_request` sum / count since 08:15 UTC; cold and warm mixed, QA traffic only):

| Route | Avg queries | Requests |
|---|---|---|
| `/cycle/today` | 1.55 | 38 |
| `/cycle/date/:date` | 1.00 | 2 |
| `/cycle/month/:year/:month` | 1.46 | 78 |
| `/home/cycle-overview` | 5.4 | 16 |
| `/home/sections/:section` | 6.9 | 30 |
| `/messages/feed` | 5.0 | 16 |
| `/fertility/today` | 2.6 | 16 |
| `/fertility/insights` | 2.5 | 2 |
| `/analysis/cycle` | 10 | 2 |
| `/companion/home` | 35 | 2 |
| `/plus/status` | 8.4 | 16 |
| `/profile/life-stage` | 5.5 | 30 |

Engine GETs are in line with the local gate (T-M9-49/49b/49c). `/companion/home` (35) is the family/companion path —
not engine, M11 suspended; not acted on.

### Visual QA (light + dark, fa, 390 px) — `docs/engine-v6/qa/T-M9-48/`

| Persona | Screens | Result |
|---|---|---|
| 1 regular | home, calendar, cycles, fertility insights, analysis cycle | ✔ day 15, next period in 14 days, window 8–13 Aban, ovulation ≈ 13 Aban; calendar months consistent |
| 2 late 8 | home | engine ✔ (10 days late now — seeded 2026-11-02, dates moved on); **layout bug**: «تأخیر پریود» under the today marker → T-M9-48e |
| 3 late 15 | home, calendar, cycles | ✔ 17 days late, > 2 weeks banner with test / last-period / no-period actions |
| 4 pill | home, calendar, cycles | ✔ withdrawal bleed predictions («خون‌ریزی قطع مصرف»), no ovulation |
| 5 postpartum | home, calendar, cycles | ✔ postpartum home (week 17); calendar empty after delivery (T-M9-48b) |
| 6 onboarding only | home | ✔ predictions from onboarding answers |
| 7 stale anchor | home | ✔ «تاریخ آخرین پریودت را بگو» banner, hollow predicted dots |
| 8 skip | home | ✔ |
| 9 partner | companion home | ✔ partner sees the shared persona tabs |

Not engine, noted only: the cycles page «نوسان ±۲» (median ± band, minimum 2, `insights/history.go`) and analysis
«نوسان ۰» (range) use the same word for two measures — copy question for M8; fertility insights still says «سیکل»
in one row.

## Beta shadow (T-M9-50)

| | |
|---|---|
| First user OK | 2026-11-04 (stage results, perf gate and visible changes I-01, E-04, E-13, E-15 shown) |
| Backup | `root@89.251.8.115:/root/backups/ritme-beta-T-M9-50-20261104T1101Z.sql.gz` (mariadb-dump `--single-transaction --routines --triggers`, 237 tables, goose 108, 2 users; outside the docker volumes) |
| Shipped | `FROM_WORKTREE=0 ./deploy-beta.sh`, `stage` at `71af5a7e`; first try failed in rsync (broken pipe, nothing changed), retry ok |
| Config | `docker-compose.beta.yml`: `CYCLE_ENGINE=${CYCLE_ENGINE:-shadow}`, `ENGINE_CACHE_VERIFY=true`, `ENGINE_CACHE_VERIFY_SAMPLE=0.1` |
| Shadow started | 2026-11-04 12:12 UTC (backend-go start; goose 108 → 116 applied) |
| First check (12:30 UTC) | env `CYCLE_ENGINE=shadow`; all deploy probes ok; `engine_cache_verify_mismatch_total` 0; users 2 (unchanged); only log error = `admin ref codes disabled` (known) |
| Rollback | `CYCLE_ENGINE=v1` in `/opt/ritme-beta/.env`, `NO_BUILD=1 SERVICES=backend-go ./deploy-beta.sh`; the data migrations work with v1; DB restore from the backup above only as a last resort |

Pending: ≥ 48 h of shadow (not before 2026-11-06 12:12 UTC) → `api engine-shadow-report` on beta + 5xx + verify
metrics → second user OK → `CYCLE_ENGINE=v6` → 48 h watch → "Beta: v6 since <date>". Beta has only 2 users, so the
shadow sample will be small; stage's 9 149 comparisons stay the main evidence.

## Next

1. T-M9-48e (late ring label) before beta.
2. Beta (T-M9-50): user OK + DB backup first.

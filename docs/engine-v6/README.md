# Engine v6 — architecture and plan

Source of truth: [`SPEC.md`](SPEC.md) (converted from `Ritme_Spec_v6_Unified.docx`, 2026-10-21).
Open questions and their defaults: [`DECISIONS.md`](DECISIONS.md). Gap → task map: [`GAP-ANALYSIS.md`](GAP-ANALYSIS.md).
Evidence (file:line, test runs): [`gap/`](gap/) — 01 cycle core · 02 inputs & consumers · 02b/02c frontend ·
03 message engine · 04 health modules · 05 family · 06 analysis/KPI/admin · 07 infra & performance.

The work is queued as `/next-task` tasks: **M9** (cycle engine, speed, safety), **M10** (messages, health modules,
analysis, KPI, admin) and **M11** (family). Web + backend only. No Android. No production cutover (T-M2-26/27 stay
blocked): production keeps running Laravel; stage and beta run Go.

---

## 1. Where we start (2026-10-21)

- **Three cycle engines plus browser math.** The legacy engine serves `/cycle/month`, `/home` and `calculation`
  (its cycle day wraps, so it never shows lateness; fertile window O−5..O+1; PMS ≥ L−6). The v1.1 resolver serves
  `cycle_view`, fertility, companion, teen, checkups. `view.Predict` adds a third window. Analysis, fertility insights,
  the home health check, messages and the frontend `schedule.ts` each keep their own regularity, ovulation or windows
  (five regularity notions, three fertile windows, two PMS windows).
- **Most v6 inputs are stored but never read**: LH, BBT, mode, hormonal contraception, the pregnancy test, the
  postpartum date. Skipped cycles, breastfeeding and the device timezone are not stored at all. The onboarding LMP is
  saved as a *confirmed real period with an assumed end*, which breaks §2.2/§2.9/§2.40 and makes §2.41 unreachable.
- **The Redis cycle cache saves CPU, not I/O**: the content-hash key needs every input row loaded first, so a hit costs
  as much DB time as a miss; the key also misses the clinical-config fingerprint (stale after an admin change).
  `/fertility/insights` runs the resolver ~170–380 times per request. Auth costs 3 queries on every request.
- **Safety gaps** (S1): a kick session with < 10 kicks in 2 h raises nothing; pregnancy BP ≥ 160/110 has no urgent tier
  and vitals ignore pregnancy; postpartum fever/pad-per-hour/breathlessness are not modelled; pregnancy emergency rules
  stop firing when an admin disables their copy row.
- **Privacy gaps**: members with the `symptoms` grant see mood (a never-share category) and private note text; a stale
  pregnancy grant shows a *new* pregnancy immediately, including after a loss.
- **The v2 message rule engine is solid but nothing in the product calls it.**

## 2. Target architecture (cycle engine)

```
 write request ─► service tx ──► rows ──► user_revisions.engine_rev += 1  (same tx)
                                   └────► commit ─► (async, optional) warm job

 read request ─► auth (1 query: token ⋈ user ⋈ user_revisions) ─► rev
            ├─► If-None-Match == W/"u{uid36}.e{rev}-{localToday}-{locale}-{paramsFP}-{cfgFP}-{build}" ─► 304
            └─► state.For(uid, rev, paramsFP)
                   L1 in-process LRU (prepared *engine.State) ─hit─► Resolve/Range (pure CPU, µs)
                   miss ─► singleflight(uid:rev) ─► L2 Redis input snapshot ritme-go:eng:in:{uid}:{rev}
                                                   miss ─► inputs.Load (≤ 4 bounded queries) ─► set L2
                          engine.Prepare(input, params) ─► set L1
```

### 2.1 Packages
| Package | Role | Rules |
|---|---|---|
| `internal/cycle/engine` | The v6 engine: `Prepare`, `Resolve`, `Range`, `FutureCycles`, `TodayUI`, §2.35 encoder | Pure: no I/O, no globals, no clock, no clinicalconfig; epoch-day `int32` inside; hand-written v6 enums; coverage ≥ 95 % |
| `internal/cycle/engine/bbt` | 3-over-6 shift detector (strict/standard) | Pure; also used by `internal/fertility/bbt` |
| `internal/cycle/engineparams` | `engine.Params` from the clinical-config snapshot valid on a date + `Fingerprint()` | Only place that reads `engine.*` config |
| `internal/cycle/inputs` | DB → `engine.Input` (`Load`, `LoadMany`); adapters | ≤ 4 queries per user, bounded history window; batch-friendly (`IN (…)`) |
| `internal/cycle/revision` | `Bump(ctx, tx, uid)`, manifest of every engine-input write query | Called inside the writer's transaction |
| `internal/cycle/state` | L1/L2/singleflight provider of `*engine.State` keyed by `(uid, rev, paramsFP, build)` | Kill switch `CYCLE_ENGINE_CACHE=off`; verify mode `ENGINE_CACHE_VERIFY` |
| `internal/cycle/service` | HTTP-facing: today/date/month/skips; rollout flag `CYCLE_ENGINE=v1|shadow|v6` | Every consumer module asks this service (or `state`) — no private loaders |

### 2.2 Engine API and cost
```go
func Prepare(in Input, p Params) *State                          // sort, validate, cycles, medians, BBT/LH per cycle,
                                                                 // luteal median, skips — today-independent
func (s *State) Resolve(target, today Day) Status                // O(log H): binary-search anchor, no allocs but output
func (s *State) Range(from, to, today Day) []Status              // one pass per cycle (calendar, charts)
func (s *State) FutureCycles(today Day) []FutureCycle            // k = 1..6, display only, never stored
func TodayUI(st Status, s *State) TodayUI                        // §2.43
```
Today-dependent parts (age, 12-month regularity window, absent flag, stale anchor, lateness) are computed at resolve
time. A `State` never depends on "today", so nothing is invalidated at midnight and there is no 00:00 stampede.

## 3. Invalidation and caching

- **Revision is DB-authoritative.** `user_revisions(user_id PK, engine_rev, analysis_rev, updated_at)`. Every write to
  an engine input bumps `engine_rev` (and `analysis_rev`) with `INSERT … ON DUPLICATE KEY UPDATE` *inside the same
  transaction*, so there is no window where data changed but the revision did not, and a Redis flush can never serve
  stale data. Non-engine analysis inputs (vitals, labs, weight, EPDS) bump only `analysis_rev` (M10).
- **The revision is free on reads**: the auth query (already 1 round-trip after T-M9-09) LEFT JOINs `user_revisions`.
- **Engine-input tables**: `cycle_histories`, `cycle_skips`, `daily_health_logs`, `health_log_entries`,
  `fertility_logs`, `user_profiles`, `cycle_preferences`, `user_life_profiles`, `contraception_methods`,
  `pregnancy_profiles` (+ v2 dating), `postpartum_profiles`. A manifest test parses `db/queries/**/*.sql` and fails when
  a write query on these tables has no manifest entry pointing at an integration test that proves the bump.
- **Data migrations** that change engine inputs must bump the affected users' `engine_rev` in the same migration.
- **L1** (in-process LRU, size-bounded) holds prepared states; **L2** (Redis, TTL 7 d) holds the input snapshot JSON,
  which is params-independent, so a clinical-config change only re-prepares (fingerprint in the L1 key) and never needs
  a global epoch. Redis errors fall back to the DB. **singleflight** collapses the 5–6 concurrent app-open requests into
  one load.
- **Config caches** (`internal/platform/configcache`): atomic snapshots of small admin tables (message contents,
  recommendations, message rules, templates, challenges, articles, plans, languages, modules), invalidated locally on
  admin write, cross-instance through Redis pub/sub `ritme-go:config-changed`, with a 60 s TTL safety net.
- **ETag/304** on engine GETs: `W/"u{uid36}.e{rev}-{localToday}-{locale}-{paramsFP}-{cfgFP}-{build}"`,
  `Cache-Control: private, no-cache`, `Vary: Authorization, Accept-Language`. The user id is part of the tag so two
  accounts on one browser can never revalidate each other's cached body. A revalidation costs only the auth query.
- **Verification**: `ENGINE_CACHE_VERIFY=1` (all tests; sampled on stage) recomputes on every hit and counts mismatches.

## 4. Async work, infrastructure — and why not RabbitMQ

| Need | Choice |
|---|---|
| Jobs (OTP SMS, course SMS, nightly snapshots, warmers, sweeps) | **asynq on the existing Redis**: retries, timeouts, `Unique`/`TaskID`, `ProcessIn/At`, weighted queues (`critical` 6 / `default` 3 / `low` 1), `Scheduler` (cron) with a single-leader key, `Inspector` metrics |
| Periodic loops (10 in-process tickers today, no leader lock) | Scheduler entries; `SKIP LOCKED` pollers (labs, health-record extract) stay |
| CPU isolation | `api worker` subcommand, second container on stage/beta (`API_RUN_WORKERS=0` on the API) |
| Message events, member fan-out | Buffered in-process batch writer (events); asynq for fan-out (member alerts) |
| Cross-instance cache invalidation | Redis pub/sub + TTL safety net |

RabbitMQ/NATS/Kafka are **not** used: one server, one API replica per environment, tens of jobs per minute. A second
stateful broker (×3 stacks on one host) would add operations and failure modes with zero latency gain — latency comes
from synchronous DB round-trips, which the revision + state cache removes. Revisit only with multi-language consumers,
cross-service fan-out or sustained > 1k jobs/s.

Infra hardening (stage/beta overrides only): gzip for JSON > 1 KB; Fiber read/write/idle timeouts; per-request DB
deadline 5 s; `/ready` (DB + Redis ping) for deploy waits; `interpolateParams=true` and pool size from env; Redis
`maxmemory` + `volatile-lru` (sessions, queue and throttles have no TTL and survive); MariaDB buffer pool + slow log
(0.2 s) + performance_schema.

## 5. User-local day (§2.3)
The web client sends `X-Timezone`; the API validates it (`time.LoadLocation`, tzdata embedded), falls back to
`user_profiles.timezone`, then `Asia/Tehran`, and stores a changed zone. `civildate.LocalToday(c)` is used by the engine
path and its direct consumers; background jobs use the stored zone. DB timestamps stay Tehran wall-clock.

## 6. Observability
Prometheus registry; `/metrics` and `/debug/pprof` on a separate listener `127.0.0.1:9090` (never routed by nginx).
Metrics: `http_request_duration_seconds{route,method,status}` (route template), `db_queries_per_request{route}`,
`db_query_duration_seconds{query}`, `cache_requests_total{cache,result}`, `engine_build_seconds`,
`engine_shadow_diff_total{class}`, `engine_cache_verify_mismatch_total`, asynq queue/latency/archived gauges, Go runtime.
`X-Request-ID` on every request and log line; slow-request (> 500 ms) and slow-query (> 200 ms) warnings.

## 7. Rollout
`CYCLE_ENGINE=v1|shadow|v6` per environment (default `v1`). Shadow serves v1, computes v6 in the same request and
records classified differences (`o_plus_one`, `lateness`, `warnings`, `confidence`, `phase`, `new_fields_only`, …).
Stage: shadow → review → v6 → visual QA. Beta (real users): DB backup, user OK, then v6. Production: Laravel, untouched.
Laravel goldens keep running in `v1` mode until T-M9-51 retires v1.1.

## 8. Budgets
Initial ceilings (T-M9-08 records the baseline; benches then lock at baseline × 1.2):

| Endpoint | warm p95 | cold p95 | queries incl. auth |
|---|---|---|---|
| `/cycle/today`, `/cycle/date/:d` | 15 ms | 60 ms | warm 1, cold ≤ 5 |
| `/cycle/month?view=calendar` | 20 ms | 80 ms | warm 1, cold ≤ 5 |
| 304 revalidation of any engine GET | 5 ms | — | 1 |
| `/home` | 40 ms | 120 ms | ≤ 8 |
| `/messages/selected`, `/messages/daily`, feed | 20 ms | 60 ms | ≤ 4 |
| `/fertility/insights` | 25 ms | 80 ms | ≤ 4 |
| `/analysis/summary` (M10 cache) | 20 ms | 150 ms | ≤ 6 |
| Mixed `app_open` at 200 RPS (k6, local docker, 2k users) | p95 < 100 ms, errors < 0.1 % | | |

| Benchmark | Fixture | Ceiling |
|---|---|---|
| `engine.Prepare` | 60 periods + 400 markers | ≤ 300 µs |
| `State.Resolve` | 24 periods | ≤ 3 µs, ≤ 3 allocs |
| `State.Range` (31 days) | 24 periods | ≤ 150 µs, ≤ 40 allocs |
| §2.35 JSON encode | one day | ≤ 20 µs |
| `rules.Select` | 200 rules, prefetched history | ≤ 200 µs |
| auth `Verify` | — | ≤ 150 µs |

## 9. Test strategy
- **Spec tables as code**: §2.36 (21/28/35), §2.35 example (full JSON), every §6.1 test, every boundary named in a
  decision. Table-driven, one case per spec sentence, case names cite the section.
- **Invariants over random histories** (seeded `math/rand/v2`, 2 000 histories × 400 days): cycle_day ≥ 1 when a start
  exists; never fertile on a menstrual day; O+1 never fertile; every start/end has a source; fertility `unknown` iff
  period_expected/unknown/suppressed; `days_to_ovulation` null rules; `future_cycles` never stored; `Range == map(Resolve)`.
- **Fuzz** (`testing.F`) for `Prepare` (malformed, overlapping, unsorted input) and the BBT detector — no panics,
  invariants hold.
- **Goldens**: Laravel goldens = v1.1 characterization (`CYCLE_ENGINE=v1`); Go-native v6 goldens over the contract
  personas with `-update`; contract allow-lists cite the D-id.
- **Integration** (`make test-int PKG=…`, MariaDB + Redis in docker): loaders, revision bumps (manifest), cache
  coherence, 00:30 Tehran on a UTC clock, `X-Timezone`, concurrency (20 goroutines on period writes).
- **Query budgets**: counting DB wrapper; one test per hot route asserting the query count.
- **Benchmarks** in CI (`make bench` + `benchstat` vs `backend-go/bench/baseline.txt`); **k6** scenarios in
  `backend-go/loadtest/` (`app_open`, `calendar_browse`, `log_burst` with zero stale reads, `analysis_tab`,
  `fertility_tab`, `soak`).
- **Safety scenarios**: every emergency path at threshold−1 / threshold / threshold+1 against the registry value of the
  day, through the real rule set (M10 persists runs).
- **Frontend**: vitest for every mapper/selector (today actions, banner priority, schedule, suppression), parse tests
  for the §2.35 payload *and* a Laravel payload, headless light/dark screenshots (memory: CDP + test-OTP token).

## 10. Milestones and critical path
- **M9** (51 tasks): decisions → engine library (T-M9-15…22) ∥ infra (07…14) ∥ schema/inputs (23…31) → serving
  (32…39) ∥ frontend (40…47) → stage (48) → perf gate (49) → beta (50) → retire v1.1 (51). Safety/privacy hotfixes
  (02, 05, 06) and quick fixes (03, 04) have no dependencies and should run first.
- **M10** (36 tasks): decisions (01), message engine v6 (02…15), health modules and the safety self-test (16…24),
  analysis/KPI/admin (25…35), release (36).
- **M11** (23 tasks): family decisions, three serial schema tasks, `check()` authorizer, projections, invites,
  proposals, nudges, guardianship, member alerts/home, admin, frontend, release.

## 11. Rules for every Engine v6 task
- Read `backend-go/CLAUDE.md` / `frontend/CLAUDE.md` first. Talk to the user in Persian; code and docs in English.
- **Schema**: goose migration in `backend-go/db/migrations` (next free number — check `ls`; `00094_app_modules.sql`
  may still be uncommitted work) **plus** a mirrored Laravel migration in `backend/database/migrations`;
  `make schema-diff` green. `make sqlc` after a migration regenerates every `internal/*/store/models.go`, so a migration
  task never shares a parallel batch with another task that runs `make sqlc`.
- **Dependencies**: `go get module@version`, never `go mod tidy`.
- **Deviations**: never edit `docs/go-migration/deviations.md` from an implementing task; cite the reserved D-id
  (DECISIONS.md X-08) and record differences in `tasks/PROGRESS.md`.
- **Dirty tree**: ~120 unrelated modified files (app modules, onboarding, `.claude/skills/deploy-beta/SKILL.md`) belong
  to other work. Stage files by path; do not edit `deploy-beta/SKILL.md` while it has uncommitted changes.
- **Environments**: stage credentials come from env, never committed. Beta has real users: back up before migrations,
  never wipe volumes, ask the user before any beta deploy or query. Production is not touched.
- **No Android, no production cutover** — ever, in any task.

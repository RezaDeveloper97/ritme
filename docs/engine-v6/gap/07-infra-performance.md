# 07 — Infra & performance review of backend-go (input for M9 "Engine v6")

Read-only review, 2026-10-21, branch `stage`. All paths are relative to `backend-go/` unless they start with `docs/`, `tasks/` or a repo-root file.
"Q" means SQL round-trips per request. Auth costs **3 Q on every authed request** and is listed separately (§4.0).

---

## 1. Summary

1. **The Redis cycle cache saves CPU, not I/O.** `/cycle/today|date|month` call `Service.Load` before the cache lookup (`internal/cycle/service/service.go:229-239`, `:73-97`). That load is profile + every history row + logs + the full `recommendations` table. Only after it does the code hash those rows into a content-addressed key (`:189-215`, `internal/cycle/cache/cache.go`).
   - A warm hit is therefore still 3 (auth) + 4–5 Q, plus a SHA-256 over the JSON of all histories and recommendations, plus a Redis GET.
   - No other endpoint uses the cache. `/home`, `/home/cycle-overview`, analysis, insights and the shop checklist all build `cycleservice.New(d.DB, nil)` (`internal/http/routes_home.go:33`, `routes_analysis.go:31`, `routes_analysis_ttc.go:26`, `routes_cycle_insights.go:21`, `routes_shop_checklist.go:35`).
2. **The engine state is rebuilt from scratch on every request, in every module.** `cycle_histories` is read again (unbounded) by the cycle service, the message source, checkups and nudges. Life mode and profile are read twice in some requests.
   - `/fertility/insights` and `/fertility/bbt` run `resolver.Resolve` **~170–380 times per request**. Each call is O(histories) plus 5 config lookups (`internal/fertility/chart.go:71,94-108`, `insights.go:283-327`).
3. **Admin/config tables are re-read on every request.** These are `message_contents` (whole table per locale), `recommendations`, `message_rules`, task templates, challenges, affirmations, home articles and plus plans. Typical counts:

   | Endpoint | Q, excluding auth |
   |---|---|
   | `/home` | ≈19 |
   | `/messages/daily` | ≈8 |
   | `/messages/selected` | 8 + 2 per rule |
   | `/analysis/*` | 9–13 |

4. **Queue and jobs.** asynq has one task type (`send_otp_sms`). The wrapper offers immediate `Dispatch` only: no ProcessIn/At, Unique, priorities or scheduler (`internal/platform/queue/queue.go:129-215`).
   - 10 periodic loops are in-process tickers started through `auth.OnLifecycle`, with no leader lock.
   - There is one backend-go container per environment and no worker container.
   - Redis runs with no `maxmemory` or eviction policy (`docker-compose.yml:27-38`).
   - MariaDB runs on defaults: no buffer-pool sizing, no slow log.
5. **The HTTP layer has no observability.** There is no compression, no ETag/Cache-Control on per-user GETs, no Fiber timeouts, no `/metrics`, no pprof, no request IDs, no readiness probe. The DB pool settings are hard-coded (25/10).
6. **There are zero Go benchmarks in the repo** (`grep "func Benchmark"` finds nothing). The requested bench run executed no benchmarks. There are no k6, vegeta or hey scripts, and CI has no performance gate.
7. **Correctness bugs** (§6):
   - The cycle cache key omits the clinical-config fingerprint, so an admin change goes stale until Tehran midnight.
   - Period writes can race: overlap checks are non-locking reads, and a duplicate start returns a 500.
   - "Today" is hard-wired to Tehran, but spec §2.3 says it is the device timezone.
8. **Recommendation:** stay on **Redis + asynq**, which are already in the stack. **RabbitMQ, NATS and Kafka are not justified** on a single server with one API replica and low-thousands of users; they would add pure operational overhead (§7.4). The speed comes from:
   - a per-user **engine version** (Redis INCR on every input write) and a cached, date-independent **EngineState**, so each per-date resolve is pure CPU;
   - in-process caches for admin config;
   - a cheaper auth path;
   - asynq used properly: unique tasks, a scheduler, and an optional worker process;
   - Prometheus metrics, benchmarks with budgets, and k6 SLOs.

---

## 2. Caching inventory (every Redis use)

Prefix is `ritme-go:` via `cache.Client.Key` (`internal/platform/cache/cache.go`). The client has default go-redis pool and timeouts (`:30`). Values are JSON strings unless noted.

| # | Key shape | TTL | What / where | Invalidation | Stampede guard | Risks |
|---|---|---|---|---|---|---|
| 1 | `cycle-engine:{uid}:{calc_version}:{locale}:{today}:{scope}:{sha256(schema 3, inputs)[:16]}`; scope = `day:YYYY-MM-DD` or `month:YYYY-MM:{full\|calendar}` | 24 h (`cache/cache.go` `TTL`) | Raw engine JSON for `/cycle/today`, `/date`, `/month` (`cycle/service/service.go:189-239`). Toggle `CYCLE_ENGINE_CACHE=off` (`routes_cycle.go:27`). | None. The content digest changes when inputs change; old keys expire after 24 h. | None (no singleflight anywhere; `x/sync` is only indirect in `go.mod`) | ① A hit still costs the full load, because the digest needs the rows. ② Clinical config is **not** in the key (bug B1). ③ Key churn: every write mints new keys, so up to `(days viewed + months) × locales` orphans per user per day. Without `maxmemory` Redis grows until TTL. ④ Today = Tehran (B3). ⑤ No cross-user risk: uid is in the key. |
| 2 | `languages.registry` | **none** (`internal/i18n/registry.go:166`); `content.RegistryTTL` adds `EXPIRENX` on some routes (`internal/content/content.go:117-120`) | Active languages; read on **every request** by the locale middleware (Redis GET + JSON decode). 67 `i18n.NewRegistry` instances (one per route file). | `Registry.Flush` on admin language writes (`:195`) | No | Should be an in-process memo with a short TTL. Never evictable under `volatile-*`. |
| 3 | `catalog.group.{group}` | 10 min (`internal/catalog/catalog.go:34,98-132`) | Admin catalog groups (meno tips, checkup groups, lab catalog…) | `Delete` on admin write | No | OK. |
| 4 | `app_modules:v1` | 60 s (`internal/modules/modules.go:59-60,105-150`) | Module kill-switches, read by the global `modules.Gate` middleware on every request | Delete on admin write | No | OK. A candidate for an in-process memo (it is on every request). |
| 5 | `throttle:{name?}:{sha1(uid\|ip)}` + `:timer` | decay window | Laravel-compatible rate limiter, a Lua script (`internal/platform/ratelimit/ratelimit.go:80-107`). Used by auth, media, files, city services, emergency, admin core, writes (`internal/http/write_throttle.go`, 60/min), message events (`routes_messages.go:37`, 120/min), support, AI (`internal/ai/access/access.go:88-99`) | expiry | n/a | Fails closed or open? The error propagates. With `noeviction` and a full Redis, writes fail and every throttled route returns 500. |
| 6 | `admin-session:{hash}`, `admin-sessions:{adminID}` | session TTL | Admin web sessions (`internal/admin/httpadmin/sessions.go:65-175`) | logout / revoke | n/a | **Must never be evicted.** That argues for `volatile-lru`, not `allkeys-lru`. |
| 7 | `companion-accept-breaker:*`, `share-code-breaker:*` | window | Brute-force breakers (`internal/http/companion_guards.go:85`, `routes_recordsharing.go:55`) | expiry | n/a | OK. |
| 8 | asynq keys `asynq:{ritme-go}:*` | n/a | Queue (`internal/platform/queue/queue.go:205`) | n/a | n/a | Must not be evicted (same eviction argument as row 6). |
| 9 | Laravel cache key mirror (`internal/admin/languages/laravel.go:75`) | — | Flushes Laravel's language cache from admin | — | — | Prod-only interop; leave alone. |

Not Redis, but process-level state:
- `clinicalconfig` is an atomic in-memory snapshot, refreshed lazily every 1 min in the background (`internal/clinicalconfig/store.go:29-31,73-132`).
- `recommendation.Repository` is memoised **per request only**: a new one is built per `Snapshot` (`cycle/service/service.go:76`).

---

## 3. Queue / jobs / deployment

**Queue wrapper** (`internal/platform/queue/queue.go`):
- API: `New` :105, `Register(Job{Type, MaxTries, Backoff, Timeout})` :129, `Dispatch` :151, `Start` :190, `Shutdown` :238, `OnFailed` :126.
- One queue, weight 1 (:205). Options passed are `Queue`, `MaxRetry` and `Timeout` only (:169-173).
- Missing: ProcessIn/ProcessAt, `TaskID`/`Unique`, priorities, scheduler. Archived (dead) tasks are never inspected; `OnFailed` only logs (`internal/auth/module.go:92-97`).
- `QUEUE_CONNECTION=sync` runs jobs inline (:46, :162).
- The worker is in-process: `asynq.NewServerFromRedisClient`, concurrency 4, shutdown 10 s (:199-215). It starts through `auth.OnLifecycle` (`internal/auth/module.go:117`, `lifecycle.go:27-40`).
- Only task: `send_otp_sms` (`auth/module.go:57-62,91`; dispatched in `auth/handlers.go:191`).

**Periodic work.** Ten in-process tickers, none with a leader lock:

| Loop | Interval | Safe with >1 replica? |
|---|---|---|
| labs runner | 5 s / 1 m / 6 h | yes, `SKIP LOCKED` |
| healthrecord extract | 5 s | yes, `SKIP LOCKED` |
| learning SMS | 1 m | yes, lease |
| admin metrics snapshot | 1 h (`internal/metrics/metrics.go:403`) | runs twice, harmless |
| partners sync / purge | 10 m / 1 h | no lock |
| telemed booking expiry | 1 m | no lock |
| sharelinks purge | 1 h | idempotent |
| files sweep | 6 h | local volume |
| media sweep | 1 h | local volume |
| safety scenarios | startup + nightly timer (`internal/safety/scenarios.go:178`) | results in **process memory**, lost on restart |

**Deployment:**
- Base `docker-compose.yml:27-38`: `redis:7-alpine --appendonly yes`, no maxmemory or policy. Stage and beta only drop the ports.
- One `backend-go` container per environment, pinned by `container_name`:

  | Environment | Where the `backend-go` service is defined |
  |---|---|
  | stage | `docker-compose.stage.yml:78` |
  | beta | `beta.yml:58` |
  | prod | `prod.yml:71` (profile `go`, off) |

- No worker container for Go. Laravel's `queue` service sits behind profile `laravel` on stage and beta.
- MariaDB 11.4 has no config at all (defaults: 128 MB buffer pool, no slow log).
- No resource limits anywhere. Three stacks share one Docker host and prod's nginx.
- No RabbitMQ, NATS or Kafka anywhere.
- `deploy-stage.sh` ships `git archive stage`. `deploy-beta.sh` ships the working tree.

---

## 4. Hot-path traces

### 4.0 Common per-request cost (every authed GET)

| Step | Cost | Where |
|---|---|---|
| Auth | **3 Q** (`GetAccessToken`, `GetClient`, `GetUserByID`) + RSA JWT verify | `internal/auth/passport/verify.go:118,128,143` |
| Locale | 1 Redis GET + JSON decode | `i18n/registry.go:149` |
| Module gate | 1 Redis GET + JSON decode | `modules.go:108` |
| Floor | **3 Q + 2 Redis** before any handler code runs | |

### 4.1 `/api/v1/cycle/today` and `/cycle/date/:date`

Path: `routes_cycle.go:32-33` → `service/handlers.go:46-91` → `Service.DayJSON` (`service.go:225`).

| Step | Q | Where |
|---|---|---|
| `GetEngineProfileByUserID` (profile + `lengths_auto` + `life_mode`, one JOIN) | 1 | `db/queries/cycle/engine.sql:26` |
| `ListCycleHistoriesNewestFirst` (**unbounded**, `SELECT *`) | 1 | `engine.sql:10` |
| `ListDailyLogsBetween(date, date)` | 1 | `engine.sql:12` |
| `ListActiveRecommendations` (whole table) for `Tips.Signature` | 1 | `recommendation/repository.go:105-129` |
| `AnyRecommendationExists`, only when no active rows | 0–1 | `repository.go:59` |
| Redis GET (hit), or engine + `view.Build` + Redis SET (miss) | — | |
| `json.Unmarshal` of the cached blob, then re-marshal into the envelope | — | `handlers.go:76-90`, an avoidable copy |

- **Total: ~7–8 Q warm or cold.** The cache saves only CPU.
- `view.Build` recomputes `metrics.Calculate` that the engine already computed (`view/builder.go:117` vs `legacy/engine.go:65`).

### 4.2 `/cycle/month/:y/:m`

Same 4–5 Q as §4.1 (logs bounded to the month), then `Snapshot.Month` loops 28–31 days (`service.go:136-156`), calling `legacy.Engine.CalculateForDate` per day. That is pure CPU; metrics and sorted starts are shared via `engine.New`. It is cached per (version, today, locale, view, digest). Warm ≈ 7–8 Q.

### 4.3 `/home` (`home/handlers.go:136`)

Uncached cycle path. **≈19 Q + auth.**

| Step | Q | Where |
|---|---|---|
| Companion check (`GetUserGender`) | 1 | |
| Pregnancy profile | 1 | |
| `cycle.Load` (profile, histories unbounded, logs −29 d … end of week) | 3 | |
| Recommendations | 1 | |
| Unread notifications | 1 | |
| Task templates (config) + completions | 2 | |
| Challenges: active (config) + completed between + completed on | 3 | |
| Doctor reminder | 1 | |
| Medication reminder | 1 | |
| Affirmations (config) | 1 | |
| Home articles (config) | 1 | |
| smart_tip messages: `GetMessageLifeMode` (dup of the engine JOIN), `ListMessageRecentLogs` 90 d (overlaps logs already loaded), `ListLiveMessageContents(locale)` (whole table) | 3 | `messages/source.go:77,124`, `messages/content/content.go:92` |

- The week calendar runs 7× `CalculateForDate` (`home/sections.go:92-97`), CPU only.
- Other `/home` endpoints:
  - `/home/sections/:s`: 6 + section Q.
  - `POST /home/tasks/:t/toggle`: ≈10 Q (rebuilds the whole context).
  - `/home/notifications`: 3 Q.
  - `/home/cycle-overview` (`home/cycle_overview.go:125`): 5–8 Q, uncached; one `Resolve`; logs over 90 d.

### 4.4 Messages (`internal/http/routes_messages.go:29-49`)

| Endpoint | Q | Notes |
|---|---|---|
| `/messages/daily` (`messages/handlers.go:68`) | ≈8 | Companion, pregnancy, life mode, profile, daily log, 90-day logs, histories (**unbounded**, `source.go:143`), contents (whole table). Engine uncached. |
| `/messages/mode` | 4 | |
| `/messages/selected` (`messages/v2.go:80`) | 8 + up to 2 per rule tried | `ListActiveMessageRules` (config), `GetMessageUserName` (redundant: the auth user is in context). `rules.Select` probes `LastUserMessageEvent` + `CountUserMessageEvents` per candidate (`rules/rules.go:262,270`): an **N+1** that grows with the rule count. |
| `/messages/nudges` | 7–8 | Profile read twice (`healthlog/entries.go:124-133`, `conditionnudges/engine.go:149`). |
| `/messages/menopause` | 3 (non-meno) / ≈10 (meno) | |

- No GET writes. Events go through `POST /messages/events` (`v2.go:182,192`), 2 Q, throttled 120/min.
- There is no `/tips` endpoint; tips come through the engine.

### 4.5 `/fertility/*` (`internal/http/routes_fertility.go:22-25`)

| Endpoint | Q | Resolves |
|---|---|---|
| `today`, `days/:date` | 3 | 1 |
| `bbt` | 3 | `bbtCycles` → `fertileWindow` resolves every day of up to 6 cycles (≤ 60 d each): **≈170 typical, ≤ 361** (`fertility/chart.go:71,94-108`) |
| `insights` | 4 | Same, plus `window()` re-resolves today and up to 10 more, plus `historyRow` up to 5: **≈185 typical, ≈380 max** (`fertility/insights.go:130,283-327`) |

- All CPU, with no DB calls inside the loops, and **no cache**.
- `fertility.cycleInputs` (`fertility/service.go:69`) loads unbounded histories.
- `Resolve` is pure (`internal/cycle/resolver/resolver.go:97`). Each call does 5 `clinicalconfig.Int` lookups (`time.Since` + struct copy each, `resolver.go:61-67`) and up to 2 O(H) scans (`:389-415`).

### 4.6 `/analysis/*` (`internal/http/routes_analysis*.go`)

- **Common `load`** (`analysis/handlers.go:70`) = **9 Q**:
  - `plus.Entitlement`: 3, uncached (`plus/service.go:218`)
  - `cycle.Load`: 3. Its daily-log query is **never used** by analysis.
  - `ListLogEntriesBetween`: 1
  - vitals: 2
- **`range=all` = 3650 days**, effectively an unbounded `SELECT *` over `health_log_entries` and vitals (`analysis/range.go:43`, `input.go:37`).
- Per endpoint:

  | Endpoint | Q |
  |---|---|
  | `summary` | 9–10; ≈5–6 `metrics.Calculate` runs (`summary.go:91,117-123`; `typical()` recomputes each time, `input.go:107`) |
  | `monthly/:ym` | 12 (second `Entitlement`, `:291`) |
  | `ttc` / `fertility` | 10 (free) / 13 (deep: 3 always-empty queries) |
  | `pregnancy` | 13; `ListWeeklyLogs` and `ListV2CalendarAppointments` are **unbounded**; everything is computed even for locked users |

- `phaseDays()` (`input.go:168`) walks every day of every confirmed cycle in history.
- Spec §4.1 says analyses should be server-side and precomputed nightly or on open when data changed. Today they are recomputed on every request.

### 4.7 `/cycle/history`, `/cycle/symptom-pattern` (`routes_cycle_insights.go:23-24`)

- Each costs 3 Q. `history` wastes the logs query.
- `symptom-pattern` reads `SELECT *` over 315 days (`cycle/insights/pattern.go:32`). It is uncached.

### 4.8 Write path: `POST /cycle/period*`

Transaction (`cycle/periods/service.go:450`). `Start` costs ≈7–9 Q:

| Step | Notes |
|---|---|
| `GetPeriodByStart` | |
| `GetBlockingOpenPeriod` | |
| `ListLoggedPeriods` | overlap check, unbounded |
| `GetPreviousPeriod` | |
| `INSERT` | |
| `DeleteEstimatesExcept` | |
| `GetProfileByUserID` | |
| `UpdateProfileLastPeriodStart` | |
| `MarkProfileRecalculated` | |

- `Store`/`Update` also run `recomputeCycleLengths`: 1 SELECT + up to H UPDATEs, written only when dirty (`:322-343`).
- Recompute is synchronous and cheap, which is fine.

---

## 5. DB & HTTP findings

**DB:**
- Pool (`internal/platform/db/db.go:22-28`): MaxOpen 25, MaxIdle 10, lifetime 5 m, idle 2 m; hard-coded, not env-configurable.
- DSN: `parseTime`, Tehran `loc`, no `interpolateParams`. Every parameterised query is prepare + execute + close: **3 network round-trips per query** on the MySQL binary protocol. Turning on `interpolateParams=true` is a cheap, large win; it is safe with utf8mb4.
- No global query timeout (about 21 ad-hoc `WithTimeout` calls). No slow-query log in the app or in MariaDB.
- Indexes for hot queries are present:

  | Table | Index |
  |---|---|
  | `cycle_histories` | `UNIQUE(user_id, period_start_date)` (baseline :143) |
  | `daily_health_logs` | `UNIQUE(user_id, log_date)` (:214) |
  | `oauth_access_tokens` | PK + `user_id` |
  | `message_events` | `(user_id, rule_id, created_at)` and `(rule_id, event, created_at)` (`00061_messages_v2.sql:68-69`) |
  | `message_rules` | `(placement, is_active, priority)` |

  The `recommendations` lookup uses `is_active ORDER BY sort_order, id` on a small table, which is fine. Volume, not indexing, is the problem.
- **Transactions and locking:**
  - Period writes use a default InnoDB REPEATABLE READ tx with **plain consistent reads**. There is no `SELECT … FOR UPDATE` or per-user lock, so the overlap and blocker checks do not serialise concurrent requests (B2).
  - Healthlog upsert is outside a transaction but handles 1062 with createOrFirst (`healthlog/service.go:139-153`), which is fine.
  - Period insert does **not** handle 1062 (`periods/service.go:437-448`).
- **Version counter:** `calculation_version` is bumped by period writes (`periods/service.go:372`), by healthlog **only when `last_period_start` is set** (`healthlog/service.go:129-135`), by profile save (`profile/service.go:121`) and by `/recalculate`. It is **not** bumped by cycle preferences (`lengths_auto`), life-mode changes, or admin recommendation and clinical-config edits. It is therefore not usable as a cache key on its own today.

**HTTP** (`cmd/api/app.go`, `main.go`):
- Middleware: recover → JSON access log with `duration_ms` (`app.go:73-102`) → `X-Backend` → test clock → CORS → modules gate.
- Missing:
  - **compression** (none in Fiber or nginx `deploy/*.inc`). `/cycle/month?view=full` is about 119 KB; Persian JSON gzips about 5–8×;
  - **ETag / Cache-Control** on per-user GETs (only storage files, `content/storage.go:82-89`, and the public city directory have them);
  - **Read/Write/Idle timeouts** (Fiber defaults are unlimited; nginx has `proxy_read_timeout 120s`);
  - global rate limit;
  - **pprof, `/metrics`, request IDs, tracing.**
- Graceful shutdown is OK: `signal.NotifyContext` + 10 s (`main.go:28,50,80`); the queue and loops stop on OnPreShutdown.
- `/up` is liveness only, by design (`routes_health.go:7-11`). There is no readiness endpoint that pings DB and Redis.
- The Redis client has default pool and timeouts (`cache.go:30`).

**Benchmarks** (task 6): `go test -run xxx -bench . -benchmem ./internal/cycle/...` → every package says `PASS`/`ok` and **no benchmark exists** (repo-wide grep for `func Benchmark` = 0 files). Missing benchmarks:
- `resolver.Resolve`
- `metrics.Calculate`
- `legacy.Engine.CalculateForDate` / `Snapshot.Month`
- `view.Build`
- `cache.Key.String` (SHA over histories)
- fertility `bbtCycles` / insights
- analysis `BuildSummary`
- messages `rules.Select`
- JSON marshal of the month payload
- auth `Verify`

There are no k6, vegeta, hey or wrk scripts. Make has no `bench` target. The only CI workflow is `.github/workflows/safety-scenarios-nightly.yml`.

---

## 6. Bugs found on the way

| ID | Sev | Bug | Evidence | Fix |
|---|---|---|---|---|
| B1 | High | **Stale cycle cache after an admin clinical-config change.** The resolver and view read `clinicalconfig` (fertile window, hard cap, warning cap, max late days), but the cache key does not include `clinicalconfig.Fingerprint`. A change is served stale until the Tehran day rolls over, for up to 24 h (`today` is in the key). | `cycle/resolver/resolver.go:61-67`, `cycle/view/services.go:51-52`, `cycle/service/service.go:200-214`; `Fingerprint` exists (`clinicalconfig/store.go:168`) and is used only by vitals (`vitals/thresholds.go:159`) | Add a global "config epoch" to the key (§7.1). |
| B2 | Med | **Concurrent period writes can create overlapping periods.** Two `POST /cycle/period/start` or `/cycle/period` calls with different dates (double tap, two devices, offline replay) both pass `overlapsExisting` (non-locking read) and both insert. The same date hits the unique key → error 1062 → **500**, where it should be an idempotent 200 or a 422. | `cycle/periods/service.go:94-166,202-247,377-398,437-448` | Lock the user row at tx start (`SELECT … FROM user_profiles WHERE user_id=? FOR UPDATE`) or use `GET_LOCK('cycle:{uid}')`; map 1062 → re-read; add an idempotency key on writes from the offline queue. |
| B3 | Med (spec gap) | **"Today" is always Asia/Tehran.** Spec §2.3 says the user-local day comes from the device timezone (default Tehran). Diaspora users get the wrong day around midnight; the cache key `today` inherits this. | `cycle/service/handlers.go:46-48,62`; `civildate.InTehran` everywhere | Add a timezone to the profile or a request header, and one `LocalToday(c)` helper used by all engines and cache keys. |
| B4 | Low | **Redis has no `maxmemory`.** Under memory pressure the default `noeviction` makes throttle scripts and queue writes fail (→ 500 on throttled routes, lost OTP jobs). Cycle cache key churn (§2 row 1 ③) grows memory. | `docker-compose.yml:27-38` | `maxmemory` + `volatile-lru` (all cache keys carry a TTL; sessions, queue and throttles survive), or a separate Redis DB/instance for the cache. |
| B5 | Low | **`languages.registry` is written with no TTL.** If a direct DB edit skips `Flush` it never refreshes, unless a route with `RegistryTTL` happens to run. | `i18n/registry.go:166`, `content/content.go:117-120` | Use a TTL on Set. |
| B6 | Low | **Safety-scenario nightly results live in process memory**, lost on every deploy or restart. | `safety/scenarios.go:178` | Persist the results, or run them as a scheduled job. |
| B7 | Low | `healthlog` only bumps `calculation_version` when `last_period_start` is valid. This is harmless with the digest key, but a version-keyed cache would go stale on the first logs before any period. | `healthlog/service.go:129-135` | Covered by the invalidation hook (§7.1). |
| B8 | Perf | Wasted work: a daily-log query in analysis and `/cycle/history`; a second `Entitlement` in monthly; a duplicate life-mode and profile read; `GetMessageUserName` when the user is already in context; `metrics.Calculate` twice per `/cycle/today` miss; JSON decode → re-encode of the cached blob. | §4 | Covered by P05/P06. |

---

## 7. Recommended architecture

### 7.1 Engine state cache (the core of "very fast")

**Invalidation: per-user version key, not a content digest.**
- `eng:v:{uid}` is a Redis counter, INCR'd by one function `engine.Invalidate(ctx, uid)`. Every engine-input writer calls it **inside the same request after commit**:
  - periods: start, end, store, update, destroy;
  - daily logs and log entries (all of them, not only once a period exists);
  - profile save, cycle preferences, life mode / pregnancy / postpartum mode change;
  - onboarding;
  - BBT / LH / fertility logs;
  - account delete.
- **Why not keep the digest:** the digest needs every input row loaded just to build the key, which is what makes hits as slow as misses today. A version key turns the hit path into one Redis MGET. Spec §2.39 ("every confirmed log recomputes; past edits rebuild history and analysis caches") is naturally an "invalidate on write" rule.
- **Global epoch** `eng:gv`: INCR'd on admin writes to recommendations, clinical-config approval (also fixes B1), message rules and contents, task templates and challenges, and on deploy (schema constant). Every engine key includes it.
- **Safety net:**
  - an architecture test that greps sqlc write queries on engine tables (`cycle_histories`, `daily_health_logs`, `health_log_entries`, `user_profiles`, `cycle_preferences`, `user_life_profiles`, fertility tables) and fails if the calling service does not reach `engine.Invalidate`;
  - in tests (`ENGINE_CACHE_VERIFY=1`), every hit is recomputed and compared with the cached value.
  - Keep `CYCLE_ENGINE_CACHE=off` as the kill switch.

**What to cache: the date-independent state, not responses.**

Today's per-response keys (`day:X`, `month:…`) explode combinatorially, and every module rebuilds its own state. Instead, cache one **`EngineState`** per user:
- sorted, validated histories (bounded projection, not `SELECT *`);
- profile projection (lengths, `lengths_manual`, `life_mode`, `no_fertility`);
- `metrics.Metrics` (effective values, layers, regularity flags);
- anchors (last confirmed start, open period, logged start);
- the predicted reference chain (future cycle starts for about 12 months);
- per-cycle fertile and ovulation windows for the last 6 cycles. These are what `fertility/chart.go` and `insights.go` currently recompute about 380 times.
- Encoding: msgpack or compact JSON, about 2–10 KB.

Keying:

| Key | TTL |
|---|---|
| `eng:s:{uid}:{v}:{gv}` | 7 d |
| `eng:d:{uid}:{v}:{gv}:{localToday}:{date}:{locale}` (optional rendered-response cache) | 24 h |

- Per-date resolution (`Resolve`, `CalculateForDate`, `view.Build`) becomes **pure CPU over the state**.
- Month = one state + 31 cheap resolves.
- `/home` week = 7.
- Fertility insights = lookups in the precomputed windows.
- **One loader for all modules:** an `engine.StateFor(ctx, uid, localToday)` service injected through `http.Deps` and used by cycle, home, cycle-overview, messages, nudges, checkups, fertility, analysis, insights and the shop checklist. This replaces the cycle-history reads now duplicated across those modules.

Layers and stampede:
- **Two layers:**
  - L1: an in-process LRU (e.g. 10k users, size-bounded), keyed by `(uid, v, gv)`. Exactly one API replica exists today, so L1 is coherent as long as the version is read from Redis on each request: 1 GET of `eng:v:{uid}` + `eng:gv`, pipelined.
  - L2: Redis.
- **Stampede:** `golang.org/x/sync/singleflight` keyed by `uid:v:gv` around the DB load (already an indirect dependency; promote it to direct). Optionally a Redis `SET NX PX 2000` build lock if more replicas ever appear.
- **Local-day correctness:** `localToday` (B3 helper) goes into the response key, never into the state. State values depend on today only through `age` (`legacy/engine.go:45-47`) and "open period" evaluation; those stay computed at resolve time.
- **Midnight:** state keys are date-free, so nothing is invalidated at midnight and no stampede happens at 00:00 Tehran.
- **Expected warm `/cycle/today`:**
  - auth (see 7.3) ~1 Q, or 0 with an auth cache;
  - 1 Redis pipeline;
  - L1 state hit;
  - one resolve + marshal.
  - Target **p95 ≤ 15 ms server time**, versus ≈7–8 Q today.

### 7.2 Process-level config caches

`message_contents` (per locale), `recommendations`, `message_rules`, task templates, challenges, affirmations, home articles, plus plans, languages and app modules are small admin-edited tables read on almost every request.
- Hold them in an in-process `configcache` (atomic snapshot + loader), the same pattern as `clinicalconfig/store.go`.
- Invalidation: `eng:gv` / a per-table epoch read with the per-request pipeline, or Redis pub/sub `ritme-go:config-changed`. Fallback TTL 60 s.
- Effect: `/home` 19 → ≈8 Q; `/messages/daily` 8 → ≈3 (with state); `/messages/selected` 8 → ≈3.
- Fix the `rules.Select` N+1: one `ListUserMessageEventStats(uid, rule_ids)` query (last shown/dismissed + counts per rule, grouped) before the loop.

### 7.3 Auth hot path

3 Q per request on every endpoint is the single largest fixed cost.
1. Merge the token and user lookups into one JOIN query (`oauth_access_tokens ⋈ users`).
2. Hold `oauth_clients` in process (they almost never change).
3. *Optional:* a 60 s Redis/L1 cache of `jti → (uid, revoked, expires_at)`, **deleted synchronously** on logout, admin block, account deletion and refresh-session revoke.
   - These four are the only revocation paths (backend-go `CLAUDE.md` § Sessions).
   - Keep 401 semantics exact; contract tests must stay green.
   - Prod is Laravel, so no cross-stack token revocation reaches the Go cache on prod. On stage and beta Go is the only writer.

### 7.4 Async: asynq vs RabbitMQ/NATS/Kafka

**Decision: asynq on the existing Redis. RabbitMQ is not justified.**
- One server, one API replica per environment, low thousands of users, and job volume of tens per minute: OTP SMS, course SMS, nightly snapshots.
- asynq already gives retries with backoff, timeouts, unique tasks (`asynq.Unique`/`TaskID`), delayed tasks (`ProcessIn/At`), weighted queues, an archive (dead letter) with `Inspector`, and a cron `Scheduler`.
- RabbitMQ would add a second stateful broker per stack (×3 stacks on one host), a new client library, new ops (vhosts, credentials, persistence, monitoring) and new failure modes, for **zero** latency gain. Request latency comes from synchronous DB round-trips, not from a queue.
- Revisit only if there are multiple consumers in other languages, cross-service event fan-out, or sustained throughput above about 1k jobs/s. Kafka and NATS have no use case here (no event streaming, no multi-service fabric).

**Queue wrapper v2** (`internal/platform/queue`):
- Add `DispatchIn`, `DispatchAt`, `Unique(ttl)` / `TaskID`.
- Three weighted queues: `critical` (OTP) 6, `default` 3, `low` (snapshots, warmers) 1.
- `Scheduler.Register(cron, type, payload)` over `asynq.Scheduler`. It must run in exactly one process; with one replica that is trivial. A Redis `SET NX` leader key gives safety later.
- Archived-task gauge + an admin view via `asynq.Inspector`. An idempotency guideline: handlers keyed by natural ids.

**Jobs to add:**

| Job | Trigger | Unique key | Purpose |
|---|---|---|---|
| `engine:warm` | after `engine.Invalidate` | `uid`, 30 s | prebuild EngineState + today's response so the next app open is a hit. The write still invalidates synchronously, so correctness never depends on the job. |
| `engine:snapshot:nightly` | cron 03:30 Tehran | date | per active user (logged-in ≤ 30 d), batched, `low` queue. Persist the daily engine output (`predicted_next_period_start`, confidence, data_quality, requires_user_input) into an `engine_daily_snapshots` table. This is the only correct source for §5.6 KPIs: MAE / ±2-day accuracy needs the *prediction as it was*, which cannot be reconstructed later, plus the confidence distribution and the requires_user_input resolution rate. |
| `analysis:refresh` | after invalidate (debounced via Unique, 5 min) + nightly | `uid` | precompute analysis payloads per range (§4.1 "nightly or on open when data changed"); store in Redis `an:{uid}:{v}:{gv}:{kind}:{range}`, TTL 7 d. GET serves cached or computes inline on a miss. |
| `messages:evaluate` | app-open hook / mode change (§5.4) | `uid`, local day | not needed for latency if config caching lands; keep for push later (there is no push channel today). |
| migrate tickers | cron | — | partners sync/purge, telemed expiry, sharelinks purge, metrics snapshot, safety nightly (persist results, B6), files/media sweeps → scheduler entries. Labs and healthrecord runners keep their `SKIP LOCKED` pollers (they work and are replica-safe). |

**Worker process:**
- Phase 1: keep the in-API worker; it is fine at this scale.
- Phase 2 (same task series): the same binary with an `api worker` subcommand as a second compose service in **stage and beta only**, sharing the storage volumes. The API then runs no scheduler or heavy jobs, so nightly snapshot CPU never competes with request latency.
- Never touch prod compose behaviour (prod = Laravel; Go is behind profile `go`).

### 7.5 HTTP and infrastructure

- **Compression:** gzip/brotli for `application/json` > 1 KB, in nginx (preferred, one place) or Fiber `compress`.
- **Conditional GETs:**
  - `ETag: W/"{uid}-{v}-{gv}-{localToday}-{locale}"` on engine-derived GETs, so a 304 is answered **before any DB load** (needs only the version pipeline).
  - `Cache-Control: private, no-cache`.
  - The frontend fetch client must pass `If-None-Match`; the browser does this automatically for `no-cache`.
- **Timeouts:** `ReadTimeout` 15 s, `WriteTimeout` 30 s (AI and upload routes override), `IdleTimeout` 60 s. Global per-request DB context deadline of 5 s, except long jobs.
- **`/ready`:** ping DB + Redis with a 1 s timeout, for the deploy scripts' health wait. `/up` stays liveness.
- **DSN and pool:** `interpolateParams=true`; pool from env (`DB_MAX_OPEN` 25 → keep; `DB_MAX_IDLE` = max open, to avoid churn).
- **Redis:** `maxmemory 256mb`, `maxmemory-policy volatile-lru`, all cache keys with a TTL. Set in the stage and beta compose override only.
- **MariaDB** (stage/beta conf mount):
  - `innodb_buffer_pool_size` ≈ 512 MB–1 GB depending on host RAM;
  - `slow_query_log=1`, `long_query_time=0.2`, `log_queries_not_using_indexes=0`;
  - `performance_schema` on, for digest stats.

### 7.6 Observability

Use `prometheus/client_golang`. It is a new dependency, justified because no metrics exist at all; the alternative is OTel metrics → Prometheus exporter, which is heavier.

| Metric | Type | Labels |
|---|---|---|
| `http_request_duration_seconds` | histogram | `route` template (not the raw path), `method`, `status` |
| `db_queries_per_request` | histogram | `route` |
| `db_query_duration_seconds` | histogram | sqlc query name, via a `DBTX` wrapper |
| `cache_requests_total` | counter | `cache` = engine_state / engine_resp / config / auth / analysis, `result` = hit / miss / error |
| `engine_build_seconds` | histogram | |
| `asynq_queue_size` | gauge | `queue`, `state` (via Inspector, every 15 s) |
| `asynq_task_duration_seconds` | histogram | |
| `asynq_archived_total` | counter | |
| Go runtime + process collectors | | |

- `/metrics` and `/debug/pprof/*`: bind a **second listener on 127.0.0.1:9090** inside the container (not routed by nginx), or put them behind the admin-web auth and the `system.metrics` permission. Prefer the separate listener; it cannot leak through a routing mistake.
- `X-Request-ID` middleware (accept from nginx or generate), added to the access log and slog context.
- A slow-request warn log (> 500 ms) with the route and query count.
- OpenTelemetry tracing is optional and later; it is not needed on one box.

### 7.7 Load tests and budgets

- **k6 scripts in `backend-go/loadtest/`:**
  - seed N=2,000 synthetic users with 6–36 months of history;
  - mint tokens through the test-OTP path, local or stage only (the stage host is behind Basic auth: pass the credentials via env, never commit them).
- **Scenarios:**
  - `app_open` (today + month calendar + home + messages/selected + banners);
  - `calendar_browse` (12 months back);
  - `log_burst` (POST log → GET today, for invalidation correctness and latency);
  - `analysis_tab`;
  - `fertility_tab`;
  - `soak` 30 min.
- **SLOs** (server-side, measured from `http_request_duration_seconds` and k6 on local docker with prod-like data; numbers are initial ceilings, to be tightened after the baseline task):

  | Endpoint | warm p95 | cold p95 | Q budget (incl. auth) |
  |---|---|---|---|
  | `/cycle/today`, `/cycle/date` | 15 ms | 60 ms | warm ≤ 1 (≤ 0 with auth cache), cold ≤ 4 |
  | `/cycle/month?view=calendar` | 20 ms | 80 ms | warm ≤ 1, cold ≤ 4 |
  | `/home` | 40 ms | 120 ms | ≤ 8 |
  | `/messages/selected`, `/messages/daily` | 20 ms | 60 ms | ≤ 4 |
  | `/fertility/insights` | 25 ms | 80 ms | ≤ 4 |
  | `/analysis/summary` | 20 ms (precomputed) | 150 ms | ≤ 6 |
  | 304 revalidation on any engine GET | 5 ms | — | ≤ 1 |
  | Mixed `app_open` at 200 RPS | p95 < 100 ms, error rate < 0.1 % | | |

- **Go benchmark budgets** (`make bench`, `benchstat` against a committed `bench/baseline.txt`; CI fails when sec/op or allocs/op regresses > 20 %):

  | Benchmark | Fixture | Ceiling |
  |---|---|---|
  | `resolver.Resolve` | H=24 | ≤ 5 µs/op, ≤ 5 allocs |
  | `metrics.Calculate` | H=60 | ≤ 50 µs |
  | `EngineState` build | H=60 | ≤ 300 µs |
  | Month calendar from state | 31 days | ≤ 400 µs, ≤ 300 allocs |
  | `view.Build` | | ≤ 30 µs |
  | Fertility insights compute | from state | ≤ 1 ms |
  | `analysis` summary compute | 12 months | ≤ 3 ms |
  | `rules.Select` | 200 rules, prefetched stats | ≤ 200 µs |

  These are proposed ceilings; T-M9-P02 records the real baseline first and then locks the ceilings at baseline × 1.2.

- **Query-budget integration tests:** a counting `DBTX` wrapper plus `TEST_DB_DSN`; one test per hot route asserting `Q ≤ budget`. This is the same idea as Laravel's `tests/Feature/Performance/*` from T-M1-12.

---

## 8. Proposed task breakdown (M9 — Engine v6, infra track)

All tasks are web + backend only. No Android work. No production cutover (T-M2-26/27 stay blocked). Compose and infra changes go to the **stage and beta override files only**.

### T-M9-P01 — Observability foundation

**Scope:**
- Prometheus registry, plus a `/metrics` and pprof listener on `127.0.0.1:9090`.
- Route-template HTTP histogram.
- `DBTX` counting/timing wrapper (`internal/platform/db`) with per-request query count in locals.
- `X-Request-ID` middleware; slow-request and slow-query warn logs (> 500 ms / > 200 ms).
- Cache hit/miss counters API in `platform/cache`.

**touches:** `cmd/api/app.go`, `cmd/api/main.go`, `internal/platform/db`, `internal/platform/cache`, `internal/platform/httpx`, `go.mod` (`prometheus/client_golang`).

**deps:** none.

**tests:**
- Unit test: route label = template.
- Integration test: query counter equals the number of sqlc calls on a known route.
- Assertion: pprof is not reachable on the public port.

### T-M9-P02 — Benchmark harness, baseline and query budgets

**Scope:**
- Synthetic fixture builder (H = 6/24/60/120 histories, 365 logs).
- `Benchmark*` for resolver, metrics, legacy month, `view.Build`, cache key hashing, fertility insights, analysis summary, `rules.Select`, month JSON marshal.
- `make bench`, `bench/baseline.txt`, `benchstat` CI step.
- Query-budget integration tests for: today, date, month, home, home cycle-overview, messages daily/selected/nudges, fertility insights/bbt, analysis summary/monthly/ttc/pregnancy. They record *current* counts first, as failing budgets get tightened in later tasks.
- Document the baseline in `docs/engine-v6/perf-baseline.md`.

**touches:** `*_test.go` across `internal/cycle`, `fertility`, `analysis`, `messages`; `Makefile`; `.github/workflows/bench.yml`.

**deps:** P01 (counting `DBTX`).

**budgets:** record now; lock at baseline × 1.2; the §7.7 ceilings are targets for P05.

### T-M9-P03 — Engine invalidation: per-user version + global epoch

**Scope:**
- `internal/engine/version` with `Invalidate(uid)`, `BumpGlobal(reason)` and `Versions(ctx, uid)` (one pipelined MGET).
- Call sites:
  - every engine-input writer: periods, healthlog store/delete, log entries, profile, cycle prefs, life mode, pregnancy/postpartum mode, onboarding, fertility/BBT/LH logs, account delete;
  - admin writers: recommendations, clinical-config approve, message rules/contents, task templates, challenges, affirmations, home articles, plus plans, languages, modules.
- Architecture test that maps write queries on engine tables to an `Invalidate` call.
- Fixes B1 (clinical config in the key) and B7.

**touches:** `internal/cycle/periods`, `internal/healthlog`, `internal/profile`, `internal/fertility`, `internal/admin/*`, `internal/clinicalconfig`, new `internal/engine/version`.

**deps:** none (parallel with P01).

**tests:**
- miniredis unit tests.
- Integration: each writer bumps the version (table-driven).
- Admin clinical-config approve → `/cycle/today` reflects it immediately (regression for B1).

### T-M9-P04 — EngineState cache (L1 LRU + L2 Redis + singleflight)

**Scope:**
- `internal/engine/state`: a `State` struct (bounded projections, metrics, anchors, prediction chain, per-cycle fertile windows), `Build(rows)`, and `StateFor(ctx, uid)` behind singleflight, L1 size-bounded LRU and L2 Redis msgpack/JSON (`eng:s:{uid}:{v}:{gv}`, 7 d).
- `ENGINE_CACHE_VERIFY` test mode; `CYCLE_ENGINE_CACHE=off` kill switch.
- Bounded history projection query (needed columns only).
- Rewire `/cycle/today|date|month` onto the state; an optional response cache `eng:d:…`; drop the content-digest key (schema 4).
- Return the cached bytes without decode/re-encode.

**touches:** `internal/cycle/service`, `internal/cycle/cache`, `internal/cycle/legacy` (accept prebuilt metrics/starts), `internal/cycle/view` (accept metrics), `db/queries/cycle/engine.sql`, `internal/http/routes_cycle.go`, `go.mod` (`x/sync` direct).

**deps:** P03, P02.

**tests:**
- Golden sweep (`legacy/golden_sweep_test.go`) and contract diff green (byte-compatible).
- Concurrency test: 100 parallel misses → 1 build.
- Verify-mode equality over fixtures.

**budgets:**

| | Q (incl. auth) | p95 | Bench |
|---|---|---|---|
| warm `/cycle/today` | ≤ 1 + auth | ≤ 15 ms | |
| cold | ≤ 4 + auth | | |
| month | | | from state ≤ 400 µs |

### T-M9-P05 — All modules on EngineState

**Scope:**
- Inject `StateFor` via `http.Deps` into home (+ cycle-overview, week calendar), messages (daily, selected, nudges, menopause/checkups `LoadPlan`), fertility (today/days/bbt/insights: use the precomputed windows instead of per-day `Resolve`), analysis (all + ttc), `cycle/history`, `symptom-pattern`, shop checklist.
- Remove the wasted daily-log query, the duplicate life-mode/profile reads, the second `Entitlement` and `GetMessageUserName`.

**touches:** `internal/home`, `internal/messages/*`, `internal/fertility`, `internal/analysis`, `internal/cycle/insights`, `internal/checkups`, `internal/shop`, the matching `routes_*.go`.

**deps:** P04.

**tests:** existing integration and contract suites green; query budgets tightened.

**budgets:**

| Endpoint | Q | p95 | Resolves |
|---|---|---|---|
| `/home` | ≤ 8 | warm ≤ 40 ms | |
| `/fertility/insights` | ≤ 4 | ≤ 25 ms | ≤ 20 |
| `/messages/daily` | ≤ 4 | | |

### T-M9-P06 — Process-level config caches + rules N+1

**Scope:**
- `internal/platform/configcache` (atomic snapshot, loader, epoch check per request via the P03 pipeline, 60 s fallback TTL).
- Apply it to message contents, recommendations, message rules, task templates, challenges, affirmations, home articles, plus plans, languages registry (TTL fix B5) and modules.
- `ListUserMessageEventStats(uid, rule_ids)`, one query replacing the per-rule probes in `rules.Select`.

**touches:** `internal/messages/content`, `internal/messages/rules`, `internal/cycle/recommendation`, `internal/home`, `internal/challenges`, `internal/plus`, `internal/i18n`, `internal/modules`, `db/queries/messages`.

**deps:** P03.

**tests:**
- Admin edit → visible on the next request (epoch).
- `/messages/selected` with 50 rules → Q constant.

**budgets:** `/messages/selected` ≤ 4 Q; `rules.Select` bench ≤ 200 µs.

### T-M9-P07 — Auth hot path

**Scope:**
- A single JOIN token+user query; in-process `oauth_clients`.
- Optional 60 s token-validity cache with synchronous deletion on logout, admin block, account delete and refresh-session revoke.
- `DSN interpolateParams=true`; pool from env.

**touches:** `internal/auth/passport`, `internal/auth/store`, `db/queries/auth`, `internal/platform/db`, `internal/platform/config`.

**deps:** P01.

**tests:**
- Session-regression suite (365-day, refresh-window, `token_revoked`/`token_expired`/`unauthenticated` codes): a revoked token is rejected on the very next request.
- Contract `auth` routes green.

**budgets:** auth ≤ 1 Q (0 with cache); `Verify` bench ≤ 150 µs.

### T-M9-P08 — Period-write concurrency and idempotency (B2)

**Scope:**
- Per-user serialisation in `periods.Service.tx` (`SELECT … FOR UPDATE` on `user_profiles`, or `GET_LOCK`).
- Map 1062 to a re-read (idempotent start); add an optional `Idempotency-Key` header for offline-queue replays (Redis, 24 h).
- Same review for log-entry and fertility writes.

**touches:** `internal/cycle/periods`, `internal/healthlog`, `internal/fertility`.

**deps:** none.

**tests:**
- 20 goroutines × `POST /period/start` (same date and different dates) → no overlap, no 500.
- Replay with the same key → same response.

### T-M9-P09 — Queue v2 + scheduler

**Scope:**
- Wrapper: `DispatchIn/At`, `Unique`/`TaskID`, three weighted queues, `Scheduler` (`asynq.Scheduler`, single-process leader key), `Inspector`-based metrics, archived alert log.
- Migrate the partners, telemed, sharelinks, metrics snapshot, files/media sweeps and safety nightly tickers to scheduler entries. Persist safety results (B6).
- Docs: `docs/engine-v6/jobs.md` (job catalogue, idempotency rules).

**touches:** `internal/platform/queue`, `internal/auth/module.go` (lifecycle), `internal/partners`, `internal/telemed`, `internal/sharelinks`, `internal/metrics`, `internal/files`, `internal/media`, `internal/safety`, the matching routes files.

**deps:** P01.

**tests:**
- miniredis/asynq tests: a unique task is deduped, scheduled tasks are enqueued once per tick, retry → archive.
- No behaviour change in the swept jobs.

### T-M9-P10 — Async recompute, nightly engine snapshots, analysis precompute

**Scope:**
- `engine:warm` (unique per uid, 30 s) dispatched by `Invalidate`.
- `engine:snapshot:nightly` → new goose table `engine_daily_snapshots(user_id, date, predicted_next_start, confidence, data_quality, requires_user_input, main_phase, …)` with a mirror Laravel migration per the schema rule. It feeds the §5.6 KPIs (MAE, ±2 d accuracy, confidence distribution, requires_user_input resolution).
- `analysis:refresh` (debounced) + nightly; the analysis GETs read `an:{uid}:{v}:{gv}:{kind}:{range}` and compute inline on a miss.
- Cap `range=all` reads (aggregate in SQL or bound to the first log).

**touches:** `internal/engine`, `internal/analysis`, `internal/metrics` (KPI source), `db/migrations`, `backend/database/migrations` (mirror), `db/queries`.

**deps:** P04, P09.

**tests:**
- Snapshot idempotent per (user, date).
- KPI MAE over a seeded fixture equals a hand-computed value.
- Analysis cached = inline-computed (equality).

**budgets:** nightly run for 10k synthetic users ≤ 10 min on the `low` queue with API p95 unaffected (k6 soak during the run); `/analysis/summary` warm ≤ 20 ms.

### T-M9-P11 — Worker process (stage/beta)

**Scope:**
- An `api worker` subcommand (asynq server + scheduler + pollers); `API_RUN_WORKERS=0` on the API container.
- A `backend-go-worker` service in `docker-compose.stage.yml` and `docker-compose.beta.yml` sharing the storage volumes; deploy-skill docs updated.
- Prod compose is untouched.

**touches:** `cmd/api/main.go`, `docker-compose.stage.yml`, `docker-compose.beta.yml`, `.claude/skills/deploy-stage/SKILL.md`, `.claude/skills/deploy-beta/SKILL.md`.

**deps:** P09.

**tests:** compose config lint; a smoke test that an OTP SMS job is processed by the worker container on stage.

### T-M9-P12 — HTTP hardening and conditional GETs

**Scope:**
- gzip/brotli for JSON (nginx `deploy/*.inc` for stage/beta vhosts, or Fiber `compress`).
- Engine-version ETag + `Cache-Control: private, no-cache` + a 304 short-circuit before any load.
- Fiber Read/Write/Idle timeouts; per-request DB deadline.
- `/ready` (DB + Redis ping) used by the deploy scripts' wait loop.
- Frontend `apiClient` honours 304 (browser cache).

**touches:** `cmd/api/app.go`, `internal/platform/httpx`, `internal/http/routes_health.go`, `deploy/*.inc` (stage/beta), `frontend/src/shared/api/apiClient.ts` (verify only).

**deps:** P03.

**tests:**
- ETag changes after a write and after a midnight rollover.
- 304 path issues ≤ 1 Q.
- Gzip ratio check on `/cycle/month?view=full`.

### T-M9-P13 — Redis and MariaDB tuning (stage/beta)

**Scope:**
- Redis `maxmemory` + `volatile-lru`; an audit that every cache `Set` has a TTL (B4/B5).
- MariaDB conf mount: buffer pool, slow log at 0.2 s, performance_schema.
- Redis client `PoolSize` and timeouts from env.

**touches:** `docker-compose.stage.yml`, `docker-compose.beta.yml`, `deploy/mariadb/*.cnf` (new), `internal/platform/config`, `internal/platform/cache`.

**deps:** none.

**tests:** a lint test that scans for `cache.Set(..., 0)` outside an allow-list; verify `CONFIG GET maxmemory-policy` on stage after deploy.

### T-M9-P14 — k6 load tests and SLO gate

**Scope:**
- `backend-go/loadtest/` with a seed script (2k users), token minting via test-OTP (local/stage), scenarios `app_open`, `calendar_browse`, `log_burst`, `analysis_tab`, `fertility_tab`, `soak`, and `thresholds` = the §7.7 SLOs.
- `make loadtest`.
- Results table in `docs/engine-v6/perf-baseline.md` before (after P02) and after (P05/P06/P07/P12).

**touches:** `backend-go/loadtest/*`, `Makefile`, `docs/engine-v6/perf-baseline.md`.

**deps:** P01, P02 (baseline run); re-run after P05/P12.

**budgets:** §7.7 table; the `log_burst` check asserts that `GET /cycle/today` right after a write reflects it (0 stale reads).

### T-M9-P15 — User-local day (B3, spec §2.3)

**Scope:**
- A timezone on the profile (default Asia/Tehran) and/or a validated `X-Timezone` header.
- One `civildate.LocalToday(c)` used by every engine and module and by the ETag and response-cache keys.
- Frontend sends the device timezone.

**touches:** `internal/platform/civildate`, `internal/cycle/service`, `internal/home`, `internal/messages`, `internal/fertility`, `internal/analysis`, `frontend/src/shared/api`, migration (profile column + Laravel mirror).

**deps:** P04.

**tests:**
- A user in UTC−5 at 23:30 local / 08:00 Tehran gets her local date.
- Cache keys differ per local day.

**Suggested order:** (P01, P03, P08, P13 in parallel) → P02 → P04 → (P05, P06, P07, P09, P12) → (P10, P11) → P14 (final gate) → P15 (can run after P04 in parallel).

# Performance baseline — frontend + backend (T-M1-10)

> Measured 2026-09-19. This is measurement only: no application code was changed.
> Every number below comes with the command and environment that produced it (§6 has the
> scripts), so T-M1-11 and T-M1-12 can re-run the same method and append a before/after table.

## 0. Environments and caveats

| Label | What it is |
| --- | --- |
| **LOCAL-FE** | `frontend/` at commit `393373c` (HEAD; no uncommitted `frontend/src` changes when it was built at 15:51). `npm run build` with `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1`, served by `npx next start -p 3110 -H 127.0.0.1`. Apple M4, 10 cores, 16 GB, Chrome 153, Lighthouse 12.8.2 |
| **LOCAL-BE** | `backend/` at HEAD, PHP 8.4.6, Laravel 12.41.1. Runs on a **copy** of `backend/database/database.sqlite` in the scratchpad (not the shared DB), `CACHE_STORE=redis` (isolated DB 12/13 with a key prefix, like prod), `QUEUE_CONNECTION=sync`, `APP_DEBUG=false`. Served by `php -S 127.0.0.1:8020` with `PHP_CLI_SERVER_WORKERS=4` |
| **Seeded user** | A new user (id 33) created through the real API: profile + **13 logged periods** over 12 months (cycles of 27–31 days) + **60 daily health logs** (the last 60 days). All seeding went through `POST /profile`, `POST /cycle/period` and `POST /health-logs`, so it also measured write latency |
| **PROD** | `root@89.251.8.115` (2 vCPU "Common KVM", 4 GB), MariaDB 11.4, Apache mpm_prefork + mod_php 8.4.25, ArvanCloud CDN in front. **Read-only access only**: HTTP GETs on public URLs, `information_schema`, `EXPLAIN`/`ANALYZE SELECT`, `ls`/`php -r` in the container. No token was minted and no authenticated prod call was made |

Caveats that affect how to read the numbers:
- **CPU ratio, prod vs LOCAL.** The same 20M-iteration PHP loop took **727 ms on prod and 206 ms locally (3.5×)**
  (`docker compose exec -T backend php -r '…for($i=0;$i<20000000;$i++)…'`). The prod estimates below
  (`≈ prod`) are LOCAL × 3.5. They are estimates, not measurements.
- **Machine load.** Other agents were building and testing at the same time (load average 9–15 at 16:15). The
  in-process harness numbers (§2.1) were taken early and are stable. The later HTTP A/B numbers (§2.4) are
  noisy, so they are interleaved and I quote the medians.
- **Loopback Lighthouse.** On `127.0.0.1`, Chrome reports `transferSize=0` for documents, scripts and fonts,
  so Lighthouse's *simulated* throttling under-counts bytes, and local LCP is optimistic. Use the local runs for
  relative comparisons. For absolute bytes on the wire, use the prod run (§1.4).
- **No INP.** A Lighthouse navigation run cannot measure INP. TBT is the proxy used here.
- **Build ownership.** Someone ran another `next build` in `frontend/` at 16:13 (`.next/BUILD_ID` mtime), after
  other agents started editing `frontend/src` at 16:02. All frontend measurements here finished by 16:11:35 and
  used the 15:51 build of HEAD. The 3110 server was stopped once that happened.

## 1. Frontend numbers

### 1.1 `next build` route table (first-load JS, as reported by Next 15.5.18)

Command: `cd frontend && NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1 NEXT_TELEMETRY_DISABLED=1 npm run build` (20.8 s wall, warm webpack cache)

| Route | Page size | First Load JS |
| --- | ---: | ---: |
| `/[locale]/home` | 8.22 kB | **209 kB** |
| `/[locale]/calendar` | 8.02 kB | **207 kB** |
| `/[locale]/cycle` | 4.25 kB | 203 kB |
| `/[locale]/pregnancy/onboarding` | 7.51 kB | 203 kB |
| `/[locale]/profile` | 5.33 kB | 200 kB |
| `/[locale]/onboarding/setting-up` | 5.86 kB | 196 kB |
| `/[locale]/log` | 9.35 kB | **192 kB** |
| other onboarding/auth routes | 1.9–7 kB | 182–194 kB |
| `/[locale]/splash`, `/welcome` | 0.8 / 3.3 kB | 153 / 150 kB |
| shared by all | — | 103 kB (React-DOM 54.2 + Next runtime 46.3) |
| Middleware | — | 64.7 kB |

Every page is SSG (`●`) and renders as a client shell. All 23 `screens/*/ui/*Page.tsx` files are `'use client'`
(`git grep -l "'use client'" HEAD -- 'frontend/src/screens/*/ui/*Page.tsx' | wc -l` → 23 of 23).
Data is fetched after hydration.

### 1.2 Bundle composition (ad-hoc `@next/bundle-analyzer`, not committed)

Method: I rsynced `frontend/` (without `node_modules`/`.next`) to the scratchpad and symlinked `node_modules`. I wrapped
`next.config.ts` with `@next/bundle-analyzer` (installed in the scratchpad only, `analyzerMode: 'json'`), ran
`npx next build`, and aggregated `.next/analyze/client.json` by package and FSD slice (script in §6.4).

First-load chunks for `/[locale]/home` add up to **695 KB parsed / 213 KB gzip** (21 files). By owner:

| Owner (on every route) | Parsed KB | Gzip KB | Note |
| --- | ---: | ---: | --- |
| react-dom (Next-compiled) | 169 | 53 | framework, unavoidable |
| next runtime | 167 + 22 | 45 + ~7 | framework |
| **zod** | **55.5** | **12.5** | v3, loaded on every route (schemas in every entity) |
| **axios** | **40.2** | **15.4** | a `fetch` wrapper would do the same job |
| **intl-messageformat** + use-intl | 39.0 + 10.8 | 11.3 + 5.1 | next-intl client runtime |
| @tanstack/query-core + react-query | 55 + 32 | 20 + 14 | spread over 10–15 chunks (tiny duplicates such as `mutation.js` 3.4 KB in many chunks) |
| zustand | 39.6 total | 20.5 | **`middleware.mjs` (2 KB) duplicated in 16 chunks** |
| dayjs + jalaliday | 7.7 + 5.8 | 3.4 + 2.7 | small, not a target |
| `src/screens/home` | 24.0 | 7.8 | biggest app slice (`HomePage.tsx` = 1,076 lines) |
| `src/shared/sheet` | 23.1 | 9.8 | across 6 chunks; sheets already use `next/dynamic` (`app/sheets/registry.tsx`) |

Whole client output: 1,330 KB parsed / 424 KB gzip in 65 chunks.

### 1.3 HTML, messages, CSS, fonts, images (static artefacts of the same build)

| Item | Measured | Command |
| --- | --- | --- |
| `/fa/home` prerendered HTML | 74,210 B (19,304 B gzip), RSC payload 63,446 B | `wc -c .next/server/app/fa/home.{html,rsc}`, `gzip -c … \| wc -c` |
| i18n messages serialised into **every** page | all 23 `fa` namespaces = **50,228 B minified / 14,467 B gzip** | `NextIntlClientProvider messages={await getMessages()}` in `app/[locale]/layout.tsx`; node one-liner in §6.4 |
| Global CSS | `globals.css` source 150 KB → one 114,466 B stylesheet (22 KB br on prod), render-blocking on every route | `ls -la .next/static/css` |
| **Fonts** | **6 Vazirmatn weights (400/500/600/700/800/900), each ~21 KB, all 6 `<link rel=preload>` on every page = 128 KB** | `grep -o '<link[^>]*rel="preload"[^>]*>' .next/server/app/fa/home.html` |
| Font-weight usage in `src/` | 400 (body default); **500: 2 uses** (1 CSS + 1 `font-medium`); 600: 32+3; 700: 98+13; 800: 83+1; 900: 10 | `grep -rhoE "font-weight:\s*[0-9]+"`, `font-(medium\|…)`, `fontWeight:` |
| Images | not a hotspot: `next/image` used for the logo only (6 `<img>`/`Image` uses in total). `icons/icon-512.png` is 154 KB but only the manifest references it. `icon-192.png` (32 KB) is fetched twice on the prod welcome page (2nd hit 0 B) | `find public -size +50k`, Lighthouse network list |

### 1.4 Lighthouse (mobile preset: Moto G Power, slow 4G, 4× CPU), median of 3

Login state (`localStorage.ritme_token` + `ritme_auth=1` cookie for the seeded user) was injected over CDP into a
headless Chrome on `--remote-debugging-port=9322`. Lighthouse then attached to it with
`lighthouse <url> --port=9322 --disable-storage-reset --only-categories=performance`. Before each run,
`Storage.clearDataForOrigin(service_workers,cache_storage)` + `Network.clearBrowserCache` gave a first visit;
without that clearing, the service worker served everything (a repeat visit).

**A. First visit, SW + HTTP cache cleared, simulated throttling (LOCAL)**

| Route | Score (3 runs) | FCP ms | LCP ms | TBT ms | CLS | SI ms | TTI ms | Bootup ms | Main-thread ms |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `/fa/home` | 90/99/99 | 687 | 1098 | 14 | **0.063** | 687 | 1098 | 368 | 818 |
| `/fa/calendar` | 97/100/100 | 670 | **1769** | 1 | 0.012 | 670 | 1769 | 287 | 691 |
| `/fa/log` | 100/97/99 | 652 | 1080 | 119 | 0.000 | 804 | 1249 | 275 | 842 |

**B. Same, but with `--throttling-method=devtools` (throttling actually applied, so it is harsher on layout shifts)**

| Route | Score | FCP | LCP | TBT | CLS | SI | TTI |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `/fa/home` | 90/90/90 | 557 | 557 | 0 | **0.201** | 639 | 557 |
| `/fa/calendar` | 97/100/100 | 513 | 1585 | 14 | 0.012 | 1463 | 2224 |
| `/fa/log` | 100/100/100 | 437 | 437 | 43 | 0.000 | 464 | 1263 |

**C. Repeat visit (SW warm, simulated)**: home 99/100/99, LCP 1029, TBT 51, CLS 0.063. Calendar LCP 1856. Log LCP 1070.

LCP elements: home `div.home-phase-desc` and calendar `div.dls-empty-sub` are both text that renders **after API data
arrives**. Log is `p.log-hdr-sub`. The home CLS culprit is `<div class="sec">` (0.179 + 0.022 in run B-1): sections pop
in with no reserved height when data lands.

**D. PROD public page (real network through the CDN)** — `CHROME_PATH=… lighthouse https://web.ritme.app/fa/welcome --chrome-flags=--headless=new`, 2 runs:
score 91/91, FCP 1223/1380, **LCP 3482/3489**, TBT 0, CLS 0, TTFB 193. **418 KB total: fonts 128 KB (31 %)**,
scripts 202 KB, CSS 23 KB, document 22 KB, icons 42 KB, 62 requests. Lighthouse "unused JavaScript" estimate: 89 KiB.

### 1.5 API requests on a cold load, and TanStack Query duplicates

Method: CDP network capture with the cache disabled (`scratchpad/cdp.mjs capture <route>`), counting the GETs to `:8020`
and their CORS preflights.

| Cold load | API GETs | Preflights | GETs issued |
| --- | ---: | ---: | --- |
| `/fa/home` | **11** | **11** | `/cycle/today`, `/messages/daily`, `/cycle/status`, **`/cycle/month/2026/8`**, **`/cycle/month/2026/9`**, `/cycle/period/history`, `/profile`, `/banners`, `/home/sections/challenge`, `/home/sections/articles`, `/messages/mode` |
| `/fa/calendar` | 7 | 7 | `/health-logs/{today}`, `/cycle/period/history`, `/profile`, `/messages/mode`, `/cycle/status`, `/cycle/month/…` ×2 |
| `/fa/log` | 3 | 3 | `/messages/mode`, `/health-logs/enums`, `/health-logs/{today}` |

- **Every cross-origin GET carries `Authorization`, so each one costs a preflight.** A cold home load is 22 round
  trips to `api.ritme.app`. `Access-Control-Max-Age: 3600` is set, but Chrome keys the preflight cache per URL,
  so each new URL (and each new month) preflights again.
- Payload: the home API responses total **274 KB uncompressed, and 261 KB of that is the two `/cycle/month`
  responses** (143 KB + 118 KB). Prod compresses them over the CDN (`content-encoding: br`), but the phone still
  parses them and the server still builds and encodes them.
- **TanStack duplicates: none.** In-app flow home → calendar → log → home → calendar → cycle → profile → home
  (`scratchpad/flow.mjs`, clicking the bottom-nav links) issued 17 GETs in total, and no URL was fetched twice.
  The global `staleTime` is 60 s and entity overrides are 5–30 min. Only these GETs happened after the first load:
  `/health-logs/{date}`, `/health-logs/enums`, the three `/home/sections/*` of `/cycle`, and `/languages` on profile.
- `useCycleStatus({poll})` polls `/cycle/status` **every 4 s while `calculation_status=processing`**. That state
  exists only because of the dead-storage job (§2.3).

### 1.6 Dead frontend code (`npx knip@5 --no-progress --reporter compact`, in `frontend/`)

Checked by hand against HEAD with `git grep`:
- **Unused dependencies**: `react-hook-form`, `@hookform/resolvers` (no imports at all, so no bundle impact, just install weight).
- **Unused files (real)**: `src/shared/lib/cookie-state/*` (3 files), `src/screens/home/ui/SectionHead.tsx`
  (a near-duplicate of `screens/cycle/ui/SectionHead.tsx`), `widgets/day-tasks/*` (only named in comments/i18n
  keys, so verify before deleting), `frontend/sample/**` (old HTML prototype, not built).
- **Unused but must NOT be deleted blindly**: `features/manage-account` (`useExportData`, `useDeleteAccount`,
  `DeleteAccountConfirm`) is **not wired into any screen**. CLAUDE.md §11 requires data export/delete, so this is
  a product gap to raise, not dead code.
- **False positives**: `public/sw.js`, `scripts/sw.template.js`, `steiger.config.ts`, `src/shared/i18n/request.ts`
  and `messages.ts` (loaded by the next-intl plugin), `eslint-config-next`, and the steiger plugin.
- 56 "unused exports", mostly `fetchX`/`xKeys` re-exported from slice `index.ts`. Webpack tree-shakes them, so
  there is no runtime cost. Cleaning them up is optional.

## 2. Backend numbers

### 2.1 Hot endpoints — p50/p95 and query count (LOCAL-BE, in-process harness, seeded user)

Method: `scratchpad/be/harness.php` (§6.1) boots a **fresh Laravel app per request**, as FPM does with opcache.
It times `$kernel->handle()` (routing + middleware + Passport auth + controller + JSON), logs every SQL statement
with `DB::listen`, and runs 1 warm-up plus **30 measured iterations per path**. Command:
`source env.sh && S=$S php -d memory_limit=2G harness.php paths.txt 30 h30.json`.
"Dup" counts exact repeats (same SQL + bindings) within one request. "Queries" includes 3 Passport auth queries on
every authenticated request (`oauth_access_tokens` exists, `oauth_clients`, `users`).

| Endpoint | Status | Queries | Dup | p50 ms | p95 ms | Bytes | ≈ prod p50 (×3.5) |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| **`/cycle/month/2026/9`** | 200 | 7 | 0 | **46.9** | **54.2** | **118,104** | ~165 |
| `/home` (aggregated; Android only) | 200 | **37** | 3 | 28.8 | 30.4 | 18,683 | ~100 |
| `/home/sections/week_calendar` (Android only) | 200 | 14 | 0 | 18.3 | 20.2 | 1,728 | ~64 |
| `/messages/daily` | 200 | 14 | 0 | 15.0 | 21.2 | 4,465 | ~53 |
| `/home/sections/smart_tip` | 200 | 14 | 0 | 14.5 | 17.8 | 890 | ~51 |
| `/health-logs` (list, 60 rows) | 200 | 5 | 0 | 12.2 | 39.0 | 45,644 | ~43 |
| `/cycle/today` | 200 | 8 | 1 | 10.7 | 15.5 | 5,799 | ~37 |
| `/cycle/date/2026-09-19` | 200 | 8 | 1 | 10.6 | 15.5 | 5,799 | ~37 |
| `/cycle/date/2026-10-05` (future) | 200 | 8 | 1 | 10.4 | 15.5 | 4,999 | ~36 |
| `/home/sections/my_cycles` | 200 | 9 | 0 | 10.8 | 12.9 | 3,168 | ~38 |
| `/home/sections/cycle_summary` | 200 | 9 | 0 | 10.7 | 12.3 | 2,831 | ~37 |
| `/home/sections/challenge` | 200 | 12 | 0 | 10.2 | 12.3 | 613 | ~36 |
| `/home/sections/tasks` | 200 | 10 | 0 | 9.6 | 13.8 | 2,428 | ~34 |
| `/home/sections/next_period` | 200 | 8 | 0 | 9.5 | 11.4 | 753 | ~33 |
| `/home/sections/articles` | 200 | 9 | 0 | 9.4 | 12.7 | 1,497 | ~33 |
| `/health-logs/2026-09-18` | 200 | 4 | 0 | 8.0 | 16.4 | 1,518 | ~28 |
| `/languages/fa/messages` (public) | 200 | 0 | 0 | 7.8 | 16.1 | 110,938 | ~27 |
| `/home/sections/weekly_summary` | 200 | 7 | 0 | 7.6 | 10.7 | 777 | ~27 |
| `/languages` (public) | 200 | 0 | 0 | 7.6 | 15.5 | 262 | ~27 |
| `/health-logs/enums` | 200 | 3 | 0 | 7.4 | 17.3 | 1,227 | ~26 |
| `/home/sections/status_charts` | 200 | 6 | 0 | 7.4 | 20.0 | 40 | ~26 |
| `/profile` | 200 | 5 | 0 | 7.2 | 13.6 | 2,183 | ~25 |
| `/auth/user` | 200 | 4 | 0 | 7.1 | 13.5 | 272 | ~25 |
| `/home/notifications` | 200 | 5 | 0 | 7.0 | 11.0 | 123 | ~25 |
| `/cycle/status`, `/cycle/period/*`, `/messages/mode`, `/banners`, `/home/sections/header` | 200 | 4–6 | 0 | 6.8–7.0 | 8–13 | ≤1.3 K | ~24 |

**The frontend's cold home load (the 11 GETs in §1.5)** adds up to **≈174 ms of server handle time locally
(≈600 ms of prod CPU)** and **79 SQL queries**. Of that, **the two `/cycle/month` calls are 94 ms (54 %)** and
**33 queries (42 %) are Passport auth** (3 per request).

End-to-end HTTP through `php -S` (includes framework boot), `httpbench.sh …/api/v1 30 <paths>`, sequential, n=30:
`/cycle/month` p50 61.0 / p95 63.7 ms · `/home` 46.6 / 55.1 · `/home/sections/week_calendar` 35.6 / 39.2 ·
`/messages/daily` 32.2 / 36.6 · `/cycle/today` 27.7 / 29.0 · `/auth/user` 23.9 / 28.6 · `/languages` 17.7 / 21.0.

**PROD origin, public endpoints only** (from the server host to `127.0.0.1:8080` with `Host: api.ritme.app`, 15
sequential GETs after 1 warm-up): `/api/v1/languages` p50 **36.0** / p95 47.0 ms, `/api/v1/info/privacy` p50 **46.4** /
p95 64.9 ms. Through the CDN from outside Iran: `/languages` TTFB 398 ms, `/languages/fa/messages` 501 ms (22 KB br),
`web.ritme.app/fa/splash` 1.23 s TTFB (a single `curl -w` sample each; this is mostly network distance).

### 2.2 Query analysis — N+1s, duplicates, and indexes

Query shapes from the harness (`jq '.["<path>"].top_shapes' h30.json`):
- **N+1 per-day health-log lookups**: `/home` runs `select * from daily_health_logs where user_id=? and
  date(log_date)=?` **9×** (plus 3 range queries of the same shape), and `/home/sections/week_calendar` runs it **7×**.
  `whereDate` wraps the column in `date()`/`strftime()`. Only the frontend's `/cycle/month` preloads with
  `HealthDataEngine::preloadDailyLogs()`. The aggregated `/home` is called by the Android app only
  (`application/…/ApiConfig.kt HOME_PATH`); the web app uses `/home/sections/*`.
- **The same `cycle_histories` read twice** in `/cycle/today` and `/cycle/date/*` (1 dup each), and 2× in `/home`.
- **`message_contents` single-row lookups ×4** per request in `/messages/daily` and `/home/sections/smart_tip` (4 in
  `/home`). This is static admin content keyed by `(group, item_key, locale)`, so it can be cached.
- **Auth: 3 queries on every authenticated request** (Passport token + client + user), and the `user_profiles` +
  `pregnancy_profiles` lookups repeat in most endpoints.
- **Indexes on PROD are adequate at the current size.** `EXPLAIN` on MariaDB (read-only, §6.5): the `date(log_date)`
  lookups still use `daily_health_logs_user_id_log_date_unique` (range on the `user_id` prefix, "Using index
  condition"), `cycle_histories` uses the `(user_id, period_start_date)` unique index, `message_contents` resolves
  on the `(group,item_key,locale)` unique, and `user_notifications` uses `(user_id, read_at)` with a small filesort.
  **No missing index was found.** One cleanup: `cycle_histories` carries both a UNIQUE and a plain INDEX on the same
  `(user_id, period_start_date)`, so the plain one is redundant.

### 2.3 Writes and the dead-storage recalculation job (the largest backend cost)

Every write (`POST /profile`, `/cycle/period*`, `/health-logs`, `/cycle/recalculate`) sets
`calculation_status=processing`, increments `calculation_version`, and dispatches `CalculateCycleDataJob`. That job
computes **366 days** with `HealthDataEngine::calculateForDate()` and upserts them into `cycle_calculations`
**under the new version**, so **every write adds 366 rows**. Nothing reads them back (memory `ritme-cycle-engine`,
and `grep` finds only the admin row count).

| Measurement | Value | Command / source |
| --- | --- | --- |
| Job run, seeded user (same version, upsert in place) | **620–943 ms** (first run 1,768 ms), **376 queries** | `php jobbench.php 33` (§6.3), 5 runs |
| Engine alone, 367 days, no writes | 712 ms, **368 queries = 1 `daily_health_logs` query per day** (the job does not call `preloadDailyLogs`) | same script |
| Stored row size | ~3.8 KB JSON per row → ~1.4 MB written per user write | same script |
| Write latency incl. sync job (LOCAL, `QUEUE_CONNECTION=sync`) | `POST /cycle/period` p50 349 / p95 491 ms (n=12); `POST /health-logs` p50 **542** / p95 724 ms (n=60) | `seed.mjs` timings (`seed-writes.txt`) |
| Seeded user after 74 writes | 27,084 rows / 74 versions (LOCAL) | `sqlite3 … count(*)` |
| **PROD table** | **`cycle_calculations` 89,304 rows for 37 users, 238.3 MB data + 7.0 MB index = ~99 % of the database** (next largest table: 2.5 MB). One user has 19,764 rows / 54 versions | `information_schema.tables`, `SELECT … GROUP BY user_id` |
| ≈ prod CPU per write | ~2–3 s of queue-worker CPU (×3.5) on a 2-vCPU box shared with web | estimate |

In prod the job runs on the redis queue worker, so it is off the request path. The client still sees `processing`
and polls `/cycle/status` every 4 s until the worker finishes. Meanwhile every read endpoint computes live anyway
(`/cycle/month` says "Always calculate fresh"). The job therefore costs CPU, disk, and a stale "updating" UI state,
and gives nothing back.

### 2.4 Framework boot: no config/route/event cache in the prod image

- PROD `bootstrap/cache/` contains only `packages.php` and `services.php`. **No `config.php`, `routes-v7.php`, or
  `events.php`** (`docker compose exec -T backend ls -la bootstrap/cache`). `docker/entrypoint.sh` never runs
  `config:cache` / `route:cache` / `event:cache`.
- The rest is fine: opcache is on with `validate_timestamps=0` and `memory_consumption=192`, and composer uses
  `dump-autoload --optimize`.
- Effect measured locally without touching `bootstrap/cache`. `APP_CONFIG_CACHE`/`APP_ROUTES_CACHE`/`APP_EVENTS_CACHE`
  pointed to scratch files. I ran `bootbench.php` (fresh process: bootstrap + `GET /api/v1/languages`) 30× interleaved
  A/B: **no caches p50 66.9 ms → cached p50 52.6 ms (−14.3 ms, −21 %)**. min went from 44.7 to 37.8 ms, and p25 from
  56.7 to 47.0 ms. The CLI numbers include autoload and compile, which opcache in Apache avoids, so treat the absolute
  numbers loosely and the delta as the signal. Rough prod value: tens of ms per request on the 2-vCPU box.

### 2.5 Cacheable per-user computations

The cycle engine is a pure function of (profile, `cycle_histories`, `daily_health_logs` in range, locale, **Tehran
"today"**). `/cycle/month` (~47 ms locally for 31 days, ~1.5 ms/day), `/cycle/today` and `/cycle/date/*` recompute it
on every request. `user_profiles.calculation_version` already increments on **every** write that changes the engine's
inputs, so a key like `cycle:{user}:{calculation_version}:{locale}:{tehranToday}:month:{y}-{m}` is invalidated for
free. It must keep incrementing after the job is removed (T-M1-13).

Payload trimming on `/cycle/month` (`php -r` over the saved response, §6.4): the full response is **118,104 B**.
With `JSON_UNESCAPED_UNICODE` it is 83,897 B (−29 %). **Keeping only the fields `cycleCalculationSchema` reads
(`frontend/src/entities/cycle/api/schema.ts`) minus `daily_tips` gives 8,390 B (−93 %)**, and gzip goes from 5.9 KB
to 0.6 KB. `daily_tips`/`text_flags`/`source_*` are 563 + 563 + 98 chars per day. The calendar and week strip do not
use per-day tips (`HomePage` reads `dailyTips` from `/cycle/today`).

## 3. Top 10 opportunities, ranked by impact ÷ effort

Impact/effort: H/M/L. Ranking weighs user-visible latency, prod CPU/disk, and risk.

| # | Opportunity | Evidence | Impact | Effort | Task |
| --- | --- | --- | --- | --- | --- |
| 1 | **Stop `CalculateCycleDataJob` and the `cycle_calculations` writes.** Keep bumping `calculation_version` and set status `completed` synchronously, so `/cycle/status`, `/cycle/recalculate` and `is_recalculating` keep their contract. Then drop the table in a new migration | 366 rows + 376 queries + 0.6–0.9 s local (~2–3 s prod) worker CPU per write; prod table 238 MB = ~99 % of DB for 37 users; it also drives the 4 s status polling | H | S | **T-M1-13** (bump/sync-status part coordinated with T-M1-12's cache key) |
| 2 | **`config:cache` + `route:cache` + `event:cache` at container start** (entrypoint, since env is runtime) | −14 ms p50 (−21 %) per request locally; missing on prod | M | XS | **T-M1-12** |
| 3 | **Slim `/cycle/month`**: an opt-in calendar view (`?view=calendar` or similar, backward compatible for Android) that returns only the fields the calendar reads and skips per-day tips/text work, plus `JSON_UNESCAPED_UNICODE` | 118–143 KB → ~8.4 KB per month (−93 %); 2 calls per cold home/calendar = 54 % of home server time | H | S–M | **T-M1-12** (API) + **T-M1-11** (consume it) |
| 4 | **Fonts**: stop preloading 6 weights. Either use one variable Vazirmatn woff2, or drop 500 (2 uses → 400/600) and 900 (10 uses → 800) and preload only the body weight | 128 KB of fonts preloaded on every page = 31 % of prod page weight (418 KB); prod LCP 3.5 s | M–H | S | **T-M1-11** |
| 5 | **Reserve space for home sections** (skeleton/min-height on `.sec`) so data arrival doesn't shift layout | CLS 0.063 (simulated) / **0.201** (devtools-throttled) on `/fa/home`, culprit `div.sec` | M | S | **T-M1-11** |
| 6 | **Per-user cache of cycle-engine results** (month/today/date), keyed by `calculation_version` + locale + Tehran date | `/cycle/month` 46.9 ms and `/cycle/today` 10.7 ms locally recomputed on every hit; ≈165 / 37 ms on prod | M–H | M | **T-M1-12** |
| 7 | **Cut the query noise**: preload `daily_health_logs` for the range (kills 9× N+1 in `/home` and 7× in `week_calendar`), memoise `cycle_histories` per request (1–2 dups), cache `message_contents` per `(group, locale)` with admin-save invalidation (4 queries/request) | query counts in §2.1/§2.2 | M (grows with MariaDB round-trip) | S | **T-M1-12** |
| 8 | **Fewer cold-load round trips**: 11 GETs + 11 preflights on home. Merge the tiny boot reads the app fires together (`/cycle/status`, `/messages/mode`, `/profile`, `/banners`) or, better, serve the API same-origin (`web.ritme.app/api/*` proxied, as staging already does), which removes every preflight | §1.5 | M (mobile RTT) | M | **T-M1-11** (client side); same-origin proxy is infra — see open items |
| 9 | **Trim the client JS every route pays for**: pass only the namespaces a route needs to `NextIntlClientProvider` (50 KB/14.5 KB gz of messages in every HTML/RSC), replace axios with a thin `fetch` client (−15 KB gz), consider zod 4 / `zod/mini` (55 KB parsed) | §1.2, §1.3; TBT is already low (1–119 ms) so the win is parse/transfer on low-end phones | M | M | **T-M1-11** |
| 10 | **Dead code/deps cleanup** — FE: `react-hook-form` + `@hookform/resolvers`, `shared/lib/cookie-state`, `screens/home/ui/SectionHead.tsx`, `widgets/day-tasks` (verify), `frontend/sample/`. BE: MatrixEngine (5 files, 2,107 lines) behind `/cycle/matrix-messages` + `/cycle/matrix-enums`, which no client calls (frontend 0, Android 0 grep hits); `/cycle/enums` and `/messages/enums` (0 client hits); local-only `Test*Controller`s + 4 `test-*.blade.php` (1,534 lines); `MessageSystem/Contracts/ContextProviderInterface` (0 refs); redundant `cycle_histories` index | knip (§1.6) + `grep -rlw` sweep | L (hygiene) | S | FE → **T-M1-11**, BE → **T-M1-13** (index → T-M1-12 migration) |

Deliberately **not** on the list, because the measurements don't justify it: moving screens to Server Components
(all screens are data-driven client shells, TBT is already 1–119 ms, and the cost is a big refactor), dayjs/jalaliday
(6 KB gz), images, missing DB indexes (none found), and TanStack duplicate fetches (none found).

## 4. Mapping and ordering notes for the follow-up tasks

- T-M1-12 and T-M1-11 run in parallel (group M1-E). Item 3 needs the backend part first. The frontend should
  switch to the slim month view only once T-M1-12 has shipped it. If T-M1-12 hasn't landed, T-M1-11 skips that
  sub-item and records it as a follow-up.
- T-M1-13 runs after T-M1-12. Item 1 changes what `calculation_version`/`calculation_status` mean. T-M1-12's cache
  key must use `calculation_version`, and T-M1-13 must keep incrementing it on every engine-input write when it
  deletes the job. Both the frontend (`useCycleStatus`, `useRecalculateCycle`) and the Android app still call
  `/cycle/status` and `/cycle/recalculate`, so those routes stay.
- Re-measure with §6 on a **quiet machine**, using the same seeded-user recipe (§6.2), so the before/after tables
  compare like with like.

## 5. Open items (not resolved by this baseline)

1. **No authenticated prod timings.** Getting them needs a token (a DB write), which is out of scope for read-only
   access. The prod figures are ×3.5 estimates plus public-endpoint measurements. If the owner wants real prod
   p95s, add a read-only `request_time` field to the nginx `log_format` (infra change) and read the logs.
2. **Same-origin API** (item 8, the preflight removal) touches `deploy/vhost-web.inc`, `backend/config/cors.php`
   and `NEXT_PUBLIC_API_BASE_URL`, all outside T-M1-11's `touches`. It needs an owner decision and a deploy.
3. **`features/manage-account` is not wired into the UI.** That is a CLAUDE.md §11 gap (data export/delete), not
   dead code. The product owner should decide.
4. **The next-intl client runtime** (`intl-messageformat`, 11 KB gz) stays as long as client components format ICU
   messages. Removing it would mean moving copy to Server Components, which is out of M1 scope.
5. **Standalone tracing root warning.** `next build` infers `/Users/rezataheri` as the workspace root because of a
   stray `~/package-lock.json`. It is harmless in Docker but pollutes local `.next/standalone`. Set
   `outputFileTracingRoot` if it matters.
6. **Build ownership.** Another `next build` ran in `frontend/` at 16:13 during this batch (§0).

## 6. How to reproduce (all scripts lived in the session scratchpad; the essentials are here)

### 6.1 Backend in-process harness (`harness.php`)

```php
<?php // usage: source env.sh && S=$S php -d memory_limit=2G harness.php paths.txt 30 out.json
$root = '/path/to/ritme/backend'; require $root.'/vendor/autoload.php';
$token = trim(file_get_contents(getenv('S').'/be/token')); $iters = (int) ($argv[2] ?? 20);
foreach (array_filter(array_map('trim', file($argv[1]))) as $path) {
  $times = []; for ($i = 0; $i < $iters + 1; $i++) {                 // +1 warm-up, discarded
    $app = require $root.'/bootstrap/app.php'; $kernel = $app->make(Illuminate\Contracts\Http\Kernel::class);
    $queries = []; $app->booted(fn ($app) => $app['db']->listen(function ($q) use (&$queries) { $queries[] = [$q->sql, $q->bindings, $q->time]; }));
    $req = Illuminate\Http\Request::create('/api/v1'.$path, 'GET', [], [], [], ['HTTP_AUTHORIZATION' => "Bearer $token", 'HTTP_ACCEPT' => 'application/json', 'HTTP_ACCEPT_LANGUAGE' => 'fa']);
    $t0 = hrtime(true); $resp = $kernel->handle($req); $ms = (hrtime(true) - $t0) / 1e6; $kernel->terminate($req, $resp);
    if ($i > 0) $times[] = $ms;                                       // p50/p95 over $times; query count/dups from $queries
  }
}
```

`env.sh` (scratch backend): `DB_DATABASE=<copy of database.sqlite>`, `CACHE_STORE=redis`, `REDIS_CACHE_DB=13`,
`REDIS_DB=12`, `REDIS_PREFIX=ritmeperf-`, `QUEUE_CONNECTION=sync`, `APP_DEBUG=false`, `LOG_LEVEL=error`,
`CORS_ALLOWED_ORIGINS=http://127.0.0.1:3110`. Server: `cd backend/public && PHP_CLI_SERVER_WORKERS=4 php -S
127.0.0.1:8020 ../vendor/laravel/framework/src/Illuminate/Foundation/resources/server.php` (cwd **must** be `public/`).

Paths measured: `/home`, `/home/sections/{header,next_period,week_calendar,cycle_summary,my_cycles,smart_tip,challenge,
articles,weekly_summary,status_charts,tasks}`, `/home/notifications`, `/cycle/{today,status}`, `/cycle/date/2026-09-19`,
`/cycle/date/2026-10-05`, `/cycle/month/2026/9`, `/cycle/period/{status,history}`, `/messages/{daily,mode}`, `/banners`,
`/auth/user`, `/profile`, `/health-logs/enums`, `/health-logs/2026-09-18`, `/health-logs`, `/languages`, `/languages/fa/messages`.

HTTP bench: `for i in $(seq 1 30); do curl -s -o /dev/null -w '%{time_total}\n' -H "Authorization: Bearer $TOKEN"
-H 'Accept: application/json' -H 'Accept-Language: fa' "$BASE$p"; done | sort -n` → p50 = value 15, p95 = value 29 (1 warm-up first).

### 6.2 Seeded user

`POST /auth/send-otp {mobile}` → read the code with `sqlite3 <copy> "select code from otp_verifications where
mobile=… order by id desc limit 1"` → `POST /auth/verify-otp` → `POST /profile {birthday 1995-03-10, weight 60,
height 165, period_duration 5, cycle_duration 29, last_period_start 2025-09-20, user_goal non_ttc}` →
`POST /cycle/period {start_date, end_date=start+4}` ×13 with cycle lengths 28,30,29,27,31,29,28,30,29,28,30,29 from
2025-09-20 → `POST /health-logs {log_date, moods:[…]}` for each of the last 60 days. Every write is timed.

### 6.3 Job benchmark (`jobbench.php <user_id>`)

Bootstraps the console kernel, listens to `DB`, and runs `(new CalculateCycleDataJob($uid, $currentVersion, 'fa'))->handle()`
5× (same version, so it upserts in place and does not grow the table). It then times 367 × `HealthDataEngine::calculateForDate()`
with no writes.

### 6.4 Frontend

- Build: `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1 NEXT_TELEMETRY_DISABLED=1 npm run build`. Serve:
  `npx next start -p 3110 -H 127.0.0.1`.
- Bundle analyzer: `npm i @next/bundle-analyzer@15` **in a scratch dir**. In a scratch copy of `frontend/`, wrap the
  config as `require('<scratch>/node_modules/@next/bundle-analyzer')({enabled:true, openAnalyzer:false,
  analyzerMode:'json'})({...orig, eslint:{ignoreDuringBuilds:true}, typescript:{ignoreBuildErrors:true}})`, then run
  `npx next build`. Walk `.next/analyze/client.json` leaves, group by `node_modules/<pkg>` or `src/<layer>/<slice>`,
  and sum `parsedSize`/`gzipSize`. First-load per route comes from `app-build-manifest.json`
  (`/[locale]/layout` + `/[locale]/<route>/page`).
- Messages size: `node -e "const o={};for(const f of fs.readdirSync('messages/fa'))o[f]=JSON.parse(fs.readFileSync('messages/fa/'+f));const s=JSON.stringify(o);console.log(Buffer.byteLength(s),zlib.gzipSync(s).length)"`.
- Login injection + network capture (CDP over Node's global `WebSocket`): launch
  `Google Chrome --headless=new --remote-debugging-port=9322 --user-data-dir=<scratch> --window-size=412,915`,
  navigate to `/fa/splash`, `Runtime.evaluate` `localStorage.setItem('ritme_token', T); document.cookie='ritme_auth=1; path=/'`.
  For captures: `Network.enable`, `Network.clearBrowserCache`, `Network.setCacheDisabled(true)`, navigate, wait 8 s,
  then count `Network.requestWillBeSent` to the API origin by method. For the duplicate check, click
  `a[href="/fa/<route>"]` in sequence and diff the GETs between steps.
- Lighthouse (12.8.2): before every run, `Storage.clearDataForOrigin({origin, storageTypes:'service_workers,cache_storage'})`
  and `Network.clearBrowserCache`, then
  `lighthouse http://127.0.0.1:3110/fa/<route> --port=9322 --disable-storage-reset --only-categories=performance --output=json`
  (add `--throttling-method=devtools` for table B). 3 runs per route; report the median of each metric.
- Prod public page: `CHROME_PATH=… lighthouse https://web.ritme.app/fa/welcome --chrome-flags=--headless=new --only-categories=performance`.
- Dead code: `cd frontend && npx --yes knip@5 --no-progress --reporter compact`.

### 6.5 Prod (read-only)

```bash
ssh root@89.251.8.115 'cd /opt/ritme && docker compose exec -T backend ls -la bootstrap/cache'
ssh root@89.251.8.115 'cd /opt/ritme && docker compose exec -T mysql sh -c "mariadb -uroot -p\"\$MYSQL_ROOT_PASSWORD\" -D \"\$MYSQL_DATABASE\" -e \"SELECT table_name, table_rows, ROUND(data_length/1024/1024,2) data_mb, ROUND(index_length/1024/1024,2) idx_mb FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY data_length DESC\""'
# EXPLAIN / ANALYZE SELECT only — e.g.
#   EXPLAIN SELECT * FROM daily_health_logs WHERE user_id = 1 AND date(log_date) = '2026-09-19';
#   ANALYZE FORMAT=JSON SELECT COUNT(*) FROM cycle_calculations WHERE user_id = 28 AND calculation_date < '2026-09-19' AND is_locked = 0;  -- 328 rows, 16.5 ms
ssh root@89.251.8.115 'for i in $(seq 1 16); do curl -s -o /dev/null -w "%{time_total}\n" -H "Host: api.ritme.app" -H "Accept: application/json" http://127.0.0.1:8080/api/v1/languages; done'
```

No health data was selected or printed. Only counts, sizes, index metadata, and `EXPLAIN` plans.

## T-M1-12 — backend before/after (2026-09-19)
Same in-process harness as §6.1: the same seeded user (13 periods, 60 logs), a scratch sqlite copy with Redis, 30 iterations × 3 interleaved rounds, median of p50/p95 in ms. Load average 6–7, so treat these as noisy.

| Endpoint | Before: q / p50 / p95 / bytes | After, miss: q / p50 / p95 | After, hit: q / p50 / p95 / bytes |
|---|---|---|---|
| `/cycle/month/2026/9` | 7 / 48.3 / 51.0 / 119,583 | 7 / 18.5 / 21.9 | 7 / 9.1 / 10.5 / 87,189 |
| `/cycle/month/2026/9?view=calendar` | (full only) | 6 / 14.6 / 18.8 | 6 / 8.5 / 11.8 / 8,390 (664 B gz) |
| `/home` | 37 (3 dup) / 30.2 / 33.0 | 20 (0 dup) / 21.6 / 28.9 | 20 / 23.7 / 28.1 |
| `/home/sections/week_calendar` | 14 / 18.6 / 21.5 | 7 / 10.6 / 13.4 | 7 / 10.5 / 13.8 |
| `/messages/daily` | 14 / 15.6 / 17.8 | 10 / 15.3 / 19.3 | 10 / 14.9 / 20.3 |
| `/home/sections/smart_tip` | 14 / 15.7 / 18.5 | 10 / 15.8 / 19.4 | 10 / 16.0 / 19.1 |
| `/cycle/today` | 8 (1 dup) / 11.4 / 13.8 | 7 / 11.2 / 13.9 | 7 / 8.1 / 11.1 |
| `/cycle/date/2026-09-19` | 8 (1 dup) / 11.6 / 14.6 | 7 / 11.2 / 14.5 | 7 / 8.0 / 11.0 |

- Config/route/event caches (fresh process + `GET /languages`, 30 interleaved runs): p50 58.97 → 49.18 ms (−17%), p95 86.0 → 67.0 ms.
- Engine CPU for a 31-day month (no queries): full 43 → 11 ms, calendar 37 → 5.7 ms.
- Only `/home`, `week_calendar`, `/messages/daily` and `smart_tip` skip the engine cache, so for them miss and hit are just two separate runs.

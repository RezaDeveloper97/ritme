# Progress notes

One `## T-Mx-NN` section per finished task: what shipped, commands/env vars, migrations, open items.

## T-M1-01 — Diagnose premature logout
- Shipped `docs/investigations/session-logout.md` (H1–H7 with evidence). Server-side lifetime ruled out (DB `expires_at` and JWT `exp` = 365 d; keys stable since 2026-08-16; no stray revokes).
- Root causes: (1) `ritme_auth` flag cookie written via `document.cookie` → WebKit 7-day cap, and SessionGuard keeps a user with a valid localStorage token on `/signup`; (2) iOS storage partitioning (Safari tab vs Home Screen app vs in-app browser) — confirmed in prod logs; (3) prod bundle v1.0.2 still clears the token on any 401 (latent).
- Scope edits: T-M1-02 → hardening (401 `error_code`, stop shipping laptop Passport keys via rsync/`COPY`, fix entrypoint creating a 2nd personal client; do NOT rotate keys); T-M1-03 → main fix; T-M1-04 → optional flush + manual tests.
- Open: proxy logs only since 2026-09-07; nginx `log_format` lacks `$host`; real-iPhone check of the 7-day cap pending (T-M1-03).

## T-M1-06 — PWA audit
- Shipped `docs/investigations/pwa-audit.md` (~50 rules, severity + evidence + fix task). Lighthouse 12.8.2 mobile `/fa/signup`: prod Perf 90 / A11y 88 / BP 100 / SEO 100, LCP 3.6s; installability errors `[]`.
- High (T-M1-08): SW `clients.claim()` + `useAppUpdate` reload on any `controllerchange` → every first visit reloads ~2s in; `sw.js` stamped only from package.json version → deploys without a bump never update the SW.
- Med: unbounded runtime cache, no ChunkLoadError recovery, HTML `s-maxage=31536000` behind ArvanCloud (relies on a panel rule), no CSP/HSTS/nosniff (T-M1-08); iOS safe-area on tabbar/toasts, dark status bar, missing `apple-mobile-web-app-capable` (T-M1-09); manifest gaps, `/favicon.ico` 404, upscaled 512 icons (T-M1-07).
- Scope/touches of T-M1-07/08/09 updated. Open: logged-in pages not audited; need ≥1024px logo source; `web.ritme.app` now fronted by ArvanCloud.

## T-M1-04 — Android shell session persistence
- `android-shell/.../MainActivity.kt`: `onStop()` now also flushes cookies. **`android-shell/` is gitignored**, so this lives only in the working tree and ships with the next `build.sh`.
- Verified: no backup/extraction rules restore a stale token, the WebView data dir is stable, and the same release signing key is used across 1.1.0 and 1.1.1 (in-place updates).
- Manual test for a human: install `ritme-1.1.1` → sign in → `am force-stop` → reopen (signed in) → `adb install -r` new release APK → reopen (signed in).

## T-M1-02 — Backend session hardening
- Token lifetime is a fixed `P365D` interval (`config/passport.php`: `token_lifetime_days`=365, `refresh_window_days`=30, env-overridable).
- JSON 401s carry `error_code` ∈ `token_revoked` / `token_expired` / `unauthenticated` (`bootstrap/app.php` report+render callbacks). The admin web redirect is unchanged.
- New `POST /api/v1/auth/refresh-session` (auth:api, throttle 10/min): issues a fresh year and revokes the old token only within 30 days of expiry, otherwise a no-op.
- `entrypoint.sh`: never regenerates existing keys, exits if only one key file is present, warns if keys are missing while tokens exist, and creates the personal client only on a real `count=0`.
- Keys no longer ship: `backend/.dockerignore` `storage/*.key`; `deploy.sh` and `deploy-stage.sh` exclude `backend/storage/*.key` (never add `--delete-excluded`). Prod reads its keys from the `ritme_backend-storage` volume (verified read-only).
- Tests: `tests/Feature/Auth/SessionTokenTest.php` (12). Suite 345 passed; pint green on changed files (14 pre-existing failures elsewhere).
- Follow-ups: rotate the laptop key pair (fingerprint `b501f48dc5d43bdc`), either a forced re-sign-in after T-M1-03 ships and SMS works, or a dual-key validator; delete the stray `/opt/ritme/backend/storage/oauth-*.key`; admin `UserController::destroy` doesn't revoke tokens; consider `dontReport(OAuthServerException)` before 2027-08.

## T-M1-03 — Frontend durable session
- **Main logout fix.** A token in `localStorage` is never dropped because the flag cookie is missing. SessionGuard (`shared/session/reconcile.ts`, pure and tested) re-asserts the flag on every start, route change and resume, and moves signed-in users off `/splash`, `/welcome`, `/signup` and `/otp`. `/signup` and `/otp` stay reachable while onboarding is pending.
- The flag cookie `ritme_auth=1` is now server-set via `POST /api/session/flag` (Next route handler, 1 year, SameSite=Lax, 403 for cross-site requests), which escapes WebKit's 7-day cap on script-set cookies. The JS write remains as a fallback.
- 401 handling (`shared/session/unauthorized.ts`): the session ends only on a JSON 401 for a request that carried the current token, with `error_code` ∈ token_expired/token_revoked/unauthenticated, or the legacy Laravel body. Non-JSON (proxy/Basic-auth) 401s, 5xx and network errors never clear the token. The orchestrator widened this to `unauthenticated` so rotated or deleted tokens don't leave the app stuck.
- Sliding refresh `features/auth` `SessionRefresher` → `POST /auth/refresh-session` when `exp` is less than 30 days away (single-flight, 10-minute backoff, stops on 404/405). `navigator.storage.persist()` runs after sign-in.
- iOS install guide notes the Home Screen app asks for one sign-in; an "open in browser" hint shows in in-app browsers (new `pwa.*` keys; run `php artisan translations:import` on deploy). The dead `ritme_onboarded` branch was removed.
- Verify: typecheck ✔, lint 0 errors (4 pre-existing warnings), steiger ✔, vitest 27 files / 216 tests ✔. Headless CDP manual checks pass (cookie deleted → still signed in; API offline → still signed in).
- Open: needs the stage vhost route for `/api/session/flag` (added in T-M1-08). Human iPhone test pending (Home Screen sign-in once, stays signed in for more than 8 days). The `ritme_onboarding` cookie is still JS-set. Ships to prod only via T-M1-14.

## T-M1-08 — SW lifecycle, caching and headers
- Fixed the double reload on first visit: `useAppUpdate` reloads on `controllerchange` only if the page was already controlled or the user tapped update. A fresh profile now has exactly 1 document load (CDP).
- Every build gets a unique `buildId` (`generate-version.mjs`; `RITME_BUILD_ID` prefix optional). It is stamped into `sw.js`, `version.json` and Next's `generateBuildId` / `NEXT_PUBLIC_BUILD_ID`, so two builds of 1.0.2 produce different `sw.js` and the same version with a new build shows the soft toast. **`public/sw.js` and `public/version.json` are now gitignored** (generated at prebuild, including in Docker).
- SW: per-build precache; one shared runtime cache capped at 150 entries; cache-first for `/_next/static`; navigation preload; 10-second navigation timeout → `offline.html`. `/api`, HTML, `sw.js` and `version.json` are never cached.
- `shared/pwa/chunkReload.ts`: one guarded reload on ChunkLoadError, installed from UpdateGate.
- Headers (`next.config.ts`): documents get `private, no-cache`; all paths get nosniff, Referrer-Policy and **CSP Report-Only** (no report-uri yet).
- nginx: HSTS (1 year, no preload or includeSubDomains) in `proxy-ssl.conf` and the stage vhost; duplicate Cache-Control removed on stage; `location = /api/session/flag` → Next on stage.
- Verify: typecheck/lint (0 errors)/steiger/vitest 216 ✔, `npm run build` ✔, `nginx -t` ✔.
- Deploy notes: `deploy/*` changes reach the shared proxy only via `NO_BUILD=1 ./deploy.sh` (`deploy-stage.sh` doesn't reload nginx). The first deploy shows every user the soft toast once. Check HSTS through ArvanCloud with `curl -I`. Enforce CSP only after checking logged-in pages on staging.

## T-M1-07 — Manifest, icons, installability
- `manifest.ts`: `start_url: '/splash'` (one redirect, and the middleware keeps the user's locale; the orchestrator changed the agent's `/fa/splash`), `display_override` [fullscreen, standalone], `categories`, 2 shortcuts (log, calendar), 3 real screenshots (2 narrow, 1 wide); `id: '/'` unchanged.
- Every icon is regenerated from one 512px master (`application/.../ritme_logo.webp`, the rebrand gradient): any-purpose icons at 88% transparent, maskable at 70% on white (inside the safe zone), one apple icon (identical files). New `src/app/favicon.ico` (16/32/48).
- InstallPrompt: localStorage access guarded; in-app browsers skip the iOS guide.
- Verify: typecheck, lint (0 errors), lint:styles, build ✔. CDP: manifest errors `[]`, installability errors `[]`.
- Open: need a ≥1024px/SVG logo master from design (then rerun the ImageMagick recipe); screenshots use local test data (curate before a store listing); `display: fullscreen` is a product decision; the in-app `logo.webp` still has the old art.

## T-M1-10 — Performance baseline
- Shipped `docs/investigations/perf-baseline.md`: route/bundle table, Lighthouse mobile for home/calendar/log, API p50/p95 and query counts for 30 endpoints (seeded user), prod EXPLAIN, knip + backend dead-code sweep, 10 ranked opportunities.
- Top hotspots: (1) `CalculateCycleDataJob` + `cycle_calculations`: 366 rows and 376 queries per write, 89k rows / 238 MB in prod (~99% of the DB), never read → T-M1-13; (2) `/cycle/month` is 118–143 KB and called twice per cold load; a slim shape would be 8.4 KB → T-M1-12 + T-M1-11; (3) all 6 Vazirmatn weights preloaded (128 KB) → T-M1-11; (4) no config/route/event cache in prod (−21% p50 locally) → T-M1-12; (5) home CLS 0.06–0.20 and 22 round trips (11 GET + 11 preflights) → T-M1-11.
- T-M1-11/12/13 scopes rewritten with concrete items. Open: INP not measured; no authenticated prod p95 (nginx lacks `request_time`); same-origin API to drop preflights is an owner decision; `features/manage-account` isn't wired into the UI.

## T-M1-05 — Session regression coverage and docs
- Local end-to-end run over CDP: OTP sign-in gives `exp` +365 d and a server-set `ritme_auth` flag (Max-Age 31536000). A token with 10 days left is refreshed on load (new token stored, old one revoked, a single request). A revoked token → `401 token_revoked` → one clean sign-out to `/fa/signup` even with 11 parallel 401s.
- Docs: a "Fixed in" table in `docs/investigations/session-logout.md`; `frontend/CLAUDE.md` §11.1 session rules; `backend/CLAUDE.MD` session and token section.
- Verify: backend 345 passed, frontend 216 passed. The staging deploy and sign-in are deferred to T-M1-14.
- Dev note: `next dev -H 127.0.0.1` redirects `/` to `localhost` (a different origin, so a different localStorage).

## T-M1-09 — iOS standalone polish
- layout: `apple-mobile-web-app-capable=yes`; 22 `apple-touch-startup-image` links (11 iPhone portrait sizes, light and dark; PNGs in `public/splash/`, ~1.6 MB, not precached).
- Fixed a duplicate `theme-color` in dark mode (React 19 hydration added a stale second meta): Next no longer renders `themeColor`; `chromeInitScript` creates one meta from the live `--page`. `frontend/CLAUDE.md` §10.3 updated.
- Status bar: dark theme only gets `black-translucent`; light stays `default`, because white status-bar text would be invisible on the light shell. This is a deliberate deviation from the task's I-3.
- globals.css safe-area: `.view` bottom padding, tabbar margin, toasts/FAB bottom, delete sheet, splash footer. Verified with CDP `setSafeAreaInsetsOverride` {47, 34} light and dark; without insets nothing changes.
- Verify: typecheck, lint (0 errors), lint:styles (baseline locked), lint:dark, build ✔.
- Open (needs a real iPhone): whether iOS reads the runtime status-bar meta, the startup images, tabbar spacing. The startup image follows the OS scheme, not the in-app theme. No iPad or landscape splash.

## T-M1-12 — Backend performance
- `/cycle/month?view=calendar` (opt-in slim view, 11 fields per day, 8.4 KB instead of 119 KB; unknown view → 422; default payload unchanged but now `JSON_UNESCAPED_UNICODE`).
- `CycleEngineCache`: a per-user cache of engine results for month/today/date. The key hashes the engine's actual inputs (profile, histories, logs, admin recommendations) plus version, locale and today, so it never goes stale even though `calculation_version` isn't bumped on write.
- N+1 and duplicate reads removed in `/home` (37 → 20 queries), `week_calendar` (14 → 7) and `MessageContentRepository` (one query per locale). Fixed a SQLite-only bug where preloading daily logs dropped the first day of the range.
- Migration `2026_09_19_000001_drop_redundant_cycle_histories_user_start_index` (plain index duplicated the unique one; tested up and down on SQLite and MariaDB 11.4).
- `entrypoint.sh` runs `package:discover` + `config:cache` + `route:cache` + `event:cache` after migrations; on failure it boots uncached. `.dockerignore` now excludes `bootstrap/cache/*.php` (orchestrator).
- Query budgets are locked in `tests/Feature/Performance/*` (14 tests). Suite: 359 passed (SQLite and MariaDB 11.4). pint clean on changed files. Before/after table appended to perf-baseline.md.
- Open: `env()` outside config/ (`SwaggerBasicAuth`, `bootstrap/app.php` ADMIN_PANEL_ENABLED, `AdminSeeder`) works with compose env but would break with `.env`-only + config cache; move them to `config()`. `message_contents` has no cross-request cache (needs a save hook). For T-M1-13: bump `calculation_version` synchronously on write and drop the "processing → return" guard.

## T-M1-13 — Backend dead code and dead storage
- Removed about 7,500 lines: `CalculateCycleDataJob`, the `CycleCalculation` model and `User::cycleCalculations()`, `MatrixEngine/*`, `Test*Controller` + `test-*` views + the local-only web routes, `ContextProviderInterface`, and `getEnums()` on the message engines. Routes removed after a caller search in the frontend, the Android app and android-shell found nothing: `/cycle/matrix-messages`, `/cycle/matrix-enums`, `/cycle/enums`, `/messages/enums`.
- Recalculation is now synchronous: `UserProfile::markRecalculated()` bumps `calculation_version` atomically and sets status `completed`. The "processing → return" guard is gone. The `/cycle/status`, `/cycle/recalculate` and `is_recalculating` contracts are kept (always false now), so frontend/Android polling stops after at most one poll.
- Migration `2026_09_19_000002_drop_cycle_calculations_table`: moves stuck `processing` profiles to `completed`, then drops the table (~89k rows / 238 MB in prod). `down()` recreates an empty table.
- Verify: 370 tests passed on SQLite and MariaDB 11.4; pint clean on changed files.
- **Prod deploy:** take a full DB backup before migrating; clear leftover `CalculateCycleDataJob` jobs from the queue (`queue:clear`), otherwise they land in `failed_jobs` (harmless).
- Open: the `recalculate` fa/en messages still use a `$locale === 'fa'` branch; `PROCESSING`/`FAILED` statuses are now unreachable (the client polling could be simplified later).

## T-M1-11 — Frontend performance
- Fonts: 6 Vazirmatn static weights replaced by one variable woff2 (45 KB, same 366-glyph subset), 1 preload instead of 6.
- Home CLS 0.063 → 0.009: the hero holds its loading state until `/cycle/today` + `/banners` settle (`useBannersSettled`, same cache key).
- Slim month: `useCycleMonth` uses `?view=calendar` (API bytes on a cold home load 208 KB → 35 KB, calendar 195 KB → 22 KB). `/cycle/status` is only polled while a recalculation runs, and below-the-fold reads are deferred.
- i18n per route: `app/message-scopes.ts` + `RouteMessages` + `pickNamespaces`. A guard test walks the import graph so the lists can't drift. Messages per page 50 KB → 21–25 KB.
- axios replaced by a thin `fetch` client (`shared/api/apiClient.ts`, 11 tests; session and 401 semantics, 15 s timeout, `Accept: application/json` kept). Removed axios, react-hook-form, @hookform/resolvers, `shared/lib/cookie-state`, and the duplicate SectionHead. `widgets/day-tasks` was kept (hidden per a product request).
- First Load JS home/calendar/log 210/207/193 → 189/186/171 kB; LCP improved on every route (before/after table in perf-baseline.md). The frontend CLAUDE.md stack table was updated.
- Verify: full gate ✔ (eslint 0 errors, steiger, styles, dark, 29 files / 253 tests, build).
- Open: `/fa/cycle` React #418 hydration mismatch (pre-existing); same-origin API proxy to drop preflights; zod/mini not tried; home CLS with real banners not measured.

## M1 summary
Milestone M1 is on `stage` (14 commits, `393373c..ab5ffec`). Staging is deployed and smoke-tested. **Production is still on v1.0.2; nothing in M1 has reached prod.**

### What shipped, by area
- **Session / logout (T-M1-01→05):** the root cause is diagnosed (the WebKit 7-day cap on the script-set `ritme_auth` cookie, iOS storage partitioning, and a latent "clear token on any 401"). Tokens are a fixed 365 days and every JSON 401 carries `error_code`. `POST /auth/refresh-session` gives sliding renewal in the last 30 days. Passport keys no longer ship in rsync/images, and the entrypoint never regenerates keys. On the frontend, the token in localStorage is the source of truth: SessionGuard reconciles the flag on start and route change, the flag cookie is set by the server (`/api/session/flag`, 1 year), 401 handling is safe (only a JSON 401 for the current token ends the session), and `storage.persist()` is requested. Android shell: cookies are flushed on stop.
- **PWA (T-M1-06→09):** the audit is written. The double reload on first visit is fixed, every build has its own buildId stamped into `sw.js`/`version.json`, runtime caches are bounded, ChunkLoadError recovers, and `/api`, HTML and `sw.js` are never cached. Headers now include `private, no-cache` HTML, nosniff, Referrer-Policy and CSP Report-Only. The manifest gained shortcuts, screenshots and `display_override`, the icons were regenerated and a favicon added. iOS standalone polish: safe-area, 22 startup images and a single theme-color.
- **Performance (T-M1-10→13):** the baseline is measured. `/cycle/month?view=calendar` returns about 8 KB instead of ~119 KB. There is an engine-result cache, `/home` dropped from 37 to 20 queries, and config/route/event caches run in the entrypoint. Dead code is gone (about 7.5k lines, including `CalculateCycleDataJob` and the `cycle_calculations` table, ~238 MB in prod), and recalculation is synchronous. Frontend: a variable font (1 preload), i18n per route, a fetch client instead of axios, the slim month view, and home CLS down from 0.063 to 0.009. First Load JS is −21 kB on home.
- **Release (T-M1-14):** `frontend/package.json` goes 1.0.2 → **1.1.0**, and the release note shown in the toast is updated. **`minSupportedVersion` stays 1.0.1**, because a forced update is not needed. The backend is additive for old clients: new `error_code` fields, an opt-in `?view=`, and the removed routes had no callers. The old v1.0.2 service worker serves navigations network-first, so every installed client loads the 1.1.0 HTML and chunks on its next launch, and the byte-different `sw.js` also gives open tabs the soft toast. A forced screen would only interrupt long-lived tabs. It would not fix anything sooner, because the 401/flag fix is only active once the new bundle runs, which is the next launch anyway.

### Verify (2026-09-19, commit `ab5ffec`)
| Gate | Result |
| --- | --- |
| backend `php artisan test` | ✔ 370 passed (2039 assertions) |
| backend `pint --test` | ✘ 73 files, **all pre-existing**. None of them are among the 52 backend files M1 touched, so there was no mass reformat |
| frontend typecheck / fsd:lint / lint:styles / lint:dark | ✔ |
| frontend lint | ✔ 0 errors (4 pre-existing warnings) |
| frontend vitest | ✔ 29 files / 253 tests |
| frontend `npm run build` | ✔ |
Nothing needed fixing.

### Staging deploy and smoke test (`deploy-stage`, branch `stage` @ `ab5ffec`)
- DB backup before migrating: `/root/ritme-stage-backups/ritme_stage-pre-M1-20260919-135759.sql.gz` (40 tables, mode 600). It is kept outside `/opt/ritme-stage` because that rsync uses `--delete`.
- Migrations ran (`drop_redundant_cycle_histories_user_start_index`, `drop_cycle_calculations_table`), and `cycle_calculations` is gone. The entrypoint built the config/route/event caches: `bootstrap/cache` holds config.php, routes-v7.php, events.php, packages.php and services.php. Every deploy-script assertion passed, including the gate cookie.
- Headers: `sw.js` and `version.json` send `no-store`, HTML sends `private, no-cache`, and every path sends CSP-Report-Only, nosniff and Referrer-Policy. **HSTS is not present yet**, and `sw.js`/`version.json` send a duplicate `Cache-Control`. Both come from the shared proxy config, which has not been reloaded (see pending).
- API: a minted token has `exp` = 365.0 days. With no token or a bad token the API returns `401 {"error_code":"unauthenticated"}`, and with a revoked token `401 {"error_code":"token_revoked"}`. `refresh-session` on a fresh token returns `{refreshed:false, expires_at:2027-09-19}`. `/cycle/month/2026/9` is 56 KB by default, 8.4 KB with `?view=calendar`, and 422 for `?view=bogus`.
- Browser (headless Chrome over CDP, token minted via tinker for staging user 1, which has since been revoked):
  - ✔ The signup page renders.
  - ✔ The signed-in user is redirected from /signup to /fa/home.
  - ✔ home, log and calendar render and the token is kept.
  - ✔ The calendar calls `?view=calendar` (~8.7 KB per month).
  - ✔ The SW is active and controlling.
  - ✔ Navigating offline shows the Persian offline page.
  - ✔ A real update, rebuilt with `--no-cache` for a new buildId, shows the soft toast on a profile controlled by the previous build: `waiting` is true, and "یه نسخه جدید اومده… به‌روزرسانی" appears with the new release note. Tapping update activates the new worker and reloads, the toast is gone, and the user is still signed in.
  - ✔ No forced screen appears.
  - ✘ `POST /api/session/flag` returns 404 from Laravel on staging, because the shared proxy's `vhost-stage.inc` lacks the T-M1-08 `location = /api/session/flag` route. The JS-set cookie fallback worked (`ritme_auth` was present). On prod the route is unaffected, because web.ritme.app goes straight to Next.
- Note: `deploy-stage.sh` with an unchanged source reuses the Docker layer cache, so the buildId stays the same and no update is shown. That is correct, since the bundle is identical.
- Real sign-in by SMS OTP was not exercised (no test mode, and I did not want to send an SMS). The token was minted instead.
- `backend/resources/translations` (the seed read by `translations:import`) lags the frontend by the T-M1-03 `pwa.*` keys plus older `profile` drift. It is harmless at runtime, because the bundled messages are the base and the server only overrides them, but running `php artisan translations:import` and committing the result would tidy it up.

### Pending for the user
1. **Production deploy** of `stage`/M1 (`./deploy.sh`). Before it: a **full prod DB backup** (the `cycle_calculations` drop removes ~89k rows / 238 MB and cannot be undone). After it: `php artisan queue:clear` for leftover `CalculateCycleDataJob` jobs.
2. **Shared proxy nginx:** `NO_BUILD=1 ./deploy.sh` (or rsync `deploy/` plus a reload) to pick up HSTS, the stage `/api/session/flag` route, and the removal of the duplicate Cache-Control on stage. Then check HSTS through ArvanCloud with `curl -I`.
3. **Passport key rotation decision** (the laptop key pair with fingerprint `b501f48dc5d43bdc`: a forced re-sign-in once SMS works, or a dual-key validator). Also delete the stray `/opt/ritme/backend/storage/oauth-*.key`.
4. **Rotate the staging Basic-auth password**, which was exposed in an earlier tool transcript (see the `deploy-stage` skill). Optionally delete `/opt/ritme/stage-gate.conf` and redeploy to rotate the gate cookie too.
5. **iPhone manual tests:** the Home Screen app signs in once and stays signed in for more than 8 days; the runtime status-bar meta; startup images; tabbar safe-area spacing; the in-app-browser hint.
6. **Android manual tests** (android-shell 1.1.1): sign in → force-stop → reopen → `adb install -r` → still signed in.
7. **A ≥1024px or SVG logo master** from design, then rerun the icon recipe (the in-app `logo.webp` still has the old art).
8. **`display: fullscreen` product decision** (`display_override` currently lists fullscreen first).
9. **`/fa/cycle` React #418 hydration mismatch** (pre-existing).
10. **`env()` outside config/** (`SwaggerBasicAuth`, `bootstrap/app.php` ADMIN_PANEL_ENABLED, `AdminSeeder`): move these to `config()` before relying on a `.env`-only setup with the config cache.
11. **Same-origin API proxy** for prod, to drop the ~11 CORS preflights per cold load (an owner decision).
12. Enforce CSP only after watching the Report-Only violations on staging's logged-in pages. Fix the SMS gateway (SMS.ir template / Kavenegar test mode) so real OTP sign-in can be tested.

## T-M2-02 — backend-go skeleton, Fiber v3 app and dev tooling
- New `backend-go/` (module `github.com/ritme/backend-go`, Go 1.25, Fiber v3.5.0, go-redis v9, go-sql-driver/mysql).
- `cmd/api`: slog JSON logs, graceful shutdown, `api healthcheck` subcommand, trust-all-proxies XFF client IP,
  25 MB body limit, CORS port of fruitcake/php-cors (headers verified against the PHP implementation).
- `internal/platform/{config,db,cache}`, `internal/http` domain registry (`routes_<domain>.go` via `init()`), `/up`.
- Dockerfile: alpine 3.22 (gcr.io distroless is 403 from here), `USER 33:33`, GOPROXY fallbacks.
- Makefile targets: run, test, test-int PKG=, test-db-up/down, lint, sqlc, schema-diff, contract(-record) ROUTES=.
- Test stack `docker-compose.test.yml`: MariaDB 11.4 on 13317, Redis on 16380 (`TEST_DB_PORT`/`TEST_REDIS_PORT`).
- New env: `HTTP_ADDR`, `STORAGE_PATH`, `REDIS_PREFIX=ritme-go:`; config refuses `APP_DEBUG=true` in production.
- verify-all skill gained a Go row; format hook runs gofmt on backend-go.
- Open: `/up` returns plain `OK` (Laravel returns HTML); registry test lives in `cmd/api/app_test.go`.
  Deps: `go get mod@version` only, no concurrent `go mod tidy` (see backend-go/CLAUDE.md).

## T-M2-01 — Laravel test clock and deterministic contract fixtures
- `TestClock` global middleware: `X-Test-Now` header → `Carbon::setTestNow()` per request, only when
  `APP_ENV∈{local,testing,contract}` **and** `TEST_CLOCK_ENABLED=true`; unparseable header → 400.
- `LogSmsProvider` (`SMS_PROVIDER=log`, refuses production), wired in `SmsService` (touches extended).
- `ContractFixtureSeeder`: content seeders + `ar` language row + 3 banners (1 expired), 18 personas (ids 1001+,
  mobiles 09900000001..18, dependent ids `userId*100+n`), dates from `CONTRACT_TODAY=2026-09-23`; idempotent.
- `docker-compose.contract.yml`: contract-mariadb/redis/laravel on `127.0.0.1:${CONTRACT_PORT:-8090}`
  (8090 is taken on this dev machine → use `CONTRACT_PORT=18090`), `contract-reset` (profile tools) rebuilds DB.
- Persona table and usage: `docs/go-migration/contract.md`. New env: `TEST_CLOCK_ENABLED`.

## T-M2-04 — Platform core — civil dates, clock, PHP-compatible JSON, envelopes and errors
- `internal/platform/civildate` (Date/NullDate, Tehran today, Saturday week start, `ParseLenient`),
  `clock` (Real/Fixed, `X-Test-Now` middleware, same gate as Laravel TestClock), `phpround` (exact PHP 8.4
  `round()` + `(string)$float`, 4000-value PHP corpus), `jsonx` (PHP `json_encode` flags, Laravel date/decimal
  casts, OrderedMap), `httpx` (envelopes, Laravel error bodies byte-identical incl. 404/405/422/429/500/503,
  paginators).
- Orchestrator wired `httpx.ErrorHandler` + `clock.Middleware` into `cmd/api/app.go`; access log uses `httpx.StatusOf`.
- Deviations D-08 (JSON 404 without Accept), D-09 (no relative date strings), D-10 (`/up` plain OK) proposed.
- "(and N more errors)" and paginator labels stay English in every locale (as Laravel does).

## T-M2-03 — Schema baseline with goose and sqlc setup
- `backend-go/db/migrations/00001_baseline.sql` (38 tables + fa/en language rows; Down refuses), embedded via `db/embed.go`.
- `sqlc.yaml`: 11 domain packages → `internal/<domain>/store` (placeholder `sample.sql` per domain — delete when
  adding real queries). NULL JSON → `db.NullRawJSON`; `TIME` → `string`; dates → `civildate.Date/NullDate`.
- `internal/platform/db/migrate.go` (Migrate, StampBaseline), `testdb` (fresh `gt_*` DB per test package; no
  `t.Parallel()` inside a package), `scripts/schema-diff.sh` (Laravel migrations vs goose → identical), `docs/go-migration/migrations.md`.
- goose v3.26.0 (v3.28 needs go 1.26). `make test-int` passes `TEST_DB_ADMIN_DSN`.
- Open: baseline not diffed against the staging DB (server read needs user approval):
  `scripts/schema-diff.sh --against <staging --no-data dump>`.

## T-M2-07 — Enums — PHP→Go generator plus hand-ported enum logic
- `cmd/enumgen` parses `backend/app/Enums` + MessageSystem enums → 58 `internal/enums/zz_generated_*.go`
  (values in PHP order, From/IsValid/Cases, Label/Description/Icon, Options). `go generate ./internal/enums/...`.
- Hand-ported logic: `cycle_logic.go`, `health_logic.go`, `recommendation_logic.go` (PHP file:line cited).
- Parity test vs `testdata/php_enums.json` (refresh: `php internal/enums/testdata/dump_enums.php ../backend > internal/enums/testdata/php_enums.json`).
- Conventions: PHP null → `""`; `labelFor` → `(string,bool)`; locale-less labels keep `Label(_ string)`.

## T-M2-05 — Contract harness — golden recorder and JSON-aware differ
- `backend-go/cmd/contract` (record/diff, strict JSON-aware differ in `internal/ojson`), cases for 14 groups in
  `contract/cases/*.yaml`, 986 goldens in `contract/golden/<group>/`, `fixtures/dump.sql`, contract-only key pair.
- `make contract-record ROUTES=<groups|all>` (flock-serialised, reuses/ups the Laravel contract stack, `CONTRACT_PORT`),
  `make contract ROUTES=<groups>` boots Go on a free port with its own `contract_<pid>_*` DB → parallel-safe.
- Harness mints Passport-compatible tokens itself (independent of Go auth); synthetic token personas added.
- Allow-list: `contract/allowlist/<group>.yaml`, each entry must reference a `D-nn` in deviations.md.
- Locked quirks: premium `/messages/daily` 500 (D-11), `retry_after` 80, `token_unknown` → token_revoked.
- Throttle-middleware 429 not recordable (`CACHE_STORE=array`); controller 429 is.

## T-M2-13 — Cycle engine v1.1 library (metrics, resolver, view builder, daily card)
- `internal/cycle/{model,metrics,resolver,view}`: ports of CycleMetricsCalculator, CycleStatusResolver (+ §35
  `ToAPI()` key order), PhaseMapper, CycleDayViewBuilder, prediction/open-period services, daily card.
- 61 PHP unit tests ported 1:1 (mapping tables in package docs); golden sweep test: 17 personas × fa/en × 60 days
  = 2040 `cycle_view`s equal to Laravel (`contract/golden/cycle-sweep`).
- `view.BaseCalc` interface is what the legacy engine result (T-M2-14) must implement.
- Daily card keeps Laravel's hardcoded fa/en (`locale == "fa"`) — same behaviour, not a deviation.
- Golden test duplicates the persona→user-id map from `cmd/contract/personas.go`.

## T-M2-06 — Laravel-compatible validation engine and i18n
- `internal/platform/validation` (+ `phpval` for PHP value semantics): rules/messages byte-identical to Laravel on
  46 HTTP + 40 `Validator::make` goldens (fa/en/ar/no header). Pass `validation.Now(clock)`; `Errors()` after `Fails()`.
- `internal/i18n`: language registry (Redis `ritme-go:languages.registry`), locale middleware/Resolve/Clamp,
  Translatable, TranslationStore (bundles equal `/languages/{code}/messages` goldens), `lang` (`trans()`).
- `resources/lang` (PHP lang → JSON via `convert.php`), `resources/translations` (fa/en seed), both embedded.
- Differences: bootstrap fallback not cached when languages table unreadable; `email` rule approximated via net/mail.
- Open: registry has no TTL → cross-stack staleness (note added to T-M2-10).

## T-M2-14 — Legacy HealthDataEngine library
- `internal/cycle/legacy` (engine, probability, text flags, hardcoded tips, `Calculation` full/calendar/localized
  JSON, month summary, implements `view.BaseCalc`) and `internal/cycle/recommendation` (Source interface,
  per-request memo repository, tip localizer). No DB/Fiber imports.
- Golden sweep: 2040 days + 204 months equal to Laravel; PHP-generated edge cases in `legacy/testdata/php_cases.json`.
- Cache signature uses sha256 (Go cache namespace is separate). Open: sqlc adapter + DailyLog builder → T-M2-15 (noted in its file).

## T-M2-08 — Auth — Passport-compatible tokens, OTP, SMS, rate limiting
- `internal/auth` (+ `passport`, `sms`), `internal/platform/ratelimit`, `internal/platform/queue` (asynq),
  `routes_auth.go`, queries in `db/queries/auth`. Deps: golang-jwt/v5, asynq, miniredis (tests).
- Tokens interoperate both ways with Laravel (cross-stack test runs with `CONTRACT_PORT=18090`); contract auth 45/45.
- API for domains (see `internal/auth/module.go`): `auth.MustGuard(...)` + per-route `guard.RequireUser`,
  `auth.CurrentUser/CurrentUserID/CurrentToken`, `auth.UserJSON`, `auth.RevokeUserTokens`, `auth.UnauthenticatedError`,
  `ratelimit.New(cache, clock).Middleware(n, window, auth.ThrottleIdentity)`, `queue.New(...)`.
- Bad Passport keys / unknown `SMS_PROVIDER` abort start-up. JWT time checks use the real clock (like Laravel).
- `mobile_verified_at` stays null (Laravel's update is a no-op: not fillable). send/verify OTP share one per-IP counter.
- Open: XFF-based rate-limit bypass (same as Laravel) → nginx fix noted in T-M2-09.

## T-M2-11 — Profile and account endpoints
- `internal/profile` (handlers, service, BMI, `model` = generic Eloquent `toArray()` serializer with cast tables for
  CycleHistory, DailyHealthLog, Pregnancy models, Reminder), `internal/notify` (Telegram, async), `routes_profile.go`.
- Contract profile: 101/104 — the 3 failing cases call cycle routes (green after T-M2-15; noted in its file).
- Concurrent `POST /profile` bumps `calculation_version` exactly once per call (`profile.MarkRecalculated`).
- Telegram is sent async (Laravel blocks up to 5s) — bodies unchanged. BMI message falls back to the default language.

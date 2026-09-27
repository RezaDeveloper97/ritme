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

## T-M2-09 — Strangler infrastructure on staging (local part; staging deploy blocked)
- `backend-go` compose service (uid 33, storage ro, `127.0.0.1:8081`); stage `ritme-stage-backend-go-1` (alias
  `stage-backend-go`); prod `ritme-backend-go-1` under compose profile `go` (inert until `COMPOSE_PROFILES=go`).
- `deploy/go-routes.inc`: one regex location per group → `upstream ${ritme_env}_route_<group>`; upstream lines in
  proxy-ssl/stage-ssl/proxy/stage-http confs (all Laravel). Stage overwrites XFF with `$remote_addr`; prod unchanged
  (CDN in front → needs `real_ip` first, see cutover.md).
- `deploy/switch-go-route.sh <stage|prod> <group> <on|off> [--dry-run|--local|--status]` (backup, in-place rewrite,
  `nginx -t`, reload, auto-rollback). Go responses carry `X-Backend: go`. Runbook: `docs/go-migration/cutover.md`.
- Blocked: staging deploy needs one prod proxy recreate (user OK). Verify needs `ADMIN_SEED_PASSWORD` in `.env`.

## T-M2-12 — Reminders and daily health log endpoints
- `internal/reminder`, `internal/healthlog` (+ `model`: `DailyHealthLog::toArray()` equivalent, typed accessors,
  implements `enums.TriggerLog`), `routes_reminder.go`, `routes_healthlog.go`, queries in `db/queries/{reminder,healthlog}`.
- `CycleHistoryService` is live (only caller: `DailyHealthLogController::store`) → fully ported in `healthlog/cyclehistory.go`.
- Side-effects integration test replays 10 POSTs and compares DB rows with Laravel (`testdata/sideeffects`).
- Contract: 83/86 — 3 cases fail only on period-route steps (green after T-M2-15; noted there). D-01/D-02 kept as Laravel (500s).
- Open: `markRecalculated`/LMP update queries duplicated with the profile store.

## T-M2-16 — Pregnancy engine and all /pregnancy endpoints
- `internal/pregnancy` (+ `calc`, `alerts`, `content`), `routes_pregnancy.go`, queries in `db/queries/pregnancy`.
  All 26 routes; contract pregnancy 194/194.
- API: `pregnancy.LoadProfile`, `calc.New(profile, locale, today)` → GestationalAge/EDD/CurrentWeek/Status().JSON(),
  `pregnancy.ProfileJSON`/`AlertJSON`. MessageManager's `pregnancyWeek` = `GestationalAge().Weeks` (0-based), not `CurrentWeek()`.
- Eloquent write semantics preserved (created model returns only set attrs, `updated_at` only on dirty).
- Open: non-numeric `{week}`/`{id}` → 404 here (D-02 behaviour) while reminders keep Laravel's 500 — align once D-02 is decided.

## T-M2-20 — Admin API I — admin auth, roles, dashboard, users, admins
- `internal/admin/{httpadmin,auth,dashboard,users,admins}`, `routes_admin_core.go`, queries `db/queries/admin/core.sql`.
  Contract: `docs/go-migration/admin-api.md` (`/api/admin/v1`, `{success,message?,data}`, lists `{items,meta,filters}`).
- Redis sessions in `__Host-ritme_admin_session` cookie, `X-CSRF-Token` on writes (419), sliding 120 min / 30 days
  remember-me, super vs editor (403), login throttle 5/min IP+email + 20/min per email. Laravel `$2y$` hashes verify.
- D-03 implemented (delete revokes tokens) — still needs user OK; D-12 proposed (self-modify 422, English messages).
- Register admin routes with `httpadmin.Handle(r, method, path, kit.Admin(h)|kit.Super(h))`, not `r.Get`.
- New env: `ADMIN_HOSTS`, `ADMIN_WEB_ORIGINS`, `ADMIN_COOKIE_SECURE` (nginx/compose follow-ups noted in T-M2-25).
  Global CORS now skips `/api/admin/`. sqlc: avoid `NOT IN sqlc.arg` and `BETWEEN` (broken codegen for MySQL).

## T-M2-10 — Content endpoints — languages, info, banners, articles, phase content, /storage
- `internal/content` (+ `sanitizer`: port of libxml2 HTML parser/serializer — bluemonday can't match PHP; equal on
  3017 inputs recorded from PHP 8.4.25/libxml 2.9.14), `routes_content.go` (also owns `GET /cycle/phase-content/:phase`).
- Contract public+content 64/64 (none/fa/en/ar). `/storage` hardened against traversal/encoded/dot-file/symlink escapes.
- Reuse: `sanitizer.Clean`/`PlainText` (T-M2-21), `content.PublicURL`, `content.RegistryTTL`.
- Language registry: content routes set a 5-min TTL on `ritme-go:languages.registry` (proper TTL belongs in i18n.Registry).
- D-13 proposed (/storage error shapes).

## T-M2-17 — MessageSystem — daily messages and mode endpoints
- `internal/messages` (`content` with PHP-exported defaults + byte-equality test, `manager` = context, cycle and
  pregnancy engines, layers, modules), `routes_messages.go`, queries `db/queries/messages`. Contract messages 84/84.
- D-05 preserved and pinned by tests; D-11 reproduced (`manager.ErrUndefinedOverrideCase` → logged 500, also covers
  missing BLOATING/ACNE cases).
- Open: `StoreSource` duplicates cycle adapters (switch to `cycle/service` later); `content.Repository` can replace
  `profile.StoreMessageContents` (one-line change).

## T-M2-15 — Cycle and period-log endpoints with engine cache
- `internal/cycle/{service,cache,periods}`, `routes_cycle.go`, queries `db/queries/cycle/{engine,periods}.sql`.
- Contract: cycle+period+cycle-sweep 313/313; profile+reminders+healthlog now 190/190; also green with
  `CYCLE_ENGINE_CACHE=off` (env read in routes_cycle.go). Period write sequence (21 steps, 8 personas) leaves rows
  identical to Laravel.
- Latency (hey, 400 req, c=8, persona regular; Laravel in docker w/ array cache, Go native):

  | Endpoint | Laravel p50/p95 | Go cache on | Go cache off |
  |---|---|---|---|
  | `/cycle/month/2026/9?view=calendar` | 18.2 / 26.4 ms | 12.0 / 16.1 ms | 12.5 / 17.9 ms |
  | full month | 23.0 / 32.3 ms | 16.2 / 21.0 ms | 18.8 / 26.3 ms |
- API: `service.New(db, cache).Load(ctx, uid, from, to, today)` → Snapshot (`Engine()`, `Day`, `Month`, histories,
  logs, profile), `DayJSON`/`MonthJSON`, adapters `HistoryFromRow`/`ProfileFromRow`/`DailyLogFromRow`/`RecommendationSource`,
  `periods.NewService(db)`.
- Open: `/cycle/status` has a ~7 ms floor (JWT/DB/registry) — profile later; on/off contract test builds the whole API (~60s).

## T-M2-22 — admin-web — Next.js admin app scaffold
- New `admin-web/` (Next 15.5 standalone, TS strict, Tailwind 4, Vazirmatn, brand tokens light/dark, FSD with steiger):
  API client (envelopes, CSRF refresh on 419, 401 → /login), shared UI kit (`/ui-kit` in dev), Jalali dates,
  `TranslatableField` (languages from API), TipTap editor, auth feature, role-aware shell; 34 vitest tests.
- Compose: `admin-web` service (internal, `expose: 3000`); stage alias `stage-admin-web`; prod `ritme-admin-web-1`
  under profile `go` (orchestrator fix: base `depends_on: backend-go` broke the prod overlay otherwise).
- Verified login → dashboard → logout in headless Chrome against local backend-go. Conventions: `docs/go-migration/admin-web.md`.
- Open: Docker image never built (Docker Desktop crashed, disk 98% full); nginx wiring noted in T-M2-25.

## T-M2-18 — Home page sections, task/challenge toggles and notifications
- `internal/home` (HomeContext on one `cycleservice.Load` snapshot — messages use the same snapshot via
  `manager.Source`; 17 sections in a registry with error/panic isolation; DailyChallengeService; notifications),
  `routes_home.go`, queries `db/queries/home/home.sql`. Contract home+notifications 94/94.
- Integration test builds the page for 17 personas × fa/en/ar × 3 dates with no failing section.
- Deterministic `id` tie-breaks added where Laravel relied on MariaDB order (tasks, reminders, articles, notifications).

## T-M2-21 — Admin API II — content CRUD, uploads, messages, languages and translations
- `internal/admin/{content (+form, admintest),media,messages,languages}`, `routes_admin_content.go`,
  `db/queries/admin/content.sql`; endpoints in `docs/go-migration/admin-api.md` §11 (languages super-only).
- Uploads: magic-byte sniffing (GIF / renamed SVG rejected), size + banner min 800×400, WebP re-encode (pure-Go
  `gen2brain/webp`, CGO off) into `app/public/<dir>`, served by `/storage`.
- Language writes flush Go's registry and Laravel's `ritme-database-ritme-cache-languages.registry` (Redis DB 1).
- New deviations D-14 (sanitize on write), D-15 (new default language copies old default), D-16 (banner link_url, kept).
- Follow-ups: storage volume must be rw for backend-go (noted in T-M2-25); lang loader for storage bundles → T-M2-21b.

## T-M2-21b — i18n lang loader reads admin-created languages from storage (D-04)
- `internal/i18n/lang`: `WithStorage(STORAGE_PATH)` overlays `app/lang/<code>/<group>.json` per group (storage wins),
  lazy per locale, reload on name/size/mtime change (checked ≤ 1/s), malformed files skipped + logged, locale regex-guarded.
  `lang.Default()` wires it from `STORAGE_PATH`, so no caller changed.

## T-M2-19 — OpenAPI spec for the Go API and new-endpoint skill for Go
- `backend-go/api/openapi.yaml` (OpenAPI 3.1, 76 operations: 74 `/api/v1` + `/up` + `/storage`), embedded via
  `api/embed.go`; `/docs` Swagger UI bundled in the binary, Basic auth `SWAGGER_USER`/`SWAGGER_PASSWORD` (fail closed).
- `openapi_test.go`: routes ↔ spec both ways (admin API excluded for now), all 3796 golden bodies validate against schemas.
- `new-endpoint` skill rewritten for Go. Deps: santhosh-tekuri/jsonschema/v6, swaggest/swgui.

## T-M2-24 — Parity gate (report: `docs/go-migration/parity-report.md`)
- `make contract ROUTES=all`: 986/986, allow-list empty, all 74 Laravel routes covered. Token interop both ways.
- Web read-flow smoke identical on both stacks (8/11 screenshots byte-identical). Go 1.2×–2.4× RPS on 7/8 hot
  endpoints, ~¼ memory; `/home` at parity (21 sequential queries).
- Verdict: conditional GO for T-M2-25 (public groups); blockers: Android smoke (F-1), web write-flow smoke (F-2),
  deviation approvals (D-01…04, D-08…16), T-M2-09 staging deploy, goose-vs-staging schema diff (F-6).
- Orchestrator follow-ups done: backend-go storage mount now read-write; cross-stack test creates `contract/.work/`.
- `pint --test` red on backend/ (68 pre-existing issues, as noted in M1).

## T-M2-23 — admin-web — all admin screens (parity with the Blade panel)
- Parity checklist vs `backend/routes/admin.php` (every Blade route except the asset routes has a screen):
  ✔ users (list/detail/edit/block/unblock/delete) ✔ articles ✔ affirmations ✔ challenges + completions report
  ✔ recommendations ✔ banners ✔ task templates ✔ info sections ✔ pregnancy weeks (40-week map, 10 sections)
  ✔ phase contents (9 sections) ✔ smart messages (filters, payload editor, approve, toggle) ✔ languages (super)
  ✔ translations editor ✔ admins (super) ✔ own password ✔ login/logout/dashboard (T-M2-22).
- 18 nav items / 40 route pages; test asserts every nav item has a page. 55 vitest tests; build green.
- Checked headless in RTL light/dark at desktop + 390 px and as editor; write flows exercised against local backend-go.
- Beyond Blade: banner live/scheduled/ended badge, coverage counters on pregnancy/phase maps.
- Open: CSP allows only `https:` images (local uploads didn't render — check on stage); `cycle_day_to` gte message
  renders an empty attribute (check against Laravel's output before changing).

## T-M2-24b — Apply the approved deviations consistently
- User approved all proposed deviations on 2026-09-23; `deviations.md` rows now `decided (approved 2026-09-23)`.
- D-01: `/reminders/enums` with a label-less locale → English labels (200). D-02: reminders PUT/DELETE non-numeric
  `{id}` → framework JSON 404 (all other typed-id routes already 404). D-16: internal banner `link_url` must start
  with a single `/` (422 `validation.regex`).
- Allow-list `contract/allowlist/reminders.yaml` (D-01, D-02 only): contract all 986 passed, 11 allow-listed diffs.
- OpenAPI keeps the Laravel-only 500s documented as "until cutover" (golden validation test ignores the allow-list).
- T-M2-09 unblocked 2026-09-23: staging deployed, prod proxy recreated once (stage-only files synced; prod
  `proxy-ssl.conf`/`vhost-api.inc` on the server are still pre-M2 and pre-HSTS → sync with T-M2-26). content flip
  on/off proven on stage.

## T-M2-28 — Staging runs Go only
- `vhost-stage.inc`: `/up`, `/api/` (all), `/storage/`, `/docs` → backend-go (request-time resolve); `/oauth/` 404;
  `/admin*` → 301 `/panel/` (no gate cookie on the redirect). Nothing routes to Laravel on stage.
- backend-go `RUN_MIGRATIONS=true` → `db.MigrateOnStart`: goose table → pending only; Laravel-only DB → StampBaseline;
  empty → baseline; unknown tables → refuse. Tested on MariaDB (5 cases). Only ONE instance may run it.
- Stage compose: Laravel `backend`/`queue` under profile `laravel` (alias now `stage-backend-laravel`); backend-go
  also answers the `stage-backend` alias (keeps stage-ssl.conf's load-time upstreams resolvable). Prod unchanged.
- `deploy-stage.sh` removes the old Laravel stage containers and asserts Go on `/up`, `/api/v1/languages`, etc.
- Consequence: no Laravel rollback on stage any more (rollback = redeploy an earlier stage commit). A fresh stage
  volume needs Passport keys + personal client created once (cutover.md).
- Follow-up: move stage-ssl.conf upstream blocks to request-time resolution so a missing stage container can
  never fail `nginx -t` on the shared prod proxy.
- Deployed 2026-09-23: goose stamped the Laravel-built stage DB (`stamped_laravel_schema`, version 1); Laravel
  stage containers removed; every deploy-stage check green (`/up`, `/api/*`, unknown `/api/v1/*` all `X-Backend: go`,
  `/admin` → 301 `/panel/`, `/oauth/token` 404). Smoke 42/42 from Go (diffs only from data written during testing).

## T-M2-29 — Go-first developer tooling
- `backend-go/CLAUDE.md` + README (THE backend; make targets; schema-change rule goose + mirror Laravel migration +
  `make schema-diff`), FROZEN banner on `backend/CLAUDE.MD`, `frontend/CLAUDE.md` §8.1 (spec =
  `backend-go/api/openapi.yaml`, local API `:8020`).
- Skills: `local-dev` (backend-go on :8020 against the test stack, goose baseline + seed, OTP via `SMS_PROVIDER=log`
  read from the DB — proven end to end), `verify-all` (Go first, Laravel only when backend/ changed).
- Agents: `backend-reviewer` rewritten for Go; Go paths/commands in `test-runner` and `security-auditor`.
- Hooks: `.claude/hooks/go-lint.sh` (golangci-lint fmt + lint gate per edited .go file).
- Open: `backend-go/resources/translations` drifted from `frontend/messages` (home/profile/pwa) — syncing means
  re-recording the `public` goldens; a `make migrate` target would help.

## T-M3-04 — Frontend — care-reminder entities, mutations, tokens, icons and i18n
- `entities/care-reminder` (types, zod parsers tolerant of nested/flat `meta`, unknown enums → safe default,
  `careKeys`, queries incl. `useCareToday`/`useCareEnums`), features `manage-medication`, `manage-appointment`,
  `log-intake` (optimistic update of every cached `careKeys.today` for the date, rollback on error).
- Tokens (light + dark): `--success(-soft)`, `--amber-tile(-ink)`, `--care-rose(-soft|-line)`, `--care-card-shadow`;
  icons capsule/tablet/video/phone/mapPin/clock/bellRing/note; `care` namespace fa/en (+ `global.d.ts` typing).
- Contract assumptions for T-M3-01/02: flat snake_case bodies; untick = `DELETE …/intakes?date=&slot=` (query);
  prep toggle = `PUT /care/appointments/{id}` with the full `prep` list (apiClient has no PATCH).
- Dev-only Next rewrites `DEV_CARE_API_ORIGIN` / `DEV_LEGACY_API_ORIGIN` (documented in `.env.example`).
- Open: `care` must be added to `message-scopes.ts` by the first screen that uses it (T-M3-05..08; the scope test
  requires exact usage); some copy not on artboards (empty/error states, duration labels) needs content review;
  new tokens not yet in `lint:dark` pairs — run `/check-colors` when UI lands.

## T-M3-01 — Care reminders — schema, reminder_intakes table and medication API (Go)
- Migration pair: `backend-go/db/migrations/00002_reminder_intakes.sql` (`CREATE TABLE IF NOT EXISTS`) +
  `backend/database/migrations/2026_09_23_000001_create_reminder_intakes_table.php`; `make schema-diff` OK (39 tables).
- `internal/care` (enums, labels via embedded `lang/{fa,en}/care.json`, medication meta v1 + `Covers(date)`,
  handlers), `routes_care.go`, queries `db/queries/care`; 8 operations in OpenAPI (tag `Care`); D-17 (care is Go-only).
- Medication rows surface in `GET /api/v1/reminders` (type=medication, subtitle, recurrence daily|weekly).
- Intake untick accepts `?date=&slot=` (query) or JSON body; ticking a future day → 422.
- **Convention (all later migrations):** post-baseline goose migrations must be no-ops where the Laravel twin already
  created the object (`IF NOT EXISTS` etc.) — prod stamps the baseline and then runs them over a Laravel-built schema.
  Migration integration tests (`cmd/api/migrate_test.go`, `testdb_test.go`) now compare against the latest version
  instead of assuming baseline-only.
- Open: `GET /reminders` OpenAPI `Reminder.meta` no longer requires `phone`.

## T-M4-05 — Frontend — checkups entity, features, tokens, icons, i18n and on-device attachments
- `shared/lib/local-files` (`createLocalFileStore`: IndexedDB + memory backend, per-file 15 MB / total 300 MB caps,
  image/* + PDF only, `LocalFilesError`), `entities/checkup` (types, `checkupIcon()` with aliases, attachments store
  keyed by record id, `checkupKeys`, tolerant zod parsers, queries incl. infinite records), features `record-checkup`,
  `manage-custom-checkup` (+ optimistic settings toggles). Tokens `--checkup-due/-body/-row-action` + `.ck-status-*`/
  `.ck-tone-*`; icons ribbon/flask/tooth/camera/history/filterLines/export; `checkups` namespace fa/en.
- Open: `checkups` must be added to `message-scopes.ts` by T-M4-06..09; orphaned local attachments after deleting a
  custom checkup → clean up in T-M4-09 (compare `useCheckupAttachmentIds` with records).

## T-M3-02 — Care reminders — appointment API with prep checklist and cancel (Go)
- `internal/care/appointment*.go`, `db/queries/care/appointments.sql`, 7 routes in `routes_care.go`, OpenAPI
  (`Appointment`, `AppointmentInput`, `PrepItem`).
- Choices: `scope=upcoming` (default; scheduled & future, soonest first) | `past` (past or cancelled, newest first) |
  `all`; `scheduled_at` accepts `Y-m-d H:i[:s]` (Tehran), response adds `remind_at`, `days_until`; prep ids `pN`
  (client temp ids replaced); PUT ignores `status` (only `/cancel`, idempotent); `PATCH …/prep/{itemId}` exists too.
- Open for T-M3-03: whether an inactive (`is_active=false`) appointment counts as `next_appointment`.

## T-M4-01 — Checkups — schema, default catalog seed and status engine (Go)
- Migration pair `00003_checkups.sql` + `2026_09_24_000001_create_checkup_tables.php` (`checkup_types`,
  `checkup_records`, `user_checkup_settings`; 6 seeded types, `INSERT IGNORE`); schema-diff OK (42 tables).
- `internal/checkups` (`EngineInputs`, converters), `internal/checkups/engine` (pure `Evaluate`, `NextDueAfter`,
  `AddMonths`, `CycleFromHistory`, Jalali port) — 98.7 % coverage; D-18 (checkups Go-only).
- Engine choices to review: `disabled` items stay listed but leave the summary; `not_yet` counts as up to date;
  only monthly cycle-timed types sit on predicted windows; users above `age_max` get no item; `soon` = 60 days.
- **Open for the user: seed copy (why/prep text, hide_in_pregnancy flags) needs medical review before production**
  (flagged in `source_note`, README, migration headers).
- Notes for T-M4-02/03: `EngineInputs` returns active types only (detail/admin need own queries); callers load
  birthday/pregnancy/cycle; column `key` must stay backticked.

## T-M3-03 — Care reminders — GET /care/today aggregate for the home card
- `internal/care/today.go` (+ unit/int tests, fixed clock), queries `ListIntakesOnDate`, `ListAppointmentsFrom`;
  `GET /api/v1/care/today[?date=Y-m-d]`, exactly 3 queries (asserted via a counting DB wrapper); OpenAPI `getCareToday`.
- Paused medications excluded; `next_appointment` = first scheduled future appointment (reminder off still counts,
  cancelled never); `days_until`/`next_appointment` relative to now, not `?date=`.
- Open: README example numbers are inconsistent (`total: 2` with one dose, `days_until: 8` for 7 days).

## T-M5-04 — Frontend — fertility entity, day-log feature, tokens, icons and i18n
- `entities/fertility` (enum tuples, BBT helpers `formatBbt/parseBbt/isBbtInRange/stepBbt`, tolerant zod parsers,
  `fertilityKeys`, queries), `features/log-fertility-day` (`toFertilityDayBody`, `useSaveFertilityDay` invalidating
  fertility/cycle/messages/health-log). 18 `--fert-*` tokens (light+dark), `.fert-tone-*`, `.fert-disc`; icons
  flaskLh/heartLine/target/moonReminder; `fertility` namespace fa/en.
- Open: add `fertility` to `message-scopes.ts` in T-M5-05..08; LH/BBT tip copy needs content review.

## T-M5-01 — Fertility — fertility_logs table, day log and today API (Go)
- Migration pair `00004_fertility_logs.sql` + `2026_09_25_000001_create_fertility_logs_table.php`; schema-diff OK (43).
- `internal/fertility` (merged day over `daily_health_logs` + `fertility_logs`, Save in one tx through
  `healthlog.Service.Store` so side effects match `POST /health-logs`), routes `GET /fertility/today`,
  `GET|PUT /fertility/days/:date`; OpenAPI; D-19. `today` = 3 queries.
- Choices: symptom chip writes severity `low` (no `mild` enum; existing medium/high kept); spotting gives luteal
  warning + recalculation flag but no period start (same as health-logs); chance from resolver `fertility_level`
  (`unknown` → `level: null`, label «نامشخص»); `bbt` decimal string, `bbt_time` `HH:MM`.
- Next goose number: 00005.

## T-M4-03 — Checkups — admin API for the checkup-type catalog and stats
- `internal/admin/checkups` (+ unit/int tests), `routes_admin_checkups.go` (self-registers), queries
  `db/queries/checkups/admin.sql`; docs `admin-api.md` §12 (+ `in_use` error code).
- Endpoints under `/api/admin/v1/checkup-types`: list (q/status/paging, `records_count`), `options`, `stats`,
  `reorder` (full id list, tx), create/show/update/delete (delete refused with 422 `in_use` when records exist).
  Custom user types are invisible (404). Translatable fields required in every active language.
- Notes for T-M4-04: options are plain string lists (admin-web translates labels); send `is_active`/
  `hide_in_pregnancy` on every save; `key` read-only after create; `overdue_users` is an approximation.
- T-M4-02 should assert admin edits show up directly on `GET /api/v1/checkups/{id}`.

## T-M7-08 — Frontend — pregnancy v2 entity/features, tokens, illustrations, icons, i18n
- `entities/pregnancy` v2 alongside v1 (`v2-types`, `v2` helpers, `v2-schema` zod parsers, `v2-queries`,
  `pregnancyKeys` moved to `api/keys.ts` with `v2.*`); v1 mutations also invalidate `pregnancyKeys.v2.all()`.
- `features/track-pregnancy` v2 mutations (`useSavePregnancyDay`, `useUpdateWeekState` optimistic,
  `usePregnancyAlertAction`) + `toPregnancyDayBody`.
- `shared/ui/illustrations` (WelcomePregnancy, ResultBaby, FetusSize; 36 `FETUS_ILLUSTRATION_KEYS`), 7 `--preg-*`
  tokens + `.pg2-*` classes, 11 icons, `pregnancyV2` namespace fa/en.
- Open: add `pregnancyV2` to `message-scopes.ts` in T-M7-09..14; clinician review for `log.spottingBody`,
  `alerts.legend.*`, `today.disclaimerBody` (T-M7-15).

## T-M4-02 — Checkups — user API (list, home, detail, records, custom, settings)
- `internal/checkups/{service,handlers,custom,json,request,labels}.go`, queries `db/queries/checkups/user.sql`,
  `routes_checkups.go` (12 routes), OpenAPI Checkups tag; labels server-side (Jalali months + Persian digits for fa).
- `LoadPlan` = 3 queries (asserted on `/checkups/home`); `POST`/`PUT` record → `{item, record}`; home `data: null`
  when nothing is enabled; records list `{items, meta}` 20/page; status `disabled` for switched-off types (frontend
  enum + `.ck-status-disabled` + labels added in the same commit).
- Admin edits visible immediately on `GET /checkups/{id}` (tested).

## T-M7-01 — Pregnancy v2 — new tables and seeds
- Migration pair `00005_pregnancy_v2.sql` + `2026_09_26_000001_create_pregnancy_v2_tables.php`:
  `pregnancy_week_details` (weeks 1–42), `pregnancy_care_items` (5), `pregnancy_daily_extras`,
  `pregnancy_week_user_state`; 118 `message_contents` rows (week tips, 8 alert rules + legend, setup copy; fa+en,
  no explicit ids — unique on group/item_key/locale). schema-diff OK (47 tables, seed contents equal).
- sqlc queries `db/queries/pregnancy/v2_*.sql` (incl. admin writes for T-M7-06); D-20.
- Admin messages/languages integration tests now clear `message_contents` first (migrations seed rows).
- **Open for the user: ALL seeded pregnancy v2 copy is placeholder needing clinical review** (week sizes/figures,
  headlines, tasks, warnings, care-plan windows, alert thresholds/levels/texts, tips, setup copy); `reviewed_at` NULL
  and `sources` carries `[needs review]`. Sign-off in T-M7-15.
- Notes: `legend` item in `pregnancy_alert` is not a rule; tip payload `{title, body, read_minutes, article_url}`;
  use `UpsertDailyExtras` / `SetDailyVisitNote`; user-state keys on `week`.

## T-M3-05 — Frontend — «یادآورهای امروز» home card on both homes + AddChooser sheet
- `widgets/today-reminders` (card + pure `dose-row` helpers, 19 tests), `screens/reminders-add` (AddChooser sheet,
  registry key `reminders-add`); mounted on the cycle home (replacing the commented DayTasks) and the pregnancy home.
  Styles in `globals.css` (`trm-*`, `rad-*`), existing tokens only. `care` added to shell + home/pregnancy scopes.
- Checked headless (fa/en, light/dark, empty state, sheet); tick round-trip updates `/care/today`.
- Links to `/reminders`, `/reminders/medication/new`, `/reminders/appointment/{new,id}` land in T-M3-06..08.
- Open: pregnancy-home mount not screenshotted (test user not pregnant).

## T-M5-02 — Fertility — BBT shift engine (3-over-6) and GET /fertility/bbt
- Pure engine `internal/fertility/bbt` (100 % coverage; hundredths as ints), `chart.go` `Service.BBT` (3 queries),
  query `ListBBTReadings`, `GET /api/v1/fertility/bbt?range=1|3|6` (422 otherwise), OpenAPI `getFertilityBbt`.
- Choices: 3-over-6 over consecutive readings (gaps skipped); provisional coverline before a confirmed shift
  (max of last 6, `null` with < 6); `pre_ovulation_avg` = mean of first 6 readings; `gaps` = days so far − logged;
  current cycle from the engine's anchor; fertile window from the cycle view's `fertile` phase; `past_shift_days`
  independent of range.
- Open (design): artboard legend says «میانگین ۶ روز اول» but the README/engine coverline is the max of the 6 readings
  before the shift — T-M5-07 labels it accordingly.

## T-M5-03 — Fertility — GET /fertility/insights (window, confidence, evidence, history)
- `internal/fertility/insights.go` (pure core + localisation), query `ListLHTests`, `GET /api/v1/fertility/insights`
  (4 queries), OpenAPI `getFertilityInsights`; confidence rule documented in `docs/fertility-ttc/README.md`.
- Window = current cycle until its ovulation passed, then the next predicted one; `null` without a resolvable cycle.
- Tips are plain localized strings; history rows also carry `date`, `source (bbt|lh|estimate)`, `cycle_start`.

## T-M3-06 — Frontend — Reminders screen (/reminders) with tabs, dose strip and lists
- Route `/[locale]/reminders` (`?tab=` kept in URL, SSR-correct), `screens/reminders` (pure `model/view.ts` + 11
  tests; TodayCard dose strip, MedicationSection with real switches, AppointmentSection upcoming list), `rmd-*`
  classes in globals.css, scope `reminders: ['care','common']`; Profile «یادآورها» row re-enabled → `/reminders`.
- Verified headless fa/en × light/dark against a local backend-go (tick, toggle, tab URL).
- Open: `slotClock`/`slotPeriod` duplicated from the home widget (move to the entity later); en full month names
  truncate in the 52px date tile (no short-month helper); back always goes to `/home` (not mode-aware).

## T-M7-06 — Admin API — week details, care plan, alert rules, create-message in registered groups
- `internal/admin/pregnancy` (week details `GET|PUT /pregnancy-weeks/{n}/details` + list/options; care items CRUD/
  reorder/toggle, delete refused 422 `in_use` when appointments link to it; alert rules list/show/update with typed
  params), `internal/admin/messages/registry` (group/key schemas, alert rule registry — single source for T-M7-04),
  `POST /messages` + `GET /messages/registry[/:group/:key]`, list `missing` per item × active language;
  queries `db/queries/pregnancy/v2_admin.sql`; docs `admin-api.md` §13.
- Fixed: legacy `PUT /messages/:id` merge flattened typed v2 payloads (params/actions/levels) — typed groups now
  validate + merge by schema.
- Enum sync: seeded illustration keys (pomegranate, butternut_squash, melon, romaine_lettuce, swiss_chard, leek,
  small_watermelon) and highlight icons (baby, face) added to the frontend lists and the Go admin enums.
- (Commit also carries T-M7-02's generated `v2_read` sqlc query so the shared `querier.go` compiles.)

## T-M7-02 — Pregnancy v2 — dating preview, today, week and week-state API
- New package `backend-go/internal/pregnancy/v2` (dating on top of `pregnancy/calc`, lang files
  `lang/{fa,en}/pregnancy_v2.json`, Jalali/Gregorian labels via `checkups/engine`) + `internal/http/routes_pregnancy_v2.go`.
- Endpoints (Go only, OpenAPI tag `PregnancyV2`): `POST /pregnancy/v2/dating-preview` (no write; basis/range text from
  `pregnancy_setup/result`), `GET /pregnancy/v2/today` (5 queries, asserted in tests), `GET /pregnancy/v2/weeks/{n}`
  (1–42, else 404), `PUT /pregnancy/v2/weeks/{n}/state` (partial; unknown/duplicate task keys dropped).
- 409 `{success:false, message, error_code:"pregnancy_not_active"}` when there is no profile, `pregnancy_mode = 0` or no dating.
- Decisions: week counts to 42 when overdue (`due.days_left` ≥ 0 plus `overdue_days`); usual birth range = due ± (14 +
  uncertainty) days; trimester 2/3 start at 13w0d / 28w0d; carousel `title` = «سه‌ماههٔ … · هفتهٔ N از ۴۰», the admin
  headline is a separate `headline`; week tip read straight from `pregnancy_week_tip` (swap for the T-M7-04 engine layer).
- Frontend contract notes: `details.body_symptoms` is a list of labels (what `v2-schema.ts` parses); keys for
  deep-linking are in the extra `details.body_symptom_items: [{key,label}]`. Everything else matches the primary shapes.
- Open: docs/pregnancy-v2/README.md not updated (outside `touches`); no contract golden (Go-only group).

## T-M7-03 — Pregnancy v2 day log API
- Shipped `GET/PUT /api/v1/pregnancy/v2/days/{date}` and `GET /api/v1/pregnancy/v2/report?from=&to=` (backend-go/internal/pregnancy/v2/daylog).
- v1 symptoms/weight written through v1 upserts (same alert rules) via new exported bridge `internal/pregnancy/daylog_bridge.go` (outside `touches`, needed because v1 helpers are unexported); extras in `pregnancy_daily_extras`.
- PUT idempotent; explicit null clears; future/invalid date 422; non-pregnant 409 `pregnancy_not_active`.
- Open: alerts map v1 levels (emergency→urgent, warning→follow_up); `actions/what_we_saw/how_sure` await T-M7-04 message engine. Multi-table save not transactional (idempotent retry). Day `weight` = that pregnancy week's weight.

## T-M3-07 — Frontend — Add/Edit medication screen
- Shipped `/reminders/medication/new` and `/reminders/medication/[id]` (screens/reminder-medication-form): times↔count, weekday summary, zod mirror of Go validation, 422 mapping, delete confirm, `?from=home` return.
- `frontend/messages/{fa,en}/care.json` keys added (this commit also carries T-M3-08's keys in the shared file).
- Open: light/dark fa/en screenshots not taken (no local stack running); home card links don't pass `from=home` yet.

## T-M3-08 — Frontend — Add/Edit appointment and appointment detail screens
- Shipped `/reminders/appointment/new`, `/[id]`, `/[id]/edit`; `shared/lib/ics` builder (Asia/Tehran TZID, VALARM = remind_before) with tests.
- Open: screenshots not taken; prep ticking uses PUT fallback (no apiClient.patch); «مسیریابی» uses a Google Maps search link.

## T-M5-06 — Frontend — «ثبت روز» fertility day log screen
- Shipped `/fertility/log?date=&focus=` (screens/fertility-log): chance card, LH/mucus/intercourse/symptom chips, BBT stepper with Persian-digit input, changed-fields-only save, dirty guard.
- Added `fertilityLog` to `frontend/src/app/message-scopes.ts` (outside touches, required by route typing) and key `fertility.log.discardConfirm`.
- Open: screenshots not taken; back goes to `/home` (no `?from=`); inline toast (no shared primitive).

## T-M4-07 — Frontend — Checkups screen (/checkups) and custom checkup form
- Shipped `/checkups?filter=` (status card, tabs, sections, plan-settings sheet) and `/checkups/custom/new`, `/checkups/custom/[id]` (edit/delete).
- Added `checkups` to `frontend/src/app/message-scopes.ts` (outside touches); new `checkups.custom.*` keys.
- Open: T-M4-08 detail should link custom checkups to `/checkups/custom/{id}`; detail API has no `note` (edit sends note only if changed); settings sheet reads remind per-row via detail. Screenshots not taken.

## T-M5-07 — Frontend — «دمای پایه» BBT chart screen
- Shipped `/fertility/bbt?range=` (screens/fertility-bbt) and `widgets/bbt-chart` (inline SVG, token colours, coverline, fertile band, previous cycles, a11y table); daily 07:00 reminder toggle via `useCreateReminder`.
- Outside touches: `fertilityBbt` in `message-scopes.ts`; `widgets/bbt-chart` added to insignificant-slice ignore in `frontend/steiger.config.ts`.
- Open: explainer is an inline AppSheet (not registry); legacy `bbt.legend.coverline` text left as-is (new `legend.baseline` used). Screenshots not taken.

## T-M4-08 — Frontend — Checkup detail screen and MarkDone sheet
- Shipped `/checkups/[id]` (screens/checkup-detail) and registry sheet `checkup-mark-done` (arg `<typeId>` or `<typeId>-<recordId>`); attachments stay in IndexedDB, only `has_attachment` sent.
- `checkups` namespace moved into `SHELL_NAMESPACES` (sheet can open over any route).
- Open: «ثبت نوبت» not prefilled (appointment form only reads `?kind=`); `/checkups/self-exam` and `/checkups/history` come in T-M4-09; toast only on `/checkups/[id]`; no delete in edit flow. Screenshots not taken.

## T-M5-08 — Frontend — «پیش‌بینی‌ها» insights screen
- Shipped `/fertility/insights` (screens/fertility-insights): window card with confidence + month calendar, evidence rows, previous-cycle ovulation list, tips, low-data state.
- `fertilityInsights` added to `message-scopes.ts`; new `fertility.insights.*` keys.
- Open: evidence icons keyed on `cycles/bbt_shift/bbt/lh/mucus/history`; window spanning two months shows one month only. Screenshots not taken.

## T-M4-06 — Frontend — «چکاپ‌های دوره‌ای» card on the cycle home
- Shipped `widgets/checkups-card` (ring, counts line, up to 2 highlight rows, disclaimer) mounted on the cycle HomePage.
- Outside touches: `checkups` in `ROUTE_NAMESPACES.home`; steiger ignore for the widget. Combined frontend build green after T-M3/M4/M5 batch.
- Open: «ثبت نوبت» passes `title=` but the appointment form ignores it; not shown on pregnancy home; visual check vs artboard pending.

## T-M7-04 — Message engine: pregnancy week tips and v2 alert rules
- `/messages/daily` pregnancy layer 0 = `pregnancy_week_tip/{week+1}`; overrides also read `pregnancy_symptom_logs` + `pregnancy_daily_extras` (mood ≤2 → mood_sad).
- New `internal/messages/pregnancyalerts` (8 DB-driven rules; rows `alert_type=v2:<rule>` in `pregnancy_alerts`, texts resolved at read time).
- New `GET /pregnancy/v2/alerts`, `POST /pregnancy/v2/alerts/{id}/actions/{ack|add_to_visit_note}`; day-log `alerts` now full v2 alerts.
- Outside touches: `db/queries/pregnancy/v2_alerts.sql` (+sqlc), v1 `AfterLogSave` hook, daylog wiring. Deviation D-21 approved by user 2026-09-26.
- Contract: messages,pregnancy 278 passed.
- Open: no scheduler (daily eval runs on GET alerts); fixture lacks v2 tables (1146 treated as no row); v1+v2 duplicate rows for some symptoms; `weight_missing_week` not withdrawn; rule param interpretations need clinical review (T-M7-15).

## T-M5-05 — Frontend — TTC quick tiles on the cycle home
- Shipped `widgets/fertility-tiles` (LH / BBT / intercourse) on HomePage when `pregnancyIntention === 'trying'`; `fertility` added to home scope; steiger ignore. Full verify incl. build green.
- Open: tile colours/icons chosen without artboard check; screenshots not taken.

## T-M4-04 — admin-web — checkup types screens
- Shipped `/checkup-types` list (drag/arrow reorder, active switch via full PUT, stats panel), create/edit form with language tabs and live preview. Touches widened to nav.ts, admin-web/messages, Icon.tsx.
- Verify (tsc, eslint, steiger, vitest 68, build) green.
- Open: local smoke vs Go admin API not run; light preview needs `[data-theme="light"]` tokens in admin globals.css; rose/teal tones borrow danger/data tokens.

## T-M7-05 — Pregnancy v2 calendar, care plan and visit stages
- `GET /api/v1/pregnancy/v2/calendar?month=` (locale calendar; days, visits, next_visit, care_plan with window/state/suggested_date, source_note).
- Appointment meta gains optional `care_item_key`, `stage` (booked|done|result), `result_note` (backwards compatible).
- Outside touches: exported date helpers appended to `internal/pregnancy/v2/labels.go`.
- Open: care_item_key not FK-checked; visit prep comes from care item, not appointment checklist; client filters selected-day visits.

## T-M4-09 — Frontend — checkup history and self-exam
- Shipped `/checkups/history` (tabs, timeline, infinite list, edit via mark-done sheet, on-device «خلاصه برای پزشک» PDF) and `/checkups/self-exam` (cycle-window hero, guide steps, findings chips, monthly record, adherence).
- `shared/lib/pdf`: hand-written PDF writer + canvas renderer (Vazirmatn shaping) — no new dependency; pages are images (text not selectable).
- Open: PDF not verified on a real device; finding labels shown as a count; adherence reads first page only; self-exam resolves `breast_self_exam` key. Screenshots not taken.

## T-M7-07 — admin-web — pregnancy v2 editors
- Week structured details tab (`?tab=details`, PUT /pregnancy-weeks/:n/details) with fa/en tabs and hero preview; new `/pregnancy-care-plan` and `/pregnancy-alert-rules` screens (typed params, sample card); messages screen gets a missing-content panel (week-tip 1–42 grid) and schema-driven create/edit for typed groups; new «بارداری» sidebar group.
- Verify (tsc, eslint, steiger, vitest 77, build) green.
- Open: fetal image picker has no real preview (SVGs live in frontend/); some highlight icons missing in admin Icon.tsx; message create lives inside `/messages`; not smoke-tested against the API.

## T-M7-11 — Frontend — pregnancy «هفته‌به‌هفته» (Week) v2 screen
- Shipped `/pregnancy/weeks` (→ current week) and `/pregnancy/weeks/[n]`: week strip 1–42, hero with fetus size, tabs (fetus / body / tasks with synced checklist), warning, reviewer + sources, bookmark, swipe navigation.
- Fixed `message-scopes.test.ts` namespace regex to allow digits (`pregnancyV2`).
- Open: body tab links `/pregnancy/log` (revisit after T-M7-12); nothing links to `/pregnancy/weeks` yet. Screenshots not taken.

## T-M7-09 — Frontend — pregnancy entry points, mode-aware nav, Setup v2
- Intention step back in signup (pregnant → pregnancyBasis → conditions); profile mode switch (→ `/pregnancy/setup`, confirm on switch back); pregnancy bottom nav (امروز · تقویم · FAB log · بارداری `/pregnancy/weeks` · پروفایل); Setup v2 flow (welcome → dating / history / result via dating-preview) at `/pregnancy/setup` (and `/pregnancy/onboarding`).
- Outside touches: `entities/user/model/steps.test.ts` updated for both branches.
- Open: signup pregnancy-basis screen doesn't reuse the new SetupSteps (needs a shared `features/pregnancy-setup` slice); `/pregnancy/calendar` comes in T-M7-13. Screenshots / running-app check pending.

## T-M7-10 — Frontend — pregnancy «امروز» (Today) v2 screen
- Rewrote `screens/pregnancy` on `usePregnancyToday`: date strip, `widgets/pregnancy-week-carousel`, due card, 40-week progress, quick actions (alerts badge), next visit, smart tip, `widgets/pregnancy-care-checklist`, disclaimer, not-active CTA → `/pregnancy/setup`. Removed v1 AlertsCard/WeekContent.
- `pregnancy` route scope now `care, common, nav, pregnancyV2`; steiger ignores for the two widgets. Commit also carries in-flight T-M7-12 keys in `pregnancy-v2.json` (keys only).
- Open: `/pregnancy/alerts` arrives with T-M7-14; kept M3 TodayRemindersCard; screenshots/build pending.

## T-M7-12 — Frontend — pregnancy Log v2 with offline outbox
- `/pregnancy/log?date=` now renders v2 DayLogPage (mood, 9 symptoms + severity, water, weight w/ last entry, visit note, spotting info, inline alerts); `?tab=symptoms|weekly|movement` keeps v1 forms.
- New `shared/lib/outbox` (IndexedDB, latest-write-per-day, replay on mount/`online`, pending badge).
- Open: manual offline → reconnect test not done yet (if unreliable, switch offline copy to honest wording); screenshots/build pending.

## T-M7-13 — Frontend — pregnancy Calendar v2 + doctor PDF
- `/pregnancy/calendar`: locale month grid with visit dots/week starts, selected-day panel, next-visit card with 3-stage stepper (PUT `{stage}` / result note sheet), care plan rows, source note, 28-day doctor PDF via `shared/lib/pdf`.
- `pregnancyCalendar` scope added; `calendar.saveResult`/`calendar.pdf.*` keys (commit may also carry in-flight T-M7-14 keys in pregnancy-v2.json).
- Open: appointment form ignores `title/date/care_item_key` prefill (also affects checkups card/detail) and `manage-appointment` body drops `stage/result_note` — needs a follow-up task; PDF not eyeballed; screenshots/build pending.

## T-M7-14 — Frontend — pregnancy «هشدارها» (Alerts) v2 screen
- `/pregnancy/alerts`: day-grouped v2 alert cards (level chip, what we saw, how sure, advice, actions incl. ack / add_to_visit_note / deep links / tel), urgent contact line, API legend, disclaimer + notifications sheet, marks shown alerts read.
- Open: log route ignores `?focus=weight`; entity schema drops API title/window_note/disclaimer (i18n used); mark-read uses v1 read-all; `open_week` → current week. Screenshots/build pending.

## T-M7-13b — Frontend — appointment form prefill and stage fields
- `/reminders/appointment/new` reads `kind,title,date,care_item_key` prefill; `manage-appointment` body + care-reminder entity carry `care_item_key/stage/result_note`; calendar stage stepper uses `useUpdateAppointment`; remind values from `REMIND_BEFORE`.
- Full frontend verify on the combined M3–M7 tree green (551 tests, build OK).
- Open: end-to-end check that booking from the care plan flips the item to `booked`.

## T-M3-09 — Care reminders rollout (staging)
- verify-all green (Go, frontend 551 tests + build, admin-web 77 tests + build); `stage` deployed; goose v5 applied (reminder_intakes, checkups, fertility, pregnancy v2 tables present).
- `/api/v1/care/*` served by Go on stage (Go-only vhost, no route change needed); API e2e med→today/home→tick→appointment→detail→cancel passed. Evidence: `docs/care-reminders/README.md` § Rollout.
- Open: UI screenshots light/dark (needs stage password, human), prod route when asked.

## T-M7-15 — Pregnancy v2 rollout (staging API e2e)
- No redeploy (T-M3-09 shipped it; stage is Go-only). Seeds present: 42 week details, 5 care items, 42+42 week tips, 9+9 alert rows.
- API e2e on stage-backend-go with 09900000903: dating-preview → onboarding → today → week/9 + state → day PUT/GET (spotting → urgent critical_symptom) → alerts ack + add_to_visit_note → calendar → appointment care_item_key=nt_scan → calendar booked → report; all 2xx. Test data deleted. Evidence: docs/pregnancy-v2/README.md § Rollout (T-M7-15).
- Human sign-off needed: week details 1–42 + week tips, care plan windows (first_visit, nt_scan, anomaly_scan, gtt, tdap), alert texts, and rule params: vomiting_streak 3 days / 2 severe in 7d; severe_symptom_count ≥3 severe in 7d over 8 symptoms; critical_symptom (bleeding, fluid_leakage, severe_sudden_pain, spotting ≤ week 12 unless severe) 1d; weight_missing_week from week 1; bp_high ≥140/90; sugar_high fasting >95 / post-meal >140; fetal_movement reduced|none from week 24.
- Open: UI light/dark click-through + screenshots (needs gate password); calendar `month` is locale-calendar (Jalali for fa); v1+v2 duplicate alert rows for spotting.

## T-M4-10 — Checkups rollout (staging)
- No redeploy (shipped by T-M3-09). Go API e2e on stage passed: list/home/detail/preview → mark done → history → record edit/delete → settings → custom create/edit/delete; admin checkup-types login/list/create/edit/reorder/deactivate/stats/delete with a temporary editor, all cleaned up. Evidence: `docs/checkups/README.md` § Rollout.
- Open (human): catalog content sign-off, UI light/dark screenshots (needs stage password), prod when asked.

## T-M5-09 — Fertility rollout (local e2e, staging still blocked)
- verify-all green (Go 63 pkgs, lint 0, fertility int tests; frontend typecheck/lint/fsd/styles/dark/551 tests).
- Local API e2e as a `trying` user: today → day log (LH+/BBT/intercourse) → 422 validation → 12 days BBT → `/fertility/bbt` coverline 36.40, shift day 13, post_shift → `/fertility/insights` confidence high. UI light/dark screenshots in `docs/fertility-ttc/screenshots/`. Evidence: `docs/fertility-ttc/README.md` § Local e2e.
- Bugs found → T-M5-10 (tile colours, gutter, card padding, Latin digits, chart decimal, 2-month window calendar).
- Open: `fertility_level` mismatch `/cycle/today` top-level (low) vs `daily_card` (medium) — product decision; staging e2e needs server access / user.

## T-M5-10 — Fertility UI polish from local e2e
- Tiles use `.fert-disc` + `.fert-tone-*` (LH amber, BBT teal, intercourse rose); tiles row `mx-4 mt-4.5`; `p-4` on fertility log/BBT/insights cards (global `.card` untouched).
- Persian digits via `formatNumber` for basedOn/gaps; chart Y-axis via new `yTickLabel` → `formatBbt`; insights calendar draws every month the window touches (`windowMonths`).
- Frontend verify green (555 tests, fsd/style/dark gates); 16 light/dark screenshots re-taken in `docs/fertility-ttc/screenshots/`.
- Open: `fertility_level` mismatch (home «متوسط» vs log «کم»), Lalezar «٫» glyph, insights window teal vs spec amber — design/product decisions.

## T-M4-10 — Checkups rollout (local e2e)
- verify-all green. Local e2e light/dark: home card → list → detail → MarkDone with on-device attachment (IndexedDB) → history → PDF → self-exam → custom form; admin-web edit of `blood_test` title visible in the app. 41 screenshots in `docs/checkups/screenshots/`; evidence in `docs/checkups/README.md` § Local e2e.
- Bugs → T-M4-11 (PDF footer raw key, card padding, Latin digits, overdue section order).
- Still open (human): catalog content sign-off (6 seeded items listed in README), staging UI click-through (no server access for the agent), production when asked.

## T-M4-11 — Checkups UI polish from local e2e
- PDF footer uses `t.raw` + new `formatPdfFooter` (locale digits; `PdfDocumentSpec.locale` now required) — fixes checkups history and pregnancy calendar PDFs.
- `p-4` on checkups list/detail/history/self-exam cards; `formatNumber` for history count and self-exam days; `groupBySection` puts this_month then overdue first (frontend, API order undocumented).
- Frontend verify green (559 tests); screenshots re-taken in `docs/checkups/screenshots/`.
- Open: pregnancy calendar PDF not generated in a browser (unit-tested only); minor design items 5a–5d in README.

## T-M7-15 — Pregnancy v2 rollout (local e2e)
- verify-all green. Local UI e2e light/dark: pregnant signup → Setup v2 → Today → Week → Log (incl. offline outbox sync) → spotting at w10 fires urgent alert → Alerts → Calendar booking from care plan (item → booked) → doctor PDF (Persian footer) → switch back to cycle. 60 compressed screenshots in `docs/pregnancy-v2/screenshots/`; evidence in `docs/pregnancy-v2/README.md` § Local e2e.
- Bugs 1–8 → T-M7-16. Clinical/content questions for the reviewer: 9c (no `contact.phone` in seed → no call-doctor button), 9j (LMP ±5 vs ±3 days copy; spotting info box says common but rule fires urgent).
- Still open (human): clinical sign-off, staging UI click-through, production when asked.

## T-M7-16 — Pregnancy v2 UI fixes from local e2e
- Today carousel `shrink-0`; trimester fills via new `trimesterFills` (API contract unchanged); Setup v2 stepper per design; locale digits on Week, Profile and Log weight.
- Go `pregnancyalerts` render localizes numeric vars (`v2.Digits`) — «وارد هفتهٔ ۱۰ شدی».
- Log: queued vs saved status (`save-status.ts`), «ذخیره شد» only after server ack.
- SettingUpPage: decision logic in `model/plan.ts` — no save without intention, pregnant path always profile→activate→onboarding, retry UI resumes from the failed step.
- Verify green (573 frontend tests, Go vet/tests/lint); screenshots re-taken in `docs/pregnancy-v2/screenshots/`.
- Open: 9a NT prefill category (in pregnancy-calendar view, outside touches); outbox `rejected` leaves status queued; retry button uses generic copy (no new i18n key).

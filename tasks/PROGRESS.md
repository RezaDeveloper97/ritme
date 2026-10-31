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

## T-M2-25 — Staging rollout (local full smoke on Go)
- verify-all green incl. admin-web (77 tests). Local stack Go-only: web smoke of every route group light/dark (auth, onboarding, cycle, health log, messages, reminders, content, language switch, notifications, home toggles, logout/login, session refresh rotation) — no 5xx, only expected 4xx. admin-web: every nav screen + detail + /new, 163 API calls all 200, CRUD round (affirmation, banner with upload). 144 screenshots in `docs/go-migration/screenshots/`; evidence in `docs/go-migration/stage-rollout-log.md` § Local full smoke on Go.
- Bugs → T-M2-30 (hydration mismatch, Latin digits, «Ritme» in fa BMI copy, raw article slugs). vhost-admin note added to T-M2-26.
- Still open (human): staging web smoke on a phone, editor click-through of `/panel` vs Blade, 48h soak sign-off.

## T-M2-30 — Web app polish from full Go smoke
- Hydration: DayLogSummary treats not-mounted as loading; CyclePage mounts its cards after hydration (`useMounted`). No hydration errors on /calendar, /cycle (light/dark).
- Locale digits via `format.number` into existing messages (challenge chip/tooltip, calendar cycle day, period-editor badges + hint).
- fa BMI copy «ریتمی» in `bmi.go` + `defaults.json` — deviation D-22 (`docs/go-migration/deviations.md`), contract allowlist `contract/allowlist/profile.yaml`; `make contract ROUTES=all` 986 pass / 16 allow-listed.
- Article category labels `articles.categories.<slug>` (fa/en) via `screens/articles/model/category.ts`.
- Verify green (576 frontend tests, Go tests/lint).
- Open → T-M2-31: stage/prod `message_contents` BMI fa rows still say "Ritme" (DB wins over code); raw slugs on home article rail and related-articles rail; backend translation seed out of sync since T-M2-06.

## T-M2-31 — Follow-ups from T-M2-30
- Migration `00006_bmi_fa_brand.sql` (data only): fa BMI normal/obese "Ritme" → «ریتمی» only where the payload is byte-equal to the old seed (admin edits kept); reversible; `TestBmiFaBrandMigration` integration test. Runs on stage at next deploy, prod at cutover.
- Article category labels on home rail, article tag and related rail; helper moved to `entities/article/model/category.ts`; `ROUTE_NAMESPACES.home` ships `articles`.
- Backend translation seed resynced with `frontend/messages` (care, checkups, fertility, pregnancy-v2 added; articles/home/profile/pwa updated); `TestSeedMatchesFrontendMessages` guards drift; i18n testdata re-recorded; deviation D-23 + `contract/allowlist/public.yaml`.
- Verify: go test 65 pkgs, lint 0, schema-diff OK, contract 986 pass / 206 allow-listed; frontend 576 tests, fsd OK.
- Open: Laravel prod translation seed still old (until cutover); stale comments in `internal/i18n/translations{,_test}.go` and `backend-go/CLAUDE.md` resync paragraph.

## T-M4-10 — Checkups rollout (staging UI e2e, 2026-09-28)
- Stage @ ea8a1eb (goose v6), light/dark: home card → list → detail → mark done with on-device attachment → history → PDF (Persian footer) → self-exam → custom form; `/panel` edit of `blood_test` title visible in the app, then restored byte-identical. All API responses `X-Backend: go`; verify passed. T-M4-11 fixes confirmed on stage. 41 `stage-*` screenshots in `docs/checkups/screenshots/`; evidence in `docs/checkups/README.md` § Staging UI e2e.
- New bug S1: fa `·` separator reads like a Persian zero in counts lines (`messages/fa/checkups.json` separator, CheckupsCard).
- Still open: catalog content sign-off (human); production when asked. Gate password appeared once in agent tool output (curl redirect URL) — consider rotating.

## T-M7-15 — Pregnancy v2 rollout (staging UI e2e, 2026-09-28)
- Stage @ ea8a1eb, light/dark: pregnant signup → Setup v2 → Today → Week → Log (offline outbox queued → saved) → spotting urgent alert → Alerts → Calendar booking from care plan (NT → booked) → PDF (Persian footer) → switch back to cycle. 185 API calls, all `X-Backend: go`, all 2xx except known 9h. T-M7-16 fixes confirmed on stage. 67 `stage-*` screenshots; evidence in `docs/pregnancy-v2/README.md` § Staging UI e2e.
- New bug (privacy): `ritme-onboarding` persisted store (name, mobile, birth date, weight, height, intention, pregnancy basis) survives logout — `useLogout` only clears token + cache.
- Still open on stage: 9a/9b/9c/9d/9h/9i; Log weight field ASCII decimal. Clinical sign-off (human); production when asked.

## T-M5-09 — Fertility rollout (staging e2e, 2026-09-28)
- Stage @ ea8a1eb as a `trying` user, light/dark: tiles → log LH/BBT/intercourse → tiles update → BBT chart (coverline 36.42, shift day 13) → insights (3 cycles, high). All `/api/v1/*` `X-Backend: go`; 422 validation in fa; verify passed. T-M5-10 fixes confirmed on stage. 16 `stage-*` screenshots; evidence in `docs/fertility-ttc/README.md` § Staging e2e. Test user deleted.
- Open: `fertility_level` mismatch home vs log (product decision); tiles wait on `GET /profile` (~5s sometimes on stage → late render/layout shift, `HomePage.tsx:885`); `DELETE /account` leaves revoked `oauth_access_tokens` row (no FK cascade); Lalezar «٫» glyph; insights window colour vs spec.

## T-M7-17 — Fixes from staging e2e
- Privacy: `shared/session/cleanup.ts` (`onSessionEnd` registry) runs from `clearAuthToken` on logout, account deletion and `endsSession()`; `ritme-onboarding` store wiped + storage cleared; offline outbox cleared (no replay into the next account). Verified headless: no per-user localStorage keys after logout/reload.
- fa separator `·` next to digits → «، » (checkups, care, fertility, pregnancy-v2 incl. PDF day line); `common.separator/groupSeparator` keys.
- NT/care-plan booking prefills the appointment topic (`careItemTopic` → `?topic=` → `parseTopic`).
- `/messages/daily` 400 = no message yet → home empty state «تاریخ آخرین پریودت رو ثبت کن» with period CTA.
- Log weight field locale decimal. Seed + i18n goldens resynced.
- Verify: 592 frontend tests; go test ./... ok; contract 986 pass / 206 allow-listed.
- Open: on-device checkup attachments not cleared on logout (product decision); unsynced outbox entries are dropped on logout (intended); `POST /profile` without `last_period_start` creates a today period — Laravel parity unchecked.

## T-M2-25 — Staging rollout (staging smoke + soak, 2026-09-28)
- Stage @ ea8a1eb: web app smoke light/dark over every route group — 525 `/api/v1` responses all `X-Backend: go`, 0 5xx, only expected 404/401 (incl. token rotation → `token_revoked`). `/panel`: 233 calls on Go, every nav screen + detail + /new, affirmation CRUD, banner upload → shown in app → deleted. `/admin` → 301 `/panel/` (Blade retired on stage, side-by-side no longer possible).
- Soak: 09-23→09-26 ~69h 0 5xx; 09-26→09-28 Go logs lost to redeploys; since today's deploy 1450 requests, 0 5xx, no error lines; proxy error log (stage host) no upstream failures. Rollback tested 09-23. Evidence: `docs/go-migration/stage-rollout-log.md` § Staging smoke + soak; 133 `stage-*` screenshots.
- Bugs → T-M2-32 (old reminder form buttons overflow, Latin digits in fa slide a11y text). Monitoring gaps noted on T-M2-26. Stage has no seeded content.

## T-M2-32 — Small web fixes from staging smoke
- Reminder form buttons: scoped `.rem-form-submit/.rem-form-cancel` flex rules (global `.btn` unchanged).
- Slide a11y text `{n, number}`/`{total, number}` in welcome + banners (fa/en), seed + i18n goldens synced.
- Verify green: 592 frontend tests, go test ./..., contract 986 pass / 236 allow-listed. Visual check of the reminder form pending (stage).

## T-M2-33 — Deploy ordering so SSG pages bake the new messages
- `deploy-stage.sh`: backend-go built + `up --wait` (goose) + proxy reload BEFORE the other images; frontend build gets `BUILD_REV=<sha>-<utc>` so only its `npm run build` layer re-runs. `deploy.sh`: same ordering, inert until prod compose lists `backend-go` (T-M2-26). Skills documented.
- Stage deployed @ 5760c4f: frontend build step not cached; prerendered `/fa/welcome` shows «اسلاید ۱ از ۳» (0 Latin). Prod containers unchanged. Evidence in `docs/go-migration/stage-rollout-log.md`.
- Note: first prod frontend build after this Dockerfile change re-runs `npm run build` once.

## T-M3-10 — Care reminders design fidelity fixes (v13 audit)
- 20 of 27 audit rows fixed (all 4 high + 12 med): appointment kind cards, detail hero (time/date tile), medication form cards/labels/slot buttons, filled fields, chips, stepper, dose strip width, cancel in rose tokens, square prep checkboxes, notes/location shown, Persian digits. New `.cfm-*` / `.apd-*` classes, no new colour tokens.
- 7 won't-fix with reasons in `docs/care-reminders/design-audit.md` (Resolution column). Follow-up L-6 (home card placement / duplicate visit) folded into T-M5-11 and T-M7-18.
- Verify: 598 frontend tests, style/dark gates, go test, contract green; check-colors OK. Screenshots re-taken in the audit folder.

## T-M2-34 — Backend fixes from the M3-M7 Go review
- Pregnancy v2 calendar/today keep bell-off appointments (only cancelled excluded); v1 log alert hook best-effort; day-log Save in one transaction, 422 before pregnancy start; HEAD alerts read-only; `weight_missing_week` from weekday 5; legacy medication switch-only PUT; medication subtitle in request locale; `DELETE /account` deletes tokens + user atomically; care-item delete guard ignores cancelled; 00005 Down keeps admin edits.
- Deviations: D-24 (no invented LMP on `POST /profile`), D-25 (account tokens deleted), D-26 (`daily_card` fertility level = v1.1). Allowlists `profile.yaml`, `cycle.yaml`, `cycle-sweep.yaml`. OpenAPI + admin-api doc updated.
- Verify: go vet/test/lint clean, full `make test-int` green, contract 986 pass / 488 allow-listed.
- Open: Today unread badge misses computed alerts until Alerts is opened (needs routes wiring); admin can't edit `from_weekday`; care_item_key delete-guard JSON scan; contract case description for `create_profile_empty_body`; prod (Laravel) unchanged until cutover.

## T-M4-12 — Checkups design fidelity fixes (v14 audit)
- 38 of 42 audit rows fixed (+ D3 by orchestrator: MarkDone sheet `half`); B10 kept (T-M7-17 separator), B13 and D7 need design decisions. Known 5a–5d fixed (attachment chip in row, hero icon per type, admin-web locale digits, self-exam «به‌زودی»).
- Backend: overdue cycle-based items now in `overdue` (monthly self-exam window stays `this_month`); self-exam status follows the next window's lead days; highlight order; migration `00007_checkups_catalog_icons.sql` (guarded data update, down). OpenAPI descriptions updated.
- Frontend: `entities/checkup/model/display.ts`, list/detail/markdone/history/self-exam restyled per design (gradient removed from heroes), `.ck-*` classes.
- Verify: 613 frontend tests, gates green, go test + test-int (checkups, migrations, admin) green, contract 986 pass, admin-web 77 tests; check-colors OK.

## T-M7-18 — Pregnancy v2 design fidelity fixes (v2 audit)
- 31 of 37 rows fixed, 4 partly (A1 welcome/dating/history copy still bundled; E2 source note not admin-editable; F4 «از امروز» in seed; D1 contrast pair not in lint:dark), B2 (week numbering kept = completed+1, now shows «۸ هفته و ۳ روز» under «هفتهٔ ۹») and C3 (content) won't fix.
- API (Go only, OpenAPI updated): carousel title = pregnancy age; dating-preview `copy` (admin result texts) and year-less `range_label`; calendar visits `age_label`; alerts `fact_date`. `--success-fill` token; alerts actions per level (no gradient per card); calendar «حدود …»; `Intl` moved into `shared/lib/date`; today reminders card under the hero, duplicate visit hidden.
- This commit also carries the T-M4-12 hunks of shared files (globals.css `.ck-*`, i18n goldens checkups keys).
- Verify: 613 frontend tests, gates, go test/lint, contract 986 pass, pregnancy + alerts int tests green; check-colors OK.
- Open → T-M7-20: setup-copy route, admin-editable source note, week_entered seed text, lint:dark pair, 9b redirect, ultrasound NumberField digits.

## T-M7-20 — Pregnancy leftovers
- `GET /api/v1/pregnancy/v2/setup-copy` (admin `pregnancy_setup` texts, locale fallback) read by all Setup steps; contract group `pregnancy-v2` (goldens from Go).
- Calendar source note admin-editable (`pregnancy_setup/calendar_note`), migration `00008_pregnancy_v2_copy.sql` (guarded: week_entered text without «از امروز», `from_weekday: 5`, calendar_note seed) + Laravel data twin `2026_09_29_000001_seed_pregnancy_calendar_note.php` (schema-diff green).
- Today evaluates calendar alert rules first (badge correct without opening Alerts); `from_weekday` editable in admin-web (labels added); `--success-fill` pair in lint:dark; `?return_to=` allow-list (calendar) for the appointment form, incl. the calendar «ویزیت جدید» links; `LocaleNumberField` for ultrasound week/day.
- Verify: go vet/test/lint, contract 991 pass / 488 allow-listed, schema-diff OK, frontend 627 tests + gates, admin-web 79 tests.

## T-M5-11 — Fertility design fidelity fixes (v19 audit) and TTC home
- 26 of 27 audit rows fixed (#18 evidence titles are backend copy → T-M5-12). Home reads top-level `fertility_level` (same word as Log). TTC home for `trying`: ring «تخمک‌گذاری تا N روز», phase pills, «شانس بارداری امروز» card, LH tip card; tiles no longer pop in late (profile in the home boot gate, fixed-height skeleton). Insights history strips per cycle, window headline hierarchy, amber dashed window days + today marker; BBT/Log restyled; reminders card under the hero (L-6).
- New: `entities/fertility/ui/BbtNumber.tsx`, `widgets/fertility-tiles/{model/ttc.ts,ui/FertilityChanceCard.tsx,ui/LhTipCard.tsx,ui/TtcPhasePills.tsx}`.
- Verify: 627 frontend tests + gates, go test, contract 991 pass; check-colors OK.
- Open → T-M5-12: evidence titles copy; `/profile` latency (home now waits on it; ~5 s seen on stage); per-cycle period length for history strips; home vs `/fertility/bbt` window off-by-one.

## T-M5-12 — Fertility follow-ups
- Evidence copy per design (count/fact titles, no «(±۰)»); `/fertility/insights` history `period_days`; new contract group `fertility` (goldens from Go; `fertility_logs` added to the fixture dump).
- `/profile` is fast locally (p95 ~6 ms with 2 years of data) — the stage ~5 s wasn't reproducible; home no longer blocks on it for returning users via a boolean `ritme_home_ttc` hint (cleared on session end). Numbers in `docs/fertility-ttc/README.md`.
- Home fertile window now from `cycle_view.anchors` per task.md §19 (= `/fertility/bbt`, days 10–15 on the day-16 case).
- Verify: go test, fertility int tests, contract 998 pass; frontend 657 tests + gates. OpenAPI hunks for `/fertility/insights` land with the T-M7-19 commit (shared file).
- Open: legacy O−5…O+1 window still in calendar `is_fertile_window` and home `CycleTimelineBar`; stage `/profile` latency to re-check after deploy.

## T-M7-19 — Security fixes from the M3-M7 audit
- (high) All on-device files (`ritme-local-files` IndexedDB) cleared on session end; orphaned checkup attachments pruned (`pruneCheckupAttachments`, only when the server list loads fully).
- (med) Attachment allow-list jpeg/png/webp/heic/heif/pdf, exact `matchesAccept`; `openLocalFile` opens only allow-listed MIME, everything else downloads. Frontend CSP now enforced (no new directives; verified with a prod build + headless run).
- (med) No health data in URLs: appointment prefill via `shared/lib/handoff` (`?prefill=<id>`, memory + sessionStorage, cleared on save/session end); `return_to` allow-list and topic prefill kept.
- (low) ICS escaping (`;`, bare CR, control chars). (low) Per-user caps (100 medications, 100 upcoming / 1000 total appointments, 50 custom checkups → 422 `limit_reached`) and a 60/min per-user write throttle on care/checkups/fertility writes (`ratelimit.Named("writes")`).
- Verify: 657 frontend tests, build OK, go vet/test/lint 0, contract 998 pass. This commit also carries the T-M5-12 `/fertility/insights` OpenAPI hunks.
- Open → T-M7-21: prune on custom-checkup delete, show the 422 cap message in forms; throttle not on Laravel-parity routes (X-RateLimit headers pinned in goldens); cap check not atomic.

## T-M7-21 — Security follow-ups
- Custom checkup delete prunes on-device attachments right away (`afterCustomCheckupDeleted`, fire-and-forget).
- `getApiErrorCode` / `getApiLimitMessage` in `shared/api`; medication, appointment and custom-checkup forms show the server's localized 422 `limit_reached` message.
- Verify: 680 frontend tests + gates.

## T-M5-13 — One fertile window everywhere
- Shared `cycleDayMarkerAt` / `fertileWindowDays` (entities/cycle): current + predicted cycles take the window and ovulation from the anchored schedule (task.md §19), past cycles read the legacy calc with the same rule (O+1 → luteal); period always wins.
- Calendar grid/intensity/aria/day sheet, home week strip, ring and `CycleTimelineBar` all use it. Verified on one TTC user: calendar, ring, timeline, home text, insights and BBT all show cycle days 10–15.
- Verify: 680 frontend tests + gates. Open: timeline band now drawn above the progress fill (design check); legacy `is_fertile_window` stays in the API for Laravel parity.

## T-M7-22 — Fixes from stage regression C (pregnancy, admin-web)
- admin-web: `.table-wrap` relative (no sideways scroll at 390 px), `useNumber()` for fa digits everywhere, week grid merges structured details (42/42), `/pregnancy-weeks/{n}` opens details; Go admin `form.JSON` stores raw UTF-8 (search finds both forms).
- Web: Profile «حذف حساب» restored (`DeleteAccountConfirm`, profile route ships `account` namespace); onboarding week fields `LocaleNumberField` (1–42); Log spotting chip uses the Alerts level colour; bell-off appointments show no «یادآور …» (calendar `remind_before: null`, home card via `remind-chip`).
- Go: Today `unread_alerts` = what Alerts shows (v2 rows, 7-day window, `v2.AlertsSince`); `week_entered` refreshes after re-dating (keeps read/ack).
- Verify: admin-web 85 tests, frontend gates + tests (after the message-scopes line), go test + int tests green. OpenAPI hunks land with T-M2-35.
- Open → T-M7-23: data export row missing from Profile (§11), `/care/today` should carry `is_active` (avoid per-appointment fetch), admin `type=number` Latin digits, PWA install banner above dialogs.

## T-M2-35 — Fixes from stage regression A+B
- Calendar day sheet reads `/fertility/days/{date}` (v1.1 level, 6 labels); PMS 3 days everywhere (`PMS_WINDOW_DAYS`); unlogged day via `GET /health-logs?from_date=D&to_date=D` (no 404 noise); chance-card copy follows days to ovulation; Log title for past dates; insights caption spans both months; lavender page on Log/BBT/Insights; ovulation row/tick turquoise.
- Go: D-27 (post-ovulation card note), D-28 (`/messages/daily` ends the window on ovulation day), D-29 (no re-enabling a cancelled appointment's reminder → 422); allowlists `cycle-sweep.yaml`, `messages.yaml`.
- Legacy reminders sheet: cancelled rows disabled, «، » separators, «ساعت ۸:۰۰», alternate-day label; checkup detail «ثبت نوبت» uses the prefill handoff; pregnant default medication duration «تا پایان بارداری»; medication form + fertility log show a localized 429 message.
- Verify: 696 frontend tests + gates, go test/lint, int tests, contract 998 pass / 714 allow-listed. This commit also carries the T-M7-22 OpenAPI hunks.
- Open → T-M7-23: Go 429 body localized; 429 message on appointment/checkup forms; move the duplicated medication schedule rule into `entities/care-reminder`.

## T-M7-23 — Last polish
- Profile «گرفتن خروجی داده‌ها» restored (`GET /profile/export` → `ritme-data-export.json`).
- `/care/today` `next_appointment.is_active`; home card no longer fetches each appointment.
- Write throttle 429 localized (`care.TooManyWrites`, `error_code: too_many_requests`, `retry_after`; `ratelimit.NamedWith`), OpenAPI `WriteThrottled` on 18 writes; appointment, custom checkup and MarkDone forms show it. Laravel-parity 429s unchanged.
- `medicationSchedule` moved to `entities/care-reminder`; admin-web numeric inputs show locale digits (text + inputMode); `--z-*` stacking tokens, PWA install banner below dialogs/sheets and hidden while a modal is open.
- Verify: frontend 699 tests + gates, admin-web 91 tests, go test/lint, care+http int tests, contract 998 pass; check-colors OK.
- Minor open: `.pwa-toast` (soft update) still above sheets; admin numeric min/max now enforced by the server (422).

## T-M2-36 — Daily tips on /cycle/today use the display fertile window (F-1)
- Shared `legacy.DisplayWindow` (§19) used by `/messages/daily` (D-28) and new `Engine.CalculateForDisplay` behind `/cycle/today` + `/cycle/date` (D-30): O+1 = luteal, not fertile, no fertility flag; tips looked up as `early_luteal` (product decision 2026-09-29); scores/subphase unchanged; `/cycle/month`, `/home`, calendar keep the legacy calc. Cycle engine cache schema → 2.
- Verify: go test/lint, cycle + messages int tests, contract 998 pass / 1994 allow-listed (all D-30 O+1 days), frontend 699 tests.

## T-M8-01 — Persian copy style guide, glossary and namespace inventory
- Shipped `docs/copy-fa/STYLE.md` (register, per-kind rules, error templates, orthography, medical-safety
  rules, sensitive-domain tone, ICU safety, 20 before/after pairs), `GLOSSARY.md`, `INVENTORY.md` (70
  namespaces → batches T-M8-03…13; 48 written / 11 colloquial / 8 mixed / 3 no prose today).
- User approved 2026-10-18: written-but-warm «تو»; scope frontend only; «چرخه» replaces «سیکل»; «پریود»
  everywhere, «قاعدگی» only in fixed medical phrases (LMP, PMS/PMDD); ezafe after ه = «هٔ».
- Open: checker baseline counts to be appended to INVENTORY.md after T-M8-02.

## T-M8-02 — copy-fa tooling: ICU-safe checker, value-only cleanup and backend sync
- `frontend/scripts/copy-fa/`: `check.mjs` (structure gate vs git base + en, ICU signature via
  @formatjs parser, sync gate vs backend-go, lint errors/warnings from STYLE + GLOSSARY), `clean.mjs`
  (runs the skill's `persian_cleanup.py --edit` on values only; ICU/tags/Latin codes protected; byte-stable
  formatting; skips `fix_ezafe` and `fix_zwnj_compound_verbs` by default), `sync.mjs`, 25 node tests.
  npm: `copy:check`, `copy:clean`, `copy:sync`. `clean.mjs` needs `--cleanup <path>` or env `PERSIAN_CLEANUP`
  (the skill's `scripts/persian_cleanup.py`).
- Main-session fix: a key added since the git base is not an error when en has it (uncommitted feature work:
  `common.moduleOff.*`, `onboarding.flow.conditions.medsOnly`).
- En dash allowed between numbers/ICU args (STYLE §3.5 amended). `index.js` exists only so
  `node --test scripts/copy-fa` works on Node 24.
- Baseline appended to `docs/copy-fa/INVENTORY.md`: 0 structure/sync, 12 lint errors, 843 warnings.
- Known risk: `convert_digits` turns `FIGO 2018` into `FIGO ۲۰۱۸` — reviewers check Latin+number pairs.

## T-M8-03 — Copy batch 1 — core, shell, auth, onboarding, profile (pilot)
- 17 namespaces rewritten by a sonnet subagent, full diff reviewed by the main session; 123 values changed
  (profile-info 24, onboarding 20, welcome 14, pwa 9, search 9 …; web-shell and banners were already right).
  Reviewer fixes: profile-info «دوره‌های قاعدگی‌ات» had become «چرخه‌ات» (meaning shift → «پریودهایت»),
  auth `errors.tooMany`, notifications empty body; «بارگیری» → «بارگذاری».
- Lessons folded into STYLE.md/GLOSSARY.md: «بارگذاری» for loading, error-title variant, named-action error
  template, asides in parentheses, «همدم» = Companion feature name, partner = «شریکت» («پارتنر» rejected),
  «دوره‌ها» (courses) / «ویزیت» / «سوابق پریود» allowed, «حذف همه» / «حذف متن» for clear actions.
- Checker: dropped the «داری؟» false positive; allowed phrases now mask longest-first.
- Go: `go test ./internal/i18n -run MatchesLaravelGoldens -update` regenerates `testdata/messages_{fa,en,ar}.json`
  from the embedded seed. Parallel batches skip that test; the main session regenerates once per round.
- Commit stages value changes only (`git update-index` from HEAD + new values): the uncommitted foreign keys
  `common.moduleOff.*` and `onboarding.flow.conditions.medsOnly` stay in the working tree, uncommitted.
- Open (user decision pending): none blocking. Left unchanged for T-M8-14: hardcoded version «۱٫۰٫۰».
- Verify: check.mjs 17 ns → 0 errors (1 justified warn before the mask fix, 0 after); vitest 1899/1899;
  `go test ./internal/i18n/...` ok.

## T-M8-04 — Copy batch 2 — home, cycle, calendar, phase details, analysis
- sonnet rewrite, full diff read by the main session: home 71, cycle 40, calendar 20, phase-details 4,
  analysis 114 values. Numbers, ranges, hedges, FIGO thresholds unchanged.
- Reviewer fixes: «نشون بده» → «نشان بده»، «حال‌وهوا» → «خلق‌وخو»، «بازهٔ PMS» → «روزهای PMS»، «FIGO ۲۰۱۸»
  reverted to «FIGO 2018» (cleaner converts years next to Latin codes — STYLE §3.4 now says so),
  ezafe «چرخهٔ {length} روزه»، «دمای پایهٔ».
- Open: `{from} – {to}` became «{from} تا {to}» (fine for RTL); visual check in T-M8-15.

## T-M8-05 — Copy batch 3 — daily logging (log, sheets, taxonomy, voice)
- sonnet rewrite, reviewed: log 11, log-period 9, log-sheet 20, log-customize 4, log-taxonomy 7 (chips left
  as they were; only ezafe/«پس از زایمان»), voice-log 12.
- Reviewer fix: voice card title «بگو امروز چطور بودی» (tense shift) → «بگو امروز حالت چطور است».
  Glossary: «خونریزی» canonical (46 vs 11), «خون‌ریزی» rejected.

## T-M8-06 — Copy batch 4 — fertility, pregnancy, IVF, pregnancy loss
- opus rewrite (most sensitive batch), every warning/alert string checked by the main session: fertility 49,
  pregnancy 31, pregnancy-v2 55, pregnancy-tools 7, analysis-pregnancy 23, ivf 89, loss 16 values.
  No number, warning-sign list or urgency changed; loss stays gentle, IVF never promises outcomes.
- IVF: «دوره/سیکل درمان» → «چرخهٔ درمان»; blood beta test = «آزمایش» (home tests stay «تست»);
  «سونو» → «سونوگرافی». Test `fertility-insights/model/view.test.ts` follows the new «چرخهٔ» copy.
- Reviewer fix: «IOM ۲۰۰۹» reverted to «IOM 2009».
- Left for the user / content owner (fa≠en, not resolved on purpose): `ivf.tww.danger.body` («نفخ شدید و
  سریع» vs en "severe or fast-growing bloating"; «کم شدن ادرار» vs "passing much less urine"),
  `pregnancy.log.criticalHint` («حتماً» vs en "right away").
- QA in T-M8-15: `ivf.actions.scan` tile grew to «ثبت نتیجهٔ سونوگرافی» — check overflow.
- i18n fixtures regenerated with `-update` for the whole round (T-M8-04/05/06) in this commit.

## T-M8-07 — Copy batch 5 — postpartum, baby log, children, companions, pelvic floor
- sonnet rewrite, full diff reviewed: postpartum 22, analysis-postpartum 18, baby-log 8, children 23,
  companions 23, companion-home 16, pelvic 10 values. Crisis lines (۱۱۵/۱۲۳/۱۴۸۰), EPDS rule urgency and
  bleeding warnings unchanged. «پارتنر» → «شریک»; «بعد از زایمان» → «پس از زایمان» as the mode name.
- STYLE §1.1: text addressed to the couple together keeps plural «شما» (`companions.family.*`).
- Open: `postpartum.recovery.colorsShort.pink_brown` «روشن» vs label «صورتی/قهوه‌ای» (fa≈en, left).

## T-M8-08 — Copy batch 6 — health record, export, labs, home lab
- opus rewrite, full diff reviewed: record 56, record-export 23, health-record 24, labs 32, home-lab 9 values.
  No range, unit, verdict or disclaimer meaning changed; emergency-card and data-sharing item lists intact.
  Doctor/medical-staff pages keep «شما» (`record.doctor.*`, `record.publicEmergency.*`).
- Open for T-M8-15: pre-existing fa/en ICU drift (`labs.verify.lowConfidence`, `labs.result.attentionTitle`
  use `{n}` in fa vs `{count, plural}` in en; `labs.upload.pageCount` `{n}` vs `#`) — check in code.
  «سوابق و اسناد» section name kept (glossary warns; it is the documents sub-section).

## T-M8-09 — Copy batch 7 — vitals, checkups, care reminders, conditions
- sonnet rewrite, full diff reviewed: vitals 22, checkups 10, care 16, reminders 12, conditions 26 values.
  Thresholds (۱۸۰/۱۲۰), emergency numbers and alert urgency unchanged. Reviewer: ezafe «دربارهٔ» ×2.
- Glossary: «نوبت» also = medication dose slot; «دوره‌ای» (periodic) allowed.
- i18n fixtures regenerated for the T-M8-07..09 round in this commit.

## T-M8-10 — Copy batch 8 — menopause, teen, contraception
- opus rewrite, full diff reviewed: menopause 49, teen 18, contraception 20 values. No fertility wording in
  teen (`no_fertility_copy`); post-menopause bleeding alert and every «مشورت با پزشک» note unchanged
  (contraception has no efficacy numbers in copy). Load-error titles lose the final period (rendered as titles).
- Open: `teen/ui/ParentViewCard.tsx:14` comment says «فقط دیدنی» (comment only).

## T-M8-11 — Copy batch 9 — shop, web shop, insurance, Plus
- sonnet rewrite, full diff reviewed: shop 19, web-shop 5, insurance 47, plus 13 values. No price, deadline,
  COD/return/coverage term changed; no new marketing claim. «بزن بریم» → «شروع کنیم»; «٪{n}» → «{n}٪».
- Open (fa≠en, left): `shop.order.notFoundBody` («سفارش دیگری» vs en "someone else's order").

## T-M8-12 — Copy batch 10 — services, city services, doctors, directory, business
- sonnet rewrite, full diff reviewed: services 3, city-services 6, city-join 41, doctors 13, web-business 24,
  web-directory 0 values. Provider-facing `city-join` + `web-business` now «شما» (formal-but-human); the rest «تو».
  Reviewer fixes: ambiguous «مادرهای نزدیک شما را پیدا کنند» → «مادرهای نزدیکتان شما را…» ×2,
  «دیده شدن برای…» → «دیده شدن نزد مادرهای نزدیک».
- QA in T-M8-15: «دوباره امتحان کنید»/«نگه دارید» button widths in city-join.
- i18n fixtures regenerated for the T-M8-10..12 round in this commit.

## T-M8-13 — Copy batch 11 — me, assistant, learn, articles, todo
- sonnet rewrite, full diff reviewed: me 68, assistant 21, learn 23, articles 11, todo 4 values. Assistant
  disclaimers, privacy/data-deletion consequences unchanged.
- Reviewer: offline lesson downloads keep «دانلود» (subagent had switched them to «دریافت»; glossary row added:
  «دریافت» only for exporting data/PDF); `assistant.history.exportError` → «فایل PDF ساخته نشد».
- Open: `me.mode.loss.*` «متأسفیم که این اتفاق افتاد» — near the STYLE §4.1 cliché list, left for the content owner.

## T-M8-14 — Hardcoded Persian strings in frontend code
- Inventory (`grep -rnE '[؀-ۿ]' src … | grep -v tests/comments/date`): 332 → 330 lines; what is left is
  dev-only `/dev/ui-kit` (146), comments inside JSX (~77), `.join('، ')` separators (26), digit tables/input
  normalisers (50), static `app/manifest.ts` (10), `layout.tsx` metadata (4), endonym/font-probe data (3).
- Fixed: `profile-info.about.sections.2.body` now takes `{version}` (real `APP_VERSION`, locale digits; the
  fallback in `InfoSheet.tsx` fills it since `t.raw` skips ICU), en + fa + mirrors; manifest copy (no em dash,
  ezafe); `NavBack` loses the untranslated «برگرد» fallback (`label` required, component unused);
  doctor review avatar fallback «؟» → «?».
- Not fixed (file has someone else's uncommitted edits): `app/[locale]/layout.tsx:77` description
  «ریتمی — همراه سلامت زنان» → should be «ریتمی، همراه سلامت زنان».
- Follow-up idea: 26 `.join('، ')` also use the Persian comma in en — a locale-aware `joinList` helper.
- i18n fixtures regenerated for T-M8-13/14 in this commit.

## T-M8-15 — M8 wrap-up (sweep, visual QA, stage)
- Done: cross-namespace sweep (commit 4829926f). `check.mjs --all` → 0 errors; warnings 843 (baseline) → 45,
  all legitimate senses (دوره = course/span, «تست» for home tests, «سقط» in clinical lists, «سوابق» in prose,
  «دکتر» before a name, FIGO/IOM years). Glossary: جست‌وجو، پیش‌رو added; لغو/یادآوری no longer rejected.
- On hold by user decision (2026-10-18): visual QA (docs/copy-fa/QA.md) and deploy to stage.
- Still open for QA: button widths that grew (`ivf.actions.scan` «ثبت نتیجهٔ سونوگرافی»، city-join
  «دوباره امتحان کنید»، plus «شروع کنیم»، shop «اشتراک‌گذاری با خانواده»); `{from} تا {to}` ranges;
  `layout.tsx:77` description em dash (file has other uncommitted work); pre-existing fa/en ICU drift in
  `labs`; fa≠en items listed under T-M8-06/08/11 for the content owner. Production runs Laravel until
  T-M2-26 — prod copy needs `php artisan translations:import` there or waits for the Go cutover.

- QA (2026-10-18): local stack (contract fixtures + goose 00002…00094), Playwright 390×844, 25 screens × light/dark →
  0 horizontal scroll, 0 clipped button/chip/tab/heading text. Report + 50 screenshots: `docs/copy-fa/QA.md`.
- verify-all on a clean worktree of HEAD (excludes the uncommitted app-modules work): go vet/test/golangci-lint
  green after re-recording `internal/analysis` goldens (text only, commit 5d07deee); frontend typecheck, lint,
  fsd:lint, lint:styles, lint:dark, 1894 tests green; `npm run build` compiled + prerendered 348 pages, then hit
  ENOSPC copying standalone output (local disk 100% full — not a code issue).
- Stage deployed from `stage` @ 5d07deee: every deploy-stage.sh assertion ok; manifest and
  `/api/v1/languages/fa/messages` serve the new copy (home.title «به ریتمی خوش آمدی»، pwa.softTitle «نسخهٔ تازه‌ای
  آماده است»، loss.care.moodQuestion «امروز حالت چطور است؟»).
- Still open (not M8 blockers): `layout.tsx:77` em dash (file has uncommitted app-modules work); IVF scan tile and
  city-join buttons not reachable with fixture personas; fa≠en items for the content owner (T-M8-06/08/11);
  production still on Laravel — needs `php artisan translations:import` or the T-M2-26 cutover.

## T-M4-10 / T-M7-15 — clinical review package (2026-10-18)
- Built a shareable review doc (Claude Docs) «بازبینی پزشکی محتوای ریتمی»:
  https://claude.ai/code/artifact/80858d2d-8499-4c0e-876b-f4f5245193be — 6 checkup catalog rows (intervals, ages,
  cycle days, remind lead, pregnancy hiding, user-facing why/prep copy), 5 pregnancy care items (week windows,
  prep), 8 alert rules (trigger, level, advice), 42 pregnancy weeks (size, length/weight, heart rate, tasks,
  warning signs), each with a «نظر بازبین» dropdown (تأیید / اصلاح لازم / حذف), open questions (spotting rule vs
  info box, call-doctor number, LMP ±5 vs ±3, IVF danger fa≠en, critical-hint urgency fa≠en) and a sign-off block
  (reviewer, specialty + medical council no., date, sources).
- Values taken from the goose seeds (00003 checkups, 00005/00008 pregnancy v2) — admin edits on stage/prod may differ.
- Not signed off: tasks stay in_progress until the reviewer returns the doc; then apply corrections to
  admin/seeds, set reviewer_name/reviewed_at, close both tasks.
- Noticed: DB-seeded copy (alert texts, care prep, week tasks/warnings, checkup prep, week tips) is still
  colloquial («رو»، «می‌شه») and week tips use «شما» — outside M8's frontend-only scope; candidate follow-up.

## T-M4-10 / T-M7-15 — closed (2026-10-21)
- Closed on the user's word: staging e2e shipped, clinical review package in c45c06b8. No production work; content sign-off remains an open review item.

## T-M9-01 — Engine v6 decisions and deviation rows (user OK)
- User accepted every E/I/X recommended default on 2026-10-21 (two groups: visible product changes, technical defaults), no changes. Recorded in `docs/engine-v6/DECISIONS.md` (approval line + `Status: accepted`).
- Code refs re-checked: `profile/service.go:463` syncOnboardingPeriodLog, `analysis/cycle.go:43` markOutlier, `home/sections.go` 21–35 constants still present — defaults hold.
- `docs/go-migration/deviations.md`: rows D-120…D-139 added (`accepted — engine v6, not shipped yet`); D-30 → superseded by D-121/D-122, D-55 → superseded by D-124.
- No task scope changed.

## T-M9-03 — Quick fixes: pregnancy_week fact, cycle cache key, language registry TTL
- `messages/v2.go`: `pregnancy_week` fact now comes from `pregv2.Resolve(p, today).CurrentWeek()` (LMP, ultrasound or manual dating) — same number as the v2 pregnancy screens and the admin rule preview; was LMP-only and could go negative. No trimester fact exists, none added.
- `cycle/service`: cache key includes `clinicalconfig.Fingerprint` of the engine group (`engine.*`, `cycle.figo.*`) for From/To/Today; engine cache schema 3 → 4.
- `i18n/registry.go`: language list cached in Redis with `CacheTTL = 10m` (was no TTL); `Flush` unchanged.
- Tests: `messages/v2_test.go`, `v2_int_test.go`, `cycle/service/cachekey_test.go`, `i18n/registry_ttl_test.go`; `registry_int_test.go` updated (asserted the old no-TTL behaviour).
- Open: `content.RegistryTTL` middleware and `languageRegistryTTL` in `internal/http/routes_content.go` are now redundant (harmless, `EXPIRE NX`); remove in a later touch of routes_content.go.

## T-M9-40 — Frontend: tolerant cycle schema for v6 payloads
- `entities/cycle` schema: v6 fields added tolerant (`future_cycles` per item → `[]`, `clinical_flags`, `today_ui`, `limits` nullable + catch, `suggest_pregnancy_test` default false, `cycle_regularity`, `ovulation_source`); nullable predictions / per-day ovulation / effective lengths; value layers default empty; month days parsed one by one; view enums map unknown strings to `'unknown'`; daily card `fertility_level` accepts `peak`/`unknown` (bad value → `unknown`).
- Schedule: anchors win over `predictions`; null ovulation or suppression → no fertile window/marker; new `CycleSchedule.hasOvulation`; `isFertilitySuppressed(view)` exported.
- Tests: `schema.v6.test.ts` (§2.35 example, Laravel, pill, missing layers, bad month day, unknown phase), schedule v6 block, `suppression.test.ts`. 1921 tests green.
- Open (for T-M9-42…47): `schedule.ovulation` stays a non-null placeholder when `hasOvulation` is false — HomePage (`slotFor(schedule.ovulation…)`) and ReadyStep must gate on `hasOvulation`/`isFertilitySuppressed`; `daysUntilOvulation` falls back to `cycleLength-14-cycleDay`. Some prettier-style reflow in schema.ts/index.ts (no prettier config in repo; cosmetic).

## T-M9-41 — Frontend: invalidate every cycle-dependent query after cycle writes
- New `shared/api/invalidation.ts`: `CYCLE_DEPENDENT_ROOTS` (`cycle`, `fertility`, `analysis`, `message` — the real `messageKeys.all`, not `messages` —, `care`, `period-status`, `period-history`) and `invalidateCycleDependents(queryClient)`; test pins the roots to the entity key factories.
- Wired into period start/end/range/edit/delete/reconcile, day-log save, fertility day log, profile update, cycle settings, contraception save + stop, and (main session, once T-M9-40 had landed) `useRecalculateCycle` in `entities/cycle/api/queries.ts`. ModePage untouched (already invalidates everything).
- Open: per-mutation "onSuccess calls the helper" tests not written (test files outside `touches`); `period-status` is now refetched after start/end (harmless), the old race comment in `log-period/api/mutations.ts` is slightly stale; pill log/undo don't invalidate cycle (not cycle inputs).

## T-M10-01 — Engine v6 M10 decisions (user OK)
- User accepted every recommended default on 2026-10-21: M-01…M-19, A-01…A-12 (A-11 as written: free users keep the same engine output and plain confidence wording; Plus gates only the "why this date", BBT confirmation and LH explanation detail views), and user-owned H-03, H-05, H-08, H-12, H-13, H-18. Recorded in `docs/engine-v6/DECISIONS.md` (approval line).
- Advisor rows H-01, H-02, H-04, H-06, H-07, H-09, H-10, H-11, H-14, H-15, H-16, H-17 → `advisor-pending`, safe defaults ship. Sheet: `docs/engine-v6/clinical-signoff.md` (+ the T-M10-16 registry keys that need a signature when changed). The 12 rows were also appended (user choice) to the shared review doc «بازبینی پزشکی محتوای ریتمی» as section «موتور v6: آستانه‌های بالینی»: https://claude.ai/artifact/GsV3aukx4tTZsQJYRPKKpZ
- `docs/go-migration/deviations.md`: D-160…D-183 added (`accepted — engine v6 M10, not shipped yet`); D-184…D-189 stay reserved.
- Code refs re-checked (`messages/rules/rules.go`, `labs/labs.go` LowConfidence 0.7, `checkups/engine/engine.go`, `metrics/metrics.go`) — no default invalidated. No task scope changed.

## T-M9-04 — Period writes: per-user lock, idempotent start, Idempotency-Key (D-131)
- `cycle/periods.Service.tx` takes `SELECT id FROM users WHERE id = ? FOR UPDATE` first (reuses the sqlc `files/store.LockOwner` query), so every overlap/blocker check and write of one user is serialised. MySQL 1062 on the period insert → re-read the same-start row (idempotent 200), else 422 `period_overlap`; never 500.
- New `internal/platform/idempotency`: optional `Idempotency-Key` on `POST /cycle/period/start`, `/cycle/period/end`, `/cycle/period`, `POST /health-logs`, `PUT /fertility/days/{date}`. Redis `ritme-go:idem:{uid}:{sha256(key)}`, `SET NX` pending marker (60 s) → stored status + content type + body for 24 h; replay = byte-identical body + `Idempotent-Replayed: true`; same key + different method/URL/body → 422 `code: idempotency_key_reused`; replay while the first runs waits ≤5 s then 409 `idempotency_key_in_flight`; handler error / 5xx releases the key (4xx returned as errors are not stored, a retry re-runs them); Redis down → processed normally. The key is never logged. Copy in `idempotency/lang/<code>/idempotency.json`.
- Fertility `Save` takes the same per-user lock (read-merge-upsert lost update between concurrent partial saves); health-log createOrFirst path unchanged.
- Tests: `periods/concurrency_int_test.go` (20× same-date start → 1 row all 200; 20× overlapping ranges → no overlap, losers 422; replay byte-identical, reused key 422, Redis down OK, per-user key scope), `periods/duplicate_int_test.go`, `fertility/save_race_int_test.go`, `idempotency` unit (miniredis) + real-Redis int test.
- **Follow-up (frontend, not here):** the offline outbox should send a stable `Idempotency-Key` per queued write (period start/end/store, health-log save, fertility day save).
- Open: `api/openapi.yaml` not updated (outside touches) — document the `Idempotency-Key` header and the 422/409 refusals; the bleeding-log bridge still writes `cycle_histories` without the lock (T-M9-28); a dedicated `LockUser` query in `db/queries/cycle` could replace the `files/store` reuse.
- Main session: OpenAPI (`Idempotency-Key` header, 422 `idempotency_key_reused`, 409 `idempotency_key_in_flight`) not documented yet — `api/openapi.yaml` outside touches; add with the next task that touches it (T-M9-06 or T-M9-31). Also add a `LockUser` query in `db/queries/cycle` later instead of `files/store.LockOwner`.

## T-M11-01 — Engine v6 family decisions (user OK)
- User accepted F-01…F-26 on 2026-10-21 (every recommended default; F-18 kept → T-M11-16 stays queued). Recorded in `docs/engine-v6/DECISIONS.md`.
- New reference `docs/engine-v6/family.md`: levels, domain registry (max / role defaults / mode scope / male-owner rule / TTC switch), templates, carer time-box, legacy → v6 mapping, per-domain member allow-lists + never lists, `check()` order with F-23 and the mode epoch, nudge rules, activity actions / sealing / retention, invite rules, copy keys for owner and member notices.
- `docs/go-migration/deviations.md`: D-190…D-206 added (`accepted — engine v6 M11, not shipped yet`); D-207…D-209 reserved.
- Note: T-M9-02 (v1 privacy hotfix) was still in flight when this was written; family.md's allow-lists are the v6 target and match T-M9-02's never-shared rules (notes, mood, lateness, postpartum ≠ pregnancy). Re-check §4 against T-M9-02's final diff when it lands. No task scope changed.

## T-M9-07 — Observability: Prometheus metrics, query counter, request id, pprof listener
- New `internal/platform/metrics` (registry + Go/process collectors; README §6 names: `http_request_duration_seconds{route,method,status}`, `db_query_duration_seconds{query}`, `db_queries_per_request{route}`, `cache_requests_total{cache,result}`), Fiber middleware labelled by route template (`unmatched` for 404s).
- DB instrumented at the `driver.Connector` level (`platform/db/instrument.go`, `OpenDSN` = `sql.OpenDB(Instrument(...))`): every sqlc/raw/tx call counted per request, query name from `-- name:`; slow query > 200 ms warns (name only, no SQL). Slow request > 500 ms warns with route/status/queries.
- `X-Request-ID` middleware (accept `[A-Za-z0-9-]{1,64}`, else 32-hex), echoed, added to every context log line (`httpx.LogHandler`). Cache GET counter via go-redis hook `TrackGets("cycle-engine:", "cycle_engine")`.
- **New env `METRICS_ADDR`** (default `127.0.0.1:9090`, empty = off): `/metrics` + `/debug/pprof/*` only on that private listener; public app returns 404 (tested). Port busy → logged, API stays up.
- Dependency: `github.com/prometheus/client_golang@v1.24.1` (+ client_model, common, procfs, perks, goautoneg; protobuf 1.36.10 → 1.36.11) via `go get`, no tidy.
- Open: engine/asynq metrics (`engine_build_seconds`, shadow diff, cache verify mismatch) to register on `metrics.Registry()` in later tasks; the hook counts an invalid cached JSON as a hit (direct `Record` in `cycle/cache` would be cleaner); Prometheus in another container needs `METRICS_ADDR=:9090` on the internal network, never via nginx; package-level registry is a global (Deps not extendable in touches) — candidate deviation note.

## T-M9-05 — Pregnancy emergency paths: kick counter, severe BP, vitals in pregnancy, GDM (D-134)
- Clinical registry keys (group `pregnancy`, v1): `pregnancy.bp.high` {140/90, urgent 1} (H-02 advisor-pending), `pregnancy.bp.severe` {160/110}, `pregnancy.gdm.targets` {fasting 95, 1 h 140, 2 h 120}, `pregnancy.kicks` {target 10, window 120 min, from_week 28}, `pregnancy.fetal_movement.manual_from_week` 24 (H-04 advisor-pending), `pregnancy.contractions_511` {5 min, 45 s (H-06 advisor-pending), 60 min}. Thresholds inclusive; for BP either number decides.
- v2: ≥160/110 always urgent (severe copy); 140/90 level from `bp.high.urgent`; GDM → follow_up; kick session without the target inside the window at ≥ 28+0 → urgent (dedupe per session), checked on stop/overview/kick/undo/daily eval; manual reduced/none from week 24. v1: ≥160/110 → emergency.
- Emergency rules (`fetal_movement`, `contractions_511`, `critical_symptom`, `bp_high`) fire with embedded fa/en copy when the admin row is missing or disabled. Vitals in pregnancy use the pregnancy table (H-03) and raise one urgent `alert` modal at the severe tier. Safety scenarios added (severe BP, vitals BP, GDM 2 h, rule without copy).
- Frontend: kick screen shows the urgent fetal_movement alert card (call / ack / see alerts); `lowBody` copy now says contact now / 115. Main session reviewed every threshold and text against H-02…H-06.
- Main session fixed `internal/pregnancy/repo_int_test.go` (165/112 is now the severe tier → 1 emergency + 1 warning).
- Open: analysis card still reads constants (pinned equal to registry v1 by a test); admin alert-row params (bp_high/sugar_high/fetal_movement/contractions_511) are now ignored — admin registry descriptions + OpenAPI should say so; fixed «۱۴۰/۹۰»/«۱۶۰/۱۱۰» inside copy text; a 0-kick tap-and-stop session raises an urgent alert (advisor: minimum duration?); no 1 h vitals context; backend translation seed not resynced for `pregnancy-tools.json`; v1+v2 double rows for weekly-log facts (pre-existing).

## T-M9-02 — Family privacy hotfix: never-shared data and stale pregnancy grants (D-133)
- User OK on the six changes (2026-10-21, this session).
- Members never get mood/EPDS; meds and appointments go out as allow-list projections (`MemberJSON`: no notes, topic, result_note, prep, stage, subtitle, remind_*…) — on section reads, the companion home **and** every `for_user_id` care response (show/store/update/switches; main session wired `medicationOut`/`appointmentOut`). Pregnancy section only in effective mode pregnancy (postpartum gets the neutral cycle view). Member cycle view has no `days_late` or any lateness-derived value (cycle_day/days_to_period/predicted_next null while late).
- `companion.ResetModeScopedGrants(ctx, db, ownerID, reason, at)`: pregnancy grants → none + `grants_changed` audit + owner notice «اشتراک بارداری با همراهت متوقف شد؛ اگر بخواهی، دوباره روشنش کن.»; called on pregnancy activate/onboarding/deactivate (wired via `h.SetFamilyGrants(d.DB)` in routes_pregnancy.go), loss (after commit), postpartum activation (in tx). Shop checklist share only while she is pregnant with a live grant. `Cache-Control: private, no-store` on member responses.
- **Migration** goose `00095_reset_stale_pregnancy_grants.sql` (data only; no Laravel twin; `down` no-op): resets pregnancy grants older than the current pregnancy episode start (GREATEST(onboarding_completed_at|created_at, last loss, last pregnancy_closed_at)) or of owners not pregnant now; audit rows with actor NULL, no notices.
- Frontend: companion parsers drop `daysLate`/ignore `days_late`; no «late» copy on the member side; `cycle.late` key removed (seed resynced, i18n testdata re-recorded).
- Tests: member privacy (mood, free text, postpartum ≠ pregnancy, no lateness, no-store), reset on activate/setup/deactivate/loss/postpartum, stale checklist grant, migration, care for_user_id projection, real-router reset; loss/companion route tests updated to the new behaviour. Contract goldens companion/companion-home re-recorded from Go (`days_late`, member med fields gone; D-133 comments in cases); OpenAPI: `MemberMedication`/`MemberAppointment` schemas, care responses `anyOf`. `make contract ROUTES=companion,companion-home` 24/24; `internal/http` int 565 s ok.
- Open: audit table has no payload column (diff stored in the owner notice `data`); deploy to stage/beta only with the user's OK.

## T-M9-15 — Engine v6 skeleton: types, params, v6 enums, Prepare and effective values
- New pure package `internal/cycle/engine` (no I/O, globals or clock; X-01): `Day` (int32 epoch days, Hinnant civil algorithm), `Opt[T]`, v6 enums with spec spellings (zero = unknown/none; `Confidence` ordered so `min()` works), `WarningSet` (21 incl. `contradictory_anchor_data`, E-02) / `ReasonSet` bitsets with zero-alloc iteration, `Input`/`Status`/`Anchors`/`Metrics`/`ClinicalFlags`/`FutureCycle` (E-05)/`Limits` (E-24)/`TodayUI` (E-17) types, `Params` + `DefaultParams()`, `Prepare(in, p) *State` (periods, contradictions excluded, skips, cycles, effective cycle/period length + source, variability, valid counts). 98.3 % coverage; `BenchmarkPrepare` 60 periods ≈ 3.6 µs / 6 allocs.
- New `internal/cycle/engineparams`: `For(d)`, `Keys()` (13 keys), `Fingerprint(d)`, `SchemaVersion = 1`.
- clinicalconfig (group engine, X-11): `engine.luteal {14, 10, 16}`, `engine.bbt {strict 1, min_readings 10, rise_centi 20}`, `engine.lh_bbt_conflict {2}`, `engine.pregnancy_test {cycle 8, ttc 1}`, `engine.stale_anchor {14}`, `cycle.teen.frequency {21, 45}`.
- Interpretations: E-02 "newer" = later start (ties: later input); only two open periods are a contradiction; declared/manual lengths outside 21..45 / 2..10 ignored.
- Open for T-M9-16+: product constants kept in code Params (OverdueDaysLate 8, PMSDays 3, LateLutealDays 6, regularity 12 months / 3 cycles, FIGO age bands 26/42) — X-11 treats such values as code constants; v1.1 `engineConfigKeys` now also fingerprints v6 keys (harmless, removed in T-M9-51); LH/BBT/tests/mode/delivery are carried but not processed yet (T-M9-17…19).

## T-M9-13 — Config caches with pub/sub invalidation and one query for rule history
- New `internal/platform/configcache`: `Hub`, `Cache[T]`, `Keyed[T]` — lock-free `Get` (atomic snapshot + per-table epoch), reload on local `Changed()`, Redis pub/sub (channel `ritme-go:config-changed`, payload `{"origin","tables"}`) or 60 s TTL; hand-rolled singleflight; stale snapshot served on reload error (5 s backoff); content `Fingerprint(tables…)` for later ETags. Disabled under `go test` unless `SetEnabled(true)` (testdb resets tables between tests). Package-level `Default` (same pattern as clinicalconfig).
- `ListUserMessageEventStats` (one grouped query: last_at + day/week/cycle counts per rule/event) replaces the per-rule history probes in `rules.Select` (50 rules: 1 query instead of 50+; bench ≈ 26 µs/op).
- Snapshots: message rules (`rules.ActiveRows`, now used by `/messages/selected` — main session swapped `internal/messages/v2.go`), message contents per locale, cycle recommendations, app modules (Redis key `ritme-go:app_modules:v1` no longer used), plus plans, challenges catalog (N+1 removed). Admin writers publish after commit via `httpadmin.Publishes(h, tables…)` (content, messages, billing plans, pregnancy alert rules, companion tips, languages).
- Main session wired `configcache.Start(ctx, rdb, logger)` in `cmd/api/main.go` (cross-process invalidation).
- Open → **T-M9-13b** (new task, depends on T-M9-08): `/home` config reads and the language registry onto configcache, query budgets + perf-baseline doc.

## T-M9-16 — Engine v6 core resolve: anchors, end model, caps, phases (§2.11–§2.26)
- `engine.(*State).Resolve(target, today Day) Status` (only new public API). New `layout.go` (one cycle's day map: fertile §2.24, luteal §2.25 with pms > late > post > early/mid, follicular §2.23, fertility_level §2.26; ovulation injected so T-M9-17 can plug BBT/LH) and `resolve.go` (anchor binary search → declared start, end model §2.11, next §2.16, menstrual tiers/caps §2.21, period_expected tiers §2.22, E-10 override, source warnings, `suggest_pregnancy_test`, `grade`). 98.2 % coverage; `BenchmarkResolve` 24 periods ≈ 78 ns, 0 allocs.
- Interpretations: hard cap checked before the warning cap (EPL ≥ 10); no end-missing tiers on a declared onboarding start; open periods inside completed cycles get the §2.21 tiers; > hard cap → all three cap warnings (E-12) + poor/low + `missing_period_end_beyond_hard_cap`; retrospective layout (E-22) also for a declared onboarding start followed by a real start; calendar ovulation = next − `Params.Luteal.Default`; `suggest_pregnancy_test` needs days_late ≥ 1; E-10 covers period_expected 0..MaxLate, then `prediction` (E-11); `prediction_based_output` not emitted yet (E-01: only predicted_reference / future, T-M9-20). Quality fields graded only where tiers force them (full grading T-M9-18).
- v6 vs v1.1 differences (for T-M9-32 shadow classification): luteal priority order; no `prediction_based_output` on ordinary days; cumulative cap warnings; hard cap first; onboarding override covers period_expected 0–14; retrospective completed cycles; anchors only from real periods then `Declared.LastPeriodStart` (no `is_estimated` rows, I-01); new `ovulation_source`, `suggest_pregnancy_test`; luteal from Params; future targets resolve from the current cycle until T-M9-20.
- Open: **`Day(0)` = 1970-01-01 is a valid day**, so a zero `Input{}` has a declared LMP/birth/delivery in 1970 — fix in T-M9-17 (treat these as `Opt`/`NoDay`) before T-M9-24 builds inputs. T-M9-17: BBT/LH ovulation into `layoutFor`, `Anchors.OvulationSource`, `next = O + effective luteal`. T-M9-18: full grading + contradictory/low-history/variability/outlier warnings. T-M9-19: suppression hooks. T-M9-20: future branch, skips, stale anchor.

## T-M9-17 — Engine v6 ovulation sources: BBT 3-over-6, LH, luteal median, conflict
- New pure `internal/cycle/engine/bbt` (`Detect[D ~int32]`, 3-over-6, strict third reading ≥ baseline + `rise_centi`, ≥ `min_readings` and ≥ 9; 0 allocs; 100 % coverage; `FuzzBBTDetect` 1.36 M execs clean). `ovulation.go` (Prepare part 2): per real cycle BBT > LH (first positive + 1, faint ignored) > calendar, `lh_bbt_conflict` when |O_lh − O_bbt| > 2; effective luteal = median of last ≤ 3 BBT-confirmed lengths in [10,16] (skips excluded, LH-only none) else 14; current cycle next = O + effective luteal; calendar estimate stays next − `Luteal.Default`; completed cycles use the detected ovulation (E-22).
- **Day(0) fix:** `Declared.LastPeriodStart`, `Input.BirthDate`, `Input.DeliveryDate` are `Opt[Day]` — a zero `Input{}` means no data. New State API: `Ovulations()`, `LutealLength()`, `BBT()`, `LH()`, type `CycleOvulation`.
- Interpretations: markers belong to [real start, next real start); markers before the first real period ignored; a declared-only onboarding cycle stays on calendar (product question below); source chosen in Prepare, not filtered by today.
- v6 vs v1.1: ovulation from BBT/LH instead of always next − 14, next moves to O + luteal; new BBT rule (v1.1 fertility: first reading ≥ +0.20, 9 readings); `ovulation_source` bbt_confirmed/lh_positive; `lh_bbt_conflict`; completed cycles laid out around detected ovulation.
- Benchmarks: Resolve 0 allocs ≈ 124 ns; Prepare 7 allocs ≈ 3.5 µs; Prepare with 60 periods + 400 BBT + 60 LH ≈ 35 µs (budget 300 µs).
- Open: `Period.End` zero = 1970 trap (loader must set `NoDay`; fixed in T-M9-18 if cheap); T-M9-24 builds dates with `Some(...)`, maps LH faint → not positive, may drop markers after today; T-M9-35 switches `internal/fertility` to `bbt.Detect`; product question — should TTC users with only an onboarding date (no logged period) get marker-based predictions? (today: calendar).

## T-M9-09 — Auth hot path: one JOIN, cached OAuth clients, interpolateParams, pool from env
- Each authed request now runs **1** auth query (`GetAccessTokenWithUser`: the token by jti joined with the user by `sub`) instead of 3. Measured on MariaDB with `metrics.QueryCounter`. The check order is the same as Passport: token revoked → client → user. A miss falls back to the old 3-step path (`verifySlow`) only to pick the exact 401 code. 401 codes, the 365-day lifetime and refresh-session are unchanged.
- New `passport/clients.go`: in-memory cache of OAuth clients.
  - Filled on first use, refreshed in the background every 5 min (stale requests are served from the last snapshot).
  - An unknown client id triggers one shared (single-flight) reload.
  - Tokens are never cached (X-02).
- `platform/db`: `interpolateParams=true` is on in the DSN, forced in `OpenDSN` too. Charset utf8mb4 is safe; the driver refuses unsafe charsets.
  - Integration test: Persian text, ZWNJ, quotes, backslash, NUL, emoji and JSON round-trip byte for byte; DATETIME stays Tehran wall-clock; `Com_stmt_prepare` stays 0.
- Pool size comes from env through `config.DB.Pool`. Invalid values fail at startup.
  - `DB_MAX_OPEN` default 25
  - `DB_MAX_IDLE` default = `DB_MAX_OPEN` (was 10)
  - `DB_CONN_MAX_LIFETIME` default 5m
  - `DB_CONN_MAX_IDLE_TIME` default 2m
- `BenchmarkVerify` ≈ 106 µs/op on M4 (budget 150 µs). Race and lint are clean. The full `make test-int PKG=./...` passes.
- Contract auth: 44/45. `auth/refresh_session_due` already failed on HEAD: the contract tool mints the near-expiry token with `time.Now()` while its request clock is fixed at 2026-09-23.
- Behaviour change: revoking an OAuth **client** reaches the guards within ≤ 5 min. Token revocation is still immediate.
- Open (task **T-M9-09b**):
  - contract clock fix
  - warm the client cache at startup (today each of ~73 guards runs one `ListClients` on first use)
  - `.env.example` DB_* vars
  - perf-baseline numbers
  - deviation entry for the client-revoke delay
  - The query budget is pinned in `passport/verify_int_test.go`, because `internal/querybudget` doesn't exist yet (T-M9-08).

## T-M9-18 — Engine v6 quality layer: regularity, clinical flags, data quality, confidence
- New `engine/quality.go` (Prepare part 3 + `grade`):
  - `cycle_regularity`: 12 months before ref = min(target, today), ≥ 3 cycles, skips out, outliers in, age bands young/adult/late.
  - `clinical_flags`: frequency per E-06 (FIGO; teen 21–45), duration per E-07, `absent` per E-07 plus `long_absence_no_period`.
  - data_quality, ordered insufficient → poor → good → partial (E-02/E-20/E-23).
  - confidence, ordered unknown → low → high → medium. It includes the E-19 BBT exception and a medium cap at age ≥ 45 with `perimenopause_age_range`.
  - All 14 reasons; `requires_user_input` is true for exactly the six warnings.
  - Age is taken at the target (E-27).
- `Period.End == 0` with `Start > 0` now means an open period. This guards against the 1970 trap; the loader should still set `NoDay`.
- New product constant `HighVariabilityDays = 5`.
- Benchmarks: Resolve 147 ns with **0 allocs**; Prepare 3.6 µs; Prepare with markers 24 µs. Engine coverage is 98.6 %.
- Interpretations:
  - Frequency and duration bases use the whole history, the same as the effective lengths.
  - Duration counts closed real periods of any length.
  - An open last period means absent = false. Only pregnancy and postpartum modes are exempt from absent.
  - A contradiction adds `insufficient_anchor_data`, as in v1.1.
  - `prediction_based_output` and `stale_onboarding_anchor` are mapped but have no producer until T-M9-20.
- v6 vs v1.1:
  - New: regularity, clinical_flags, `contradictory_anchor_data`, `long_absence_no_period` and `perimenopause_age_range`.
  - New low conditions: a predicted_reference start, hormonal contraception, and postpartum with fewer than 2 periods.
  - Cap reasons now add up (v1.1 used XOR).
  - Skipped cycles never raise outlier warnings.
  - `prediction_based_output` is no longer added on ordinary prediction days (E-01).
- **Analysis tab (`analysis/cycle.go`) disagrees with v6** (input for T-M9-35):
  1. Its window is the report range, not 12 months before ref.
  2. It drops cycles outside 21–45; v6 counts them, so 18-day cycles read `frequent` in v6 and have no status in the tab.
  3. `markOutlier` drops one lone outlier.
  4. Its median covers every cycle in the range; v6 uses the last ≤ 3.
  5. Its period median uses only closed periods of 2–10 days, including the current one.
  6. Under 18 or over 45 it sets `Applies=false`, and it has no teen band.
  7. It takes age at today.
  8. It spells the empty value `not_enough_data`; v6 uses `insufficient_data`.
  9. It has no `absent` flag and keeps skipped cycles.
  10. It falls back to the v1.1 typical lengths, not the declared length.
- Open items:
  - T-M9-19:
    - `postpartum_cycles_not_returned` plus E-14 `unknown` before the first postpartum period.
    - Hormonal phase suppression.
    - Pregnancy mode confidence `unknown` (E-25).
  - T-M9-20: raise `stale_onboarding_anchor` and `prediction_based_output`; the future k ≥ 2 medium cap; skip anchors.
  - T-M9-21: encode `clinical_flags` and `cycle_regularity` in §2.35.
  - T-M9-24: the loader sets `NoDay` and maps `teen_profiles.age_band` to `TeenBandOnly`.
  - Product question: should frequency and duration use the whole history even for past targets (today), or "as of ref"?

## T-M9-11 — Queue v2: delayed and unique tasks, priorities, scheduler; tickers migrated
- Queue API:
  - `DispatchWith`, `DispatchIn`, `DispatchAt`; options `Unique(ttl)`, `TaskID(id)`, `RunAtStart()`. A dropped duplicate returns `ErrDuplicate`.
  - `Job.Priority` / `Job.OnFailed`.
  - `Dispatch` keeps its old signature.
- Queues: `ritme-go-critical` (weight 6, the OTP SMS job), `ritme-go` (weight 3, the old name, so tasks queued before a deploy still run) and `ritme-go-low` (weight 1).
- Scheduler (`platform/queue/scheduler.go`):
  - Runs `asynq.Scheduler` on the Tehran clock behind the leader key `ritme-go:scheduler:leader` (SET NX PX 30 s, renewed every 10 s, deleted on shutdown).
  - Sync mode runs an in-process cron.
- `auth.Jobs(r, cache, logger)` gives one shared queue per `*fiber.App` (starts on listen, drains on shutdown).
- Seven tickers became scheduled jobs: `partners:sync`, `partners:purge`, `telemed:booking_sweep`, `sharelinks:purge`, `admin_metrics:snapshot`, `files:sweep`, `media:sweep`, `safety:nightly_scenarios`.
  - The old start-up run is kept through `RunAtStart`.
  - Each job has a run-twice idempotency test.
  - The labs, health-record extract and learning-dispatch pollers are unchanged.
- Metrics:
  - `asynq_queue_size{queue,state}` (read through the inspector at scrape)
  - `asynq_archived_total{queue}`
  - `asynq_task_duration_seconds{type,result}`
- No new env vars. Docs: `docs/engine-v6/jobs.md`.
- Behaviour change: migrated jobs now return errors, so asynq retries them with backoff, instead of logging and waiting for the next tick. The files and media sweeps still swallow their errors.
- Open items:
  - T-M9-12: the worker container must mount `STORAGE_PATH` and `MEDIA_STORAGE_PATH` for the sweeps.
  - T-M10-24: the safety nightly result lives in process memory until it is persisted, so the API keeps a start-up run.
  - No live smoke run against the shared test DB.

## T-M9-19 — Engine v6 limiting conditions: contraception, pregnancy, postpartum
- New `engine/limits.go`:
  - `State.Suppression(d) Limits`, 0 allocs. Precedence: pregnancy > postpartum not returned > hormonal.
  - `limits` block: `fertility_suppressed`, `suppression_reason`, `engine_suspended`, `prompt_pregnancy_mode`.
  - Positive pregnancy tests are prepared in Prepare part 4; E-26 prompts in cycle/ttc mode.
- Hormonal contraception:
  - No fertile window or ovulation; fertility level is unknown; BBT and LH are ignored, so the next period stays on the calendar.
  - Period-expected tiers are kept; other days are unknown with resolution `prediction` (confidence low).
- Pregnancy: a suspended status.
  - Warnings and reasons cleared; data_quality `insufficient`; confidence unknown.
  - `cycle_day` only on menstrual days.
  - No next period, ovulation, `days_to_period` or `days_late`.
- Postpartum and breastfeeding: no prediction until 2 real periods after delivery.
  - Confidence is unknown before the first postpartum period and low after it (E-14).
  - New warning `postpartum_cycles_not_returned`.
- `suggest_pregnancy_test` is false under every suppression. The §2.40 rule-9 invariant is checked over 16 fixtures × 2 todays × 141 days.
- Logged period days keep the full §2.21 menstrual tiers under every condition. A declared onboarding start keeps its assumed menstrual days.
- Resolve: 0 allocs, ≈ 220 ns. Prepare: 7 allocs. Coverage 98.9 %.
- Open items:
  - T-M9-20: future_cycles reads `Suppression(today)`. Under hormonal only period dates are kept; under pregnancy or postpartum the list is empty.
  - T-M9-21: encode `limits` (none → null) and the null `days_to_*` / `cycle_day` fields.
  - T-M9-24 loader: fill `PregnancyTests` with positives, `Breastfeeding` (I-04) and `DeliveryDate`. The hormonal-method map is T-M9-25.
  - Product questions:
    - Should postpartum suppression skip targets before the delivery date? Today it applies to them too.
    - Should hormonal suppression start only from a contraception start date? There is no such input today, so it covers the whole history.

## T-M9-09b — Auth follow-ups: contract clock, DB pool env docs, perf numbers, D-140
- `cmd/contract/token.go`: the DB row for the `token_near_expiry` session now takes `expires_at` from the contract clock (`DefaultNow + 10d`). The JWT `exp` still uses real time, because the guard checks it against the real clock. A regression test is in `harness_test.go`. `make contract ROUTES=auth` → **45/45**.
- New `backend-go/.env.example` (the file didn't exist before): the DB section plus `DB_MAX_OPEN`, `DB_MAX_IDLE`, `DB_CONN_MAX_LIFETIME` and `DB_CONN_MAX_IDLE_TIME`, with no secrets.
- `docs/investigations/perf-baseline.md`: new T-M9-09 numbers (auth queries per request 3 → 1, Verify ≈ 106 µs/op, interpolateParams = 1 round-trip per query).
- `docs/go-migration/deviations.md` **D-140**: revoking an OAuth client takes effect within ≤ 5 min; token revocation stays immediate. Status: awaiting user OK.
- Not done: scope item 2, warming the client cache at startup. The auto-mode permission check refused the `guard.go` edit, and I did not try to work around it. It moved to **T-M9-09c**, which needs a user OK first.

## T-M9-10 — HTTP and infra hardening: gzip, timeouts, readiness, Redis eviction, MariaDB config
- `platform/httpx`:
  - `Compress()`: brotli or gzip for JSON responses of 1 KB and up (D-132). Not compressed: HEAD, Range/206/204, streams (SSE, file, media), responses that already have a Content-Encoding, `no-transform`, and any non-JSON type. It applies to every large JSON response, not only the engine GETs; the bytes after decompression are unchanged.
  - `RequestDeadline(5s, LongRunning)`: a deadline on `c.Context()`, with an allow-list for long routes (multipart uploads, `…/export`, assistant, voice logs, document extract, instructor media).
- `GET /ready`: pings the DB and Redis, each within 1 s. 200 `{"status":"ok"}`, or 503 with `failed: [db|redis]`. `/up` stays liveness only. `api healthcheck ready` probes it. Documented in OpenAPI (added by the main session; outside the agent's touches).
- Fiber Read/Write/Idle timeouts, with a 1 h override for media upload and streaming.
- Redis pool and timeouts come from config. New test `TestCacheWritesHaveTTL` finds cache writes with no TTL.
- New env vars:

  | Var | Default |
  |---|---|
  | `HTTP_READ_TIMEOUT` | 15s |
  | `HTTP_WRITE_TIMEOUT` | 120s |
  | `HTTP_IDLE_TIMEOUT` | 60s |
  | `HTTP_REQUEST_TIMEOUT` | 5s |
  | `HTTP_MEDIA_TIMEOUT` | 1h |
  | `REDIS_POOL_SIZE` | 20 |
  | `REDIS_DIAL_TIMEOUT` | 2s |
  | `REDIS_READ_TIMEOUT` | 2s |
  | `REDIS_WRITE_TIMEOUT` | 2s |
  | `MARIADB_BUFFER_POOL_SIZE` (compose only) | 512M |
  | `REDIS_MAXMEMORY` (compose only) | 256mb |

- Stage and beta compose:
  - MariaDB: `--innodb-buffer-pool-size`, plus a read-only `deploy/mariadb/ritme.cnf` (slow log at 0.2 s, performance_schema ON).
  - Redis: `--maxmemory … --maxmemory-policy volatile-lru` with AOF.
- `deploy-stage.sh` / `deploy-beta.sh`: `wait_ready` polls `/ready` up to 30 × 2 s and aborts the deploy if it never passes. Not run.
- **User/server notes:**
  - The next stage or beta deploy recreates the mysql and redis containers (a short DB restart; volumes are kept).
  - The server has 4 GB RAM for three stacks. 512M buffer pool plus performance_schema may be tight, so consider `MARIADB_BUFFER_POOL_SIZE=256M` on stage.
  - `slow-query.log` is never rotated.
  - Under `volatile-lru`, keys without a TTL (queue, state) are never evicted.
- Open items:
  - **T-M9-10b**: most service calls pass the `fiber.Ctx` itself, which has no deadline, so the 5 s cap reaches only about 93 of about 550 call sites. The fix is in the db connector.
  - A request timeout gives 500, never 401.
  - The `.env.*.example` files don't list the new vars yet.

## T-M9-20 — Engine v6 future cycles, stale anchor, skipped cycles, range
- New `engine/anchors.go`:
  - Skip anchors (E-15) are matched in Prepare part 5. A skip must equal the predicted next of its base, or the previous skip + ECL.
  - Unmatched skips go to `State.Diagnostics().IgnoredSkips`.
  - Stale onboarding anchor (§2.41).
- New `engine/future.go`:
  - `(*State).FutureCycles(today) []FutureCycle`: 6 items (E-03/E-04/E-05/E-28). The k≥2 ovulation is start − 14.
  - `(*State).Range(from, to, today) []Status`, at most `MaxRangeDays = 400`.
  - A property test checks `Range == map(Resolve)` over 2 000 histories × 400 days.
- Behaviour:
  - **E-03:** a target after today, on or after today's next start, resolves inside future_cycles[k]. Its source is predicted_reference with the `prediction_based_output` warning, and its confidence is `min(today's, k==1 ? high : medium)`, then the age cap. Example: L 29, start 07-01, today 07-10 → 07-30 is menstrual, FutureK 1, confidence high. A target after future_cycles[6].period_end is unknown.
  - **E-04:** while late, period_start[1] = today + 1. Example: next 01-29, today 02-01 → k1 = 02-02, and 01-29…02-01 are period_expected. Late > 14 gives `[]`, and future targets are unknown.
  - **E-15:** start 07-01, L 28, skip 07-29 → 08-10 is cycle_day 13 (predicted_reference, low, `skipped_cycle_recorded`) and next is 08-26. Without the skip it would be late 12.
  - Stale anchor: on a declared-only history, diff 43 is stale exactly when late would be 15, so the E-11 unknown is never reached there. One resolve_test assertion was updated for this.
  - Suppression: on hormonal contraception the future cycles carry no fertile or ovulation dates. Pregnancy and postpartum-not-returned give `[]`.
  - Future targets inherit today's stale/skip warnings. A real start logged in (today, target] resolves normally.
- Benchmarks:

  | Benchmark | Result | Budget |
  |---|---|---|
  | Range31 | 9.1 µs, 1 alloc | ≤ 150 µs |
  | FutureCycles | 0.54 µs | ≤ 5 µs |
  | Resolve | 356 ns, **0 allocs** | ≤ 3 µs |
  | Prepare | 5.4 µs, 8 allocs | — |

  Prepare has 8 allocs because the bench input has an ignored skip. Coverage 99.0 %.
- Task text fixed: k=6 is 12-22, not 12-24.
- v6 vs v1.1:
  - The roll-forward is replaced by the FutureK projection, with no period_expected after today.
  - The late rebase replaces "period day 5 tomorrow".
  - Nothing is predicted after k6.
  - The stale anchor replaces unknown / 62 days late.
  - Skips act as virtual anchors.
  - `prediction_based_output` is raised only on stale, skip and future statuses.
- Open items:
  - T-M9-21:
    - Fill `FutureCycles` on the today payload.
    - Encode `CurrentPeriodEnd = NoDay` for skip anchors, and `FutureK`.
    - Encode `[]` vs null for an empty list.
    - Decide whether `Diagnostics` is exposed.
  - T-M9-22: lock the Range31 / FutureCycles ceilings. The benches are in `future_test.go`.
  - T-M9-24: the loader passes `Skips`.
  - T-M9-29: the skip endpoint must store exactly the predicted next. Otherwise the skip is ignored but still excludes its cycle from medians.
  - Product questions:
    - Should an unmatched skip stop excluding its cycle?
    - Should stale/skip warnings carry onto future targets?
    - Should regularity for far-future targets use today as ref (as it does now)?

## T-M9-10b — Request deadline reaches every query
- New `httpx.DriverContext(ctx)`. When `ctx` is the `fiber.Ctx` itself, it returns the bounded context from `RequestDeadline`; any other context is returned unchanged.
  - This covers the ~457 call sites that pass `c`.
  - Only a direct `fiber.Ctx` matches, so `context.WithoutCancel(c)` detached work is never killed when the request ends.
- `platform/db/instrument.go`: every call into the base driver goes through `DriverContext` (connect, prepare, begin, query, exec, ping, stmt). Metrics and the slow-query log still see the caller's ctx. `interpolateParams` is unchanged.
- Integration tests:
  - `SELECT SLEEP(6)` run with `c` is cancelled at about 5 s and the response is 500 (never 401).
  - Queries with args, exec and transactions are bounded.
  - Long-running routes and `WithoutCancel(c)` are not bounded.
- Open items:
  - Contexts derived with `WithValue(c)` / `WithCancel(c)` are not bounded. This is deliberate: they can't be told apart from detached work.
  - The database/sql pool wait still uses the caller's ctx.
  - `platform/db` now pulls in fiber through httpx (harmless).

## T-M9-21 — Engine v6 today_ui mapping and §2.35 JSON encoding
- New engine API:
  - `(*State).TodayUI(st)`: 0 allocs.
  - `(*State).Today(today) (Status, TodayUI)`: fills `FutureCycles`. Resolve and Range still leave it nil.
  - `AppendJSON(dst, *Status, *TodayUI)`: a hand-written encoder into a caller-owned buffer, 1.7 µs and 0 allocs.
  - `DecodeJSON`: strict (unknown keys and enums are rejected), for tests and the T-M9-32 shadow diff.
- `TodayUI.PredictionCard` is now `Opt[…]`, so the card can be null.
- Golden `engine/testdata/spec_2_35_today.json`: byte-identical to the encoder output. It follows the spec key order, with the additions `limits`, `future_k` and `today_ui` at the end.
- Round trip: decode(encode(x)) == x over 1 000 statuses.
- today_ui rules:
  - The ring and start/end windows follow §2.43.
  - The end prompt counts "days_since" as cycle_day.
  - Banners follow table order, so late > 14 shows `overdue`; `ask_last_period` comes only from the stale anchor.
  - The pregnancy-test button is shown only with the overdue banner.
  - The prediction card is null with no next start or when the engine is suspended. Under suppression, fertile and ovulation are null.
- Encoding:
  - `future_cycles` is omitted when nil and written `[]` when empty.
  - `warnings` and `reasons` are always arrays.
  - Sources stay "unknown" when only the dates are null.
  - `Diagnostics` is not exposed, because the spec is silent.
- v6 vs v1.1 JSON (for T-M9-32):
  - New keys: `clinical_flags`, `metrics.cycle_regularity`, `future_cycles`, `suggest_pregnancy_test`, `limits`, `future_k`, `today_ui`.
  - New nullable fields: `cycle_day`, `days_to_*`, `days_late`, predicted next / ovulation, and `current_period_end` (skip anchors).
  - New enum values: `predicted_reference`, `bbt_confirmed` / `lh_positive`, 8 warnings, `insufficient_data`.
  - `future_cycles` carry the biological fertile window; the card carries the display window.
- Open items:
  - T-M9-22: lock EncodeDay / TodayUI / TodayPayload, and re-measure Range31 (29.9 µs under load, against 9.1 µs earlier).
  - T-M9-24/32: serve through `State.Today` + `AppendJSON` with a pooled buffer.
  - Product/frontend questions (T-M9-42):
    - Should late > 14 show overdue, or ask for the last period?
    - When the next date is past, should the card show future_cycles[1]?
    - Should PMS be hidden under hormonal contraception?
    - Should a user with no data get a start ring or a banner?

## T-M9-12 — Worker process for stage and beta
- New `api worker` (`cmd/api/worker.go`): it mounts the same routes, so every job, scheduler entry and poller registers exactly as in the API. It has no HTTP listener and never runs migrations. `/up`, `/ready` and `/metrics` are served on `METRICS_ADDR`. It drains on SIGTERM.
- New `api healthcheck worker [ready]`.
- `auth/lifecycle.go`: all `OnLifecycle` hooks go into one set per app. New `API_RUN_WORKERS` (default 1; 0 turns all background work off, and dispatch still enqueues). If a hook fails to start, the hooks already started are stopped.
- Stage and beta compose:
  - backend-go gets `API_RUN_WORKERS: "0"`.
  - New `backend-go-worker` service: same image and env (except `RUN_MIGRATIONS=false`), `command: [worker]`, the `backend-storage` and `media-storage` volumes mounted for the sweeps, no ports, depends on a healthy backend-go, 30 s grace. No volume changes.
- The deploy scripts start the worker together with backend-go. `wait_worker` health-checks it, and a `SERVICES` list that names backend-go adds the worker automatically. The deploy-stage and deploy-beta skills are updated. Nothing was deployed.
- Local smoke run: with the API on `API_RUN_WORKERS=0`, the OTP job stayed pending; `api worker` then processed it and took the scheduler leader key.
- The main session:
  - fixed the verify command to `go build -o /dev/null ./cmd/api` (`backend-go/api/` is a directory);
  - updated the `docs/engine-v6/jobs.md` header.
- Behaviour on stage and beta:
  - The safety nightly result lives in the worker's memory. The API runs it once at start-up, so its dashboard row refreshes only on deploy, until T-M10-24 persists it.
  - Lab and record-extract jobs can start up to about 5 s later (worker poll).
  - `asynq_queue_size` is now exported by the worker.
- Open: after the next stage deploy, request an OTP and run `docker logs ritme-stage-backend-go-worker-1 | grep "OTP SMS"`, then record the result here.

## T-M9-08 — Benchmarks, baseline and query budgets with a CI gate
- `internal/benchfixtures`: deterministic fixtures (fixed day 2026-10-21, seeded PCG). Histories H = 6/24/60/120, a 365-day log with BBT/LH/mucus, and users in all six modes. `cyclesnap` builds `service.Snapshot` without a DB.
- Bench files in resolver, view, legacy, cycle cache, fertility, analysis, messages/rules and auth/passport, all with `ReportAllocs`.
- `internal/querybudget`: cold/warm query budgets for 14 routes plus auth, measured on the real router and MariaDB (`make test-int PKG=./internal/querybudget/...`).

  | Route | Cold / warm queries (auth included) |
  |---|---|
  | today | 8 / 4 |
  | month full | 8 / 4 |
  | home | 20 / 15 (README target ≤ 8) |
  | messages daily | 11 / 8 (target ≤ 4) |
  | fertility insights | 8 / 5 (target ≤ 4) |
  | analysis summary | 13 / 11 (target ≤ 6) |
  | auth | 2 / 1 |

- `bench/run.sh` + `bench/baseline.txt` (M4, go1.27.1, recorded under load). `bench/gate`: a Go gate that fails on any allocs/op increase and only warns on a > 20 % time slowdown.
- New make targets: `bench`, `bench-baseline`, `bench-compare`, `bench-gate`. benchstat is pinned in the Makefile (not in go.mod).
- `.github/workflows/bench.yml`: runs on push/PR under `backend-go/**`. Time is compared against the base commit on the same runner; allocs against `baseline.txt`. No secrets needed. Timeout 60 min.
- Doc: `docs/engine-v6/perf-baseline.md`. Example baselines: `view.Build` H=24 36 µs; `DayJSON` 139 µs / 880 allocs; `MonthJSON` full 6.5 ms / ~27 k allocs; analysis summary 1y 2.3 ms.
- Open items:
  - CI goes green only after a push. The first amd64 run may differ in allocs from arm64 (M4); if so, re-record the baseline from the CI artifact.
  - **Re-run `make bench-baseline`** after T-M9-22. The engine benches added by T-M9-20/21 (EncodeDay, TodayPayload) landed while the baseline was being recorded.
  - Re-record later on an idle machine; only the time column changes.

## T-M9-22 — Engine v6 hardening: invariants, fuzz, goldens, budgets
- `invariants_test.go`: 2 000 seeded histories × 400 days, plus `Range == map(Resolve)` byte for byte and deterministic Prepare. `TestInvariants_DetectBreakage` proves each checker catches a violation. A one-off run over 40 000 more histories was clean.
- `FuzzPrepare` (new) and `FuzzBBTDetect`, with seed corpora committed under `testdata/fuzz/`. About 8 min of fuzzing after the fixes was clean.
- **Three engine bugs found by fuzzing and fixed** (regressions in `TestInvariants_FuzzRegressions`):
  1. A logged bleeding day could show as fertile. A closed period ending after today, or longer than the cycle, sent its later days to the future projection. Now any target inside a closed real period resolves from history (`loggedPeriodCovers`).
  2. A skip that fell on a logged bleeding day became an anchor. Now the log wins and the skip goes to `IgnoredSkips`.
  3. A positive LH on the last cycle day put ovulation on the next period start. That LH no longer counts; new invariant "ovulation < next start".
- 21 persona goldens (`testdata/personas`, ~750 KB): the today payload plus 365 days of Range. The §2.36 table and the §2.35 anchors are also checked by hand. Regenerate with `-run TestGoldens -update`.
- `budget_test.go`: every engine bench is built from one table, with a time ceiling (enforced when b.N ≥ 1000) and an allocs lock (`TestBudgets_Allocs`, runs in every `go test`).

  | Bench | Median | Ceiling | Allocs |
  |---|---|---|---|
  | Prepare | 2.9 µs | 300 µs | 8 |
  | Resolve | 0.21 µs | 3 µs | 0 |
  | Range31 | 7.6 µs | 150 µs | 1 |
  | FutureCycles | 0.28 µs | 5 µs | 1 |
  | TodayUI | 0.015 µs | 1 µs | 0 |
  | EncodeDay | 0.62 µs | 20 µs | 0 |
  | TodayPayload | 1.1 µs | 30 µs | 1 |

  The FutureCycles, TodayUI and TodayPayload ceilings were chosen by the agent; README sets none.
- The engine rows of `bench/baseline.txt` were re-recorded; `make bench-gate` shows 0 regressions. Coverage 99.4 %.
- New `docs/engine-v6/engine.md`: API, rule → E-id → file table, invariants, perf locks, how to add a rule.
- Open items:
  - T-M9-24 loader: `End = NoDay`, `Some(...)`, LH faint → negative, Skips passed as stored; decide whether to drop markers after today.
  - T-M9-29: reject or warn on a skip on a logged bleeding day.
  - Product question: should the API reject a period end in the future, or a period longer than the cycle? Suggestion: reject it in period-write validation.

## T-M9-14 — k6 load tests and SLO baseline (scripts; baseline in T-M9-14b)
- `backend-go/loadtest/`:
  - Six k6 scenarios (`app_open`, `calendar_browse`, `log_burst`, `analysis_tab`, `fertility_tab`, `soak`), plus `lib.js`, `thresholds.js` (README §8 p95 budgets) and `run.sh`.
  - `seed/`: 2 000 users with 6–36 months of history; persona mix TTC 15 %, pregnant 5 %, postpartum 4 %, pill 5 %, the rest cycle users. It is idempotent (`RESET=1` starts over). Tested against a scratch DB: 2 000 users and 516 k daily logs in 25 s.
  - `metricsdiff/`: a per-route table from `/metrics`, scraped before and after a run.
- Guards: the seeder accepts only a loopback DSN whose DB name contains test/local, and always refuses `ritme_test` and `gt_*`. The metrics scraper and `lib.js` accept only loopback URLs.
- Tokens are minted with `passport.Issuer` straight into the scratch DB (the OTP rate limits make the API path impossible for 2 000 users). They go to `loadtest/.tokens.json`, mode 0600, git-ignored. No OTP bypass was added to the API.
- Make targets: `loadtest-db`, `loadtest-seed`, `loadtest-api`, `loadtest SCENARIO=…`, `loadtest-lint`, `loadtest-drop`. The k6 image is pinned to `grafana/k6:1.3.0` (unverified); `K6_RUNNER=local` uses a local k6.
- **Not measured:** the disk filled up (mostly per-agent Go build caches in the scratchpad) and Docker Desktop hung. The baseline moved to **T-M9-14b**.

## T-M9-13b — Config caches on /home and the language registry
- `/home`: task templates, affirmations, challenges (+ `challenge_programs`) and published articles are configcache snapshots (`internal/home/config.go`), with filtering in Go:
  - phase match follows the `utf8mb4_unicode_ci` column rules;
  - a copy of MariaDB's `JSON_OVERLAPS`;
  - articles: tagged first, then untagged, limit 6.
  - The queries now read whole tables: `ListActiveTaskTemplates`, `ListActiveAffirmations`, `ListPublishedHomeArticles`.
- `internal/i18n/registry.go`: the language list is a configcache snapshot (no Redis GET per request). `Flush` publishes `languages` and also deletes the legacy key `ritme-go:languages.registry`, for older builds during a rolling deploy. `CacheTTL` is removed.
- The main session:
  - removed `content.RegistryTTL` (`routes_content.go`, `content.go`, its test);
  - set the measured budgets in `internal/querybudget/budgets.go` (home 20/11; analysis cold +1);
  - relaxed `TestMessagesSelectedConstantInRules` to warm + 1, because the seeded capped rules trigger the one stats prefetch;
  - updated `admin/languages` `TestProvisionNewLanguage`: Go no longer writes the Redis key, so the test plants it;
  - added the before/after table to `docs/engine-v6/perf-baseline.md`.
- Tests: new unit tests and integration tests (`home/config_int_test.go`, i18n snapshot + pub/sub across two processes). `registry_ttl_test.go` was deleted (it tested the removed TTL).
- Verify: `make test-int` for home/i18n/querybudget/admin/content is green. The full `make test-int PKG=./...` run was interrupted by a crash of the local test MariaDB (InnoDB semaphore timeout after the earlier Docker hang); after restarting it, all 56 affected packages passed.
- Also fixed: `internal/auth/lifecycle_test.go` `listenOnce` hung when Shutdown landed before fasthttp entered Serve (a T-M9-12 test race). It now closes the listener after Shutdown; 20 runs in a row are green.

## T-M9-23 — user_profiles.user_id UNIQUE with a safe dedupe (D-127)
- Migration **00096** `user_profiles_user_id_unique`, mirrored by Laravel `2026_10_21_000096_add_unique_user_id_to_user_profiles.php`. `make schema-diff` is green.
- Dedupe, deterministic by id:
  - **Kept:** for each user_id, the row with the lowest id (the one every reader already returned).
  - **Merged:** each NULL column on the kept row is filled from the newest removed row that has a value for it. Non-NULL kept values and the NOT NULL columns never change.
  - **Removed:** the other rows. Each is first copied verbatim into `user_profiles_dedupe_backup` (original id as PK, plus `removed_at` in Tehran time).
- Indexes: `UNIQUE user_profiles_user_id_unique` is added and the old `user_profiles_user_id_foreign` key is dropped (the unique key covers the FK).
- Re-running is safe (`INSERT IGNORE`, merge fills only NULLs, `IF [NOT] EXISTS`).
- Down: restores the FK index, drops the unique key, restores the backup rows with their original ids, drops the backup table. Columns that were filled on the kept row stay filled.
- `profile.InsertProfile` is an upsert (`ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`). The first save retries a deadlock (1213) up to 5 times. 10 concurrent first saves give one row and all 200.
- Contract profile/onboarding: 124 passed, 0 failed.
- Local count: 0 duplicates (the local sqlite has 18 rows for 18 users).
- **Before beta/prod:**
  - Run the read-only count `SELECT user_id, COUNT(*) FROM user_profiles GROUP BY user_id HAVING COUNT(*) > 1` on stage and beta (needs a user OK), and record the result here.
  - Take a dump of `user_profiles_dedupe_backup` before any rollback.
  - The `ALTER TABLE` locks `user_profiles` briefly; run it on prod at a quiet hour.
- Open items:
  - **T-M9-23b**: the admin writer still uses a plain INSERT (1062 in a race).
  - On Laravel sqlite (local dev/phpunit) the dedupe is skipped, so a duplicate there stops the migration.

## T-M9-23b — Admin profile writer upserts
- `CreateProfilePlan` (admin `core.sql`) is an upsert: `ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`, the same pattern as `profile.InsertProfile`. `users.Update` uses the returned id, so a race loser updates the existing row.
- New `inTxRetry` retries the transaction up to 5 times on a deadlock (1213).
- `TestUpdate_ConcurrentFirstProfileWrites`: 8 concurrent admin `PUT /users/:id` for a user with no profile → all 200, exactly one row. This closes the T-M9-23 open item.

## T-M9-06 — Postpartum red flags, visit reminders and EPDS urgency as config (D-135)
- **Migration 00097** creates the new table `postpartum_recovery_logs`, mirrored by Laravel `2026_10_21_000097_create_postpartum_recovery_logs_table.php`. There was no recovery table to extend: the old log lives in taxonomy slots of `health_log_entries`.
  - Columns: `temperature_c`, `pad_soaked_within_hour`, `severe_pain`, `breathless`, `postpartum_visit`.
  - Unique on (user_id, log_date); cascades on user delete.
  - `make sqlc` added the model struct to all 48 store packages. schema-diff: 215 tables.
- Registry keys:
  - `postpartum.redflags {fever_c 38.0}`
  - `postpartum.visits {21, 42, 84}`
  - `epds.urgent_total {score 13}` (31 = off; **H-01 advisor-pending**)
- Red flags give urgent alerts with «همین حالا با ۱۱۵ تماس بگیر یا به اورژانس برو» and a 115 call: fever ≥ 38.0, a pad soaked within an hour, severe pain, breathlessness. Heavy lochia and clots stay warnings. If the admin row is missing, the embedded copy is used.
- EPDS: Q10 > 0 is always urgent (115 + 123). The total threshold comes from config; with 31 (off), a total of 13 shows «زودتر با پزشکت صحبت کن».
- Visits: for a delivery on 10-01, contact by 10-22, the 6-week reminder on 11-12, the comprehensive visit by 12-24. «امروز ویزیت داشتم» (via `PUT /postpartum/recovery` `postpartum_visit`) marks the right one done. The lazy notice is written once per kind on `GET /postpartum`.
- Frontend:
  - The recovery sheet adds a temperature stepper and three yes/no chips; the urgent 115 card scrolls into view after saving.
  - The postpartum home has a «ویزیت‌های بعد از زایمان» card; the old 6-week row was removed.
  - The style and dark gates pass; no screenshots were taken.
- Safety scenarios: `postpartum_fever`, `postpartum_pad_per_hour`, `postpartum_breathless`, `epds_q10_low_total`. `epds_high_total_urgent` now reads the config.
- The main session fixed a flaky order in `TestPostpartum_RecoveryRedFlags` (it looped over a map; now an ordered slice), and added H-20 (fever 38.0) and H-21 (visit windows, `pain_level = severe`) to `docs/engine-v6/clinical-signoff.md`.
- Open items:
  - **T-M9-06b**:
    - **data export of the new table**
    - the daily message ignores the new red flags
    - postpartum goldens and OpenAPI `required`
    - translation seed resync
    - H-20/H-21 in the Claude Doc
  - The «ویزیت ۶ هفته» label is fixed text even though the day is configurable.
  - Notices are created only when the user opens the postpartum home.
- Deploy to stage/beta: not done (needs user OK).

## T-M9-24 — Schema for engine v6 inputs and revisions
- **Migration 00098** `engine_v6_inputs`, mirrored by Laravel `2026_10_21_000098_add_engine_v6_inputs.php`:
  - `user_profiles.onboarding_last_period_start DATE NULL` and `user_profiles.timezone VARCHAR(64) NULL`
  - `postpartum_profiles.breastfeeding TINYINT(1) NULL`
  - new `cycle_skips` (UNIQUE user_id + expected_start, cascade)
  - new `user_revisions` (PK user_id; `engine_rev` / `analysis_rev` BIGINT UNSIGNED default 0, cascade)
  - Everything uses `IF [NOT] EXISTS`. It is safe on populated tables (nullable columns, empty tables, no data moves). Down drops only the new columns and tables. schema-diff: 217 tables.
- New queries:
  - `cycle/revisions.sql`: `BumpUserRevisions` / `BumpAnalysisRevision` / `GetUserRevisions`. 10 concurrent bumps give 10.
  - `cycle/skips.sql`: list, idempotent insert, user-scoped delete.
  - profile: `GetProfileEngineInputs`, plus upserts `SetOnboardingLastPeriodStart` / `SetProfileTimezone`.
  - postpartum: `GetPostpartumBreastfeeding` / `SetPostpartumBreastfeeding`.
  - Timestamps come from `sqlc.arg(now)` (Tehran clock), never `NOW()`.
- Enums: `DataSourceBleedingLog` with `IsKnown` / `ParseDataSource`; `IsActual()` now covers `bleeding_log`. PHP-parity `DataSourceFrom` is unchanged (D-138).
- Profile JSON (option a, the same as Laravel `toArray()` after the twin migration): `onboarding_last_period_start` and `timezone` appear as null at the end of the column list.
  - `TestColumnNamesMatchSchema` now replays the later `ALTER TABLE`s.
  - 41 goldens (89 profile objects) were hand-edited, because the Laravel contract image could not be built offline. `contract/fixtures/dump.sql` was updated.
  - OpenAPI: both fields added to the three profile schemas.
  - `make contract ROUTES=profile,onboarding,healthlog`: 169 passed, 0 failed.
- `go test ./...` is green except `resources/translations` `TestSeedMatchesFrontendMessages`, which has been red since T-M9-06 (the postpartum copy was not resynced) and is fixed in T-M9-06b.
- **Full contract (`ROUTES=all`), the same on HEAD:** 1870 passed, 44 failed (postpartum 17, pregnancy 9, analysis-ttc 7, analysis 6, pregnancy-tools 3, logs 1, teen 1). Causes:
  - `dump.sql` lacks 00096/00097, so `postpartum/activate` returns 500;
  - Persian copy differences such as «سیکل» vs «چرخه».
  - Added to T-M9-06b.
- Open items:
  - Re-record the profile/healthlog goldens with `make contract-record` once the Laravel contract stack builds.
  - The loader notes (NoDay, Some, LH faint, skips, positive pregnancy tests, breastfeeding/delivery date, teen band) go to T-M9-25.
  - `PromoteLoggedStart` ignores `bleeding_log` (removed in T-M9-28 per I-02).
  - Revision bumps are wired in T-M9-30/31. A missing `user_revisions` row reads as 0/0.

## T-M9-26 — User-local day: X-Timezone header, stored zone, LocalToday
- Implements **D-126** (X-07 scope): `X-Timezone` header → stored `user_profiles.timezone` → Asia/Tehran, on the engine path.
- Frontend `apiClient` sends `X-Timezone` (from `Intl…resolvedOptions().timeZone`) on every call; it is omitted when the zone is unknown.
- New `internal/platform/usertz`:
  - parse + cache via `time.LoadLocation` (tzdata embedded); rejects `""`, `Local`, names over 64 chars and invalid characters;
  - global middleware in `cmd/api/app.go`, after CORS. It reads the stored zone lazily, only when a handler asks, and caches it per user for 15 min;
  - a changed zone is stored after the response in a goroutine with a 5 s timeout, using `StoreUserTimezone` (`UPDATE … WHERE timezone IS NULL OR timezone <> ?`). It writes no `updated_at`, bumps no revision and creates no profile row. Errors are only logged.
- `civildate.LocalToday(ctx, now)` / `LocalNow`; with no middleware this is exactly `civildate.Today` (Tehran).
- The cycle service handlers (`/cycle/today`, today inside `/cycle/date/{date}`, `/cycle/month`) use the user's local day. `markRecalculated` stays Tehran wall-clock.
- CORS: preflight test with `x-timezone, if-none-match` (new `cmd/api/cors_test.go`); `cors.go` already echoes requested headers. Stage/beta nginx does not strip the header (read only).
- `docs/engine-v6/README.md` §5 lists the 249 non-test `civildate.Today`/`InTehran` call sites that stay on Tehran (engine-adjacent ones move in T-M9-32…38).
- Task fix: the Nowruz test line was one minute off (20:29Z is still 03-20 in Tehran).
- Open:
  - in-flight zone-store goroutines are not awaited on shutdown (harmless; the next request re-stores);
  - with several API instances, a zone change can take up to 15 min to reach each one's cache;
  - `ForDate` still parses its Y-m-d param in Tehran (no effect on dates).

## T-M9-25 — Engine input loader: Load and LoadMany in at most 4 queries
- New `internal/cycle/inputs`:
  - `Load(ctx, q, uid, today)` and `LoadMany(ctx, q, uids, today)` (batches of ≤ 500, duplicate ids merged). `Load` is `LoadMany` of one id. `today` is a parameter, so the loader never calls `NOW()`; callers pass `civildate.LocalToday` (T-M9-26).
  - `Encode`/`Decode` JSON with `v: 1`; an unknown or missing `v` → `ErrVersion`, which forces a reload.
- Queries (`db/queries/cycle/inputs.sql`), all `IN (sqlc.slice)`:
  1. one joined profile row from `users`, so users without a profile still get mode, teen band and periods;
  2. narrow periods;
  3. skips;
  4. one `UNION ALL` of markers (`health_log_entries` measurements, `fertility_logs.lh_test`, `daily_health_logs.basal_body_temperature`).
  Measured with the T-M9-07 counter: exactly 4 queries; a bare user costs 1.
- `internal/contraception/hormonal.go` (I-03):
  - a saved method decides alone (pill → copper IUD = not hormonal);
  - with no saved method, onboarding `contraceptive_pill` is hormonal, and `iud`/`hormonal_medication` are not.
- Mapping:
  - **Periods:** `is_estimated` rows are skipped. `user_profile_confirmed` rows are never periods; the newest one becomes the declared anchor only when `onboarding_last_period_start` is NULL. `bleeding_log` → `PeriodBleedingLog`. Open period → `End = NoDay`.
  - **BBT:** v2 wins, then the 34.00–43.00 range check (no fallback to the legacy value); exact decimal → hundredths.
  - **LH and pregnancy tests:** positives only, one per day.
  - **Marker window:** from the earlier of the 7th-newest real period's start and today − 400, through today + 1.
  - **Lengths:** manual when `lengths_auto = 0`.
  - **Postpartum:** delivery and breastfeeding are passed in every mode (the engine applies E-14).
- Inputs covered by unit tests only (no real-DB fixture):
  - TTC mode from `user_goal`, menopause, the default and unknown stored modes;
  - malformed `medications` JSON, Fahrenheit BBT typos;
  - more than 500 ids;
  - non-estimated `onboarding_initial`/`system_default`/`engine_prediction` rows (read as real periods);
  - an out-of-range v2 BBT on a day that also has a legacy value.
- Open:
  - **p95 ≤ 5 ms is not proven.** Load average was 58–71 and a bare `SELECT ?` took 3.4 ms; measured p50 5.7 / p95 15.8 ms. `TestLoad_Budget` asserts the limit only when a trivial round trip is < 1 ms, so re-run it on an idle machine.
  - **Import-cycle risk:** `inputs` → `contraception` pulls in care/notifications/i18n. If T-M9-32+ hits a cycle, move the I-03 table to a leaf package.
  - **T-M9-31:** the snapshot depends on `today` (the marker window), so key or validate the L2 snapshot by day as well as by revision.

## T-M9-06b — Postpartum follow-ups: daily message red flags, goldens, translation seed, advisor doc
- Daily message (`messages/manager/postpartum_engine.go`): the urgent red flags come first, in `guide.RedFlagAlerts` order (fever, pad_per_hour, severe_pain, breathless), with level `urgent` and 115. They use `cp.Urgent`, so an empty admin row falls back to the embedded copy. Then heavy_bleeding › large_clots › the due check-in, as before.
- `contract/fixtures/dump.sql`: 00096 (`user_profiles_user_id_unique`) and 00097 (`postpartum_recovery_logs`).
- 35 Go-only goldens re-recorded from Go (postpartum 17, analysis 6, analysis-ttc 7, logs 1, teen 1, pregnancy-tools 3), each diff read by hand.
  - Causes: the missing dump tables, the new T-M9-06 fields, M8 Persian copy, and v2 emergency rules firing with embedded copy (D-134).
  - New `contract/allowlist/pregnancy.yaml`: 11 entries, all D-134 (severe tier, shared alert ids, summary count).
  - D-134/D-135 notes in `deviations.md`. No Laravel golden was touched.
- OpenAPI: postpartum `visits`, the recovery red-flag fields and EPDS `thresholds.urgent_total` are now `required`.
- Translation seed resynced (`postpartum.json` was the only drift); i18n testdata regenerated.
- Advisor doc: H-20 (postpartum fever 38.0 °C) and H-21 (visits 21/42/84, severe pain) added; the lead now says 15 decisions.
- Verify on the combined tree: `go vet ./...` and `go test ./...` green; `make contract ROUTES=all` → 1914 passed, 0 failed.
- Open:
  - **Export → T-M9-06c (blocked on a user decision):** D-96 keys, EPDS in the export.
  - `postpartum_source.go` uses version-1 thresholds (`Alerts()` instead of `AlertsOn(date)`) → T-M9-06c.
  - The v1 and v2 pregnancy alerts share one table, so `/pregnancy/alerts/summary` double-counts (3→6 in the fixture). Product question.
  - `cmd/contract` has no `record-go` mode; this task used a scratch `go build -overlay` (scratchpad `recgo/main.go`).
  - Laravel `translations:import` was not run.

## T-M10-17 — Health schema for vitals targets, lab value confirmation and checkup test kind
- Migration **00099** `health_targets_confirmation_test_kind`, with the Laravel twin `2026_10_21_000099_add_health_targets_confirmation_test_kind.php`:
  - new table `vital_targets`: UNIQUE (user_id, context), FK cascade;
  - `lab_markers.confirmed_at`, backfilled from `lab_reports.verified_at` only for verified reports;
  - `checkup_records.test_kind`;
  - `checkup_types.informational` (default 0).
  - Down reverses everything; `IF [NOT] EXISTS` throughout.
- sqlc:
  - vitals: `ListVitalTargets`, `UpsertVitalTarget`, `DeleteVitalTarget`;
  - labs: `ConfirmLabMarker(id, user_id, now)`;
  - checkups: `InsertCheckupRecord`/`UpdateCheckupRecord` with `test_kind`; type reads pick up `informational` via `SELECT *`.
  - `models.go` churn in every store package (inherent to `make sqlc`).
- `contract/fixtures/dump.sql`: main session added `vital_targets`, the three columns, and `informational = 0` on the 15 seeded checkup types.
- Verify: `schema-diff` OK (218 tables); `make test-int PKG=./db/migrations/...` ok; `make contract ROUTES=vitals,labs,health,profile` → 142 passed, 0 failed.
- No behaviour change (T-M10-19/20/21).
- Open:
  - `UpdateCheckupRecord` now writes `test_kind`. Today's callers don't send it, so an edit sets it to NULL; T-M10-21 must pass the existing value.
  - `low_mgdl`/`high_mgdl` are `sql.NullInt16` in Go (max 32767 is enough for mg/dL).

## T-M10-16 — Clinical registry v2: table-valued keys, per-version source, coverage keys
- `clinicalconfig`:
  - **`rows.go`:** `Def.Kind` (numbers | rows) plus a typed `Table` schema (number/string/enum columns with bounds, integer, nullable, max length; min/max rows; sort; range contiguity with group-by; unique; default). `ParseRows`/`ValidateRows` return problem codes with the row and column. The stored value is `{"rows":[…]}`.
  - **`store.go`:** `Version.Rows` and `Version.Source` (v1 = `Def.Source`, v2+ = the stored column). New getters `Rows(key, d)` and `SourceOf(key, d)`.
  - **`coverage.go`:** 21 new keys in two new groups, `child` and `screening`. None is in the engine group, so the cycle fingerprint is unchanged.
- The full list of keys, v1 values and sources is in `docs/engine-v6/clinical-signoff.md` §B. A pin test ties every v1 value to the constant or seed it replaces.
- Admin API: `kind`, `schema` and per-version `source`; propose accepts rows (422 names the row); the two-advisor flow is unchanged. The Loader now reads `ListClinicalConfigVersions` filtered to approved, for `source`.
- admin-web: a rows table editor (add/remove/reorder, per-column validation, diff against the version in force) and the source shown per version. vitest 47 files / 166 tests.
- Verify on the combined tree: `go vet ./...` ok; clinicalconfig/admin + consumer packages ok; `test-int clinicalcfg` ok; admin-web typecheck, lint and test green.
- Notes for consumers:
  - **T-M10-18:** `pregnancy.trimester` v1 = SPEC (t2 from day 98), but today's `calc.go` uses day 91, so weeks 13+0…13+6 differ until T-M10-18 switches.
  - **T-M10-20:** `lab.ocr` = 0.9 (H-08), today 0.7.
  - Re-dating: re-date when the difference is more than the limit; the band uses GA by LMP on the scan day (ACOG CO 700 footnote, from memory, not fetched).
  - IOM weekly rates come from IOM 2009 Table S-1 (not in code or SPEC).
  - `glucose.diabetic` field names say "fasting", but SPEC says "before meals".
  - Spotting `until_week 12` is today's behaviour; SPEC has no limit.
  - **Screening:** `type_key` uses checkup_types keys (`bp_screening`/`glucose_screening` are added by T-M10-21).
  - **Vaccine schedule:** `vaccine_code` = `child_vaccine_doses.dose_code` (T-M10-22).
- Open:
  - document `kind`/`schema`/`source` in `docs/go-migration/admin-api.md` §23;
  - `ListApprovedClinicalConfigVersions` is now unused (remove it or add `source`).

## T-M10-18 — Pregnancy dating: trimester by completed days, ACOG re-dating, optional cycle correction (D-169)
- New `internal/pregnancy/calc/dating.go`, the one place for the dating rules:
  - `TrimesterOn(totalDays, day)` (`pregnancy.trimester`, 98/196 completed days);
  - `EDDFromLMP` (`pregnancy.edd`, plus a cycle correction only when `pregnancy.edd.cycle_correction.enabled = 1`);
  - `Redate(lmp, hasLMP, scanDate, gaAtScan)` → `{basis, replaced, discrepancy_days, band_until_days, limit_days}`. The band is chosen by GA by LMP on the scan day; replace only when the discrepancy is **more** than the limit.
  - The registry version is the one valid on the subject day.
- `calc.Status` and `/pregnancy/status` gain a `dating` block (OpenAPI `PregnancyDating`). v2 `Dating`/`labels.go:107` use completed days.
- `pregnancy.ApplyScanRule` and `ApplyUltrasoundDatingByRule` are ready, but not yet wired into the save endpoints → **T-M10-18b**.
- `TestNoSecondTrimesterRule` greps `internal/` for week-based trimester rules.
- Fixed `v2_int_test.go:224`, which asserted the old T2 start.
- Frontend: `features/track-pregnancy` `datingResultKey` + `DatingResultLine`; fa/en `common.datingResult.{replaced,kept}`; seed resynced.
- Contract: `contract/allowlist/pregnancy.yaml` D-169 entries (13+x trimester, the additive `dating` block). Result: pregnancy, pregnancy-v2 and analysis-pregnancy 206 passed, 0 failed. Frontend tests 1952 passed; copy:check clean.
- Open:
  - **T-M10-18b:** handlers, healthrecord extract, analysis/v2 callers, frontend onboarding response, pin test.
  - The i18n bundle goldens (`TestBundle_MatchesLaravelGoldens`) are re-recorded together with T-M10-19's commit (both changed seed files).
  - `ApplyScanRule` always compares LMP vs scan when an LMP exists, even if the current basis is manual.
  - The cycle correction moves only the EDD, not the GA.

## T-M10-19 — Vitals: glucose contexts, diabetic targets, report minimums, copy placeholders (D-170)
- `vitals/thresholds.go`:
  - new classes `discuss`, `below_target`, `in_target`, `above_target`;
  - `glucose.contexts` and `glucose.diabetic` are read from the registry version of the reading's date and are part of the version fingerprint;
  - `ClassifyGlucoseReading` for the spec rule (H-09, default 0 = today);
  - lab-method post-meal readings keep the 140/200 bands;
  - order: hypo always wins, then pregnancy (unchanged, T-M9-05), then diabetic target, then the spec rule.
- `vitals/targets.go` + `GET/PUT/DELETE /api/v1/vitals/targets`:
  - owner only; 422 on bad values; bounds 50–300; `set_by` doctor|user;
  - PUT replaces the whole set, DELETE clears it;
  - diabetes comes from `chronic_illnesses`; non-diabetic users never get target classes.
- Reports: stats only with ≥ 2 readings on ≥ 2 days. Otherwise `status: not_enough_data` + `readings_needed`, and the stats are null (the series stays).
- Copy: `{systolic}/{diastolic}/{mgdl}/{mmol_l}` placeholders in the lang files and the translation seed, filled from the registry version of the reading's date.
- Frontend: not-enough-data state on the hub and report; a «هدف‌های قندی که پزشکت تعیین کرده» card and sheet for diabetic users; classes and bands from the API.
- Main session:
  - applied the OpenAPI diff;
  - left the new report/hub fields out of `required` until the vitals goldens are re-recorded (marked `T-M10-19b`);
  - re-recorded the i18n bundle goldens (covers T-M10-18's pregnancy-v2 seed change too).
- Verify: vitals unit + int ok; `internal/http` OpenAPI tests ok; contract vitals and public 39 passed, 0 failed; frontend 1952 tests; copy:check clean.
- Open → **T-M10-19b**:
  - the `message_contents` `vitals_alert` seed rows still hold the numbers (needs a data migration, and possibly safety-content approval);
  - the add-form preview ignores `method` and targets;
  - no contract cases yet for `/vitals/targets`;
  - golden re-record, then `required`.
- Interpretations to review:
  - the fasting/pre-meal/bedtime upper bound is inclusive, post-meal `< 180`;
  - bedtime has a target only when the user set one (no fallback to fasting).

## T-M10-21 — Checkups on the USPSTF v6 registry: intervals, HPV test kind, informational items (D-172)
- Engine (`checkups/engine/screening.go`): cervical, mammography, BP, glucose and dental types read `screening.uspstf` rows on the evaluation day:
  - age band;
  - the last record's `test_kind` (cytology 36, HPV/co-test 60);
  - BMI from the profile via `bmi.who` (≥ boundary; unknown BMI → shown).
- Other types keep the catalog columns. A last record with `result = follow_up` gets no default next due.
- Informational types have no due date or reminder; POST/PUT → 422, history readable, DELETE allowed.
- While pregnant, non-urgent items get `label: after_pregnancy` and no reminder, and are excluded from the summary.
- Migration **00100** `checkups_uspstf_catalog`, with the Laravel twin:
  - new types `bp_screening` and `glucose_screening` (fa/en copy, USPSTF `source_note`);
  - `breast_self_exam` → informational «آگاهی از سینه»;
  - Pap copy mentions HPV; mammography 40–74 every 24 months.
- Record API: `test_kind` is validated (required from age 30, must be a registry kind) and kept on PUT. This fixes the T-M10-17 NULLing.
- admin-web checkup-types: an informational switch; registry-driven interval/age fields are read-only, with a link to clinical-config.
- Frontend:
  - informational card, «پیگیری با پزشک» and «بعد از بارداری» pills;
  - test-kind picker on the Pap record form;
  - self-exam in awareness mode;
  - fa/en, light/dark.
- Outside touches, all needed:
  - sqlc `db/queries/checkups/{user,admin}.sql` (BMI fields, record result/test_kind, `informational`);
  - `features/record-checkup/model/body.ts` (`test_kind`);
  - `screens/home/ui/MenopauseHome.tsx` (the status-tone map);
  - two test fixtures;
  - admin-web message keys.
- Main session:
  - applied the OpenAPI diff;
  - `dump.sql` `checkup_types` rebuilt by running 00100 Up on the dump (rows 16–17, updated rows 1/3/5/6, AUTO_INCREMENT 18);
  - i18n goldens re-recorded;
  - contract: D-172 allow-list in `menopause.yaml` and `messages.yaml`. That file also got the D-169 daily-message entries missed by T-M10-18; the other D-169 entries went in the previous commit.
  - `CheckupItem.informational`/`label` stay optional until the menopause goldens are re-recorded.
- Verify: `go vet ./...` and `go test ./...` green; checkups/admin/migrations test-int ok; `schema-diff` OK; `make contract ROUTES=all` → 1914 passed, 0 failed. Frontend: 1955 tests passed, copy:check clean. admin-web: 168 tests passed.
- Open: H-10, H-11 and H-16 are advisor-pending; document admin-api §12 (`informational`, `registry_key`).

## T-M10-18b — Wire the ACOG re-dating rule into the save endpoints and remaining trimester callers (D-169)
- `pregnancy/handlers.go`: Onboarding and PUT profile run `ApplyScanRule` before saving. Explicit = `age_source` sent. Onboarding returns `data.dating`; PUT returns it in `status.dating`. An optional `CycleLength` source is not wired yet.
- `healthrecord/extract/dating.go`: uses `ApplyUltrasoundDatingByRule`, with `Offer.Redating`. `same_dating` no longer needs `source == ultrasound`.
  - `extract_int_test`: 12+3 with a 2-day difference keeps the LMP; a new 12-day case re-dates.
- Callers switched to the dated functions: `analysis/pregnancy.go`, `v2/service.go` (`TrimesterStartDays`, `TrimesterOfWeek`, `WeekLabelOn`), `v2/calendar`. `v2/preview.go` returns `dating`.
- Main session removed the v1 wrappers `calc.Trimester` and `v2.WeekLabel`, the unused `trimesterStartWeeks`, and an ineffassign in `redatingBand`.
- `coverage_pin_test` pins `pregnancy.edd` through the calc functions.
- OpenAPI: `dating` on onboarding and dating-preview (optional until goldens are re-recorded).
- Frontend: `useCompleteOnboarding` returns `{profile, dating}`; the onboarding page shows `DatingResultLine` before continuing.
- Verify: pregnancy, healthrecord, analysis and clinicalconfig unit + int ok; golangci 0 issues; frontend 1963 tests. Contract for pregnancy, pregnancy-v2, analysis-pregnancy, home, messages, loss, analysis and companion: 497 passed, 0 failed. The remaining `ROUTES=all` failures (labs, consents, auth, privacy, healthrecord-extract) were caused by T-M10-20/23, still in flight at commit time.
- Open → **T-M10-18c:** extract kept-LMP copy, cycle-length source, schema dedupe, goldens + `required`, and a question about onboarding vs a previous pregnancy's LMP.

## T-M10-20 — Labs v6: per-value OCR confirmation, mode-aware ranges, HbA1c bands, borderline as config, JSON summary (D-171)
- Per-value confirmation:
  - every value read below `lab.ocr.min_confidence` (0.9, H-08; was 0.7) waits for the user's confirmation;
  - unconfirmed values are left out of status, red flags, trends and AI;
  - editing a value confirms it;
  - new owner-only `POST /api/v1/labs/{id}/markers/{mid}/confirm` (route wired by the main session from the agent's diff);
  - "confirm" / "confirm all" in lab-verify.
- `labs/ranges.go`: general ranges, always after the sheet's own range. They depend on mode and trimester on the lab date (via `calc.TrimesterOn`):
  - Hb non-pregnant / pregnant / T2;
  - ferritin with CRP;
  - vitamin D bands, TSH in pregnancy, HbA1c bands, all without diagnosing.
- Borderline enabled/margin come from `lab.borderline` (H-07), with a `near_edge` hint.
- AI summary:
  - validated JSON (`DisallowUnknownFields`);
  - categories outside the rules, disease names and doses are rejected, falling back to the rules summary;
  - `labs_ai_filtered_total{reason}` counts the rejections;
  - `ai/pii.go` is unchanged; unconfirmed values are filtered before the prompt.
- Contract: `contract/allowlist/labs.yaml` (D-171). The main session added the `data.marker.*` single-marker paths.
- Verify on the combined tree: labs unit + int ok; `make contract ROUTES=all` 1914 passed, 0 failed; frontend 1972 tests; copy:check labs clean.
- Open:
  - H-07 is advisor-pending;
  - no contract case yet for the confirm endpoint;
  - `ai/fake_chat.go` has no JSON labs fixture, so the contract stack always shows the rules summary;
  - `ConfirmLabMarker` should also be scoped by `lab_id` in SQL (checked in Go today).

## T-M10-23 — Consent ledger: append-only consent events, scope, versioned signup consents (D-174)
- Migration **00101** `consent_events`, with the Laravel twin `2026_10_22_000101_create_consent_events_table.php`:
  - append-only, enforced by two triggers that refuse UPDATE and DELETE; a user delete still cascades (FK cascades don't fire triggers);
  - backfilled from `user_consents`;
  - `schema-diff` OK (219 tables).
- `consent/ledger.go`: every grant and withdrawal writes an event with a source (settings, feature, signup, backfill, telemed, sharelink).
  - Re-granting the same version is a no-op; a revoke is recorded only after a grant.
  - The user row is locked, so writes are serialized.
- Scoped evidence:
  - `telemed_share` {booking_id, doctor_id, items};
  - `share_link` {link_id, expires_at, items};
  - `assistant_profile_items` {items}.
- Versioned signup consents (`terms`, `health_data_processing`, v1) on verify-otp:
  - a new account without them → 422 `consent_required`; no account is created and the OTP is not spent;
  - existing accounts are never refused;
  - token and session unchanged; only `user_id` is logged.
- New owner-only `GET /api/v1/consents/pending` and `/consents/history` (newest first, capped at 200).
- Frontend:
  - the signup payload carries the consents;
  - `PendingConsentsSheet`, once per app open, inside the app lock (mounted in `[locale]/layout.tsx`);
  - consent history on the privacy screen;
  - `consent_required` error copy;
  - the `auth` namespace added to the shell/onboarding/profile message scopes.
- Main session applied the agent's 4 diffs:
  - OpenAPI;
  - wiring in `routes_ai/privacy/auth`, `assistant/service.go` and `telemed/booking_service.go`;
  - contract cases `auth.yaml`/`learning.yaml` (new-user verify-otp sends consents; Laravel ignores the field);
  - the frontend shell.
- Main session also added the `consent_events` table and triggers to `dump.sql`, and re-recorded the i18n goldens.
- Verify on the combined tree:
  - `go vet ./...` and `go test ./...` green; `schema-diff` OK;
  - test-int ok for consent, auth, labs, profile, telemed, sharelinks, assistant, healthrecord and migrations;
  - `make contract ROUTES=all` 1914 passed, 0 failed;
  - frontend typecheck, lint and fsd:lint green, 1972 tests.
- **Deploy note:** creating the triggers needs the TRIGGER privilege on stage/beta/prod. With binlog on, it also needs SUPER or `log_bin_trust_function_creators=1`; without them migration 00101 fails. Check this before the next stage/beta deploy.
- Open:
  - the codes in `consent/summary.go` record no evidence events;
  - history has no pagination;
  - Laravel `translations:import` not run.

## T-M10-18c — Re-dating follow-ups: extract kept-LMP copy, cycle-length source, dating schema dedupe, goldens
- Health-record extract: an apply that keeps the LMP (`Redating != nil && !Replaced`) answers `messages.dating_kept`
  and `offer.kept`; the apply response carries `dating` (`Redating.JSON()`); `same_dating` uses `offer.same_dating_kept`
  when the profile's age source isn't ultrasound. New fa/en keys in `extract/lang`. Int + unit tests in
  `extract_int_test.go` / `extract_test.go` (applied by the main session from the agent's diff).
- `routes_pregnancy.go`: `h.SetCycleLength(effectiveCycleLength(...))` from the cycle engine profile + history
  (`metrics.Calculate`; 0 when only the system default) — no effect until `pregnancy.edd.cycle_correction` is on.
  Costs two queries per calculator call.
- Frontend: `features/track-pregnancy` reuses `entities/pregnancy` `onboardingDatingSchema` (duplicate zod object gone).
- Goldens: `dating` merged into 56 pregnancy goldens + healthrecord-extract (3) + loss (2) from Go (merge-only, other
  Laravel values kept; no "record from Go" make target exists — done with a temporary test). pregnancy-v2 needed none.
  `dating` is now `required` in openapi (onboarding, profile status GET/PUT, `/pregnancy/status`,
  `PregnancyV2DatingPreview`); optional `RecordDatingOffer.dating`. D-169 `dating` allow-list entries dropped
  (pregnancy, healthrecord-extract, loss → `entries: []`); trimester entries stay.
- Verify (combined tree): `make contract ROUTES=all` 1914 passed, 0 failed; pregnancy/healthrecord int ok; frontend
  1974 tests.
- Open: product question — onboarding compares a scan against the profile's previous LMP even when it belongs to an
  earlier pregnancy (ignore LMPs older than ~44 weeks / from a previous episode?). A later GET of an applied document
  still shows the applied text when the LMP was kept (fix lives in `dating.go`). No test for `effectiveCycleLength`.
  The health-record screen may want to show the new server message.

## T-M10-22 — Children: percentile-line crossing, CDC 2022 milestone bands, versioned vaccine schedule (D-173)
- Growth: crossing ≥ `child.growth.cross_lines` major lines (P3/15/50/85/97, strict — on-the-line isn't a crossing)
  between consecutive values of one indicator → `discuss` (verdict; point `discuss`/`crossed_lines`; series
  `crossing {lines, from, to, status, label}`); band and threshold from `child.growth` in force on the measurement day;
  the < P3 / > P97 band rule stays `check` (check > discuss > normal).
- goose 00102 `child_milestones_cdc2022` + Laravel twin `2026_10_23_000102`: CDC 2022 15- and 30-month bands (5
  milestones, 1 activity, doctor note each; `needs_review=1`, `band_source=cdc_2022`); 3-month band marked
  `band_source=ritme` (H-15). Down removes the rows and user checks on them. API `band.source`; app note on the
  Ritme band. `dump.sql` `catalog_items` rebuilt via temp DB (+14 rows).
- Vaccines: due ages from `child.vaccine_schedule` in the version in force on the child's birth date (H-17); catalog
  keeps names/copy; dose records unchanged; `GET /vaccines` adds `schedule {version, source}`. Admin child-content:
  ages read-only from the registry + link to clinical-config; vaccine meta no longer edits age/visit.
- `contract/allowlist/children.yaml` (D-173); openapi additive optional fields; i18n bundle goldens re-recorded
  (3 new keys); milestone copy listed in `docs/engine-v6/clinical-signoff.md` §C for review.
- Main session also kept: `db/migrations/consent_events_int_test.go` now clears `version_id >= 101` (assumed 101 was
  the last migration); admin-web `childContent.vaccinesHint` copy updated.
- Verify: children unit/int, migrations int, contract children 18 passed / 0 failed; frontend typecheck/lint/fsd/dark/
  1974 tests/copy:check; admin-web 169 tests; schema-diff OK (219 tables).
- Open: children goldens not re-recorded (new fields allow-listed); a catalog dose missing from the registry doesn't
  appear in a child's schedule (admin shows it under "no visit"); no `child_age_notes` for 15/30 months while H-15 is
  advisor-pending; milestone copy needs clinical review.
- Migration 00102 must run on stage/beta with the next deploy.

## T-M9-27 — Declared onboarding anchor replaces the seed period row
- D-123 (I-01; user-visible, approved in T-M9-01): `profile.Save` writes `user_profiles.onboarding_last_period_start`
  whenever `last_period_start` is set to a new value outside pregnancy mode (re-sending the stored value is no
  declaration); `last_period_start` keeps its legacy meaning. `syncOnboardingPeriodLog` and
  `InsertOnboardingCycleHistory` / `UpdateOnboardingCycleHistory` are gone.
- Goose **00103** `onboarding_declared_anchor` (data only, Go-only, no Laravel twin; prod untouched):
  untouched seed (end/length = the old re-sync rule, or both NULL for a running one; start = profile LMP unless a later
  real period exists) → start copied into the column, row deleted; any other seed → `user_logged`, confirmed;
  `user_revisions` +1 for every affected user. Down re-creates the seeds from the column with the old rule and NULLs
  the column; converted rows stay `user_logged`. Int test: 5 personas, up → down → up.
- `inputs.Load`: no seed fallback; a stray `user_profile_confirmed` row is neither a period nor the anchor. Int test:
  onboarding-only user → `onboarding_declared`; +90 days → stale anchor (§2.41).
- Visible (v1 mode too): onboarding-only users resolve `onboarding_based`, their period history is empty and the
  calendar no longer paints the declared period as logged. Assumed-range rendering → T-M9-33/T-M9-45.
- Main session: `routes_shop_checklist.go` supply-reminder predictor now also runs from the declared LMP when the
  history is empty (else onboarding-only users lost the reminder); `TestConsentEndpoints` fixed separately (T-M10-23
  leftover: GET /consents omits the mandatory signup consents).
- Contract: `allowlist/profile.yaml` +3 entries (D-123). `ROUTES=all` 1914 passed, 0 failed; full `make test-int` green;
  schema-diff OK (219 tables).
- Open: pregnancy mode writes no anchor (kept the old seed rule; Scope says "whenever"); `Save` revision bump → T-M9-30;
  the Down can't rebuild seeds faithfully for users onboarded after Up; stale comment on `ListEngineInputPeriods`.
- Migration 00103 runs with the next stage/beta deploy and is user-visible there.

## T-M9-28 — Bleeding-log bridge v6, postpartum flow and breastfeeding switch
- Bridge (`internal/healthlog/bridge.go`): a `flow` day write rebuilds the affected period in one transaction under
  the per-user lock (`periods.LockUser`) — grow, shrink, split, delete or merge with the same adjacency rule — then
  the exported `periods.RecomputeCycleLengths` and `Reanchor`. Rows are `source=bleeding_log`, confirmed (D-124
  supersedes D-55; D-138: `bleeding_log` is Go-only until Laravel `DataSource` gets it). Button rows (`user_logged`)
  are never rebuilt. `PUT /cycle/period/{id}` makes the row confirmed `user_logged`; `PromoteLoggedStart` removed.
- Pregnancy (I-02): flow creates no period and leaves the LMP; `alerts.ForLoggedFlow` reuses the v1 bleeding symptom
  alert (low→mild, medium→moderate, high/very_high→severe = emergency branch), raised only when the day's flow rank
  goes up. No new copy.
- Postpartum (I-05): flow counts as a period only from delivery + `postpartum.flow_from_day` (registry, group
  pregnancy, default 42, range 21–84, H-14 advisor-pending; in `coverageDefs`); earlier bleeding stays lochia.
- Breastfeeding (I-04): `PATCH /api/v1/postpartum {breastfeeding: bool|null}`, `breastfeeding` on the profile (OpenAPI,
  optional); «شیر می‌دهی؟» switch card on the postpartum home (fa/en, RTL, tokens only). Screenshots not taken
  (playwright MCP down).
- Goose **00104** `bleeding_log_source` (data only): unconfirmed non-estimated `user_logged` → `bleeding_log`
  confirmed; revisions bumped; Down reverts by source. Int test up/down/up.
- `TestStoreSideEffectsMatchLaravel` `d124` overlay; allow-lists `healthlog.yaml`, `postpartum.yaml` (D-124/D-138);
  translations seed + i18n goldens resynced (main session applied the route and seed diffs).
- Verify (combined): full unit + `make test-int PKG=./...` green; schema-diff OK; contract all 1914/0; frontend 1977
  tests, lint:dark, fsd:lint.
- Open: a closed bridge period shrinks to its last bleeding day even when an end was set; deleting a period doesn't
  reopen the previous one; menopause flow still creates periods; Laravel `DataSource.php` needs `bleeding_log` before
  cutover; bridge revision bumps → T-M9-30; `TestLoad_Budget` timing flake.
- Migration 00104 runs with the next stage/beta deploy.

## T-M9-14b — Record the k6 load-test baseline (before table)
- Ran app_open, calendar_browse, log_burst, analysis_tab and fertility_tab (no soak) at `b08f02c8`, all local:
  `make loadtest-api` on 127.0.0.1:8020, scratch DB `ritme_loadtest` (2000 seeded users), local k6 v2.3.0
  (`K6_RUNNER=local`), Apple M4 / 16 GB at load average 4–7 — read numbers ±30%.
- `docs/engine-v6/perf-baseline.md` "Load-test baseline (before)": client cold/warm + server p50/p95/p99, error rate
  (0% everywhere) and queries per request per endpoint.
- All budgets pass except calendar_browse warm p95 29.7 ms (budget 20 ms; 4.0 queries/request vs target 1). Warm query
  counts above README §8 targets: `/home` 11.1 (≤ 8), `/messages/selected` 6.9 (≤ 4), `/analysis/*` 11 (≤ 6),
  `/fertility/insights` 5 (≤ 4); `POST /health-logs` 13.4. These are inputs for the later perf tasks.
- Script fixes: none needed (POST /health-logs 200/201, `data.notes`, Idempotency-Key, `/cycle/month`,
  `/messages/selected`, `/analysis/*` all as assumed). `grafana/k6:1.3.0` pin still unverified (image not pulled).
- Scratch DB `ritme_loadtest` left in place (`make loadtest-drop` removes it).

## T-M9-30 — Revision bump on every engine-input write, with a manifest test
- New `internal/cycle/revision`: `Bump` / `BumpAnalysis(ctx, DBTX, uid)` over the T-M9-24 queries (request clock).
  Rule: the transaction owner bumps once before commit; helpers on a caller's tx (healthlog on tx, profile `Service`
  inside `EngineTx`, `pregnancy.saveProfile`, periods recompute/reanchor) never do — one write request = `engine_rev`
  +1; a refused or rolled-back write = 0.
- Writers: periods `tx`; healthlog `inTxLocked` (v1/v2 save + delete, bridge included); `fertility.Save`;
  `profile.Service.EngineTx` (new: per-user lock D-131, deadlock retry ×5, joins a caller's tx) for `Save`, onboarding
  steps/complete, life-stage, `SwitchLifeMode`, cycle-settings — an existing-profile save is now transactional;
  postpartum activate + breastfeeding (now tx); `pregnancy.engineWrite` (activate/deactivate/onboarding/update/confirm/
  fetal; profile + EDD in one tx; family-grant reset after commit); contraception set/stop; IVF cycle start; menopause
  profile (now tx); admin user edit; conditions PBAC/pain, loss record, healthrecord re-dating (main session applied
  these three diffs).
- `revision/manifest.go`: 48 sqlc write queries on the README §3 tables → `Bumps: TestBump_*` or `Exempt` (cycle skips
  with no caller yet, dead bridge queries, MarkProfileRecalculated, timezone D-126, baby feeds_count, fetal movement).
  `TestManifest_CoversEveryEngineWrite` parses `db/queries/**/*.sql`; fixture `testdata/queries/fake` proves it fails.
- `calculation_version` / `MarkRecalculated` unchanged. Verify (combined): full unit + `make test-int PKG=./...` green;
  contract all 1914/0; golangci-lint clean after a separate gofmt fix of `labs/ranges.go`.
- Open: `POST /loss` bumps +2 (two transactions); over-bumps on non-engine columns (is_locked, fetal, ivf_iui,
  menopause, name-only profile save); `teen_profiles` isn't in the README §3 table list; raw SQL outside sqlc isn't
  covered by the manifest; no int test for healthrecord re-dating bump; the cycle-skip endpoint task must move
  `InsertCycleSkip`/`DeleteCycleSkip` to Bumps. Using the revision → T-M9-31.

## T-M9-29 — Skipped-cycle endpoint «پریود نداشتم» (D-125)
- Go-only (D-125): `POST /api/v1/cycle/skips {expected_start}` and `DELETE /api/v1/cycle/skips/{date}` (locale → auth →
  per-user write throttle 60/min). Prod's cycle group is still Laravel, so these 404 there until cutover.
- `internal/cycle/skips`: `Check` runs the v6 engine for the user's local today (D-126). Allowed when `expected_start`
  = today's `predicted_next_period_start`, today ≥ it (any lateness) and no real period started in [expected, today];
  else 422 `code` `not_expected_yet` / `period_started` / `prediction_mismatch` (also when the engine would ignore the
  skip). POST 201 `{expected_start}` when recorded, 200 for a repeat (no write, no bump); DELETE 204, 404 for unknown /
  another user's / non-Y-m-d. Validation 422, never 401. Copy in `skips/lang`.
- One transaction: `periods.LockUser` (D-131), write, `revision.Bump`; manifest `cycle.InsertCycleSkip` /
  `DeleteCycleSkip` → `Bumps: TestBump_CycleSkips`. OpenAPI `postCycleSkips` / `deleteCycleSkipsByDate` +
  `CycleSkipResponse`.
- Tests: unit `Check` (expected day, 1/15/60 days late, before, wrong date, real start, pregnancy, no history, chained
  skips); int through the router (bump +1 / 0, delete restores the identical engine status, 401/422, other user → 404,
  8 concurrent taps → 1 row, 1 bump).
- Open: until T-M9-32 the body is `{expected_start}` / empty 204 — switch to the today cycle_view encoder there; no
  contract case (cycle goldens are Laravel); on a declared-only history > 14 days late the roll-forward prediction
  can't be skipped (`prediction_mismatch`) — engine follow-up candidate.

## T-M9-31 — Engine state cache v2 (L1 + L2 + singleflight) and ETag/304 (D-132)
- Auth: `GetAccessTokenWithUser` LEFT JOINs `user_revisions` → `passport.Session.Revisions{Engine, Analysis}` (0/0
  without a row), still **1 query** (int test). Exposing it on the request context (`auth.CurrentRevisions`) needs a
  `guard.go` edit → split into **T-M9-31b** (blocked until the user OKs the auth-guard edit; diff in
  `docs/engine-v6/drafts/T-M9-31b-guard-revisions.diff`). T-M9-32 and T-M10-28 now depend on T-M9-31b.
- `internal/cycle/state`: `Provider.For(ctx, uid, rev, Params{Engine, FP})` — L1 golang-lru/v2 v2.0.7 keyed
  (uid, rev, paramsFP, build), `ENGINE_L1_ENTRIES` default 20 000, shared immutable states; L2 Redis
  `ritme-go:eng:in:{uid}:{rev}` = `inputs.Encode` snapshot, TTL 7 d, Redis error / corrupt / unknown `v` → DB + rewrite;
  singleflight on uid:rev (detached ctx, 5 s); DB loads go through the snapshot format so DB- and L2-built states are
  identical. `CYCLE_ENGINE_CACHE=off` bypasses both; `ENGINE_CACHE_VERIFY` (on under go test) /
  `ENGINE_CACHE_VERIFY_SAMPLE` rebuild on hits → `engine_cache_verify_mismatch_total`. Metrics
  `cache_requests_total{cache=engine_l1|engine_l2}`, `engine_build_seconds{source}`. Config `config.EngineCache`.
- `internal/platform/etag`: `Engine{…}.Tag()` = `W/"u{uid36}.e{rev}-{localToday}-{locale}-{paramsFP}-{cfgFP}-{build}"`,
  weak `Match`, `SetHeaders` (`private, no-cache`, Vary appended), `NotModified` / `Middleware` → 304 before the
  handler; `Build()` = clean VCS revision else executable sha256. Not wired into routes (T-M9-32…35); v1 cache unchanged.
- New env vars: `ENGINE_L1_ENTRIES`, `ENGINE_CACHE_VERIFY`, `ENGINE_CACHE_VERIFY_SAMPLE` (plus existing
  `CYCLE_ENGINE_CACHE`). `go mod tidy` promoted golang-lru, x/sync and a few already-used modules to direct (no version
  changes).
- Benchmarks (M4, loaded): L1 hit 61–69 ns, 0 allocs (≤ 2 µs); L2 hit + Prepare 24 periods 12–14 µs in-memory,
  218–230 µs with docker Redis (≤ 400 µs); 304 path p50 313 µs / p95 465 µs, 1 query (≤ 5 ms).
- Verify (combined with T-M9-29): `-race` state/etag/auth ok; full unit + `make test-int PKG=./...` green; contract all
  1914/0; golangci-lint 0 issues.
- Open: L2 snapshot keyed by rev only while `inputs.Load` bounds markers by today (future markers > 1 day, the −400 d
  edge) — verify mode measures it; Redis-down cold path waits for go-redis retries (consider MaxRetries / breaker);
  `LocalToday` may cost the stored-zone read on a 304; callers must list every config table in cfgFP.

## T-M10-25 — Schema for metric alerts, user activity, segments, reference codes, audit chain and safety runs
- goose **00105** `admin_metrics_audit_chain` + Laravel twin `2026_10_23_000105_create_admin_metrics_audit_chain_tables`
  (schema-diff green, 226 tables; triggers are compared — `mariadb-dump --no-data` includes them, proven with a
  deliberate mismatch).
- New tables: `user_activity_days`, `metric_alert_rules` (5 seeded: `ocr_correction_rate gt .12/7d`, `message_ctr
  drop_vs_mean .30/7d min_volume 200`, `rui_users rising_days 7`, `engine_mae gt 2.5/30d`, `safety_selftest_failed gt
  0/1d` — window days chosen here, T-M10-31 may tune), `metric_alerts` (STORED `open_flag` + UNIQUE(rule_id, subject,
  open_flag) → one open alert per subject), `admin_user_activity`, `admin_segments`, `admin_audit_chain_head` (CHECK
  id = 1, one row), `safety_scenario_runs` (+ KEY run_id). Columns: `users.ref_code` UNIQUE,
  `admin_audit_logs.prev_hash/row_hash`. KPI indexes: `daily_health_logs(log_date,user_id)`,
  `cycle_histories(period_start_date)`, `users(created_at)`.
- `admin_audit_logs` and `clinical_config_approvals` are append-only in the DB (BEFORE UPDATE/DELETE → SIGNAL 45000).
  Writers must compute `row_hash` before INSERT (T-M10-33); neither table may gain an FK whose action updates rows.
- sqlc: new `db/queries/metrics` → `internal/metrics/store`; queries in admin (user_directory, audit_chain), safety
  (runs), auth (ref_code). No writers/readers yet (T-M10-24, T-M10-30…33).
- Main session applied: `contract/fixtures/dump.sql` (new schema + triggers, `users.ref_code`), rbac security int test
  drops/recreates the audit no-update trigger to backdate rows, `profile/model` column test glob fixed (`000*.sql`
  missed migrations ≥ 00100), Laravel `User` hides `ref_code` so prod JSON is unchanged.
- Tested on MariaDB 11.4.13 and MySQL 8.4.13. Verify: schema-diff OK, full unit + `make test-int PKG=./...` green,
  contract all 1914/0, golangci-lint 0 issues.
- ⚠ Deploy: the 4 triggers need the TRIGGER privilege (+ SUPER or `log_bin_trust_function_creators=1` with binlog) for
  the migrating user on stage/beta (and prod via the Laravel twin) — grant first, else 00105 stops after the tables
  (re-run is safe: IF NOT EXISTS / INSERT IGNORE).

## T-M11-02 — Family schema A: link role, template and expiry, grant mode epoch, owner mode state, activity columns (D-190)
- goose **00106** `family_schema_a` + Laravel twin `2026_10_23_000106_add_family_schema_a` (schema only):
  - `companions`: `role`, `template`, `expires_at`, `urgent_sms`, `owner_confirmed_at`; keys
    `(companion_user_id, status, owner_id)` (replaces the prefix key `(companion_user_id, status)`; Down restores it),
    `(owner_id, companion_user_id, status)`, `(status, expires_at)`. `status` / `revoked_by` / grant `section` / `level`
    were already wide enough varchars — no widening.
  - `companion_grants.mode_epoch`; new `family_owner_state` (owner_id PK, mode, mode_epoch, review_pending, updated_at;
    cascades with the user).
  - `companion_audit_logs`: subject_type/subject_id/diff_json/payload_sealed (MEDIUMBLOB)/payload_key_id/undo_deadline/
    undone_at/undone_by/proposal_id + keys `(subject_type, subject_id, created_at)`, `(undo_deadline)`, `(created_at)`.
- Goose backfill (idempotent): `role` from `type` (partner, spouse → partner; parent → teen_parent; F-17),
  `template = 'legacy'`.
- sqlc (companion): `FamilySnapshotForPair`, `FamilySnapshotForMember` (active links ⋈ grants ⋈ owner state ⋈ owner
  `blocked_at`), `UpsertFamilyOwnerState` (epoch +1 and review_pending on a mode change; rows-affected 1/2/0),
  `InsertFamilyActivity`. No readers/writers yet (T-M11-05). `contract/fixtures/dump.sql` updated via temp DB (D-47).
- Tests: `family_schema_a_int_test.go` (up/down/up, backfill per legacy type, idempotency, EXPLAIN on the new keys,
  upsert epoch, activity insert, owner-state cascade). Verify: schema-diff OK (227 tables); full unit +
  `make test-int PKG=./...` green; contract all 1914/0.
- Open: `family.md` §3 says `parent → parent` but F-17/task say `teen_parent` (followed the task) — align the doc;
  sqlc maps `payload_sealed` to `sql.NullString` (T-M11-05 may want a `[]byte` override in sqlc.yaml); links created
  by v1 after Up keep role/template NULL until T-M11-05; D-190 status text to update once shipped.

## T-M11-03 — Family schema B: proposals, share requests, nudges, member alerts, member notices, attendance (D-191)
- goose **00107** `family_schema_b` + Laravel twin `2026_10_23_000107_add_family_schema_b` (schema only; schema-diff
  green, 233 tables; twin also checked on MySQL 8.4.13):
  - `family_proposals` (subject owner|child, nullable link FK, actor/approver FKs, sealed payload, status, `expires_at`;
    STORED `pending_key` + UNIQUE → one pending proposal per dedupe key, decided ones never collide),
    `family_share_requests` (domains JSON), `family_nudges` (UNIQUE owner/target/local_date/slot, named
    `family_nudges_target_slot_unique` for the 64-char limit; slot '' for appointments), `family_member_alerts`
    (UNIQUE member/source_alert_id/source_hit), `family_member_notices` (outbox, UNIQUE member/dedupe_key),
    `family_attendance` (→ reminders).
  - `user_notifications.link_id` + KEY + FK companions cascade — member notices purged by link (FAM-L4).
  - Every link FK cascades: deleting a link removes its proposals, share requests, nudges, alerts, notices and linked
    notifications.
- sqlc (`companion/family.sql`, part B): 37 statements — proposal insert/get (lazy expiry)/lock/lists/quota/decide/
  sweep/cancel-for-link, share requests, nudges, alert/notice idempotent inserts (`ON DUPLICATE KEY UPDATE id = id`,
  FK still enforced), dispatcher reads, purge by link, linked notification insert, attendance. No readers/writers yet
  (T-M11-05, T-M11-08…13).
- Contract dump follows (before→after diff only). D-191 wording updated (no family consent table — `consent_events`).
- Verify: schema-diff OK; full unit + `make test-int PKG=./...` green; contract all 1914/0; golangci-lint 0 issues.
- Open: `family.md` §6 says `due_date`, schema uses `local_date`; `decided_by` / `notification_id` have no FK (spec
  gives none); `payload_sealed` maps to `sql.NullString`; the revoke-path statements are a proposal the behaviour
  tasks must call; pre-existing: Laravel migration 000096 fails on MySQL 8.4 (prod is MariaDB).
- Migration 00107 runs with the next stage/beta deploy.

## T-M11-04 — Family schema C: child guardians, actor columns, child medication log, household to-do scope (D-192)
- goose **00108** `family_schema_c` + Laravel twin `2026_10_23_000108_add_family_schema_c` (schema-diff green, 235 tables):
  - `child_guardians` (level write|limited + sub_scopes JSON, status invited|active|left|removed, `invited_by` FK users
    SET NULL — added beyond the spec so a deleted inviter leaves no dangling id, `link_id` FK companions SET NULL;
    UNIQUE child/user, KEY user/status; child and user FKs cascade).
  - `created_by` / `updated_by` (FK users SET NULL) on children, child_measurements, child_vaccine_doses,
    child_milestone_checks, baby_feeds, baby_sleeps, baby_diapers, todo_tasks, todo_items — not backfilled.
  - `baby_med_doses` (F-26), `todo_tasks.scope` (default personal) + `assigned_to` + KEY (user_id, scope, done_at).
  - `children.owner_id` keeps ON DELETE CASCADE (frozen Laravel deletion; the Go transfer is T-M11-11).
  - Goose backfill (INSERT IGNORE): owner → active write guardian; spouse of an active spouse link → active write
    guardian of each shared `family_children` child (F-17); revoked / invited links give nothing.
- sqlc: children (guardians by child set / by user, get, count active write, upsert, guarded status move), babylog
  (dose-log CRUD + timeline), todo (household board, get, assigned-open list). No readers/writers yet.
- Tests: `family_schema_c_int_test.go` (up/down, idempotent backfill on three fixtures, account deletion nulls actor
  columns and still cascades a sole owner's child, every new statement). Contract dump follows (before→after diff).
- Verify: schema-diff OK; full unit + `make test-int PKG=./...` green; contract all 1914/0. No JSON change (no Laravel
  models for these tables; Go handlers use explicit DTOs). D-192 wording updated.
- Open: T-M11-14 must add `scope = 'personal'` to the personal todo queries before household rows exist (note added to
  its task file); writers don't set the actor columns yet (T-M11-11, T-M11-14); twin not run on MySQL 8.4.
- Migration 00108 runs with the next stage/beta deploy.

## T-M9-31b — Expose engine and analysis revisions on the request context (auth guard)
- User OK'd the `guard.go` edit (2026-10-23). Applied the T-M9-31 draft diff: `auth.Revisions` (alias of
  `passport.Revisions`), `RequireUser` stores them in Locals, `auth.CurrentRevisions(c) (Revisions, bool)`;
  `Authenticate` keeps its signature (private `authenticate` + `tokenOf`). New `revisions_int_test.go` (0/0 without a
  row, 4/6 after an insert). Draft diff deleted.
- security-auditor: nothing blocks — rejection paths, error types and locals unchanged; the LEFT JOIN is on a PK so it
  can't change the authenticating row; tags carry the uid so no cross-user 304. Low: `private, no-cache` allows the
  browser disk cache — note added to T-M9-32 (Clear-Site-Data on logout or `no-store` fetch); Info: don't log ETag
  headers.
- Verify: vet, `-race` auth, int auth + http green; golangci-lint 0 issues.
- Also recorded user decisions: T-M10-19b (move vitals alert copy as-is, no approval flow), T-M9-06c (postpartum
  export without EPDS; extend D-96) — both unblocked; T-M9-09c guard OK.

## T-M9-32 — cycle_view v6 on today/date with CYCLE_ENGINE flag and shadow diff
- Shipped (D-120, D-132): `CYCLE_ENGINE = v1 | shadow | v6` (config, default `v1`, invalid fails at start).
  - `v6`: `GET /cycle/today` and `/cycle/date/:date` answer from the engine state cache (`internal/cycle/service/v6.go`,
    encoder + legacy keys rebuilt in `internal/cycle/view/v6.go`, X-05); ETag/304 (`Cache-Control: private, no-cache`).
  - `shadow`: v1 served unchanged; a bounded pool (2 workers, drop when full, `internal/cycle/shadow`) computes v6
    after the response, classifies into `engine_shadow_diff_total{class}` and keeps ≤1 000 samples in Redis
    (`ritme-go:eng:shadow:samples`, 7 d, uid hashed). `go run ./cmd/api engine-shadow-report [-n N]` prints them.
  - Logout now sends `Clear-Site-Data: "cache"` (T-M9-31b security note: shared-device disk cache).
- Goldens: Laravel goldens unchanged under `v1` (`make contract ROUTES=cycle,period` 211 passed / 12 allow-listed, no
  new allow-list entries — v1 bytes are identical). Go-native goldens: `contract/golden-v6/cycle` (80 cases),
  `make contract-v6` / `make contract-v6-update` (`cmd/contract v6 [--update]`).
- OpenAPI: v6 fields, `today_ui`, `limits`, nullable legacy fields (cites D-120).
- Budgets: v6 today/date cold 9 / warm 1 queries (`internal/querybudget`); one request-scoped recommendation
  repository shared by the ETag and the daily tips (was read twice cold). Local latency: warm p95 0.6 ms,
  cold-after-write p95 3 ms.
- Null in v6 but non-nullable in the current frontend zod schema (pill persona, until T-M9-40):
  `data.calculation.final_probability`, `data.calculation.phase`. Other legacy nulls (pill persona): calculation
  `age_factor`, `base_probability`, `cycle_score`, `estimated_ovulation_day`, `is_luteal_spotting`, `subphase`,
  `symptom_score`, `text_flags`, `uncertainty_range`, `source_*_data`; cycle_view `phase`, `days_to_ovulation`,
  ovulation/fertile dates in `anchors`, `predictions` and `future_cycles[]` (fertility suppressed).
- Open: frontend nullable fields (T-M9-40); `/cycle/month` (T-M9-33), `/home` (T-M9-34); stage switch to `shadow`
  then `v6` is a deploy step (T-M9-48).

## T-M9-36 — Small consumers A on the engine: companion, teen, checkups, cycle settings, challenges
- Shipped (D-136): companion, teen, checkups, cycle settings, challenges and the reminders planner read the v6 engine
  under `CYCLE_ENGINE=shadow|v6`; v1 unchanged (contract companion/companion-home/teen/cycle-settings 62/0/0).
- New shared provider: `state.UserStates` / `state.Source` (`ForUser(ctx, uid, today)`, reads `user_revisions` for the
  user it serves — companion/parent cards read other users) and `engineStates(r, d)` in `internal/http/engine_source.go`
  (nil under v1). Each consumer has `WithEngine(...)`.
- Companion: phase from engine status, cap from `Params`, same eight keys as v1 (pinned by a test), lateness nulls
  `cycle_day`/`days_to_period`/next date. `fertility_suppressed` is only used to map fertile → follicular and is never
  emitted (partners can't infer contraception/pregnancy).
- Checkups: `LoadPlanWith(..., states)`, projections from `FutureCycles`. Reminders: `CycleFrom(*engine.State, today)`
  on `FutureCycles` + `Params.PMSDays`. Challenges: shared `cycleservice.Load` replaces the duplicated loader.
- Edits outside touches (wiring): `internal/cycle/state/source.go` (+ int test), `internal/http/engine_source.go`,
  one-line `.WithEngine` in six `routes_*.go`.
- Open: menopause still calls `checkups.LoadPlan` (v1); `LocalToday` now applies in v1 mode too (challenges, checkups,
  teen); companion uses the partner's zone for the owner's day (decision pending); two L1 state caches under v6
  (routes_cycle + engineStates) to merge later; no dedicated "pregnant, no stored life_mode" fixture.

## T-M9-33 — /cycle/month on the v6 Range: lateness, future cycles, assumed ranges
- Shipped (D-121): `/cycle/month` under `shadow|v6` is built from one `State.Range(first, last, LocalToday)`
  (`internal/cycle/service/month.go`); legacy keys via `calculationV6`; new day fields `main_phase`, `fertility_level`,
  `is_predicted`, `is_period`, `period_kind` (logged|assumed|predicted|null), `days_to_period`, `days_late`; month
  `future_cycles` (k ≤ 6) and `limits`; ETag (`MonthTag`, per month+view) / 304. v1 byte-identical
  (contract cycle,cycle-sweep 227/0).
- Shadow: `ClassifyMonthDay` into `engine_shadow_month_diff_total{class}` and `engine_shadow_month_jobs_total{result}`
  (Prometheus only — not in the Redis samples / `engine-shadow-report`).
- Goldens: `contract/golden-v6/cycle-month/` (19 files), rewritten by
  `UPDATE_GOLDEN_V6=1 make test-int PKG=./internal/cycle/service/...` (not under `make contract-v6` yet).
- Budgets: warm 1 query, warm p95 ~1.4 ms, cold-after-write p95 2.5 ms (≤ 5 queries), cold process 8 queries.
- v6 nulls per month day: `final_probability`, `is_luteal_spotting`, scores, `uncertainty_range`, `text_flags`,
  `source_*`; unknown days also null `phase`/`subphase`/`cycle_day` (T-M9-40/45).
- Outside touches: `internal/cycle/service/handlers.go` (`Month()` v6 branch + shadow submit).
- Open (product question): open-period `menstrual_possible` days up to the hard cap (day 12) are `assumed` +
  `is_period=true` — should the calendar colour them? `onboarding_declared` fixture persona has no anchor in v6.

## T-M9-37 — Small consumers B on the engine: todo, shop, labs, conditions, search, nudges
- Shipped (D-136, I-10): todo, shop checklist, labs, conditions (PMDD/PBAC), condition nudges, search and the
  luteal-spotting warning read v6 via `engineStates` (T-M9-36) under `shadow|v6`; v1 unchanged
  (contract todo,shop-checklist,labs,conditions,search 95/0).
- Shop reminder stores `meta.engine_rev` and re-syncs when the revision moves (query `UpdateSupplyCareReminder` now
  writes `meta`). Labs store the phase of the lab date with `context.phase_on`; prompt key
  `approximate_cycle_phase_on_lab_date`. PMDD/PBAC use the engine's actual periods; current cycle ends at
  predicted_next − 1 (skips respected). Nudges/search: no current-cycle window without anchor/prediction or > 14 days late.
  `todo_suggestion_events.ref_date` unchanged.
- Outside touches: `db/queries/shop/checklist.sql` + `internal/shop/store`, `labs/interpret.go`,
  `healthlog/service.go`, route wiring in six `routes_*.go`.
- Open: `labs.phaseOn` kept on the v1 path (remove in T-M9-51); `PUT /fertility/days` still uses the v1 spotting rule
  (fertility's `saveHealthLog` not wired — T-M9-35); nudges under v6 lose the 28-day fallback for users with no anchor;
  shop stored-row shortcut keys on `engine_rev` only; `currentCycleStart` (search) and `EngineCycleWindow` (nudges) are
  near-duplicates.

## T-M9-34 — /home, cycle-overview and the daily card on the v6 engine
- Shipped (D-122, I-08): under `CYCLE_ENGINE=shadow|v6`, `/home` and `/home/cycle-overview` read the shared engine
  state (`engineStates`; engine_rev from auth → no query on an L1 hit) on `LocalToday`; v1 unchanged (contract home 76/0).
  - week_calendar from `State.Range` (+ main_phase, fertility_level, period_kind, is_predicted, days_late); next_period
    with days_to_period/days_late (late title); cycle_prediction from `today_ui.prediction_card` (null under
    suppression; + pms_window, cycle_regularity); challenges use the v6 cycle day (no challenge past the last
    day-targeted `cycle_day_to` — admin validator should warn); `HasCycleData` = engine anchor.
  - cycle_summary health check on clinical flags / v6 regularity; digest from actual periods with §2.8 bounds (v1
    21–35 / 2–8 constants stay for v1 until T-M9-51). smart_tip falls back to `DailyCardV6`'s subtitle.
  - cycle-overview: ETag (engine rev, local day, locale, params FP, Plus banner crc, build) / 304.
- Goldens: `contract/golden-v6/home` (9 personas × index fa + cycle_overview en), rewritten by
  `UPDATE_GOLDEN_V6=1 make test-int PKG=./internal/home/...`.
- Budgets: home v6 cold 22 / warm 9 (v1 20/11), cycle-overview v6 11/5. **Acceptance "warm /home ≤ 8" not met (9)** →
  follow-up T-M9-34b (merged challenge-completion + reminder queries).
- Copy (main-session review): late title «پریودت N روز دیر کرده», subtitle «تاریخ پیش‌بینی‌شده DATE بود»,
  regularity hint uses «چرخه» (glossary), flag labels «کوتاه‌تر/طولانی‌تر از محدودهٔ معمول».
- Outside touches: `internal/http/routes_home.go` (`.WithEngine`), `internal/querybudget/budgets.go` (2 rows).
- Open: overdue beyond max_late still shows the past prediction card (product call); fixture persona
  `onboarding_declared` has no v6 anchor (dump predates goose 00103?).
- Local env note: the macOS SDK 27 linker breaks cgo (`tapi ... unknown architecture`); run Go tests/contract with
  `CGO_ENABLED=0` until Command Line Tools are fixed.

## T-M9-42 — Today screen on today_ui: ring actions, one banner, lateness tiers, disclaimer
- Shipped: `screens/home/model/today-ui.ts` (`todayHero`, `latenessTier` 1–7 calm / 8–14 overdue / 15+ ask,
  `bannerActions`), `ui/TodayBanner.tsx` (one banner: end_missing | overdue | ask_last_period), fixed disclaimer
  «پیش‌بینی روش پیشگیری نیست» on PredictionsCard; no client roll-forward while `days_late > 0`; `unknown` phase shows the
  neutral ring. Laravel payload keeps the old path (tested).
- Copy reviewed in the main session; added `nb.banner.overdue.textNoTest` / `textLongNoTest` so the overdue banner
  never mentions pregnancy when the test button is hidden (teen, menopause, suppression). `ring.noData` shortened to
  «پریود ثبت نشده» (ring label ≤ 14 chars, pinned by `ring-copy.test.ts`).
- Open → T-M9-42b: pregnancy-test deep link into the log sheet's measurement panel, banner CSS, legacy schedule
  helpers, light/dark screenshots.

## T-M9-45 — Calendar on v6: future cycles, lateness, assumed ranges
- Shipped: month schema parses the v6 per-day fields, `future_cycles` and `limits` (Laravel month parses as before;
  unplaced null-`cycle_day` days go to a new `days: CycleMonthDay[]` list); painting from the server
  (`entities/cycle/model/monthPaint.ts`): logged solid, assumed + predicted soft, late day outlined, fertile/ovulation/PMS
  from per-day fields, `future_cycles` day map (k ≤ 6) as fallback; suppression hides fertility + ovulation dot.
- Deleted (no importers): `phase.ts`, `highlights.ts`, `fertilityStyle.ts`, `ui/DailyStatusCard.tsx`,
  `markerIntensityByDate`, `cycleProgressPercent`, `fertileWindowDays`, `calcToPhase` (+ tests).
- Touches extended with `entities/cycle/api/schema.ts` and `entities/cycle/index.ts` (the parser dropped every v6 field).
- New copy: `calendar.nb.legend.late` «دیرکرد پریود» / "Late period". i18n goldens
  (`internal/i18n/testdata/messages_*.json`) re-recorded with `-update` for the home + calendar copy.
- Open: `cycleScheduleFor` / legacy `cycleDayMarkerAt` stay while home imports them (T-M9-42b); late style inline
  until globals.css (T-M9-42b); assumed days look the same as predicted (product question); screenshots (T-M9-42b).

## T-M9-35 — Fertility, analysis and health record read ovulation and regularity from the engine
- Shipped (D-128, D-129, A-01, E-09) under `CYCLE_ENGINE=shadow|v6`; v1 unchanged (contract
  fertility,analysis,analysis-ttc,healthrecord 54/0/0).
  - Fertility: today/days/bbt/insights from `State.Resolve`/`Range`/`BBT()`; BBT chart on the engine rule
    (`bbt.BuildRule`, `engine/bbt.Detect`, provisional coverline until confirmed); per-cycle ovulation + source from engine
    anchors (BBT > LH > calendar; conflict shows BBT); insights ≤ 1 Range per cycle, 3 queries warm; ETag/304 on
    today/bbt/insights; `PUT /fertility/days` now uses the v6 luteal-spotting rule (T-M9-37 open item closed).
  - Analysis: verdict = engine `cycle_regularity` (incl. `insufficient_data`), no `markOutlier` (E-09), cycles = engine
    real periods (`excluded: skipped`); TTC ovulation engine marker → chart shift → LH → calendar.
  - Health record: v6 `last_period_start` = latest actual start; adds `effective_cycle_length`/`effective_period_length`.
- Goldens: `internal/{fertility,analysis}/testdata/golden-v6`, rewritten with `UPDATE_GOLDEN_V6=1 make test-int`.
- Outside touches (added to the task): seven one-line `.WithEngine(engineStates(r, d))` in `internal/http/routes_*.go`.
- Open: analysis median/`counted` still window-based (21–45) while regularity counts outliers; `CycleStatus` not yet
  tied to `clinical_flags.frequency`; FIGO age band copied into analysis (needs a public engine helper);
  `querybudget` fertility bbt row still has v1 numbers; `onboarding_declared` fixture has no v6 anchor.

## T-M9-34b — Home v6 warm query budget to 8
- Shipped: `db/queries/home` `ListChallengeCompletionsSince` (window incl. today) and `ListHomeRemindersV6`
  (ROW_NUMBER per type, v1 ordering kept), used on the v6 path only and memoised per request; v1 unchanged (20/11).
- Budget `home v6`: cold 22 → 21, warm 9 → **8**. `TestMergedReads_MatchV1Queries` pins merged = v1 results.
- Verify: contract home 76/0, contract-v6 80/0.

## T-M9-42b — Today banner follow-ups
- Shipped: overdue banner «ثبت تست بارداری» opens the log sheet on the measurement panel (`openSheet('log',
  'YYYY-MM-DD:pregnancy_test')`, parsed in `LogSheet`, `LogDay focus`); banner buttons, `.ch-pred-disclaimer` and the
  calendar late-day style moved to `globals.css` (tokens, light+dark); `lateStyle.ts` deleted.
- Moved to T-M9-42c: legacy schedule helpers (home week strip / fertile / ovulation / PMS slots, CalendarPage,
  ReadyStep) and the light/dark screenshots.

## T-M9-38 — Messages read v6 facts (daily, selected, admin preview)
- D-130, I-11: `/messages/daily`, `/messages/selected` and the admin rule preview read the v6 engine (`engineStates`) under shadow|v6. New `manager/v6.go` (`EngineSource`/`EngineDay`, `EngineFacts`, `cycleContextV6`, `neutralBase`); `source.go` `WithEngine(states)` loads engine state once per request for the user-local day.
- New facts: main_phase, subphase, phase, cycle_day (no wrap), cycle_length, days_to_period (0 when late), days_late, data_quality, confidence, suggest_pregnancy_test, fertility_suppressed, is_pms_window, is_fertile_window. Fertile = O−5..O (O+1 never). Suppression/teen/menopause strip fertility (ovulation→follicular, `user_goal=non_ttc`); `suggest_pregnancy_test` false in teen/menopause. `unknown` day with data → neutral default copy, no supplements.
- Rule variables: `pms_start_day`, `top_symptoms` removed (never filled), `days_late` added — admin meta endpoint list changes.
- v1 path unchanged (contract messages 117/0, no allow-list change); `engineProfile`/`historyFromRow` replaced by `cycleservice.ProfileFromRow`/`HistoryFromRow`.
- v6 goldens: `internal/messages/manager/testdata/golden-v6/` (rewrite with `UPDATE_GOLDEN_V6=1 go test ./internal/messages/manager/`). Agreement test daily phase == hero phase, 8 personas × 60 days.
- Wiring outside the original touches (added to the task's `touches`): `messages/handlers.go` (`WithEngine`, `civildate.LocalToday`), `http/routes_messages.go`, `http/routes_admin_content.go`, `messages/v6_int_test.go`.
- Open items → T-M9-38b: home `msgSource` still legacy, 400 without `last_period_start` under v6, admin preview uses Tehran time, query budget, product review of late=luteal / pill+TTC=non-TTC / teen+meno fertility_suppressed.

## T-M9-42c — Home timeline on v6 fields, delete legacy schedule helpers, Today/calendar screenshots
- HomePage on a v6 payload (`isEnginePayload`): no browser schedule; week strip/ring dots from `cycle/month` per-day fields (`monthDayPaint`, `futureCycleDayMap`); ovulation/fertile slots from new `entities/cycle/model/serverEvents.ts` (`serverFertileEvents`, first server window not over); PMS from cycle-overview; next period from overview or `anchors.predictedNextPeriodStart`; countdown from `cycle_view.days_to_period`. No roll-forward on v6.
- CalendarPage: `legacySchedule` null on v6 month payloads. ReadyStep: new `EngineRows` on v6; `hasFertileWindow`/`deriveCycleSchedule` only for Laravel.
- Nothing deleted: `cycleScheduleFor`, `daysUntilNextPeriod`, `hasFertileWindow`, `cycleDayMarkerAt` still have Laravel-branch importers (kept until T-M9-51).
- Frontend does not fetch `/home` (`week_calendar`/`cycle_prediction` unparsed — would need `entities/cycle/api`); equivalent server fields from cycle/month, cycle_view, cycle-overview used instead.
- Screenshots reviewed in the main session (late 10 overdue banner + pregnancy-test link, open period day 11 end_missing banner, regular calendar). Gaps → T-M9-42d. Found bug: v6 `/cycle/today` sends `final_probability: null`, zod rejects → Home load error → T-M9-40b.
- Local note: screenshot personas needed `cycle_histories` shifted +34 days and terms/health consents inserted; `contract/fixtures/dump.sql` has no `app_modules` table (Go warns but runs).

## T-M9-40b — Frontend: nullable final_probability (and other null v6 fields) in cycle/today schema
- Evidence: all 17 `backend-go/contract/golden-v6/cycle/today.*.en.json` personas send `calculation.final_probability: null`; `overdue_20` (and no_profile, pregnant…) send `calculation.phase: null` — that was the late-15+ Home error, not final_probability.
- `schema.ts`: `phase` `z.string().nullish()` → `''`; `final_probability` `z.number().nullish()` → `NaN` ("no number", never 0%). Other null fields were already nullable or are not in the schema. New "v6 nulls" tests on the overdue_20 payload.
- No UI reads `fertilityPercent` today. Sentinels → real nulls in T-M9-40c (needs `model/types.ts` + `predictions.ts`).

## T-M9-38b — Messages v6 follow-ups
- `home/context.go` `msgSource` implements `manager.EngineSource` from `hc.v6` (same state + user-local today as `/messages/daily`, no second engine read). Int test `TestHomeV6_MessageEqualsDaily` (47 persona-days) proves Home smart_tip == daily message.
- `/messages/daily` under v6: 400 only when the engine has no anchor (`hasCycleAnchor` → `EngineDay.HasAnchor`), so logged periods with empty `last_period_start` now get 200. v1 unchanged.
- `(*Handlers).UserToday`: v6 uses the user's stored zone (`usertz.ZoneOf`, Tehran fallback); admin preview uses it.
- Query budgets: `messages daily v6` cold 15 / warm 8, `messages selected v6` cold 14 / warm 7 (warm = v1); rows added to `docs/engine-v6/perf-baseline.md`.
- Golden `contract/golden-v6/home/index.overdue_20.fa.json`: smart_tip «…تاریخ آخرین پریود خود را وارد کنید» → neutral «هر روز فرصتی برای مراقبت از خودته.» (regenerated by main session).
- Contract v1: messages 117/0, home 76/0.
- Open: low-energy day in cycle mode still 500s (D-05 `ErrUndefinedOverrideCase`, persona 1013 on 2026-09-10; Home swallows it). `-run TestHomeV6_Goldens` alone shows an `art-2` article diff that the full-package run doesn't (pre-existing test-DB order issue). Product calls from T-M9-38 still await the user: late day = luteal content; teen/menopause fertility_suppressed; pill + TTC goal = non-TTC copy.

## T-M9-46 — Remaining browser cycle math, clinical-flag hints and "normal" copy audit
- `screens/cycle/model/bars.ts`: fertile O−5..O, PMS last 3 days. `companion-home/model/phase-bar.ts`: optional `{periodLength, ovulationDay, fertilitySuppressed}`; fertile never starts inside the period; suppression drops fertile+luteal; old fallback without engine values. `phase-details/model/tabs.ts`: ovulation from the engine anchor only (no L−14). `predictions.ts`: `normalizePhase('unknown')` → null, `calcMainPhase` nullable; unknown day never fertile.
- §2.36 table (21/28/35, derived from O = L−14 — SPEC has no literal table) tested in each of the 4 test files (copies, FSD forbids a shared cross-screen test).
- Flag hint on `/cycle` and `/analysis/cycle` from `cycleView.clinicalFlags` (absent > frequency > duration; nothing for normal/unknown/Laravel). New copy `flagHint.*` (cycle) and `reports.cycle.flagHint.*` (analysis), non-diagnostic, reviewed in the main session. `flag-hint.ts` duplicated in both screens on purpose (FSD).
- "Normal" copy reworded to «در بازهٔ معمول» / "In the usual range"; backend-go translations kept identical.
- Main session: deleted dead `entities/cycle/api/sections.ts` (no importer of hooks or types) and its exports in `entities/cycle/index.ts` (added to touches).
- Open → T-M9-40c: unknown day shows a 'follicular' placeholder (types non-null); companion bar still uses the fallback (`SharedCycleView` lacks the fields).

## T-M9-40c — Frontend: real nulls for unknown phase and missing fertility number
- `model/types.ts`: `CycleCalculation.phase` `string | null`; `fertilityPercent` (calculation + predictions) `number | null`; `CyclePredictions.phase` `CyclePhase | null`.
- `predictions.ts`: `normalizePhase` accepts null; `markerPhase` → null for an unknown day (no 'follicular' placeholder); `deriveCyclePredictions` passes nulls through (no NaN, no 0%). `schema.ts` sentinels (`NaN`, `''`) replaced with `?? null`.
- No consumer outside touches broke (CalendarPage/HomePage already tolerate null).
- Open: companion bar keeps the L−14 fallback — `SharedCycleView` (`entities/companion`) has no periodLength/ovulationDay/fertilitySuppressed; needs the companion payload + schema extended (not queued yet).

## T-M9-43 — Prediction card v6: ovulation source, regularity badge, engine ranges
- New `screens/home/model/prediction-card.ts` `enginePredictionRows(todayUi)` maps `today_ui.prediction_card`; when present it owns period/PMS/fertile/ovulation slots, cycle length and regularity on Home; null card → old Laravel path.
- `PredictionsCard.tsx`: `ovulationSource` (BBT/LH-confirmed shows the plain date, no «حدود») and `hideWindow` (fertile row hidden when null).
- `entities/cycle/api/overview.ts`: `normalizeRegularity()` maps v6 + old overview enums to `regular | irregular | insufficient_data` (unknown → no pill).
- Copy `home.nb.predictions.*`: sourceBbt «تأیید با دمای پایه», sourceLh «بر اساس تست LH», regularity منظم / «کمی متغیر» (was «نامنظم», calmer) / دادهٔ کم. Old `relatively_regular`/`irregular_possible`/`not_enough_data` keys kept (copy:check fails on removed keys) but unused.
- Visual QA (3 states, light + dark) moved to T-M9-42d.

## T-M9-39 — Engine daily snapshots and prediction log
- D-137, A-05, A-06. Go-only analytics tables: goose `00109_engine_snapshots` + Laravel twin `2026_10_27_000109_create_engine_snapshot_tables` (schema-diff green, 237 tables). `engine_daily_snapshots` (PK user_id+day, KEY day; codes only: phases, confidence, data_quality, warnings_mask, days_late, mode, engine_rev) and `engine_prediction_log` (UNIQUE user_id+cycle_start+kind, kind first|last). Both cascade on user delete.
- New package `internal/cycle/snapshots`:
  - `engine:snapshot_nightly` (queue low, 03:30 Tehran, `Unique(23h)`): active users (A-06: token issued/refreshed, `user_activity_days`, daily/v2 log, period row or analytics event in the last 30 days), pages of 500, grouped by each user's stored zone (Tehran fallback), `inputs.LoadMany` + `Prepare` + `Resolve`, idempotent upsert in one transaction.
  - `engine:snapshot_purge` (03:50): keeps 400 days. Prediction log is kept forever.
  - `engine:prediction_log`: queued by the new `periods.StartHook` (`WithStartHook`) after a committed real start in `Start`/`Store` (new row or estimate/seed promoted). Payload = ids + `rev_before`/`rev_after`. Records `last` (state without the new start, real anchor only), fills `actual_start`/`error_days`, then `first` for the new cycle. Edits, backfills, estimates, the declared anchor and skips never resolve a cycle.
- Wiring (main session): `internal/http/routes_cycle.go` registers the jobs and the start hook when a DB is present. In sync queue mode the job runs inline after the write (responses unchanged; contract cycle 125/0).
- `make sqlc` added the two table structs to every `store/models.go` (additions only).
- Catalogue rows in `docs/engine-v6/jobs.md`. No health data in logs.
- Open: bleeding-log bridge starts not logged → T-M9-39b. T-M10-30 computes MAE and must exclude pregnancy/hormonal/postpartum via `engine_daily_snapshots.mode` (prediction log has no mode column).

## T-M9-39b — Prediction log: count real starts from the bleeding-log bridge
- `healthlog.Service.WithStartHook(periods.StartHook)`: `inTxLocked` (service-owned tx only) records a bridge start, reads `engine_rev` after `revision.Bump` in the same tx and fires the hook after commit with ids + `rev_before`/`rev_after` only. A caller-owned tx (`NewService(tx)` from fertility/conditions/healthrecord) never fires it.
- `bridge.go`: only a new `bleeding_log` row from `addBleedingDay` is a new start; moving a start earlier, extend, merge and split-on-delete are not.
- Wired in `routes_healthlog.go` (v1 `/health-logs`) and, by the main session, `routes_logs.go` (v2 `PUT /logs/days/:date`). The job handler stays registered by `routes_cycle.go` on the shared queue.
- Int test `TestBridgePredictionLog_NewStartsOnly`. Contract `healthlog` 45/0, `logs` 34/0 (task verify fixed: group is `healthlog`, not `health-logs`).
- Open: flow written through a caller tx (fertility/conditions/healthrecord) isn't logged; moving the newest start earlier leaves the old `first` row (same as `periods` Update).

## T-M9-42d — Today/calendar v6 screenshot gaps
- 24 screenshots (light + dark, 390x844) + README table in `docs/engine-v6/qa/T-M9-42d/`: overdue 15+ (persona 1011, no zod error now), ask_last_period (1002), unknown ring (combined pill), overdue banner tap → log sheet on the pregnancy-test chip, calendars (pill, onboarding-only, overdue 15+), prediction card calendar / LH-confirmed / suppressed with regularity badge. BBT-confirmed not shot (same `ovulationSource` path as LH, label only).
- No source change. Reviewed in the main session.
- Bugs → T-M9-52 (stale anchor shows «پریود · روز ۲» next to the ask_last_period banner; onboarding_estimate row → unknown) and T-M9-42e (phase card header «—» when phase is null).
- QA notes: the engine caches by `user_revisions.engine_rev` — bump it after hand-editing fixtures; `send-otp` is 5/min per IP.

## T-M9-44 — Limits on Home: hidden fertility, withdrawal bleed, postpartum, 45+, pregnancy prompt
- New `screens/home/model/limits.ts` `homeLimits(view)` → `{hideFertility, withdrawalBleed, note: hormonal|postpartum|age45|null, promptPregnancy}` from `limits.suppressionReason` + warnings. HomePage: `isTtc = ttcProfile && !hideFertility` (drops TTC chance card, tiles, LH tip, ovulation ring), fertility pill hidden, `PredictionsCard` gets `withdrawalBleed` (label «خون‌ریزی قطع مصرف») and `note` line.
- New `PregnancyPromptCard` («رفتن به حالت بارداری» → `/profile/mode`), today only, dismissible for the day (`PromptKind 'pregnancy'`).
- ReadyStep hides the fertile row for a hormonal pick (`onboarding-flow/model/hormonal.ts` `usesHormonalMethod`; `iud` can't be split copper/hormonal at onboarding, engine hides later). Notification settings hide the `fertile_window` switch while suppressed.
- Copy (main session): `home.nb.predictions.withdrawalBleed`, `.note.{hormonal,postpartum,age45}`, `home.nb.limits.pregnancyPrompt.{title,body,cta,dismiss}`.
- QA: pill + postpartum light/dark reviewed (no fertile UI for pill). Pregnancy prompt and age-45 covered by unit tests only.
- Follow-ups: T-M9-42e (hero shows a huge «▬» when the countdown is null), T-M9-53 (postpartum shows «روز ۱۶ چرخه» 8 days after birth; server-side fertile_window push skip).

## T-M9-52 — Engine: stale anchor painted a period day next to ask_last_period
- Case 1 (QA persona 1002, 57-day declared anchor): `engine/anchors.go cycleAt` rolls a > 42-day anchor forward by n·ECL (§2.41), and `resolve.go menstrual()` then painted days 1..EPL of that projected start as `menstrual`, so Today showed «پریود · روز ۲» beside the ask_last_period banner. Fix: with `stale_onboarding_anchor` and `futureK == 0`, days ≤ EPL resolve `unknown/unknown` (§2.20 priority 1 needs a real or open period; §2.43). Prediction card, ring `edit` button, later days and `future_cycles` (E-03) unchanged.
- Tests: `TestStaleAnchor_NoPeriodDayUnderAskLastPeriod` (days 43–200), int check in `TestLoad_OnboardingOnlyUserIsDeclaredThenStale`; engine persona golden `onboarding_stale.golden.jsonl` (4 lines). No contract golden had a stale anchor. Contract v1 cycle 125/0, v6 cycle 80/0.
- Case 2 (persona 1003, onboarding-only resolves `unknown`): caused by fixtures in the pre-I-01 shape (estimate row, NULL column). The agent's engine fallback was reverted on the user's decision (keep I-01) → T-M9-52b fixes the fixtures.
- Frontend follow-up folded into T-M9-42e: ring must show the neutral state for `unknown` with a cycle_day.

## T-M9-42e — Home: neutral states instead of dashes (phase header, null countdown, unknown day)
- Phase card header with a null v6 phase: «روز n چرخه» (existing `ring.cycleDay`) when a cycle_day exists, else «امروز» / «این روز» (`home.phaseCard.title.neutral`, `.neutralDay`); no phase dot.
- Hero ring with a null countdown (`noCountdown`: not loading, not in period, not late, not TTC ring, `daysLeft == null`): grey overline «پیش‌بینی پریود» / "Forecast" + «فعلاً نیست» / "Not yet" (`home.ring.noPrediction`, `.noPredictionValue`) instead of «پریود بعدی» + a huge «▬».
- Stale anchor `unknown` + cycle_day already never shows «پریود · روز n» (`phaseFromMainPhase` → null); confirmed on screenshots.
- QA light/dark (pill, postpartum, stale anchor) reviewed in the main session. Laravel path unchanged.
- Open: postpartum ring still paints projected period dots and «روز ۵۰ چرخه» (T-M9-53); engine daily-card copy «به سلامت خود توجه کنید» uses the formal register (message content, not this task).

## T-M9-47 — «پریود نداشتم» skip flow and the last-period prompt end to end
- New slice `features/skip-period`: `useSkipPeriod` / `useUndoSkip` (refetch via `invalidateCycleDependents`; 422 `code` → calm messages for not_expected_yet / period_started / prediction_mismatch, plus network/generic; undo 404 = already undone), `canOfferSkip` (v6, today, not in period, not TTC ring, today ≥ predicted_next, overdue banner or ring `start`; hidden under ask_last_period), zustand store keeping the undo date for the day, `SkipPeriodSheet` (confirm) and `SkipUndo` (8 s inverse toast above the nav + a quiet note until midnight).
- Entry points: overdue banner link (`TodayBanner onSkip`) and a third button next to «هنوز نه» in the ring start prompt.
- `frontend/steiger.config.ts`: `features/skip-period` added to the "consumed by screens" block (main session; added to touches).
- Copy `home.skip.*` reviewed; main session simplified three strings (no nested «», en "Period didn't come?").
- Checked end to end on local Go v6 (persona 1010, 10 days late): skip from banner and ring → banner gone, predictions +1 cycle, toast + note; undo → banner back. API: 201 skip, 204 undo, 422 prediction_mismatch. Light/dark shots reviewed.
- Not covered end to end: ask_last_period → PeriodDateEditor → banner gone (existing wiring).
- Incident: the run filled the disk (ENOSPC) and Docker hung; `ritme_dev` may still exist in the test-stack MariaDB until Docker is restarted and it is dropped.

## T-M9-09c — Warm the OAuth client cache once per process
- `passport.(*Verifier).WarmClients(ctx)` runs one `ListClients` and fills the cache under the usual TTL.
- `auth/guard.go` keeps one shared `*passport.Verifier` per `*fiber.App` (in `app.State()`, same pattern as `auth.Jobs`), warmed once with a 5 s timeout; on failure it logs and falls back to lazy load; skipped when `db == nil`.
- No snapshot wrapper; the client revocation bound stays ≤ 5 min (D-140). Security review: no critical/high/medium findings. One low, pre-existing item (a stale client snapshot can outlive the 5-min bound) → T-M9-09d (needs user OK).
- Verify: `go vet` + `go test ./internal/auth/...` green (`-race` needs CGO, which the macOS SDK linker bug blocks); `make contract ROUTES=auth` 45/45.

## T-M9-52b — Contract fixtures in I-01 shape
- `ContractFixtureSeeder.php` + `contract/fixtures/dump.sql`: the declared LMP lives only in `user_profiles.onboarding_last_period_start`; no `onboarding_estimate` cycle_histories rows (mirrors the promote migration).
- v1 goldens re-recorded (only history row ids move: insert instead of promote); contract all 1914/0, no new allow-list entries. Case `period/start_onboarding_estimate` renamed to `period/start_on_declared_lmp`.
- v6 goldens (cycle, cycle-month, home, fertility) regenerated: `onboarding_declared` now gets a declared anchor instead of `insufficient_anchor_data`. Analysis goldens unchanged (no confirmed cycles).
- `periods/testdata/sequence/laravel.jsonl` re-captured (step 13 inserts); `docs/go-migration/contract.md` persona table updated.
- Verify: contract cycle 125/0, contract-v6 80/0, test-int cycle/fertility/home green.

## T-M9-53 — Postpartum before cycles return: no cycle day
- `engine/resolve.go`: under `postpartum_not_returned` (E-14), like pregnancy (E-25), `cycle_day` is set only on logged bleeding days. `view/v6.go`: the "predicted cycle day" daily card is hormonal-contraception only (E-13), so postpartum shows no «روز N چرخه» / «یک روز معمول چرخه».
- Tests: invariant `noCycleSuppression`, postpartum QA persona (birth 2026-10-20) in engine and view (fa/en); persona golden `postpartum_breastfeeding` regenerated (only `cycle_day` changed).
- No `fertile_window` push sender exists (reminders planner has no production caller), so nothing to skip.
- `docs/engine-v6/engine.md` invariant + E-14 row updated. Follow-up T-M9-53b (frontend ring). Open: cycle day after the first postpartum period (E-14 needs 2).
- Verify: vet, unit, test-int cycle+notifications green; contract cycle 125/0 (12 allow-listed, unchanged); contract-v6 80/0.

## T-M9-53b — Postpartum ring
- Not reproducible after T-M9-53: the v6 ring reads `cycle_view.cycle_day` (null) and draws no dots or arc. Light/dark QA shots in `docs/engine-v6/qa/T-M9-53b/`. Only a postpartum regression test added to `entities/cycle/model/schedule.test.ts`.
- Verify: typecheck, vitest entities/cycle + screens/home (182 tests), fsd:lint, lint:styles, lint:dark green.

## T-M11-05 — family.Authorizer
- New `internal/family`: domain registry (roles, caps, defaults, templates, carer expiry), pure `Decide` (10 ordered steps, deny by default; ~9 ns/op, 0 allocs; 177k-row oracle matrix), per-request `Loader` (one statement per pair), lazy `SyncMode` epoch, `RecheckTx` (`LOCK IN SHARE MODE`), `Grants.Apply` (registry caps → 422 `ErrLevelNotAllowed`, epoch stamp, activity `diff_json` + `family_sharing` consent event in the caller's tx), Redis read-audit coalescing (`family:ra:*`, 15 min, DB fallback).
- `GET /api/v1/family/me` (private, no-store). Companion/teen/delegation/shop now decide through `family`; `/companions/*` keep their shape and `section_not_shared` (D-204). Consent catalog: `family_sharing` v1.
- Migration `00110_family_grants_v6` (data only, no Laravel twin): legacy grants → v6 codes/levels, symptoms rows deleted, epoch 1 on owner-data grants, pregnancy grants of non-pregnant owners NULL.
- Care member writes (meds, appointments) run in a tx with `RecheckTx` + re-decide (revoke during a request wins).
- Security review fixed: teen beats an active pregnancy (H1); RecheckTx wired (M1); accept keeps invite-time epochs so a mode change before accept stays paused (M2); teen domains teen-only (L1); unchanged levels keep their epoch unless the owner re-approves (L2); only legacy shapes count as epoch 1 (L3); v6 domain in the Redis key (L6). M3 (mode-write hooks, A→B→A) and L5 (loss notice recipients) moved to T-M11-07.
- Behaviour change (F-23): any owner mode change pauses owner-data grants until she re-approves (`PUT /companions/{id}/grants`); the review card comes in T-M11-07.
- Not wired here: `children.Access` → Decide; `/family/me` contract case.
- Verify: vet, golangci-lint, schema-diff, unit + test-int family/companion/care/http/shop green; contract companion,companion-home,reminders,shop-checklist,teen 104/0 (58 allow-listed); partners,pregnancy,loss 252/0. `-race` skipped (CGO).

## T-M9-06c — Postpartum data export and dated red-flag thresholds
- Account export gains a tenth Go-only section `postpartum` = `{profile|null, recovery_logs}` (`db/queries/postpartum/export.sql`, scoped by `user_id`); EPDS deliberately excluded (D-54). Allow-listed under D-96 (`data.postpartum` on `export*`); profile goldens untouched (recorded from Laravel).
- Daily message: postpartum red flags use the threshold version in force on the log date (`Recovery.AlertsOn(date)`).
- Tests: `TestPostpartumExport_OwnerRowsOnlyNoEPDS` (other user's rows never appear, no EPDS keys), `postpartum_source_test.go` (fever threshold by date).
- Verify: accountexport, messages, postpartum, http export, migrations (Postpartum) green; contract profile,postpartum 136/0 (107 allow-listed). The full `go test ./...` was red only on T-M10-26 in-flight analysis code at the time.

## T-M10-26 — Analysis methods per SPEC §4.1 (D-175)
- `internal/analysis/params.go`: one documented `P` table of every §4.1 minimum/method; boundary tests per minimum. Below a minimum: `ready:false` + `days_needed`/`readings_needed`/`reports_needed`, and a "not enough data" line in the UI.
- Correlations: rate ratio, ≥30 paired days; strong = 30 days and ≥2×, medium 1.5×, weak 1.2×; not-causal note on every correlation and the top finding. Mood by phase (≥10 mood days per phase, ≥2 phases). Top finding = strength × recency (half-life 30 days). Weight ready with 3 logs in 14 days; vitals with 2 readings on 2 days; monthly with 10 logged days; BBT with 10 temperatures; regularity on 3 cycles within 12 months.
- New `GET /api/v1/analysis/labs?marker=` (Plus): direction-only lab trend with a non-diagnostic note, needs 2 confirmed reports.
- Contract: new allowlists `analysis.yaml`, `analysis-ttc.yaml` (D-175). Goldens re-recorded (analysis v1 + v6 ttc). `internal/i18n/testdata` bundle goldens re-recorded after the analysis copy change; `make contract ROUTES=public` 26/0.
- Frontend: entities/analysis (lab trend API/model), hub/correlations/body/monthly/labs/ttc screens; copy fa+en synced.
- Follow-ups: T-M10-26b (pregnancy weight minimum, menopause rate ratio, top finding on TTC/pregnancy hubs); EPDS minimums noted on T-M10-27.
- Verify: go vet, analysis unit + test-int, contract analysis* 48/0 (155 allow-listed), frontend typecheck/lint/fsd/dark/test (2082)/copy:check green; full `go test ./...` green.

## T-M10-02 — Message engine schema v6, versioned approval and message authority (D-160)
- Migration goose `00111_message_engine_v6` + Laravel twin `2026_10_29_000111_add_message_engine_v6_schema`: `message_rules` v6 columns (`rule_key` UNIQUE, `approved_version`, `topics`, `caps`, `variants`, `starts_at`/`ends_at`, `hour_windows`, `goal_event`, `locked`); `message_events.version/slot`, variant varchar(16); new `message_event_daily`, `message_inbox` (generated `open_flag`, one open item per rule); `challenge_programs.contraindications/start_phase`. Goose-only data step (M-18): `daily_advice`+product → content, `modal`+product → marketing.
- Versioned approval (M-02/M-03): the served version is the approved snapshot; an edit to a medical rule goes to `pending_approval` while the old approved version keeps serving. Authority: `CapClinical` for safety/clinical/content.
- `internal/messages/rules/v6.go`: level×kind and slot×kind matrices, goal events, topics, schedule parsing; validator covers the v6 fields.
- Open: rules converted to content need medical approval before they serve (priority 1–99 is outside the 100–799 content band). No client calls `/messages/selected` yet, so nothing user-visible changes; review the list on stage/prod when T-M10-04 lands. Slot/kind mismatches M-18 does not convert are rejected at selection (`kind_not_allowed`, T-M10-04). `/messages/events` version/slot writes → T-M10-06. Admin notes added to T-M10-12.
- Verify: sqlc, schema-diff OK (239 tables), vet, unit + test-int messages/admin/migrations green; contract messages 117/0 (312 allow-listed, unchanged); full `go test ./...` green.

## T-M10-27 — Postpartum analysis hub (D-176)
- Go-only `GET /api/v1/analysis/postpartum` (`internal/analysis/postpartum.go`, pure `BuildPostpartum`): six cards (epds, bleeding, breastfeeding, sleep, growth, weight), each `plus/locked/ready/data/days_needed`. 409 `not_postpartum` outside postpartum mode; 403 `module_disabled` when the postpartum module is off (checked in the handler); child cards hidden when the children module is off; only the user's own child (birth closest to the delivery), never a shared one.
- EPDS: minimums from `P` (1 score to show, 2 for a trend); only totals/bands/`urgent` leave the server, never item answers or `self_harm` (D-54, int-tested). EPDS card is never Plus.
- Bleeding: no urgent "finding"; a red flag logged today (`Recovery.AlertsOn`) only sets `alert_today` and links to `/postpartum` (UrgentCard).
- Paywall change: weight trend is now Plus (SPEC §4.3); sleep stays Plus; EPDS, bleeding, breastfeeding, growth free. Hub minimums (not in §4.1) as constants: bleeding 4 days in 21, breastfeeding/sleep 3 of 7, weight 2 logs since birth.
- Frontend `screens/analysis-postpartum`: all cards from the endpoint; on 404 (Laravel) only the EPDS card from `/postpartum/epds`. New copy: `hub.daysNeeded`, `hub.bleeding.alertToday`, `hub.weight.vsPre`. Bundle goldens re-recorded.
- Open: no contract case for the endpoint (Go-only; int tests + OpenAPI cover it).
- Verify: analysis test-int, typecheck, lint, fsd, dark, vitest (2079), copy:check green; contract public 26/0.

## T-M10-03 — Message fact registry v6 with lazy producers (D-161)
- `internal/messages/facts`: typed registry (`Def{Name, Type, Enum, Unit, Group, Description{fa,en}, MemberSafe, Pending, Variable}`), each group registers itself in `init()`; producer groups (engine, account, logs, vitals, pregnancy, postpartum, contraception) run at most once per request and only when a rule needs them. Warm engine facts ≈2.7 µs; a cold build for a non-pregnant, non-diabetic user is ≤ 4 queries outside the engine state.
- Under `CYCLE_ENGINE=shadow|v6`, `/messages/selected` reads facts from the registry; v1 keeps the legacy facts. Fertility suppression strips `is_fertile_window`, maps the fertile phase to follicular and `user_goal=ttc` to non_ttc (reuses `manager.EngineFacts`). Missing input → absent.
- Admin: `GET /api/admin/v1/messages/facts` (sorted, golden); rule validation 422 `unknown_fact` / `bad_value`.
- Behaviour change: the `mode` fact is now the engine mode (`cycle|ttc|pregnancy|postpartum|menopause|teen`); `life_mode` is the stored life mode. A rule with `mode in [cycle]` no longer matches ttc/teen users. Date facts accept only `eq`/`ne` until selection compares dates (T-M10-04).
- Open: `admin-api.md` entry for `/messages/facts` (noted on T-M10-12); vitals queries duplicated from `internal/vitals` (its handlers have no `*sql.DB`); Plus tier SQL duplicates `plus.tierOf`.
- Verify: sqlc, vet, unit + test-int messages/admin, contract messages 117/0 (312 allow-listed) green.

## T-M10-04 — Message selection v6: slots, invariants, caps, tie-break, weighted variants
- Shipped (D-162): `rules.SelectSlot(slot, rules, facts, *UserHistory, userID, now)` in `internal/messages/rules/select_v6.go` — slot table (capacities, allowed kinds), per-message caps (smart_tip 1/cycle, daily_advice no repeat in 7 d), slot caps (push 1/day + 3/7 d, modal 1/7 d), push quiet hours 23–08 local, invariants (requires_user_input, low confidence M-05, hormonal contraception + `fertile_window` topic M-06, 24 h after safety), tie-break safety → priority → least recently shown → lower id, safety exempt from audience/A-B (M-07), 17 `SelectionReasons`.
- `LoadHistory` reads the whole history in one query (`ListUserMessageEventStats` now returns slot + kind, `all_rules` flag); `CapWindows` uses user-local day/week and the cycle anchor `current_period_start` (rolling `cycle_length` fallback, M-04).
- `Assign`/`bucket` now go through `WeightedVariant`/`StableHash`; hash input stays `"user:rule:salt"` so no existing user changes variant (golden vectors + 10k-user test).
- Bench: `BenchmarkSelectSlot_200Rules` ≈ 31 µs/op (budget 200 µs).
- Open: not wired to any handler yet — the feed endpoint (T-M10-06) calls `LoadHistory` + `SelectSlot`; the legacy `Select`/`Eligible` stay for v2 and admin until then. Date comparisons on `current_period_start` not added (out of scope).

## T-M11-06 — Member projections v6: allow-list DTOs per domain, symptoms removed, effective grants only
- Shipped (D-194, D-204): `internal/family/projections` — one allow-listed DTO per domain (cycle_phase, ttc_fertility, pregnancy, fetal_activity, appointments, meds, vitals), each field set pinned by a reflection test; sentinel int test proves never-shared data is unreachable. Cycle phase carries no numbers; `period_soon` and the TTC switch are partner-only; fertile label hidden under hormonal contraception; med dose status only with nudge; vitals latest reading only; fetal activity pregnancy-only; pregnancy gated by mode + epoch. Reads audited, `Cache-Control: no-store`.
- `companion/shared` is now a compat mapper onto the projections (old `mode.go` removed); companion home uses it. Symptoms section removed (F-03): 404, gone from home and grants. Link views list effective grants only (no `none`, paused or out-of-mode rows).
- OpenAPI + contract allowlists updated (new `babylog.yaml`, `teen.yaml`); contract companion/companion-home/shop-checklist 35/0.
- Open: frontend `entities/companion/api/companion-side.ts` still parses the old section shapes (noted in T-M11-20); weight `category` always null; under CYCLE_ENGINE=v1 the TTC window is empty.

## T-M10-33 — Admin role matrix per §5.8 and hash-chained audit log (D-182)
- §5.8 default matrices (spec→capability table in `httpadmin/permissions.go`); new caps `kpi`, `okr`, `thresholds` (`/clinical-config`), `segments`; `/users/directory/export` → `export`, `/audit*` → `audit`. Deny by default; matrices already saved in `admin_roles` are never rewritten (user OK 2026-10-30: follow the spec — marketing/PM lose content, PM loses users-manage/export, support loses dashboard; customised roles need a super admin to grant the new caps).
- `internal/admin/auditchain`: sha256(prev ‖ canonical JSON) chained append under a `FOR UPDATE` head lock, idempotent genesis over pre-chain rows, `Verify(from)` in a REPEATABLE READ snapshot (break reasons genesis_mismatch / unchained / prev_mismatch / hash_mismatch / head_mismatch). Wired in `httpadmin.Wire` (`UseAuditChain`) and admintest.
- `GET /api/admin/v1/audit/verify?from=` (super); nightly job `admin.audit_verify` 03:40 Tehran (gauge `admin_audit_chain_broken`, counter `admin_audit_chain_verify_total`, head-hash anchor in the log).
- Health reads fail closed (503 `audit_unavailable`): `/health-access/users/:id`, `/users/:id`, `/users/:id/overview`.
- Test fixtures updated for the new defaults: clinicalcfg, okr, cityservices admin_dir, shop/admin. Docs: `docs/go-migration/admin-api.md` §21, `docs/engine-v6/jobs.md`. No migration (00105 already had the columns).
- Open: legacy `/users` still shows the full mobile (noted in T-M10-32). Rollout: an older binary writing unchained rows after genesis shows up as an `unchained` break — deploy all stacks together.

## T-M10-12 — Admin-web message rule editor v6
- Shipped: reusable condition builder `admin-web/src/features/rule-conditions` (facts picker from `GET /messages/facts`, typed inputs, operators per type, all_of / any_of / exclude groups); v6 rule form (kind/level/slot matrices with disabled reasons, up to 6 weighted variants, caps + cooldown, topics with forced `fertile_window`, datetime schedule + several hour windows, goal event, spec names); `ApprovalBanner` (approved vN · draft vM, word diff, approve/reject only with the clinical capability); authority gating (`lib/authority.ts`) mirroring the API; 422/409/403 mapping (`lib/errors.ts`); fa/en strings.
- Docs: `docs/go-migration/admin-api.md` §22 "Message rules v6 additions" (fields, enums, authority, `/messages/facts`, `unknown_fact` / `bad_value`).
- QA: `docs/engine-v6/qa/admin/rule-editor-v6-{light,dark}.png` (mocked API data, reviewed by the main session).
- Verify: typecheck, lint, fsd:lint clean; vitest 48 files / 186 tests passed; build ok.
- Open: the detail API does not return the `approved` snapshot yet, so the banner shows "diff unavailable" until the backend serves it; diff labels use field paths (`a.body`); no live-backend browser pass yet (do it in T-M10-36).

## T-M10-05 — Compiled message rules cache with pub/sub invalidation
- Shipped: `internal/messages/compiled` — `Compile(rows)` / `CompileRules`, `Compiled{BySlot, NeedFacts, Rev, Fingerprint, Errors}` (rules filed per slot, pre-sorted safety → priority DESC → id; hour windows as a 24-bit mask; caps, cooldown and variant weights precomputed; conditions coerced once with `rules.Match` semantics), `(*Compiled).Select` (same pipeline and `rules.Choice` output as T-M10-04 `SelectSlot`, early stop, 1 alloc), `Need(slots...)` + `Prefetch`, `Cache` on `configcache.Cache[*Compiled]` over `message_rules` (recompile on local change / pub/sub / 60 s TTL; failed load or snapshot compile keeps the previous snapshot). Thin exports added in `rules/select_v6.go`.
- Metrics: `messages_rules_compile_errors_total`, `messages_rules_compile_snapshot_errors_total`, `messages_rules_rev`.
- Bench (M-series, 2000x): compile 500 rows 4.6 ms (budget 20 ms); Select 200 compiled rules 0.32 µs, worst case 6.2 µs, 1 alloc/op (budget 50 µs / 2 allocs). Row appended to `docs/engine-v6/perf-baseline.md`.
- Verify: `go vet ./...` clean; tests green without `-race` (CGO linker bug on macOS — the 50-goroutine swap test must run under `-race` in CI/Linux); golangci-lint 0 issues. Equivalence with the parsed path over 1 000 random rule sets.
- Deviations: `compiled.FactSet` (sorted `[]string`) instead of `facts.Set` (does not exist); slots keyed by name; dates kept as Y-m-d strings (parsed path only has eq/ne); placeholders not in NeedFacts (T-M10-06 uses `rules.Placeholders`).
- Open: no wiring needed — T-M10-06 calls `compiled.Load` → `Prefetch` → `Select` and returns `comp.Rev`; the compile-error counter adds 1 per reload per broken rule.

## T-M9-09d — Hard ≤5 min OAuth client revocation
- Shipped: `passport/clients.go` `lookup` never serves a client snapshot at or past `ClientCacheTTL` (5 min): it reloads synchronously first (`loadIfOlder`, shared by concurrent requests) and fails closed with a plain error → 500, never 401, when the reload fails. Background refresh now starts 30 s before the TTL (`clientRefreshAhead`), so steady traffic stays one query; `refreshAsync` skips when another reload already replaced the snapshot it saw. D-140 bound is now hard (also after long idle or repeated `ListClients` failures).
- Tests: within TTL no extra query; refresh ahead of TTL; stale snapshot + revoked client rejected on the first request; reload error past TTL is not an `AuthError` (→ 500); concurrent stale requests share one reload. Old `ClientCacheRefreshesAfterTTL` test (asserted the old lenient behaviour) removed.
- Verify: vet clean; `go test ./internal/auth/...` green (passport also green under `-race`); contract auth 45/45.
- Open: the optional `MustGuard` doc note was not added (`internal/auth/guard.go` is outside touches); no backoff while the DB is down past the TTL (each authed request tries one `ListClients` behind the mutex, then 500).

## T-M10-26b — Analysis follow-ups: pregnancy weight minimum, menopause rate-ratio, top finding on TTC and pregnancy hubs
- Shipped (D-175, SPEC §4.1): pregnancy weight needs `P` = 2 logs since the pregnancy start (`min_logs`, `logs_needed`, `ready:false` + «not enough data» line on the hub and weight page); menopause `Binary` moved to the shared rate-ratio method with the 30-day minimum from `analysis.P` (old φ/Cohen bands and `CorrMin*` constants removed); TTC and pregnancy hubs return `top_finding` from the cycle hub's strength × recency scorer (`finding_hubs.go`), rendered as a card in `screens/analysis-ttc` and `screens/analysis-pregnancy`.
- Copy: `analysis-pregnancy` fa/en (`hub.topFinding`, `hub.weight.notEnough`, `findings.*`), synced to `resources/translations`; `internal/i18n/testdata` re-recorded.
- Goldens: unit + v6 TTC goldens gain `top_finding` only (no existing value changed); contract `menopause` allow-lists `min_days` 20→30 and rate-ratio `strength` (D-175); new `allowlist/analysis-pregnancy.yaml`.
- Verify: vet, unit, golangci-lint clean; test-int analysis + menopause ok; contract analysis/analysis-ttc/analysis-pregnancy/menopause 103 passed 0 failed, public 26/0; frontend typecheck, fsd:lint, vitest 2085 passed.
- Open: the TTC hub does not load daily logs, so symptom patterns / Plus correlations are not candidates there; LH positive is not a candidate (no finding sentence in `analysis.json`); `preg_bp_severe` says "call today" even for an old reading — needs clinical copy review.

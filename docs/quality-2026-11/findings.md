# Quality pass 2026-11 — security, speed, logic tests (evidence for tasks/M12)

Read-only audits run 2026-11-18 on branch `stage` (four opus agents: backend security, web/infra security,
performance, test suites). No secret values are recorded here. Each row is owned by one M12 task; the task records
the resolution in `tasks/PROGRESS.md`. Prior audits (`docs/security/*.md`, `docs/engine-v6/perf-baseline.md`) were
read first; rows resolved there are not repeated.

## 1. Security

| ID | Sev | Where | Problem | Task |
|---|---|---|---|---|
| S-01 | High | `backend-go/internal/auth/handlers.go:33-38,232-261,305-315`, `internal/http/routes_auth.go:40-41` | 4-digit OTP, 5 tries per code, new code every 60 s resets attempts; no per-number send cap, no failure budget across codes; send+verify share one unnamed per-IP counter. One IP can brute-force a victim's number in ~1 day → 365-day token. | T-M12-01 |
| S-02 | Low | `auth/handlers.go:232-261` | send-otp is check→delete→insert without a lock: parallel sends → N SMS (bombing, cost). | T-M12-01 |
| S-03 | High | `deploy/beta-ssl.conf:14-17`, `deploy/vhost-beta.inc:31-32`, `deploy/vhost-beta-admin.inc:23-24`, `backend-go/cmd/api/app.go:44-47` | Client IP = rightmost XFF with no `set_real_ip_from` for the CDN; origin :443 public; Go trusts `0.0.0.0/0`. A direct-to-origin request with forged XFF bypasses every per-IP limit (OTP, share lookups, admin login). | T-M12-02 |
| S-04 | Medium (suspected) | `deploy/stage-gate.sh:44-49`, `deploy/vhost-stage.inc:51-58` | Stage password gate skipped for any private `$remote_addr`; via docker-proxy (IPv6/hairpin) external clients appear as 172.x → no password, one shared OTP bucket. | T-M12-02 |
| S-05 | Low | `deploy/vhost-stage.inc:58` | `ritme_stage` gate cookie lacks `Secure`. | T-M12-02 |
| S-06 | High | `.github/workflows/claude-to-stage.yml:12-16,62-95`, `deploy/systemd/ritme-stage-autodeploy.service`, `deploy/stage-autodeploy.sh:48-60` | Any push to `claude/**` auto-merges into `stage` and is deployed **as root on the production host** by a script taken from the new checkout. No human review, no signature check. | T-M12-03 |
| S-07 | High | `frontend/package-lock.json` (next 15.5.18), `admin-web/package-lock.json` (next 15.5.26) | npm audit: next cache-poisoning (GHSA-mcj8-r9mp-w47p, GHSA-4jqv-mc3x-m676), image-optimizer RCE advisories, zod <4.6.6 DoS, + postcss/sharp/nanoid/… `/_next/image` enabled though unused. | T-M12-04 |
| S-08 | Low | `laravel-site/composer.lock` | `enshrined/svg-sanitize` stored-XSS advisory (used by `SvgSanitizer.php`), `symfony/mime`, `symfony/html-sanitizer`. | T-M12-04 |
| S-09 | Medium | `frontend/src/shared/pwa/offline/offline-media.ts:51`, `shared/session/cleanup.ts:90-101`, `scripts/sw.template.js` (`OFFLINE_PAGES`) | Offline lesson downloads (+ plaintext titles such as IVF / loss courses) survive logout, delete and account switch. | T-M12-05 |
| S-10 | Low | `backend-go/internal/http/routes_profile.go:40-41` | `GET /profile/export` and `DELETE /account` need only the bearer; no throttle, no step-up. Export is ~50 unbounded queries (DoS). | T-M12-06 |
| S-11 | Low | `backend-go/internal/http/routes_docs.go:47-66` | `/docs` basic auth has no failure throttle. | T-M12-06 |
| S-12 | Medium (suspected) | `.env.example:2`, `.env.stage.example:8` (`APP_KEY`) | A real-looking Laravel `APP_KEY` is committed in both examples. If a server `.env` was copied from it, the key is public. | T-M12-07 |
| S-13 | Info | `backend-go/contract/fixtures/keys/oauth-private.key` | Committed test Passport key; nothing stops prod from loading it. | T-M12-07 |
| S-14 | Low | `frontend/src/shared/autopilot/config.ts:154-164` | Sharing controls (`invite.create`, `invite.consent`, `companion.join-submit`, `teen.share:*`, `report.send`, `privacy.export`) are named but not in `confirmTargets`. | T-M12-08 |
| S-15 | Medium | `deploy/vhost-api.inc:17-21`, `deploy/go-routes.inc:29-35`, `backend/bootstrap/app.php:43-49` | Prod: Go reads leftmost XFF (client-controlled); Laravel sees CDN edge as client (shared OTP throttle). Known, waits for cutover. | T-M12-09 (blocked) |
| S-16 | Low | `deploy/vhost-api.inc` (`location /`) | Prod exposes Passport `/oauth/*` routes. | T-M12-09 (blocked) |
| S-17 | Low | `db/queries/auth/otp.sql`, `SendOTPJob.Code` | OTP stored in plain text in DB and queue (Laravel-compatible). | T-M12-09 (blocked) |

Checked clean (backend): JWT/Passport verify, OTP crypto/rand + atomic attempts + constant-time compare, IDOR on every
id-scoped query, admin API guard/RBAC/CSRF, payments (lock, amount, unique bank ref), credit reserve/refund, uploads
(magic bytes, re-encode, `os.OpenRoot`), outbound URLs fixed, prod fail-closed for fake providers, pprof/metrics on
loopback, log masking, share tokens hashed. Web: all `dangerouslySetInnerHTML`, admin URLs validated, laravel-site
raw Blade sanitized, token only in `Authorization`, SW skips `/api/*`, autopilot route checks, admin-web CSRF +
`safeNext`, HSTS/frame-ancestors, no committed real `.env`.

## 2. Speed

| ID | Where | Problem | Task |
|---|---|---|---|
| P-01 | `frontend/src/app/globals.css` (962 KB, 14k lines) | All screen-prefixed CSS (`.nb-`, `.lab-`, `.pgn-`, …) is render-blocking on every route; ~6× growth since T-M1-10. No CSS/JS size gate. | T-M12-27 |
| P-02 | `frontend/src/app/autopilot/AutopilotMount.tsx:3-12` | ~230 KB autopilot catalog + ~65 KB engine loaded eagerly in the root layout. | T-M12-28 |
| P-03 | `frontend/src/screens/home/ui/HomePage.tsx:64-118`, `LogPage.tsx:6`, `LogSheet.tsx:4` | Mode-specific homes, editors and the voice-log/AI stack imported statically. | T-M12-28 |
| P-04 | `frontend/src/app/message-scopes.ts:61` | Home ships 24 whole namespaces (146 KB min) in HTML + RSC. | T-M12-29 |
| P-05 | `backend-go/internal/querybudget/budgets.go:39-83`, `loadtest/app_open.js:22-26` | Web home fans out to ~18-20 GETs; 14 of them have no query budget; k6 hits Android-only `/home`. | T-M12-30 |
| P-06 | `backend-go/internal/healthlog/entries.go:49,185-203`, `handlers_logs.go:173-185` | `PUT /logs/days/:date`: delete+insert per entry inside a row lock, `Day` read twice; `log_burst.js` tests the legacy route. | T-M12-31 |
| P-07 | `db/queries/home/home.sql:152-160` | `user_notifications` list filesorts (only `(user_id, read_at)` index), OFFSET paging. | T-M12-32 |
| P-08 | `queries/care/appointments.sql` (`ListAppointmentsFrom`), `assistant.sql:78` | No LIMIT on next-appointment read; whole conversation loaded per assistant turn. | T-M12-32 |
| P-09 | `HomePage.tsx:189-205,754-755,915-929` | Previous-month `/cycle/month` waits for `/cycle/today`; 3-4 month requests per cold home. | T-M12-33 |
| P-10 | `platform/etag` used only by cycle/analysis | `/modules`, `/banners`, `/logs/taxonomy`, `/health-logs/enums`, `/learning/courses`, `/checkups/home` have no ETag/304. | T-M12-34 |
| P-11 | `cmd/api/cors.go:55` | Per-URL preflight: ~11 OPTIONS per cold home cross-origin. | T-M12-35 |

## 3. Logic tests

Suite run 2026-11-18: `CGO_ENABLED=0 go test ./...` → 180 ok / 0 fail (cgo link fails on this Mac: SDK 27 tapi
error); the 297 `*_int_test.go` files skip silently without `TEST_DB_DSN`. vitest → 7 of 2291 fail, all in
`frontend/src/app/message-scopes.test.ts:107` (red for ~8 weeks, `continue-on-error` in CI).

| ID | Gap | Task |
|---|---|---|
| T-01 | CI never runs integration tests (no MariaDB/Redis service), never `-race`; CI only on `claude/**` pushes; local `make test` breaks on cgo. | T-M12-11 |
| T-02 | vitest red and non-blocking; admin-web tests never run; no coverage reports. | T-M12-12 |
| T-03 | Account delete relies on FK cascade + hand deletes (`profile/service.go:209,231`); ~139 `user_id` columns vs ~123 FKs; test checks only `users`/`oauth_*`. | T-M12-13 |
| T-04 | Export completeness not checked against the user-table list. | T-M12-14 |
| T-05 | No end-to-end routine flow test (signup→period→today; checkout→spend→expiry; invite→grant→revoke; module off→403). No e2e harness at all. | T-M12-15..18, T-M12-25 |
| T-06 | Credits: no test at the period-expiry second, refund after expiry, stacked purchase re-grant, spend+refund race. Frontend `features/ai-gate` untested. | T-M12-19, T-M12-23 |
| T-07 | 106 int-test files use `time.Now()`; `labs_int_test.go:786` self-skips on days 1-2; month-boundary flakes (`admin/users/v2_int_test.go:73`, …); 13 `time.Sleep` tests; golden sweeps never recorded (`cycle/legacy/golden_sweep_test.go:72`, `cycle/view/golden_sweep_test.go:81`). | T-M12-20 |
| T-08 | No 30 Esfand leap-day test in the Go engine, no non-Tehran `usertz` medication schedule, no 23:30 vs 00:30 period-log boundary. | T-M12-21 |
| T-09 | Push dedupe only tested single-tick; no concurrent ticks / retry-after-fail. | T-M12-22 |
| T-10 | Kill-switch checked for one module only; a new ungated route would pass. | T-M12-18 |
| T-11 | No frontend test for several requests hitting a 401 at once. | T-M12-23 |

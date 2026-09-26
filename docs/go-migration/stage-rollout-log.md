# Staging rollout log (T-M2-25)

Environment: `stage.ritmeapp.ir` (stage branch @ f35a2c9), both backends on the staging MariaDB/Redis,
Laravel still running (instant rollback via `deploy/switch-go-route.sh stage <group> off`).

## Method
- Smoke account: user `09900000901` ("Go Smoke", staging only), Laravel-issued Passport token (tinker `createToken`),
  profile + 3 health logs created through Laravel before any flip.
- Smoke = 21 authenticated GETs × fa/en = 42 requests (content, reminders, health-logs, profile, pregnancy, cycle
  status/today/month full+calendar/period status+history, messages/daily, home, plus 404 paths).
  Baseline recorded with every group on Laravel; after each flip the same 42 requests were compared with the
  baseline (status + semantic JSON equality). Which stack answered = `X-Backend: go` header.
- Gate: staging Basic auth → `ritme_stage` cookie, then Bearer.

## Flips (2026-09-23T11:39Z)
| # | Group | Go-served requests (cumulative) | Diffs vs Laravel baseline | Result |
|---|---|---|---|---|
| 1 | content | 8 / 42 | 0 | ✔ on Go |
| 2 | reminders | 12 / 42 | 0 | ✔ on Go |
| 3 | healthlog | 16 / 42 | 0 | ✔ on Go |
| 4 | profile | 18 / 42 | 0 | ✔ on Go |
| 5 | pregnancy | 20 / 42 | 0 | ✔ on Go |
| 6 | cycle | 32 / 42 | 0 | ✔ on Go |
| 7 | messages | 34 / 42 | 0 | ✔ on Go |
| 8 | home | 36 / 42 | 0 | ✔ on Go |
| 9 | auth | 36 / 42 | 0 | ✔ on Go |

(auth routes themselves are POST-only and not part of the GET smoke; see below.)

## Auth checks (after `auth` → Go)
- Laravel queue drained before the flip: `jobs` = 0, `failed_jobs` = 0.
- Laravel-issued token keeps working on every group (smoke above ran with it after the auth flip: 0 diffs).
- `POST /api/v1/auth/refresh-session` → 200 from Go, `{"refreshed": false, "expires_at": "2027-09-23T15:04:10+03:30"}`
  (token too fresh to rotate — same rule as Laravel).
- Garbage token → 401 `{"message":"Unauthenticated.","error_code":"unauthenticated"}` from Go.
- Write through Go: `POST /api/v1/health-logs` → 201, `X-Backend: go`.
- Go-issued token accepted by Laravel: covered by `internal/auth/crossstack_int_test.go` (not re-run on stage — would
  need a real OTP/SMS login).

## Rollback test
`profile` group, same token: on Go → 200 `X-Backend: go`; `switch-go-route.sh stage profile off` → 200 from Laravel;
`on` again → 200 `X-Backend: go`. Each flip = `nginx -t` + graceful reload of the shared proxy.

## Not yet done
- Human smoke via the web app and the Android shell against staging (all groups on Go).
- 30-minute log watch per group and the 48 h soak: staging has almost no real traffic (1 real user), so the soak
  only means something if someone uses staging; started $NOW with all groups on Go.

## Admin on staging (2026-09-23)
Staging is a single origin, so the new admin lives under **`/panel/`** (admin-web built with
`NEXT_PUBLIC_ADMIN_BASE_PATH=/panel`) and its API at `/api/admin/` → backend-go; the Blade panel stays at `/admin`
for side-by-side comparison. `vhost-stage.inc` written in place on the shared proxy (`nginx -t` + reload, no
recreate; backup `/root/vhost-stage.inc.bak-20260923-114953`). First admin-web image build: OK, container healthy.
- deploy-stage checks: `/panel/login` 401 without the gate / 200 behind it, `/api/admin/v1/auth/me` JSON 401 from Go.
- Login with the existing staging admin (Laravel bcrypt hash) through Go: 200, `X-Backend: go`, returns `admin` +
  `csrf_token`. Then `auth/me`, `dashboard`, `users`, `articles`, `banners`, `languages`, `messages` → all 200.

## Remaining for T-M2-25
- Human smoke of the web app against staging (all groups on Go). (Android excluded from tasks by user decision, 2026-09-23.)
- Editors click through every `/panel` screen (compare with `/admin` Blade).
- 48 h soak with all groups on Go, started 2026-09-23 ~11:40 UTC → earliest sign-off 2026-09-25 ~11:40 UTC.

## 48 h soak evidence + automated smoke (2026-09-26 ~09:05 UTC, T-M2-25 close-out)
No new code deployed for this. Stage is **Go-only** since 2026-09-23 12:36 UTC (T-M2-28: `backend`/`queue` are compose
profile `laravel` and their containers are removed; `switch-go-route.sh stage <g> off` now lands on Go too). So the
soak ran on Go alone — the task's "Laravel still running (instant rollback)" precondition no longer holds on stage;
rollback was proven on 2026-09-23 (profile flip above) before Laravel was removed. Prod rollback remains T-M2-26's job.

### Soak window 2026-09-23 11:40 → 2026-09-26 09:00 UTC (~69 h)
Containers: `ritme-stage-backend-go-1` started 2026-09-23 12:36, admin-web 11:58, frontend 11:30 — `RestartCount` 0 for all.
```
$ docker logs --since 2026-09-23T11:40:00Z ritme-stage-backend-go-1 | jq -r 'select(.msg=="request")|"\(.time[0:10]) \(.status)"' | sort | uniq -c
     36 2026-09-23 200      # all from the 12:36-12:37 deploy/smoke run
      4 2026-09-23 401      # deploy-stage checks (/api/admin/v1/auth/me, /api/v1/banners without token)
     11 2026-09-23 404      # smoke's expected 404s (unknown route, no content/log for date, /notifications, /user)
     ...                    # 2026-09-24 and 2026-09-25: ZERO requests
$ ... | jq 'select(.level!="INFO")'        -> (none: no WARN/ERROR lines, no panics)
$ docker logs --since 2026-09-23T11:40:00Z ritme-proxy-1 | awk '$9 ~ /^5/'   -> (none: 0 5xx on any vhost)
```
- 5xx: **0** (Go and proxy). Go-caused regressions: **none**.
- 401 `error_code` distribution: all `unauthenticated` (only the deliberate no-token / garbage-token probes).
- Caveat: **no organic traffic at all** on 09-24/09-25 (the proxy log has no host field; backend-go's own request log
  is the source of truth for Go traffic). The soak therefore proves stability of an idle stack, not behaviour under use.
- Non-Go observation: proxy error log shows `recv() failed (104: Connection reset by peer)` from the **stage Next.js
  frontend** for `/icons/icon-192.png` (3× during the headless run, client still got a response — no 5xx in the access log).

### API smoke through Go (fresh token via OTP read from `otp_verifications`, smoke user 09900000901)
Gate cookie (`ritme_stage`) + Bearer; 39 GET paths × fa/en = 78 requests (`smoke.sh`, paths from the deployed
`backend-go/api/openapi.yaml`):
```
  66 200   12 404   — 78/78 `X-Backend: go`, 0 5xx
404s (all data-driven, localized): health-logs/<today> (no log yet), cycle/phase-content/menstrual, home/sections/cycle
(unknown section), pregnancy/profile (not pregnant), pregnancy/content/12 (no content seeded), /api/v1/does-not-exist
```
Writes/auth: `POST /api/v1/health-logs` → 201 (go); `POST /auth/refresh-session` → 200 `refreshed:false`;
send-otp/verify-otp → 200 (go); garbage Bearer and no Bearer → 401 `{"error_code":"unauthenticated"}`;
wrong Basic credentials → nginx 401.

### Admin API smoke (/api/admin/v1 via Go, staging admin, session cookie)
```
login 200 go | auth/me dashboard users admins articles articles/options banners challenges challenge-completions
info-sections phase-contents pregnancy-weeks recommendations task-templates affirmations languages languages/options
languages/1/translations messages  -> all 200, X-Backend: go | logout 200
```
(covers the list endpoint of every `/panel` screen in admin-web). `/panel/login` 200; `/admin` → 301 `/panel/` → `/panel/login`.

### Headless web smoke (Chrome CDP, gate cookie + `ritme_token` in localStorage, 390 px mobile)
`/fa/home /en/home /fa/calendar /fa/cycle /fa/log /fa/profile /fa/pregnancy /en/profile` all rendered logged-in
(correct `dir` rtl/ltr, bottom nav, home cycle card "۱۲ روز تا پریود بعدی"); every `/api/` call the pages made
answered 200 `X-Backend: go` except `pregnancy/profile` + `pregnancy/content/1` → 404 (user not pregnant → pregnancy
onboarding screen shown, expected).

### Still open — needs a human (not faked here)
- **Editor click-through of every `/panel` screen** incl. create/edit/upload flows (only list APIs were exercised).
- **Real-user smoke of the web app** on a phone (log a period, edit profile, onboarding, pregnancy on/off, reminders).
- A soak **with real traffic**: 09-24/09-25 had zero requests; if that matters, have someone use staging for a day.

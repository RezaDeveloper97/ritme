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

## Local full smoke on Go (2026-09-28)

Everything below ran **locally** on branch `stage`; no staging or production host was contacted (the agent has no server
access). backend-go ran on `:8020` (`make run`, `SMS_PROVIDER=log`, `ADMIN_COOKIE_SECURE=false`) against the docker
test-stack MariaDB. It used a scratch DB `ritme_m225`: goose `00001` baseline, then the data rows of
`contract/fixtures/dump.sql` (content, 18 personas, the fixture admin, the personal-access client), then goose up to v5.
The Next frontend ran on `:3000` with `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1` passed as an env var
(`.env.local` untouched). admin-web ran on `:3001` with `ADMIN_API_PROXY_TARGET=http://127.0.0.1:8020`. So **every
route group (auth, profile, cycle, healthlog, messages, reminders + care, content, home, pregnancy, admin API) was
served by Go**; Laravel was not running. Headless Chrome 153 was driven over CDP (Node's global `WebSocket`, 20 s
timeout on every call, every script under a 120 s alarm). The app ran at 390×844 @2x, mobile + touch, locale fa,
light and dark. Two throwaway users signed up through the real UI, with the OTP read from `otp_verifications`:
`09120000801` (light) and `09120000802` (dark). A temporary `super` admin `smoke@local.test` (random password in a
0600 scratch file, never printed) was used. The scratch DB was dropped afterwards and the test stack left running.
The Blade `/admin` side-by-side comparison is **not possible locally**: it needs Laravel, and the stack is Go-only.

### verify-all

| Check | Result |
|---|---|
| `go vet ./...` | ✔ pass |
| `go test ./...` | ✔ pass (63 packages ok) |
| `golangci-lint run` | ✔ 0 issues |
| frontend `npm run typecheck` | ✔ pass |
| frontend `npm run lint` | ✔ pass |
| frontend `npm run fsd:lint` | ✔ No problems found |
| frontend `npm run lint:styles` | ✔ style gate passed (459 files) |
| frontend `npm run lint:dark` | ✔ passed (57 contrast pairs, 123 tokens; only the pre-existing light-mode ⚠ pairs) |
| frontend `npm run test` | ✔ 70 files, 573 passed |
| admin-web `typecheck` / `lint` / `fsd:lint` / `test` | ✔ pass / ✔ pass / ✔ No problems found / ✔ 25 files, 77 passed |
| Laravel (`pint`, `artisan test`) | skipped: `backend/` unchanged |

### Web app flow (fa, light and dark, same steps)

The status codes come from CDP `Network.responseReceived` for every `/api/v1` call the page made. Light made 253 calls
and dark made 201. Every screen also got a DOM check: no horizontal overflow (`scrollWidth` ≤ `innerWidth`, no element
outside the viewport), no raw i18n keys, Latin digits in fa text, and console errors/exceptions. **No 5xx anywhere, no
overflow, and no raw keys.** The Latin digits and the console errors are bugs 1–3.

| # | Step (route group) | API calls → status | Result |
|---|---|---|---|
| 1 | splash → `/signup` → OTP (auth) | `send-otp` 200, `verify-otp` 200 | ✔ new user created, lands on onboarding |
| 2 | onboarding: name + terms, birthday, weight, height, intention «فعلاً قصد بارداری ندارم», period-len, cycle-duration, cycle-len, conditions → setting-up (profile) | `POST /profile` 200, then all GETs 200 | ✔ lands on `/fa/home`, cycle ring «روز ۱», phase card |
| 3 | cycle home, `/calendar` (month view), «ویرایش پریود» → pick ۱۰ شهریور → «ذخیره» (cycle) | `cycle/today`, `cycle/month/2026/9|10?view=calendar`, `cycle/period/history` 200; `POST cycle/period` 200 | ✔ run ۱۰–۱۴ شهریور saved (`cycle_histories` row, 5 days); Analysis shows the previous cycle 26 days. ✘ hydration error on `/calendar` (bug 1); ✘ Latin badge digits in the editor (bug 2) |
| 4 | `/log` → «خلق‌وخو» → «شاد» → «ثبت» (healthlog), `/cycle` analysis (messages) | `health-logs/enums` 200, `health-logs/<today>` **404 (expected: no log yet)**, `POST health-logs` 201; `messages/daily`, `messages/mode`, `home/sections/*` 200 | ✔ saved; weekly summary «روحیه ۱۰۰٪», smart tip, BMI card. ✘ hydration error on `/cycle` (bug 1) |
| 5 | `/reminders` → new medication (قرص) → save → «ثبت مصرف» → switch off/on; `/reminders/appointment/new` form; Profile «یادآورها» sheet → new legacy reminder → save → delete (reminders + care) | `POST care/medications` 201, `POST …/intakes` 200, `PUT care/medications/{id}` 200 ×2; `GET reminders` 200, `POST reminders` 201, `DELETE reminders/{id}` 200 | ✔ all persisted; home «یادآورهای امروز» shows the ticked medication |
| 6 | articles sheet → article `pms-management`; «بیشتر درباره این فاز» (phase content); banners (content) | `articles?page=1&per_page=12`, `articles/pms-management`, `cycle/phase-content/menstrual?locale=fa`, `banners` 200 | ✔ content renders. Fixture banner images `/storage/contract/*.webp` don't exist on disk → 404 → `ERR_BLOCKED_BY_ORB` (fixture artefact, not a bug). A banner **uploaded through admin-web** (step A4) renders on the app home (`naturalWidth` 1200). Article category chips show raw slugs (bug 4) |
| 7 | Profile → language sheet → English → `/en/home` → back to فارسی | `languages` 200 | ✔ `lang=en dir=ltr`, whole UI in English; back to `fa/rtl` |
| 8 | bell → notifications sheet → «همه رو خوندم» (one row inserted in the scratch DB) | `home/notifications?per_page=50` 200, `POST home/notifications/read-all` 200 | ✔ |
| 9 | home «چالش امروز» toggle on/off (home); home tasks via API | `POST home/challenges/{id}/toggle` 200 ×2; `POST home/tasks/1/toggle` 200 ×2 (`is_completed` true → false, progress 1/5 → 0/5) | ✔. The DayTasks widget is commented out in the app (`frontend/src/screens/log/ui/LogPage.tsx:299`), so tasks were checked through the API only |
| 10 | Profile → «خروج از حساب» | `POST auth/logout` 200 | ✔ token removed, `/fa/home` → `/fa/signup` |
| 11 | log in again (same number) | `send-otp` 200, `verify-otp` 200 | ✔ returning user lands straight on `/fa/home` |
| 12 | session refresh (`POST auth/refresh-session`, browser token) | fresh token → 200 `refreshed:false`; after pulling `expires_at` to +10 d in the scratch DB → 200 `refreshed:true`, new token; old token → 401, new → 200 | ✔ both users |

Expected non-2xx only: `health-logs/<today>` 404 before the first log of the day; 401s on the next page load after the
step-12 rotation had revoked the browser's token (the app cleared the session and went to `/signup`, which is correct);
one `send-otp` 429 when the harness re-ran within 60 s (the limit working as designed).

### admin-web (`/panel` screens, 1440×900, fa, light)

Every item in `widgets/shell/model/nav.ts` was opened: the list, the first detail/edit page, and `/new` where the route
exists. The check looked for an `h1`, `[role=alert]` errors, raw i18n keys, overflow, console errors and non-2xx API
calls. The result: **163 API calls in the tour (`/api/admin/v1` + `/api/v1/languages`), all 200, and every CRUD write 2xx** (plus the expected 401 `auth/me` before
login and after logout), **no console errors, no overflow, no error alerts, and no raw keys.** The keys on the
translations screen (`delete.cancel` …) are the key column of the editor itself. `/ui-kit` is dev-only and was skipped.

| Screen | List | Detail / edit | New | Notes |
|---|---|---|---|---|
| login / logout | ✔ | — | — | login 200, logout 200 → `/login`; `/users` without a session → `/login` |
| dashboard | ✔ counters + 8 recent users | — | — | the two smoke users show at the top |
| users | ✔ 20 rows | ✔ `/users/1020`, `/users/1019` | — | |
| admins | ✔ 2 | ✔ | ✔ | |
| articles | ✔ 6 | ✔ | ✔ | |
| affirmations | ✔ 7 | ✔ | ✔ | **CRUD:** create 201 → edit 200 → delete 200 (A1–A3) |
| challenges / challenge-completions | ✔ 20 / ✔ 4 | ✔ | ✔ | |
| task-templates | ✔ 5 | ✔ | ✔ | |
| phase-contents | ✔ 12-phase grid | ✔ | ✔ | |
| recommendations | ✔ 20 | ✔ | ✔ | |
| checkup-types | ✔ 12 | ✔ | ✔ | |
| banners | ✔ 3 | ✔ | ✔ | **CRUD:** create with an image upload 201 → shown in the app → delete 200 (the file is removed from storage) (A4–A5). Thumbnails blank locally (note 6) |
| info-sections | ✔ 5 | ✔ `/info-sections/1` | ✔ | |
| pregnancy-weeks / care-plan / alert-rules | ✔ 40-week grid / ✔ 5 / ✔ 8 | ✔ / ✔ / ✔ `vomiting_streak` | ✔ / ✔ / — | |
| messages | ✔ 20 | ✔ `/messages/172` | — | |
| languages / translations | ✔ 3 | ✔ `/languages/1`, `/languages/1/translations` | ✔ | |
| account/password | ✔ | — | — | |

### Bugs (not fixed here)

> **Fixed in T-M2-30** (bugs 1–4). 1: `/calendar` holds `DayLogSummary`'s loading state until mount; `/cycle`
> mounts its cards (my cycles, smart tip, cycle summary, week summary, BMI — all token-gated queries) only after
> hydration. 2: formatted numbers are passed into the messages (challenge chip + tooltip, calendar day card, period
> editor badges and hint). 3: fa BMI fallback copy says «ریتمی» — deviation **D-22**; `message_contents` rows already
> seeded by Laravel keep "Ritme" until edited in admin → messages. 4: `articles.categories.<slug>` labels (fa + en),
> unknown values shown as stored. Re-check (local, Go `:8020`, persona `09900000004`, fa, light + dark): no hydration
> error on `/fa/calendar` or `/fa/cycle`, no Latin digits on either; `web-06`, `web-07`, `web-12`, `web-22`,
> `web-29`, `web-30` re-taken (persona data, so the dates differ from the first run).

1. **Hydration mismatch on `/calendar` and `/cycle`** (dev console `Hydration failed…` on every full page load, both
   themes). Queries gated with `enabled: isAuthenticated()` are disabled during SSR (no `localStorage`) but enabled on
   the client. So the server renders the "not loading" branch and the client's first render the loading branch:
   `frontend/src/screens/calendar/ui/DayLogSummary.tsx:164` (`logQuery.isLoading` → `.dls-loading` vs `.dls-empty`;
   the query is `frontend/src/entities/health-log/api/queries.ts:62`) and `frontend/src/screens/cycle/ui/MyCyclesCard.tsx:35-36`
   (skeleton vs `null`; the query is `frontend/src/entities/cycle/api/sections.ts:138`). React re-renders the tree
   on the client, so there is no visible damage, but it is a real error. To reproduce: open `/fa/calendar` or
   `/fa/cycle` logged in and watch the console.
2. **Latin digits in fa.**
   - «روز 1 چرخه» on the home «چالش امروز» chip: `frontend/messages/fa/challenge.json:3` `"روز {n} چرخه"`, used by `widgets/today-challenge/ui/TodayChallengeCard.tsx:67`.
   - «روز 1 سیکل» under the selected day on `/calendar`: `frontend/messages/fa/calendar.json:48`, used by `screens/calendar/ui/CalendarPage.tsx:398`.
   - The period-editor day-order badges 1–5: `frontend/src/features/log-period/ui/PeriodDateEditor.tsx:318` renders `{order}` raw.

   Fix: use `{n, number}` like `home.json`'s `ring.cycleDay`, and `format.number(order)`.
3. **Latin brand name in fa BMI copy**: «… Ritme تلاش می‌کند …» (`backend-go/internal/profile/bmi.go:125` and
   `backend-go/internal/messages/content/defaults.json:2045`). This is verbatim from Laravel, so it is a copy fix, not a Go regression.
4. **Article category chips/tags show raw slugs** (`nutrition`, `cycle_science`, `period_health`, …) in the fa articles
   sheet: `frontend/src/screens/articles/ui/ArticlesSheet.tsx:96` (`label={name}`) and
   `frontend/src/entities/article/ui/ArticleCard.tsx:55`. It is data-driven (the fixture articles store slugs), but
   nothing maps the slug to a label. It needs either a label map or translated categories from the API.
5. (Harness, not app) The first OTP auto-fill once landed before the 4 inputs had mounted. Waiting for the inputs fixed it.
6. (Local-only, not a bug) admin-web's CSP `img-src 'self' data: blob: https:` blocks the `http://127.0.0.1:8020/storage/…`
   thumbnails in local dev, so banner previews are blank locally. On stage/prod the URLs are https (or same-origin).
   Separately, `deploy/vhost-admin.inc` still describes the Blade host (`/api/` → 404, `/` → Laravel). When the prod
   admin host moves to admin-web (T-M2-26), `/storage/` must reach backend-go or APP_URL-absolute https URLs must be
   used, or uploaded previews will 404.

### Screenshots (`screenshots/`)

App screens are `web-NN-<step>-light|dark.png` (585 px wide, pngquant). Admin screens are `admin-<screen>[-detail|-new].png`
plus `admin-crud-*` (585 px wide, light).

| Files | Shows |
|---|---|
| `web-01…03` | splash, signup, OTP filled |
| `web-04-onb-*` | every onboarding step + setting-up |
| `web-05`, `web-06` | cycle home after signup, calendar |
| `web-07`, `web-08` | period date editor with ۱۰–۱۴ شهریور picked, home after save |
| `web-09…12` | log, mood sheet, saved, analysis (daily/smart messages, summaries, BMI) |
| `web-13…21` | care reminders (empty, medication form, list, intake), appointment form, profile reminders sheet (form, saved, delete confirm) |
| `web-22…24` | articles sheet, article, phase sheet |
| `web-25…27` | language sheet, profile + home in English |
| `web-28…30` | notifications, today challenge before/after toggle |
| `web-32`, `web-33` | after logout (signup), home after logging in again |
| `web-34-home-admin-banner-light` | the admin-uploaded banner on the app home |
| `admin-*` | every admin-web screen listed above; `admin-crud-affirmation-*`, `admin-crud-banner-*` for the CRUD rounds |

### Still open (human)

- **Staging** web app smoke on a real phone, and an editor click-through of every `/panel` screen incl. uploads,
  compared with Blade `/admin` on stage.
- **48 h soak sign-off** (a soak with real traffic, if that is wanted — see the evidence above). The agent has no
  server access, so none of this could be done here.

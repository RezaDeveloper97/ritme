# Parity report — Laravel vs Go (T-M2-24)

Run on 2026-09-23 (the real date is the same as `CONTRACT_TODAY`) on an Apple M4 (10 cores) dev machine, branch `stage`.
Commits tested: `24c1904` (clean worktree) and `988337c` (HEAD after T-M2-19/T-M2-21b landed, clean `backend-go/`).
This is an investigate task: no code changed. The only file written is this report.

## Verdict

**Conditional GO for T-M2-25, public API groups only. NO-GO for finishing T-M2-25 until the blockers below are closed.**

- ✔ API parity: `make contract ROUTES=all` passes 986/986 cases on a clean checkout and on HEAD. The allow-list is
  empty. All 74 Laravel `/api/v1` routes have at least one contract case.
- ✔ Tokens work across both stacks in both directions, and revocation shows up on the other side as `token_revoked`.
- ✔ Web smoke: signup → OTP → full onboarding → home, calendar, cycle, log, profile, `/en` home, and pregnancy
  tracker/log. The web client sent the same API calls with the same status codes to both stacks. 8 of 11
  screenshots are byte-identical.
- ✔ Go is faster on 7 of 8 hot endpoints, with 1.2×–2.4× the RPS, and uses about ¼ of the memory. `/home` is about
  equal (see caveats).
- ✘ Blockers before T-M2-25 is complete:
  1. The Android smoke was not run.
  2. Some web write flows were not clicked through.
  3. 12 deviations are still `proposed` and need the user's approval.
  4. T-M2-09 (staging deploy) is `blocked`.
  5. T-M2-23 (admin-web screens) is `todo`.

## 1. Verification (`verify-all`)

| Command | Result |
|---|---|
| `backend: vendor/bin/pint --test` | ✘ `FAIL 348 files, 68 style issues`. **Existed before this task**: `backend/` has no changes in the working tree, and pint `--test` was already red before M2. |
| `backend: php artisan test` | ✔ `Tests: 378 passed (2110 assertions)` |
| `backend-go: go vet ./...` | ✔ exit 0 (run twice, on the WIP tree and on HEAD `988337c`) |
| `backend-go: go test ./...` | ✔ exit 0 (50 packages ok) |
| `backend-go: golangci-lint run` | ✔ `0 issues.` |
| `frontend: typecheck / lint / fsd:lint / lint:styles / lint:dark / test` | ✔ all exit 0. Lint gave 2 warnings (unused `_intention`, `_mode`); vitest `29 files, 262 tests passed`. This tree includes another person's uncommitted frontend work. |

## 2. Contract diff — `make contract ROUTES=all`

```
$ git worktree add --detach <scratch>/clean HEAD      # 24c1904, `git status --short` → 0 lines
$ cd <scratch>/clean/backend-go && make contract ROUTES=all
go run ./cmd/contract diff --routes 'all'
contract diff [auth,content,cycle,cycle-sweep,health,healthlog,home,messages,notifications,period,pregnancy,profile,public,reminders]: 986 passed, 0 failed, 0 allow-listed differences
make contract ROUTES=all  15.11s user 6.78s system 41% cpu 52.379 total

$ cd backend-go && make contract ROUTES=all            # HEAD 988337c (after T-M2-19 / T-M2-21b)
contract diff [auth,…,reminders]: 986 passed, 0 failed, 0 allow-listed differences
```

- **Allow-list:** `backend-go/contract/allowlist/` is empty, so nothing is accepted as a difference and no entry
  needs a deviation id.
- **Route coverage:** I compared `php artisan route:list --path=api/v1` (74 routes) with the method and path of
  every case in `contract/cases/*.yaml`, using parameter-aware matching. Result: `uncovered: 0`. T-M2-19's
  `openapi_test.go` also checks that Go registers all 74 routes and that the routes and the spec match in both
  directions.
- **Not covered by goldens:** the 429 from the throttle middleware can't be recorded, because the contract stack
  uses `CACHE_STORE=array` (T-M2-05). 429s raised by the controllers are covered. The admin API
  (`/api/admin/v1`) has no Laravel counterpart, so it is outside the contract.

### Deviations and allow-list status

No deviation is allow-listed, because the contract passes without any. So none of the deviations changes a recorded
golden. They only affect paths that have no golden: no `Accept` header, bad params, admin, and `/storage` edge
cases.

| Id | Summary | Status in deviations.md | What Go does now | Allow-listed |
|---|---|---|---|---|
| D-01 | `/reminders/enums` with a 3rd locale returns 500 | proposed | keeps Laravel's 500 (T-M2-12) | no |
| D-02 | non-numeric `{id}` → 404 instead of 500 | proposed | **mixed**: reminders return 500 (Laravel), pregnancy returns 404 (T-M2-16) | no |
| D-03 | deleting a user from admin revokes the user's tokens | proposed | **implemented** (T-M2-20) without user approval | no |
| D-04 | admin-created language files stored as JSON on the storage volume | proposed | implemented (T-M2-21, T-M2-21b) | no |
| D-05 | MessageSystem dead-column bugs | decided: preserve | preserved and pinned by tests | n/a |
| D-06 | admins log in again (new Go sessions) | decided | implemented | n/a |
| D-07 | Android crash-report endpoint returns 404 | decided | 404 | n/a |
| D-08 | 404/405 always JSON, even without an `Accept` header | proposed | implemented (T-M2-04) | no |
| D-09 | no relative or military-zone date strings | proposed | implemented in `civildate.ParseLenient` | no |
| D-10 | `/up` returns plain `OK` | proposed | implemented | no |
| D-11 | premium `/messages/daily` returns 500 | proposed: preserve | preserved (a golden locks the 500) | no |
| D-12 | admin self-modification → 422, English messages | proposed | implemented (admin API only) | no |
| D-13 | `/storage` error shapes and range handling | proposed | implemented | no |
| D-14 | admin article body sanitized when written | proposed | implemented | no |
| D-15 | a new default language copies the old default's translations | proposed | implemented | no |
| D-16 | banner `link_url` accepts `javascript:` | proposed | kept as Laravel | no |

**Action:** the user needs to decide on D-01…D-04 and D-08…D-16 before T-M2-25. D-03, D-08, D-09, D-10 and
D-12…D-15 already ship in Go. D-02 has to be made consistent across domains.

## 3. Token interop (T-M2-08 acceptance, rechecked)

```
$ CONTRACT_PORT=18090 make test-int PKG='./internal/auth/... -run CrossStack -v'
--- PASS: TestCrossStack_TokensInterop (1.64s)
    --- PASS: …/Laravel-issued_token_is_accepted_by_Go;_Laravel_logout_→_token_revoked_in_Go (0.78s)
    --- PASS: …/Go-issued_token_is_accepted_by_Laravel;_Go_revoke_→_token_revoked_in_Laravel (0.36s)
    --- PASS: …/expired_and_garbage_tokens_get_the_same_codes_on_both_stacks (0.24s)
ok  	github.com/ritme/backend-go/internal/auth	2.374s
```

The first two runs failed because of the harness, not because of the product:

1. The test runs in a fresh worktree, but `contract/.work/record.lock` is only created by the recorder, so the test
   fails with `no such file or directory`.
2. The contract DB had been `contract-reset` without the recorder's normalisation step. The `oauth_clients.id` was
   `01a0cdce-…` instead of `0199c0de-…c0ffee1`, and the test failed with `contract DB not normalised`.

To fix the second one I restored `contract/fixtures/dump.sql` into `contract-mariadb`. The key pair was already
identical: `sha1 cb99caf5…` on both sides. Follow-up **F-7**.

## 4. Web frontend smoke

**Setup.** The real `frontend/` was left untouched: another person has uncommitted work in it and is running
`next dev` on :3000. I rsynced a copy of the current tree, without `node_modules`, `.next` or `.env*`, into
scratch, symlinked `node_modules`, and ran `next dev -p 3124` with `NEXT_PUBLIC_API_BASE_URL` set first to Go and
then to Laravel.

The browser was headless Chrome driven over CDP, with a 400×860 mobile viewport. It ran with
`--disable-web-security`, because :3124 is not in either stack's CORS list. CORS headers themselves were verified in
T-M2-02.

- Go ran natively on a DB loaded from the fixture dump. Laravel was the contract stack.
- Both stacks ran on the real clock, and the real date equals `CONTRACT_TODAY`.
- Personas were `regular` (1004) and `pregnant_lmp_w8` (1014), both logged in through OTP with the code read from
  each DB.
- The new signup used mobile `09900009998`.

| Step | Go | Laravel | API calls identical | Screenshot |
|---|---|---|---|---|
| signup → OTP (code from the DB) → `/fa/onboarding/name` | ✔ | ✔ | yes (2) | differs (animation frame) |
| onboarding name → birthday → weight → height → period-len → cycle-duration → cycle-len → conditions → setting-up → `/fa/home` | ✔ | ✔ | yes (11) | byte-identical |
| `/fa/home` (regular) | ✔ | ✔ | yes (20) | byte-identical |
| `/fa/calendar` | ✔ ¹ | ✔ ¹ | yes (6) | byte-identical |
| `/fa/cycle` | ✔ ¹ | ✔ ¹ | yes (7) | differs (animation frame; I compared them by eye and they look identical) |
| `/fa/log` | ✔ | ✔ | yes (3) | byte-identical |
| `/fa/profile` | ✔ | ✔ | yes (3) | byte-identical |
| `/en/home` | ✔ | ✔ | yes (10) | byte-identical |
| `/fa/pregnancy` (tracker, week 8) | ✔ | ✔ | yes (16) | byte-identical |
| `/fa/pregnancy/log` | ✔ ² | ✔ ² | yes (3) | differs (animation frame; looks identical) |
| `/fa/home` (pregnant) | ✔ | ✔ | yes (10) | byte-identical |

¹ `Hydration failed because the server rendered … didn't match the client` happens on **both** stacks. It is a
frontend SSR issue in the current (WIP) tree and has nothing to do with Go (**F-8**).

² `404 /pregnancy/symptoms/2026-09-23` also happens on both stacks. It is the expected "no log for this day" answer
and is covered by the contract.

**Not clicked through in the UI (✘ → F-2):**

- period start/end
- saving a daily log
- articles list/detail
- settings: export and delete account
- pregnancy onboarding

The API side of all of these is covered by the contract, including the 21-step period write sequence and the
10-POST daily-log side-effect replay.

The screenshots were written to the session scratchpad (`shots/{go,laravel}/*.png`) and are not committed. The
script was `smoke.mjs`, based on the CDP recipe in the "Headless UI verification" memory.

## 5. Android smoke

**Not run (✘ → F-1, blocks T-M2-25).** A debug build of `application/` needs a JDK 21 Gradle build (offline
mirror) and an emulator or device pointed at Go. There was no emulator and Docker/disk were tight (~20 GB free),
so it was not attempted here. The WebView shell (`android-shell/`) smoke happens on staging after T-M2-25 moves a
group, as the task itself says.

The API surface the Android app uses is the same `/api/v1` covered by the contract. D-07 (crash reports → 404) is
unchanged.

## 6. Load comparison

**Setup.**

- Tool: `hey`. Each endpoint got 100 warm-up requests, then `-n 1000 -c 10`.
- Headers: `Accept: application/json`, `Accept-Language: fa`, `X-Test-Now: 2026-09-23T10:00:00+03:30`.
- Persona `regular` (1004), same fixture data (`contract/fixtures/dump.sql`) on both stacks.
- Laravel: the contract stack in Docker Desktop (PHP 8.4.25, Apache mod_php, opcache on, JIT off,
  `CACHE_STORE=array`, `APP_DEBUG=false`), on `127.0.0.1:18090`.
- Go: `go build ./cmd/api` from the clean worktree, running **natively on macOS** against a DB loaded from the dump
  on the go-test MariaDB 11.4 in Docker (:13317) and Redis (:16380), with the cycle-engine cache on (the default).

**Caveats.** This is not the identical-hardware comparison the task asks for.

- Go reaches MariaDB through Docker Desktop's port forwarder, so every query crosses the VM boundary. Laravel talks
  to its DB inside the Docker network. This penalises Go on endpoints with many queries.
- During a Go `/home` run, Go used 128% CPU (out of 1000%) and MariaDB 103%, so Go was waiting on DB round trips.
  It was not CPU-bound.
- Laravel runs without a cross-request cache (`array`), while Go uses its Redis cycle-engine cache.
- The client runs on the same machine as both servers.
- **Re-measure on staging, with both stacks in containers on the same host (F-3).**

### Concurrency 10 (ms; RPS)

| Endpoint | Laravel p50 / p95 / p99 | Laravel RPS | Go p50 / p95 / p99 | Go RPS | Go ÷ Laravel RPS |
|---|---|---|---|---|---|
| `GET /cycle/today` | 19.8 / 33.3 / 41.2 | 466 | 14.8 / 21.1 / 26.4 | 648 | 1.39× |
| `GET /cycle/month/2026/9?view=calendar` | 27.5 / 53.1 / 74.4 | 327 | 15.3 / 25.5 / 49.3 | 583 | 1.78× |
| `GET /cycle/month/2026/9` (full) | 36.6 / 66.6 / 91.8 | 235 | 23.8 / 42.3 / 70.2 | 379 | 1.61× |
| `GET /home` | 37.2 / 73.0 / 97.0 | 239 | 41.4 / 61.6 / 76.4 | 229 | **0.96×** |
| `GET /messages/daily` | 25.2 / 53.8 / 75.8 | 328 | 22.6 / 43.3 / 62.8 | 393 | 1.20× |
| `GET /profile` | 15.7 / 36.4 / 64.0 | 523 | 9.0 / 14.5 / 25.1 | 1031 | 1.97× |
| `GET /auth/user` (token-validation path) | 15.3 / 41.1 / 76.9 | 521 | 7.6 / 12.2 / 14.0 | 1245 | 2.39× |
| `GET /banners` | 14.7 / 38.4 / 66.8 | 548 | 8.1 / 12.5 / 22.1 | 1140 | 2.08× |

- All 16,000 measured requests returned `[200]`.
- A second Go `/home` run of 3,000 requests at c=10 gave 257 RPS, p50 37.1 ms, p95 53.5 ms.
- `send-otp` and `verify-otp` were not load-tested. They are throttled by design (5/min and 10/min per IP) and
  write SMS/OTP rows. `/auth/user` stands in for "auth".

### Concurrency 1 (300 requests, ms p50 / p95 / p99)

| Endpoint | Laravel | Go |
|---|---|---|
| `/home` | 30.1 / 38.0 / 44.9 | 25.2 / 45.0 / 61.8 |
| `/messages/daily` | 19.8 / 26.6 / 28.8 | 12.0 / 15.4 / 22.2 |
| `/cycle/today` | 16.7 / 28.8 / 48.6 | 9.1 / 13.6 / 22.0 |

### DB queries per request

Measured as the change in `Questions` on each MariaDB for one request.

| Endpoint | Laravel | Go |
|---|---|---|
| `/home` | 25 | 21 |
| `/messages/daily` | 19 | 10 |
| `/cycle/today` | 12 | 8 |
| `/profile` | 10 | 6 |

### Memory

- Go RSS: 35 MB idle, 36 MB under `/home` load.
- Laravel container: 108.6 MiB idle, 166 MiB peak during the run (`docker stats`, sampled every 3 s).

**Reading.** Go is not slower anywhere once the port-forward penalty is taken into account. `/home` is the only
endpoint at parity: 21 sequential DB round trips, and the tail is longer at c=1. Look at batching its section
queries or loading sections in parallel (**F-4**, not a blocker).

## 7. Follow-ups (proposed; create as `T-M2-NNb` tasks)

| # | Follow-up | Owner task | Blocks T-M2-25? |
|---|---|---|---|
| F-1 | Android smoke: debug build pointed at Go (log in, main screens, log a period); WebView shell on staging per group | T-M2-24b (new) | **yes** |
| F-2 | Web write-flow smoke: period start/end, daily-log save, articles, settings export/delete, pregnancy onboarding (UI) | T-M2-24c (new) | **yes** (could be folded into the T-M2-25 per-group smoke) |
| F-3 | Re-run the load comparison on staging with both stacks containerised on the same host and DB | T-M2-25 | no |
| F-4 | Reduce `/home` DB round trips (21 sequential queries) or load sections in parallel | T-M2-18 | no |
| F-5 | User decision on the proposed deviations D-01…D-04 and D-08…D-16. Make D-02 consistent (pregnancy 404 vs reminders 500) | T-M2-12 / T-M2-16 | **yes** (approval) |
| F-6 | Diff the goose baseline against a `--no-data` dump of the staging DB (still open from T-M2-03; needs user OK to read the server) | T-M2-03 | **yes** |
| F-7 | Cross-stack test harness: create `contract/.work/` if missing; normalise or restore the contract DB itself instead of failing | T-M2-05 | no |
| F-8 | Frontend hydration mismatch on `/calendar` and `/cycle` (same on both backends; frontend WIP tree) | frontend owner | no |

## 8. Open items and risks from PROGRESS.md (M2)

**Must be settled before or during the staging rollout:**

- **T-M2-09:** the staging deploy is blocked. It needs one prod proxy recreate (user OK), and `ADMIN_SEED_PASSWORD`
  in `.env`.
- **T-M2-23:** admin-web screens are `todo`, and T-M2-25 depends on them.
- **T-M2-22:** the admin-web Docker image has never been built (Docker had crashed and the disk was full). T-M2-25
  also needs nginx wiring for admin.
- **T-M2-21 vs T-M2-09:** admin uploads and language files need the `backend-storage` volume to be **writable** for
  backend-go. The compose file and `cutover.md` currently mount it read-only.
- **T-M2-20:** new env vars `ADMIN_HOSTS`, `ADMIN_WEB_ORIGINS`, `ADMIN_COOKIE_SECURE`. Without `ADMIN_HOSTS` the
  admin API disables itself outside local (seen in this run's log). `vhost-admin.inc` must proxy `/api/admin/` to Go.

**Before `auth` moves:**

- Drain Laravel's OTP queue (`cutover.md`).
- On prod, configure `real_ip` for the CDN. The XFF-based rate-limit bypass carried over from Laravel (T-M2-08) is
  still open.

**Known quirks deliberately kept:**

- D-05 and D-11: premium `/messages/daily` returns 500 on some dates.
- Throttle-middleware 429s are not covered by the contract.
- The daily card keeps Laravel's hard-coded fa/en.

**Cache and cross-stack staleness:**

- The language registry in Redis has no TTL in `i18n.Registry` (T-M2-06). Content routes set a 5-minute TTL
  (T-M2-10), and admin writes flush both stacks (T-M2-21).
- Go and Laravel keep separate cycle caches. A write on one stack is picked up by the other through
  `calculation_version`, which the ported write paths bump.

**Code debt (not blocking):**

- The `markRecalculated` and LMP queries are duplicated in the profile and healthlog stores (T-M2-12).
- `messages.StoreSource` duplicates the cycle adapters, and `content.Repository` could replace
  `profile.StoreMessageContents` (T-M2-17).
- `/cycle/status` has a floor of about 7 ms (T-M2-15).
- The on/off contract test builds the whole API (~60 s).

**Behaviour notes:**

- Telegram name-change notices are sent asynchronously; response bodies are unchanged (T-M2-11).
- `/up` returns plain `OK` (D-10). The prod/stage healthchecks use `/app/api healthcheck`.
- `mobile_verified_at` stays null, matching Laravel (T-M2-08).

**Stale docs:** the `contract.md` "Open items" entry about `SmsService` not mapping `log` is out of date. T-M2-01
wired it.

**Test-only artifact seen in this run:** Laravel `send-otp` with an `X-Test-Now` earlier than the real `created_at`
of an existing OTP returns a negative `retry_after` (`-14636`). It can't happen in production, where the clock
header is ignored.

## 9. Environment and cleanup

Everything I started was stopped and removed:

- Go API (:18321), frontend copy (:3124), headless Chrome (:9333)
- the `t24_load` DB and the `ritme-go-t24:*` Redis keys
- the scratch worktree and the frontend copy

`contract-mariadb` was restored from `contract/fixtures/dump.sql`, so the smoke signup user is gone and the DB is
back in its normalised fixture state.

Left untouched: the Docker stacks (the contract stack on 18090 and the go-test stack) and the user's `frontend/`
dev server. No images were built, and no prune or volume deletion was run.

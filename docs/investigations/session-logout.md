# Investigation — premature logout (T-M1-01)

Date: 2026-09-19 · Scope: read-only (prod `root@89.251.8.115`, `/opt/ritme`, compose project `ritme`; staging `ritme-stage`)
· No code changed.

## TL;DR

The server never ends a session early. Every token is issued for 365 days, the signing keys have not changed since
the prod volume was created, nothing revokes tokens except an explicit logout, and **no real user got a 401 from the
API** in the log windows we have (backend since 2026-08-31, proxy since 2026-09-07). Every "signed out" case we found
in the logs started on the client, and all of them were on **iOS**.

Ranked root causes:

| # | Root cause | Status | Fix → task |
|---|---|---|---|
| 1 | **The app's sign-in check reads only the `ritme_auth` flag cookie, which JavaScript writes.** If that cookie is missing, the edge middleware sends the user to `/signup`. `SessionGuard` then puts the cookie back but **leaves the user on the sign-in screen**, so they type an OTP again even though a valid token is still in `localStorage`. On WebKit (iOS Safari, Home Screen apps), a cookie written by JavaScript is capped at 7 days. Any cookie loss (cap, flush, "clear cookies") therefore looks like a logout. | confirmed in code · WebKit 7-day cap is documented but not reproduced on a device | T-M1-03 |
| 2 | **iOS keeps separate storage per context.** Safari tabs, the Home Screen web app and in-app browsers (Instagram) each have their own storage. A user who signs in inside Safari and then opens the Home Screen icon starts with no session. | **confirmed in prod logs** (user 36, 2026-09-17/19) | T-M1-03 (UX mitigation). The platform behaviour itself can't be fixed. |
| 3 | The deployed prod bundle (v1.0.2, built 2026-09-01) clears the token on **any** 401, not just JSON 401s. Today nothing sends real users a 401, so this is a latent risk and not an observed cause. | code confirmed; observed impact: none | T-M1-03 (+ `error_code` from T-M1-02) |
| — | Token lifetime, key rotation, `APP_KEY`, revocation, the service worker, the Android shell | ruled out (evidence below) | T-M1-02 / T-M1-04 re-scoped |

---

## Hypotheses and evidence

### H1 — Tokens are issued with a short lifetime · **RULED OUT**

Passport is v13.4.1. It defaults to `P1Y` for all three lifetimes, and `AppServiceProvider::boot()` sets one year
again, turned into a `DateInterval` when the app boots (`vendor/laravel/passport/src/Passport.php:316-325`). The
`PersonalAccessGrant` is registered with `Passport::personalAccessTokensExpireIn()`
(`PassportServiceProvider.php:140`), which is the grant `createToken()` uses.

Prod DB (read-only SELECT):

```
$ mariadb … < q1.sql        # SELECT TIMESTAMPDIFF(DAY, created_at, expires_at) life_days, revoked, COUNT(*) FROM oauth_access_tokens GROUP BY 1,2
| life_days | revoked | COUNT(*) |
|       365 |       0 |       54 |
|       365 |       1 |       13 |
# newest rows
| 09873772 | 47 | 01a00acd-… | auth_token | 0 | 2026-09-19 13:22:08 | … | 2027-09-19 13:22:08 | 365 |
| 68eb1d61 | 46 | 01a00acd-… | auth_token | 0 | 2026-09-19 10:57:26 | … | 2027-09-19 10:57:26 | 365 |
```

JWT `exp`: we can't decode a prod JWT because only the token id is stored. Instead we minted one against a
**throwaway copy** of the local sqlite DB, using the same code path (`createToken`):

```
$ cp database/database.sqlite $SCRATCH/copy.sqlite && DB_DATABASE=$SCRATCH/copy.sqlite php artisan tinker --execute='…createToken("auth_token")…decode payload…'
passport=v13.4.1
iat=2026-09-19T15:51:58+03:30 exp=2027-09-19T15:51:58+03:30 days=365
row.expires_at=2027-09-19 15:51:58 revoked=false
interval=0 y 11 m 30 d 23 h 59 i 59 s      # boot-time diff(now()->addYear()) — 1 s short of a year, harmless
```

Side note: `oauth_access_tokens.created_at` is stored in `Asia/Tehran` (`config('app.timezone')`), and the DB clock is
UTC. That is why the newest token looks "ahead" of `NOW()`. It is not a bug, but keep it in mind when correlating
with nginx logs, which are in UTC.

### H2 — Tokens get revoked by something other than logout / block / delete · **RULED OUT**

`revoke()` has exactly three callers: `OtpAuthController::logout` (`:381`), admin block (`Admin/UserController.php:82`)
and account deletion (`ProfileController.php:413`). There is no scheduled `passport:purge`, because
`routes/console.php` only has `inspire`.

```
$ SELECT COUNT(*) users, SUM(blocked_at IS NOT NULL) blocked FROM users;      -> 40 | 0
$ SELECT DATE(updated_at), COUNT(*) FROM oauth_access_tokens WHERE revoked=1 GROUP BY 1;
  2026-08-21 1 · 08-25 3 · 08-26 2 · 08-27 1 · 08-29 1 · 09-01 1 · 09-08 2 · 09-12 2
$ docker logs ritme-backend-1 | grep -cE 'POST /api/v1/auth/logout[^"]*" 200'   # log starts 2026-08-31
  5
```

Since 2026-08-31 there are 5 revocations (09-01 ×1, 09-08 ×2, 09-12 ×2) and 5 successful logout calls, so every
revocation is an explicit logout. No users are blocked.

The strongest single piece of evidence: users who "got logged out" hold **several un-revoked tokens**. The server still
considered their old session valid when they signed in again:

```
| user_id | tokens | revoked | first               | last                | span_h |
|      36 |      3 |       0 | 2026-09-15 17:04:36 | 2026-09-19 07:43:16 |     86 |
|      12 |      2 |       0 | 2026-08-29 16:50:03 | 2026-08-31 17:23:24 |     48 |
|       6 |      2 |       1 | 2026-08-18 17:01:27 | 2026-08-20 10:43:54 |     41 |
```

### H3 — Signing keys / personal-access client change across deploys · **RULED OUT** (with a hygiene finding)

```
$ docker exec ritme-backend-1 ls -la --time-style=full-iso storage/oauth-*.key
-rw------- www-data 3322 2026-07-07 14:35:48 storage/oauth-private.key
-rw-rw---- www-data  812 2026-07-07 14:35:48 storage/oauth-public.key
$ … openssl pkey -pubin -in storage/oauth-public.key -outform DER | sha256sum | cut -c1-16
b501f48dc5d43bdc            # identical in ritme-queue-1
$ docker volume inspect ritme_backend-storage --format '{{.CreatedAt}}'   -> 2026-08-16T13:33:40Z
$ docker inspect … created  -> backend/queue 2026-08-31, frontend 2026-09-01, proxy 2026-09-07 (restarts=0)
$ docker logs ritme-backend-1 | grep -iE 'passport|oauth|key'   -> nothing (boot ran "Nothing to migrate", seeders, swagger)
$ SELECT id, name, grant_types, revoked, created_at FROM oauth_clients;
| 01a00acd-… | Ritme Personal Access | ["personal_access"] | 0 | 2026-08-16 17:10:29 |     # exactly one client, never recreated
```

The key pair has not changed since the volume was created, and the 2026-08-31 redeploy neither regenerated keys nor
created a client. `APP_KEY` has the same hash in both containers (`sha256 a6062d97d49f…`, truncated). Passport doesn't
sign JWTs with `APP_KEY` anyway; it uses the RSA pair.

**Hygiene finding (not a logout cause, but it matters for T-M1-02):** the prod key pair is the **developer laptop's**
key pair:

```
$ openssl pkey -pubin -in backend/storage/oauth-public.key -outform DER | sha256sum | cut -c1-16   # local
b501f48dc5d43bdc
$ ls -la /opt/ritme/backend/storage/oauth-*.key      # on the server, owned 501:staff, mtime 2026-07-07
```

`deploy.sh` rsyncs `backend/storage/oauth-*.key`. The keys are only git-ignored (`backend/.gitignore:19`), not
rsync-excluded. `backend/Dockerfile` then runs `COPY . .` and the keys end up in the image, and Docker seeded the fresh
named volume from the image on 2026-08-16. Staging generated its own pair (`adf055deda38b608`, 2026-08-30), so staging
is fine. Anyone holding the laptop's `storage/` can mint prod tokens. Rotating the key now would invalidate every live
token, so any rotation has to be planned together with the refresh endpoint.

A smaller finding: `entrypoint.sh` runs `passport:client --personal` when `tinker` prints nothing (`[ -z "$clients" ]`).
If the DB or tinker fails for a moment, that creates a **second** client. It would not log anyone out, but it is not
strictly idempotent.

### H4 — The API or proxy sends valid users a JSON 401 · **RULED OUT for the observed window**

```
$ docker logs ritme-backend-1 | grep -E '"[A-Z]+ /api/v1[^ ]* HTTP/[0-9.]+" 401 '      # apache, since 2026-08-31
[31/Aug/2026:08:38:03 "GET /api/v1/banners 401     # = backend container start → deploy.sh smoke test
[01/Sep/2026:13:15:06 "GET /api/v1/banners 401     # = frontend deploy smoke test
[07/Sep/2026:15:13:37 "GET /api/v1/banners 401     # = proxy deploy smoke test
total api lines: 6064
$ docker logs ritme-proxy-1 | grep /api/v1 | awk '$9==401'      # proxy, since 2026-09-07
  9 × GET /api/v1/banners  UA curl/8.x                (deploy smoke tests)
  6 × GET /api/v1/.env     UA scanner, HTML body 581 B (staging Basic-auth gate)
```

Out of ~6,000 API calls, zero real-user requests got a 401. Across the whole window, API responses were: 3,498 × 200,
1,347 × 204, 172 × 404, 102 × 444, 15 × 401 (above), 13 × 201, 6 × 400, 2 × 429, 1 × 502, 1 × 500, 1 × 422. The only
explicit 401 in app code (`ProfileController.php:174`) sits behind `auth:api` and can't fire.

The code path is still dangerous, and **the deployed prod bundle is older than the repo**:

```
$ docker exec ritme-frontend-1 grep -oE 'interceptors.response.use.{1,200}' /app/.next/static/chunks/*.js
…interceptors.response.use(e=>e,e=>{…(null==(n=e.response)?void 0:n.status)===401&&(0,r.Tp)(),Promise.reject(e)})
```

Prod clears the token on **any** 401. The `content-type: json` guard was added to the repo in 3016230 (2026-09-07), and
only staging runs it. Both carry version `1.0.2`. Limitation: nginx's `log_format main` has no `$host`, so we can't
split prod and staging 401s from the log line alone. Staging traffic was told apart by its referer.

### H5 — Client storage loses the session · **CONFIRMED (iOS)**

#### H5a — iOS storage partitions (Safari tab vs Home Screen app vs in-app browser) · **CONFIRMED**

Timeline of prod user 36 (iPhone, iOS 26 — WebKit freezes the UA at `iPhone OS 18_7`, `Version/18.7.6 Mobile/15E148`).
This is the proxy log with IPs truncated. Token rows match to the second: 13:34:36Z ≙ 17:04:36 Tehran, 11:28:08Z ≙
14:58:08, 04:13:16Z ≙ 07:43:16.

```
2026-09-15 13:34:36Z verify-otp 200   UA "… Instagram 446.0.0.28.66 …"         → token 345ec1a4 (Instagram in-app browser)
2026-09-17 11:27:32Z GET / → /fa → /fa/splash → sw.js install → welcome → signup 200   (no ritme_auth: new context)
2026-09-17 11:28:08Z verify-otp 200 → /fa/home, API 200s                          → token f2a51fdf
2026-09-17 11:28:27Z GET / (referer "-")  → /fa/splash → offline.html fetched BY sw.js (a 2nd SW install) → /fa/signup 200
2026-09-17 11:28:44Z send-otp 429, 11:28:53Z send-otp 429                          (re-sign-in blocked by OTP cooldown)
2026-09-17 11:40:10Z banners/profile 200, /fa/home nav                            (back in the first, signed-in context)
2026-09-19 04:12:40Z GET / → sw.js → /fa 307 → /fa/splash 304 (SW-controlled) → /fa/signup 200
2026-09-19 04:13:16Z verify-otp 200                                               → token 74eef7bf
2026-09-19 08:08:28Z GET / → /fa/splash → /fa/signup 307 → /fa/home               (session kept 4 h later)
```

Nineteen seconds after signing in, the user launched the app in a **second storage context**. It installed its own
service worker (the second `offline.html` precache) and had no `ritme_auth` cookie. That is the Home Screen web app
added right after signing in inside Safari. The OTP cooldown then blocked the new sign-in (429), and the user went back
to the Safari tab. Two days later they opened the Home Screen app, which had never held a session. The "41-hour logout"
is this context split, not an expiry. The server still holds f2a51fdf un-revoked. The earlier Instagram → Safari
switch is the same effect.

#### H5b — `ritme_auth` flag lost while the token is still in `localStorage` · **CONFIRMED (code); WebKit trigger documented, not device-reproduced**

- `middleware.ts:61-66` checks `request.cookies.has('ritme_auth')` and nothing else. The cookie is written with
  `document.cookie` (`token.ts:36`). WebKit's ITP caps every script-written cookie at **7 days** of expiry, whatever
  `max-age` says. This is documented WebKit behaviour; we did not reproduce it on a device in this task.
- Cold launch: `/` → `/fa` → splash → `/fa/signup`. Only the middleware sends a signed-in user on to `/home`
  (log: `/fa/signup 307 → /fa/home`). Without the cookie the signup page renders (`/fa/signup 200`).
- `SessionGuard.tsx:44-48` sees a token with no flag and calls `setAuthToken(token)`, which restores the cookie. But it
  **does not navigate**. `SignupPage` and `OtpPage` never check `getAuthToken()` (grep shows no hits), so the user is
  shown the sign-in form, enters an OTP, and gets another token. This matches the "multiple un-revoked tokens"
  pattern in H2.
- The flag is only rewritten when it's missing, never refreshed while present. So the WebKit 7-day cap counts from
  sign-in and is never extended.
- The reverse case (flag present, token gone) clears the flag and redirects to signup (`SessionGuard.tsx:50-56`).
  That is correct, since the token really is gone.

User 12 (2026-08-29 → 08-31, 48 h, both tokens un-revoked) was also an iPhone (`Version/26.5.2 Mobile/15E148`). The
proxy logs from before 2026-09-07 are gone, so we can't tell whether it was H5a or H5b. Status: unknown, consistent
with both.

#### H5c — Android WebView shell loses storage · **RULED OUT as a recurring cause**

Survival of cold launches after sign-in, per platform, from the proxy log (2026-09-07 → 09-19). "Kept" means the
launch reached `/fa/signup 307 → /fa/home`; "lost" means it rendered `/fa/signup` or `/fa/welcome` with 200.

```
platform       devices_with_login  launches_kept  launches_lost  max_days_session_kept
android-shell          24               21              2              11.1
ios                     4                1              2               1.9
android-browser         1                0              0               0.0
LOST android-shell 2026-09-16 → /fa/welcome   explicit logout before (09-08)
LOST android-shell 2026-09-12 19:39 → /fa/welcome   32 min after sign-in; intro cookie ALSO gone + SW re-installed
LOST ios           2026-09-17 11:28 / 2026-09-19 04:12   (H5a above)
```

The one unexplained shell event wiped the **whole** WebView profile at once: auth flag, intro cookie and service
worker. That points to an app reinstall or "clear data" (`android:allowBackup="false"`, so a reinstall starts empty),
not to an expiry. The shell loads `https://web.ritme.app/`, keeps one data directory, never calls
`clearCache`/`clearHistory`/`WebStorage.deleteAllData`, and flushes `CookieManager` in `onPause`
(`MainActivity.kt:387-392`).

Residual theoretical risk: Chromium commits `localStorage` lazily. A process killed within about a second of
`setAuthToken` could keep the cookie but lose the token. That case signs the user out cleanly (H5b reverse path).
Nothing in the logs shows it.

#### H5d — Origin/domain change hides the token · **RULED OUT for current users (one-off, historical)**

On 2026-08-16 the app moved to `web.ritme.app`, with a fresh DB (memory `ritme-server-deployment`). Every token from
before the move was lost once, and all current tokens were issued after it (`MIN(created_at)` = 2026-08-16 21:33). The
origin hasn't changed since. `http→https` is already handled by `SessionGuard`.

### H6 — `SessionGuard` repair logic clears a valid session · **RULED OUT (it doesn't clear), but see H5b (it doesn't route either)**

- The `visibilitychange`/`pageshow` revalidation (`:66-89`) only redirects when `getAuthToken()` is null. It never
  clears anything. No race: `getAuthToken` reads the in-memory mirror, then `localStorage`.
- `clearAuthToken` has four callers: the 401 interceptor, `useLogout`, account deletion, and the flag-without-token
  repair. There is no `localStorage.clear()` anywhere.

### H7 — Service worker / update flow · **RULED OUT**

`scripts/sw.template.js:87-91` (identical to the deployed `/app/public/sw.js`) handles navigations network-first and
only falls back to `offline.html`. It never serves a cached `/signup` or a cached redirect. `UpdateGate` and the SW
`activate` step delete only Cache Storage (`caches.delete`), never `localStorage` or cookies.

### Other findings on the way

- `app/[locale]/page.tsx:17` checks a `ritme_onboarded` cookie that nothing sets (dead branch). Every cold launch
  therefore goes through splash → signup and depends on the middleware redirect in H5b.
- The OTP cooldown (429) blocked a legitimate re-sign-in 36 s after the previous one (user 36). A lost session costs
  more than one retry.
- `nginx log_format main` has no `$host` and no request time. Adding `$host` would make the next investigation of this
  kind trivial.

---

## Fix recommendations → tasks

**T-M1-03 (frontend) — this is where the real fix lives:**
1. Treat the `localStorage` token as the source of truth on the client. When `/splash`, `/welcome`, `/signup` or
   `/otp` mount while a token exists, restore the flag and `router.replace` to `/home` (or to the pending onboarding
   step). Today `SessionGuard` only restores the flag.
2. Keep the flag cookie from lapsing. Set it with an HTTP `Set-Cookie` from a same-origin Next route handler (for
   example `POST /session/flag`): a server-set cookie isn't subject to WebKit's 7-day cap on script-written cookies.
   Also re-assert it on every app start and resume, not only when it's missing.
3. 401 handling: clear only on a Laravel JSON 401 (`error_code` from T-M1-02, or `message: "Unauthenticated."`). The
   repo guard exists, but **prod still runs the any-401 bundle**. Ship it.
4. `navigator.storage.persist()` after sign-in (as already scoped).
5. iOS: the platform won't share storage between Safari, the Home Screen app and in-app browsers. Mitigate it: the iOS
   install hint should say the Home Screen app needs one sign-in, and a signed-in Safari user should be nudged to add
   the app before, not after, relying on it. Consider an "open in Safari" hint when the UA is an Instagram/Telegram
   in-app browser.
6. Remove the dead `ritme_onboarded` branch, or implement it.

**T-M1-02 (backend) — no server-side cause; make it hardening:**
1. Lifetime is already 365 days. Keep the tests that lock it in (token row + JWT `exp`).
2. Stable `error_code` on 401 so the client can tell a real logout apart from other 401s (needed by T-M1-03 §3).
3. The sliding refresh endpoint is optional for M1. With 365-day tokens and zero expiry-driven logouts, it only matters
   close to 2027-08. Keep it if cheap.
4. **Key hygiene:** exclude `backend/storage/*.key` from `deploy.sh` rsync and from the Docker build context
   (`backend/.dockerignore`), so prod never again inherits a laptop key. **Do not rotate the current pair** as part of
   this task: that would log out every user. Rotation needs its own plan (dual-key window or a forced re-sign-in).
5. `entrypoint.sh`: only create a personal client when the count is a real `0`, and not when tinker failed and
   printed nothing.

**T-M1-04 (Android shell) — not implicated:** 21 of 22 post-sign-in cold launches kept the session, and it held for up
to 11.1 days. The one loss was a full profile wipe (reinstall / clear data). Optional hardening: also
`CookieManager.flush()` in `onStop`. Verify that store updates install over the existing app (same signing key), so
they don't force an uninstall that wipes the profile.

## Open items / limits

- Proxy access logs exist only from 2026-09-07 (the container was recreated) and backend logs from 2026-08-31. Older
  complaints (users 6 and 12) can't be traced to a single cause.
- The WebKit 7-day cap on `document.cookie` (H5b) comes from WebKit's published ITP behaviour. It was not reproduced on
  a real iPhone here. A device test belongs in T-M1-03's manual check: sign in, advance the clock or wait 8 days,
  reopen the Home Screen app.
- `Mobile/15E148` vs a real build token (`Mobile/23F84`) *suggests* Home Screen or WebView versus a Safari tab, but
  this isn't a reliable signal.
- Only 4 iOS devices signed in during the window, so the sample is small. The pattern is consistent, but not
  statistically strong.

---

## Fixed in (closed out in T-M1-05, 2026-09-19)

| Root cause / finding | Fix | Commit |
|---|---|---|
| #1 flag cookie lost → user stuck on `/signup` with a valid token | The `localStorage` token is the source of truth. `SessionGuard` (`shared/session/reconcile.ts`) re-asserts the flag on every start, route change and resume, and moves signed-in users off `/splash`, `/welcome`, `/signup` and `/otp`. The flag is now **server-set** by `POST /api/session/flag` (1 year, `SameSite=Lax`), so WebKit's 7-day cap on script cookies doesn't apply. | 0dcf8e7 (T-M1-03); stage vhost route in 876a408 (T-M1-08) |
| #2 iOS storage partitions | Can't be fixed (platform behaviour). Mitigated: the iOS install guide says the Home Screen app needs one sign-in, in-app browsers get an "open in browser" hint, and `navigator.storage.persist()` runs after sign-in. | 0dcf8e7 (T-M1-03), 008a685 (T-M1-07) |
| #3 any 401 cleared the token | Backend: JSON 401s carry `error_code` ∈ `token_revoked` / `token_expired` / `unauthenticated`. Client (`shared/session/unauthorized.ts`): the session ends only on a JSON 401 for a request that carried the current token, with one of those codes (or Laravel's legacy body). | 35e972c (T-M1-02) + 0dcf8e7 (T-M1-03) |
| Lifetime / future expiry | Fixed `P365D` lifetime (`config/passport.php`), tested. Sliding refresh: `POST /api/v1/auth/refresh-session` issues a fresh 365-day token only within 30 days of expiry. The client (`features/auth` `SessionRefresher`) calls it on start/resume. | 35e972c, 0dcf8e7 |
| Laptop key pair shipped to prod | `storage/*.key` is excluded from rsync (`deploy.sh`, `deploy-stage.sh`) and from the Docker context (`.dockerignore`). `entrypoint.sh` never regenerates keys and creates the personal client only when the count is really 0. | 35e972c (T-M1-02) |
| Android shell | Not implicated. `CookieManager.flush()` also runs in `onStop`. `android-shell/` is git-ignored, so this change lives only in the working tree. | c43d023 (T-M1-04) |

**End-to-end check (T-M1-05, local dev):** backend on :8050 with a scratch copy of the sqlite DB, `next dev` on
:3150, headless Chrome over CDP.

```
(a) UI sign-in (OTP)       → /fa/home; stored JWT exp = +365.0 d; POST /api/session/flag 204
                              Set-Cookie: ritme_auth=1; Path=/; Max-Age=31536000; SameSite=lax
(b) seeded a 10-day JWT    → on load POST /auth/refresh-session 200 {"refreshed":true,…,"expires_at":"2027-09-19…"}
    (minted with PASSPORT_TOKEN_LIFETIME_DAYS=10)   localStorage token REPLACED (jti 4e047b4a → 9f991313, 365.0 d left)
                              DB: 4e047b4a revoked=1, 9f991313 revoked=0; 0 × 401; a later resume sends no 2nd refresh
(c) revoked 9f991313 in DB → API: 401 application/json {"message":"Unauthenticated.","error_code":"token_revoked"}
                              app: 11 parallel 401s → one clean sign-out: /fa/signup, token null, ritme_auth gone;
                              a cold start lands on /fa/welcome and makes no further 401 calls
```

**What remains:**
- **Staging and prod verification happens in T-M1-14.** T-M1-05 deployed nothing, because the working tree held
  other agents' uncommitted work. Prod still runs the v1.0.2 any-401 bundle until T-M1-14 ships.
- **Manual iPhone test** (Home Screen app): sign in once, check it stays signed in after more than 8 days, and confirm
  the flag cookie is the server-set one.
- **Key rotation decision:** the prod key pair is still the laptop's (`b501f48dc5d43bdc`). Rotating it signs out
  every user. Choose between a planned forced re-sign-in (after T-M1-14 ships and SMS works) and a dual-key
  validator. Also delete the stray `/opt/ritme/backend/storage/oauth-*.key`.
- Smaller follow-ups from T-M1-02: admin `UserController::destroy` doesn't revoke tokens, the `ritme_onboarding`
  cookie is still set from JS, nginx `log_format` lacks `$host`.

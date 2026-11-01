# Admin API (`/api/admin/v1`)

The Blade admin panel (`backend/routes/admin.php`) is replaced by a JSON API in `backend-go` and a separate
Next.js app, `admin-web/` (T-M2-22/23). This page is the contract between them. T-M2-20 built the core (auth,
dashboard, users, admins); T-M2-21 adds content CRUD, uploads and languages on the same conventions.

Code: `backend-go/internal/admin/{httpadmin,auth,dashboard,users,admins}`, routes in
`internal/http/routes_admin_core.go`, queries in `db/queries/admin/`. T-M2-21:
`internal/admin/{content,content/form,media,messages,languages}`, routes in `internal/http/routes_admin_content.go`,
queries in `db/queries/admin/content.sql` (endpoints in §11). T-M4-03: `internal/admin/checkups` (checkup-type
catalog, §12). T-M7-06: `internal/admin/pregnancy` (pregnancy v2 content) and `internal/admin/messages/registry`
(registered message groups, create-message), §13.

## 1. Where it is served

- Base path **`/api/admin/v1`**, served **only on the admin host** (`adpanell.ritme.app`, stage: the stage host).
  Any other `Host` gets the framework 404, so `api.ritme.app/api/admin/...` is a 404 (as `/admin` is today).
- The Go side enforces this with `ADMIN_HOSTS`; nginx must also route it (see §10).
- `admin-web` is expected to be served on the same admin host, so browser calls are **same-origin** and need no
  CORS. For local development (`admin-web` on another port) list its origin in `ADMIN_WEB_ORIGINS`.

| Env var | Meaning | Default |
|---|---|---|
| `ADMIN_HOSTS` | Comma-separated host names the admin API answers on. Empty: any host when `APP_ENV` is `local`/`testing`, otherwise the admin API is **disabled** (fail closed, logged at start-up). | empty |
| `ADMIN_WEB_ORIGINS` | Comma-separated origins allowed cross-origin **with credentials** (e.g. `http://localhost:3001`). | empty |
| `ADMIN_COOKIE_SECURE` | `false` only for plain-http local dev. When true, cookies are `Secure` and carry the `__Host-` prefix. | `true` |

## 2. Envelope and errors

Success (always JSON, `Content-Type: application/json`):

```json
{"success": true, "message": "User blocked.", "data": { … }}
```

`message` is optional. `data` is an object (or `null`).

Failure — every error the admin API itself produces:

```json
{"success": false, "message": "CSRF token mismatch.", "error_code": "csrf_mismatch"}
```

`message` is English and meant for logs; **admin-web translates `error_code`**.

| Status | `error_code` | When |
|---|---|---|
| 401 | `unauthenticated` | no session cookie |
| 401 | `session_expired` | cookie present but the session is gone (idle timeout, logout elsewhere, password change) |
| 401 | `admin_inactive` | the admin was deactivated or deleted; the session is destroyed |
| 403 | `forbidden` | editor calling a super-only endpoint |
| 403 | `origin_forbidden` | mutating request with a foreign `Origin` header |
| 404 | `not_found` | record not found (`"User not found."`), also non-numeric ids |
| 415 | `unsupported_media_type` | login body not `application/json` |
| 419 | `csrf_mismatch` | missing / wrong `X-CSRF-Token` on a mutating request |
| 422 | `validation_failed` | field errors, see below |
| 422 | `invalid_credentials` | login failed (same answer for unknown email, wrong password, inactive account) |
| 422 | `cannot_modify_self` | demoting, deactivating or deleting your own admin account |
| 422 | `default_language_protected` | deleting or deactivating the default language (T-M2-21) |
| 422 | `in_use` | deleting a checkup type that users have records of (T-M4-03, §12); body has `records_count` — or a pregnancy care item that appointments reference (T-M7-06, §13); body has `appointments_count` |
| 429 | `too_many_attempts` | login throttle; body has `retry_after` (seconds), header `Retry-After` |

Validation (422) adds the Laravel error bag; messages are in the **default language** (the admin chain pins
`setlocale:default`, like the Blade panel):

```json
{"success": false, "message": "فیلد ایمیل الزامی است. (and 1 more error)", "error_code": "validation_failed",
 "errors": {"email": ["…"], "password": ["…"]}}
```

Unknown routes and methods fall through to the framework's JSON 404/405 bodies (`{"message": …}` only).

**401 is only ever an authentication answer** (same rule as `/api/v1`): admin-web may drop its state and go to
the login page on any 401, and must never see a 401 for anything else.

## 3. Formats

- Timestamps: ISO 8601 in Asia/Tehran, `"2026-09-23T13:00:00+03:30"`, or `null`.
- Calendar dates: `"2026-09-23"`, or `null`.
- Booleans are JSON booleans; ids are numbers.
- Translatable columns (T-M2-21): objects keyed by language code, `{"fa": "…", "en": "…"}`; the key set is the
  `languages` table, only the default language is required (`i18n.TranslatableRules`). Never hard-code `fa`/`en`.
- Request bodies: `application/json` (login requires it); uploads use `multipart/form-data` (§7).

## 4. Authentication, sessions, CSRF

1. `POST /auth/login` `{"email", "password", "remember": bool?}` →
   `data: {"admin": Admin, "csrf_token": "…"}` and two cookies:
   - `__Host-ritme_admin_session` — opaque 256-bit id, `HttpOnly; Secure; SameSite=Lax; Path=/`, no `Domain`
     (host-only). Browser-session cookie, or `Max-Age` 30 days with `remember: true`.
   - `__Host-ritme_admin_csrf` — the CSRF token, readable by JS (same attributes minus `HttpOnly`).
2. Every **mutating** request (`POST`/`PUT`/`PATCH`/`DELETE`) sends header **`X-CSRF-Token: <csrf_token>`**
   (from the login response, `GET /auth/me`, or the CSRF cookie). The server compares it with the token stored in
   the session (constant time); missing or wrong → 419. `GET` needs no token.
3. Browsers send the cookie automatically (`fetch(..., {credentials: "include"})` for cross-origin dev).
4. Sessions live in Redis (`ritme-go:admin-session:<sha256(id)>`), sliding: 120 min idle, 30 days with
   remember-me. Every authenticated request extends them. Each request re-reads the admin row, so deactivating or
   deleting an admin locks them out on their next request.
5. `POST /auth/logout` destroys the session and expires both cookies.
6. Changing a password ends the admin's other sessions; deactivating/deleting an admin ends all of theirs.
7. Login throttle: 5 attempts per minute per client IP + email, and 20 per minute per email from any IP (caps
   guessing even with a spoofed `X-Forwarded-For`); every attempt counts. Unknown emails cost the same bcrypt time
   as wrong passwords.
8. Passwords: Laravel's `$2y$` bcrypt hashes verify as-is; new hashes are bcrypt cost 12 (`$2a$`, which Laravel
   also accepts). Admins log in once again after the switch (D-06): PHP sessions are not shared.

Defence in depth: mutating requests with an `Origin` that is neither the admin host nor a configured
`ADMIN_WEB_ORIGINS` entry get 403 `origin_forbidden`; the login endpoint requires a JSON body (no cross-site HTML
form posts).

`Admin` object:

```json
{"id": 1, "name": "Root", "email": "root@ritme.app", "role": "super", "is_super": true, "is_active": true,
 "last_login_at": "…", "created_at": "…", "updated_at": "…"}
```

Roles: `super` (everything) and `editor` (everything except admin accounts and languages).
B-N9-02 adds nine capability-based roles (`product_manager`, `medical_advisor`, `support`, `clinical_support`,
`analyst`, `marketing`, `engineering`, `doctor`, `instructor_moderator`) — see §21. Login and `/auth/me` also return
`permissions` (`{capability: "view"|"manage"}`).

## 5. Lists, filters, pagination

`GET` list endpoints take `?page=` (≥ 1, default 1) and `?per_page=` (1–100, default 20) and answer:

```json
{"success": true, "data": {
  "items": [ … ],
  "meta": {"current_page": 1, "last_page": 3, "per_page": 20, "total": 45},
  "filters": {"q": "sara", "status": "all"}
}}
```

- `items` is always an array (possibly empty); a page past the end returns `items: []` with the real `meta`.
- Filters are plain query parameters; the list echoes the values it applied in `filters` (unknown values fall
  back to the default). Search is `?q=` (trimmed, substring, `%`/`_` are literal). Status-like filters use
  `?status=`; `all` is the default.
- Ordering is fixed per endpoint (newest first unless the resource has a `sort_order`, T-M2-21).

## 6. Endpoints (T-M2-20)

`A` = any active admin, `S` = super admin only. All paths are under `/api/admin/v1`.

| Method | Path | Who | Body / query | `data` |
|---|---|---|---|---|
| POST | `/auth/login` | public | `email`, `password`, `remember?` | `{admin, csrf_token}` |
| POST | `/auth/logout` | A | — | `null` |
| GET | `/auth/me` | A | — | `{admin, csrf_token}` |
| PUT | `/auth/password` | A | `current_password`, `password` (8–72), `password_confirmation` | `null` |
| GET | `/dashboard` | A | — | `{stats:{users, users_blocked, users_new_week, users_new_today, articles, affirmations, challenges, task_templates, messages, messages_pending}, recent_users:[8]}` |
| GET | `/users` | A | `q` (name/mobile/email), `status=all\|active\|blocked`, `page`, `per_page` | list of `{id, name, mobile, email, is_blocked, blocked_at, subscription_type, user_goal, created_at}` |
| GET | `/users/:id` | A | — | `{user, profile\|null, stats:{health_logs, reminders, notifications}, options:{subscription_types, user_goals}}` |
| PUT | `/users/:id` | A | `name?` (nullable; absent = unchanged), `subscription_type`, `user_goal` | same as show |
| POST | `/users/:id/block` | A | — | `{id, is_blocked, blocked_at, revoked_tokens}` — revokes every Passport token |
| POST | `/users/:id/unblock` | A | — | same shape (tokens stay revoked) |
| DELETE | `/users/:id` | A | — | `{id, revoked_tokens}` — revokes the tokens, then deletes (D-03) |
| GET | `/admins` | S | `page`, `per_page` | list of `Admin` (newest first) |
| POST | `/admins` | S | `name`, `email` (unique), `password` (8–72) + `password_confirmation`, `role`, `is_active?` (default true) | 201 `{admin}` |
| GET | `/admins/:id` | S | — | `{admin}` |
| PUT | `/admins/:id` | S | `name`, `email`, `password?` + confirmation (empty = unchanged), `role`, `is_active?` (absent = unchanged) | `{admin}` |
| DELETE | `/admins/:id` | S | — | `{id}` |

`options` entries are `{value, label}` with labels in the default language. Self-protection: a super admin cannot
change their own role, deactivate or delete themselves (422 `cannot_modify_self`; the form should disable those
controls for the current admin).

## 6c. Users list v2 and detail overview (B-N9-04, T-M10-32)

Capability `users` (view). `GET /users/directory?q=&status=&plan=free|plus|trial&activity=today|7d|30d|inactive&cohort=YYYY-MM&churn=new|low|medium|high&mode=&data_quality=&main_warning=&segment=<id>&page=&per_page=`
reads the nightly projection `admin_user_activity` (job `admin.user_activity`, 04:30 Tehran; 2 queries per page, 3 with
`segment`; explicit filters win over the saved segment's) -> rows `{id, ref, mobile_masked, plan, is_blocked, last_activity_at,
logged_days_30, churn_risk, mode, data_quality, main_warning, projected, created_at}` plus `options` (the filter vocabulary);
no name, email or health value (422 on a bad filter). `ref` is the stored code `RT-` + 6 base32 chars of
HMAC-SHA256(id, `ADMIN_REF_SECRET`) (7/8 on a collision; signup hook + `admin.ref_code_backfill`), or null when the
secret is unset; `q` matches a substring or an exact RT- code. Churn (A-08): `new` (account < 14 days) → `high` (recent
14 days ≤ 50 % of the prior 14, or 7+ days inactive) → `medium` (≤ 75 %) → `low`.
`GET /users/directory/export` (same filters, also needs `export`): anonymised CSV (ref, plan, signup_month, logged_days_30, churn_risk,
mode, data_quality, blocked), audited as `users.export`. `GET /users/:id/overview` -> `{ref, summary, engine, cycle_lengths, events,
messages, challenges, subscriptions, consents, support_reports (no text), health_access, health}`; `engine` is today's
snapshot as codes (phase, confidence, warnings, days_late), `cycle_lengths` lengths only (no dates), each message carries
`rule {id, key, kind, title, version}`. `health` (`{user_goal}`) and clinical/safety/content/engine rule titles are null
without a live health grant (D-84). The legacy `GET /users` / `GET /users/:id` show the full mobile only with
`users:manage`; view-only roles get the masked number.

Saved segments (capability `segments`, D-181): `GET /segments`, `GET /segments/:id` (`size` null + `size_suppressed: true`
below 10 users), `POST /segments`, `PUT /segments/:id`, `DELETE /segments/:id` `{name, filters}` — the directory filters
without free-text `q`; writes audited as `segment.create|update|delete`. The message fact `segment` is the ids a user
matches.

## 6c. KPI & cohorts (B-N9-08, D-89)

`GET /kpi?days=30|60|90&event=any|log_saved` (capability `dashboard`; 422 on other values): computed on demand from `analytics_events` and existing tables.
`{generated_at, events_since, funnel:{days, steps:[{key, pct, count, of, suppressed}]}, retention:{event, offsets:[0,1,7,14,30,60], cohorts:[{week_start, size, suppressed, cells[]}]}, engine, data_quality:null, ai:{...}, notes}`.
Groups under 10 users are `null`/`suppressed`; cells not yet reached are `null`. Aggregates only, never a user id.

## 6d. OKR (B-N9-09, D-95)

Capability `dashboard` (view reads, manage writes; 403 without it). `GET /okr?period=` -> `{objectives:[{id, title, owner, period, starts_on, ends_on, progress_pct, status, key_results:[{id, title, metric_key|null, source live|manual, unit, baseline, target, manual_value, current|null, progress_pct|null, status on_track|at_risk|done|no_data, owner}]}], periods, summary|null, metrics:[{key, unit, current|null}], units, notes}`.
`POST /okr/objectives` and `PUT /okr/objectives/:id` `{title, owner?, period, starts_on, ends_on (Y-m-d), sort_order?}`; `DELETE` cascades. `POST /okr/objectives/:id/key-results` and `PUT /okr/key-results/:id` `{title, metric_key? (catalogue), unit?, baseline, target (!= baseline), manual_value?, owner?}`; a bound key result takes the metric's unit and ignores `manual_value`. Progress is computed on every read from the newest snapshot (aggregates only); `/overview` `metrics.okr` is the summary of the objectives active today.

## 6b. Overview metrics (B-N9-03, D-85)

`GET /overview` (capability `dashboard`, any role with it): the newest hourly aggregate snapshot —
`{day, computed_at, metrics}`; `metrics` is `null` before the first run. `metrics`: `dau|mau|stickiness_pct|signups_today|plus_active`
`{value, prev}` (DAU = last full day, MAU = 30 days, prev = day before / 30 days before), `trial_to_paid {pct, count, of, suppressed}`,
`mrr_toman`, `modes [{mode, count, suppressed}]`, `engine {status healthy|watch|null, mean_error_days, within_2_days_pct, users, samples,
suppressed, confidence null}`, `alerts []`, `okr null`, `series` (90 days: `date, dau, loggers, signups`), `notes`. Aggregates only; groups
under 10 are suppressed (null). Nullable = not available, never a fake number. The existing `/dashboard` stays (recent users).

Automatic metric alerts (T-M10-31, D-180): after each snapshot the enabled `metric_alert_rules` are evaluated
(`ocr_correction_rate`, `message_ctr` drop vs the mean per rule version above `min_volume`, `rui_users` rising days,
`engine_mae`, `safety_selftest_failed`); one open alert per (rule, subject), resolved when the condition clears. The
overview payload carries `alerts` (open, newest first: `{key, subject, state, day, value, threshold, opened_at}`) and
`resolved_alerts` (last 10). `GET /dashboard/alert-rules` (dashboard view) lists the rules; `PUT
/dashboard/alert-rules/:key` `{threshold, enabled}` (dashboard manage) validates and audits
`dashboard.alert_rule.update`; the stored lists move at the next hourly snapshot.

## 7. Uploads (conventions for T-M2-21)

- `multipart/form-data`, one file per field, field name = the column concept (`image`, `cover`), other fields as
  form values (translatable fields as `title[fa]`, `title[en]`; `validation.Input` parses the PHP bracket syntax).
- Mutating upload requests carry `X-CSRF-Token` like any other (the admin chain does not require JSON).
- Limits as Laravel: banners jpeg/png/webp ≤ 4 MB and ≥ 800×400; article covers go through the image optimiser
  (fit 1080×1080, WebP q82, random 40-char name). nginx allows 25 MB.
- Files land on the `backend-storage` volume under `app/public/<dir>/`; the DB stores the relative `image_path`,
  responses return both `image_path` and the absolute `image_url` (`APP_URL/storage/<path>`). Articles already have an
  `image_url` column (an external URL the form edits), so article responses carry `image_url` (that column),
  `image_path` and `cover_url` (what the app shows: the upload wins over the external URL).
- The type is decided by the file's bytes (JPEG/PNG/WebP magic + a header decode), never by the file name or the
  part's Content-Type: GIF/BMP → the `mimes` message, SVG/HTML/anything else → the `image` message. A text value
  where a file belongs is an `image` error too. Too large → `max.file`; too small (banners) → `dimensions`.
- Uploads are `POST` (create) and `PUT` **or** `POST /<resource>/:id` (update — some clients cannot send multipart
  PUT). `remove_image=1` clears an article cover.
- Validation failures are the normal 422 with the file field in `errors`. Replacing a file deletes the old one
  after the row is saved.

## 8. Adding admin endpoints (T-M2-21 and later)

One routes file per area (`internal/http/routes_admin_<area>.go`), each building its own kit:

```go
Register("admin_content", func(r fiber.Router, d *Deps) {
    kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
    h := func(m, path string, chain httpadmin.Chain) { httpadmin.Handle(r, m, httpadmin.Prefix+path, chain) }
    h(fiber.MethodGet, "/articles", kit.Admin(articles.List))
    h(fiber.MethodPost, "/languages", kit.Super(languages.Store))
})
```

- Always register through `httpadmin.Handle` + `kit.Admin(h)` / `kit.Super(h)` / `kit.Public(h)`. Fiber runs a
  route's handlers **in argument order**; `r.Get(path, handler, middleware...)` would run the endpoint first.
- Do not register another `OPTIONS` catch-all; `routes_admin_core.go` answers preflights for the whole prefix.
- Helpers: `httpadmin.Validate` (Laravel rules, default locale, admin 422), `FieldError`, `NotFound`, `Fail`,
  `OK`/`Created`, `PageOf` + `Page`, `ID` (route ids → 404 when not numeric), `Now` (request clock), `DBTime`,
  `Time`, `CurrentAdmin`, `Audit`.
- Audit every mutation: `httpadmin.Audit(c, logger, "article.update", "article", id, extra...)` writes one slog
  line `{"msg":"admin audit","audit":"article.update","target_type":"article","target_id":7,"admin_id":1,"ip":…}`.
  Never log passwords, tokens or full phone numbers.

## 9. Deviations

- **D-03** — deleting a user revokes their Passport tokens (Laravel only deletes the row).
- **D-06** — admin sessions are new Go sessions; admins log in once more after the switch.
- Not a deviation (new API): JSON shapes above replace the Blade pages; Blade messages were Persian literals,
  the API returns English `message` + `error_code`.

## 10. Infra follow-ups (not in T-M2-20/21's files)

- `backend-go/cmd/api/cors.go`: the global Laravel-compatible CORS middleware handles every `/api/*` path,
  answers preflights before routing and overwrites `Access-Control-Allow-Origin` with the public origin. It must
  skip `/api/admin/` so the admin CORS (credentials, `ADMIN_WEB_ORIGINS`) applies. Only needed for cross-origin
  dev; same-origin production works without it.
- nginx: `deploy/vhost-admin.inc` currently 404s `/api/` on the admin host — it must proxy `/api/admin/` to the
  Go backend (and keep 404 for the rest of `/api/`); `vhost-api.inc` must 404 `/api/admin/`. Update the
  `deploy.sh` / `deploy-stage.sh` host assertions accordingly.
- Compose/env: set `ADMIN_HOSTS` (prod `adpanell.ritme.app`, stage its host) for the Go service.
- T-M2-21: the Go service writes uploads (`app/public/{articles,banners}`), UI bundles (`app/translations/<code>/`)
  and lang files (`app/lang/<code>/`) to the `backend-storage` volume, so it must be mounted **read-write** (staging
  mounts it read-only today), still as uid 33 so Laravel and Go can both read and write the files.
- T-M2-21: while Laravel still serves `/api/v1/languages`, the admin flushes Laravel's cache entry too
  (`LaravelCache` in `internal/admin/languages`): `DEL <REDIS_PREFIX><CACHE_PREFIX>languages.registry` on Laravel's cache
  DB. Defaults match `APP_NAME=Ritme` (`ritme-database-ritme-cache-languages.registry`, DB 1); override with
  `LARAVEL_CACHE_KEY_PREFIX` / `LARAVEL_CACHE_REDIS_DB`, disable with `LARAVEL_CACHE_FLUSH=false` after cutover.
- T-M2-21 (D-04): lang groups of an admin-added language are written as JSON to `app/lang/<code>/<group>.json`;
  `internal/i18n/lang` (embedded files only) does not read them yet, so validation messages of a new language fall back
  to English on both stacks until it does.

## 11. Endpoints (T-M2-21)

`A` = any active admin, `S` = super admin only. All under `/api/admin/v1`; bodies are JSON or multipart (uploads).
Translatable fields (`T`) are `{code: text}` objects (multipart: `title[fa]=…`); only the default language is required
where the field is required, and only active languages are stored. Booleans (`is_active`, `is_published`, …) follow
`$request->boolean()`: absent = `false`. On update, an optional field that is **absent** keeps its stored value; sent
as `null`/`""` clears it. Every create answers 201, everything else 200; `DELETE` answers `{id}`; `/toggle` answers
the updated record.

| Method | Path | Who | Body / query | `data` |
|---|---|---|---|---|
| GET | `/articles` | A | `page`, `per_page` | list of `Article` (sort_order, newest first) |
| GET | `/articles/options` | A | — | `{phases:[{value,label,legacy}], max_image_width, max_image_kb}` |
| POST | `/articles` | A | `slug` (unique), `title` T req, `excerpt` T, `body` T (HTML, sanitised on save), `cycle_phases[]` (sub-phases + legacy main phases), `category?`, `read_time_minutes?` (1–120), `image_url?` (≤1000), `sort_order?`, `is_published?`, `image` file (jpeg/png/webp ≤ 8 MB → WebP ≤1080², `articles/`) | `{article}` |
| GET | `/articles/:id` | A | — | `{article, options:{phases}}` (phases include the row's legacy values) |
| PUT/POST | `/articles/:id` | A | as POST, plus `remove_image?` | `{article}` (old cover deleted when replaced) |
| DELETE | `/articles/:id` | A | — | `{id}` (cover file deleted) |
| POST | `/articles/:id/toggle` | A | — | `{article}` (publishing sets `published_at` = now) |
| GET | `/affirmations` · `/affirmations/options` | A | — | list of `Affirmation` · `{phases}` |
| POST · PUT | `/affirmations` · `/affirmations/:id` | A | `text` T req, `cycle_phase?`, `sort_order?`, `is_active?` | `{affirmation}` |
| GET · DELETE · POST | `/affirmations/:id` · `/:id` · `/:id/toggle` | A | — | `{affirmation}` · `{id}` · `{affirmation}` |
| GET | `/challenges` | A | `q` (title/description/category), `status=all\|active\|inactive`, `cycle_day=1…35` (also matches untargeted) | list + `filters` |
| GET | `/challenges/options` | A | — | `{max_cycle_day: 35}` |
| POST · PUT | `/challenges` · `/challenges/:id` | A | `title` T req, `description` T, `cycle_day_from?`, `cycle_day_to?` (1–35, `gte:cycle_day_from` — fails without a `from`, as in Laravel), `category?`, `sort_order?`, `is_active?` | `{challenge}` |
| GET · DELETE · POST | `/challenges/:id` · `/:id` · `/:id/toggle` | A | — | `{challenge}` · `{id}` · `{challenge}` |
| GET | `/challenge-completions` | A | `challenge_id?` (exists), `q` (user name/mobile), `from?`, `to?` (dates, inclusive), `page`, `per_page` | list of `{id, completion_date, completed_at, user{id,name,mobile}, challenge{id,title}}` + `filters`, `stats{total,users,today}`, `per_challenge[{challenge_id,title,completions,users}]`, `challenges[{id,title}]` |
| GET | `/challenges/:id/stats` | A | — | `{challenge_id, stats:{participants, completed, active, completion_rate (%), rewards_granted, plus_days_granted, per_day:[{day, users}]}}` (B-N9-07, D-87) |
| POST · PUT | `/challenges` · `/challenges/:id` (v2, additive) | A | `duration_days?` (1–366; present = multi-day run, needs `tasks`), `start_rule?` (`anytime`\|`menstrual`\|`fixed_date`, the last needs `starts_on` Y-m-d), `reminder_time?` (H:i), `modes_include[]`, `modes_exclude[]` (life modes), `phases_include[]`, `phases_exclude[]` (main phases), `reward_badge_code?`, `reward_badge_title` T, `reward_plus_days?` (0–365), `completion_min_days?` (≤ duration), `tasks[]` = `{id?, day_from, day_to (≤ duration), title T, sort_order?}` — replaces the set: ids kept, others deleted/created | `{challenge}` now also `program` (null for a legacy challenge), `tasks`, `stats{participants, completed, completion_rate}`; list rows carry `program` + `stats` |
| GET | `/recommendations` | A | `phase=general\|<phase>`, `type` | list + `filters` (phase-less last, then phase, sort_order, id) |
| GET | `/recommendations/options` | A | — | `{phases, subphases, subphase_phases:{sub:phase}, types, triggers}` |
| POST · PUT | `/recommendations` · `/:id` | A | `type` req, `title` T (≤255, all empty → null), `text` T req (≤2000), `cycle_phase?`, `cycle_subphases[]` (only those the phase reaches), `symptom_trigger?`, `sort_order?`, `is_active?` | `{recommendation}` |
| GET · DELETE · POST | `/recommendations/:id` · `/:id` · `/:id/toggle` | A | — | `{recommendation}` · `{id}` · `{recommendation}` |
| GET | `/banners` · `/banners/options` | A | — | list of `Banner` (position, sort_order, newest) · `{positions, link_types, image:{max_kb,min_width,min_height,recommended_width,recommended_height,types}}` |
| GET | `/banners/:id/stats` | A | `?days=` (1–366, default 30) | B-N10-02 (D-93): `{banner_id, days, since, totals:{impressions,clicks,ctr}, items:[{day,impressions,clicks}]}` — per-Tehran-day aggregates, no user data |
| POST | `/banners` | A | multipart: `image` **req** (jpeg/png/webp ≤ 4 MB, ≥ 800×400, stored as uploaded in `banners/`), `title` T (≤255), `position` req, `link_type?`, `link_url?` (required with `link_type`; `url` when external, a single leading `/` path when internal — D-16; empty drops both), `starts_at?`, `ends_at?` (≥ starts_at), `sort_order?`, `is_active?`; B-N10-02 additive: `kind?` (`banner`\|`ad`; an ad needs an `ad_slider\|ad_square\|ad_strip` position, a banner a `home_*` one), `advertiser?` (≤120, admin-only), `cta_label` T (≤60), `modes?` (list of life modes; empty = all) | `{banner}` (`image_path`, `image_url`, + `kind`, `advertiser`, `cta_label`, `modes`); `/banners/options` also returns `kinds`, `ad_slots`, `modes` |
| PUT/POST | `/banners/:id` | A | as POST, `image` optional (replaces + deletes the old file) | `{banner}` |
| GET · DELETE · POST | `/banners/:id` · `/:id` · `/:id/toggle` | A | — | `{banner}` · `{id}` (file deleted) · `{banner}` |
| GET | `/task-templates` · `/task-templates/options` | A | — | list · `{phases, categories}` |
| POST · PUT | `/task-templates` · `/:id` | A | `key` req unique, `title` T req, `description` T, `category` req, `icon?`, `cycle_phase?`, `sort_order?`, `is_active?` | `{task_template}` |
| GET · DELETE · POST | `/task-templates/:id` · `/:id` · `/:id/toggle` | A | — | `{task_template}` · `{id}` · `{task_template}` |
| GET | `/info-sections` | A | `group=help\|privacy\|terms\|about\|support` (unknown → help) | list + `filters{group}` (sort_order, id) |
| GET | `/info-sections/options` | A | `group` | `{groups, group, next_sort_order}` |
| POST · PUT | `/info-sections` · `/:id` | A | `group` req, `key?` (≤64, slug `^[a-z0-9]+([-_][a-z0-9]+)*$`, unique within the group; PUT: absent = unchanged, null/"" = cleared — B-N1-12b), `heading` T req (≤200), `body` T req, `link_label` T (≤60, empty → null), `link_url?` (≤500, `https?://`, `mailto:` or `tel:`), `sort_order?`, `is_active?` | `{info_section}` |
| GET · DELETE · POST | `/info-sections/:id` · `/:id` · `/:id/toggle` | A | — | `{info_section}` · `{id}` · `{info_section}` |
| GET | `/pregnancy-weeks` | A | — | `{items:[{week, id\|null}] (1…40 + stored weeks above), fields:[…10]}` |
| POST · PUT | `/pregnancy-weeks` · `/:id` | A | `week_number` req (1–42, unique), each of the 10 fields T optional | `{pregnancy_week}` |
| GET · DELETE | `/pregnancy-weeks/:id` | A | — | `{pregnancy_week}` · `{id}` |
| GET | `/phase-contents` | A | — | `{items:[{value,label,legacy,id\|null}], fields:[…9], phases}` |
| POST · PUT | `/phase-contents` · `/:id` | A | `phase` req (content-backed sub-phase, unique), each of the 9 fields T optional | `{phase_content}` |
| GET · DELETE | `/phase-contents/:id` | A | — | `{phase_content}` · `{id}` |
| GET | `/messages` | A | `group`, `locale`, `status=approved\|pending`, `page`, `per_page` | list of `Message` (group, item_key, locale) + `filters`, `groups[]`, `locales[]` |
| GET | `/messages/:id` | A | — | `{message}` |
| PUT | `/messages/:id` | A | `payload{field: string \| [strings]}` (stored keys only; list fields also take a newline-separated string; an empty scalar keeps its text), `label?` (absent = unchanged) | `{message}` |
| POST | `/messages/:id/approve` · `/messages/:id/toggle` | A | — | `{message}` (toggles `is_approved` · `is_active`) |
| GET | `/languages` · `/languages/options` | S | — | `{items:[Language], default_code}` · `{directions, sources:[{code,name}], default_code, next_sort_order}` |
| POST | `/languages` | S | `code` (BCP-47 shape, normalised lower-case, unique), `name`, `english_name` (≤60), `direction`, `sort_order?`, `is_active?`, `is_default?`, `copy_from?` (active code, default: the default language) | 201 `{language, source, provisioned:{messages, lang_files, smart_messages}}` — bundles copied, lang files (D-04), smart messages cloned unapproved; both registry caches flushed |
| GET · PUT | `/languages/:id` | S | as POST (a code change does not rename files) | `{language}`; one default kept (a default is forced active; no default left → first active) |
| DELETE | `/languages/:id` | S | — | `{id}`; removes `app/translations/<code>`, `app/lang/<code>` and the language's smart messages (422 `default_language_protected` for the default) |
| POST | `/languages/:id/toggle` | S | — | `{language}` (422 `default_language_protected` for the active default) |
| POST | `/languages/:id/regenerate` | S | `copy_from?` | `{language, source, provisioned}` — bundles rewritten, missing lang files / message rows filled |
| GET | `/languages/:id/translations` | S | `namespace` (unknown → first) | `{language, namespaces, namespace, rows:[{key,value,reference}], default_code, default_name, is_default_locale}` |
| PUT | `/languages/:id/translations` | S | `namespace`, `rows:[{key (≤255), value?}]` | `{language, namespace, saved}` — writes `app/translations/<code>/<ns>.json` (empty values dropped → fall back to the default) |

Record shapes are the table columns (translatable ones as objects, timestamps ISO 8601) — `Article` adds `cover_url`,
`Banner` adds `image_url`. Options are `{value, label}` in the default language.

Security notes: uploads are sniffed and header-decoded (SVG/HTML never stored), stored under random 40-character
names with an extension chosen from the detected type, written atomically inside `app/public/<dir>` only; article
covers are re-encoded (EXIF and any appended payload dropped), banners are stored as uploaded (like Laravel) but can
only be served as `image/*`. Article HTML is sanitised on save (T-M2-10 sanitizer) and again on read. Language codes and
namespaces are checked against strict patterns before any path is built.

## 12. Checkup types (T-M4-03)

Code: `backend-go/internal/admin/checkups`, routes in `internal/http/routes_admin_checkups.go`, queries in
`db/queries/checkups/admin.sql` (generated into `internal/checkups/store`). Product contract:
[`docs/checkups/README.md`](../checkups/README.md). Only the **shared catalog** (`user_id IS NULL`) is reachable: a
user's custom checkup id answers 404 everywhere and is never listed, counted or reordered. The user API
(`/api/v1/checkups…`) reads the same rows without a cache, so every change is live on its next request.

`A` = any active admin (`editor` and `super`). All under `/api/admin/v1`; JSON bodies; mutating requests need
`X-CSRF-Token` like every admin call. Create answers 201, everything else 200.

| Method | Path | Who | Body / query | `data` |
|---|---|---|---|---|
| GET | `/checkup-types` | A | `q` (key / title, substring), `status=all\|active\|inactive`, `page`, `per_page` | list of `CheckupType` (sort_order, id) + `filters{q,status}` |
| GET | `/checkup-types/options` | A | — | `{categories[], performed_by[], icons[], tones[], audiences[], default_tone, default_remind_lead_days, max_steps, max_cycle_day, max_interval_months, next_sort_order}` (plain value lists; admin-web labels them) |
| GET | `/checkup-types/stats` | A | — | `{items:[{id, key, title, category, is_active, records_total, users_with_records, records_last_30_days, overdue_users}] (sort order), window_days: 30, since, today}` |
| POST | `/checkup-types/reorder` | A | `ids[]` — every catalog id exactly once, in the new order | `{items:[{id, sort_order}]}`; `sort_order` becomes 1…n (one transaction). Partial lists, duplicates, custom or unknown ids → 422 on `ids` / `ids.N` |
| POST | `/checkup-types` | A | see *Fields* | `{checkup_type}` |
| GET | `/checkup-types/:id` | A | — | `{checkup_type}` |
| PUT | `/checkup-types/:id` | A | as POST without `key` (the key is fixed at creation — the frontend routes on it, e.g. `breast_self_exam`) | `{checkup_type}` |
| DELETE | `/checkup-types/:id` | A | — | `{id}`; **422 `in_use`** (`records_count` in the body) while any user has a record of the type — deleting would cascade to users' health records, so admin-web offers *deactivate* (`is_active: false`) instead. User settings rows of a deleted type go with it |

Fields (`T` = `{code: text}` over the active languages; unlike §11, **every active language is required** for a
translatable text that is sent — the catalog is shown in each of them; other codes are dropped):

| Field | Rules |
|---|---|
| `key` | create only; required, `^[a-z][a-z0-9_]*$`, ≤ 64, unique (seeded keys included) |
| `title` | T, required |
| `subtitle` (≤ 255), `why` (≤ 2000) | T, optional; `null` clears |
| `category` | required: `monthly`, `six_monthly`, `annual`, `multi_year`, `age_based` (`custom` is users' own) |
| `performed_by` | required: `self`, `doctor`, `lab`, `dentist` |
| `icon` | optional, one of `options.icons` (the names `frontend/src/entities/checkup/model/icon.ts` resolves) |
| `tone` | optional: `rose`, `violet`, `amber`, `teal`, `green`, `neutral`; missing/null → `neutral` |
| `interval_months` | required integer 1–120 |
| `interval_months_max` | optional 1–120, ≥ `interval_months` (a range: due from the minimum, overdue after the maximum) |
| `age_min`, `age_max` | optional 0–120, `age_max ≥ age_min` |
| `cycle_day_from`, `cycle_day_to` | optional 1–45, both or neither (`required_with`), `to ≥ from` |
| `remind_lead_days` | optional 0–365; missing/null → 7 |
| `prep_steps` | optional list ≤ 10 of T (each item: every active language, ≤ 500) |
| `guide_steps` | optional list ≤ 10 of `{title: T (≤ 120), body: T (≤ 1000)}` |
| `finding_options` | optional list ≤ 10 of `{key (^[a-z][a-z0-9_]*$, ≤ 40, distinct), exclusive?: bool, label: T (≤ 120)}`; `exclusive` is stored only when true |
| `hide_in_pregnancy`, `is_active` | `$request->boolean()`: absent = `false` (send both on every save) |
| `audiences` | optional list ≤ 6 of life modes (`options.audiences`: `cycle`, `ttc`, `pregnancy`, `postpartum`, `menopause`, `teen`; distinct, order kept). `null` / `[]` = every mode (stored as SQL `NULL`, answered as `null`). The user API shows a shared row only to users whose resolved life mode is listed (CB-MENO-01b) |
| `sort_order` | optional integer; create default = last + 1 |
| `source_note` | optional string ≤ 1000 (medical-review note) |

On update an optional field that is **absent keeps its stored value**; `null` (or `[]` for a list) clears it. The
range checks compare against the stored value of an absent field. An empty list is stored as SQL `NULL` and always
answered as `[]`.

`CheckupType` = the `checkup_types` columns (`id, key, category, title, subtitle, why, performed_by, icon, tone,
interval_months, interval_months_max, age_min, age_max, cycle_day_from, cycle_day_to, remind_lead_days, prep_steps,
guide_steps, finding_options, hide_in_pregnancy, audiences, is_active, sort_order, source_note`) + `records_count` (records of
all users) + `created_at`, `updated_at` (ISO 8601).

Stats: `records_last_30_days` counts records with `done_on ≥ today − 30`. `overdue_users` counts users whose
**latest** record of the type is past its due-by date — `next_due_on` (user override) ?? `done_on +
(interval_months_max ?? interval_months)` months — before today, leaving out users who switched the type off. It is
the calendar approximation of the engine's `overdue` (cycle windows, age and pregnancy are not evaluated per user);
users who never recorded the type are not counted.

Audit lines: `checkup_type.create|update|delete|reorder` (target `checkup_type`).

## 13. Pregnancy v2 content and registered messages (T-M7-06)

Code: `backend-go/internal/admin/pregnancy` (routes `internal/http/routes_admin_pregnancy.go`),
`backend-go/internal/admin/messages` + `messages/registry` (routes with §11's messages). Queries: the T-M7-01 files
`db/queries/pregnancy/v2_*.sql` plus `db/queries/pregnancy/v2_admin.sql` (care-item delete guard, message_contents
writes), generated into `internal/pregnancy/store`. Product contract: [`docs/pregnancy-v2/README.md`](../pregnancy-v2/README.md)
(*What the admin defines*, *Stored shapes*). Every endpoint is `A` (editor + super); mutations need `X-CSRF-Token` and are
audited. The user API reads the rows without a cache, so an edit is live on its next request.

`T` below = `{code: text}` over the active languages; as in §12, **every active language is required** once a
translatable field is sent (other codes are dropped). On update an absent optional field keeps its stored value;
`null` (or `[]` for a list) clears it; an empty list is stored as SQL `NULL` and answered as `[]`.

### Week details (`pregnancy_week_details`)

`:n` is the **week number** (1–42; anything else, `08` included, is 404) — not the row id of the v1 text editor
(`/pregnancy-weeks/:id`, §11), which stays as it is next to it.

| Method | Path | `data` |
|---|---|---|
| GET | `/pregnancy-week-details` | `{items:[{week_number, exists, illustration_key, size_label, headline, reviewed_at, updated_at}]}` — weeks 1…42 |
| GET | `/pregnancy-week-details/options` | `{min_week: 1, max_week: 42, illustration_keys[36], highlight_icons[8], highlight_tones[3], log_symptom_keys[9], max_items: 10}` |
| GET | `/pregnancy-weeks/:n/details` | `{week_details}` (a week without a row answers `exists: false`, every field null / `[]`) |
| PUT | `/pregnancy-weeks/:n/details` | upsert → `{week_details}` |

`week_details` = `{week_number, exists, size_label, illustration_key, length_cm, weight_g, heart_rate, headline,
highlights, body_symptoms, body_text, tasks, warning, reviewer_name, reviewed_at, sources, created_at, updated_at}`.

| Field | Rules |
|---|---|
| `size_label` (≤ 60), `headline` (≤ 255), `body_text`, `warning` (≤ 2000), `reviewer_name` (≤ 255) | T, optional |
| `illustration_key` | optional, one of `options.illustration_keys` = `FETUS_ILLUSTRATION_KEYS` (`frontend/src/shared/ui/illustrations/fetus-keys.ts`) |
| `length_cm`, `weight_g`, `heart_rate` | optional ASCII measurement ≤ 20: `^<?N(.NN)?(-N(.NN)?)?$` (`"1.6"`, `"<1"`, `"150-170"`); the app adds units / `~` |
| `highlights` | list ≤ 10 of `{icon?: options.highlight_icons, tone?: options.highlight_tones, title: T ≤ 120, body: T ≤ 1000}` (`tone` stored only when set) |
| `body_symptoms` | list ≤ 10 of `{key (^[a-z][a-z0-9_]*$, ≤ 40, distinct), label: T ≤ 60}`; keys in `log_symptom_keys` deep-link to the Log screen |
| `tasks` | list ≤ 10 of `{key (same pattern, distinct — stored in users' `done_task_keys`), text: T ≤ 255}` |
| `sources` | list ≤ 10 of `{title: T ≤ 255, url?: http(s) URL or `/path`, ≤ 1000}` |
| `reviewed_at` | optional `Y-m-d`, not in the future; `null` = not clinically reviewed |

### Care plan (`pregnancy_care_items`)

| Method | Path | Body | `data` |
|---|---|---|---|
| GET | `/pregnancy-care-items` | — | `{items:[CareItem]}` (sort_order, id; active and inactive) |
| GET | `/pregnancy-care-items/options` | — | `{kinds, min_week, max_week, max_remind_before: 30, default_remind_before: 1, next_sort_order}` |
| POST | `/pregnancy-care-items/reorder` | `ids[]` — every item id exactly once | `{items:[{id, sort_order}]}` (1…n, one transaction); partial / duplicate / unknown → 422 `ids` / `ids.N` |
| POST | `/pregnancy-care-items` | see below | 201 `{care_item}` |
| GET | `/pregnancy-care-items/:id` | — | `{care_item}` |
| PUT | `/pregnancy-care-items/:id` | as POST without `key` (fixed: appointments reference it in `meta.care_item_key`) | `{care_item}` |
| POST | `/pregnancy-care-items/:id/toggle` | — | `{care_item}` (`is_active` flips) |
| DELETE | `/pregnancy-care-items/:id` | — | `{id}`; **422 `in_use`** (`appointments_count`) while any non-cancelled appointment (`reminders.type = 'appointment'`, `meta.status` ≠ `cancelled`; T-M2-34) has `meta.care_item_key` = the key — admin-web offers deactivate instead. The check and the delete are one statement |

Fields: `key` (create only, `^[a-z][a-z0-9_]*$`, ≤ 64, unique), `title` T req (≤ 255), `prep` T (≤ 2000),
`kind` req `visit|test|scan|vaccine`, `week_from` / `week_to` req 1–42 with `week_to ≥ week_from`,
`remind_before` 0–30 days (create default 1), `sort_order` (create default last + 1), `is_active` (`$request->boolean()`,
absent = false). `CareItem` = the columns + `appointments_count` + `created_at`, `updated_at`.

### Alert rules (`message_contents` group `pregnancy_alert`)

One row per rule and locale; payload `{enabled, level, window_days, params{…}, title, what_we_saw, how_sure, advice,
actions:[{key,label}], contact}`. The **behaviour** (`enabled`, `level`, `window_days`, `params`) is written to every
locale's row of the rule (they never drift; the engine reads it from the default-language row); the **texts** are per
locale. `legend` is not a rule: it is edited through `/messages` (its row ids are in the list's `legend`).

| Method | Path | `data` |
|---|---|---|
| GET | `/pregnancy-alert-rules` | `{items:[{key, configured, enabled, level, window_days, params, title, locales[], missing_locales[], updated_at}], legend:{group, item_key, rows:[{id, locale}]}}` — the 8 rules in registry order |
| GET | `/pregnancy-alert-rules/options` | `{levels, actions, min_window_days: 1, max_window_days: 30, text_fields: Schema, rules:[{key, params_schema: Schema, placeholders[]}]}` |
| GET | `/pregnancy-alert-rules/:key` | `{rule: {key, configured, enabled, level, window_days, params, texts:{code: {title, what_we_saw, how_sure, advice, actions, contact}}, rows:[{id, locale, is_active, is_approved, updated_at}], missing_locales, params_schema, placeholders, updated_at}}` |
| PUT | `/pregnancy-alert-rules/:key` | `{rule}` (same shape) |

PUT body: `enabled` (boolean, req), `level` req `info|suggestion|follow_up|urgent`, `window_days` req 1–30, `params`
(object, typed per rule, unknown keys dropped), `texts?: {code: {title ≤ 255, what_we_saw ≤ 500, how_sure ≤ 500,
advice ≤ 2000 (all req), actions: list ≤ 4 of {key: ack|add_to_visit_note|log_weight|open_week|call (distinct), label ≤ 60},
contact? ≤ 500}}`. A locale's texts are optional (absent keeps them) except the default language's while its row does
not exist; sending texts for an active language without a row creates it (active, approved). One transaction.

| Rule | `params` | Placeholders |
|---|---|---|
| `vomiting_streak` | `min_streak_days` 2–14, `severe_min_count` 0–14 | `{days}`, `{severe_count}` |
| `severe_symptom_count` | `min_count` 1–50, `symptoms[]` ⊂ Log symptoms (≥ 1, distinct) | `{count}` |
| `critical_symptom` | `symptoms[]` ⊂ `spotting, bleeding, fluid_leakage, severe_sudden_pain`, `spotting_until_week` 0–42 | `{symptom}` |
| `weight_missing_week` | `from_week` 1–42, `from_weekday` 0–6 or null (0-based day of the week it fires from; null / missing = engine default 5, the 6th day; T-M2-34, editable since T-M7-20) | `{week}` |
| `week_entered` | — (`{}`) | `{week}`, `{basis}` |
| `bp_high` | `systolic_min` 90–200, `diastolic_min` 50–130 | `{systolic}`, `{diastolic}` |
| `sugar_high` | `fasting_max` 60–200, `post_meal_max` 80–300 | `{fasting}`, `{post_meal}` |
| `fetal_movement` | `from_week` 12–42, `statuses[]` ⊂ `FetalMovementStatus` | `{status}` |
| `contractions_511` | `interval_max_minutes` 2–15, `duration_min_seconds` 20–120, `run_minutes` 20–180 (contraction timer, bloom B-N5-03, D-60; fires once per session), `contact_phone` text or null (B-N5-09: the number the «تماس» action dials — ASCII digits, optional leading `+`, inner spaces / dashes, 3–20 chars, else 422 `regex`; `""` clears it; when set, the user alert's `contact` is `{text, phone}` instead of the plain text) | `{count}`, `{minutes}`, `{interval}`, `{duration}` (m:ss) |

`Schema` = `[{key, kind, nullable, …}]` with `kind` ∈ `text|text_list|integer|boolean|enum|enum_list|url|object|object_list`
plus `max_length` (text/url/text_list), `min`/`max` (integer), `values` (enums), `min_items`/`max_items` (lists) and
nested `fields` (objects) — enough for admin-web to render a form.

### Messages: create in a registered group (additions to §11)

The registry (`internal/admin/messages/registry`) lists the groups the engine reads: `pregnancy_week_tip` (`1`…`42`),
`pregnancy_alert` (the 8 rules + `legend`), `pregnancy_setup` (`welcome, dating, source_lmp, source_ultrasound,
source_manual, history, result, due_disclaimer, calendar_note`) with explicit schemas (**typed**), then every group of the code
fallback (`messages/content/defaults.json`) with its keys and a schema derived from the default copy (strings / string
lists).

| Method | Path | Body / query | `data` |
|---|---|---|---|
| GET | `/messages` | as §11 | as §11 plus `registered_groups[]` and `missing:[{group, item_key, locale}]` — every registered item × active language without a row, narrowed by `group` / `locale` |
| GET | `/messages/registry` | — | `{groups:[{group, typed, keys[]}]}` |
| GET | `/messages/registry/:group/:key` | `locale?` | `{group, item_key, typed, fields: Schema, placeholders[], existing:[{id, locale}], template}` — `template` prefills a new row: the default-language row, else any row, else the code fallback copy in `locale` (default: the default language), else an empty shape. Unregistered → 404 |
| POST | `/messages` | `group` (registered), `item_key` (registered for the group), `locale` (active), `payload` (validated against the item's schema, errors on `payload.<path>`; stored in schema order, unknown keys dropped), `label?` (default `"<group> / <item_key>"`), `is_active?`, `is_approved?` (absent = **true**), `sort_order?` (0–65535) | 201 `{message}`; an existing (group, item_key, locale) → 422 `unique` on `item_key` |

`PUT /messages/:id` on a **typed** item no longer uses the shape-keeping string merge (which would flatten `params`,
`actions` and `levels` into string lists): the sent top-level keys replace the stored ones and the whole payload is
validated against the schema (422 on `payload.<path>`). Untyped groups keep the §11 behaviour.

Audit lines: `pregnancy_week_details.update` (target `pregnancy_week`, id = week), `pregnancy_care_item.create|update|
toggle|delete|reorder`, `pregnancy_alert_rule.update` (attr `rule`, `rows_created`), `message.create`.

## 14. Support reports inbox (B-N1-12b)

«گزارش مشکل» reports sent from the app (`POST /api/v1/support/reports`, `support_reports`). Any active admin (**A**, the
same role as `/users`: rows carry the user's mobile and free text). No health data beyond the report text itself.

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/support-reports` | A | `status=open\|resolved\|all` (default / unknown → `open`), `page`, `per_page` | list, newest first (`created_at DESC, id DESC`); items `{id, user{id, name, mobile}, preview (first 160 chars), has_screenshot, app_version, status, created_at, updated_at}` + `filters{status}` + `counts{open, resolved}` |
| GET | `/support-reports/:id` | A | — | `{support_report: {id, user{…}, message, has_screenshot, app_version, user_agent, status, created_at, updated_at}}` |
| GET | `/support-reports/:id/screenshot` | A | — | the WebP bytes (`image/webp`, `Cache-Control: private, no-store`, `nosniff`); 404 when none / file gone / path outside `app/private/support-reports` |
| POST | `/support-reports/:id/resolve` · `/:id/reopen` | A | — | `{support_report}` (idempotent), message `Report resolved.` / `Report reopened.` |

The screenshot lives on the private part of the storage volume and has no public URL; the storage path is never
sent. admin-web fetches it with the session cookie (`credentials: 'include'`) and shows it from an object URL.
Audit lines: `support_report.resolve|reopen|screenshot`.

## 15. Subscriptions & payments — «اشتراک‌ها و پرداخت» (B-N2-09)

Ritme Plus admin over `plus_plans`, `plus_discount_codes`, `plus_settings`, `plus_subscriptions`, `plus_invoices`,
`plus_receipts` (B-N2-04/05/06). Package `internal/admin/billing`, routes `routes_admin_billing.go`. Money is integer
**rials** (`*_rials`); admin-web shows and takes toman (÷/× 10). VAT is in basis points (1000 = 10 %).

**Roles.** Every read: any active admin (**A**) — list rows carry the subscriber's name and a **masked** mobile
(`0912•••4567`, the Admin_Users artboard rule; the full number stays on `/users/:id`). Every write: super admin
(**S**) — plan prices, discount codes, settings, refunds and extensions all change what users pay or get. Writes pass
the CSRF check of the admin chain (419 without the token) and each one writes a **`plus_admin_actions`** row (migration
00019: admin id, action, target, user id, amount, days, gateway + gateway refund id, note, details JSON) plus the slog
`admin audit` line `plus.<action>`.

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/plus/plans` | A | — | `{items: [Plan], currency: "IRR", limits{min_price_rials, max_price_rials, max_duration_months}, next_sort_order}` (every plan, `sort_order, id`; not paginated) |
| POST | `/plus/plans` | S | `code` (`^[a-z][a-z0-9_]*$`, unique, fixed afterwards), `title` (translatable, default language required, ≤ 64), `badge?` (translatable, ≤ 32), `duration_months` (1–36), `price_rials` (10,000 – 10¹¹), `monthly_display_rials?`, `is_highlighted?` (false), `is_active?` (true), `sort_order?` (default max + 1) | 201 `{plan}` |
| GET | `/plus/plans/:id` | A | — | `{plan}` |
| PUT | `/plus/plans/:id` | S | as POST without `code`; absent optional fields keep their value | `{plan}` — existing invoices keep the price they were charged |
| DELETE | `/plus/plans/:id` | S | — | `{id, deleted, deactivated}` — a plan referenced by invoices or subscriptions is **deactivated** (and un-highlighted) instead |
| GET | `/plus/discount-codes` | A | `q` (code substring), `status=all\|active\|inactive`, `page`, `per_page` | list of `DiscountCode` (newest first) + `filters` |
| POST | `/plus/discount-codes` | S | `code` (3–64, `[A-Za-z0-9_-]`, stored upper-case, unique case-insensitively, fixed afterwards), `kind=percent\|amount`, `value` (percent 1–100 / rials), `max_redemptions?` (null = unlimited), `per_user_limit?` (absent = 1, null = unlimited), `plan_ids?` (existing plan ids; empty / null = every plan), `starts_at?`, `expires_at?` (≥ starts_at), `is_active?` | 201 `{discount_code}` |
| GET | `/plus/discount-codes/:id` | A | — | `{discount_code}` |
| PUT | `/plus/discount-codes/:id` | S | as POST without `code`; absent optional fields keep their value | `{discount_code}` |
| DELETE | `/plus/discount-codes/:id` | S | — | `{id, deleted, deactivated}` — a code used on any invoice is deactivated instead |
| GET | `/plus/settings` | A | — | `{settings: {trial_offer_percent, vat_rate_bps (effective), vat_override_bps\|null, vat_env_rate_bps, vat_source: env\|admin, trial_days, ai_cost_usd_irr_rate\|null}}` |
| PUT | `/plus/settings` | S | `trial_offer_percent` (0–100, 0 = no offer), `vat_rate_bps?` (0–10000; **null** removes the override → `PLUS_VAT_RATE_BPS`; absent = unchanged), `ai_cost_usd_irr_rate?` (IRR per USD, number 1–100 000 000; **null** removes it → KPI `ai_cost.share` null; absent = unchanged; stored as `plus_settings.ai_cost_usd_irr_rate`, read by the KPI snapshot job) | same as GET; a ledger row only when something changed |
| GET | `/plus/subscriptions` | A | `status=all\|active\|canceled\|expired\|refunded` (expired = active/canceled with `ends_at` ≤ now), `plan_id`, `from`/`to` (start date, Tehran, inclusive), `q` (part of the mobile; Persian digits accepted), `page`, `per_page` | list of `{id, user{id, name, mobile (masked)}, plan{id, code, title}\|null, invoice_reference, status, effective_status, source, starts_at, ends_at, days_left, auto_renew, canceled_at, created_at}` + `filters` + `counts{active, canceled, expired, refunded}` |
| POST | `/plus/subscriptions/:id/extend` | S | `days` (1–365), `note` (3–500, required) | `{id, ends_at, days, queued_shifted}` — only a running period (active / canceled, not ended; else 422 `subscription_not_active`); the user's queued periods starting at/after the old end move back by the same days |
| GET | `/plus/payments` | A | `status=all\|pending\|paid\|failed\|expired\|refunded`, `gateway`, `from`/`to` (created date), `q` (invoice reference, bank reference or part of the mobile), `page`, `per_page` | list of `{id, reference, user{…masked}, plan, duration_months, status, effective_status, subtotal_rials, discount_rials, vat_rate_bps, vat_rials, total_rials, discount_code, gateway, receipt{ref_id, amount_rials, paid_at}\|null, paid_at, created_at}` + `filters` + `summary{paid_count, paid_rials, refunded_rials}` (of the filtered rows) + `gateways` + `statuses` |
| GET | `/plus/payments/:id` | A | — | `{payment: {…, authority, receipt{gateway, ref_id, amount_rials, paid_at}, subscriptions[{id, status, source, starts_at, ends_at}], refund{refundable, gateway_available, amount_rials}, actions[{id, action, admin{id, name}, amount_rials, days, gateway, gateway_ref, note, details, created_at}]}}` |
| POST | `/plus/payments/:id/refund` | S | `mode=gateway\|manual`, `note` (manual: required 3–500; gateway: optional) | `{payment}` (refreshed) |

`Plan` = `{id, code, title, badge, duration_months, price_rials, monthly_display_rials, monthly_price_rials, is_highlighted,
is_active, sort_order, invoices_count, subscriptions_count, active_subscriptions, created_at, updated_at}`.
`DiscountCode` = `{id, code, kind, value, max_redemptions, per_user_limit, plan_ids|null, starts_at, expires_at, is_active,
uses{paid, pending}, state: active|scheduled|expired|exhausted|inactive, created_at, updated_at}`.

**Card data** never leaves the API: `plus_receipts.card_pan` is not selected by any admin query.

**Refunds** are full refunds of a `paid` invoice, in one READ COMMITTED transaction holding the invoice row lock (a
second refund sees `refunded` → 422 `invoice_not_refundable`). Effects: invoice `refunded`, the subscription period(s)
it bought `refunded` (auto-renew off; no longer entitles), one ledger row.
- `mode=gateway` calls `payments.Gateway.Refund` (the `PAYMENT_PROVIDER` adapter) with the invoice authority, the
  receipt's bank reference and the **receipt amount** (never a client value). The fake provider refunds; **Zarinpal
  answers not-supported** (QUESTIONS #71) → 422 `refund_not_supported` and nothing changes; refund in the Zarinpal
  panel, then `mode=manual`. Also `refund_not_supported` when no provider is configured or the receipt came from a
  different provider. Provider refusal → 422 `refund_rejected`; transport failure → 502 `refund_failed`. A free
  (100 %-discount) invoice has nothing to refund through a gateway (422 `invoice_not_refundable`) — use manual to revoke.
- `mode=manual` records a refund made outside Ritme; the note is required. Ledger action `invoice.refund_manual`.

**VAT override.** `plus_settings.vat_rate_bps` (no row = env `PLUS_VAT_RATE_BPS`). Checkout (`plus.Service` quote) and
the public `GET /api/v1/plus/plans` read it at request time; every invoice keeps its own `vat_rate_bps` snapshot.

Ledger actions: `plan.create|update|delete|deactivate`, `discount.create|update|delete|deactivate`, `settings.update`,
`invoice.refund`, `invoice.refund_manual`, `subscription.extend`. Error codes: `invoice_not_refundable`,
`refund_not_supported`, `refund_rejected`, `refund_failed`, `subscription_not_active` (+ the generic ones of §2).

## 16. Companion «همدم» — tips and link overview (B-N4-07)

Package `internal/admin/companions`, routes `routes_admin_companions.go`. Every route: any active admin (**A**);
mutations pass the CSRF check (419 without the token) and write the slog `admin audit` line. No migration.

**Tips** are the companion panel's «امروز چه کار کنی؟» copy (`GET /api/v1/companion/home`, B-N4-03): per partner phase
(`menstrual, follicular, fertile, luteal, pregnancy, general`) one note under the cycle card and up to three tips, per
language. Stored as the fixed `message_contents` slots of group `companion_tip` (registry `CompanionTipGroup`, also
editable row by row in `/messages`): `<phase>_note` `{body}` and `<phase>_tip_1..3` `{title, body}`, placeholder
`{name}` (the partner's name). The panel resolves each slot as: live (active + approved) row of the request language →
live row of the default language → built-in copy (`internal/companion/home/lang`). An empty title hides a tip, an
empty body hides the note.

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/companions/tips` | A | — | `{items: [PhaseTips] (display order), phases[], tips_per_phase: 3, placeholders: ["name"], limits{note: 500, title: 255, body: 1000}, default_locale}` |
| GET | `/companions/tips/:phase` | A | — | `{tips: PhaseTips}`; unknown phase → 404 |
| PUT | `/companions/tips/:phase` | A | `texts` (req, object): `{code: {note?: string\|null ≤ 500, tips?: [{title (req, ≤ 255), body? ≤ 1000}] ≤ 3}}` over active languages (an unknown code → 422 on `texts.<code>`; at least one non-null) | `{tips: PhaseTips}`, message `Companion tips saved.` — for each sent language all four slots are written in one transaction (created when missing, set live): the note (`""` when absent) and the tips in the sent order, unused slots emptied |
| DELETE | `/companions/tips/:phase` | A | `locale` (query, req, active code) | `{tips: PhaseTips}`, message `Companion tips reset.` — the language's rows of the phase are deleted (back to the default language / built-in copy) |

`PhaseTips` = `{phase, texts: {code: {customized (the language has rows of its own for the phase), note: {body, source},
tips: [{slot, title, body, source}] ×3}}, rows: [{id, item_key, locale, is_active, is_approved, updated_at}],
updated_at}` — the texts are what the panel shows **now** (placeholders not filled); `source` ∈ `locale |
default_language | built_in`. Audit lines: `companion_tip.update` (attrs `phase`, `locales`, `rows_created`),
`companion_tip.reset` (`phase`, `locale`, `rows_deleted`).

**Link overview** (read-only):

| Method | Path | Role | Query | `data` |
|---|---|---|---|---|
| GET | `/companions/links` | S | `status=all\|invited\|active\|revoked`, `type=all\|partner\|spouse\|parent` (unknown → `all`), `page`, `per_page` | list, newest first (`created_at DESC, id DESC`), items `{id, type, status, label, owner{name, mobile}, companion{name, mobile}\|null, invite{phone, expires_at, expired}\|null (status invited only), grants_count, invited_at, accepted_at, revoked_at, revoked_by, created_at}` + `filters{status, type}` + `counts{by_status{all, invited, active, revoked}, by_type{partner{…, all}, spouse{…, all}}}` (over every link, not the filter) + `statuses[]` + `types[]` |

Masking: names and the owner's label keep the first letter (`س•••`), mobiles and the invite phone keep only the
last 2 digits (`•••••••••67`). Never sent: user ids (B-N4-08b, CMP-L6: the relationship graph), invite codes or their
hash, attempts, which sections are shared (only `grants_count`), anything of the owner's health data. Super admins only
(CMP-L6).

## 17. AI usage and cost (B-N6-05)

Aggregates of `ai_usage_logs` (one row per AI provider call: feature, op, provider, model, tokens, audio / image
bytes, estimated cost in micro-USD from `AI_PRICES`, latency, ok — never content). Any active admin (**A**): no user
ids, no content. The admin-web screen comes with B-N9.

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/ai/usage` | A | `from`, `to` (`Y-m-d`, Tehran days, inclusive; default the last 30 days up to today; ≤ 92 days, 422 `validation_failed` otherwise) | `{from, to, today{spent_micros, spent_usd, cap_micros, cap_usd, exhausted}, totals{calls, ok_calls, input_tokens, output_tokens, cost_micros, cost_usd}, by_day[{day, calls, ok_calls, input_tokens, output_tokens, cost_micros, cost_usd}], by_feature[{feature, provider, model, calls, ok_calls, users, input_tokens, output_tokens, audio_bytes, image_bytes, avg_latency_ms, cost_micros, cost_usd}]}` (by_feature: highest cost first) |

`today` is the spend of the current Tehran day against `AI_DAILY_COST_CAP_USD`; `exhausted: true` means every AI
feature currently answers 503 `ai_budget_exhausted`.

## 18. Child catalogs, WHO viewer and postpartum copy (B-N5-09)

**Child catalogs** (B-N5-02) are `catalog_items` groups edited through the generic catalog API
(`/api/admin/v1/catalog/{group}`, docs/canvas-build/catalog.md) — no new write endpoint. Groups and the `meta` shape
`internal/children` reads (an item whose meta lacks a required key is skipped by the app, never an error):

| Group | admin-web | `meta` |
|---|---|---|
| `child_vaccines` | «برنامهٔ واکسن» | `visit` (req; doses sharing a visit form one appointment: `birth, m2, m4, m6, m12, m18, y6` or a new snake-case code), `age_months` (req, ≥ 0; the due date is the birth date + that many calendar months), `admin_note?` (editorial only, never served) |
| `child_milestones` | «نقاط عطف» (by month) | `age_months` (req), `domain` `social\|language\|motor\|cognitive` |
| `child_milestone_activities` | same page, «بازی‌های این ماه» | `age_months` |
| `child_milestone_notes` | same page, «کی به پزشک بگم» | `age_months` |
| `child_age_notes` | same page, «این هفته …» | `age_months` (the latest band ≤ the child's age is shown) |
| `child_learn` | «آموزش کودک» | `topic` `sleep\|feeding\|play\|health\|mother`, `from_months`, `to_months` (req), `minutes`, `featured`, `article_slug?` |

Title = the fa/en label, body = «در برابر …» (vaccines) / the text. Rows start `needs_review: true`; «بازبینی شد» is a
`PUT` with `needs_review: false` (title is required on every PUT, absent optional fields are kept).

**Super-only writes (clinical content):** `POST/PUT/DELETE /catalog/{group}[/{id}]` and `POST /catalog/{group}/reorder`
on the six child groups answer **403** for editors (`catalog.Admin.WithSuperGroups(children.Groups...)` in
`routes_admin_catalog.go`); reads stay **A**. The same holds in `/messages` for `postpartum_week_tip`,
`postpartum_alert` and `postpartum_safety` (`POST /messages`, `PUT /messages/:id`, `POST …/approve`, `POST …/toggle` →
403 for editors; `registry.SuperOnlyGroups`); `GET /messages` now also returns `super_only_groups[]` so admin-web hides
the buttons. Existing audit lines (`catalog_item.*`, `message.*`) and CSRF apply.

**Postpartum copy** (B-N5-01, `internal/postpartum/guide`) has no endpoint of its own: the registered rows are created
and edited in `/messages` (`?group=postpartum_week_tip|postpartum_alert|postpartum_safety`, «ایجاد» from `missing`).
Unwritten slots fall back to the embedded fa/en copy; the EPDS call numbers 115 / 123 / 1480 are fixed in code.

**WHO growth standards** (read-only; package `internal/admin/children`, `routes_admin_children.go`):

| Method | Path | Role | Query | `data` |
|---|---|---|---|---|
| GET | `/children/who` | A | `indicator` `weight\|length\|head` (default weight), `sex` `girl\|boy` (default girl), `step` `month\|week\|day` (default month; an unknown value of any of the three → 422), `page`, `per_page` (≤ 100) | list `{items:[{age (in step units), day, l, m, s, p3, p15, p50, p85, p97}], meta}` + `filters{indicator, sex, step}`, `unit` (`kg`\|`cm`), `options{indicators[], sexes[], steps[], max_day: 1856, percentiles[]}`, `source` (citation), `read_only: true` |

Month sampling is `day = round(month × 30.4375)` (months 0–60), weeks `day = 7 × week` (0–265), days 0–1856. L/M/S
are the embedded WHO expanded-table values verbatim; percentiles are `growth.ValueAt` at Φ⁻¹(p), rounded to 3
decimals (weight-for-age uses WHO's restricted method beyond ±3 SD). **Import is out of scope:** the standards are
fixed reference data embedded in `backend-go/seeds/who` (golden-tested); replacing them is a code change with new CSVs,
not an admin action.

## 19. Doctors directory — «پزشکان و ماماها» (B-N7-02)

Package `internal/telemed` (`admin.go`, `admin_schedule.go`), `routes_admin_telemed.go`. The minimal CRUD the public
directory (`/api/v1/telemed/*`, D-67) needs; the console UI (bookings, doctor replies, `doctor` role via
`telemed_doctors.admin_id`) is B-N7-08. **A** = editor or super, **S** = super only. Writes are audit-logged
(`telemed_doctor.create|update|delete|photo|photo_delete|visit_types|availability|time_off|time_off_delete`,
`telemed_review.update`) with ids only. Translatable fields (`name`, `headline`, `bio`, visit-type `note` / `address`)
are `{lang: text}` objects; only the default language of `name` is required. Specialty / city / insurer values are
`catalog_items` codes of the groups `telemed_specialties` (six seeded by goose 00045, `needs_review`), `telemed_cities`
and `telemed_insurers` (empty: admins add them through `/catalog/{group}`); an unknown or inactive code is a 422.
Money is integer **rials** (`price_rials`; clients show toman).

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/telemed/doctors` | A | `q` (name in any language or licence no.), `status` `all\|active\|inactive`, `page`, `per_page` | `{items:[doctor + modes[] (active visit modes)], meta, filters}` |
| GET | `/telemed/doctors/options` | A | — | `kinds[]`, `modes[]`, `weekdays` (0 = Sunday … 6 = Saturday), `catalog_groups`, `limits`, `currency: IRR`, `next_sort_order` |
| POST | `/telemed/doctors` | A | `kind` `doctor\|midwife`, `name{}`, `headline{}?`, `bio{}?`, `specialty`, `city?`, `licence_no` (≤ 32), `experience_years?` (0–80), `response_minutes?` (1–10080), `insurers[]?` (≤ 30 codes), `is_active?` (default true), `sort_order?` (default last) | 201 `{doctor}` |
| GET | `/telemed/doctors/{id}` | A | — | `{doctor}` = id, kind, name, headline, bio, specialty, city, licence_no, experience_years, photo_path, photo_url, response_minutes, visits_count, rating, reviews_count, satisfaction_percent, is_active, sort_order, admin_id, created_at, updated_at, `insurers[]`, `visit_types[]` (inactive included), `availability[]`, `time_off[]` (ending after now) |
| PUT | `/telemed/doctors/{id}` | A | the POST body; `kind`, `name`, `specialty`, `licence_no` required again; absent optional fields keep their value, `null` clears; `insurers` replaces the set when sent | `{doctor}` |
| DELETE | `/telemed/doctors/{id}` | S | — | `{id}` — cascades visit types, availability, time off, insurers, reviews; the photo file is removed. Prefer `is_active: false` |
| POST | `/telemed/doctors/{id}/photo` | A | multipart `photo` (jpeg/png/webp ≤ 4 MB, ≥ 200×200) → WebP ≤ 1080² under `doctors/` on the public disk | `{doctor}` |
| DELETE | `/telemed/doctors/{id}/photo` | A | — | `{doctor}` |
| PUT | `/telemed/doctors/{id}/visit-types` | A | `visit_types: [{mode, duration_minutes (5–240), price_rials (0–10¹⁰), note{}?, address{}?, is_active?}]` — the full set, each mode once; modes left out are deleted | `{doctor}` |
| PUT | `/telemed/doctors/{id}/availability` | A | `rules: [{weekday 0–6, start_time "HH:MM", end_time "HH:MM" (≤ 24:00, after start), slot_minutes 5–240 (≤ the window), modes[]? (null = all)}]` (≤ 50) — replaces the weekly schedule | `{doctor}` |
| POST | `/telemed/doctors/{id}/time-off` | A | `starts_at`, `ends_at` (`Y-m-d H:i`, Tehran, end after start, ≤ 366 days), `note?` | 201 `{doctor}` |
| DELETE | `/telemed/doctors/{id}/time-off/{off}` | A | — | `{doctor}` |
| GET | `/telemed/doctors/{id}/slots` | A | `mode`, `from`, `days` as the public route | the public slot grid (inactive doctors too) |
| GET | `/telemed/reviews` | A | `doctor_id?`, `visibility` `all\|visible\|hidden`, `page`, `per_page` | `{items:[{id, doctor_id, user_id, booking_id, rating, body, is_visible, created_at, updated_at}], meta, filters}` |
| PUT | `/telemed/reviews/{id}` | A | `is_visible` (required) | `{review}` — the doctor's rating is recomputed from visible reviews |

A doctor is listed publicly only while `is_active` and with at least one active visit type. Slots come from the
availability grid (a visit must end inside its window, start ≥ 30 min from now, within 60 days) minus time off and the
booked intervals B-N7-03 reports through `telemed.Busy`.

### 19b. Doctor console — chat, prescription, video (B-N7-04)

For the admin account linked to a doctor by `telemed_doctors.admin_id` (any role; 403 `doctor_required` otherwise).
Only that doctor's confirmed / completed bookings exist here (anything else 404). Message bodies are decrypted for the
response only and never logged.

| Method | Path | Role | Input | Output |
|---|---|---|---|---|
| GET | `/telemed/doctor/bookings` | A | — | `{items:[{id, reference, mode, status, starts_at, ends_at, for_whom, chat_open}]}` |
| GET | `/telemed/doctor/bookings/{id}/messages` | A | `after_id?` | `{items:[message], chat:{open, closes_at}}` — marks the user's lines read |
| POST | `/telemed/doctor/bookings/{id}/messages` | A | `body` (required, ≤2000) | 201 `{message}`; 409 `chat_closed` / `message_limit` |
| GET | `/telemed/doctor/bookings/{id}/prescription` | A | — | `{items}` |
| POST | `/telemed/doctor/bookings/{id}/prescription` | A | `drug`, `dose?`, `form` (care forms), `times[1–4]` H:i, `duration_days?` 1–365, `instructions?` | 201 `{item}`; 409 `prescription_limit` (10) |
| DELETE | `/telemed/doctor/bookings/{id}/prescription/{item}` | A | — | `{id}` (a reminder the user made stays) |
| GET | `/telemed/doctor/bookings/{id}/video` | A | — | `{video:{active, opens_at, closes_at, url}}` |

### 19c. Bookings console, refund queue, assistant departments (B-N7-08)

`internal/telemed/admin_bookings.go`, `internal/assistant/admin.go`, registered in `routes_admin_telemed.go`. No notes,
reasons or message text is exposed; the patient mobile is masked (`0912•••0001`). Audit lines carry ids only.

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/telemed/bookings` | A | `status`, `payment_status`, `doctor_id`, `q` (reference), `page`, `per_page` (unknown values = all) | `{items:[{id, reference, user_id, user_mobile, doctor{id,name{}}, mode, status, payment_status, starts_at, ends_at, total_rials, refunded_rials, refund_ref, refunded_at, paid_at, gateway, created_at}], meta, filters, counts{refund_pending}, statuses[], payment_statuses[]}` |
| GET | `/telemed/bookings/refunds` | A | as above | the `refund_pending` queue (newest first) |
| POST | `/telemed/bookings/{id}/refunded` | S | `refund_ref?` (≤ 64) | `{id, payment_status:"refunded"}`; 409 `not_refund_pending` when not pending; audit `telemed_booking.mark_refunded` |
| DELETE | `/telemed/doctors/{id}` | S | — | now 409 `doctor_has_bookings` when the doctor has held / confirmed / completed bookings or paid / refund_pending money; deactivate instead |
| GET/POST | `/assistant/departments` | S | POST: `code` (a-z0-9_, ≤ 32, unique), `name{}`, `tagline{}?`, `intro{}?`, `does_not_do{lang:[]}?`, `quick_replies{lang:[]}?` (≤ 10 × 120), `icon?`, `sort_order?`, `system_prompt` (≤ 4000), `guardrails?` (≤ 2000) | list `{items, limits}`; create → 201 `{department}`, always inactive + `needs_review` |
| GET/PUT | `/assistant/departments/{id}` | S | PUT = full form + `is_active?` | `{department}`; changing `system_prompt` / `guardrails` sets `needs_review`; switching an unreviewed department on is a 422 on `is_active` |
| POST | `/assistant/departments/{id}/approve` | S | — | `{department}` (needs_review → false); audit `assistant_department.approve` |

The emergency floor (keywords → 115 card, diagnosis guard, base system guard) is code-held and not editable here. Updates
are audited as `assistant_department.update` with the changed-field flags, never the prompt text.

## 20. Instructors & courses moderation — «مدرسین و دوره‌ها» (B-N8-08)

Package `internal/admin/learning`, routes `routes_admin_learning.go`, queries `db/queries/admin/learning.sql`, over the
B-N8-01 courses domain (`internal/learning`, goose 00046) and goose **00054** (+ Laravel twin
`2026_10_07_000054_add_review_to_learning_lessons.php`): `learning_lessons.review_status|reviewed_at|reviewed_by|
review_note` and the `learning_moderation_log` table. **A** = editor or super, **S** = super only. Every write passes
the CSRF check (419) and writes one `learning_moderation_log` row (admin id, action, target, instructor id, note) plus the
slog `admin audit` line `learning.<action>` (ids only — never notes, phone numbers or lesson text).

**No health data.** Students are only counted; completion is aggregated per course; grant phone numbers are never
selected; the instructor's own mobile is masked (`0912•••4567`, the full number stays on `/users/:id`).

**Review state** of a lesson (`review.state`, what B-N9-10's safety queue consumes): `pending` (never reviewed — every
existing and new lesson), `approved`, `changed` (approved, then edited: `updated_at > reviewed_at`; the instructor code
does not reset anything — an edit or a B-N8-02 media replacement bumps `updated_at`), `flagged` (needs attention, with
the admin's note). `open` = pending + changed + flagged. The queue holds published lessons plus flagged ones (also after
a takedown). Approve / flag write only the review fields (not `updated_at`); unpublish sets `status=draft` +
`flagged` and bumps `updated_at`.

| Method | Path | Role | Body / query | `data` |
|---|---|---|---|---|
| GET | `/learning/stats` | A | — | `{instructors{all, pending, approved, revoked}, courses{all, draft, published}, lessons{all, published}, review{open, pending, changed, flagged, approved}, grants{all, active, pending, expired, revoked, new_30d}, students{with_access (distinct users with a running grant), active_30d (opened a lesson)}, completion{percent (mean course % over enrolments), enrolments (course × student with access), completed (at 100 %), lessons_completed}, active_days: 30}` |
| GET | `/learning/instructors` | A | `status=all\|pending\|approved\|revoked` (unknown → all), `q` (display name, account name, part of the mobile; Persian digits ok), `page`, `per_page` | list, pending first then newest; items `Instructor` + `filters{status, q}` + `counts{all, pending, approved, revoked}` + `statuses[]` |
| GET | `/learning/instructors/:id` | A | — | `{instructor: Instructor, history: [{id, action, target_type, target_id, admin{id, name}\|null, note, created_at}] (last 20, newest first: status changes and reviews of her lessons)}` |
| POST | `/learning/instructors/:id/approve` | S | `note?` (≤ 300) | `{instructor, changed}` — pending / revoked → approved (`approved_by` = the admin): she reaches `/api/instructor/v1` and students see her published courses. Already approved → `changed: false`, message `No change.`, no log row |
| POST | `/learning/instructors/:id/revoke` | S | `note?` (≤ 300) | `{instructor, changed}` — approved / pending (= rejecting an application) → revoked: her panel answers 403 `instructor_required`, students stop seeing her courses; nothing is deleted (she may re-apply → pending) |
| GET | `/learning/courses` | A | `status=all\|draft\|published`, `instructor_id?`, `q` (title), `page`, `per_page` | list, newest first; items `{id, title, kind, status, instructor{id, display_name, status}, chapters_count, lessons_count, published_lessons, review_open, flagged_count, students_count, completion_percent, completed_count, published_at, created_at, updated_at}` + `filters{status, q, instructor_id}` + `counts{all, draft, published}` + `statuses[]` |
| GET | `/learning/courses/:id` | A | — | `{course: {id, title, description, kind, status, instructor{id, display_name, title, status}, chapters[{id, title, sort_order, unlock_at}], lessons[Lesson + sort_order] (drafts included), grants{all, active, pending, expired, revoked} (direct + through groups), usage{students_count, completion_percent, completed_count}, published_at, created_at, updated_at}}` |
| GET | `/learning/reviews` | A | `state=open\|pending\|changed\|flagged\|approved\|all` (default / unknown → open), `page`, `per_page` | list of `Lesson + course{id, title, kind, status} + instructor{id, display_name, status}`; open states oldest change first (a queue), approved / all newest first; + `filters{state}`, `counts{open, pending, changed, flagged, approved}`, `states[]` |
| POST | `/learning/lessons/:id/approve` | A | — | `{lesson}` — clears a flag and its note; publication status unchanged |
| POST | `/learning/lessons/:id/flag` | A | `note` (req, 3–300) | `{lesson}` — still visible to students |
| POST | `/learning/lessons/:id/unpublish` | A | `note` (req, 3–300) | `{lesson}` — takedown: `status=draft` (students stop seeing it at once), flagged |

`Instructor` = `{id, display_name, title, bio, status, user{id, name, mobile (masked)}, courses_count, published_courses,
students_count, approved_at, approved_by{id, name}|null, revoked_at, created_at, updated_at}`.
`Lesson` = `{id, chapter_id, kind, title, description, duration_seconds, page_count, size_bytes, media_status, status,
published_at, review{state, status, reviewed_at, reviewed_by{id, name}|null, note}, created_at, updated_at}`.

Completion per (course, student) is the student app's own formula (`learning.CoursePercent`: summed lesson percent —
a completed lesson counts 100 — over the course's published lessons); a course's `completion_percent` is the rounded
mean over the students with a running grant (students who never opened it count 0). Media playback for reviewers comes
with the B-N8-02 media pipeline; the queue shows the metadata and `media_status`. Moderation log actions:
`instructor.approve|revoke`, `lesson.approve|flag|unpublish`.

## 21. Roles & permissions, audit log, temporary health-data access (B-N9-02, D-84)

Package `internal/admin/rbac` (+ `httpadmin/{permissions,audit,health}.go`), routes `routes_admin_rbac.go`, queries
`db/queries/admin/rbac.sql`, goose **00060** (+ Laravel twin `2026_10_10_000060_create_admin_rbac_audit_tables.php`).

**Permission check.** `kit.Admin` routes now run `RequirePermission`: the route path maps to one capability
(`httpadmin.capabilityOf`; GET = `view`, other methods = `manage`), the admin's role must grant it → else **403**
`{"error_code":"forbidden","permission":"<capability>"}` (never 401). `super` passes everything; `editor` holds every
capability except `health` (narrowed in B-N9-02b, below). Paths not in the map are super only (B-N9-02b).
`kit.Super` routes are unchanged. Capabilities: `dashboard, users, health (manage), messages, clinical, challenges,
config, export (view), content, billing, support, learning, telemed, telemed_doctor (manage), audit (view)`.

**Audit log.** `httpadmin.Audit(...)` (every module) and the login write the slog line **and** an
`admin_audit_logs` row: admin, action, target type + id, ip, time, `detail` = the scalar extras only (numbers, flags,
strings ≤ 64 chars). Never pass free text or health values to `Audit`.

**Health data.** `/users/:id` returns the profile's health fields (`birthday`, `last_period_start`,
`cycle_duration`, `period_duration`, `pregnancy_intention`) as null unless the caller has a live grant, plus
`health_access: {granted, expires_at}`. A grant = role with `health` (clinical support) + a request for that user
approved by another super admin, valid **30 minutes**. Super admins need one too.

| Method | Path | Role | Body / query | `data` |
| --- | --- | --- | --- | --- |
| GET | `/roles` | S | — | `{capabilities:[{key, levels}], roles:[{key, editable, customized, admins, permissions}]}` |
| PUT | `/roles/:role` | S | `permissions: {capability: view\|manage\|none}` | `{role}`; 422 `role_locked` (super), 404 unknown, 422 bad capability/level; audit `role.update` |
| POST | `/roles/:role/reset` | S | — | `{role}` back to the code defaults; audit `role.reset` |
| GET | `/audit-logs` | audit | `admin_id, action` (prefix), `target_type, target_id, from, to` (YYYY-MM-DD, Tehran), `page, per_page` | list of `{id, admin {id,name,role}\|null, action, target_type, target_id, ip, detail, created_at}` |
| GET | `/audit-logs/export` | audit + export | same filters | `text/csv` (≤ 10 000 rows, formula-safe); audit `audit_log.export` |
| GET | `/health-access` | health | `status?, page` | own requests (super: all) `{id, admin, user_id, reason, ticket, status pending\|approved\|rejected\|revoked\|expired, decided_by, decided_at, expires_at, created_at}` |
| POST | `/health-access` | health | `user_id, reason` (10–500), `ticket` (≤ 64) | 201 `{request}`; audit `health_access.request` |
| POST | `/health-access/:id/approve` | S | — | `{request}` approved for 30 min; 422 `self_approval`, 409 `not_pending`; audit `health_access.approve` |
| POST | `/health-access/:id/reject` | S | — | `{request}`; 409 `not_pending`; audit `health_access.reject` |
| POST | `/health-access/:id/revoke` | health | — | requester or super (else 404); 409 `not_active`; audit `health_access.revoke` |
| GET | `/health-access/users/:id` | health + grant | — | `{user_id, expires_at, profile, stats}`; 403 `health_access_required`; audit `health_access.read` (written first; 503 `audit_unavailable` and no data when it fails, T-M10-33) |
| GET | `/audit/verify` | S | `from?` (audit row id; default 0 = the whole log, genesis included) | `{ok, started, from, checked, genesis_last_id, head:{id, hash}, break:{id, reason}\|null}`; 422 on a non-numeric `from`; read-only (T-M10-33) |

**Security follow-up (B-N9-02b).**
- *Approver independence:* approve also answers 422 `approver_not_independent` when the approving super admin's account
  is younger than 24 h, or the requester wrote an `admin.create` / `admin.update` audit row about the approver in the
  last 30 days (derived from `admin_audit_logs`, no new column); 422 `requester_not_permitted` when the requester is
  inactive or their role lost `health`. Every approval logs a WARN line `alert=health_access_approved`.
- *Expiry & clock:* a request pending for 24 h reads `expired` and cannot be decided (409 `not_pending`); grants and
  decisions use the real clock (X-Test-Now is ignored there).
- *`user_goal` is health data:* null in the `/users` list and in `/users/:id` (`user` + `profile`) without a grant;
  `PUT /users/:id` takes `user_goal` as nullable (absent / null = unchanged); the `user.update` audit carries only
  `goal_set`.
- *Least privilege:* analyst and engineering get `users: view`; product manager and engineering lose `audit`; unmapped
  paths are **super only**; `editor` becomes editable (`PUT /roles/editor`) with the default "everything except
  `health`, `audit`, `export`, `telemed_doctor`". **Migration path for existing editors:** they keep their content /
  users / billing reach; if one still needs the audit log or exports, a super admin grants `audit` / `export` to
  `editor` on the Roles screen (or moves that account to a narrower role).
- *Audit log:* the `ip` is null (CSV: empty) for non-super viewers; every CSV string cell goes through the formula
  guard after leading whitespace; failed logins write `auth.login_failed` (target = the account id when known, never
  the email). All `/api/admin/v1` answers carry `Cache-Control: no-store`.

**SPEC v6 §5.8 role matrix and hash-chained audit log (T-M10-33, D-182).**
- *Capabilities:* new rows `kpi`, `okr` (split from `dashboard`), `thresholds` (`/clinical-config`, split from
  `config`) and `segments` (`/segments`); `/users/directory/export` maps to `export` (not `users`) and `/audit/*` to
  `audit`. Defaults (deny by default; the spec→capability table is in `httpadmin.DefaultPermissions`):
  product_manager `dashboard m, kpi m, okr m, users v, messages m, challenges m`; medical_advisor `dashboard v,
  messages m, clinical m, thresholds m`; support `users v, billing v, support m`; clinical_support = support + `health
  m`; analyst `kpi v, export v`; marketing `messages m, challenges m, segments m`; engineering `config m, thresholds
  v`. Matrices already saved in `admin_roles` are not rewritten (they hold none of the new capabilities, so a saved
  role loses `/kpi`, `/okr`, `/clinical-config` until a super admin grants them).
- *Hash chain* (`internal/admin/auditchain`, goose 00105 columns): every `httpadmin.Audit` row is appended in one short
  transaction that locks `admin_audit_chain_head` (`FOR UPDATE`), sets `prev_hash` = the head's hash and
  `row_hash = sha256(prev_hash ‖ canonical JSON {admin_id, action, target_type, target_id, ip, detail, created_at})`,
  then moves the head. Rows written before the chain started keep NULL hashes; the first append (or the job) folds
  them oldest first into `genesis_hash` / `genesis_last_id`. `GET /audit/verify` reports the first break:
  `genesis_mismatch` (a pre-chain row edited or deleted), `unchained` (a row without hashes after genesis),
  `prev_mismatch` (a row deleted or reordered), `hash_mismatch` (a row edited), `head_mismatch` (the tail deleted).
  The nightly `admin.audit_verify` job (docs/engine-v6/jobs.md) sets the `admin_audit_chain_broken` gauge and logs the
  head hash as the off-database anchor.
- *Fail closed:* every health-data read (`/health-access/users/:id`, `/users/:id`, `/users/:id/overview`) writes its
  `health_access.read` row through `httpadmin.AuditHealthRead` before any health field is built; when the row is not
  written the answer is **503** `audit_unavailable` with no data.

## 22. Smart messages v2 — rules (B-N9-05, D-86)

Capability `messages` (GET view, writes manage); the medical-approval actions additionally need `clinical` at manage.
Rule shape: `{id, title, level, kind, priority, placement, content{a{title,body,cta_label,cta_target}, b?, seen?, confidence?,
contact?}, variants_enabled, ab_split, conditions[{op:and|or, items[{fact,op,value}]}], exceptions[{fact,op,value}], cap_count,
cap_period, cooldown_days, hours_from, hours_to, starts_on, ends_on, audience_percent, goal, status, is_active, version,
needs_medical_approval, approved_by, approved_at, …}`.

| Method + path | Notes |
|---|---|
| `GET /messages/rules?placement=&kind=&status=&page=` | items + `stats_7d`; `enums` (placements, levels, kinds, bands, ops, variables, cap_periods) |
| `POST /messages/rules` | 422 per field (`priority` outside the kind's band, product without cap, follow_soon rules, unknown variable…) |
| `GET /messages/rules/{id}` | rule, `versions`, `variant_stats_7d`, `conflicts` |
| `PUT /messages/rules/{id}` | whole definition; version +1, snapshot, status → `draft` (approval withdrawn) |
| `DELETE /messages/rules/{id}` | |
| `POST /messages/rules/{id}/submit` | medical rules, draft → pending_approval (422 `approval_not_needed`, 409 `not_pending`) |
| `POST /messages/rules/{id}/approve` · `/reject` | `clinical` manage; only from pending (409 `not_pending`) |
| `POST /messages/rules/{id}/toggle` | publish / unpublish; 409 `approval_required` for an unapproved medical rule |
| `POST /messages/rules/conflicts` `{rule\|rule_id}` | overlapping messages of the placement with the winner (`this` / `other` / `tie`) |
| `POST /messages/rules/preview` `{rule\|rule_id, user_id?, facts?}` | `{eligible, reasons[], variant, title, body, cta_label, level}`; audited when a user id is used; fact values never returned |
| `GET /messages/rules/engine` | `{placements[{placement, urgent, count, rules[{rule_id,title,kind,level,priority,served,unlimited,cap,cooldown_days,audience_percent,variants,conflicts}]}], bands}` — live rules per placement in engine order (B-N9-06) |
| `POST /messages/rules/simulate` `{facts, at?, user_id?, placements?}` | `{at, placements[{placement, winner\|null, trace[{rule_id,title,kind,level,priority,eligible,reason,failed_groups[],variant,winner}]}]}` — pure engine over synthetic facts (no writes, no real user data; `user_id` only seeds A/B/audience buckets; empty history so caps never trigger); `messages` capability, manage level (POST) |
| `POST /messages/simulate` `{user_id?, date?, time?, slots?, facts_override?}` (T-M10-11) | `{source: user\|synthetic, at, date, time, zone, resolution, history, facts_overridden, engine{main_phase, subphase, cycle_day, days_late, confidence, data_quality, warnings, requires_user_input}, slots[{slot, capacity, winners[{rule_id, variant, version, title, body, …}], dropped[{rule_id, reason}]}], reasons[], notes[]}` — winners equal the feed's; with `user_id`: `messages` capability **plus** `health` manage and a live health-access grant (403 `health_access_required` otherwise, nothing about the user leaks), audit `health_access.read` (fail closed, 503) + `messages.simulate`; real facts and history in the user's stored zone; no events written. Without `user_id`: override facts, empty history |
| `GET /messages/rules/conflicts?slot=` (T-M10-11) | `{data: [{rule_id, title, priority, kind, winner, safety, slot, tie_break, co_shown, overlap{schedule, hours, audience, facts{<fact>: range\|values}}}]}` — conflict check v2 over every active pair (DNF of conditions, fact value intersection, schedule, hours minus slot quiet hours, audience); `messages` capability, view. The rule show/save `conflicts` list carries the same v2 fields |

Static `/messages/rules*` routes are registered before `/messages/{id}`. Audit actions: `message_rule.create|update|delete|submit|approve|reject|toggle|preview`.

### Message rules v6 additions (T-M10-02, T-M10-03, T-M10-12)

New rule fields (all optional on write; a legacy a/b rule round-trips unchanged): `rule_key` (unique, `a-z0-9_.-`, 1-64;
duplicate = 422), `approved_version` / `served_version`, `topics[]` (`fertile_window` is required when a condition reads a
fertility fact), `caps[{count, period: day|week|cycle}]`, `variants[{key, weight}]` with each variant's text in
`content.variants{key: {title, body, cta_label, cta_target}}` (at most `enums.max_variants` = 6, weights positive; a safety
rule has exactly one), `starts_at` / `ends_at` (datetime,
user-local wall clock), `hour_windows[[from_hour, to_hour], ...]`, `goal_event`, `locked`, and read-only `type` / `slot` / `name` /
`level_name` (spec names). New `enums` on the list: `goal_events`, `topics`, `level_kinds` (level -> allowed kinds),
`slot_kinds` (slot -> allowed kinds), `approval_kinds` (`safety`, `clinical`, `content`), `max_variants`. Stored levels stay
`worth_followup` / `follow_soon`; the spec names `follow_up` / `urgent` are accepted as aliases.

Authority: without the `clinical` capability, create / update of `safety|clinical|content` rules and toggle / delete of
`safety` rules answer 403. Editing the text of a medical rule sends it straight to `pending_approval`. A `locked` (seeded)
rule answers 409 `locked` on delete / switch-off, and only the clinical capability edits its copy. A suggestion needs exactly
one CTA; slot x kind and level x kind mismatches are 422 on `placement` / `level`.

`GET /messages/facts` (capability `messages`, view): the fact registry for the condition builder.
`{facts[{name, type: int|float|bool|date|enum|set|string, enum|null, unit|null, group, description{<lang>: text},
member_safe, pending|null}], groups[], fertility_facts[]}`, sorted by name. `pending` is the id of the task that will fill
the fact (the editor disables it). Text variables (`name`...) are not facts and stay in the rules `enums.variables`.

422 codes on create / update: besides the per-field `errors`, the body carries `error_code` = `unknown_fact` when a
condition or exception names a fact that is not in the registry (wins), or `bad_value` when an operator or value does not
fit the fact's type (e.g. `between` on a bool, a value outside the enum). Other error codes: 409 `locked`,
409 `approval_required`, 409 `not_pending`, 422 `approval_not_needed`. admin-web maps them to a banner and the
`conditions` field (`screens/message-rules/lib/errors.ts`).


## 23. Clinical thresholds & config — «آستانه‌ها و پیکربندی» (B-N9-11, D-88)

Package `internal/admin/clinicalcfg` (store side) + `internal/clinicalconfig` (leaf registry + cached lookup the
engines call), routes `routes_admin_clinical_config.go`, queries `db/queries/admin/clinical_config.sql`, goose
**00063** (+ Laravel twin `2026_10_11_000063_create_clinical_config_tables.php`). Capability `config` (GET = view,
writes = manage); approve / reject additionally need role `medical_advisor` (403 `medical_advisor_required`).

**Model.** Every key (`bp.crisis`, `glucose.fasting`, `epds.cutoff.likely`, `cycle.figo.frequency`,
`engine.period_hard_cap`, `engine.fertile_window`, `lab.ferritin.low`, … — `clinicalconfig.Defs`) has a value
(named numbers, e.g. `{"systolic":180,"diastolic":120}`), unit, source, version and effective date. **Version 1 is
the code default** (the pre-B-N9-11 constants; never stored). A stored version is `pending` → `approved` after **two
distinct medical advisors** approve (never the proposer) or `rejected`. The server reads the approved version with the
latest `effective_from` ≤ the engine's target date (reading day, check day, cycle target, report day), so historic
outputs keep their thresholds. Snapshot cached in-process, refreshed on approval and every minute; an empty or
unreachable store answers the code defaults.

| Method | Path | Who | Body / query | `data` |
| --- | --- | --- | --- | --- |
| GET | `/clinical-config` | config | `group?` (`engine\|vitals\|pregnancy\|lab`) | `{groups, keys:[{key, group, unit, fields:[{name, default, min, max}], current, upcoming, pending}], can_approve, required_approvals, today}`; a version = `{id, version, value, source, note, effective_from, status, proposed_by {id,name}, approvals:[{id,name,at}], required_approvals, decided_at, created_at}` (v1: `id`/`effective_from` null) |
| GET | `/clinical-config/resolve` | config | `key, date` (Y-m-d) | `{key, date, version, value, effective_from}` — what the server reads |
| GET | `/clinical-config/:key` | config | — | `{key, versions:[stored newest first…, v1], can_approve, required_approvals}`; 404 unknown key |
| POST | `/clinical-config/:key/proposals` | config manage | `value {field: number}` (all fields, in range, ordered pairs increasing), `effective_from` (≥ today), `source?` (≤ 255, default the v1 source), `note?` (≤ 500) | 201 `{proposal}`; 409 `pending_exists` (one pending per key); 422 validation; audit `clinical_config.propose` |
| POST | `/clinical-config/proposals/:id/approve` | medical advisor | — | `{proposal}` (approved on the 2nd distinct approval); 403 `self_approval`, 409 `already_approved` / `not_pending`, 422 `effective_date_passed`, 404 unknown; audit `clinical_config.approve` |
| POST | `/clinical-config/proposals/:id/reject` | medical advisor or the proposer | — | `{proposal}`; 409 `not_pending`; audit `clinical_config.reject` |

## 24. Safety & clinical content — «ایمنی و محتوای بالینی» (B-N9-10, D-90)

Capability `clinical` (path segment `safety`). Aggregates and review metadata only: counts, titles, ids, ages and admin /
instructor names — never a user id or a health value. Approve / reject stay on each source's own screen; the queue
deep-links there. Claims are rows of `admin_audit_logs` (`safety.claim` / `safety.release`, target type
`safety.<source>`, last 30 days) — no new table.

| Method | Path | Who | Body | Response |
|---|---|---|---|---|
| GET | `/safety` | clinical:view | — | `{window_days: 7, today, signals[{key, count, previous, delta}] (bp_crisis, glucose_urgent_low, epds_q10, reduced_kicks, contractions_511, postpartum_heavy_bleeding — thresholds in force today), queue{pending, overdue, sla_hours: 72, by_source{message, threshold, lesson, assistant, catalog}, items[{source, id, title, ref, status (pending\|changed\|partially_approved), approvals, required_approvals, items, submitted_by, submitted_at, age_hours, overdue, claimed_by{id, name, at}\|null}]}, checks{modal_latency_p95_ms: null, call_115_ctr: null (not instrumented, D-90), text_parity{ok, unapproved, groups[]}, scenarios{ran_at, passed, total, failed[]}\|null}}` |
| POST | `/safety/queue/:source/:id/claim` | clinical:manage | — | `{claimed_by}`; idempotent for the claimer; 409 `already_claimed`; 404 when the item is not in the queue; audit `safety.claim` |
| POST | `/safety/queue/:source/:id/release` | clinical:manage (the claimer or super) | — | `{claimed_by: null}`; 403 `not_claimer`; audit `safety.release` |
| GET | `/safety/runs` | clinical:view | — | `{runs:[{run_id, started_at, scenarios, passed, failed}], limit: 30}` — the persisted `safety.nightly` self-test runs, newest first (T-M10-24, D-183) |
| GET | `/safety/runs/:run_id` | clinical:view | — | `{run_id, started_at, results:[{scenario, kind: path\|config, passed, duration_ms, detail{threshold, probes[{input,want,got}], problems[]}}]}`, failures first; 404 for an unknown or malformed id |

## 25. Shop catalog — «فروشگاه» (B-N10-03, D-91)

Capability `content`. Categories: `GET|POST /shop/categories`, `GET|PUT|DELETE /shop/categories/:id` (body `slug` unique `[a-z0-9_-]`, translatable `title`, `icon`, `sort_order`, `is_active`; delete answers 422 `in_use` while products exist). Products: `GET /shop/products?category_id=&status=all|active|inactive&q=&page=`, `POST /shop/products` (JSON or multipart; optional `image` jpeg/png/webp ≤ 4 MB ≥ 200x200), `GET|PUT|DELETE /shop/products/:id` (`remove_image`), `POST /shop/products/:id/toggle`. Fields: `category_id`, translatable `title`, `price_text`, `badge`, `external_url` (nullable, http/https), `partner_label`, `sort_order`, `is_active`; reads add `image_url` and `clicks_30d`. `GET /shop/options` returns the categories and image limits.

**Internal shop (CB-SHOP-05, package `internal/shop/admin`, same `/shop` section).** Capability `content` for the catalog: `GET /shop/commerce-options` (sellers, category tree with `parent_id`, layette groups, skin types, shipment / return / review statuses, discount kinds, image limits); sellers `GET|POST /shop/sellers`, `GET|PUT|DELETE /shop/sellers/:id` (slug, translatable `name`, `is_verified`, `ship_min_days`/`ship_max_days`, `discreet_packaging`, `shipping_fee`, `free_shipping_over`, `cod_max_total`, `return_days`, `sort_order`, `is_active`; delete → 422 `in_use` while products or orders reference it); `PUT /shop/categories/:id/parent` (`parent_id` a root or null; two levels); `GET|PUT /shop/products/:id/commerce` (`seller_id`, translatable `subtitle`/`description`, `attributes` object of text or translations, `irc_code`, `expires_on` Y-m-d, `skin_types` ⊂ oily/combination/normal/dry/sensitive, `badges` list, `discreet_packaging`, `images` = reorder/remove the stored paths); `POST /shop/products/:id/images` (multipart `image`, ≤ 8); variants `GET|POST /shop/products/:id/variants`, `PUT|DELETE /shop/variants/:id` (`sku` unique, translatable `size`/`color`/`volume`, `price`, `compare_at_price`, `stock`), `POST /shop/variants/:id/stock {stock}`, `GET /shop/stock?q=&low=1&page=`; delivery slots `GET /shop/slots?seller_id=&from=&to=`, `POST /shop/slots/generate {seller_id, from, to, weekdays[0..6]?, windows[{starts_at, ends_at, capacity?}]}` (existing kept), `PUT|DELETE /shop/slots/:id` (capacity ≥ booked; delete → 422 `slot_booked`); reviews `GET /shop/reviews?status=`, `POST /shop/reviews/:id/moderate {status: pending|published|rejected}`, `DELETE /shop/reviews/:id` (rating cache refreshed); discount codes `GET|POST /shop/discount-codes`, `GET|PUT|DELETE /shop/discount-codes/:id` (code upper-cased `[A-Z0-9_-]{3,32}`, `kind` percent ≤ 100 | fixed, caps, window, limits); layette template `GET|POST /shop/layette-items`, `GET|PUT|DELETE /shop/layette-items/:id` (`group_key` clothing/sleep/feeding/bath/outing). Capability **`support`** for the order data (recipient name, phone, address): `GET /shop/shipments?status=&seller_id=&q=<code>&page=` (queue per seller shipment with `allowed_statuses`), `POST /shop/shipments/:id/move {status: confirmed|shipped|delivered, tracking_code?}` (awaiting_seller → confirmed → shipped → delivered; delivered runs `order.Service.MarkShipmentDelivered`; otherwise 422 `invalid_transition`), `GET /shop/orders/:id`, `POST /shop/orders/:id/cancel` (placed, nothing shipped: stock, slots, code use and checklist rows given back), `GET /shop/returns?status=`, `POST /shop/returns/:id/approve {restock?}` / `reject` (422 `not_pending` once decided). Every write is audited with ids and statuses only; no order or address field reaches a log line.

### City services (B-N10-05, D-92)

Capability `content` for `/city-services/*`. Cities and categories: `GET|POST /city-services/cities|categories`, `PUT|DELETE /city-services/cities|categories/:id` (body `slug` unique `[a-z0-9_-]`, translatable `name`, `icon` (categories), `sort_order`, `is_active`; delete answers 422 `in_use` while listings use it). Listings: `GET /city-services/listings?city_id=&category_id=&status=all|active|inactive&q=&page=`, `POST /city-services/listings`, `GET|PUT|DELETE /city-services/listings/:id`, `POST /city-services/listings/:id/toggle`; fields `name`, `category_id`, `city_id`, `district`, `address`, `phone`, `description`, `price_text`, `hours_text`, `booking_mode` (`link` needs an http/https `booking_url`, `phone` needs `phone`), `sort_order`, `is_active`; delete answers 422 `in_use` once requests reference the listing. Photos: `POST /city-services/listings/:id/photos` (multipart `photo`, jpeg/png/webp ≤ 4 MB ≥ 200x200, at most 8) and `DELETE /city-services/listings/:id/photos/:index`. `GET /city-services/options` returns every city and category plus the enums and limits.

Request queue, capability `support` (it holds contact details): `GET /city-service-bookings?status=&listing_id=&page=` (open requests first), `GET /city-service-bookings/:id`, `PUT /city-service-bookings/:id {status, status_note}` where the status machine allows requested → confirmed \| rejected \| cancelled and confirmed → completed \| cancelled (422 `invalid_transition` otherwise); `status_note` is shown to the user.

#### Mother & child directory (CB-DIR-05)

Added to the same city-services admin (no second section). Capability `content` unless noted.

- Listings grow the directory columns in every listing response (`status`, `verified`, `lat`, `lng`, `about`, `age_ranges`, `amenities`, `weekly_hours`, `photo_file_ids`, `file_photo_urls`, `website_url`, `cancel_free_hours`, `cancel_policy`, `rules`, `rating_avg`, `rating_count`); `GET /city-services/listings` takes `listing_status=draft|pending|published|suspended`; `booking_mode` also accepts `online`. `PUT /city-services/listings/:id/directory` writes those fields (catalog codes of `dir_age_ranges` / `dir_amenities` only, http(s) website, lat+lng together); `DELETE /city-services/listings/:id/file-photos/:fileId` drops a structured photo (the file stays with its owner). `GET /city-services/options` adds the directory enums and the age-range / amenity chips.
- Services: `GET|POST /city-services/listings/:id/services`, `PUT|DELETE /city-services/listings/:id/services/:serviceId` (delete 422 `in_use` once a booking names it).
- Slots: `GET /city-services/listings/:id/slots?from=&to=&service_id=`, `POST …/slots {service_id?, starts_at "Y-m-d H:i", ends_at?, capacity}`, `POST …/slots/generate {from, to (≤ 92 days), weekdays [0=Sat..6], times ["H:i"], duration_minutes?, capacity, service_id?}` (same starts skipped, ≤ 500 per run), `PUT …/slots/:slotId {capacity, status open|closed}` (422 `capacity_below_booked` under the booked seats — guarded in the UPDATE itself), `DELETE …/slots/:slotId` (422 `in_use` while booked). A service / slot id of another place is a 404.
- Reviews: `GET /city-services/reviews?status=pending|published|hidden&listing_id=` (pending first; author as a first name), `PUT /city-services/reviews/:id {status published|hidden}` recomputes the place's `rating_avg` / `rating_count` in the same transaction.
- Reports: `GET /city-services/reports?status=open|resolved|dismissed`, `PUT /city-services/reports/:id {status resolved|dismissed}` (open only; the reporter is never returned).
- Applications `/city-service-applications` (capability `content`; `contact_name` / `contact_phone` only when the admin also holds `support`, else null + `contact_hidden: true`): `GET ?status=&q=` (drafts never listed; open work first), `GET /:id` (steps, photos `{id, url, expires_at, pending}` — a fresh 5-minute signed link while pending, the public URL once approved —, which document slots hold a file), `GET /:id/documents/:slot` (a fresh 5-minute signed link per view, audited), `POST /:id/review {status in_review|needs_info|rejected, note}` (note required for needs_info / rejected; rejecting deletes the photos, documents stay), `POST /:id/approve {category_id?, phone?, note?}` → a published, verified listing (or the one it already points at) with the application's place, photos (the private pending photos are promoted to public `place_photo` files in the same transaction; listing and application get the new ids), hours and services (`phone` required in phone mode, never copied from the applicant).
- Booking queue (`support`): every booking adds `code`, `service`, `slot`, `child` (first name + age in months), `price_toman`, status timestamps, `cancelled_by`, `review`. A move to `confirmed` / `rejected` sends the mother a discreet notice (booking id + status only; fake provider logs ids).

## 26. Home lab requests — «آزمایش در منزل» (B-N10-04, D-94)

Capability `support` (view = GET, manage = writes). Under `/api/admin/v1/partners/*`, cookie session + CSRF like every module.

| Method | Path | Notes |
|---|---|---|
| GET | `/partners/lab-services` | `{items[{id, code, kind, title{fa,en}, description, partner_name, partner_code, is_active, sort_order}], kinds, partners{codes}}`: every service, active or not |
| POST | `/partners/lab-services` | `{code (a-z0-9_, unique), kind home_draw\|checkup, title{fa, en?}, description?{fa,en}, partner_name, partner_code? (default `fake`, must be a registered adapter), is_active?, sort_order?}` → 201 |
| PUT | `/partners/lab-services/:id` | same body, full replace; `is_active:false` hides the Today cards |
| GET | `/partners/lab-requests?status=&q=<reference>&page=` | paged queue, newest first: reference, masked account mobile, status, service, partner, preferred day + window, consent_import, imported, purged, `manual_status`. Never the name, address, note or a result |
| POST | `/partners/lab-requests/:id/status` | `{status: accepted\|sampled\|result_ready\|rejected}` — the console plays the fake partner; 409 `lab_bad_transition` for a move the machine does not allow, 409 `lab_manual_status_unsupported` for real partners (they update their own statuses) |

Every write is audited (`partner_lab_service.create|update`, `partner_lab_request.status`).

## 27. Insurance catalog — «بیمه: بیمه‌گرها و مراکز طرف قرارداد» (CB-INS-01, D-INS-01)

Capability `content`. Admin data only: no user policy, claim or questionnaire is reachable here.

| Method | Path | Notes |
|---|---|---|
| GET | `/insurance/insurers` | `{items[{id, code, name, website, phone, is_active, sort_order, updated_at}]}` |
| POST | `/insurance/insurers` | `{code (a-z0-9_, unique), name, website? (http/https), phone?, is_active?, sort_order?}` → 201 `{item}` |
| PUT | `/insurance/insurers/:id` | same body, full replace |
| DELETE | `/insurance/insurers/:id` | policies / centres keep their rows with the link emptied (deactivate to keep the name) |
| GET | `/insurance/centres` | `{items[{id, insurer_id, insurer_name, name, type, services, city, address, phone, lat, lng, is_direct, is_active, sort_order, updated_at}], types}` |
| POST | `/insurance/centres` | `{name, type gyn\|lab\|imaging\|hospital\|pharmacy\|pediatric\|other, insurer_id? (null = every insurer), services?, city?, address?, phone?, lat?+lng?, is_direct?, is_active?, sort_order?}` → 201 `{item}` |
| PUT | `/insurance/centres/:id` | same body, full replace |
| DELETE | `/insurance/centres/:id` | claims keep the centre name |

Every write is audited (`insurance_insurer.save|delete`, `insurance_centre.save|delete`).

## 28. App modules — «ماژول‌های اپ» (kill-switches)

Table `app_modules` (goose 00094 + Laravel twin), one row per key: `companion`, `ttc`, `ivf`, `menopause`, `teen`,
`postpartum`, `children`, `contraception`, `programs`, `mother_child`, `learning`, `my_courses`, `shop`, `insurance`,
`record`, `devices`, `doctors`, `todo`, `reminders`, `home_articles` (all seeded on; a missing row = on). `my_courses`
and `devices` are UI-only (a «من» row each); `reminders` (the «یادآورها» hub + the «یادآورهای امروز» card) and
`home_articles` (home's «بر اساس سیکل فعلی‌ات» strip) are UI-only too — `/care` stays open because pregnancy visits,
IVF, checkups and the log sheet book through it. `doctors` gates `/telemed`, `todo` gates `/todo`. The
gate keeps `/health-record/emergency-card*` open with `record` off (safety). Capability `config` (view / manage).
Package `internal/modules`; the switches are cached 60 s in Redis and cleared on every write.

| Method | Path | Body / response |
| --- | --- | --- |
| GET | `/modules` | → `{modules: {key: bool}, keys: [...]}` |
| PUT | `/modules` | partial `{companion?: bool, …}` → same shape; each change audited (`modules.update`, attr `module`, `enabled`) |

Effects of an off module:

- **Public** `GET /api/v1/modules` → `{key: bool}` (no auth; the onboarding reads it).
- **Gate** (`modules.Gate`, installed in `cmd/api` before every domain): the module's `/api/v1` routes answer
  403 `{error_code: "module_disabled", module}` — never 401.
- **Onboarding / life stage:** `gender=male` (companion off), `goal` / `mode` ∈ ttc|menopause|teen|postpartum and
  `ivf_iui=true` / `track_contraception=true` are 422. With companion off, `POST /onboarding/complete` stores a missing
  gender as `female` (the frontend skips the gender screen). `GET /profile/life-stage` reports an off stored mode as
  `cycle` and an off IVF / contraception switch as `false`; the stored values stay and come back when the module is on.

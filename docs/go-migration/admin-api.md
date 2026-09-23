# Admin API (`/api/admin/v1`)

The Blade admin panel (`backend/routes/admin.php`) is replaced by a JSON API in `backend-go` and a separate
Next.js app, `admin-web/` (T-M2-22/23). This page is the contract between them. T-M2-20 built the core (auth,
dashboard, users, admins); T-M2-21 adds content CRUD, uploads and languages on the same conventions.

Code: `backend-go/internal/admin/{httpadmin,auth,dashboard,users,admins}`, routes in
`internal/http/routes_admin_core.go`, queries in `db/queries/admin/`.

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

## 7. Uploads (conventions for T-M2-21)

- `multipart/form-data`, one file per field, field name = the column concept (`image`, `cover`), other fields as
  form values (translatable fields as `title[fa]`, `title[en]`; `validation.Input` parses the PHP bracket syntax).
- Mutating upload requests carry `X-CSRF-Token` like any other (the admin chain does not require JSON).
- Limits as Laravel: banners jpeg/png/webp ≤ 4 MB and ≥ 800×400; article covers go through the image optimiser
  (fit 1080×1080, WebP q82, random 40-char name). nginx allows 25 MB.
- Files land on the `backend-storage` volume under `app/public/<dir>/`; the DB stores the relative `image_path`,
  responses return both `image_path` and the absolute `image_url` (`APP_URL/storage/<path>`).
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

## 10. Infra follow-ups (not in T-M2-20's files)

- `backend-go/cmd/api/cors.go`: the global Laravel-compatible CORS middleware handles every `/api/*` path,
  answers preflights before routing and overwrites `Access-Control-Allow-Origin` with the public origin. It must
  skip `/api/admin/` so the admin CORS (credentials, `ADMIN_WEB_ORIGINS`) applies. Only needed for cross-origin
  dev; same-origin production works without it.
- nginx: `deploy/vhost-admin.inc` currently 404s `/api/` on the admin host — it must proxy `/api/admin/` to the
  Go backend (and keep 404 for the rest of `/api/`); `vhost-api.inc` must 404 `/api/admin/`. Update the
  `deploy.sh` / `deploy-stage.sh` host assertions accordingly.
- Compose/env: set `ADMIN_HOSTS` (prod `adpanell.ritme.app`, stage its host) for the Go service.

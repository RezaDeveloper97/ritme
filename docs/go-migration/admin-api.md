# Admin API (`/api/admin/v1`)

The Blade admin panel (`backend/routes/admin.php`) is replaced by a JSON API in `backend-go` and a separate
Next.js app, `admin-web/` (T-M2-22/23). This page is the contract between them. T-M2-20 built the core (auth,
dashboard, users, admins); T-M2-21 adds content CRUD, uploads and languages on the same conventions.

Code: `backend-go/internal/admin/{httpadmin,auth,dashboard,users,admins}`, routes in
`internal/http/routes_admin_core.go`, queries in `db/queries/admin/`. T-M2-21:
`internal/admin/{content,content/form,media,messages,languages}`, routes in `internal/http/routes_admin_content.go`,
queries in `db/queries/admin/content.sql` (endpoints in §11). T-M4-03: `internal/admin/checkups` (checkup-type
catalog, §12).

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
| 422 | `in_use` | deleting a checkup type that users have records of (T-M4-03, §12); body has `records_count` |
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
| GET | `/recommendations` | A | `phase=general\|<phase>`, `type` | list + `filters` (phase-less last, then phase, sort_order, id) |
| GET | `/recommendations/options` | A | — | `{phases, subphases, subphase_phases:{sub:phase}, types, triggers}` |
| POST · PUT | `/recommendations` · `/:id` | A | `type` req, `title` T (≤255, all empty → null), `text` T req (≤2000), `cycle_phase?`, `cycle_subphases[]` (only those the phase reaches), `symptom_trigger?`, `sort_order?`, `is_active?` | `{recommendation}` |
| GET · DELETE · POST | `/recommendations/:id` · `/:id` · `/:id/toggle` | A | — | `{recommendation}` · `{id}` · `{recommendation}` |
| GET | `/banners` · `/banners/options` | A | — | list of `Banner` (position, sort_order, newest) · `{positions, link_types, image:{max_kb,min_width,min_height,recommended_width,recommended_height,types}}` |
| POST | `/banners` | A | multipart: `image` **req** (jpeg/png/webp ≤ 4 MB, ≥ 800×400, stored as uploaded in `banners/`), `title` T (≤255), `position` req, `link_type?`, `link_url?` (required with `link_type`; `url` when external, a single leading `/` path when internal — D-16; empty drops both), `starts_at?`, `ends_at?` (≥ starts_at), `sort_order?`, `is_active?` | `{banner}` (`image_path`, `image_url`) |
| PUT/POST | `/banners/:id` | A | as POST, `image` optional (replaces + deletes the old file) | `{banner}` |
| GET · DELETE · POST | `/banners/:id` · `/:id` · `/:id/toggle` | A | — | `{banner}` · `{id}` (file deleted) · `{banner}` |
| GET | `/task-templates` · `/task-templates/options` | A | — | list · `{phases, categories}` |
| POST · PUT | `/task-templates` · `/:id` | A | `key` req unique, `title` T req, `description` T, `category` req, `icon?`, `cycle_phase?`, `sort_order?`, `is_active?` | `{task_template}` |
| GET · DELETE · POST | `/task-templates/:id` · `/:id` · `/:id/toggle` | A | — | `{task_template}` · `{id}` · `{task_template}` |
| GET | `/info-sections` | A | `group=help\|privacy\|terms\|about` (unknown → help) | list + `filters{group}` (sort_order, id) |
| GET | `/info-sections/options` | A | `group` | `{groups, group, next_sort_order}` |
| POST · PUT | `/info-sections` · `/:id` | A | `group` req, `heading` T req (≤200), `body` T req, `link_label` T (≤60, empty → null), `link_url?` (≤500, `https?://`, `mailto:` or `tel:`), `sort_order?`, `is_active?` | `{info_section}` |
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
| GET | `/checkup-types/options` | A | — | `{categories[], performed_by[], icons[], tones[], default_tone, default_remind_lead_days, max_steps, max_cycle_day, max_interval_months, next_sort_order}` (plain value lists; admin-web labels them) |
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
| `sort_order` | optional integer; create default = last + 1 |
| `source_note` | optional string ≤ 1000 (medical-review note) |

On update an optional field that is **absent keeps its stored value**; `null` (or `[]` for a list) clears it. The
range checks compare against the stored value of an absent field. An empty list is stored as SQL `NULL` and always
answered as `[]`.

`CheckupType` = the `checkup_types` columns (`id, key, category, title, subtitle, why, performed_by, icon, tone,
interval_months, interval_months_max, age_min, age_max, cycle_day_from, cycle_day_to, remind_lead_days, prep_steps,
guide_steps, finding_options, hide_in_pregnancy, is_active, sort_order, source_note`) + `records_count` (records of
all users) + `created_at`, `updated_at` (ISO 8601).

Stats: `records_last_30_days` counts records with `done_on ≥ today − 30`. `overdue_users` counts users whose
**latest** record of the type is past its due-by date — `next_due_on` (user override) ?? `done_on +
(interval_months_max ?? interval_months)` months — before today, leaving out users who switched the type off. It is
the calendar approximation of the engine's `overdue` (cycle windows, age and pregnancy are not evaluated per user);
users who never recorded the type are not counted.

Audit lines: `checkup_type.create|update|delete|reorder` (target `checkup_type`).

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
| GET | `/plus/settings` | A | — | `{settings: {trial_offer_percent, vat_rate_bps (effective), vat_override_bps\|null, vat_env_rate_bps, vat_source: env\|admin, trial_days}}` |
| PUT | `/plus/settings` | S | `trial_offer_percent` (0–100, 0 = no offer), `vat_rate_bps?` (0–10000; **null** removes the override → `PLUS_VAT_RATE_BPS`; absent = unchanged) | same as GET; a ledger row only when something changed |
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
| GET | `/companions/links` | A | `status=all\|invited\|active\|revoked`, `type=all\|partner\|spouse` (unknown → `all`), `page`, `per_page` | list, newest first (`created_at DESC, id DESC`), items `{id, type, status, label, owner{id, name, mobile}, companion{id, name, mobile}\|null, invite{phone, expires_at, expired}\|null (status invited only), grants_count, invited_at, accepted_at, revoked_at, revoked_by, created_at}` + `filters{status, type}` + `counts{by_status{all, invited, active, revoked}, by_type{partner{…, all}, spouse{…, all}}}` (over every link, not the filter) + `statuses[]` + `types[]` |

Masking: names and the owner's label keep the first letter (`س•••`), mobiles and the invite phone keep 4 + 4 digits
(`0912•••4567`). Never sent: invite codes or their hash, attempts, which sections are shared (only `grants_count`),
anything of the owner's health data. The full mobile stays on `/users/:id`.

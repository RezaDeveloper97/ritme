---
name: new-endpoint
description: Scaffold a new Ritme backend API endpoint in backend-go (Go + Fiber v3) following project conventions — sqlc query, store, service, handler in routes_<domain>.go, Laravel-compatible validation, httpx envelope, contract case, OpenAPI entry, multi-language content. Use when adding or changing any /api/v1 endpoint.
---

# New backend API endpoint (Go)

The API lives in `backend-go/` (renamed to `backend/` at T-M2-27). Read `backend-go/CLAUDE.md` first — its rules
(routing, sqlc only, PHP-compat helpers, 401 contract, multi-language) are mandatory. Copy the shape of a recent
domain before writing code: `internal/reminder` (small CRUD), `internal/pregnancy` (service + calc),
`internal/home` (composed read model).

## Steps

1. **Query** — add SQL to `db/queries/<domain>/*.sql` (sqlc annotations `-- name: X :one|:many|:exec|:execlastid`),
   then `make sqlc`. Generated code lands in `internal/<domain>/store` (package `store`; never edit by hand).
   Always scope user data by `user_id` in the query itself (IDOR is the top risk for health data).
   A new domain needs its package in `sqlc.yaml` (copy an entry; only `queries`/`out` change).
   Schema changes need a Laravel migration *and* a goose migration until T-M2-27 (see `docs/go-migration/migrations.md`).
   sqlc/MySQL: avoid `NOT IN sqlc.arg` and `BETWEEN` (broken codegen).

2. **Service** — business logic in `internal/<domain>/` as plain Go taking `store.Querier` / `*sql.DB`, a
   `clock.Clock` and explicit deps. No Fiber types, no SQL strings. Dates via `platform/civildate` (Tehran,
   Saturday week start); PHP rounding/number output via `platform/phpround`; `decimal` columns stay strings.

3. **Handler** — `internal/<domain>/handlers.go`, methods on a `Handlers` struct built by `NewHandlers(...)`.
   Current user: `auth.CurrentUserID(c)` / `auth.CurrentUser(c)`. Locale: `i18n.Locale(c)` (never default to a
   hard-coded language). Return errors; `httpx.ErrorHandler` renders them (`httpx.NotFound()`,
   `httpx.ModelNotFound(...)`, validation errors, wrapped `%w` errors → 500).

4. **Route** — register in `internal/http/routes_<domain>.go` from `init()` with
   `Register("<domain>", func(r fiber.Router, d *Deps) { … })`. One file per domain; never edit `router.go` or
   another domain's file. Attach middleware per route: `locale` (`i18n.Middleware(...)`), then
   `auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser` unless intentionally public, then throttles
   (`ratelimit`). Static paths before param paths (`/x/enums` before `/x/:id`). Don't register HEAD/OPTIONS.
   Public routes live under `/api/v1/*`; admin routes under `/api/admin/v1/*` via `httpadmin.Handle`
   (documented in `docs/go-migration/admin-api.md`, not in the OpenAPI file).

5. **Validation** — `platform/validation` reproduces Laravel rules and messages byte for byte:
   `validation.Make(lang.Default(), i18n.Locale(c), validation.Input(c), rules, validation.Now(clock…))` with
   `validation.Rules{validation.F("field", "required", "integer", validation.In(...))}`. Framework-style 422
   (`{message, errors}`) → return `v.Errors()`; controller-style 422 (`{success:false, message, errors}`) → build it
   like `internal/reminder`. Translatable inputs: `i18n.TranslatableRules(...)` (only the default language is
   required). Never trust unknown keys — pick validated fields only.

6. **Response** — `httpx.OK(c, data[, msg])` → `{"success":true[,"message"],"data":…}`; `httpx.Created` for 201
   (use 201 only where the contract says so); `jsonx.Obj(k, v, …)` / `jsonx.OrderedMap` to keep Laravel's key order;
   `httpx.NewPage` for paginators. Dates: `Y-m-d` for date casts, `…T..:..:...000000Z` for Eloquent datetimes,
   `+03:30` ISO for `toIso8601String()`. Empty maps encode as `[]` where Laravel did.

7. **Tests** — unit tests next to the code (`testify`); integration tests use `platform/db/testdb.New(t)` and
   `t.Skip()` without `TEST_DB_DSN` (`make test-int PKG=./internal/<domain>/...`). Cover the happy path, a
   validation failure, the 401 body, and **user A cannot read or change user B's rows**.

8. **Contract** — add a case to `contract/cases/<group>.yaml` (format: `cmd/contract/cases.go`; personas:
   `docs/go-migration/contract.md`). While Laravel exists, record its golden with
   `make contract-record ROUTES=<group>` and diff Go with `make contract ROUTES=<group>` (must be green). For an
   endpoint that only exists in Go (and after T-M2-27), record the golden from Go and review it by hand.
   An intentional difference needs a `D-nn` in `docs/go-migration/deviations.md` (user OK) and an allow-list entry.

9. **OpenAPI** — add the operation to `backend-go/api/openapi.yaml` (OpenAPI 3.1, hand-maintained, served at
   `/docs` behind `SWAGGER_USER`/`SWAGGER_PASSWORD`): tag, summary, unique `operationId`, `security:
   [{bearerAuth: []}]` (or `[]` when public), every path parameter, `AcceptLanguage` parameter, `requestBody`,
   and one response per status with the real schema. Reuse `components`: `Date`, `LaravelDateTime`,
   `IsoDateTime`, `DecimalString`, `EmptyPhpArray`, `RawTranslations`, and responses `Unauthenticated`,
   `ValidationFailed`, `ModelNotFound`, `TooManyAttempts`, `ServerError`. Then run
   `cd backend-go && go test ./internal/http/... -run OpenAPI` — it fails when a registered route has no
   operation (or vice versa), when a golden body does not validate against its status's schema, or when a
   status seen in a golden is undocumented.

10. **Multi-language content** — never hard-code user-facing strings or a language list in Go:
    - UI strings for the clients: `resources/translations/<code>/<group>.json` (served by
      `GET /api/v1/languages/{code}/messages`; admins edit copies on the storage volume).
    - Server-side messages (validation, controller messages): `resources/lang/<code>/*.json` via
      `lang.Default().Trans(key, params, locale)`.
    - Smart-message / BMI copy: `message_contents` rows through `messages/content.Repository`
      (code defaults in `defaults.json` only as the fallback).
    - Translatable DB columns are JSON keyed by language code: read with `i18n.Pick` / `i18n.PickString`
      (fallback to the default language), validate with `i18n.TranslatableRules`.

11. **Verify** — `go vet ./... && go test ./... && golangci-lint run` in `backend-go/` (or `/verify-all`).
    If the frontend needs the endpoint, add the API function + types in the matching
    `frontend/src/entities/*/api` slice (see the `new-fsd-slice` skill).

## Until T-M2-27 (Laravel still serves traffic)

The strangler proxy (`deploy/go-routes.inc`) decides per path group which stack answers. A new endpoint in a group
that is still on Laravel must also exist in `backend/` (route in `routes/api.php`, FormRequest, thin controller,
repository, feature test with an IDOR case, `vendor/bin/pint`, `php artisan test --filter=…`) — or move the group to
Go first. Laravel-only gotchas: SQLite date casts break `updateOrCreate`; `Passport::actingAs` reuses one User
instance (`$user->refresh()`).

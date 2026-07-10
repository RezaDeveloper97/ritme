---
name: new-endpoint
description: Scaffold a new Ritme backend API endpoint following project conventions (FormRequest, Resource, repository, feature test, bilingual content). Use when adding any /api/v1 endpoint.
---

# New backend API endpoint

Follow the existing pattern end-to-end — look at a recent similar endpoint (e.g. pregnancy or banners) as the reference before writing code.

## Steps
1. **Route** in `routes/api.php` under the `/v1` group with `auth:api` middleware (unless intentionally public like `GET /banners`).
2. **FormRequest** for validation — never validate inline in the controller, never pass `$request->all()` to models.
3. **Controller** stays thin: validate → call service/repository → return Resource. Reuse existing repositories (e.g. MessageContentRepository pattern) or create one alongside.
4. **API Resource** for the response shape — match the JSON conventions of existing v1 endpoints (check an existing Resource for envelope/casing style).
5. **Migration/Model** if needed: `$fillable` explicit, indexes on queried columns, reversible `down()`. Bilingual content goes in content tables with fa/en columns following the existing pattern — not hardcoded strings.
6. **Feature test** in `tests/Feature/`: happy path + validation failure + **authorization test proving user A cannot access user B's data** (health data — IDOR is the top risk).

## Project gotchas (will bite you)
- SQLite: date-cast attributes break `updateOrCreate` matching — match on raw string columns or query-then-save.
- `Passport::actingAs` reuses one User instance across requests in a test — call `$user->refresh()` before asserting on relations.
- Run `vendor/bin/pint` and `php artisan test --filter=YourTest` before finishing.

## After
If the frontend needs this endpoint, add the API function + types in the matching `frontend/src/entities/*/api` slice (see the new-fsd-slice skill).

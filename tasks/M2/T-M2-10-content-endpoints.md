---
id: T-M2-10
title: Content endpoints — languages, info, banners, articles, phase content, /storage
milestone: M2
type: backend
status: todo
depends_on: [T-M2-05, T-M2-06, T-M2-07, T-M2-08]
parallel_group: M2-E
touches: [backend-go/internal/content, backend-go/internal/http/routes_content.go, backend-go/db/queries/content, backend-go/contract/allowlist/content.yaml, backend-go/contract/allowlist/public.yaml]
skills: []
verify: cd backend-go && go test ./internal/content/... && make contract ROUTES=public,content
---

# T-M2-10 — Content endpoints

## Why
The content group is read-mostly and low-risk, so it moves to Go first in the strangler order. It also ports
HtmlSanitizer, which T-M2-21 (admin articles) reuses. Read api-inventory §1.3, §1.4, §1.7 (phase-content), §1.11
and domain-inventory §2 (scopes/accessors).

## Scope
1. Public: `GET /languages`, `GET /languages/{code}/messages` (via i18n Registry/TranslationStore), `GET /info/{group}`
   (help/privacy/terms/about else pretty `abort(404)` body; fa/en clamp), `GET /privacy` alias.
2. Auth: `GET /banners` (`Banner::active` window + order `sort_order, id DESC`; `?position` filter; grouped
   `positions` object incl. empty positions exactly as Laravel), `GET /articles` (validation 422 framework family,
   search on `title->locale`/`excerpt->locale` + fa, `forPhase` ordering, `meta` paginator, categories list),
   `GET /articles/{slug}` (404 controller family, sanitized body, ≤4 related).
3. `GET /cycle/phase-content/{phase}` (invalid subphase 422, missing 404, `locale→fa→en` pick, `sections` `[]` when empty).
4. `content/sanitizer`: port `app/Services/Content/HtmlSanitizer.php` (DOM allow-list) with bluemonday + custom
   policy; golden tests on every seeded article body + adversarial inputs (script, on* attrs, javascript: URLs,
   nested lists, iframes) — output must equal the PHP output.
5. `image_url` building: `APP_URL + /storage/<image_path>` (image_path overrides image_url for articles).
6. `GET /storage/*`: serve `STORAGE_PATH/app/public` (safe path join, correct content types, cache headers like
   Apache's defaults; no directory listing).

## Out of scope
Admin CRUD for these tables (T-M2-21). Home "articles" section (T-M2-18).

## Acceptance
- `make contract ROUTES=public,content` green for all personas and locales (fa, en, ar, none).
- Sanitizer golden tests green; allow-list file empty or every entry linked to `deviations.md`.
- `/storage/..%2f..` traversal attempts return 404.

## Note from T-M2-06
The Go language registry is cached in Redis under `ritme-go:languages.registry` with no TTL. During the strangler
period a language edited in the Laravel admin will not reach Go until that key is flushed — give it a short TTL
(or flush on read-miss) here, and make T-M2-21's admin writes call `Flush`.

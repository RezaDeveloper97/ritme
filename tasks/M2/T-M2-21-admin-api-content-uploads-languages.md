---
id: T-M2-21
title: Admin API II — content CRUD, uploads, messages, languages and translations
milestone: M2
type: backend
status: todo
depends_on: [T-M2-10, T-M2-20]
parallel_group: M2-F
touches: [backend-go/internal/admin/content, backend-go/internal/admin/media, backend-go/internal/admin/languages, backend-go/internal/admin/messages, backend-go/internal/http/routes_admin_content.go, backend-go/db/queries/admin/content.sql, docs/go-migration/admin-api.md]
skills: [security-review]
verify: cd backend-go && go test ./internal/admin/... && make test-int PKG=./internal/admin/...
---

# T-M2-21 — Admin API II: content CRUD, uploads, messages, languages and translations

## Why
All editorial content (articles, banners, challenges, recommendations, pregnancy weeks, phase content, smart messages,
translations) is managed from the admin. Read infra inventory §3 (pages, uploads) and §7, and each
`app/Http/Controllers/Admin/*Controller.php` for validation rules and side effects.

## Scope
1. CRUD (+ toggle where Blade has it) with Laravel's validation rules and the "only the default language is
   required" translatable rule (active languages from the i18n registry): articles (HTML body sanitized with the
   T-M2-10 sanitizer on save, like today; cover image), affirmations, challenges (+ completions report), recommendations
   (subphase picker data), banners (image upload: jpeg/png/webp, ≤4 MB, ≥800×400, stored as today under
   `app/public/banners`), task templates, info sections, pregnancy weeks, phase contents.
2. `admin/media`: port `ImageOptimizer` (fit 1080×1080, WebP q82, 30 MP cap, EXIF orientation, random 40-char name) —
   choose the WebP encoder (cgo libwebp via `chai2010/webp`, or `govips`) and record the choice; files under
   `STORAGE_PATH/app/public/articles`, DB keeps relative `image_path`.
3. Smart messages (`message_contents`): list/filter by group+locale, edit payload, approve, toggle.
4. Languages (super only): CRUD, toggle, set default, regenerate; translation editor (per namespace JSON under
   `STORAGE_PATH/app/translations/<code>/`); provisioner (create bundle files, clone `message_contents` rows with
   `is_approved=false`, store the `lang/` equivalents as JSON on the volume — D-04). Flush the Go registry cache
   **and** the Laravel `languages.registry` cache key while both stacks run (document how).
5. Extend `docs/go-migration/admin-api.md` with every endpoint.

## Out of scope
UI (T-M2-23).

## Acceptance
- Integration tests per resource (create/update/toggle/delete, validation 422s, super-only guards).
- Upload tests: oversize, wrong mime, too-small dimensions rejected; valid JPEG → WebP ≤1080px written to the volume
  and served by `/storage/*`.
- Provisioning a new language makes `GET /api/v1/languages` (Go **and** Laravel) list it after the cache flush.

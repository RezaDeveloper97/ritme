# Seo

Decides every SEO head tag: `SeoManager` (request-scoped) layers settings defaults → `seo_meta` (static page by route
name, or a model via `Concerns\HasSeo`) → controller overrides, plus forced noindex (non-production, search / cart /
checkout / done routes, filtered listings). `<x-seo.head/>` renders the result; `php artisan seo:audit` checks
rendered pages (`Audit/`). JSON-LD: request-scoped `Schema\SchemaGraph` (pages add nodes), node builders in
`Schema/Nodes`, rendered once by `Schema\PageGraph` in the head — see `docs/SEO.md`.

Static-pages SEO manager (L7-01): `Actions/ListStaticPageSeo` (registry pages + effective title/description and
checks), `SaveStaticPageSeo` / `ResetStaticPageSeo` (the page's `seo_meta` row by route name),
`StaticPages/StaticPageSeoDefaults` (each page controller's lang default; pinned by `StaticPageSeoTest`). A new
static page controller must keep its default copy in sync there.

Folders: `Models/`, `Data/`, `Contracts/`, `Repositories/` (Eloquent + Cached, `seo` namespace), `Observers/`,
`Support/` (Robots, CanonicalUrl, DescriptionText), `Audit/`. Bindings live in `App\Providers\Domain\SeoServiceProvider`.
See `docs/ARCHITECTURE.md`.
- `Analysis/` (L7-02): pure Persian-aware `SeoAnalyzer` (25 checks, score 0–100, red-line words + diagnosis claims as errors), `PersianText`, `ContentDocument`, `RedLines`, `TextWidth`; the live panel is part of `SeoFields`. Only `Queries/FindDuplicateSeoMeta` does I/O.
- `Indexing/` (L7-04): settings-backed robots rules + validator, per-type robots defaults, sitemap settings, IndexNow (queued, production-only), verification metas, safe head code (same-origin meta/link only). See `Indexing/README.md`.
- `Redirects/` (L7-03): admin redirect manager (cached map, chain collapse, regex, 410), 404 monitor (buffered aggregate counts, no IP), auto-redirects for taxonomy/city slug changes. See `Redirects/README.md`.

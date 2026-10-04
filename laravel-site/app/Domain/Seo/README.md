# Seo

Decides every SEO head tag: `SeoManager` (request-scoped) layers settings defaults → `seo_meta` (static page by route
name, or a model via `Concerns\HasSeo`) → controller overrides, plus forced noindex (non-production, search / cart /
checkout / done routes, filtered listings). `<x-seo.head/>` renders the result; `php artisan seo:audit` checks
rendered pages (`Audit/`). JSON-LD builders (L1-04) live in `Schema/`.

Folders: `Models/`, `Data/`, `Contracts/`, `Repositories/` (Eloquent + Cached, `seo` namespace), `Observers/`,
`Support/` (Robots, CanonicalUrl, DescriptionText), `Audit/`. Bindings live in `App\Providers\Domain\SeoServiceProvider`.
See `docs/ARCHITECTURE.md`.

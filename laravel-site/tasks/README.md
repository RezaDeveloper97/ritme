# laravel-site/tasks — Ritme marketing site → Laravel queue

Converts the static HTML export in `laravel-site/*.html` (moved to `design/html/` by L0-01) into a full Laravel
application: Blade frontend, DDD-style domains, a Filament admin with a strong SEO suite, image pipeline, PWA, and a
cPanel-friendly deployment. Driven by the `/site-task` skill (`../.claude/skills/site-task/SKILL.md`). Independent of
`tasks/`, `bloom/`, `roadmap/` at the repo root — those queues know nothing about this one.

All commands run from `laravel-site/`.

## Decisions (user, 2026-10-04) — binding

| Topic | Decision |
|---|---|
| Stack | **Laravel 12**, PHP ≥ 8.2 (cPanel-safe; dev runs 8.4). MySQL/MariaDB in production, SQLite for dev + tests. |
| Frontend | **Blade** server-rendered. **Tailwind CSS v4** compiled by **Vite** (purged, hashed filenames + manifest → no stale cache). Tiny vanilla ES modules, no jQuery/Alpine on the public site. |
| No externals | **Zero CDN / third-party requests** on public pages: fonts self-hosted, icons as a local SVG sprite, no external maps, embeds, analytics or captcha. Composer/npm packages are fine (they are bundled at build time). |
| Architecture | DDD-lite: `app/Domain/<Context>` (Models, Actions, Data, Contracts, Repositories, Events, Enums), `app/Support` shared kernel, `app/Http` + `app/Filament` are thin delivery layers. SOLID, KISS, YAGNI, DRY; enforced by Pest arch tests. |
| Caching | **Cache-aside everywhere reads are hot** via `App\Support\Cache\CacheAside` + cached repository decorators, invalidated by **versioned namespaces** (works on `file`/`database` drivers — no tags needed on cPanel). Guest full-page cache on top. |
| SEO | Top priority. Every public page: unique title/description, canonical, robots, OG + Twitter, JSON-LD graph, breadcrumbs, single h1, alt + width/height on every image, sitemap entry. Admin-editable everywhere. `seo:audit` must stay green. |
| Performance | GTmetrix/Lighthouse targets on every route: Performance ≥ 95 (mobile), SEO 100, Best Practices 100, A11y ≥ 95, CLS < 0.1, LCP < 2.5 s. Render-blocking work minimised (critical CSS, preloaded fonts, deferred JS). |
| Images | Every upload is optimised: auto-orient, strip metadata, **mobile (480/768) + desktop (1280/1920)** widths, AVIF (when GD supports it) + WebP + fallback, OG 1200×630, thumb; served through `<x-picture>` with `srcset`/`sizes`, optional separate mobile art-direction image. |
| Admin | **Filament v4** at `/admin` (path configurable), Persian + RTL, local Vazirmatn font (no Bunny/Google fonts), roles via spatie/laravel-permission, activity log. |
| PWA | Manifest + icons + build-generated service worker (never hand-edited), offline page, **two-tier update** (soft toast / forced), install prompt. Follows the `pwa` skill. |
| Hosting | **cPanel-compatible**: no symlink requirement (public disk writes into `public/media`), no Node on server (build shipped in the package), cron `schedule:run` drives the database queue, works with `public_html` as document root. |
| Language | Persian only (`fa`, RTL, `fa_IR`). UI strings in `lang/fa`; no hreflang. Page URLs are English (`/cycle`, `/blog/...`); content slugs admin-editable (Persian allowed). Old `*.html` and WordPress-style `/cycle/` URLs 301 to the new ones. |
| Shop payments | Cash on delivery only, behind a `PaymentGateway` contract (other gateways later). |
| Maps | No embedded third-party maps; address + "open in maps" deep links (`geo:`/Neshan/Balad URLs) and a local static illustration. |
| Git | Commit on the current `stage` branch of the monorepo, **only paths under `laravel-site/`**, one commit per task. Never push or deploy without the user's OK. |
| Content red lines | From the design brief: no diagnosis claims, no «حتماً/قطعاً/دقیق‌ترین/تضمینی», no countdown/sales pressure. Brand: ریتمی / Ritme. |

## Commands

| Command | Purpose |
|---|---|
| `tasks/bin/next.sh` | runnable tasks now |
| `tasks/bin/next.sh --all` / `--summary` | every task + status / done per milestone |
| `tasks/bin/next.sh --status L0-01 in_progress` | change status (appends to `tasks/LOG.md`) |
| `tasks/bin/next.sh --conflicts L3-04 L3-05` | parallel safety (touches overlap → exit 1) |
| `tasks/bin/next.sh --check` | validate ids, deps, statuses |
| `tasks/bin/new.sh ID "Title" TYPE "deps" GROUP "touches" "skills" "verify" [< body.md]` | add a task |
| `tasks/bin/index.sh` | regenerate `tasks/INDEX.md` |
| `composer verify` *(after L0-01)* | pint --test + phpstan + pest + `npm run build` |
| `node tools/shot.mjs` *(after L0-08)* | design-HTML vs Laravel screenshots at 390/1440 + diff |
| `php artisan seo:audit` *(after L7-05; basic version from L1-03)* | crawl every indexable route and report SEO issues |

Task ids: `L<milestone>-<nn>` (follow-ups get a letter suffix, `L3-04b`). Frontmatter: `id, title, milestone, type,
status, depends_on, parallel_group, touches, skills, verify`. Body: Why / Scope / Out of scope / Acceptance.
`touches` paths are relative to `laravel-site/`.

## Milestones

| | Theme |
|---|---|
| L0 | Foundation — scaffold, architecture, cache-aside kernel, HTML audit, Tailwind/Vite, style converter, icons, screenshot tool |
| L1 | Core — settings, layout, SEO head, JSON-LD, routing/redirects, sitemaps/robots, performance middleware + `.htaccess`, Filament base |
| L2 | Media — upload/optimise pipeline, `<x-picture>`, media library |
| L3 | Marketing pages — every static page converted to Blade, FAQ + contact domains, fidelity sweep |
| L4 | Magazine (blog) — domain, list/category/tag, article, feed + search, admin |
| L5 | Mother & child directory — domain, listing, place page, booking, business join flow, admin |
| L6 | Shop — catalog, listing, product, cart, COD checkout + orders, admin |
| L7 | Admin SEO suite — page SEO manager, content analyser, redirects/404, indexing controls, audit dashboard, bulk editor |
| L8 | PWA — manifest/icons, service worker + two-tier update, verification |
| L9 | Quality — critical CSS/asset budget, Lighthouse sweep, a11y, security, tests |
| L10 | cPanel packaging + go-live checklist |

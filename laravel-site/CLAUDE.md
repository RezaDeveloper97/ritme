# laravel-site — Ritme marketing site (Laravel 12)

Work is driven by the `/site-task` skill and the queue in `tasks/`. **`tasks/README.md` decisions are binding.**

## Layout
- `design/html/` — the original static export (29 pages + `assets/`). Fidelity source for markup and copy. Never
  edit it; never serve it.
- `app/Domain/<Context>` — domain code (Models, Actions, Data, Contracts, Repositories, Events, Enums, Observers).
- `app/Support` — shared kernel (cache-aside, Jalali, Money, HTML helpers).
- `app/Http`, `app/Filament` — thin delivery layers. See `docs/ARCHITECTURE.md` (from L0-02).

## Rules
- PHP: `declare(strict_types=1)`, final classes for actions/controllers, no `env()` outside `config/`, no queries in
  controllers/Blade (repositories / query objects; DTOs into views), PHP 8.2-compatible syntax (cPanel).
- Hot reads use cache-aside (`App\Support\Cache\CacheAside` + cached repository decorators); observers bump the
  cache namespaces they affect (incl. `pages` for the full-page cache).
- Frontend: Blade + Tailwind v4 via Vite. No inline `style=""`, no raw hex outside `@theme`, RTL logical utilities
  (`ps-/pe-/ms-/me-/start/end`). JS only as lazy `data-module` ES modules. Images only through `<x-picture>`.
- Tokens: add colours/sizes/radii/shadows only in `resources/css/app.css` `@theme`, then use the generated utility
  (`bg-primary`, `text-on-night-muted`, `bg-stage-ttc/10`) or `var(--color-*)`. Tailwind's default palette is removed.
  Desktop-first breakpoints: `max-xl:` (≤1180), `max-lg:` (≤1024), `max-sm:` (≤700); display sizes step down by
  themselves. RTL: logical utilities only (`ps-/pe-/ms-/me-/start-/end-`, `text-start/end`, `rounded-s/e`).
- JS: add `resources/js/modules/<name>.js` (default export `(el) => void`) and mark the element
  `data-module="<name>"`; `resources/js/app.js` lazy-imports it. Fonts are self-hosted from `resources/fonts`.
- **No external requests** from public pages (no CDN, web fonts, maps, analytics, captcha).
- Every public page: SEO via `SeoManager` (title, description, canonical, robots, OG/Twitter, JSON-LD graph,
  breadcrumbs), one `<h1>`, alt + width + height on images.
- Copy is Persian. Red lines: no diagnosis claims, no «حتماً/قطعاً/دقیق‌ترین/تضمینی», no sales pressure.

## Commands
- `composer verify` — pint --test, phpstan, pest, `npm run build`. Must be green before a task is done.
- `php artisan serve` — local site at http://127.0.0.1:8000 (SQLite at `database/database.sqlite`).
- `tasks/bin/next.sh` — next runnable task; see `tasks/README.md` for the rest.

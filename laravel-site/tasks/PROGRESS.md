# Progress log

One section per finished task (appended by `/site-task`).

## L0-01 — Scaffold Laravel 12 app, move design HTML, tooling and conventions
- Laravel **12.69** skeleton in `laravel-site/`; design export moved untouched to `design/html/` (29 pages, `assets/`,
  `wordpress/`, `README-WordPress.md`) — relative paths still work.
- Dev deps: Pest 3 (+ laravel, arch plugins), Larastan 3 (level 6), Pint (laravel preset + `declare_strict_types`,
  `strict_comparison`). Removed `laravel/sail` (not used on cPanel) and `axios` (YAGNI).
- `composer verify` = `lint` + `analyse` + `test` + `npm run build`. `tests/TestCase` calls `withoutVite()`.
- `.env.example`: `fa` locale, `APP_TIMEZONE=Asia/Tehran` (Iran has no DST since 2022 → no ambiguity), SQLite dev,
  commented MySQL block, `CACHE_STORE=file`, `SESSION_DRIVER=file`, `QUEUE_CONNECTION=database`.
- Placeholder `/` (`Route::view`, named `home`, Persian, noindex, no external fonts — Laravel's default welcome page
  pulled fonts.bunny.net and was replaced).
- `CLAUDE.md` (conventions), `docs/qa/README.md` (PNG screenshots untracked).
- Verify: pint ok, phpstan "No errors", pest 3 passed, vite build ok (empty-page CSS 3.75 KB gz).

## L0-02 — DDD architecture skeleton, service providers and arch tests
- 11 bounded contexts under `app/Domain/*` (README only; sub-folders created on demand), `app/Support/README.md`.
- Abstract `App\Providers\DomainServiceProvider` (`$repositories` incl. `[Eloquent, Cached]` decorator binding via `$inner`,
  `$observers`, `$composers`) + 11 final context providers in `app/Providers/Domain/`, registered in `bootstrap/providers.php`.
- `AppServiceProvider`: `preventLazyLoading` + `preventSilentlyDiscardingAttributes` outside production.
- Arch tests `tests/Arch/{Layers,Conventions,DomainServiceProvider}Test.php`; `phpunit.xml` gained an `Arch` suite.
- `docs/ARCHITECTURE.md`. Verify: pint ✔, phpstan ✔, pest 15 passed, build ✔.

## L0-03 — Cache-aside kernel with versioned namespaces
- `config/cacheaside.php` (store via `CACHE_ASIDE_STORE`, prefix `rt`, jitter 0.1, lock via `CACHE_ASIDE_LOCK`,
  `always_bump=['pages']`, 10 namespaces with default TTLs). Env vars: `CACHE_ASIDE_STORE`, `CACHE_ASIDE_LOCK`.
- `app/Support/Cache/`: `CacheKey`, `CacheNamespace`, `NamespaceVersions` (keys `rt:{ns}:v{n}:{key}`, add+increment bumps),
  `CacheAside` (jitter, stampede lock when store is a LockProvider, null sentinel), abstract `CachedRepository` (`$inner`),
  `NamespaceBumper` (bumps after commit, plus `pages`), abstract `CacheBumpingObserver`.
- `php artisan cache:ns {list|bump ns...|bump-all}`.
- Tests on array/file/database stores; pest 92 passed. Coverage driver not installed → 100% coverage not measured.
- Open: `BumpsCacheNamespaces` model trait dropped (phpstan trait.unused); add it with its first user if wanted.

## L0-04 — HTML design audit: components, tokens, content, SEO gaps, URL map
- `docs/AUDIT.md` (8 sections): page table (29 pages), component inventory with Blade names + owning tasks, tokens
  (colours, stage colours, alpha→opacity map, Lalezar + Vazirmatn 400–800, radii, shadows, gradients, breakpoints
  1180/1024/700), 76 icons + 28 illustrations, SEO/a11y gaps, URL map with a machine-readable ```json urlmap``` block.
- Audit corrections added to the Scope of L1-02, L3-01, L3-03, L3-07, L3-08, L3-09, L5-02, L6-01, L6-05.
- Open (user decisions): multi-seller vs single-seller shop (L6-01), real copy for `[...]` placeholders + legal texts,
  hide donations until a gateway exists, show Plus prices on site, demo slugs English vs Persian.

## L1-01 — Settings context: typed, cached site settings
- Migration `2026_10_04_000100_create_settings_table` (group/key unique, JSON value).
- `app/Domain/Settings`: `Setting` model, `SettingGroup` enum (8 groups → DTOs), readonly DTOs (General, Contact, Social,
  AppLinks, SeoDefaults, Organization, Legal, Pwa) + `SiteSettings`/`LayoutSettings` aggregates, `SettingsRepository`
  contract, Eloquent + Cached (forever, `settings` ns, caches arrays not DTOs) repos, `SettingObserver` (bumps
  `settings`, `seo`, `pages`), `UpdateSettings` action, `LayoutSettingsComposer` (`$siteSettings` for `layouts.*`).
- `SettingsSeeder` (insert-missing only; called from `DatabaseSeeder`), seeded from design copy; placeholders kept as `[...]`.
- Cache namespaces: `settings`, `seo`, `pages`. Tests: 13 new, pest 105 passed. 1 query cold, 0 warm.
- Open: composer lives in `app/Domain/Settings/View` (ARCHITECTURE says `App\Http\View`) — move when touching app/Http;
  L1-02 must strip enamad HTML to allowed tags; real enamad badge is external (conflicts with no-externals rule).

## L0-05 — Tailwind v4 + Vite asset pipeline, design tokens, self-hosted fonts
- `resources/css/app.css`: Tailwind v4 with `source(none)` + explicit sources; one `@theme` with all AUDIT §3 tokens
  (default palette removed), breakpoints sm 701 / lg 1025 / xl 1181 so `max-*` match the design's 700/1024/1180,
  display scale stepping down by media query, base layer, `bg-hero-glow`, `.rt-prose`.
- `resources/css/fonts.css` + 12 self-hosted woff2 (Vazirmatn 400–800, Lalezar 400; Arabic/Latin unicode-range, swap).
- `resources/js/app.js`: lazy `data-module` loader (`import.meta.glob`), `resources/js/modules/`.
- `vite.config.js`: manifest, hashed names, lightningcss minify, es2022 targets. CSS 2.96 KB gzip.
- Orchestrator added CLAUDE.md token/JS docs and `tests/Feature/Frontend/ViteAssetsTest.php` (scope items outside touches).
- Open: `lightningcss` only present transitively — pin it in package.json when the lockfile is next touched.

## L1-03 — SEO core: SeoMeta, SeoManager, head component, basic seo:audit
- Migration `2026_10_04_000300_create_seo_meta_table` (morph `seoable_*` unique or `route_name` unique; `og_media_id`
  without FK until L2).
- `app/Domain/Seo`: `SeoManager` (scoped; layers settings defaults → static page meta → model meta → controller
  override), `SeoMeta` model + `HasSeo` trait, DTOs (`SeoMetaData`, `SeoHead`, `SeoImage`, `PageContext`), cached repo
  (`seo` ns, caches misses), `SeoMetaObserver` (bumps `seo`, `sitemap`, `pages`), `Robots`, `CanonicalUrl`
  (https, app.url host, no trailing slash, allow-listed query: `page`>1), `DescriptionText` (Persian-aware trim),
  `NullOgImageResolver` behind `OgImageResolver`, audit engine (`SeoAuditor`, `PageAudit`, `AuditIssue`, `Severity`).
- `<x-seo.head/>` (title, description, canonical, robots, OG, Twitter, verification, RSS when `blog.feed` exists;
  `@stack('seo.jsonld')` for L1-04). Auto-noindex: non-production, search/cart/checkout/order/done/booked, filtered lists.
- `php artisan seo:audit {--path=*} {--strict}` (in-process render of parameterless GET routes).
- Tests: 53 new; SEO + arch suites green. Missing og:image is a warning until L2 binds a real resolver.
- Open: noindex route patterns / feed route are constants in `SeoManager` (config/ not in touches); real `seo:audit` on
  `/` fails until the layout (L1-02) + routes (L1-05) exist.

## L0-07 — Icon sprite and illustration pipeline
- `tools/build-sprite.mjs` (`--extract` from design HTML, `--optimize` via SVGO, size report; rejects `style=""` and
  external refs). Package added: `svgo@^4` (dev).
- `resources/svg/icons/` 76 icons → Vite-emitted hashed sprite (`resources/svg/sprite.svg` manifest key), 9.0 KB /
  2.4 KB gzip; `resources/svg/illustrations/` 28 SVGO-optimised files (build fails if not optimised).
- `<x-icon>` (`App\View\Components\Icon`: same-origin `<use>`, `label`/`stroke`/`size`, inline fallback without a build)
  and `<x-illustration>` (`Illustration`: inlined, width/height from viewBox, cached via CacheAside `media` ns).
- `docs/AUDIT.md` §4.3 added; `tests/Feature/View/SvgComponentsTest.php` (12 tests). No sprite preload (reason in tool header).
- Orchestrator placed `Illustration.php`, its test and AUDIT §4.3 (outside touches). Pest 171 passed.
- Note: illustrations keep their multi-colour hex fills (SVG artwork, not CSS) — deliberate exception to the hex rule.

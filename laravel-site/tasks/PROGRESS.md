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

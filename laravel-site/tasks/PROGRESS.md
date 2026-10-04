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

## L1-04 — Structured data (JSON-LD graph) builders
- Hand-rolled builders (no spatie/schema-org; reason in ARCHITECTURE "Structured data"): `app/Domain/Seo/Schema/` —
  scoped `SchemaGraph` (merge by `@id`, breadcrumbs, `reviewedBy`, `searchUrlTemplate` for L4-04, JSON_HEX_TAG),
  `PageGraph` (auto Organization/WebSite/WebPage/BreadcrumbList + `schema_overrides`), stable `SchemaIds`, nodes for
  Organization, WebSite, WebPage, Person, BreadcrumbList, MobileApplication, FAQPage, BlogPosting, Product,
  LocalBusiness, ItemList, typed DTOs + enums.
- Wired: `SeoServiceProvider` (scoped graph), `<x-seo.head/>` prints one `ld+json` script. `docs/SEO.md` (Rich Results
  requirements per builder). Tests incl. snapshot `tests/Unit/Seo/Schema/snapshots/sample-graph.json`.
- Open: legacy `@stack('seo.jsonld')` kept (L1-03 test); breadcrumbs must be registered before the head renders
  (L1-02 breadcrumbs component → `SchemaGraph::breadcrumbs()`); Organization logo waits for L2's OgImageResolver.

## L0-06 — style2tw: inline-style → Tailwind converter for design pages
- `tools/style2tw.mjs` (Node built-ins only): reads `@theme` tokens from `app.css`, maps declarations (alpha →
  opacity modifier, 2px spacing grid, `size-*`, `rounded-full`, palette gradients, `bg-hero-glow`, shadows), RTL logical
  mapping, responsive variants parsed from `ritme.css` `[style*=…]` rules (+4 hand-written structural ones), links →
  `route()` via the AUDIT §7 urlmap, inline SVG → `<x-icon>`, `--section N`, stderr report of unmapped/arbitrary/links.
- `tools/README.md`, `tests/tools/style2tw.test.mjs` (27 node:test cases incl. Tailwind validity of all 29 pages).
- All 29 design pages convert with zero `style=`.
- Open: legacy `rt-*` classes kept for L1-02 to replace; substring logic reproduces the design's real rendering
  (e.g. hero `max-lg:py-8`), `max-lg:flex-wrap` is broad; sprite icons use one stroke width (design had 1.8/2.2);
  fake QR stays arbitrary until `x-ui.qr`.

## L0-08 — Fidelity screenshot + diff tool (design HTML vs Laravel route)
- `tools/shot.mjs` (no packages: system Chrome via CDP over Node `WebSocket`, zlib PNG codec): full-page shots at
  390/1440 (configurable `--widths`), reduced motion, mobile emulation <768, pixel diff (>32/255) + diff PNG,
  `--strict` (exit 2 over 3%), `--all` from the AUDIT §7 urlmap, `--only`, `--json`. Any non-local request is blocked
  and fails the run (exit 1); console errors / ≥400 on the Laravel side fail too. `CHROME_PATH` env override.
- Usage for page tasks (with `php artisan serve` running):
  `node tools/shot.mjs --design <page>.html --route <route> --out docs/qa/<TASK-ID>`.
- Open: parameterised routes need an explicit `--route`; height mismatches count fully in the diff; unit tests for the
  exported helpers not added (tests/tools not in touches).

## L1-08 — Filament v4 admin foundation: Persian RTL, local fonts, roles, activity log
- Packages: filament/filament ^4 (4.14, livewire 3.8), spatie/laravel-permission ^6, spatie/laravel-activitylog ^4.10,
  bacon/bacon-qr-code ^3 (TOTP QR as SVG). composer post-autoload-dump runs `filament:upgrade` (public/*/filament, gitignored).
- `AdminPanelProvider` (path `config('filament.admin.path')`, fa/RTL, local Vazirmatn via `LocalFontProvider` + vite
  theme `resources/css/filament/admin/theme.css`, brand from settings, TOTP MFA + recovery codes, local initials avatar,
  `EnforceSessionTimeout`, `RequireMultiFactorForRoles`, `admin:create`).
- `app/Filament`: `AdminRole` (6 roles), deny-by-default `AdminAccess` + policies, Users (super-admin only) and
  Activities (read-only) resources, Settings pages (General/Contact/Social/AppLinks/Legal via `UpdateSettings`), overview widget.
- Migrations 000800 (users admin columns), 000810 (permission tables), 000820 (activity log). `AdminRolesSeeder`;
  test user removed from `DatabaseSeeder` (no users seeded). First admin: `php artisan migrate --seed && php artisan admin:create`.
- Env: `ADMIN_PATH` (admin), `ADMIN_SESSION_TIMEOUT` (60). `config/blade-icons.php` disables blade-icons components so
  `<x-icon>` stays ours. `phpunit.xml` memory_limit 512M.
- Verify: pest 229 passed; no external requests on /admin (shot.mjs + network log).
- Open: panel logo waits for L2; Jalali dates in tables need a Support helper; admin theme CSS 67 KB gzip (admin only);
  L10 package must ship `php artisan filament:assets` output; tune primary button shade if contrast is an issue.

## L2-01 — Media context: upload, optimise, mobile + desktop variants
- Packages: intervention/image ^3.11 (GD/imagick auto), enshrined/svg-sanitize ^0.22 (0.21 had CVE-2025-55166).
- `app/Domain/Media`: `Media` model, DTOs (`MediaUpload`, `MediaData` with url/srcset fallback to original, `VariantSpec`),
  cached repo (`media` ns), `StoreMedia` (finfo sniffing, 15 MB / 8000 px limits, sha256 dedupe, auto-orient, strip
  metadata, 2560 px cap, SVG sanitised + external refs removed), `GenerateMediaVariants` (mobile 480/768, desktop
  1280/1920 × avif(if supported)/webp/fallback, thumb 320, og 1200×630 focal crop, square on demand, LQIP ≤600 B,
  animated GIF kept + poster), `OptimizeMedia` job (after commit), `MediaObserver` (bumps media/seo/pages, deletes files),
  `MediaOgImageResolver` now bound for `OgImageResolver`.
- `php artisan media:regenerate {--id=*} {--preset=*}`; `config/media.php`; public disk root `public/media` (URL `/media`,
  no symlink; gitignored). Migration `2026_10_04_001100_create_media_table`.
- Env (optional): `MEDIA_DISK`, `MEDIA_DRIVER`, `MEDIA_QUEUE_CONNECTION`, `MEDIA_QUEUE`, `MEDIA_PUBLIC_ROOT`, `MEDIA_PUBLIC_URL`.
- Pest 280 passed (whole suite).
- Open: queue worker schedule belongs to L10-01 (until then variants need `media:regenerate` when queue≠sync);
  `seo_meta.og_media_id` FK to media not added; duplicate upload ignores new alt; focal change must trigger
  regenerate (L2-03); memory raised to 512M while processing.

## L1-02 — Base layout, header/nav, mobile menu, footer, breadcrumbs
- `app/Domain/Content`: `StaticPage` registry (24 parameterless pages: route, path, label, nav item, breadcrumb parent,
  header variant (10 dark), `#download`, indexable, sitemap priority/changefreq), `NavItem`, `HeaderVariant`, DTOs,
  `SiteNavigation` (falls back to registry path until routes exist), `AllowedHtml` (enamad sanitiser; external images dropped).
- `app/View/Components/Layout/{Header,Footer,Assets}` (header/footer HTML fragment-cached in `menu` ns, key includes
  `settings` ns version + Vite manifest hash), `Ui/Breadcrumbs` (registers into `SchemaGraph` in the constructor).
- `layouts/app.blade.php` (skip link, `main#main`, dark header/hero shared background via subgrid), components
  `layout/*`, `ui/{button,badge,section,container,breadcrumbs}`, `resources/js/modules/menu.js` (0.34 KB gz).
- Non-production preview route `/_preview/layout/{dark|light}` (`preview.layout`) in `routes/web.php`.
- Diff (header / footer regions): dark 1440 0.70/1.74%, 390 1.71/1.45%; light 1440 0.50/0.94%, 390 0.81/2.74%.
  Whole-page diff meaningless (empty demo body). Shots in `docs/qa/L1-02/`.
- Tests: `tests/Unit/Content`, `tests/Feature/Layout` (47 tests). Pest 327 passed.
- Open: `SettingObserver` doesn't bump `menu` (settings version is in the fragment key instead); shop/social/login
  links hidden while settings empty; labels in enum (no `lang/` yet); real enamad badge is an external image → stripped
  (owner decision); `/` still the welcome view until L3-02.

## L2-03 — Admin media library + reusable media picker
- `app/Filament/Resources/Media/*`: grid with thumb, missing-alt badge, size/savings, search + alt filter, bulk "delete
  unused", multi-upload (≤20, shared alt), edit (alt required, title, caption, focal point, variants panel, usages),
  regenerate/usages/delete actions, deny-by-default `MediaPolicy` (Editor, SeoManager, ShopManager, DirectoryManager, super-admin).
- Reusable fields: `MediaPicker` (Select of media ids, search with thumbs, upload in modal via `StoreMedia`,
  `->mobileOverride()`), `FocalPointPicker` (SVG, keyboard). View namespace `ritme-admin-forms`.
- Domain actions: `UpdateMediaDetails` (focal change regenerates existing crop presets only), `RegenerateMediaVariants`,
  `FindMediaUsages` (settings `*media_id`, `seo_meta.og_media_id`, registered columns), `DeleteUnusedMedia`.
  Duplicate upload now fills a missing alt/title on the existing record.
- **Every context with a media FK must register it**: `FindMediaUsages::column('blog_posts', 'featured_media_id', 'تصویر شاخص مقاله', 'title');`
  in its provider `boot()`, otherwise bulk "delete unused" may remove it.
- Tests: `tests/Feature/Admin/MediaTest.php` (8). One 403 test depended on the in-progress L1-05 error page at commit time.
- Open: Livewire temp upload cap is 12 MB vs media 15 MB (publish livewire config later); picker search queries
  Eloquent directly (Filament-style); savings % is vs the stored (already capped) original.

## L4-01 — Blog context: posts, categories, tags, authors, medical reviewers
- Migrations `2026_10_04_001400`–`001450`: blog_authors, blog_categories (parent, label, life_stage), blog_tags,
  blog_posts (sources, cover + mobile cover media FKs nullOnDelete, reviewer_id/reviewed_at, updated_content_at,
  reading_time, word_count, views), blog_post_tag, blog_post_slugs (slug history; Persian slugs, 191 chars).
- `app/Domain/Blog`: models (+`HasSeo`, factories), DTOs (`PostData`, `PostCardData`, `PostPage`, `AuthorData::toPerson()`
  …), Eloquent + Cached repos (`blog` ns; sitemap rows in `sitemap` ns), queries (Latest, ByCategory incl. children,
  Related scoring, Sitemap*), observers (bump blog + sitemap + pages; `SyncPostTags` bumps for pivot changes), actions
  (PublishScheduledPosts, SyncPostTags, RecordPostView, FlushPostViews), `blog:publish-scheduled` (every minute),
  `blog:flush-views` (every 5 min), sitemap providers `posts` / `blog-categories` (`key/count/entries` for L1-06),
  morph map `blog_*`, media usages registered for covers + author avatar.
- `app/Support/Html`: `RichHtmlSanitizer` (symfony/html-sanitizer; images only `data-media-id` or `/media/`),
  `ExternalLinks`, `HeadingAnchors` (+outline for TOC), `HtmlText` (Persian word count, 200 wpm), `TextSlug`, `HtmlFragment`.
- `BlogSeeder` (idempotent, `--class=BlogSeeder`, not in DatabaseSeeder): «درد پریود» `period-pain` (old slug
  `dard-period` in history) + 5 posts + 7 categories; reviewer placeholder `[نام متخصص]`; red lines tested. 62 tests.
- Open: L1-06 must define `SitemapProvider` matching the Blog providers; L4-03 needs a SeoManager lookup by morph/id
  (controllers get DTOs); media delete via FK null doesn't bump `blog`; sitemap URLs need https in production.
- Incident: the agent's `migrate:fresh --env=testing` wiped the dev SQLite DB (no `.env.testing`); re-seeded settings
  + roles. Recreate the admin with `php artisan admin:create`.

## L1-05 — Routing, clean URLs, legacy 301s, trailing slash, error pages
- `routes/web.php`: all 29 AUDIT §7 routes + `/terms`, named (see table below), no closures, owner task commented per
  line; every page currently served by the final invokable `PlaceholderPageController` (one h1, registry breadcrumbs,
  noindex) — each page task swaps its own line. Preview route kept.
- `CanonicalizeUrl` global middleware (one 301 hop, query kept): collapse/strip slashes, `*.html`, `/ritme-static/*.html`,
  WordPress slugs → new routes (`App\Domain\Content\LegacyUrlMap`), lowercase static paths only, 410 for
  `/ritme-static/assets/*` + `/wp-content/uploads/ritme/*`, optional scheme/host redirect via `app.canonical_redirect`
  (not yet in config — off), sub-folder base URL aware.
- Error views: 404/410/419/429 on `errors/shell` (site layout, noindex, popular links; 404 zero queries warm),
  500/503 on `errors/standalone` (no DB/settings/cache). Laravel's `errors::minimal` deliberately not overridden.
- Tests: `tests/Feature/Routing/{Routing,ErrorPages}Test.php` (134, driven by the AUDIT urlmap). `route:cache` works.
- Route names: `home`, `stage.{cycle,ttc,pregnancy,postpartum,menopause,teen}`, `services`, `plus`, `tools`, `about`,
  `social-responsibility`, `privacy`, `terms`, `faq`, `contact`, `blog.index`, `blog.show`, `directory.{index,business,
  join,join.done,place,booked}`, `shop.{index,category,product,cart,checkout,order}`; `{code}` = `[A-Za-z0-9-]+`.
- Open: add `canonical_redirect` (`APP_CANONICAL_REDIRECT`) to config/app.php + .env.example (L1-07/L10); L5-02 must
  register `/directory/{city}[/{category}]` after the fixed directory routes; L7-03 DB redirects go before LegacyUrlMap;
  `welcome.blade.php` unused.

## L2-02 — <x-picture> responsive image component with art direction
- `App\View\Components\Picture` + `components/picture.blade.php`: accepts `MediaData` or id (cached `MediaRepository`),
  `<source>` per format (avif when encoded, webp) over mobile_480/768 + desktop_1280/1920, fallback `<img>` with
  width/height always, `decoding=async`, lazy by default; `mobile` art direction (`max-width: 767px`); alt required
  unless `decorative` (throws outside production); `priority` → eager + fetchpriority high + `@pushOnce('head')` preload.
  Opaque images get `bg-lavender`; LQIP/dominant colour deliberately not emitted (would need inline style).
  `Picture::mediaUrl($media, 'og')` for absolute URLs. 12 tests.
- Usage: `<x-picture :media="$page->hero" :mobile="$page->heroMobile" sizes="(max-width: 768px) 100vw, 50vw" priority />`.
  Pass `MediaData` DTOs prepared with `findMany` on list pages. Use `priority` only inside page sections (not cached fragments).
- Open: global `media_url()` helper needs composer autoload files; Lighthouse check with the first page task.

## L1-06 — Sitemaps index + robots.txt (cached)
- `app/Domain/Seo/Sitemap/`: `SitemapProvider` contract (`key/count/entries(page, perPage)`), `SitemapEntryData`,
  `SitemapRegistry` (tag `seo.sitemap.providers`, key validation), `PagesSitemapProvider` (indexable `StaticPage`s,
  honours `seo_meta.sitemap_include` + noindex), `SitemapXml` (index/urlset + `image:image`), `Sitemaps` (cache-aside
  `sitemap` ns, 5000/file, `pages.xml`, `posts-2.xml`…), `SitemapUrl` (absolute https on app.url via `CanonicalUrl`),
  `RobotsRules` → `DefaultRobotsRules`, `RobotsTxt` (non-production: `Disallow: /`; production: rules + Sitemap line).
- Routes `robots`, `sitemap.index`, `sitemap.file` (`/sitemaps/{file}.xml`) without session/cookies/CSRF; 301 from
  `/sitemap_index.xml` + `/wp-sitemap.xml`. Static `public/robots.txt` removed. Blog providers implement the contract
  (old `Blog\Data\SitemapEntryData` removed). 18 tests; warm requests run zero queries.
- New contexts register: `$this->app->tag([XSitemapProvider::class], SitemapRegistry::TAG);` + bump `sitemap` in observers.
- Open: placeholder pages are noindex but listed until real pages land; static lastmod omitted; robots rules
  hard-coded until L7-04 rebinds `RobotsRules`.

## L1-07 — Performance core: page cache, HTTP caching, minify, security headers, .htaccess
- Middleware: `SecurityHeaders` (outermost; strict all-`'self'` CSP for public pages, relaxed but still same-origin for
  /admin + Livewire; HSTS in production https; nosniff, Referrer/Permissions-Policy, XFO, COOP; opt-in nonce),
  `HttpCacheHeaders` (cookie-free first visits `public, max-age=0, s-maxage=300, swr=60`, else `private, no-cache`;
  weak ETag + 304 ignoring nonce), `PageCache` (guest GET/HEAD 200 HTML via CacheAside `pages` ns, key includes Vite
  manifest hash; bypasses admin/livewire/filament/preview/transactional routes, logged-in, remember/cart cookies,
  flash/errors, no-cache, non-`page` queries; CSRF token placeholder swapped per visitor; `X-Page-Cache` header),
  `MinifyHtml` (−16% raw), `AdminPaths`. `config/pagecache.php`.
- `public/.htaccess` rewritten (public_html-safe, immutable `/build` + `/media`, brotli/gzip, MIME, dotfiles 403,
  short cache for sw.js/manifest, every directive commented; Apache trailing-slash redirect removed). Tested on Apache 2.4.
- `config/app.php` `canonical_redirect` (closes L1-05 item). Env: `APP_CANONICAL_REDIRECT`, `PAGE_CACHE_ENABLED`,
  `PAGE_CACHE_TTL`, `PAGE_CACHE_MINIFY`, `HTTP_CACHE_S_MAXAGE`, `HTTP_CACHE_SWR`, `SECURITY_HEADERS`, `SECURITY_HSTS`,
  `SECURITY_HSTS_SUBDOMAINS`, `SECURITY_CSP_NONCE`.
- Tests: `tests/Feature/Http` (44). Second anonymous `/` request is a HIT with 0 queries.
- Orchestrator: `tools/shot.mjs` now calls `Page.setBypassCSP` (strict CSP blocked its injected style). Media variant
  versioning split out as **L2-01b** (immutable cache vs same-name regenerated variants).
- Open: deploys should `php artisan cache:ns bump pages`; cookie-free public responses still write a session file
  (keep file sessions in production); use sha256 hashes, not nonces, for any future inline script.

## L3-01 — Shared UI component kit from the audit
- Anonymous Blade components (no queries; props/DTOs): `ui/{icon-tile,eyebrow,section-header,pill,chip-nav (links +
  aria-current),accordion (<details>),alert-emergency,steps,stepper,timeline,success-hero,rating,price,toggle-row,
  newsletter,gallery,store-badges,qr (real SVG QR via bacon, cached in media ns),app-cta,promise-banner,promo-split,
  page-intro}`, `ui/form/{field,input,textarea,select,radio-card (has-checked:),checkbox}`, `cards/{stage,feature,
  service,value,article,product,place,review}`, `stage/{tools-block,help-block,readings}`. L1-02 components untouched.
- `docs/COMPONENTS.md` = hand-off for page tasks. Preview `/_components` (non-production) + `previews/components.blade.php`.
  `tests/Feature/Components/ComponentKitTest.php` (14; also scans kit for `style=`, raw hex, physical left/right).
- Open: Persian digits / Jalali / Toman helpers not built (app/Support outside touches) → follow-up **L3-01b**;
  components use `Footer::persianDigits()` and an inline Toman formatter meanwhile. app-cta aside built as the design's
  night card (AUDIT said white).

## L2-01b — Versioned media variant URLs
- Variant files are named `{stem}-{variant}.{hash10}.{ext}` (xxh128 of the encoded bytes,
  `GenerateMediaVariants::version()`): changed pixels → new URL, identical bytes → same name (safe under immutable cache).
  New set written before stale files are deleted; observer bumps media/seo (+pages). No migration; legacy unversioned
  rows keep working until their next regenerate. 3 new tests in `MediaLibraryTest`.

## L4-05 — Admin: posts editor with SEO tab, categories, tags, authors (newsletter → L4-05b)
- Reusable SEO tab `app/Filament/Components/Seo/SeoFields` (title/description counters with pixel width incl. title
  template, focus keyword, SERP preview desktop/mobile, OG fields + card preview, canonical override, index/follow,
  sitemap include/priority) for any `HasSeo` model: `SeoFields::make()->titleFrom('name')->descriptionFrom(...)
  ->imageFrom('cover_media_id')->urlUsing(fn (Get $get) => route(...))`. Static pages (L7-01) need a route_name save path.
- Blog resources: Posts (local TipTap editor, library image insert via `data-media-id`, sanitised on save, no h1,
  status/schedule, featured, author + reviewer, life stage, inline category/tag creation, cover + mobile cover,
  signed 24 h draft preview at `admin/blog/preview/{post}`, duplicate, revision trail in activity log, bulk
  publish/unpublish), Categories (parent, label, life stage, reorder), Tags, Authors (credentials, sameAs, avatar,
  reviewer flag). Policies: editors full; SEO managers SEO tab only (server-enforced, `SeoOnlyForSeoManagers`); others 403.
- Domain actions: `SavePost`, `DuplicatePost`, `SetPostsStatus`, `FindOrCreateTag`. Tests: `BlogAdminTest` (12). Pest 632 passed.
- Split out **L4-05b**: newsletter subscribers list/export (needs L4-02's Newsletter model) + media usage scan of post bodies.
- Open: L4-03 renders body images by `data-media-id` via `<x-picture>`; tag/author URLs assumed `/blog/tag/{slug}`,
  `/blog/author/{slug}` (L4-02 to confirm); admin date picker is Gregorian.

## L3-01b — Support helpers: Persian digits, Jalali dates, Toman formatting
- No package (morilog/jalali's own global `jdate()` would shadow ours): `App\Support\Jalali\{JalaliCalendar,JalaliDate}`
  (integer port of jalaali-js; tested daily 1300–1500 + against ICU), `App\Support\Text\{PersianDigits,Toman}`,
  `app/Support/helpers.php` via composer `autoload.files`: `fa_digits()`, `jdate($date, 'j F Y', $persianDigits)` (Tehran).
- Components switched (price, rating, gallery, steps, stepper, alert-emergency, cards/article; timeline accepts
  DateTimeInterface → `<time>`). `Footer::persianDigits()` removed. 74 unit tests; pest 706 passed.
- Filament: `->formatStateUsing(fn ($s) => $s ? jdate($s, 'Y/m/d H:i') : null)`.
- Open: error views + `SerpMeasure` still have inline digit maps (switch when next edited); Toman shows «۱٬۵۰۰ هزار»
  for 1.5M (L6-01 may change).

## L3-02 — Home page (index) in Blade
- `HomeController` (title/description from design via `lang/fa/home.php`, admin `seo_meta` for route `home` wins;
  MobileApplication node; readings via cached `PostRepository::latest(1, 3)`; app links/QR from settings; static middle
  sections fragment-cached in `pages` ns keyed by manifest hash + template/lang mtimes). Route `home` → controller;
  `welcome.blade.php` removed.
- Views `pages/home.blade.php` + `pages/home/{hero,static,split}` + phone mockups (`aria-hidden`), built from the kit.
  LCP is the h1 text. `tests/Feature/Pages/HomeTest.php` (7). `seo:audit --path=/`: 0 errors (og.image warning).
- Diff: 1440 whole 19.58% (FAQ slot intentionally empty, −680 px) — aligned top 0–5700 px 1.55%, bottom 2.16%;
  390 whole 46.77% — top 3.45%, bottom 2.51%. Deliberate fix: design's `#how` card kept 120 px side margin on mobile
  (→ `max-sm:mx-5 max-sm:p-6`). Kit gaps: `cards/article` cover 180 vs 216 px, footer store badges stack vertically.
- Open: L3-09 replaces the marked FAQ slot with `x-faq` group `home`; L3-07 must provide `/tools#due-date`, `#fertility`,
  `#hospital-bag`, `#sisemoni`; mockup demo names/numbers live in `lang/fa/home.php`; «۸۶ مورد» claim to confirm.

## L3-03 — Life-stage page template + cycle page
- Data-driven stage template: `app/Domain/Content/Stages/` (`StageDefinition` base, `Cycle`, `StageRegistry`,
  `StagePageBuilder` (throws on missing copy), `StageNavigation`, specs + DTOs), `StagePageController`; copy in
  `lang/fa/stages/{common,cycle}.php`; views `pages/stages/show` + partials (stage-nav, hero, feature, faq) + mock
  screens (fragment-cached in `pages`). All six `stage.*` routes point to the controller; stages without a class fall
  back to the noindex placeholder.
- SEO: lang title/description unless admin `seo_meta` wins; WebPage + BreadcrumbList + FAQPage + MobileApplication;
  descriptive FAQ h2; `#how` anchor fixed; emergency alert number from settings; readings = same-stage posts topped up
  with latest. `StagesTest` (15; checks title 30–60 / description 70–160 for every defined stage).
- Diff: with app/social links filled 390 2.33% / 1440 0.76%; with empty links (current dev DB) 7.47% / 4.84% (download
  buttons, login and social links are hidden by design while settings are empty). `seo:audit --path=/cycle` 0 errors.
- Orchestrator: `RoutingTest` placeholder check now picks any remaining placeholder route.
- Add a stage (L3-04/05): `app/Domain/Content/Stages/<Studly>.php` extending `StageDefinition` + `lang/fa/stages/<slug>.php`
  (same keys as cycle) + optional partials in `pages/stages/partials/<slug>/` and screens in `mock/screens/`.
- Open: stage-nav/hero/faq/phone are includes under `pages/stages` (components dir was outside touches); FAQ copy in
  lang until L3-09 wires group `stage-cycle`; `cards/article` cover worked around with `[&_a>span:first-child]:box-content`;
  "all" readings link → `/blog` until category routes are wired.

## L4-02 — Magazine list, category, tag, author pages + newsletter signup
- `BlogListingController` (index/category/tag/author), routes `blog.{index,category,tag,author}`; `BlogUrls::tag()/author()`
  (`/blog/tag/{slug}`, `/blog/author/{slug}` confirmed). Pagination `?page=n` self-canonical + «صفحه n», `?page=1` 301,
  invalid → 404; other params noindex + uncached; empty lists / tags < 3 posts (`config('blog.tag_index_min_posts')`,
  default 3) / authors without posts (unless reviewer with bio) → `noindex,follow`. Admin `seo_meta` overrides.
  JSON-LD CollectionPage + ItemList + BreadcrumbList; ProfilePage + Person for authors. New sitemap providers
  `blog-tags`, `blog-authors`.
- Newsletter context: `Subscriber` (table `newsletter_subscribers`, migration `2026_10_04_001500`), `SubscriptionStatus`,
  `Subscribe`/`ConfirmSubscription`/`Unsubscribe`, `ConfirmSubscriptionMail`; double opt-in, honeypot `website`,
  `throttle:newsletter` (3/min, 20/day per IP), identical responses, unsubscribe confirmation form + RFC 8058 one-click
  POST (token-authorised, CSRF-free), token pages bypass the page cache. Mail sent synchronously.
- `ListingTest` (12). `seo:audit` 0 errors on list/category/tag/author pages (og.image warnings).
- Diff (seeded data): 1440 8.58%, 390 23.22% — design repeats the featured post as the 6th card; with equal content
  1440 2.35%, 390 8.66% (kit padding deviations + real reading times). Kit fixes collected in **L3-01c**.
- Open: newsletter input is email-only (design says «ایمیل یا شماره همراه»); consent note dropped (privacy in success
  message + mail); `/blog/{slug}` is L4-03; admin subscriber list is L4-05b.

## L3-04 — TTC and pregnancy stage pages
- `Stages/Ttc.php` (fertility/companion/treatment features, `/tools#fertility`), `Stages/Pregnancy.php` (weekly,
  appointments, birth-prep; tools → `#hospital-bag`, `#sisemoni`, `#due-date`; help → directory), copy in
  `lang/fa/stages/{ttc,pregnancy}.php`, mock screens `ttc-companion`, `pregnancy-week`. `StagesTtcPregnancyTest` (6).
- Diff (links filled): ttc 390 2.67% / 1440 0.96%; pregnancy 390 3.45% (template −8 px drift accumulating over 6 tool
  cards) / 1440 1.66%. Empty links: ~8% / ~5%. `seo:audit` 0 errors. Design `#FFB86B` → `phase-luteal` token.
- h1 wording reworded to fit the template's leading highlight → fixed by **L3-03b** (template hooks).

## L3-07 — Tools page: due-date and fertility calculators (JS + no-JS fallback)
- `ToolsController` (design title/description unless admin seo_meta; submitted URLs noindex + canonical `/tools`; one
  free `WebApplication` node per calculator), `app/Domain/Content/Tools/` (`CalculatorInput` parses Persian/Arabic/Latin
  + «۱۲ اردیبهشت ۱۴۰۵», `DueDateCalculator` LMP+280 adjusted by cycle−28, `FertilityWindowCalculator` ovulation −5..0,
  DTOs, enums), `JalaliCalendar::toDayNumber()/fromDayNumber()` wrappers, `resources/js/lib/jalali.js` (same algorithm),
  lazy `resources/js/modules/calculators.js`. No-JS GET fallback (query URLs bypass page cache). Anchors `#due-date`,
  `#fertility`, `#hospital-bag`, `#sisemoni`. Estimate wording + "not a diagnosis" note.
- Shared PHP/JS vectors `tests/js/fixtures/calculator-vectors.json`; `ToolsTest` (9), `CalculatorVectorsTest` (6),
  `node --test tests/js` (7). `seo:audit --path=/tools` 0 errors.
- Diff: 1440 1.96%; 390 9.34% (≈35 px shift from the added two-line note + next-period detail). Later re-shoots skewed
  by app-link settings being NULL in the dev DB (their seeded default).
- Open: example dates (1405) in `lang/fa/tools.php` will age; `x-ui.warn` + checklist card inline (extract if reused);
  confirm wording «الان حدود N هفته و M روز…» (design was off by one week).

## L3-05 — Postpartum, menopause and teen stage pages
- `Stages/{Postpartum,Menopause,Teen}.php` (teen: no help block; «ساده» + «همراهی مادر» features), copy in
  `lang/fa/stages/{postpartum,menopause,teen}.php` (+`mock` keys), screens `postpartum-baby`, `postpartum-partner`,
  `menopause-status`, `teen-today`, `teen-mother` (no fertility/partner content for teens). `StagesOtherTest` (9).
- Diff (links filled) 390 / 1440: postpartum 6.04 / 1.90%, menopause 6.41 / 3.92%, teen 4.95 / 0.95%; above the
  readings block 390 is 1.18 / 1.62 / 1.46% — the overrun comes from real two-line article titles in the readings
  block. `seo:audit` 0 errors.
- Copy changes: menopause insurance line adapted (design said maternity cover); teen age/consent placeholder answered
  by pointing to terms/privacy + parental involvement (legal text still owed); h1 highlight spans leading words until
  **L3-03b**.

## L4-03 — Article page: TOC, author/reviewer box, related posts, BlogPosting schema
- `ShowPostController` (slug history 301 keeping query, 404 for unknown/unpublished), `app/Domain/Blog/Rendering/`
  (`ArticleBodyRenderer`: re-sanitise, `data-media-id` → `<x-picture>` (first eager if no cover), table scroll wrappers,
  cached in `pages`; `ArticlePageBuilder`, `ShareLinks`), `pages/blog/show` + `components/blog/*` (meta, cover LCP
  priority, toc, app-card, reviewer, share, adjacent, sources, disclaimer, body-image), `resources/css/prose.css`
  (imported in app.css). `PostRepository::adjacentInCategory()` (cached).
- SEO: BlogPosting (headline ≤110, dates +03:30, author, publisher, section, keywords, wordCount, OG crop image),
  WebPage reviewedBy + lastReviewed, BreadcrumbList, `og:type article` + `article:*`, `max-image-preview:large`.
  `ArticleTest` (10). `seo:audit` 0 errors.
- Diff whole page 1440 18.16% / 390 27.41% (height: hidden store badges/login with NULL app links, mobile sidebar
  stacking, design's red callout not in seeded body); article region 1440 1.73% / 390 3.46%.
- Views counted on MISS only → superseded by **L4-03b** (view beacon counts cache HITs).
- Orchestrator: `RoutingTest` seeds `BlogSeeder` for `blog.show`; slug-case test asserts "no 301" instead of 200.

## L3-08 — About, social responsibility, privacy (+ terms) pages
- Controllers/views/lang for `/about` (AboutPage), `/social-responsibility`, `/privacy` (full policy in `<details
  id="policy">`, no-JS), `/terms` (dark header via controller); dates from `privacy.updated_at` / `terms.updated_at`
  → `dateModified`; DPO + support email and emergency number from settings; donation card → contact CTA (no gateway);
  transparency link hidden while empty; `#review-policy` anchor on the scientific council box. `InfoPagesTest` (10).
- Policy text states only what the site does (no trackers/external requests, newsletter double opt-in, contact form,
  session/CSRF cookies, order/booking data sharing, user rights). `seo:audit` 0 errors on all four.
- Diff body-above-footer: about 390 2.69% / 1440 2.09%; privacy 2.78% / 1.92%; social 13.27% / 4.73% (hidden store
  badges with NULL app links, donation amounts replaced by CTA per AUDIT §8). Whole page +5–8% from the shorter footer.
- **Legal/product placeholders to fill**: company legal name/registration/address; age + parental consent; app data
  inventory + retention; host/server location; SMS + email providers; legal-request process; backup purge window;
  message + server-log retention; response deadline; governing law; account suspension terms; IP owner; republishing
  policy; refund policy; shipping/returns; provider liability; dispute resolution; `data_protection_email`,
  `support_email`; about: founder story, mission, stats, team, council names, careers URL; social: programmes,
  founder quote, transparency period + URL.
- Open: move `*_updated_at` + transparency URL into `LegalSettings`; editorial review policy text not written yet.

## L3-09 — FAQ context, /faq page and reusable FAQ blocks + admin
- Migrations `2026_10_04_001900` (faq_groups: slug, title, is_listed, sort_order) + `001910` (faq_items, cascade FK).
- `app/Domain/Faq`: models, DTOs, cached repo (`faq` ns, all groups in one entry), `FaqObserver` (bumps faq + pages,
  sanitises answers, wraps plain text), `PageFaq` (one FAQPage node per page from visible Q&As only), `InvalidateFaqCache`
  (drag-sort/seeder). `FaqServiceProvider` view composer gives `$faq` + JSON-LD to `pages.home`→home, `pages.plus`→plus,
  `pages.contact`→contact, `pages.directory.business`→directory-business — those views MUST render `$faq`.
- `/faq` (`FaqController`, sidebar anchors, `faq-filter` data-module search), `x-faq` (list/grid, bare, slot), home FAQ
  slot filled, stage pages read `stage-<slug>` groups (lang fallback kept). Filament `FaqGroupResource` + items relation
  manager (drag-sort, publish toggles, preview, local RichEditor), `FaqPolicy` (Editor + super-admin).
- `FaqSeeder` (idempotent; 15 groups / 45 items, stage groups seeded from lang) called from `DatabaseSeeder`. `FaqTest` (11).
- Diff /faq: NULL app links 390 16.2% / 1440 8.51%; links filled 3.93% / 2.81%. Home 1440 dropped 19.58% → 6.99%.
  `seo:audit` 0 errors (title lengthened to ≥30 chars).
- Plus usage: `<x-faq :faq="$faq" variant="grid" eyebrow="قبل از خرید" :title="$faq?->title" bg="surface"/>`.
- Open: /faq copy in Blade/controller (move to `lang/fa/faq.php`); design placeholders seeded verbatim for editors;
  blog `CategoryResource` drag-sort doesn't bump cache; controllers holding request-scoped SchemaGraph/SeoManager in
  constructors keep stale graphs across multiple requests within one test.

## L4-04 — RSS feed, site search, WebSite SearchAction
- `FeedController` `/blog/feed` (`blog.feed`, before `/{slug}`, no session/cookies): RSS 2.0, 30 latest, absolute https,
  RFC 822 dates, permalink guids, OG-crop enclosure; XML cached in `blog` ns. `<x-seo.head/>` now prints the alternate link.
- `/search` (`search`, `throttle:30,1`, noindex meta + `X-Robots-Tag`, never page-cached, SearchResultsPage, GET form).
  `app/Domain/Search`: `SearchTerms` (Persian normalisation ي/ك/ة/ZWNJ/tatweel/diacritics/digits, ≤5 tokens),
  `SearchRegistry` (posts + FAQ built in; others via `SearchRegistry::TAG` — L5/L6), anonymous daily term log in cache
  (35 days, `zeroResults()` for L7-05), `SearchSite` (merge + rank, 300 s cache in `pages`). `Blog/Queries/SearchPosts`
  (LIKE + LOWER/REPLACE chain, works on SQLite + MySQL). `searchUrlTemplate()` wired → WebSite SearchAction.
- `FeedSearchTest` (21). Updated ErrorPages/HeadComponent/SeoManager tests that asserted the routes' absence.
- Orchestrator: `PostSlugger::RESERVED` (feed, category, tag, author, page, search) + test.
- Open: no search box in header/mobile menu yet (layout components); MySQL FULLTEXT not added; term log in cache.

## L3-01c — Kit fixes from page fidelity
- `cards/article` cover `h-54` (216 px like the design), featured text `py-8 ps-0 pe-8` at all widths;
  `ui/store-badges` content-box 54 px (app-cta 390 height now equals the design); `ui/newsletter` `value`,
  `describedby`, named `error` slot (aria-invalid/aria-describedby), keeps `p-9` on mobile. Page workarounds removed
  (stages/show, blog/index, blog/show). Error views use `fa_digits()`. Footer badges already match the design.
- Diff before → after (NULL app links): `/` 390 43.02→42.50%, 1440 6.99→4.61%; `/cycle` unchanged 7.47/4.84%;
  `/blog` 390 28.80→28.25%, 1440 14.62% unchanged; `/blog/period-pain` unchanged. Pest 823 passed.
- Open: `BlogListingController::digits()` and `SerpMeasure` still have inline digit maps.

## L3-06 — Services and Plus pages
- `ServicesController` (CollectionPage; emergency number + app links from settings; in-app care cards as `<li>`+h3, no
  dead links; bottom cards → `directory.index`, `shop.index`), `PlusController` (WebPage; FAQ grid via L3-09 composer;
  no Product/Offer). Views + `lang/fa/{services,plus}.php`, `ServicesPlusTest` (7). Route lines swapped.
- Pricing: design only has placeholders → `plus.pricing.show` in `lang/fa/plus.php` defaults **off** («قیمت در اپ»;
  free tier «۰ تومان» always shown); when on, integer Toman amounts render via `Toman::withUnit`. «با تخفیف» removed,
  placeholder feature rows dropped. No countdown/scarcity.
- Diff: links NULL services 12.12/8.24%, plus 26.33/9.89%; links filled services 1.31/0.77%, plus 390 16.2% (dropped
  placeholder rows + design's `max-lg:flex-wrap` render bug not reproduced) / 1440 1.16%. `seo:audit` 0 errors.
- Open: owner decision on prices (move to a `PlusSettings` group); seeded FAQ answer still shows
  «[سیاست بازگشت وجه…]» on /plus; `promo-split` mobile padding (worked around with `max-sm:[&>a]:p-9`); `x-cards.pricing` not built.

## L3-03b — Stage template hooks: inline h1 highlight, per-stage mock copy, shared companion screen
- `hero.title` supports `:highlight` (split into `titleBefore`/`titleAfter`; no placeholder = old leading highlight).
  All six stage h1s now match the design wording.
- `StagePageBuilder::screenCopy($group, $screen)` reads `<stage>.mock.<screen>` then `stages/common.mock.<screen>`
  (nested lists kept); stage screens read `$data` only, so the fragment key (md5 of data) follows lang changes.
- One shared `mock/screens/companion` for home, ttc, postpartum, teen (teen: mother view, no partner content);
  `teen-today` → `cycle-today` with teen copy. Five old screen views removed (the deletions were accidentally swept
  into the L3-06 commit `d64f67c5`; this commit restores consistency).
- Orchestrator: `HomeController::templateVersion()` also watches `pages/stages/mock/screens/*`.
- Diff (NULL links) unchanged or better on all pages; hero regions improved (ttc 1440 3.69→2.64%, pregnancy 4.85→2.49%).

## L4-03b — Article view beacon, share copy module, blog UI copy to lang
- `POST /blog/{slug}/view` (`blog.view`, `PostViewController`, 204 + no-store, no session/cookies/CSRF, `throttle:30,1`,
  id must match the published post; bots/empty UA/Save-Data/prefetch/prerender/cross-site ignored with 204) →
  `RecordPostView`. `ShowPostController` no longer counts (no double counts; no-JS visitors are not counted).
- `resources/js/modules/view-beacon.js` (sendBeacon / keepalive fetch, once per post per tab session, waits for
  visibility), `share.js` (copy-link button revealed by JS, Clipboard API + fallback, Persian aria-live status).
- `components/blog/*` strings moved to `lang/fa/blog.php` (`article.*`). `ViewBeaconTest` (16); CSP unchanged.
- Open: `ShareLinks` labels + two headings in `pages/blog/show.blade.php` still inline.

## L5-01 — Directory context: places, categories, cities, amenities, reviews
- Migrations `2026_10_04_002000`–`002100` (11 tables: directory_cities, _districts, _categories, _amenities, _places,
  _place_slugs, _place_amenity, _place_media (with id), _place_services, _reviews, _landings).
- `app/Domain/Directory`: models (+`HasSeo`), enums (PlaceStatus, ReviewStatus, ReviewAspect, PlaceSort, Weekday from
  Saturday), DTOs (`PlaceData::toLocalBusiness()`, `PlaceCardData`, `PlaceSearchCriteria`, `RatingSummaryData`), Place +
  Taxonomy repos (Eloquent + Cached, `directory` ns), queries (SearchPlaces with Persian normalisation + open-now
  (5-min cache) + sorts without paid placement, LandingCombos, SitemapPlaces), observers, actions (rating/price
  recalculation, amenities/gallery sync, SubmitPlaceReview, ModeratePlaceReviews), support (OpeningHours incl. past
  midnight + Tehran time, AgeRange, MapLinks geo:/Neshan/Balad/Google links only, LandingCopy, PlaceSlugger,
  DirectoryUrls), sitemap providers (places, city×category landings), `PlaceSearchProvider`; media usages registered.
- `DirectorySeeder` (idempotent, not in DatabaseSeeder): آب‌پری `ab-pari` + 5 places, Tehran + 5 districts, 8 categories;
  all `is_demo` (excluded from rating + sitemap), addresses `[آدرس کامل مجموعه]`, no phones; design rating «۴٫۸ (۱۲۶)» not seeded.
- 59 tests; full pest 921 passed.
- Open: L5-02 route names assumed `directory.city`, `directory.category` (city slugs reserved vs fixed routes);
  `LandingCopy` templates hard-coded; L5-03 must noindex demo places, label demo reviews «نمونه», show the pool health
  note; Neshan/Balad URL formats unverified; LIKE normalisation duplicated with `SearchPosts` (candidate for app/Support).

## L4-05b — Admin newsletter subscribers + media usage scan of rich bodies
- `app/Filament/Resources/Newsletter/SubscriberResource` (read-only list: email, status badge, source, Jalali dates;
  status filter; header "CSV" link carrying filter + search), `SubscriberPolicy` (Editor + super-admin; `export`
  ability), `ExportSubscribersController` (separate streamed route, `private, no-store`, noindex),
  `App\Domain\Newsletter\Actions\ExportSubscribers` (UTF-8 BOM, Persian headers, Jalali Tehran dates with Latin digits,
  email/status/dates only, `lazyById(500)`, formula-injection guard, activity log `newsletter.exported`).
- `FindMediaUsages::html($table, $column, $label, $titleColumn)` scans rich HTML for `data-media-id="N"` (chunked);
  `blog_posts.body` registered → body-only images survive "delete unused". Tests: `NewsletterAdminTest` (4) + 1 media test.
- Open: Activity filter lacks a `newsletter` option; no subscriber delete action (data-erasure requests).

## L8-01 — Web app manifest, icons, favicons, install metadata
- `app/Domain/Pwa`: `WebManifest` (cache-aside in `settings` ns + media version; id `/`, fa/rtl, `start_url /?source=pwa`,
  standalone, colours from settings, categories, icons, screenshots with form_factor, shortcuts tools/blog/shop),
  `IconSetResolver`, `GeneratePwaIcons` (from an uploaded raster logo ≥512 px: any 96/192/512, maskable, monochrome,
  apple-touch 180, favicon png/ico into versioned `pwa/{id}-{hash}/` on the media disk), `PwaIconFiles` (`?v=` hashes).
- `ManifestController` `/manifest.webmanifest` (`pwa.manifest`, `application/manifest+json`, 1 h + ETag/304, no session).
  `<x-pwa.head>` (manifest, light/dark theme-color, apple meta, favicon.ico + SVG, apple-touch, mask-icon) replaces the
  old theme-color in `Layout/Assets`. Filament `PwaSettings` page (auto-discovered; regenerates icons on save).
  `PwaSettings` DTO gained `icon_media_id`. Default assets in `public/icons/` + real `public/favicon.ico`;
  `tools/pwa-icons.mjs` (brand mark via headless Chrome, ICO packing, `--screenshots`). `ManifestTest` (8).
- Chrome DevTools: no manifest/installability errors, no external requests.
- Orchestrator: `source` added to `SeoManager::TRACKING_PARAMS` so PWA launches hit the page cache.
- Open: seeded `pwa.name` is still «ریتمی» until saved in admin; mask-icon stays the default vector with raster logos.

## L5-02 — Directory listing + city/category SEO landing pages
- `ListPlacesController` (one invokable for `/directory`, `/directory/{city}`, `/directory/{city}/{category}` — routes
  `directory.index|city|category`, landings registered after the fixed directory paths): GET filters (q, district,
  age, open, amenity[], sort without "nearest", page); every non-canonical query 301s to one canonical form (form
  fields → path, sorted amenities, `?page=1` dropped, tracking kept); filter combos `noindex,follow` with canonical on
  the nearest landing; pagination self-canonical; unknown city/category/page → 404. Copy from `LandingCopy` + admin
  `directory_landings` rows; CollectionPage + ItemList + BreadcrumbList. "Open now" pill only if open for the whole
  1 h page-cache TTL. Category illustrations as covers for places without photos.
- Views `pages/directory/index` + partials (intro, toolbar, pagination, areas, business): native `<details>` filter
  panel, no JS; map column replaced by a decorative local illustration + crawlable city/district/landing links.
  `lang/fa/directory.php`; `ListingTest` (10). `seo:audit` 0 errors on the three URL shapes.
- Diff 1440 26.12% / 390 49.15%: height (NULL app links, 6 demo places vs 4-page mock pagination, no fake ratings or
  distances, areas panel instead of the map, design's broken 390 search form fixed). Top section matches visually.
- Open: demo-only lists are noindex (incl. /directory in dev); booking slots (L5-04); bookmark button inert.

## L3-10 — Contact page + Contact context (messages inbox, anti-spam)
- Migration `2026_10_04_001950_create_contact_messages_table` (topic, name, email/phone, message, status, read_at; no
  IP/UA). `app/Domain/Contact`: model, enums (5 topics, unread/read/archived), `ReplyChannel` (email RFC-only or
  Iranian mobile normalised), `FormTimer` (encrypted time trap 3 s–24 h), `ContactRecipients` (privacy → DPO,
  partnership → partnership mailbox, fallback support), `SubmitContactMessage` (queued `ContactMessageReceived`
  notification after commit: topic + name + panel link only), `ChangeContactMessageStatus` (activity log),
  `ExportContactMessages` (CSV BOM, Jalali, formula guard).
- `/contact` GET + POST (`contact.store`, `throttle:contact` 3/min 10/day); honeypot/time-trap spam answered exactly
  like success; Persian validation; PRG to `#contact-form`. **Page is `no-store` (bypasses the page cache)** because
  the time trap needs a per-visitor render time. ContactPage + FAQPage JSON-LD; FAQ group `contact` rendered.
- Filament inbox (`ContactMessageResource`, Support + super-admin): tabs, topic filter, unread badge, open = read,
  mailto/tel reply, single + bulk status actions, delete, CSV export. Tests: 20.
- Diff body-above-footer 390 4.47% / 1440 4.46% (x-faq paragraph sizing + gap); whole page 8.26 / 9.77% (footer).
- Orchestrator: `OrganizationNode` drops `[placeholder]` values and invalid emails from contactPoint/legalName on
  every page (+ `OrganizationPlaceholdersTest`).
- Open: queue worker/cron needed for notifications (L10-01); message retention/prune policy undefined.

## L8-02 — Service worker (build-generated), offline page, two-tier update, install prompt
- `npm run build` = `vite build && node tools/build-sw.mjs` → generated `public/sw.js` (gitignored; never hand-edited):
  precache hashed build assets (minus admin theme/maps/json) + `/offline` + `public/icons/*`; pages network-first
  (3 s) → cache → `/offline`; `/build` + icons cache-first; `/media` SWR capped at 60. Never cached: admin (incl. custom
  `ADMIN_PATH`), Livewire/Filament, non-GET, Range, foreign origins, cart/checkout/order/done/booked, search,
  newsletter, `/pwa`, sw.js, manifest, previews, non-200/redirect/no-store/admin-CSP responses. Per-build caches,
  old ones deleted on activate; new SW waits for SKIP_WAITING.
- Build id `YYYYMMDDHHmmss-sha` (`BUILD_ID=` override) baked as `__BUILD_ID__` + `public/build/build-id.json`;
  `/pwa/version.json` (no-store, no session) via `App\Domain\Pwa\Version\{BuildInfo,AppVersion}` (caps a min newer
  than deployed). `resources/js/modules/pwa.js` on `<body data-module="pwa">` (layouts/app): registers in production,
  polls on load/focus/15 min, soft toast «نسخه جدید آماده است», forced blocking screen when `min_build_id` > build,
  install prompt (Chromium + iOS hint, not on first view, 30-day dismissal). `/offline` page (noindex).
- Filament PwaSettings «به‌روزرسانی اپ» section + confirmed «اجبار به به‌روزرسانی» action (activity log); DTO keys
  `min_build_id`, `update_message`. Tests: `ServiceWorkerTest` (6), `tests/js` 16. Verified offline/toast/forced in Chromium.
- Open: deploy package must ship `public/sw.js` + `public/build/build-id.json`; generated logo icons not precached;
  `pwa.js` 3.6 kB gz on every page.

## L5-03 — Place page with gallery, services, hours, reviews, LocalBusiness schema
- `ShowPlaceController` (`show`: slug-history 301 keeping query, 404 unknown/unpublished, `?page` review pagination;
  `storeReview`: `directory.place.review`, `throttle:3,10`, honeypot `company_url`, pending via `SubmitPlaceReview`,
  Persian errors). SeoManager/SchemaGraph injected per method (constructor injection goes stale across requests in tests).
- Views `pages/directory/show` + `components/directory/{section,header,gallery,hours,rating-summary,review-form,
  address,booking}`; `gallery.js` (lazy `<dialog>` lightbox with RTL keys; `<details>` fallback). Demo places noindex +
  «مجموعه نمونه»; demo reviews labelled; rating only from real approved reviews. "Open until" pill / today row only
  if stable for the whole page-cache TTL. No map embed: `map-place` illustration + geo:/Neshan/Balad/Google links.
  Pool health note for the pool category. Booking CTAs → `#book` aside with a marked `data-booking-slot` for L5-04.
- JSON-LD ItemPage + LocalBusiness subtype (PostalAddress, geo, openingHoursSpecification, priceRange, image,
  aggregateRating/reviews only if real) + BreadcrumbList. `PlaceTest` (14). `seo:audit` 0 errors.
- Diff 1440 14.04% (body 13.07%, header region 3.75%) / 390 46.73%: kit mobile gallery is one tile vs design's five
  stacked, booking widget is L5-04, no fake rating bars, 9 vs 8 amenities, NULL app links.
- Orchestrator: `RoutingTest` seeds `DirectorySeeder` for `directory.place`.
- Open: og:image empty for photo-less places; Neshan/Balad URL formats unverified.

## L5-05 — Business landing + multi-step join form + join-done page
- Migrations `2026_10_04_002150` (directory_join_requests: code, status pending, place/category/city/district, one
  contact person, address, landline, lat/lng, about, age_groups, amenity_ids, opening_hours, services, booking_mode,
  terms_accepted_at; no IP/UA) + `002160` (join request media). `app/Domain/Directory/Join` (model, enums AgeGroup /
  BookingMode / status, DTO, `JoinForm` limits from ini, `SubmitJoinRequest`: photos via `StoreMedia`, one transaction,
  6-digit code, queued `JoinRequestReceived` notification to the partnership mailbox).
- `JoinController` (business/create/store/done), `JoinRequest` form request (honeypot + `FormTimer`, Persian messages,
  Persian digits, field→step map), routes business/join/join.store (`throttle:directory-join` 3/10 min, 10/day)/join.done.
  Views business (FAQ group `directory-business`), join (one long form without JS; `stepper.js` turns it into steps,
  per-step validation, client photo checks, district filter by city), join-done. join + done noindex + no-store.
  Media usages registered for join photos. `JoinTest` (9).
- Diff (NULL app links): business 25.81/14.36%, join (step 1) 67/68% — step 2 (as designed) 33.9/21.59%, done
  14.89/21.92%: footer height, no fake rating, design's 390 CTA margin bug fixed, map picker → address + lat/lng text,
  no sample photos. `seo:audit` 0 errors.
- Open: `[زمان بررسی]` + partnership terms copy still placeholders (→ DirectorySettings); review/approve → Place is L5-06;
  pending photos are public under /media; dev `upload_max_filesize=2M` (check on cPanel); queue worker needed (L10-01).

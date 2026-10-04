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

# Security — threat model, controls, findings, go-live checklist (L9-04)

Scope: the Laravel marketing site in `laravel-site/` (public Blade site + Filament admin), as deployed on cPanel.
Audit date 2026-10-05 (security-auditor agent + manual review). Regression tests: `tests/Feature/Security/`.

## 1. Threat model

| Asset | Who wants it | Main threats |
|---|---|---|
| Customer / parent personal data: orders (name, mobile, address), bookings (parent name, child age, mobile), join requests, contact messages, newsletter emails | criminals, scrapers, curious insiders | IDOR on order/booking codes, admin account takeover, CSV exports, logs, backups |
| Admin panel (content, settings, head code, redirects) | defacers, SEO spammers | credential stuffing, missing authorization between roles, stored XSS, CSRF, open redirects |
| Site integrity + reputation (health content, brand domain) | spammers, phishers | XSS, uploaded malicious files, open redirects, abuse of forms for spam/mail flooding |
| Availability on shared hosting | bots | form/search flooding, expensive image uploads, cache stampedes |
| Server / secrets (`.env`, `APP_KEY`, DB) | anyone | `.env`/`storage`/`vendor` exposure on a `public_html` layout, PHP execution in upload folders, SSRF, unsafe unserialisation |

Out of scope / assumptions: the public site has **no user accounts**; payments are cash on delivery (no card data);
no third-party scripts or requests on public pages; the hosting account itself (cPanel login, SSH, backups) is
secured by the owner.

## 2. Controls per area

| Area | Control | Where |
|---|---|---|
| HTTP headers | Strict CSP, all `'self'`, no `unsafe-inline` on public pages (critical CSS allowed by sha256 only), relaxed but same-origin CSP on admin/Livewire; nosniff, XFO SAMEORIGIN, Referrer-Policy, Permissions-Policy, COOP; HSTS over https in production | `app/Http/Middleware/SecurityHeaders.php`, `config/pagecache.php` (`security.*`) |
| Static files | `nosniff` on every static file, sandboxing CSP on uploaded SVGs (`/media/*.svg`), no folder listings | `public/.htaccess` |
| Web root | dotfiles 403 (except `.well-known`), project folders/files 403 when they exist on disk (`vendor`, `storage`, `config`, `composer.json`, `*.sqlite`, `*.md`…), no PHP/script execution except `index.php` | `public/.htaccess` |
| Canonical host / https | one-hop 301 to `APP_URL` scheme + host | `app/Http/Middleware/CanonicalizeUrl.php`, `config/app.php` `canonical_redirect` |
| Proxies | `X-Forwarded-*` trusted only from `TRUSTED_PROXIES` (default none) | `config/app.php` `trusted_proxies`, `app/Providers/AppServiceProvider.php` |
| Sessions / cookies | `http_only`, `same_site=lax`, `secure` by default when `APP_URL` is https; only `ritme_cart_count` (an integer for the JS badge, never read by the server) is unencrypted | `config/session.php`, `bootstrap/app.php`, `app/Http/Controllers/Shop/CartController.php` |
| CSRF | on every POST except the view beacon (no state but a counter; Origin/Sec-Fetch checks) and the RFC 8058 one-click unsubscribe (token-authorised) | `routes/web.php`, `app/Http/Controllers/Blog/PostViewController.php` |
| Rate limits | named limiters per IP (+ per mobile): contact 3/min 10/day, newsletter 3/min 20/day, join 3/10 min 10/day, booking 5/10 min 20/day + 5/day per mobile, checkout 5/10 min 20/day + 10/day per mobile; numeric limits now **per route** (`ThrottlePerRoute`): reviews 3/10 min, search 30/min, view beacon 30/min, cart 60/min, order/booked/newsletter-token pages 20/min; admin login 5/min/IP (Filament) | `app/Providers/Domain/*ServiceProvider.php`, `app/Http/Middleware/ThrottlePerRoute.php`, controllers' `HasMiddleware` |
| Anti-spam | honeypots + encrypted `FormTimer` (3 s – 24 h), spam answered exactly like success | `app/Domain/Contact/Support/FormTimer.php`, `app/Http/Requests/*` |
| Unguessable codes | order/booking codes: 12 symbols from 31 (~59 bits, `random_int`); newsletter tokens 48 random chars; checkout one-time token 160 bits compared with `hash_equals`; order recipient rows only for the owning session | `app/Domain/Shop/Ordering/Support/{OrderCode,CheckoutSession}.php`, `app/Domain/Directory/Booking/Support/BookingCode.php` |
| Mass assignment | public flows build DTOs (`toSubmission()`), never `$request->all()`; `forceFill` only with server values; `User::$fillable` has no roles | `app/Http/Requests/*`, `app/Domain/*/Actions` |
| XSS | Blade escaping; every `{!! !!}` sink is sanitised server-side: post body/sources (`RichHtmlSanitizer`, symfony/html-sanitizer allow-list, own-media images only) on save **and** at render (article + admin preview), product descriptions, FAQ answers, enamad (`AllowedHtml`), head code (rebuilt same-origin `<meta>`/`<link>` only), JSON-LD (`JSON_HEX_TAG`), app-store links http(s) only | `app/Support/Html/RichHtmlSanitizer.php`, `app/Domain/Blog/Support/PostContent.php`, `app/Domain/Seo/Indexing/HeadCode.php`, `app/Domain/Settings/Data/AppLinksSettings.php` |
| Uploads | magic-byte sniffing (`finfo`), allow-list jpeg/png/webp/gif/avif (+ sanitised SVG), 15 MB / 8000 px, re-encode + EXIF strip, extension from the sniffed type, random directory; Livewire temp files on the private disk | `app/Domain/Media/Actions/StoreMedia.php`, `app/Domain/Media/Support/{MimeSniffer,SvgSanitizer}.php`, `config/media.php` |
| Admin auth | Filament login (Timebox, same answer for unknown/inactive), no password reset route, `canAccessPanel` = active + admin role, TOTP MFA (secret + recovery codes encrypted) for `ADMIN_MFA_ROLES`, inactivity timeout (`ADMIN_SESSION_TIMEOUT`, remember-me logins are sent back to the form), `AuthenticateSession`, path `ADMIN_PATH` | `app/Providers/Filament/AdminPanelProvider.php`, `app/Filament/Http/Middleware/*`, `app/Models/User.php`, `config/filament.php` |
| Admin authorization | deny-by-default policies on every resource model, `canAccess` + server checks on custom pages, SEO-only tab enforced server-side, exports/invoice `Gate::authorize`, activity log | `app/Filament/Auth/AdminAccess.php`, `app/Filament/Policies`, `app/Filament/Resources/**/*Policy.php` |
| Signed links | blog draft preview: temporary signed URL (24 h), noindex, no-store | `app/Filament/Resources/Blog/Posts/PostPreviewController.php` |
| Redirects | admin redirects validated (no scheme-less/self/loops); regex results confined to the own site / template host | `app/Domain/Seo/Redirects/Actions/SaveRedirect.php`, `app/Domain/Seo/Redirects/Data/RedirectMap.php` |
| Outbound requests | IndexNow only to the fixed `api.indexnow.org`, own-host URLs, production only; sitemap pings https to public host names only, no redirects | `app/Domain/Seo/Indexing/IndexNow/IndexNow.php`, `app/Domain/Seo/Indexing/Jobs/PingSitemaps.php`, `app/Support/Http/OutboundUrl.php` |
| Cache | only scalars/arrays are cached; `cache.serializable_classes = false` (no objects on unserialize); page cache bypasses admin, transactional pages, logged-in users, flash/errors | `config/cache.php`, `app/Support/Cache`, `app/Http/Middleware/PageCache.php` |
| Personal data | masked mobiles in lists/exports/SMS log, street address never on the order page, no IP/UA stored with messages/bookings/orders, 404 log without IP | `app/Domain/Directory/Booking/Support/MobileMask.php`, `app/Domain/Directory/Booking/Sms/LogSmsSender.php` |
| CSV exports | UTF-8 BOM, formula guard (`= + - @ TAB CR` → `'`) on every free-text cell: contact, orders, bookings, subscribers, bulk SEO, redirects | `app/Domain/*/Actions/Export*.php` |
| Error pages | 404/410/419/429 on the site shell, 500/503 standalone without DB/settings; no exception data unless `APP_DEBUG=true` | `resources/views/errors/*` |
| Dev-only routes | `/_preview/*`, `/_components` not registered in production | `routes/web.php` |

## 3. Findings

Severity: Critical / High / Medium / Low / Info. **No Critical or High findings.**

| # | Finding | Severity | Status | Fix / reason |
|---|---|---|---|---|
| F1 | All numeric `throttle:N,M` routes shared one counter per IP: 3 article views blocked the review form; a 1-min route set the window for the 10-min review limit | Medium | **fixed** | `ThrottlePerRoute` registered as the `throttle` alias (per route + client); named limiters unchanged |
| F2 | MFA required only for super-admin, while shop/directory/support roles read bulk personal data | Medium | **fixed (config) + go-live item** | `ADMIN_MFA_ROLES` env (`config/filament.php`); default stays `super-admin` because existing admin tests act as those roles without TOTP — production **must** set the PII roles (checklist) |
| F3 | Admin inactivity timeout (and MFA) bypassed by the remember-me cookie | Medium | **fixed** | `EnforceSessionTimeout` logs out logins restored via remember-me |
| F4 | Blind SSRF via admin-entered sitemap ping URLs (+ redirects followed) | Low | **fixed** | `OutboundUrl` guard on save and at send time; `withoutRedirecting()`. Residual: DNS names resolving to private IPs (admin-only, blind) — accepted |
| F5 | Regex redirect `/$1` could produce `//evil.com` (open redirect); absolute templates could gain user-info/other host | Low | **fixed** | `RedirectMap::confine()` |
| F6 | Cart refusal redirected to `url()->previous()` (Referer) | Low | **fixed** | always `route('shop.cart')` |
| F7 | Redirects CSV export without formula guard on the note | Low | **fixed** | `ExportRedirects::cell()` |
| F8 | App-store / web-app links not scheme-checked in the header + shop (`javascript:` possible from settings) | Info | **fixed** | `AppLinksSettings` keeps http(s) only |
| F9 | Code / token GET pages (order, booked, newsletter confirm/unsubscribe) not throttled | Info | **fixed** | 20/min per IP per route (`HasMiddleware`) |
| F10 | No trusted-proxy configuration | Info | **fixed** | `TRUSTED_PROXIES` (default none) |
| F11 | Session cookie not `secure` unless `SESSION_SECURE_COOKIE` was set | Low | **fixed** | defaults to true for an https `APP_URL` |
| F12 | Cache unserialisation allowed any class (gadget chains if a cache file/row is tampered with) | Low | **fixed** | `serializable_classes => false` |
| F13 | `.htaccess`: unpacked-project layout would expose `vendor`/`storage`/`composer.json`; stray `*.php` in `/media` would run; no nosniff on static files; `-Indexes` only with mod_negotiation | Low | **fixed** | new rules (verified on Apache 2.4.62 with mod_rewrite/headers) |
| F14 | Draft preview rendered stored HTML raw under the relaxed admin CSP | Low | **fixed** | re-sanitised at render |
| F15 | SMS log driver wrote order/booking codes (which open status pages) to the log | Info | **fixed** | codes redacted |
| F16 | Newsletter: an already-confirmed address answers faster than a new one (synchronous mail) → subscription status by timing | Low | deferred → L10-01 | make `ConfirmSubscriptionMail` `ShouldQueue` once the queue worker/cron exists (L10-01); response text is already identical |
| F17 | Join-request photos are public under `/media` before moderation; 8000 px decode on a public endpoint | Low | deferred | needs a private "pending" disk + promote-on-approve in `SubmitJoinRequest`/`ApproveJoinRequest` (follow-up task); throttled 3/10 min per IP, re-encoded, no script types |
| F18 | Per-mobile daily limits (booking, checkout) let someone lock a victim's number for a day | Low | accepted | protects the place/team from SMS + order floods; IP limits stay primary |
| F19 | Newsletter confirmation is a state-changing GET (mail scanners may confirm) | Low | deferred | needs a confirm button page (view change in `resources/views/pages/blog/newsletter/*`, outside this task) |
| F20 | Booked page shows parent name + child age to anyone with the code (orders gate recipient rows by session) | Info | accepted | ~59-bit code, now throttled; owner gating needs a session marker at booking time + view change |
| F21 | Forged FormTimer token can be reused within 24 h | Info | accepted | per-IP limits cap it; tokens are MAC'd |
| F22 | `/_preview` and `/_components` reachable on non-production (staging) | Info | accepted | static demo views only; staging is behind Basic auth |
| F23 | No boot-time refusal of `APP_DEBUG=true` in production | Info | accepted | go-live checklist + L10 deploy script check |

## 4. Dependency audits (2026-10-05)

| Command | Result |
|---|---|
| `composer audit` | No security vulnerability advisories found. |
| `npm audit --omit=dev` | found 0 vulnerabilities |

Re-run both before every release; never auto-upgrade a major version for an advisory without the test suite.

## 5. Go-live checklist

Environment (`.env` on the server, never in the package or git):

- [ ] `APP_ENV=production`, **`APP_DEBUG=false`**, `LOG_LEVEL=warning` (no stack traces or SQL in responses).
- [ ] **`APP_KEY`** generated on the server once (`php artisan key:generate --show` → `.env`); keep a copy in the
      password manager. Rotating it invalidates sessions, FormTimer tokens and encrypted TOTP secrets (use
      `APP_PREVIOUS_KEYS` for a rotation).
- [ ] `APP_URL=https://<canonical host>` and **`APP_CANONICAL_REDIRECT=true`** (http → https and www/non-www in one hop).
- [ ] HTTPS certificate (AutoSSL) valid for every host that 301s; **HSTS** is sent automatically in production over
      https (`SECURITY_HSTS` to override, `SECURITY_HSTS_SUBDOMAINS=true` only if every subdomain is https).
- [ ] `TRUSTED_PROXIES` empty on plain cPanel; set to the proxy/CDN ranges only if one sits in front (otherwise every
      rate limit sees the proxy IP).
- [ ] Session cookies: `SESSION_SECURE_COOKIE` unset (defaults to true for https) or `true`; `SESSION_HTTP_ONLY=true`;
      `SESSION_SAME_SITE=lax`; `SESSION_DRIVER=file` (or database); `SESSION_DOMAIN` unset.
- [ ] **`ADMIN_MFA_ROLES=super-admin,shop-manager,directory-manager,support`** (or all six roles); every admin enrols TOTP
      on first login. Consider a non-default `ADMIN_PATH`; `ADMIN_SESSION_TIMEOUT` ≤ 60.
- [ ] Mail + SMS drivers configured (the log SMS driver is for development); queue worker cron (`schedule:run`) runs.

Files and permissions (cPanel):

- [ ] Application **outside** the web root (e.g. `~/ritme/`), only the contents of `public/` in `public_html`
      (`index.php` pointing at `../ritme/`). Never unpack the whole project into `public_html` (the `.htaccess` blocks
      known paths as a second line of defence only).
- [ ] `storage/` and `database/*.sqlite` are not reachable from the web; `curl -I https://<host>/.env`,
      `/storage/logs/laravel.log`, `/vendor/autoload.php`, `/composer.json`, `/media/x.php` → 403/404.
- [ ] Permissions: folders `755`, files `644`, `.env` `600` (or `640`), `storage/` and `bootstrap/cache/` writable
      by the account user only; no `777`.
- [ ] `php artisan config:cache route:cache view:cache` after each deploy, then `php artisan cache:ns bump pages`.
- [ ] Production DB user has only the privileges of its own database; DB not reachable from outside (`localhost`).
- [ ] `php artisan admin:create` for the first super-admin; no default/seeded users exist.

Verification after deploy:

- [ ] `curl -sI https://<host>/` shows `Content-Security-Policy` (no `unsafe-inline`), `Strict-Transport-Security`,
      `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`; no `X-Powered-By`.
- [ ] `http://` and the non-canonical host answer one 301 to `https://<host>/…`.
- [ ] A forced 500 (e.g. temporarily broken DB password in a staging copy) shows the standalone error page only.
- [ ] `composer audit` and `npm audit --omit=dev` clean on the release commit.

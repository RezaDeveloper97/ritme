---
id: L1-07
title: Performance core: page cache, HTTP caching, minify, security headers, .htaccess
milestone: L1
type: backend
status: done
depends_on: [L1-05,L0-03]
parallel_group: L1-E
touches: [app/Http/Middleware,public/.htaccess,config/pagecache.php,tests/Feature/Http]
skills: []
verify: composer verify
---

# L1-07 — Performance core: page cache, HTTP caching, minify, security headers, .htaccess

## Why
GTmetrix rewards low TTFB, compression, long-lived caching of static assets and small HTML. On shared hosting the
biggest TTFB win is not booting the DB for anonymous page views.

## Scope
- `PageCache` middleware (guest GET/HEAD, 200 HTML only): cache-aside keyed by `pages` namespace + normalised URL
  (whitelisted query params only); bypass when session has flash/errors, cart cookie present on shop pages, auth
  user, `Cache-Control: no-cache`, admin paths, `noindex` transactional pages. `X-Page-Cache: HIT|MISS` header.
  Namespaces bumped by every content observer, settings and SEO changes. CSRF-bearing forms on cached pages get the
  token refreshed via a tiny endpoint or forms post to routes that don't require a pre-rendered token — choose,
  document, test.
- `HttpCacheHeaders`: `Cache-Control: public, max-age=0, s-maxage=…, stale-while-revalidate` for cacheable HTML,
  weak ETag + 304 support.
- `MinifyHtml` middleware (safe: keeps `pre`, `textarea`, `script`, `style`, conditional comments; collapses
  inter-tag whitespace) — measure gain, keep only if > 5%.
- `SecurityHeaders`: strict CSP with **no external origins** (`default-src 'self'`; `img-src 'self' data:`;
  `script-src 'self'` + nonce for any inline bootstrap; `style-src 'self'`), HSTS (prod), `X-Content-Type-Options`,
  `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy`, `X-Frame-Options: SAMEORIGIN`.
- `public/.htaccess` (Apache/LiteSpeed on cPanel): Laravel rewrite, force https (commented toggle), `mod_deflate`
  + `mod_brotli` when present, `mod_expires` + `Cache-Control: public, max-age=31536000, immutable` for `/build/*`,
  fonts, `/media/*`; short cache for `sw.js`, `manifest.webmanifest`; deny dotfiles; correct MIME for woff2/avif/webp/
  webmanifest; `Vary: Accept-Encoding`.
- Tests: hit/miss/bypass rules, headers present, CSP has no external hosts.

## Out of scope
- Critical CSS (L9-01).

## Acceptance
- Second anonymous request to `/` serves from cache with 0 DB queries; headers verified by tests; `.htaccess`
  commented line by line.

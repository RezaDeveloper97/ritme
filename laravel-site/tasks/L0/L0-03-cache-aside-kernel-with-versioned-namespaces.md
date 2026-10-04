---
id: L0-03
title: Cache-aside kernel with versioned namespaces
milestone: L0
type: backend
status: done
depends_on: [L0-02]
parallel_group: L0-A
touches: [app/Support/Cache,config/cacheaside.php,app/Console/Commands/CacheNamespaces.php,tests/Unit/Support/Cache]
skills: []
verify: composer verify
---

# L0-03 — Cache-aside kernel with versioned namespaces

## Why
The user asked for heavy cache-aside use. cPanel usually means the `file` or `database` cache driver, which has no
tags — so invalidation must be driver-agnostic.

## Scope
- `CacheAside` service (injectable, no facade in domain code): `remember(CacheKey $key, int|\DateInterval $ttl,
  Closure $loader)`, `forget`, `rememberForever`; TTL jitter (±10%) to avoid synchronized expiry; optional stampede
  protection with `Cache::lock` when the store supports locks (fallback: no lock); never caches `null` unless asked.
- `CacheNamespace` / `NamespaceVersions`: keys look like `rt:{ns}:v{version}:{key}`; `bump('blog')` increments the
  version (old entries expire naturally). Namespaces declared in `config/cacheaside.php` with default TTLs
  (`settings`, `seo`, `pages`, `blog`, `faq`, `directory`, `shop`, `media`, `sitemap`, `menu`).
- `CachedRepository` abstract decorator helper + `BumpsCacheNamespaces` model trait/observer helper: on
  saved/deleted/restored bump the configured namespaces (and `pages` for the full-page cache).
- `php artisan cache:ns {list|bump ns|bump-all}`.
- Unit tests on `array` and `file` stores: hit/miss, bump invalidates, jitter bounds, lock path.

## Out of scope
- Full-page cache (L1-07). Specific repositories (each domain task).

## Acceptance
- 100% of the kernel covered by tests; `config/cacheaside.php` documented inline; `composer verify` green.

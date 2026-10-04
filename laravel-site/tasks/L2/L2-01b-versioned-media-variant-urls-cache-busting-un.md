---
id: L2-01b
title: Versioned media variant URLs (cache-busting under immutable caching)
milestone: L2
type: backend
status: todo
depends_on: [L2-01,L1-07]
parallel_group: L2-A
touches: [app/Domain/Media,tests/Feature/Media,tests/Unit/Media]
skills: []
verify: composer verify
---

# L2-01b — Versioned media variant URLs (cache-busting under immutable caching)

## Why
`.htaccess` (L1-07) serves `/media/*` as `public, max-age=31536000, immutable`, but `GenerateMediaVariants` rewrites
variants under the same filename (e.g. after a focal-point change via `UpdateMediaDetails`). Browsers and proxies keep
the old crop forever.

## Scope
- Give every generated variant set a version (e.g. a short content hash or a `variants_version` counter stored on the
  media row) that appears in the URL path or filename, so a regenerate produces new URLs.
- `MediaData::url()/srcset()`, `Picture::mediaUrl()` and `MediaOgImageResolver` use the versioned URLs.
- Old variant files are deleted after the new set is written; cache namespaces `media`, `seo`, `pages` bumped.

## Out of scope
- CDN purging, changes to `.htaccess`.

## Acceptance
- Regenerating thumb/og after a focal change yields different URLs; old files removed; tests cover it.
- `composer verify` green.

---
id: L4-03b
title: Article view beacon on page-cache hits, share copy module, blog UI copy to lang
milestone: L4
type: fullstack
status: done
depends_on: [L4-03]
parallel_group: L4-B
touches: [resources/js/modules/share.js,resources/js/modules/view-beacon.js,app/Http/Controllers/Blog/PostViewController.php,resources/views/components/blog,lang/fa/blog.php,tests/Feature/Blog/ViewBeaconTest.php]
skills: []
verify: composer verify
---

# L4-03b — Article view beacon on page-cache hits, share copy module, blog UI copy to lang

## Why
L4-03 counts views only on page-cache MISS (`RecordPostView` runs in the controller), so counts are a lower bound;
the share "copy link" needs a JS module; blog UI strings are hard-coded in the components.

## Scope
- Cookie-free, CSRF-safe view beacon: lazy `data-module="view-beacon"` sends `navigator.sendBeacon` POST to a
  throttled route (no session middleware, ignores bots / `Save-Data`, one per post per tab session via sessionStorage);
  controller calls `RecordPostView`. Stop counting in `ShowPostController` to avoid double counts. Route line in
  `routes/web.php` allowed.
- `share` data-module: copy-to-clipboard with Persian status text, graceful without JS.
- Move hard-coded Persian strings in `components/blog/*` to `lang/fa/blog.php`.

## Acceptance
- A cached (HIT) article still increments the counter once per visit; bots ignored; tests.
- `composer verify` green; no external requests; CSP still strict (no inline JS).

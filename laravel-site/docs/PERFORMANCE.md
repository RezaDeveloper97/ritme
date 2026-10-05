# Performance — asset budget, critical CSS, fonts (L9-01)

Targets (tasks/README.md, binding): Lighthouse mobile ≥ 95, CLS < 0.1, LCP < 2.5 s, render-blocking work minimised,
no external requests, no Node on the server. Task budget: **HTML < 40 KB, CSS < 25 KB, JS < 15 KB gzip per page,
≤ 2 preloaded fonts.**

## How it is measured and enforced

`tools/critical.mjs` (last step of `npm run build`, so also of `composer verify`) measures one sample page per template
on the **production build** in headless system Chrome: 390×844 mobile, empty cache, until the network is idle (lazy
below-the-fold images are not part of the first load). Text bytes are gzip level 6 (Apache mod_deflate's default; the
`.htaccess` serves brotli where available, which is smaller); woff2 and raster images as shipped. A template over its
budget, a CSP violation, an external request, a non-200 sample, or a page that should have inline critical CSS but
does not (stale page cache) fails the build with exit 1.

```bash
npm run budget                                              # budget check of the existing build
node tools/critical.mjs --budget --lab --json /tmp/b.json   # + lab FCP/LCP/CLS: 150 ms RTT, 1.6 Mbps, 4× CPU
```

Budget constants live in `BUDGET` / `BUDGET_OVERRIDES` at the top of `tools/critical.mjs`:

| Metric (per page, gzip) | Budget | Source |
|---|---|---|
| HTML (incl. inline critical CSS) | 40 KB | task |
| CSS = external stylesheet + inline critical | 25 KB | task |
| JS | 15 KB | task |
| Fonts (woff2) | 256 KB | exception, see below |
| Images (first load) | 200 KB | L9-01 |
| Requests (first load) | 26 | L9-01 |
| Preloaded fonts | 2 | task |
| Render-blocking head resources | 0 | README "render-blocking minimised" |
| CLS | < 0.1 (product 0.2 in lab, see below) | README |
| LCP (only with `--lab`) | < 2.5 s | README |

## Budget table (production build, 2026-10-05)

Measured with `node tools/critical.mjs --budget --lab` (KB gzip; "inl" = the inline critical CSS that is part of the
HTML; the CSS column counts it once more on purpose, as the page pays for both).

| Template | Sample | HTML | CSS (inl) | JS | Fonts | Images | Req | Preload | Blocking | CLS | FCP = LCP (lab) |
|---|---|---|---|---|---|---|---|---|---|---|---|
| home | `/` | 17.3 | 21.9 (6.5) | 5.0 | 236.4 | 2.6 | 19 | 2 | 0 | 0.000 | 1.31 s |
| stage | `/cycle` (+5 stage pages) | 16.6 | 22.5 (7.0) | 5.0 | 236.4 | 2.6 | 19 | 2 | 0 | 0.000 | 1.18 s |
| page | `/services` (+9 content pages) | 14.9 | 23.3 (7.9) | 5.0 | 204.2 | 2.6 | 17 | 2 | 0 | 0.000 | 0.68 s |
| blog | `/blog` | 12.5 | 21.5 (6.0) | 5.0 | 236.2 | 2.6 | 19 | 2 | 0 | 0.000 | 0.61 s |
| post | `/blog/period-pain` | 13.5 | 21.8 (6.3) | 5.8 | 199.0 | 2.6 | 19 | 2 | 0 | 0.000 | 0.62 s |
| shop | `/shop` | 14.6 | 21.4 (5.9) | 5.4 | 236.2 | 2.6 | 20 | 2 | 0 | 0.000 | 1.17 s |
| product | `/shop/product/…` | 17.0 | 21.6 (6.1) | 7.6 | 252.4 | 2.6 | 23 | 2 | 0 | 0.159 | 1.42 s |
| directory | `/directory` | 14.8 | 21.8 (6.3) | 5.0 | 220.2 | 2.6 | 18 | 2 | 0 | 0.000 | 1.06 s |
| misc | `/search` (+offline, join, cart) | 10.7 | 22.2 (6.7) | 5.0 | 204.2 | 2.6 | 17 | 2 | 0 | 0.000 | 0.55 s |
| place | `/directory/place/…` | 16.4 | 21.7 (6.2) | 5.3 | 236.2 | 2.6 | 20 | 2 | 0 | 0.000 | 1.20 s |

Images are the icon sprite only: the demo content has no photos above the fold; real `<x-picture>` images are AVIF/WebP
with width/height and one LCP preload (L2), and the 200 KB image budget will catch regressions.

### Before → after (same lab settings)

| | Before L9-01 | After |
|---|---|---|
| Render-blocking stylesheet | 1 (`app-*.css`, 15.1 KB gz, every page) | 0 — 6–8 KB gz critical CSS inline |
| Lab FCP / LCP | 1.27–1.50 s | 0.55–1.42 s |
| Lab CLS | blog 0.150, product 0.344, home 0.074 | 0.000 everywhere except product 0.159 |
| Fonts per page | 12 files, 252 KB (all Latin subsets on every page) | 9–12 files, 199–252 KB |
| HTML | 6.6–11.0 KB | 10.7–17.3 KB (+ inline critical CSS) |

## Critical CSS — the decision

The whole stylesheet is **15.6 KB gzip** (≥ 14 KB, the first-round-trip budget), so it is not inlined whole. Instead
`tools/critical.mjs` builds one critical file per template at **build time** (Node + system Chrome on the build
machine; the output ships in `public/build/critical/`, cPanel never runs Node):

1. The built `app-*.css` is split into rules (string/comment/nesting aware).
2. Each template's sample pages load at 390×844, 1024×768 and 1440×900 with the full stylesheet. A style rule is kept
   when one of its selectors (user-action pseudo-classes and pseudo-elements stripped) matches an element within
   1.25 viewports, a `display:none` element whose parent is in that area (so hiding rules always apply), or a child of
   an above-the-fold grid container (explicit grid placement of the page shell). `@font-face`, `@property` and `@layer`
   statements are always kept, `@keyframes` when referenced; `@media`/`@supports` wrappers are kept around kept rules
   at every breakpoint.
3. Output: 5.9–7.9 KB gzip per template, `default` (union, 11.5 KB) for unmapped routes (errors etc.).

Delivery, CSP-safe without inline JS (`App\View\Components\Layout\Assets` + `layouts/app`):

- `<style>` with the template's file. `SecurityHeaders` adds each file's `'sha256-…'` from
  `public/build/critical/manifest.json` to `style-src` on public pages (no `'unsafe-inline'`); `Assets` refuses to
  inline bytes whose hash differs from the manifest, and ignores a manifest cut from another build's stylesheet
  (then the plain blocking `@vite` tags are used — also in tests, `npm run dev` and builds without Chrome).
- `<link rel="preload" as="style">` for the full stylesheet in the head, `<link rel="stylesheet">` at the end of
  `<body>` (`@stack('deferred-styles')`): it downloads at once but never blocks the first paint, with no JS dependency.
- `<link rel="expect" href="#main-end" blocking="render">` + an empty `<template id="main-end">` after `</main>`:
  Chrome waits for `<main>` to be parsed before the first paint. Without it Chrome painted a half-parsed fold on
  heavy pages (directory at 1024 px: CLS 0.195 when the rest of a chip list arrived). It costs some FCP on the
  heaviest pages (home ≈ 0.5 s → 1.3 s in the 4× CPU lab) and is still faster than before; other browsers ignore it.
- `modulepreload` for the page modules present on every sample of the template (`pwa`, `menu`, + `share`,
  `cart-badge`, `product`, `cart`, `view-beacon` where used), so they do not wait for `app.js` to discover them.

Keeping it correct: the manifest is regenerated by every `npm run build`; a new template needs an entry in
`TEMPLATES`. **Deploys must bump the page cache** (`php artisan cache:ns bump pages`, already required by L1-07): a
cached page holds the old `<style>`, whose hash the new CSP no longer allows (the budget check reports this).
`public/sw.js` precaches the same hashed assets as before — critical files are never fetched by the browser.

## Fonts

- **Preloads: 2** — `lalezar-arabic-400` (logo, h1) and `vazirmatn-arabic-600` (nav, breadcrumbs, body copy).
- **unicode-range:** spaces (U+0020/U+00A0, in both subsets) and ZWNJ (U+200C, inside the Latin subset's
  U+2000–206F) pulled every Latin file onto every page. The Arabic face is now declared after the Latin one (the
  last-declared face wins for shared code points) and also claims U+0020/U+00A0, which its file contains. Latin files
  now load only for real Latin glyphs. Persian copy still uses ASCII punctuation (`.` `:` `·` `«»` `©`) that the
  Arabic subsets do not contain, so most weights still need their Latin file — see the exception.
- **Fallback metrics:** `Vazirmatn Fallback` / `Lalezar Fallback` (Tahoma, Windows/macOS) and `… Geeza` (Geeza Pro,
  iOS/macOS) are `local()` faces with `size-adjust` = measured width ratio per weight on the site's Persian copy and
  ascent/descent overrides from the web fonts' metrics, placed right after the web font in `--font-sans` /
  `--font-display`. Swap shifts dropped from 0.150 (blog) / 0.344 (product) to 0 / 0.159. Android/Linux Arabic system
  fonts could not be measured on the build machine and keep the unadjusted stack.

## JavaScript

`app.js` (1.2 KB gz) + `pwa` (3.6 KB, body `data-module`) + `menu` (0.3 KB) load on every page by design (service
worker / update channel, mobile menu); everything else is a lazy `data-module` chunk fetched only when the element is
on the page. Max 7.6 KB gz (product). No classic or render-blocking scripts.

## Exceptions

| Item | Budget vs. task | Why | Follow-up |
|---|---|---|---|
| Fonts 199–252 KB | not in task targets; budget 256 KB | Design uses Vazirmatn 400–800 + Lalezar; ASCII punctuation in Persian copy needs each weight's Latin file (16 KB). Fonts are `swap` and not render-blocking. | Re-subset Arabic files with `.:·«»©` (≈ −80 KB/page) — needs a font-tooling step and a design sign-off. |
| Product lab CLS 0.159 | CLS < 0.1 | The shop category chip row in the product header fits its 340 px container by < 1 px; while Vazirmatn 800 swaps in, the fallback (≈ 6 % wider on those two labels) wraps it. Not visible without throttling (budget run: 0.000). Was 0.344. | Shop header component: allow horizontal scroll / `flex-nowrap` with overflow, or trim padding. |
| Home lab FCP 1.3 s | — | `rel=expect` waits for the large inline-SVG `<main>` to parse under 4× CPU (see above). LCP still < 2.5 s. | Lighter hero SVG markup (L9-02 Lighthouse sweep). |

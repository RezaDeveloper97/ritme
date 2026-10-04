---
id: L1-02
title: Base layout, header/nav, mobile menu, footer, breadcrumbs
milestone: L1
type: frontend
status: todo
depends_on: [L0-06,L0-07,L1-01]
parallel_group: L1-B
touches: [resources/views/layouts,resources/views/components/layout,resources/views/components/ui,resources/js/modules/menu.js,app/View,app/Domain/Content]
skills: []
verify: composer verify
---

# L1-02 — Base layout, header/nav, mobile menu, footer, breadcrumbs

## Why
Every page shares the shell; getting it right once (semantics, a11y, performance) multiplies across 29 pages.

## Scope
- `layouts/app.blade.php`: `<html lang="fa" dir="rtl">`, `<meta charset>` first, viewport, `@stack('head')`,
  `<x-seo.head/>` slot (filled by L1-03), font `preload` for the 1–2 weights above the fold, `@vite`, theme-color,
  skip-to-content link, `<main id="main">`, footer, `@stack('scripts')` at end (all JS `type=module`/deferred).
- `Content` context: `StaticPage` enum/registry (route name, title key, nav group, breadcrumb parent, header variant
  dark|light, sitemap priority/changefreq) — the single source for nav, breadcrumbs, sitemap and admin SEO list.
- Components (from `docs/AUDIT.md`): `x-layout.header :variant`, `x-layout.nav` (active state from current route,
  `aria-current="page"`), `x-layout.mobile-menu` (button with `aria-expanded/controls`, `hidden` attribute, Esc closes,
  focus returns), `x-layout.footer` (links from registry + settings: socials, enamad, app links), `x-ui.breadcrumbs`
  (visual + feeds BreadcrumbList JSON-LD), `x-ui.button`, `x-ui.badge`, `x-ui.section`, `x-ui.container`.
- Menu JS module (< 1 KB) loaded via `data-module="menu"`; works without JS (nav links in footer remain).
- Header/footer HTML is cached as a fragment (cache-aside `menu` ns) keyed by variant + active route.
- Visual fidelity vs `design/html/index.html` header/footer at 390 and 1440 (`tools/shot.mjs`).

## Out of scope
- Page bodies, SEO tags (L1-03).

## Acceptance
- A blank page using the layout matches the design header/footer (diff < 3% on those regions); keyboard-only menu
  works; no inline styles; `composer verify` green.

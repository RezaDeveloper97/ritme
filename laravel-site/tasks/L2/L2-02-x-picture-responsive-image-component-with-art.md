---
id: L2-02
title: <x-picture> responsive image component with art direction
milestone: L2
type: frontend
status: done
depends_on: [L2-01,L1-02]
parallel_group: L2-B
touches: [app/View/Components/Picture.php,resources/views/components/picture.blade.php,tests/Feature/View/PictureTest.php]
skills: []
verify: composer verify
---

# L2-02 — <x-picture> responsive image component with art direction

## Why
Optimised files only help if markup lets the browser pick the right one without layout shift.

## Scope
- `<x-picture :media="$m" :mobile="$m2" sizes="(max-width: 768px) 100vw, 50vw" priority />`:
  `<picture>` with `<source type="image/avif" srcset sizes>`, `<source type="image/webp" …>`, optional
  `media="(max-width: 767px)"` sources from a separate mobile image (art direction), fallback `<img>` with `src`,
  `srcset`, `width`, `height`, `alt` (required — empty string only with explicit `decorative`), `decoding="async"`,
  `loading="lazy"` by default.
- `priority` → `loading="eager"` + `fetchpriority="high"` + pushes `<link rel="preload" as="image" imagesrcset
  imagesizes type>` into the head stack (LCP image).
- Background placeholder: `dominant_color` as inline CSS custom property via a class + `style` only for the color
  variable (allowed by CSP `style-src` — or use a data attribute + tiny CSS; choose what keeps CSP strict).
- Helper `media_url($media, 'og')` for OG images.
- Tests: markup per case (no variants, avif off, art direction, priority preload).

## Acceptance
- Lighthouse "properly size images", "next-gen formats", "explicit width/height" pass on a sample page.

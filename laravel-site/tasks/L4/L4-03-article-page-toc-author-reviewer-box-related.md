---
id: L4-03
title: Article page: TOC, author/reviewer box, related posts, BlogPosting schema
milestone: L4
type: frontend
status: done
depends_on: [L4-02]
parallel_group: L4-B
touches: [resources/views/pages/blog/show.blade.php,resources/views/components/blog,app/Http/Controllers/Blog/ShowPostController.php,app/Domain/Blog/Rendering,resources/css/prose.css,tests/Feature/Blog/ArticleTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design article.html --route /blog/dard-period
---

# L4-03 — Article page: TOC, author/reviewer box, related posts, BlogPosting schema

## Scope
- `/blog/{slug}` (old slugs 301 via slug history); design `article.html` layout; `.rt-prose` typography tuned for
  Persian (line-height, ZWNJ-friendly, numerals).
- Content renderer: replace `<img data-media-id>` with `<x-picture>` (lazy, sizes for the content column), first
  image not lazy when it's the LCP, tables scroll-wrapped, auto TOC (h2/h3) with anchor links.
- Author box + «بازبینی پزشکی: …» with date; medical disclaimer component; «در ریتمی چطور ثبت کنیم» app CTA.
- Related posts (cached), prev/next in category, breadcrumbs (خانه › مجله › دسته › عنوان).
- SEO: `BlogPosting` (headline ≤ 110, image 1200 wide, datePublished, dateModified = updated_content_at, author,
  reviewedBy, publisher, mainEntityOfPage, articleSection, keywords), OG `article:*` tags, `max-image-preview:large`.
- Share links as plain URLs (Telegram, WhatsApp, X, copy-link button) — no third-party scripts.

## Acceptance
- Diff < 3% vs `article.html`; schema valid; `seo:audit` passes; tests green.

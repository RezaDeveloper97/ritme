---
id: L4-01
title: Blog context: posts, categories, tags, authors, medical reviewers
milestone: L4
type: backend
status: done
depends_on: [L2-01,L1-03]
parallel_group: L4-A
touches: [app/Domain/Blog,database/migrations,database/factories,database/seeders/BlogSeeder.php,app/Support/Html,tests/Feature/Blog,tests/Unit/Blog]
skills: []
verify: composer verify
---

# L4-01 — Blog context: posts, categories, tags, authors, medical reviewers

## Why
The magazine is the long-term organic-traffic engine. Health content is YMYL: authorship and medical review must be
first-class data for E-E-A-T.

## Scope
- Models: `Post` (title, slug unique + slug history, excerpt, body HTML, cover media + mobile cover, category,
  tags, author, reviewer, life_stage enum, status draft|scheduled|published|archived, published_at, updated_content_at,
  reading_time (Persian word count), is_featured, views counter (cheap, batched)), `Category` (tree-light: parent),
  `Tag`, `Author` (name, slug, bio, avatar media, credentials, sameAs links, is_medical_reviewer).
- Body sanitising on save (`symfony/html-sanitizer` allowlist: headings, lists, links, tables, figure, img from own
  media only, blockquote) + heading ids for TOC + `rel="noopener"` external links.
- Repositories + cached decorators (`blog` ns), query objects (`LatestPosts`, `PostsByCategory`, `RelatedPosts`
  by shared tags/category with recency weighting), observers bump `blog`, `pages`, `sitemap`.
- Scheduler: publish scheduled posts every minute (works through cPanel cron).
- Seed the design article («درد پریود…») and the blog-list samples as **demo** seeders (separate from production
  seeders).
- Sitemap provider (`posts`, `blog-categories`) with lastmod + image.

## Acceptance
- Factories + tests for slug history, sanitiser, scheduling, related posts; `composer verify` green.

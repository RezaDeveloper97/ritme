---
id: L4-05
title: Admin: posts editor with SEO tab, categories, tags, authors, newsletter
milestone: L4
type: admin
status: done
depends_on: [L4-01,L2-03,L1-08]
parallel_group: L4-D
touches: [app/Filament/Resources/Blog,app/Filament/Resources/Newsletter,app/Filament/Components/Seo,tests/Feature/Admin/BlogAdminTest.php]
skills: []
verify: composer verify
---

# L4-05 — Admin: posts editor with SEO tab, categories, tags, authors, newsletter

## Scope
- Post resource: rich editor (Filament RichEditor, local assets) with media-library images (stores `data-media-id`),
  status/schedule, featured, author + reviewer, life stage, categories/tags (create inline), cover + mobile cover.
- **Reusable `SeoFields` component** (used by every content resource later): meta title/description with live
  character + pixel-width counters, live Google SERP preview (desktop/mobile), OG preview card, focus keyword,
  canonical override, robots toggles, sitemap include/priority, OG image picker. (Analyser scoring comes in L7-02.)
- Draft preview via signed URL (noindex), duplicate post, revision trail (activity log diff), bulk publish/unpublish.
- Category/Tag/Author resources (author credentials + sameAs), Newsletter subscribers list + export.
- Policies: editor writes, seo-manager edits SEO fields only.

## Acceptance
- Create → schedule → auto-publish → visible on `/blog` with correct meta; tests green.

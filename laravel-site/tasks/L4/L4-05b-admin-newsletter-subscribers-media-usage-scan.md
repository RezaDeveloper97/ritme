---
id: L4-05b
title: Admin newsletter subscribers + media usage scan of rich bodies
milestone: L4
type: admin
status: done
depends_on: [L4-02,L4-05]
parallel_group: L4-D
touches: [app/Filament/Resources/Newsletter,app/Domain/Media/Actions/FindMediaUsages.php,app/Providers/Domain/BlogServiceProvider.php,tests/Feature/Admin/NewsletterAdminTest.php,tests/Feature/Admin/MediaTest.php]
skills: []
verify: composer verify
---

# L4-05b — Admin newsletter subscribers + media usage scan of rich bodies

## Why
L4-05 shipped the blog admin but the newsletter subscribers list was blocked (no Newsletter model/table until L4-02).
It also found that "delete unused media" only checks registered FK columns, so images used only inside a post body
(`data-media-id` in `blog_posts.body`) count as unused and can be deleted.

## Scope
- `app/Filament/Resources/Newsletter`: subscribers list (search, status filter, confirmed/unsubscribed), CSV export
  (UTF-8 BOM for Excel, Persian headers), deny-by-default policy (Editor + super-admin), activity log on export.
- `FindMediaUsages`: support registering rich-HTML columns (e.g. `FindMediaUsages::html('blog_posts', 'body', …)`)
  that are scanned for `data-media-id="N"`; register `blog_posts.body` in `BlogServiceProvider`.

## Out of scope
- Sending newsletters / campaigns.

## Acceptance
- Subscribers visible + exportable by permitted roles, 403 for others; tests.
- A media id referenced only in a post body is reported as used and survives bulk "delete unused"; test.
- `composer verify` green.

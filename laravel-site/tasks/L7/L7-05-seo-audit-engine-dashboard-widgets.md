---
id: L7-05
title: SEO audit engine + dashboard widgets
milestone: L7
type: admin
status: todo
depends_on: [L7-02,L7-03,L5-06,L6-06]
parallel_group: L7-D
touches: [app/Domain/Seo/Audit,app/Console/Commands/SeoAudit.php,app/Filament/Widgets/Seo,app/Filament/Pages/Seo/AuditReport.php,database/migrations,tests/Feature/Seo/AuditTest.php]
skills: []
verify: composer verify && php artisan seo:audit
---

# L7-05 — SEO audit engine + dashboard widgets

## Scope
- Extend `seo:audit` into an engine crawling every indexable URL in-process (registry + sitemap providers): title /
  description length + uniqueness, single h1, canonical self & absolute, robots consistency with sitemap, OG
  complete + OG image reachable, JSON-LD parse + required props, images alt/width/height/lazy (LCP not lazy),
  broken internal links (incl. in post bodies), orphan pages (no internal inbound links), redirect chains, thin
  content, page weight (HTML bytes) and number of requests from the HTML.
- Results stored (`seo_audit_runs`, `seo_audit_issues` with severity + fix URL), scheduled weekly + "run now".
- Dashboard widgets: SEO health score, issues by severity, top 404s, zero-result searches, content needing work
  (analyser score < 60), pages missing OG image, last audit time.

## Acceptance
- Audit on a clean seed reports 0 errors; injected faults detected in tests; widgets render.

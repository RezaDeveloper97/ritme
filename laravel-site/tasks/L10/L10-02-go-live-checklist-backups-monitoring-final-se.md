---
id: L10-02
title: Go-live checklist: backups, monitoring, final SEO/perf verification
milestone: L10
type: release
status: done
depends_on: [L10-01]
parallel_group: L10-B
touches: [docs/GO-LIVE.md,config/backup.php,app/Console/Kernel.php,routes/console.php]
skills: []
verify: composer verify && php artisan seo:audit
---

# L10-02 — Go-live checklist: backups, monitoring, final SEO/perf verification

## Scope
- Backups (spatie/laravel-backup: DB + `public/media` to local disk, retention, admin download page; optional
  remote target later), `/up` health route, daily log rotation, mail on job failures.
- Production config check command (`app:doctor`): debug off, https, cache/session drivers, queue processed in last
  5 minutes, writable paths, optimize caches present, robots production, sitemap reachable.
- `docs/GO-LIVE.md` (Persian): DNS/SSL, Google Search Console + Bing Webmaster (sitemap submit, URL inspection),
  301 map verification from the old site (import CSV in L7-03), final `seo:audit` + Lighthouse run, monitoring.
- **Outward actions (uploading to the host, DNS) only with the user's explicit OK.**

## Acceptance
- `app:doctor` green on the local production-like run; checklist complete.

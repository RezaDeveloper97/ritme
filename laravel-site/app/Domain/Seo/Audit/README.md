# Seo / Audit (L1-03, engine L7-05)

`SeoAuditEngine` crawls the site in-process (`Contracts/PageFetcher` → `Crawl/KernelPageFetcher`: HTTP kernel,
`Cache-Control: no-cache`, bot UA, fresh controller per request — no network): every parameterless GET route + every
sitemap URL + internal links found on the way, bounded by `AuditOptions` (max pages, time limit, fetch budget).
Rules: `SeoAuditor` (v1 single-page tags), `PageInspector` (canonical host/self, JSON-LD, LCP/lazy images, thin
content, weight, requests, third-party requests, og:image file, slow render), and cross-page checks in the engine
(links, fragments, redirects, sitemap vs robots/canonical, orphans, duplicates, analyser score). Production robots are
simulated by default (`app.env` is restored after the run). `Actions/RunSeoAudit` stores runs (`seo_audit_runs` /
`_pages` / `_issues`, last 30 kept, previous failures crawled first); `Actions/QueueSeoAudit` + `Jobs/RunSeoAuditJob`
for "run now" and the weekly schedule (SeoServiceProvider). Read side: `Queries/` (report + dashboard widgets).
CLI: `php artisan seo:audit [--path=*] [--strict] [--max-pages=] [--time-limit=] [--no-store] [--as-is] [--queue]`.

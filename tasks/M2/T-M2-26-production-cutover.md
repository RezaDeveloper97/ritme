---
id: T-M2-26
title: Production cutover to Go (per route group, with rollback)
milestone: M2
type: release
status: todo
depends_on: [T-M2-25]
parallel_group: M2-K
touches: [deploy/go-routes.inc, deploy/proxy-ssl.conf, deploy/vhost-api.inc, deploy/vhost-admin.inc, docs/go-migration/prod-cutover-log.md]
skills: [deploy, verify-all]
verify: test -s docs/go-migration/prod-cutover-log.md
---

# T-M2-26 — Production cutover

## Why
The actual switch of real users. **Outward-facing and hard to reverse → ask the user before every production step**
(deploy, each group switch, admin host switch).

## Scope
1. Preconditions: T-M2-25 soak green; `verify-all` green; DB backup taken and verified (restore test on a scratch DB);
   announce a low-traffic window with the user.
2. Deploy with backend-go started but every group on Laravel; assert both healthy.
3. Switch groups in the same order as staging, pausing between groups to watch nginx logs (5xx, latency, 401
   `error_code` mix vs. the previous 7-day baseline). Any anomaly → switch that group back immediately.
4. Auth group last, after draining the Laravel queue container.
5. Admin host: admin-web + Go admin API; admins log in again (D-06) — tell the user in advance.
6. Keep Laravel running (no traffic) for 7 days as the rollback path; record everything in
   `docs/go-migration/prod-cutover-log.md`.

## Out of scope
Removing Laravel (T-M2-27).

## Acceptance
- All groups and the admin host on Go in production, with the log showing metrics before/after each switch.
- No increase in 401 `token_*` errors or forced logouts (compare with the baseline); no data-integrity issue.
- User confirms the 7-day observation period ended OK before T-M2-27 starts.

## Note from T-M2-25 local smoke (2026-09-28)
`deploy/vhost-admin.inc` is still the Blade config (`/api/` 404, `/` → Laravel). For the prod admin host: `/` →
admin-web, `/api/admin/` and `GET /api/v1/languages` → backend-go, and `/storage/` must reach backend-go or uploaded
image previews 404 in admin-web.

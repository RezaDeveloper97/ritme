---
id: T-M2-25
title: Staging rollout — move every route group and the admin host to Go
milestone: M2
type: release
status: todo
depends_on: [T-M2-09, T-M2-23, T-M2-24]
parallel_group: M2-J
touches: [deploy/go-routes.inc, deploy/vhost-stage.inc, deploy/vhost-admin.inc, docs/go-migration/cutover.md, docs/go-migration/stage-rollout-log.md]
skills: [deploy-stage]
verify: test -s docs/go-migration/stage-rollout-log.md
---

# T-M2-25 — Staging rollout

## Why
Staging (stage.ritmeapp.ir, behind Basic auth) is where the strangler switch is exercised for real with real clients
before production. Follow `docs/go-migration/cutover.md` (T-M2-09).

## Scope
1. Deploy the `stage` branch with backend-go (deploy-stage skill).
2. Move groups to Go one at a time in the README order (content → reminders+healthlog → profile → pregnancy → cycle →
   messages → home → auth), each with: switch on, smoke via web + Android shell against staging, check nginx logs
   for 4xx/5xx and 401 `error_code` distribution for 30+ minutes, log the result in
   `docs/go-migration/stage-rollout-log.md`. Roll back a group on any regression and open a follow-up.
3. Auth last: confirm the Laravel queue is empty first; verify tokens issued by Laravel before the switch still work
   after it (stay signed in on a device across the switch).
4. Admin host on staging: route `/admin` UI to admin-web and `/api/admin/` to backend-go; editors (or the user)
   smoke every screen; Blade stays reachable on an internal path for comparison until prod cutover.
5. 48h soak with all groups on Go; Laravel still running (instant rollback).

## Out of scope
Production (T-M2-26).

## Acceptance
- Rollout log shows every group on Go with evidence, 48h soak without Go-caused regressions.
- Rollback tested at least once per environment (flip back and forth for one group).

## Note from T-M2-20
- nginx: `vhost-admin.inc` must proxy `/api/admin/` to Go (it 404s all `/api/` today); `vhost-api.inc` must 404
  `/api/admin/`; deploy scripts' host checks need the same update.
- Compose (stage + prod) must set `ADMIN_HOSTS`, `ADMIN_WEB_ORIGINS`, `ADMIN_COOKIE_SECURE` for backend-go — with
  `ADMIN_HOSTS` empty the admin API is disabled outside local/testing.

## Note from T-M2-22
Admin host nginx: `/` → admin-web (`stage-admin-web` / `ritme-admin-web-1`), `/api/admin/` → backend-go, and also
`GET /api/v1/languages` → backend-go (the translatable field needs it). Build the admin-web image once to prove the
Dockerfile (it was never built — Docker Desktop failed during T-M2-22).

## Note from T-M2-21
backend-go now writes to the `backend-storage` volume (uploads, `app/translations`, `app/lang`) → mount it
read-write for backend-go in stage and prod compose. Optional env: `LARAVEL_CACHE_KEY_PREFIX`,
`LARAVEL_CACHE_REDIS_DB`, `LARAVEL_CACHE_FLUSH` (set `false` after cutover).

## Note from T-M2-19
Route `/docs` (Swagger UI) to Go when the API host moves; set `SWAGGER_USER`/`SWAGGER_PASSWORD` for backend-go
(without a password `/docs` is 401 outside local/testing).

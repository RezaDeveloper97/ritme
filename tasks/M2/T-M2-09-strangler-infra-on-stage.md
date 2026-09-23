---
id: T-M2-09
title: Strangler infrastructure on staging (compose service, nginx route groups, rollback switch)
milestone: M2
type: release
status: todo
depends_on: [T-M2-02, T-M2-08]
parallel_group: M2-D
touches: [docker-compose.yml, docker-compose.stage.yml, docker-compose.prod.yml, deploy/go-routes.inc, deploy/vhost-stage.inc, deploy/vhost-api.inc, deploy/proxy-ssl.conf, deploy/stage-ssl.conf, deploy/switch-go-route.sh, .claude/skills/deploy-stage/SKILL.md, .claude/skills/deploy/SKILL.md, docs/go-migration/cutover.md]
skills: [deploy-stage]
verify: bash deploy/switch-go-route.sh --dry-run stage content on && docker compose -f docker-compose.yml -f docker-compose.stage.yml config -q
---

# T-M2-09 — Strangler infrastructure on staging

## Why
The strangler cutover needs Go running next to Laravel on the same MariaDB/Redis/storage volume, and an nginx switch
per route group that can be flipped back in seconds. Read infra inventory §6 (pinned container names, staging path
routing, deploy assertions) and README → Strangler route groups.

## Scope
1. Compose: new service `backend-go` (build `backend-go/`), same env as `backend` + Go vars, `backend-storage` mounted
   (read-only except `app/public` and `app/translations` once admin moves), uid 33, healthcheck `/up`, no published
   port on prod except `127.0.0.1:8081`; staging gets alias `stage-backend-go` on `ritme-edge`. Pinned container
   names: prod `ritme-backend-go-1`, stage `ritme-stage-backend-go-1`.
2. `deploy/go-routes.inc`: one nginx `location` block per route group (content, reminders, healthlog, profile,
   pregnancy, cycle, messages, home, auth), each generated as either Laravel upstream or Go upstream. Groups start on
   **Laravel**.
3. `deploy/switch-go-route.sh <stage|prod> <group> <on|off> [--dry-run]`: rewrites the group's upstream, runs
   `nginx -t`, reloads; prints the current map with `--status`. Never touches the other environment.
4. `deploy-stage` and `deploy` skills + scripts: build and start `backend-go`, add assertions (`/up` on Go 200 via
   internal curl), keep the existing Laravel assertions; `/storage` stays on Laravel until the content group moves.
5. `docs/go-migration/cutover.md`: order of groups, per-group checklist (contract green, smoke, 24h soak on stage,
   watch 401/5xx rates in nginx logs), rollback command, queue drain for auth (Laravel `queue` must be empty before
   the auth group moves).
6. Deploy to **staging only** (behind Basic auth) with every group still on Laravel; prove the switch with the
   `content` group on/off once.

## Out of scope
Moving groups for real (T-M2-25). Any prod deploy (T-M2-26).

## Acceptance
- Staging runs both backends; `switch-go-route.sh stage content on` → `/api/v1/banners` served by Go (response header
  `X-Backend: go` added by Go in non-prod), `off` → Laravel again; each flip under 5 s with zero failed requests.
- `docker compose … config -q` passes for base, stage and prod overlays; prod compose changes are inert until T-M2-26.
- The pinned-upstream rule (no Docker-DNS aliases shared between prod and stage) is preserved.

## Note from T-M2-08
Rate limiting keys on the leftmost `X-Forwarded-For` value, which the client controls (Laravel has the same
weakness). When writing the Go upstream locations, have nginx overwrite the header
(`proxy_set_header X-Forwarded-For $remote_addr;` at the edge) so the per-IP OTP limits can't be bypassed.

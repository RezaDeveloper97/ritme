---
name: deploy
description: Deploy Ritme to production (root@89.251.8.115). Rsyncs the working tree to /opt/ritme and builds images ON the server (it has full internet), then restarts the stack and verifies.
---

# Deploy Ritme to production

Server: `root@89.251.8.115` (Ubuntu 24.04, 2 vCPU / 4 GB RAM + 4 GB swap, 24 GB disk, Docker + compose v5, key auth with `~/.ssh/id_ed25519`).

Unlike the old host (62.60.198.240), **this server has full international connectivity**, so images are built there. No local cross-compilation, no `docker save` streaming. Everything is handled by `./deploy.sh` at the repo root.

## Pre-deploy checklist
1. Run `/verify-all` (or at minimum backend `php artisan test` + frontend `npm run typecheck`). A broken Next build fails the deploy after several minutes.
2. Nothing needs to be committed or pushed — `deploy.sh` rsyncs the **working tree**. What you have locally is what ships.
3. Check `/opt/ritme/.env` if you're changing build-time config: `NEXT_PUBLIC_API_BASE_URL` is baked into the browser bundle from there.

## Deploy
```bash
./deploy.sh
```
Steps: rsync source → `docker compose -f docker-compose.yml -f docker-compose.prod.yml build` → `up -d` → prune dangling images → verify. The verify block **asserts exact status codes and exits 1 on mismatch** — notably 401 on a protected route, because a PHP fatal in `bootstrap/app.php` still answers 200 with an error page, which once hid a completely dead API behind a green-looking deploy. Switches:
- `SERVICES="frontend" ./deploy.sh` — only rebuild/restart some services
- `NO_BUILD=1 ./deploy.sh` — ship files + restart, skip builds
- `SKIP_SYNC=1 ./deploy.sh` — rebuild from what's already on the server

## Layout on the server
`/opt/ritme` holds the mirrored source. Server-only paths, excluded from rsync so they survive every deploy:
- `.env` — prod secrets (APP_KEY, DB passwords, SMS keys, admin seed)
- `ssl/` — copied TLS cert + key (nginx container can't follow letsencrypt symlinks)
- `certbot-www/` — ACME webroot

## Hostnames — three vhosts, one stack

| Host | Serves | nginx include |
| ---- | ------ | ------------- |
| `https://api.ritme.app` | Laravel API (`/api/v1`, `/oauth`, `/storage`, `/up`). `/admin` returns 404 here. | `deploy/vhost-api.inc` |
| `https://adpanell.ritme.app` | Blade admin panel. `/` → `/admin`; `/api/` returns 404. | `deploy/vhost-admin.inc` |
| `https://web.ritme.app` | Next.js frontend. Calls `api.ritme.app` directly — nothing proxies to the backend here. | `deploy/vhost-web.inc` |

The `proxy` service (nginx:alpine, in `docker-compose.prod.yml`) owns :80/:443. mysql/backend/frontend ports are bound to 127.0.0.1 only, and a `default_server` returns 444 so the bare IP and unknown hostnames serve nothing. The HTTP/HTTPS variants (`deploy/proxy.conf` / `proxy-ssl.conf`) share the same `vhost-*.inc` fragments so they can't drift.

**Because web and api are separate origins, every browser API call is cross-origin.** The allow-list lives in `backend/config/cors.php` (override at runtime with `CORS_ALLOWED_ORIGINS`, comma-separated). A new frontend origin must be added there or the browser blocks it. `bootstrap/app.php` also configures `trustProxies` — without it Laravel can't see that TLS terminated at the proxy and the admin panel emits `http://` assets on an https page.

## Go backend (strangler, milestone M2) — inert until T-M2-26

`docker-compose.prod.yml` defines `backend-go` (container `ritme-backend-go-1`, `127.0.0.1:8081`) under the compose
**profile `go`**, so today's `./deploy.sh` neither builds nor starts it. T-M2-26 turns it on with
`COMPOSE_PROFILES=go` in `/opt/ritme/.env` (compose reads it from there; nothing in `deploy.sh` changes).

The proxy already routes `api.ritme.app` through `deploy/go-routes.inc`: one regex location per route group, each
proxying to `upstream prod_route_<group>` in `deploy/proxy-ssl.conf` — **all pointing at `ritme-backend-1`**
(Laravel), so behaviour is unchanged, including `X-Forwarded-For` (prod sits behind a CDN; see cutover.md →
Prod prerequisites before changing it). Moving a group is `deploy/switch-go-route.sh prod <group> on|off`
(nginx -t guarded, auto-rollback), only after it soaked on stage — `docs/go-migration/cutover.md`.

- `go-routes.inc` is a **new bind mount** on the proxy: the first deploy that ships it recreates the proxy
  (`up -d` does that on its own), a one-to-two-second blip. It is required: `vhost-api.inc` includes it.
- `deploy/proxy.conf` (the HTTP-only pre-certificate config) does **not** define the `prod_route_*` upstreams yet;
  falling back to it would 502 every route-group path. Prod runs `proxy-ssl.conf`.
- Once `backend-go` runs on prod, add to the verify block: `exec -T backend-go /app/api healthcheck` (exit 0) and
  `exec -T proxy wget -qSO- http://ritme-backend-go-1/up` (200), keeping every existing Laravel assertion.
- Pinned names only: the upstreams use `ritme-backend-1` / `ritme-backend-go-1`, never `backend` / `backend-go`,
  which the stage project also registers on `ritme-edge`.

### Build order once Go is on (T-M2-33) — inert today

The frontend's `/[locale]/*` pages are prerendered and fetch the UI messages from the live API at build time, so
building the frontend while the old API serves bakes stale copy (stage bug B1). `deploy.sh` asks the server
`compose config --services`; **only if `backend-go` is listed** (i.e. `COMPOSE_PROFILES=go`, T-M2-26) it builds
backend-go, `up -d --wait`s it, reloads the proxy, and then builds with `--build-arg BUILD_REV=<sha>-<timestamp>`
(consumed right before `npm run build` in `frontend/Dockerfile`, so that layer always re-runs). While prod is
Laravel the list has no `backend-go`, so the build is the old `build ${SERVICES}` — no build arg, same cache, same
order. (Adding the `ARG` line changed the Dockerfile, so the first prod frontend build after it re-runs `npm run
build` once; the resulting image is equivalent.)

## HTTPS
One SAN certificate covers all three hostnames (issued 2026-08-16, expires 2026-11-14):
```bash
ssh root@89.251.8.115 'cd /opt/ritme && ./deploy/enable-https.sh'
```
Idempotent — it issues/renews the cert, copies it to `ssl/`, sets `PROXY_CONF=./deploy/proxy-ssl.conf` in `.env`, and installs the certbot deploy hook that re-copies and reloads nginx on renewal.

Note: `certbot renew --dry-run` may fail because the ACME *staging* host doesn't resolve from Iranian resolvers; real renewals still work.

## Post-deploy
1. Migrations run automatically via the backend entrypoint (`RUN_MIGRATIONS=true`). Seeders do not — run them manually and idempotently:
   `ssh root@89.251.8.115 'cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T backend php artisan db:seed --class=...'`
2. Verify: `curl https://api.ritme.app/up`, `curl -H 'Accept: application/json' https://api.ritme.app/api/v1/banners` (401 = auth stack alive), `https://adpanell.ritme.app/admin/login`, `https://web.ritme.app/`. `deploy.sh` runs exactly these at the end.
3. Logs: `ssh root@89.251.8.115 'cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --tail=100 backend frontend proxy'`

## Notes
- Never run destructive artisan commands (`migrate:fresh`, `db:wipe`) on the server.
- Each rebuild orphans the previous image; `deploy.sh` prunes automatically — the old host died of ENOSPC without this.

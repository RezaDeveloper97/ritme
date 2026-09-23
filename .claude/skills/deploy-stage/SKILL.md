---
name: deploy-stage
description: Deploy Ritme staging to https://stage.ritmeapp.ir from the `stage` branch. Runs as a second compose project on the production server, behind HTTP Basic auth.
---

# Deploy Ritme staging

`https://stage.ritmeapp.ir` — a second, fully isolated stack on the **same** server as
production (`root@89.251.8.115`), gated by HTTP Basic auth.

```bash
./deploy-stage.sh
```

## What makes it separate from prod

| | production | staging |
| --- | --- | --- |
| source on server | `/opt/ritme` | `/opt/ritme-stage` |
| compose project | `ritme` | `ritme-stage` |
| overlay | `docker-compose.prod.yml` | `docker-compose.stage.yml` |
| volumes | `ritme_mysql-data`, … | `ritme-stage_mysql-data`, … |
| ships | your **working tree** | tip of the **`stage` branch** (`git archive`) |
| published ports | 80/443 (proxy), rest loopback | **none** |

Different databases, different Redis, different secrets (`/opt/ritme-stage/.env`,
server-only — model it on `.env.stage.example`). The two stacks share exactly two
things: the Docker daemon and production's nginx.

## One hostname, path routing

Only `stage.ritmeapp.ir` has a DNS record here — there is no `api.stage` or
`admin.stage`. So staging puts everything on one origin (`deploy/vhost-stage.inc`):

- `/api`, `/oauth`, `/storage`, `/docs` → stage backend
- `/admin` → stage Blade panel (kept for comparison until the prod admin cutover)
- `/panel` → stage **admin-web** (new Next.js admin, built with basePath `/panel`), `/api/admin/` → stage
  backend-go (admin API; `ADMIN_HOSTS=stage.ritmeapp.ir`). Both are resolved at request time (Docker DNS), so a
  missing admin-web/Go container 502s these paths instead of failing `nginx -t` for the shared proxy.
- everything else → stage Next.js frontend
- **except the strangler route groups** (`deploy/go-routes.inc`, included by `vhost-stage.inc`): each group
  (`content`, `reminders`, `healthlog`, `profile`, `pregnancy`, `cycle`, `messages`, `home`, `auth`) goes to Laravel
  *or* Go per its `upstream stage_route_<group>` line in `deploy/stage-ssl.conf`. All start on Laravel. See
  "Go backend (strangler)" below.

Same-origin is a real benefit: **staging needs no entry in `backend/config/cors.php`**,
because the browser never makes a cross-origin call. `NEXT_PUBLIC_API_BASE_URL` is
`https://stage.ritmeapp.ir/api/v1`, baked into the bundle at build time from the
staging `.env`.

## The password, and why the gate is a cookie

The whole vhost is private. Credentials live in `/opt/ritme/stage.htpasswd`
(server-only, excluded from both rsyncs) and a copy of what was provisioned is in
`/root/ritme-stage-credentials.txt`.

**Basic auth alone does not work for this app, and never did.** A request carries
exactly one `Authorization` header, and the app spends it on `Bearer <token>`;
Chrome will not overwrite that with the cached Basic credentials. So every
`/api/*` XHR reached nginx un-credentialed and got a 401 — which the frontend's
interceptor read as "session expired", wiping the token and bouncing to
`/signup`. Onboarding could never save a profile. Requests the browser sends with
`credentials: omit` (the web manifest) 401'd for a second reason and re-opened the
password dialog on every single page.

The gate is therefore a **cookie**, which rides alongside the bearer token:

1. A plain page load has no `Authorization` header of its own, so Basic auth still
   works there — nginx asks for the password once.
2. Any authenticated 2xx/3xx response hands out `ritme_stage` (never a 401, or the
   challenge would mint the credential that opens the gate).
3. `$stage_auth_realm` evaluates to `off` while that cookie is present, so
   XHRs, service-worker fetches and RSC prefetches all pass.

The maps live in `/opt/ritme/stage-gate.conf`, generated once by
`deploy/stage-gate.sh` (called from both deploy scripts) and mounted into the proxy
as `conf.d/aa-stage-gate.conf`. It holds the token, so it is server-only like the
htpasswd. **Delete the file and redeploy to rotate it** — everyone is then asked for
the password again.

The same map turns the gate off for container-to-container traffic (`10/8`,
`172.16/12`, `127/8`): the Next.js server calls `/api/v1/languages` and the message
endpoints through this proxy during SSR and can offer neither password nor cookie —
without that, staging silently ran on bundled locales.

Exempt from the gate on purpose: `/up` (uptime probes, deploy verification),
`/manifest.webmanifest` (`credentials: omit`), and `/sw.js`, `/version.json`,
`/offline.html`, `/favicon.ico`, `/icons/` — public static assets whose gating breaks
service-worker registration. Each exempt location sets its own `add_header`, which
cancels the inherited `Set-Cookie`, so a public file cannot mint a session.

Change the password:
```bash
ssh root@89.251.8.115 "printf 'ritme:%s\n' \"\$(openssl passwd -apr1 'NEWPASS')\" > /opt/ritme/stage.htpasswd \
  && cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T proxy nginx -s reload"
```
(`openssl passwd -apr1` rather than `htpasswd` so nothing extra has to be installed.)

## How the proxy serves two environments

Production's `proxy` container is the only thing on :80/:443. It gained:

- an extra bind-mount, `${STAGE_CONF:-./deploy/stage-off.conf}` → `conf.d/zz-stage.conf`.
  Same switch-by-env pattern as `PROXY_CONF`. Default is an **empty** file, so a stack
  with no staging environment is completely unaffected. `deploy-stage.sh` sets it to
  `stage-http.conf`; `deploy/enable-https-stage.sh` flips it to `stage-ssl.conf`.
  The `zz-` prefix keeps it loading after `default.conf`, which owns the
  `default_server` blocks.
- membership in the external `ritme-edge` bridge network. Staging's backend/frontend
  join it under the aliases `stage-backend` / `stage-frontend`, which is how nginx
  resolves them across compose projects. Both deploy scripts create the network if
  it's missing.

`STAGE_CONF` changes a **mount source**, so the proxy must be *recreated*, not
reloaded — `up -d proxy`, which both scripts do.

**Gotcha:** `deploy-stage.sh` only rsyncs `/opt/ritme-stage`. If you change
`docker-compose.prod.yml` or anything in `deploy/`, production's copy is stale until
you also run `NO_BUILD=1 ./deploy.sh` (or rsync those paths by hand).

## Go backend (strangler, milestone M2)

Staging runs **both** APIs: `backend` (Laravel) and `backend-go` (Go, `backend-go/`), on the same staging MariaDB,
Redis and `backend-storage` volume (Go mounts it read-only, runs as uid 33 to read the Passport keys). The plain
`build` / `up -d` of `deploy-stage.sh` builds and starts it like any other service; `SERVICES="backend-go"` ships
only Go. It is container `ritme-stage-backend-go-1`, reachable by the proxy only as the stage-only alias
**`stage-backend-go`** on `ritme-edge` — never as the bare `backend-go`, which Docker also registers there (the
2026-08-31 incident; prod's own Go container is addressed by its pinned name `ritme-backend-go-1`).

Flip a route group (details, order, per-group checklist, rollback: `docs/go-migration/cutover.md`):
```bash
deploy/switch-go-route.sh --status stage          # live map
deploy/switch-go-route.sh stage content on         # -> Go   (nginx -t guarded, auto-rollback, graceful reload)
deploy/switch-go-route.sh stage content off        # -> Laravel (the rollback)
```
It edits only `stage_route_*` lines of the live `STAGE_CONF` file; the prod upstreams are never touched. A live
flip is overwritten by the next rsync of `/opt/ritme` — make it permanent with `--local` + commit.

Go on stage answers with `X-Backend: go` (added outside production), so which stack served a request is visible:
```bash
curl -s -o /dev/null -D - -u "$STAGE_BASIC_AUTH" -H "Accept: application/json" https://stage.ritmeapp.ir/api/v1/banners | grep -i x-backend
```

Verify Go (internal — it has no public path of its own; `/up` on the vhost stays Laravel):
```bash
ssh root@89.251.8.115 'cd /opt/ritme-stage && docker compose -p ritme-stage -f docker-compose.yml -f docker-compose.stage.yml exec -T backend-go /app/api healthcheck && echo GO-UP-OK'
ssh root@89.251.8.115 'cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T proxy wget -qSO- http://stage-backend-go/up 2>&1 | head -1'   # HTTP/1.1 200 OK — proves the proxy can reach it
```

**Shipping the nginx side.** `go-routes.inc`, `vhost-stage.inc` and `stage-ssl.conf` are read by *production's*
proxy from `/opt/ritme/deploy/`, which `deploy-stage.sh` does not sync (see the gotcha above), and
`go-routes.inc` is a **new bind mount** in `docker-compose.prod.yml`, so the proxy must be *recreated* once
(`up -d proxy`), not just reloaded. Until that has happened the staging vhost keeps its old (all-Laravel)
routing. Recreating the proxy blips production for a second or two — schedule it with a normal prod deploy.

`deploy/stage-http.conf` (the pre-certificate variant) does not define the `stage_route_*` upstreams yet, so
route-group paths would 502 on it; staging runs `stage-ssl.conf`.

After a redeploy recreates `backend-go` (or `backend`), nginx may still hold the old container IP (upstream
hostnames resolve at load): `… exec -T proxy sh -c 'nginx -t && nginx -s reload'`.

## HTTPS

Its own certificate — `stage.ritmeapp.ir` is on a different registrable domain than the
production hostnames, and it is written to `ssl/stage-*.pem` so a botched staging cert
can never take production's TLS down. Issued 2026-08-30, expires 2026-11-28.

```bash
ssh root@89.251.8.115 'cd /opt/ritme && ./deploy/enable-https-stage.sh'
```
Idempotent; installs its own certbot deploy hook (`ritme-stage-proxy.sh`).

Note `deploy/acme.inc` now carries `auth_basic off` — without it the staging vhost's
password would make the ACME challenge unfetchable, and an unfetchable challenge means
no certificate. It's a no-op for the production vhosts.

## Switches

```bash
BRANCH=my-feature ./deploy-stage.sh   # ship a different branch
FROM_WORKTREE=1   ./deploy-stage.sh   # ship the working tree instead of a branch
SERVICES="frontend" ./deploy-stage.sh # rebuild/restart only some services
NO_BUILD=1  ./deploy-stage.sh         # ship files + restart, skip builds
SKIP_SYNC=1 ./deploy-stage.sh         # rebuild from what's already on the server
STAGE_BASIC_AUTH='user:pass' ./deploy-stage.sh   # also verify BEHIND the gate
```

## Verification

The script asserts exact status codes and exits 1 on mismatch, same discipline as
`deploy.sh`. The important one is `401` on `/` with no credentials — a `200` there
means staging is world-readable.

## Resources

4 GB RAM / 2 vCPU / 24 GB disk now runs two full stacks. Keep an eye on it:
```bash
ssh root@89.251.8.115 'free -m; df -h /; docker system df'
```
Both deploy scripts `docker image prune -f`; if the disk gets tight,
`docker builder prune -af` reclaims the most (it freed ~9.8 GB when staging was set up).
Seeders don't run automatically — run them manually against the `ritme-stage` project.

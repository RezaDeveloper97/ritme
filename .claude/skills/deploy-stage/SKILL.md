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
- `/admin` → stage Blade panel
- everything else → stage Next.js frontend

Same-origin is a real benefit: **staging needs no entry in `backend/config/cors.php`**,
because the browser never makes a cross-origin call. `NEXT_PUBLIC_API_BASE_URL` is
`https://stage.ritmeapp.ir/api/v1`, baked into the bundle at build time from the
staging `.env`.

## The password

The whole vhost is behind HTTP Basic auth. Credentials live in
`/opt/ritme/stage.htpasswd` (server-only, excluded from both rsyncs) and a copy of
what was provisioned is in `/root/ritme-stage-credentials.txt`.

Because the API is on the same origin as the page, the browser replays the cached
credentials on every `fetch`/XHR automatically — no client-side change is needed.
`/up` is deliberately exempt so uptime probes and the deploy verification keep working.

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

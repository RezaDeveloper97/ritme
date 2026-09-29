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

**Staging runs Go only (T-M2-28, 2026-09-23)** — no Laravel container serves anything:

- `/up`, `/api/…` (every route group + the catch-all), `/storage`, `/docs` → stage **backend-go**
- `/api/admin/` → backend-go (admin API; `ADMIN_HOSTS=stage.ritmeapp.ir`)
- `/panel` → stage **admin-web** (Next.js admin, built with basePath `/panel`)
- `/admin`, `/admin/*` → **301 `/panel/`** (Blade panel retired on stage); `/oauth/*` → **404**
- `/api/session/flag` and everything else → stage Next.js frontend

The backend-go / admin-web targets are resolved at **request time** (variable `proxy_pass` + Docker DNS), so a
missing stage container 502s its paths instead of failing `nginx -t` for the shared production proxy. The strangler
groups still pass through `deploy/go-routes.inc` and the `upstream stage_route_<group>` lines in
`deploy/stage-ssl.conf` — all on Go.

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
- membership in the external `ritme-edge` bridge network. Staging's backend-go,
  frontend and admin-web join it under the aliases `stage-backend-go` (+ `stage-backend`),
  `stage-frontend` and `stage-admin-web`, which is how nginx resolves them across
  compose projects. Both deploy scripts create the network if
  it's missing.

`STAGE_CONF` changes a **mount source**, so the proxy must be *recreated*, not
reloaded — `up -d proxy`, which both scripts do.

**Gotcha:** `deploy-stage.sh` only rsyncs `/opt/ritme-stage`. If you change
`docker-compose.prod.yml` or anything in `deploy/`, production's copy is stale until
you also run `NO_BUILD=1 ./deploy.sh` (or rsync those paths by hand).

## Go backend — the only stage backend (T-M2-28)

Staging's API is `backend-go` (container `ritme-stage-backend-go-1`), on the staging MariaDB, Redis and
`backend-storage` volume (uid 33, reads the Passport keys). Laravel's `backend` and `queue` are behind the compose
profile **`laravel`**: a plain `build` / `up -d` skips them, and `deploy-stage.sh` removes the old
`ritme-stage-backend-1` / `ritme-stage-queue-1` containers if they are still around. Production is unchanged
(Laravel until T-M2-26/27).

- **Aliases on `ritme-edge`**: `stage-backend-go` (what the vhost and the route-group upstreams use) and also
  `stage-backend` — `stage-ssl.conf` still declares `upstream stage_backend_upstream { server stage-backend:80; }`,
  resolved at nginx load, so that name must keep resolving or the *shared prod proxy* fails `nginx -t`. Never the
  bare `backend-go` (Docker registers that too — the 2026-08-31 incident; prod's Go is `ritme-backend-go-1`).
- **Schema**: goose owns it on stage. `RUN_MIGRATIONS=true` (stage overlay only) makes backend-go run
  `db.MigrateOnStart` before serving: stamps a Laravel-built DB (first deploy), builds an empty one, or applies
  pending migrations; it logs one `"msg":"migrations"` line and exits on failure. Details:
  `docs/go-migration/migrations.md`. Schema changes still need a goose **and** a Laravel migration until T-M2-27.
- **Fresh stage volumes**: Go creates no Passport keys, no personal-access client, no seed rows. Bootstrap once
  with Laravel before backend-go first starts — `docs/go-migration/cutover.md` → "Stage runs Go only".
- **Rollback**: `switch-go-route.sh stage <group> off` now also lands on Go (the `stage-backend` alias). A stage
  problem is rolled back by redeploying an earlier `stage` commit (`BRANCH=<sha-or-branch>`).

`SERVICES="backend-go" ./deploy-stage.sh` ships only Go. Go answers with `X-Backend: go` (outside production):
```bash
curl -s -o /dev/null -D - https://stage.ritmeapp.ir/up | grep -i x-backend
curl -s -o /dev/null -D - -u "$STAGE_BASIC_AUTH" -H "Accept: application/json" https://stage.ritmeapp.ir/api/v1/languages | grep -i x-backend
ssh root@89.251.8.115 'docker logs ritme-stage-backend-go-1 2>&1 | grep "\"msg\":\"migrations\"" | tail -1'
```

Side-by-side Laravel (debugging only, not proxied):
`docker compose -p ritme-stage -f docker-compose.yml -f docker-compose.stage.yml --profile laravel up -d backend`
— its entrypoint runs Laravel's migrations on the goose-owned schema; remove it again with `docker rm -f
ritme-stage-backend-1`.

**Shipping the nginx side.** `vhost-stage.inc`, `go-routes.inc` and `stage-ssl.conf` are read by *production's*
proxy from `/opt/ritme/deploy/`, which `deploy-stage.sh` does not sync (see the gotcha above). They are
**single-file bind mounts**: a plain `rsync` writes a new file and renames it over the old one, and the running
container keeps seeing the *old* inode. Either recreate the proxy (`up -d proxy`, a 1–2 s prod blip — do it with a
normal prod deploy) or update the file **in place** and reload:
```bash
rsync -a --inplace deploy/vhost-stage.inc root@89.251.8.115:/opt/ritme/deploy/vhost-stage.inc
ssh root@89.251.8.115 'cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T proxy sh -c "nginx -t && nginx -s reload"'
```
(`nginx -t` failing → fix before anything else; the old config keeps serving until a successful reload.)

`deploy/stage-http.conf` (the pre-certificate variant) does not define the `stage_route_*` upstreams, so
route-group paths would 502 on it; staging runs `stage-ssl.conf`.

After a redeploy recreates `backend-go`, nginx may still hold the old container IP for the upstream{} names
(request-time `$stage_backend_go_target` paths re-resolve by themselves); `deploy-stage.sh` reloads the proxy
gracefully at the end of step 4.

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

## Build order: backend-go before the frontend (T-M2-33)

`/[locale]/*` pages are **prerendered** at `npm run build`, and prerendering fetches the UI messages from the
**live** stage API (`frontend/src/shared/i18n/messages.ts`; the remote bundle wins over the bundled JSON). So the
HTML bakes whatever backend-go serves *while the frontend image builds*. Bug B1 (2026-09-28): the script used to
build everything while the OLD backend-go was still serving and only then `up -d`, so changed copy stayed stale —
and a plain redeploy didn't help because the build layer was cached (same source).

`deploy-stage.sh` step 3 therefore:
1. `build backend-go` → `up -d --wait backend-go` (healthy ⇒ goose ran; a failure aborts the deploy) → logs the
   `"msg":"migrations"` line → graceful proxy reload (the SSG fetch goes through the proxy, whose route-group
   `upstream{}`s resolve at load).
2. `build --build-arg BUILD_REV=<sha>-<utc-timestamp> ${SERVICES}`. `frontend/Dockerfile` declares `ARG BUILD_REV`
   right before `RUN npm run build`, so exactly that layer re-runs every deploy (`npm ci` stays cached — no
   blanket `--no-cache`). The build log prints `frontend build rev: …`.
3. `up -d ${SERVICES}` as before.

Step 1 runs only when `SERVICES` is empty or contains `backend-go`; `SERVICES="frontend"` rebuilds the frontend
against whatever backend-go is already live (still uncached). Don't reach for `build --no-cache frontend` any more.

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
means staging is world-readable. Go-only assertions (T-M2-28): `/up` 200 with
`X-Backend: go`; `/api/v1/languages` served by Go (asked from inside the proxy, which
the gate exempts); `/admin` and `/admin/users` → 301 `…/panel/` without minting the gate
cookie; `/oauth/token` → 404. With `STAGE_BASIC_AUTH`, banners and an unported `/api`
path must also carry `X-Backend: go`.

## Resources

4 GB RAM / 2 vCPU / 24 GB disk now runs two full stacks. Keep an eye on it:
```bash
ssh root@89.251.8.115 'free -m; df -h /; docker system df'
```
Both deploy scripts `docker image prune -f`; if the disk gets tight,
`docker builder prune -af` reclaims the most (it freed ~9.8 GB when staging was set up).
Seeders don't run automatically — run them manually against the `ritme-stage` project.

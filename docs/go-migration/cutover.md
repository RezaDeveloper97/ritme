# Strangler cutover — moving route groups from Laravel to Go

Go (`backend-go`) runs beside Laravel (`backend`) on the same MariaDB, Redis and `backend-storage` volume. nginx
decides per **route group** which one answers. Moving a group, and moving it back, is one command.

## Moving parts

| Piece | What it does |
|---|---|
| `docker-compose.yml` → `backend-go` | Go API, uid 33, storage volume **read-only**, healthcheck `/app/api healthcheck` (`/up`). Local: `:8081`. |
| `docker-compose.stage.yml` | Container `ritme-stage-backend-go-1`, no published port, alias **`stage-backend-go`** on `ritme-edge`. |
| `docker-compose.prod.yml` | Container `ritme-backend-go-1`, `127.0.0.1:8081`, **profile `go`**: not built or started until T-M2-26 sets `COMPOSE_PROFILES=go` in `/opt/ritme/.env`. |
| `deploy/go-routes.inc` | One regex `location` per group, shared by `vhost-api.inc` (prod) and `vhost-stage.inc` (stage). Each proxies to the upstream `${ritme_env}_route_<group>`. |
| `deploy/stage-ssl.conf` / `deploy/proxy-ssl.conf` | One `upstream stage_route_<group>` / `prod_route_<group>` line per group. **This line is the switch.** |
| `deploy/switch-go-route.sh` | Rewrites that line in the live file on the server, `nginx -t`, graceful reload; restores the backup on any failure. |

Targets (pinned; never a bare service name such as `backend-go`, which Docker also registers on `ritme-edge` for the
*stage* project — the 2026-08-31 incident):

| | Laravel | Go |
|---|---|---|
| stage | `stage-backend:80` | `stage-backend-go:80` |
| prod | `ritme-backend-1:80` | `ritme-backend-go-1:80` |

## Groups and their paths

| # | Group | Paths | Notes |
|---|---|---|---|
| 1 | `content` | `/api/v1/{languages,info,privacy,banners,articles}…`, `/api/v1/cycle/phase-content/…`, `/storage/…` | `/storage` moves with content (Go serves uploads from the volume). |
| 2 | `reminders` | `/api/v1/reminders…` | |
| 2 | `healthlog` | `/api/v1/health-logs…` | Same wave as reminders. |
| 3 | `profile` | `/api/v1/profile…`, `/api/v1/account` | |
| 4 | `pregnancy` | `/api/v1/pregnancy…` | |
| 5 | `cycle` | `/api/v1/cycle/…` except `phase-content` | |
| 6 | `messages` | `/api/v1/messages…` | |
| 7 | `home` | `/api/v1/home…` | |
| 8 | `auth` | `/api/v1/auth…` | **Last**, after token interop is proven both ways and the queue is drained (below). |
| — | stays on Laravel | `/oauth/*`, `/docs`, `/up`, `/admin`, any unclaimed `/api/*` | Admin moves separately (adpanell → `admin-web`). |

## Commands

```bash
deploy/switch-go-route.sh --status                  # live map, both environments
deploy/switch-go-route.sh stage content on          # stage: content -> Go
deploy/switch-go-route.sh stage content off         # ROLLBACK: content -> Laravel
deploy/switch-go-route.sh --dry-run stage content on  # diff against the repo copy, touches nothing
deploy/switch-go-route.sh --local stage content on    # rewrite the repo copy (make it permanent)
```

A live flip lives only in the server's copy of the file; the next deploy rsyncs the repo copy over it. Once a group
has passed its soak, make it permanent with `--local`, commit, and deploy — otherwise a routine deploy silently
moves the group back to Laravel.

`nginx -t` refuses a Go target whose container is not running ("host not found in upstream"), and the script then
restores the backup — so `on` is safe to try. It also refuses to change anything but the one upstream line.

nginx resolves upstream hostnames at load time. After **recreating** a backend container (redeploy), reload the
proxy so it picks up a changed container IP:
`ssh root@89.251.8.115 'cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T proxy sh -c "nginx -t && nginx -s reload"'`.

## Per-group checklist

Do every step on **stage** first; prod only after stage passed the soak.

1. **Contract green** — `make contract ROUTES=<group pattern>` passes against the Laravel goldens for every route of
   the group, and any difference has a `deviations.md` entry the user approved.
2. **Deployed** — the Go build with the group's routes is running (`docker compose … ps backend-go` healthy;
   `exec -T backend-go /app/api healthcheck` exits 0).
3. **Flip** — `deploy/switch-go-route.sh <env> <group> on`.
4. **Smoke** (within minutes) — on stage, a response of the group carries `X-Backend: go` (Go adds it outside
   production); run through the screens that use the group in the web app and the Android app, logged in; check a
   401 still has `error_code` (a wrong code logs users out).
5. **Soak 24 h on stage** (prod: 24 h as well, watched closely for the first hour).
6. **Watch the logs** — compare 401 and 5xx rates for the group's paths before/after the flip:
   ```bash
   ssh root@89.251.8.115 'cd /opt/ritme && docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --since 1h proxy' \
     | grep -E '"(GET|POST|PUT|DELETE) /api/v1/<prefix>' | grep -oE '" [0-9]{3} ' | sort | uniq -c
   ```
   plus `logs backend-go` for `"level":"ERROR"`. Any rise in 401s or 5xx → roll back first, investigate second.
7. **Make it permanent** — `--local`, commit, deploy.

**Rollback** at any step: `deploy/switch-go-route.sh <env> <group> off` (about a second, graceful, no dropped
requests). Both stacks share the DB, so data written by Go is read by Laravel after a rollback.

## Before the `auth` group moves — drain the queue

Laravel sends OTP SMS through its Redis queue (`SendOtpSmsJob`, worker = `queue` service); Go uses asynq under its
own `ritme-go:` prefix and never reads Laravel's jobs. The Laravel queue **must be empty before `auth` moves**:

1. Check it right before the flip — every Laravel queue list has length 0 and nothing new in `queue:failed`:
   ```bash
   C='docker compose -f docker-compose.yml -f docker-compose.stage.yml -p ritme-stage'   # prod: -f docker-compose.prod.yml
   # Laravel keeps queues:default (list) plus :delayed / :reserved (zsets) — all must be 0 (`:notify` may be ignored).
   $C exec -T redis sh -c 'for k in $(redis-cli --scan --pattern "*queues:*"); do t=$(redis-cli type "$k"); case $t in list) n=$(redis-cli llen "$k");; zset) n=$(redis-cli zcard "$k");; *) n="?";; esac; echo "$k $t $n"; done'
   $C exec -T backend php artisan queue:failed
   ```
   A backlog means SMS delivery is already delayed — fix that first, do not flip on top of it.
2. Flip `auth`. OTPs requested in the last second before the flip may still be enqueued on the Laravel side, so
   **keep the `queue` worker running** and re-run the check until it is 0 again.
3. The `queue` service is removed only in T-M2-27, never as part of a flip. Rolling `auth` back needs no drain: Go's
   asynq worker finishes its own jobs in-process.

## Prod prerequisites (T-M2-26)

- `COMPOSE_PROFILES=go` in `/opt/ritme/.env`, then deploy (builds and starts `ritme-backend-go-1`).
- **Client IP behind the CDN.** `api.ritme.app` resolves to CDN edges, so on prod `$remote_addr` is the CDN node.
  `vhost-api.inc` therefore still forwards `$proxy_add_x_forwarded_for` (the forgeable leftmost value, same as
  Laravel today). Before `auth` moves on prod, configure `set_real_ip_from <CDN ranges>` + `real_ip_header` and
  switch `$ritme_client_xff` to `$remote_addr`; otherwise either the OTP per-IP limits are bypassable (forged XFF)
  or every user shares the CDN node's limit. Stage is the edge itself and already forwards `$remote_addr` only.
- Prod deploy assertions for Go (see the deploy skill).

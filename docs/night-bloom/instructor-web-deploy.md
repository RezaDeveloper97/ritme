# instructor-web — deploy (bloom B-N8-05)

The instructor panel is its own Next.js app (`instructor-web/`) on its own host:

| Env | Host | Containers | nginx |
|---|---|---|---|
| stage | `instructor.ritmeapp.ir` (behind the stage Basic-auth/cookie gate) | `ritme-stage-instructor-web-1` (alias `stage-instructor-web`) + stage backend-go | `deploy/vhost-instructor-stage.inc`, selector `INSTRUCTOR_STAGE_CONF` → `instructor-stage-{http,ssl}.conf` |
| prod | `instructor.ritme.app` | `ritme-instructor-web-1` + `ritme-backend-go-1` (both profile `go`, inert until T-M2-26) | `deploy/vhost-instructor.inc`, selector `INSTRUCTOR_CONF` → `instructor-{http,ssl}.conf` |

On each host nginx sends `/api/` to backend-go (`/api/v1/auth/*` OTP + `/api/instructor/v1/*`) and everything else to
instructor-web, so the browser only ever talks to one origin (no CORS). `/api/admin/` answers 404 there. Upstreams are
resolved at request time, so a missing container gives that host a 502 and never breaks `nginx -t` of the shared proxy.
Both selectors default to `deploy/instructor-off.conf` (empty): **nothing changes on the server until the steps below
are done.** Each host gets its own Let's Encrypt certificate (`ssl/instructor-stage-*.pem`, `ssl/instructor-*.pem`);
prod's SAN certificate and the stage certificate are not touched.

`deploy-stage.sh` builds and starts `instructor-web` with the rest of the stage stack (it is a normal service there);
`deploy.sh` builds it only once `COMPOSE_PROFILES=go` is set (T-M2-26). Both scripts check the instructor host only
after its selector is on HTTPS.

## Steps for the operator (outward actions — need approval)

### Stage — instructor.ritmeapp.ir

1. **DNS:** at the `ritmeapp.ir` DNS provider add `A  instructor  89.251.8.115` (same target as `stage`). If the zone sits
   behind a CDN/proxy (e.g. ArvanCloud), keep this record **DNS-only** until the certificate is issued (HTTP-01 must
   reach the origin). Check: `dig +short instructor.ritmeapp.ir` → `89.251.8.115`.
2. **Ship the stack** (builds `instructor-web`): `./deploy-stage.sh` (after the change is on the `stage` branch).
3. **Turn on the :80 vhost** (serves the ACME challenge, already gated):
   ```bash
   ssh root@89.251.8.115 "cd /opt/ritme && (grep -q '^INSTRUCTOR_STAGE_CONF=' .env || echo 'INSTRUCTOR_STAGE_CONF=./deploy/instructor-stage-http.conf' >> .env) \
     && docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d proxy \
     && docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T proxy nginx -t"
   ```
   (`/opt/ritme` must contain this tree — the next `./deploy.sh` or an rsync of `deploy/` + `docker-compose.prod.yml`.)
4. **Certificate + HTTPS:** `ssh root@89.251.8.115 'cd /opt/ritme && ./deploy/enable-https-instructor.sh stage'`
5. **Verify:** `curl -sI https://instructor.ritmeapp.ir/up` → 200; `curl -sI https://instructor.ritmeapp.ir/login` → 401
   (gate); with the stage password the login page loads and `/api/instructor/v1/me` answers Go's JSON 401.
   `STAGE_BASIC_AUTH='user:pass' ./deploy-stage.sh` runs these checks automatically from now on.

### Prod — instructor.ritme.app (only after T-M2-26 puts prod on Go)

1. **DNS:** at the `ritme.app` DNS provider add `A  instructor  89.251.8.115` (DNS-only until the cert exists, same as
   above).
2. **Go profile on:** `COMPOSE_PROFILES=go` in `/opt/ritme/.env` (T-M2-26), then `./deploy.sh` — builds and starts
   `backend-go`, `admin-web` and `instructor-web`.
3. **:80 vhost:** as stage step 3 with `INSTRUCTOR_CONF=./deploy/instructor-http.conf`.
4. **Certificate + HTTPS:** `ssh root@89.251.8.115 'cd /opt/ritme && ./deploy/enable-https-instructor.sh prod'`
5. **Verify:** `https://instructor.ritme.app/login` → 200, `/api/instructor/v1/me` → 401 JSON, `/api/admin/v1/auth/me`
   → 404 (`./deploy.sh` checks these from now on).

### Before real instructors use it

- An admin must approve applications (B-N8-08 admin moderation) — until then every applicant sees «در انتظار تأیید».
- Server env at N8 rollout (QUESTIONS #121): `LEARNING_SMS_PROVIDER`, `KAVENEGAR_TEMPLATE_COURSE_UNLOCKED`.

## Rollback

Set the selector back to `./deploy/instructor-off.conf` (or delete the line) and `up -d proxy`; stop the container
with `docker compose … stop instructor-web`. Nothing else depends on it.

---
name: deploy-beta
description: Deploy Ritme beta to https://beta.ritme.app — a PUBLIC environment for real users that ships the current working tree (uncommitted edits included), as a third isolated compose project (ritme-beta) on the production server. Use when the user says "/deploy-beta", "دیپلوی بتا", "بتا رو آپدیت کن".
---

# Deploy Ritme beta

`https://beta.ritme.app` — real users, no password gate. Ships the **working tree** as it is on disk
(uncommitted edits included); `FROM_WORKTREE=0 ./deploy-beta.sh` ships the tip of the `stage` branch instead
(`BRANCH=x` for another branch).

```bash
./deploy-beta.sh                      # full
SERVICES="frontend" ./deploy-beta.sh  # one service
NO_BUILD=1 ./deploy-beta.sh           # restart only
```

## How it is separated

| | production | staging | beta |
| --- | --- | --- | --- |
| source | `/opt/ritme` | `/opt/ritme-stage` | `/opt/ritme-beta` |
| project / volumes | `ritme` | `ritme-stage` | `ritme-beta` (own DB `ritme_beta`) |
| APP_ENV | production | staging | **production** (fakes refused) |
| gate | — | Basic auth + cookie | **none** |
| nginx | `proxy-ssl.conf` | `STAGE_CONF` | `BETA_CONF=./deploy/beta-ssl.conf` + `vhost-beta.inc` / `vhost-beta-admin.inc` |
| admin | adpanell.ritme.app | `/panel` | **beta-admin.ritme.app** (admin-web at `/`, `ADMIN_HOSTS`) |

- Beta publishes no host ports; production's `ritme-proxy-1` reaches it on `ritme-edge` via the aliases
  `beta-backend-go`, `beta-frontend`, `beta-admin-web`, all as request-time variables — a down beta stack can
  never break the shared proxy's `nginx -t`.
- ArvanCloud is in front (TLS terminated there). Client IP = rightmost X-Forwarded-For (`$beta_client_ip`).
- The script never ships the working tree to `/opt/ritme`: it copies only the beta nginx files and adds the two
  mount lines to the server's `docker-compose.prod.yml` in place (the repo copy has them too).

## First deploy (automatic, once)

- `/opt/ritme-beta/.env` generated on the server (fresh secrets, SMS keys copied from staging).
  Admin login: `ssh root@89.251.8.115 cat /root/ritme-beta-credentials.txt`, panel at https://beta-admin.ritme.app
  (`/panel` and `/admin` on beta.ritme.app redirect there; `/api/admin/` is 404 on the app host).
- Empty DB bootstrapped with Laravel **before** backend-go first starts (marker `/opt/ritme-beta/.bootstrapped`);
  if backend-go already ran on the volume the script refuses rather than collide with goose.
- Certificates per host via `deploy/enable-https-beta.sh <host> <prefix>` (placeholder = prod cert until then).

## Real-user rules

- Never wipe `ritme-beta_*` volumes — they hold real user data. Never `down -v`.
- Payments / AI / companion SMS are `none` (503) until configured in `/opt/ritme-beta/.env`
  (`PAYMENT_PROVIDER=zarinpal` + merchant id, `AI_PROVIDER=gemini` + paid key, `COMPANION_SMS_PROVIDER=gateway`).
- Never rotate the `*_KEY` values without moving the old one to `*_PREVIOUS` first.

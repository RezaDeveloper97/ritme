---
name: deploy
description: Deploy Ritme to production (ritmeapp.ir / 62.60.198.240). Builds amd64 images locally, streams via docker save/ssh (server has no international network), restarts the stack, then verifies.
---

# Deploy Ritme to production

The prod server has ZERO international connectivity — images can never be built there. Everything is handled by `./deploy.sh` at the repo root.

## Pre-deploy checklist (do these BEFORE deploying)
1. Working tree committed and pushed to gitlab.norahand.com (the server does `git pull --ff-only`). Uncommitted/unpushed compose or config changes will NOT reach the server.
2. Run quick verification: backend `php artisan test`, frontend `npm run typecheck && npm run build` (a broken Next build fails mid-deploy after minutes of waiting).
3. Check `deploy.sh` env args are still correct — notably `OTP_TEST_MODE` (currently `true` because SMS providers are broken; must become `false` once SMS.ir/Kavenegar are fixed).
4. New migrations? They do NOT run automatically — after deploy, run them manually (step below).

## Deploy
```bash
./deploy.sh
```
This builds both images `--platform linux/amd64`, streams them (`docker save | gzip | ssh ... docker load`), then `git pull + docker compose up -d --no-build` on the server. Takes several minutes; the image transfer is the slow part.

## Post-deploy
1. If there are new migrations:
   `ssh -i ~/.ssh/id_ed25519 root@62.60.198.240 'cd /opt/ritme && docker compose exec -T backend php artisan migrate --force'`
2. New seeders (content/messages/banners) must also be run manually the same way (`db:seed --class=...`). Seeders are written idempotent — safe to re-run.
3. Verify: `curl -s https://ritmeapp.ir/up` (health) and `curl -s https://ritmeapp.ir/api/v1/banners` or another public endpoint; open https://ritmeapp.ir for the frontend.
4. If something is wrong, check logs: `ssh ... 'cd /opt/ritme && docker compose logs --tail=100 backend frontend proxy'`

## Notes
- Server-only files (`/opt/ritme/.env`, `proxy.conf`, `docker-compose.override.yml`) are NOT in git — never assume repo compose files match prod exactly.
- No HTTPS yet — port 80 only.
- Never run destructive artisan commands (`migrate:fresh`, `db:wipe`) on the server.

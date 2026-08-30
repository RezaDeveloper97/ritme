---
name: local-dev
description: Start the Ritme local dev environment (sqlite backend on :8010, Next.js frontend) and log in when the SMS gateway is unreachable. Use when running or manually testing the app locally.
---

# Ritme local development

## Backend (Laravel + sqlite)
```bash
cd backend && php artisan serve --port=8010
```
- `backend/.env` already uses sqlite (`database/database.sqlite`), `QUEUE_CONNECTION=sync`, file cache. Passport keys + personal client are already generated — don't regenerate.
- If the sqlite file is missing: `touch database/database.sqlite && php artisan migrate --seed`

## Frontend (Next.js)
```bash
cd frontend && npm run dev
```
- `frontend/.env.local` points `NEXT_PUBLIC_API_BASE_URL` to `http://127.0.0.1:8010/api/v1`. Keep it for local dev; it must NOT leak into prod builds (prod URL is baked in via deploy.sh build args).

## Login when SMS is unreachable
There is **no test mode and no bypass** — not in the request body, not in the
config, not in the environment. Every one of those is a way into every account,
so `send-otp` always generates a random code and always queues the SMS.

Read the code out of the local sqlite DB instead (it is stored in plain text,
which is exactly why this works locally and proves nothing about production):

```bash
cd backend && php artisan tinker --execute="echo App\Models\OtpVerification::latest()->value('code');"
```

Request the code in the UI, run that, then type what it prints. With
`QUEUE_CONNECTION=sync` the SMS job runs inline and fails against the
unreachable gateway; the controller catches that and still returns 200, so the
only sign of it is a line in the log. The OTP row is written before the job
runs, so the code is always there to read.

## E2E / browser automation
Skip the login UI entirely: set `localStorage.ritme_token` via `addInitScript` plus cookie `ritme_auth=1` (get a real token first via the OTP API with 1111).

## Rules
- Never point local frontend/tools at the prod API for destructive flows (account delete, data wipes).
- Admin panel is a separate container gated by `ADMIN_PANEL_ENABLED` — locally enable it via env, not code changes.

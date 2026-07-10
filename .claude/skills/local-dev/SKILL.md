---
name: local-dev
description: Start the Ritme local dev environment (sqlite backend on :8010, Next.js frontend, test OTP 1111) and log in without SMS. Use when running or manually testing the app locally.
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

## Login without SMS (providers unreachable locally)
`POST /api/v1/auth/send-otp` with `{"is_test": true}` → OTP is always `1111`. In the UI, request the code then enter 1111.

## E2E / browser automation
Skip the login UI entirely: set `localStorage.ritme_token` via `addInitScript` plus cookie `ritme_auth=1` (get a real token first via the OTP API with 1111).

## Rules
- Never point local frontend/tools at the prod API for destructive flows (account delete, data wipes).
- Admin panel is a separate container gated by `ADMIN_PANEL_ENABLED` — locally enable it via env, not code changes.

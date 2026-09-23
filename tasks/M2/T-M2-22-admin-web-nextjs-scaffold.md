---
id: T-M2-22
title: admin-web — Next.js admin app scaffold
milestone: M2
type: frontend
status: done
depends_on: [T-M2-20]
parallel_group: M2-F
touches: [admin-web, docker-compose.yml, docker-compose.stage.yml, docker-compose.prod.yml, docs/go-migration/admin-web.md]
skills: [frontend-design:frontend-design]
verify: cd admin-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# T-M2-22 — admin-web: Next.js admin app scaffold

## Why
The user chose a new Next.js admin instead of porting Blade. It is a separate app so the user-facing PWA bundle
stays untouched. It talks to the Go admin API (T-M2-20/21) with cookie sessions + CSRF.

## Scope
1. `admin-web/`: Next.js (same major as `frontend/`), TypeScript strict, App Router, Tailwind, the same lint/format
   setup as `frontend/` where it applies; Persian-first **RTL** layout (Vazirmatn variable font like the frontend),
   light/dark tokens reusing the brand palette (see `frontend/CLAUDE.md` §10.2) — admin UI can be denser/simpler.
2. Structure (FSD-lite): `shared/api` (fetch client with `credentials: 'include'`, CSRF header, 401 → login redirect,
   typed envelopes), `shared/ui` (table with server pagination/filter, form fields, toasts, confirm dialog, image
   upload with preview, **translatable field** = one input per active language with only the default required,
   rich-text editor (TipTap or CKEditor 5 React; output must pass the backend sanitizer)), `features/auth`
   (login page, logout, me), `widgets/shell` (sidebar with role-aware items, header).
3. Dashboard page wired to the real API as the reference implementation.
4. Dockerfile (standalone output), compose service `admin-web` (internal only; nginx on the admin host proxies `/`
   to admin-web and `/api/admin/` to backend-go — wired in T-M2-25), env `NEXT_PUBLIC_ADMIN_API_BASE_URL`.
5. `docs/go-migration/admin-web.md`: architecture, how to add a CRUD screen.

## Out of scope
All other screens (T-M2-23). nginx cutover of the admin host (T-M2-25).

## Acceptance
- `verify:` green; login → dashboard → logout works against a local backend-go with a seeded admin.
- The translatable field renders one input per active language from `/api/v1/languages` and marks only the default
  as required.
- RTL/LTR and dark mode render correctly (screenshot in PROGRESS).

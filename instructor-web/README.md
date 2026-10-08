# instructor-web

Ritme's instructor panel (bloom B-N8-05): Next.js 15, TypeScript, FSD (`steiger`), vitest, eslint — the same tooling as
`admin-web/`. Persian/RTL, Night & Bloom light + dark tokens (a copy of the user app's token layer in
`src/app/globals.css`; apps never import each other), mobile-first (390) and responsive (side rail from 1024px).

- **Auth:** OTP through the shared user API (`POST /api/v1/auth/send-otp` + `/verify-otp`), bearer token in
  localStorage (`ritme_instructor_token`, this origin only), then `GET /api/instructor/v1/me` routes to `/apply`
  (never applied or revoked → `POST /apply`), `/pending` (awaiting admin approval) or the panel. 403
  `instructor_required|instructor_pending` from any panel call re-reads `/me`.
- **Panel:** shell + empty dashboard and placeholder tabs only; B-N8-06 / B-N8-07 build the content.

## Develop

```bash
npm install
INSTRUCTOR_API_PROXY_TARGET=http://127.0.0.1:8020 npm run dev   # http://localhost:3002, /api proxied to backend-go
npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
```

## Deploy

Compose service `instructor-web` (stage: `instructor.ritmeapp.ir`, prod: `instructor.ritme.app`), nginx vhosts
`deploy/vhost-instructor*.inc`. DNS, certificates and switching the vhosts on are manual steps:
**[docs/night-bloom/instructor-web-deploy.md](../docs/night-bloom/instructor-web-deploy.md)**.

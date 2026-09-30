---
id: B-N8-05
title: instructor-web — new app scaffold, auth, deploy (instructor.ritmeapp.ir / instructor.ritme.app)
milestone: N8
type: fullstack
status: todo
depends_on: [B-N8-01]
parallel_group: N8-E
touches: [instructor-web,deploy,docker-compose.stage.yml,docker-compose.prod.yml,deploy.sh,deploy-stage.sh,backend-go/internal/auth]
skills: [verify-all]
verify: cd instructor-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# B-N8-05 — instructor-web — new app scaffold, auth, deploy (instructor.ritmeapp.ir / instructor.ritme.app)

## Why
User decision: the instructor panel is a separate app on its own subdomain.

## Scope
- Next.js app mirroring admin-web tooling (TS, FSD, steiger, vitest, eslint), fa RTL, Night & Bloom light/dark tokens, mobile-first per artboards (390) and responsive to desktop.
- Login: OTP for users with the instructor role (reuse `/auth/send-otp`), instructor-scoped API under `/api/instructor/v1`.
- Compose services + nginx vhosts: stage `instructor.ritmeapp.ir` (behind the same Basic auth as stage) and prod `instructor.ritme.app`. DNS records + TLS certs are outward actions: list the exact steps for the user and stop for approval before touching the server/DNS.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- App builds and serves locally; vhost configs pass `nginx -t`
- Deploy steps documented in `instructor-web/README.md`
- `verify` green

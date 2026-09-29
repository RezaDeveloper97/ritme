---
id: T-M7-19
title: Security fixes from the M3-M7 audit
milestone: M7
type: frontend
status: todo
depends_on: [T-M4-12,T-M2-34,T-M7-18]
parallel_group: M7-E
touches: [frontend/src/entities/checkup,frontend/src/shared/lib,frontend/src/shared/session,frontend/src/screens/checkup-history,frontend/src/screens/checkup-mark-done,frontend/src/features/record-checkup,frontend/src/widgets/checkups-card,frontend/src/screens/pregnancy-calendar,frontend/src/screens/reminder-appointment-form,frontend/src/app/[locale]/reminders/appointment/new/page.tsx,frontend/next.config.ts,backend-go/internal/care,backend-go/internal/checkups,backend-go/internal/http,backend-go/internal/platform/ratelimit,backend-go/api/openapi.yaml,docs/security]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run test && npm run build && cd ../backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all
---

# T-M7-19 — Security fixes from the M3-M7 audit

## Why
`docs/security/audit-m3-m7.md` (2026-09-29): 1 high, 2 med, 2 low.

## Scope
1. (high) On-device checkup attachments cleared on session end (`onSessionEnd`), and when a custom checkup type is
   deleted, its records' files too. Test.
2. (med) Attachment allow-list (jpeg/png/webp/heic/heif/pdf) enforced in `matchesAccept` and `<input accept>`;
   opening: images via `<img>`/allow-listed blob only, everything else forced download. Move the frontend CSP from
   Report-Only to enforced — only after checking the app works with it locally (build + headless run of home, a
   sheet, PDF export, attachment open); if something needs a directive, add it narrowly. Test.
3. (med) No health data in URLs: replace the `?title=…&topic=…&care_item_key=…&date=…` appointment prefill with a
   one-time prefill held in memory/`sessionStorage` under a random id (`?prefill=<id>`), or an opaque catalog id.
   Keep the T-M7-17 topic behaviour. Tests.
4. (low) ICS escaping (`;`, bare CR, control chars). Test.
5. (low) Per-user caps on active medications/appointments/custom checkups (422 with a fa/en message) and a per-user
   write rate limit on `/api/v1/*` writes via `platform/ratelimit`. OpenAPI updated; integration test.

## Acceptance
- Each finding fixed with tests; audit doc gets a Resolution column; verify green (incl. `npm run build`).

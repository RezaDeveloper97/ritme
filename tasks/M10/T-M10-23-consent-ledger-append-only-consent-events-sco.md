---
id: T-M10-23
title: Consent ledger: append-only consent events, scope, versioned signup consents
milestone: M10
type: fullstack
status: todo
depends_on: [T-M10-01]
parallel_group: M10-C
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/consent, backend-go/internal/consent, backend-go/internal/profile/consents.go, backend-go/internal/auth/handlers.go, backend-go/internal/assistant/consent.go, backend-go/internal/telemed/consent.go, backend-go/internal/sharelinks/service.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, frontend/src/screens/auth-signup, frontend/src/features/auth, frontend/src/screens/privacy, frontend/src/entities/user, frontend/messages/fa/auth.json, frontend/messages/en/auth.json, frontend/messages/fa/account.json, frontend/messages/en/account.json, backend-go/resources/translations/fa/auth.json, backend-go/resources/translations/en/auth.json, backend-go/resources/translations/fa/account.json, backend-go/resources/translations/en/account.json]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/consent/... ./internal/profile/... ./internal/auth/... && make test-int PKG="./internal/consent/... ./internal/auth/... ./db/migrations/..." && make contract ROUTES=consents,privacy,auth && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run copy:check -- auth account
---

# T-M10-23 — Consent ledger: append-only consent events, scope, versioned signup consents
**Model:** opus (consent is legal evidence). Runs alone in a batch (migration).

## Why
SPEC §4.2: `consent(user_id, type, version, granted_at, revoked_at, scope_json)` for every consent, including terms +
health-data processing at signup. Today `user_consents` keeps one mutable row per (user, code) (UNIQUE), so granting v2
overwrites what v1 recorded; there is no `scope_json`; and nothing records the signup consents at all — the two
checkboxes on the signup page are client-only (`screens/auth-signup/ui/SignupPage.tsx:33-37`; the gap report's
`users.terms_accepted_at` is a city-services column, not a users column). DECISIONS.md H-13, H-18; D-174.

## Scope
1. Migration + Laravel twin: `consent_events (id BIGINT UNSIGNED AI PK, user_id FK users ON DELETE CASCADE, consent
   VARCHAR(64), version INT UNSIGNED, action VARCHAR(8) (grant | revoke), scope JSON NULL, source VARCHAR(24)
   (signup | settings | feature | family | telemed | sharelink | backfill | admin), created_at,
   KEY (user_id, consent, id))`; triggers `BEFORE UPDATE` and `BEFORE DELETE` → SIGNAL (cascades from a user delete
   do not fire triggers, so account deletion still works — prove it in a test).
2. Backfill: one `grant` event per granted `user_consents` row (`created_at = granted_at`, source `backfill`) and a
   `revoke` event when `revoked_at` is set.
3. `consent.Service`: every grant/revoke writes the event and updates the `user_consents` projection in one
   transaction; `Require` is unchanged (projection, version-aware).
4. New codes `terms` and `health_data_processing` (mandatory, versioned, texts fa/en in `consent/lang`); the signup
   request (OTP verify for a new user) carries `consents: {terms: v, health_data_processing: v}` → events with source
   `signup`; a new Go signup without them → 422 `consent_required` (the Laravel API ignores the field).
5. Existing users (H-18): `GET /api/v1/consents/pending` lists mandatory codes not yet granted at the current version;
   the frontend shows a one-time sheet on the next app open; nothing is blocked server-side while it is pending.
6. Scoped events (evidence only, no behaviour change): assistant per-item switches (`scope` = the items),
   telemed share per booking (`{booking_id, doctor_id}`), share links (`{link_id, expires_at}`); family scopes come in
   T-M11-08.
7. `GET /api/v1/consents/history` (own events, newest first); the privacy screen lists it.
8. Frontend: signup sends the versions of the two texts shown; the one-time sheet; the history list; fa/en.

## Tests
- grant → revoke → grant = 3 events and projection granted at the latest version; an outdated version → 403
  `outdated` (unchanged).
- UPDATE or DELETE on `consent_events` → error; deleting the user removes their events (cascade).
- Backfill counts on fixtures; signup with and without consents; pending list for an old user; telemed and share-link
  events carry their scope.
- Frontend vitest: signup payload, pending sheet.

## Out of scope
- Family consents (T-M11-08); changing any feature's gate.

## Acceptance
- Verify passes; PROGRESS.md cites D-174.

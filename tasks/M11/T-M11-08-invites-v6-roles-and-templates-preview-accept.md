---
id: T-M11-08
title: Invites v6: roles and templates, preview, accept or decline, mutual request, phone auto-bind, owner confirmation, share requests
milestone: M11
type: backend
status: blocked
depends_on: [T-M11-05]
parallel_group: M11-C
touches: [backend-go/internal/family/invites, backend-go/internal/family/sharerequests, backend-go/internal/companion, backend-go/internal/http/routes_family.go, backend-go/internal/http/routes_companion.go, backend-go/internal/http/routes_auth.go, backend-go/internal/http/companion_guards.go, backend-go/internal/platform/config/config.go, backend-go/db/queries/companion, backend-go/api/openapi.yaml, backend-go/contract/allowlist, docs/engine-v6/jobs.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/family/... ./internal/companion/... && make test-int PKG="./internal/family/... ./internal/companion/... ./internal/http/..." && make contract ROUTES=companion,auth
---

# T-M11-08 — Invites v6: roles and templates, preview, accept or decline, mutual request, phone auto-bind, owner confirmation, share requests
**Model:** opus (invite codes are an attack surface; security review findings CMP-L2/L3/L8 apply).

## Why
SPEC §1.7–§1.8: the owner picks a role and template, the member previews what she shares and may accept or decline,
can answer a mutual request, a phone invite binds on signup, and either side may ask for more access, applied only
after the owner approves. Today accepting a code activates immediately, there is no preview, decline, mutual request,
phone auto-bind or share request, and stage keys codes with the public default pepper (gap 05 §2.6; FAM-M5, FAM-L1,
FAM-L3). DECISIONS.md F-11, F-12, F-13, F-17; D-196.

## Scope
1. `POST /api/v1/family/members` (owner): relation/role, template, row edits, `expires_at` (carer default F-13),
   urgent switch and `urgent_sms`, optional `mutual_request {domains}`, phone or code, consent text version → link
   `invited`, invite (code 24 h, phone 7 days), grants through `Grants.Apply`.
2. `POST /api/v1/family/invites/preview {code}`: same per-user/IP throttles, global breaker and generic refusal as
   accept (preview is another code oracle); returns owner first name, role, domain summary and the mutual request.
3. `POST /api/v1/family/invites/accept {code, mutual?}`: active, or `awaiting_owner` when a code-only invite
   includes children write, urgent alert or vitals (F-11); a mutual answer creates an independent B→A link with B's
   grants and B's consent; accepting without mutual is allowed.
4. `POST /api/v1/family/invites/decline {code}` → `revoked_by = declined` (F-12).
5. Phone auto-bind: an `auth` signup hook binds pending phone invites to the new account as pending acceptance;
   `GET /api/v1/family/invites/pending` lets the member accept or decline without the code.
6. `POST /api/v1/family/members/{linkId}/confirm` (owner) for `awaiting_owner`; the member sees nothing before it.
7. Share requests: `POST /api/v1/family/share-requests {to_user_id, domains}` (either side; one per domain per pair
   per 7 days → 429), `GET …?box=in|out`, `POST …/{id}/approve|decline` — only the data owner approves; approval
   applies through `Grants.Apply`; nothing changes before.
8. The owner cancels an invite any time. Daily job `family.invites_sweep` (queue v2): expired invites and carer links
   past `expires_at` → `revoked_by = expired`, activity row, owner notice.
9. Hardening: invite pepper mandatory outside local/testing/contract (fail at start-up, FAM-L1); constant-time accept
   and preview paths (CMP-L2); phone-bound lock DoS mitigation per `docs/security/bloom-companion.md` (CMP-L8).
10. `/companions/*` invite and accept run on the new service (allow-listed, D-196).

## Tests
- `TestInvite_AcceptWithoutMutual`, `TestFamily_MutualLinksIndependent` (revoking A→B leaves B→A, its grants and
  activity untouched), `TestInvite_PreviewIsThrottledAndGeneric`, `TestInvite_DeclineMarksRevoked`,
  `TestSignup_BindsPhoneInviteAsPending`, `TestInvite_PhoneTTL7Days`, `TestInvite_HighImpactCodeNeedsOwnerConfirm`.
- `TestShareRequest_NoChangeUntilOwnerApproves`, `TestShareRequest_RateLimit`, `TestShareRequest_IDOR`.
- `TestConsent_RecordedPerInviteAndChange`, `TestCarerExpiry_SweepRevokesAndLogs`, `TestPepper_RequiredOutsideLocal`.
- Existing invite tests (single use, expiry, concurrent redeem, lock after 5) green on the new service.

## Out of scope
- Screens (T-M11-17, T-M11-18).

## Acceptance
- Verify passes; PROGRESS.md cites D-196.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.

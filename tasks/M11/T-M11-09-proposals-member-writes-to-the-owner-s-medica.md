---
id: T-M11-09
title: Proposals: member writes to the owner's medications and appointments wait for her approval
milestone: M11
type: fullstack
status: blocked
depends_on: [T-M11-06]
parallel_group: M11-C
touches: [backend-go/internal/family/proposals, backend-go/internal/family/seal, backend-go/internal/care/delegation.go, backend-go/internal/care/handlers.go, backend-go/internal/care/appointment_handlers.go, backend-go/internal/http/routes_family.go, backend-go/internal/http/routes_care.go, backend-go/internal/platform/config/config.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, docs/engine-v6/jobs.md, frontend/src/features/manage-medication, frontend/src/features/manage-appointment, frontend/messages/fa/care.json, frontend/messages/en/care.json, backend-go/resources/translations/fa/care.json, backend-go/resources/translations/en/care.json]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/family/... ./internal/care/... && make test-int PKG="./internal/family/... ./internal/care/... ./internal/http/..." && make contract ROUTES=reminders && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run copy:check -- care
---

# T-M11-09 — Proposals: member writes to the owner's medications and appointments wait for her approval
**Model:** opus (writes into another person's health records).

## Why
SPEC §1.8: a member's new medication or appointment for the owner is a proposal (pending → approved | rejected |
expired after 7 days) and a medication proposal never activates a reminder without her approval. Today a member with
`edit` writes straight into her reminders, can pause or resume them, and can move a telemed-booked visit (gap 05 §2.7,
FAM-H2). DECISIONS.md F-04, F-05, F-14, F-19; D-197.

## Scope
1. `internal/family/seal`: AES-256-GCM, keys from `FAMILY_SEAL_KEYS` (id → key; newest encrypts, all decrypt),
   mandatory outside local/testing (fail at start-up); AAD binds the payload to its row.
2. Member `POST/PUT /care/medications|appointments?for_user_id=` → `202 {proposal}` (kinds create_med,
   create_appointment, update_appointment) with the payload sealed; members never change `is_active`, `notify`,
   pause or resume on her records (403 `family_owner_only`).
3. Owner: `GET /api/v1/family/proposals?status=pending`, `POST /api/v1/family/proposals/{id}/approve|reject` —
   approver must be the caller (else 404), `RecheckTx` for link and level, apply as the owner through the care service
   with `created_by_member` provenance, activity rows, a neutral in-app notice to the member.
4. Member: `GET /api/v1/family/proposals/mine`, `POST /api/v1/family/proposals/{id}/cancel`.
5. Expiry: lazy (a pending proposal older than 7 days reads as expired; approve → 409 `proposal_expired`) plus daily
   job `family.proposals_expire`, which also purges sealed payloads 30 days after the decision (F-14, F-19).
6. Frontend: the «ثبت برای چه کسی؟» flow in `manage-medication` and `manage-appointment` handles 202 → «برای تأیید
   فرستاده شد» (Laravel's 201 path keeps working); fa/en.

## Tests
- `TestProposal_MedicationNotAppliedUntilApproved` (her reminders and today's doses unchanged; approve → one active
  reminder; reject/expire → none), `TestProposal_ExpiresAfter7Days` (lazy and job, fake clock), `TestProposal_IDOR`,
  `TestProposal_RevokeDuringApprove`, `TestProposal_ApproveTwiceIdempotent`, `TestMember_CannotPauseOwnerReminder`,
  `TestProposal_PayloadSealedAtRest` (raw column holds no plaintext), `TestSeal_KeyRotation`.
- Frontend vitest: the 202 path.

## Out of scope
- Owner approval screen (T-M11-19); child proposals (T-M11-11).

## Acceptance
- Verify passes; PROGRESS.md cites D-197.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.

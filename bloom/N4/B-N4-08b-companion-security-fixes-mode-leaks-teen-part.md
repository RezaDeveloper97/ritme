---
id: B-N4-08b
title: Companion security fixes (mode leaks, teen partner block, delegated reads, brute force, audit flood)
milestone: N4
type: backend
status: todo
depends_on: [B-N4-08]
parallel_group: N4-H2
touches: [backend-go/internal/companion,backend-go/internal/care,backend-go/internal/http,backend-go/internal/admin/companions,backend-go/api,backend-go/contract]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N4-08b — Companion security fixes (mode leaks, teen partner block, delegated reads, brute force, audit flood)

## Why
Findings of the companion security review (`docs/security/bloom-companion.md`).

## Scope
- **CMP-H1 (high, release blocker):** pregnancy/postpartum leak through the cycle and symptoms views without a pregnancy grant (`companion/shared/shared.go`): resolve the owner's life mode in `Reader.Read`; pregnancy/postpartum/menopause without a pregnancy grant → neutral cycle view (`has_data:false`, numbers null); symptoms keep only options available in cycle mode; `days_late` beyond `maxLateDays` → null. Integration tests.
- **CMP-M1 (medium, blocker):** CB-TEEN-01 already blocks new partner/spouse invites/accepts for teen owners (`teen_parent_only`, `invite_not_allowed`). Remaining here: map/null the fertile phase in companion views when `NoFertilityCopy`; existing ACTIVE partner/spouse links of an owner who switches to teen are not paused/revoked — product decision pending (QUESTIONS #98): default = hide all sections of those links while the owner is in teen mode (no revoke).
- **CMP-M2:** delegated single-record reads (`for_user_id`) return inactive meds / past or cancelled appointments — 404 when subject≠actor and the record is outside the shared view (also on update); throttle delegated GETs.
- **CMP-M3:** blind guessing of code-only invites — add a global failed-accept circuit breaker (and/or default to phone-bound invites); document the choice.
- **CMP-L1:** coalesce companion read audits per section within 15 min; audit endpoint pagination + action filter. **CMP-L5:** truncate/clean companion names in owner notifications. **CMP-L6:** admin links overview `kit.Super`, stronger phone mask, no raw user ids.
- Rebase on the canvas CB-TEEN-01 (`parent` link type) and CB-LOSS-01 (private appointments) changes in `internal/companion` first.

## Out of scope
- 

## Acceptance
- Each finding fixed with tests; `docs/security/bloom-companion.md` findings table statuses updated (fixed / accepted)
- `verify` green

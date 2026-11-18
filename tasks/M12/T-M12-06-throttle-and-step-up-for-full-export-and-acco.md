---
id: T-M12-06
title: Throttle and step-up for full export and account delete; throttle /docs basic auth
milestone: M12
type: backend
status: done
depends_on: [T-M12-01]
parallel_group: M12-SEC
touches: [backend-go/internal/auth, backend-go/internal/http/routes_auth.go, docs/go-migration/deviations.md, frontend/src/shared/api, backend-go/internal/http/routes_profile.go, backend-go/internal/http/routes_docs.go, backend-go/internal/profile, backend-go/internal/accountexport, backend-go/api/openapi.yaml, frontend/src/features/account-delete, frontend/src/features/account-export, frontend/messages/fa/profile.json, frontend/messages/en/profile.json]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && make test-int PKG="./internal/http/... ./internal/profile/... ./internal/accountexport/..."
---

# T-M12-06 — Throttle and step-up for full export and account delete; throttle /docs basic auth
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-10/S-11: a stolen year-long token can pull every health record or delete the account at once; export is a cheap heavy-query DoS; `/docs` password can be guessed without limit.

## Scope
1. Named limiter: export 3 / hour / user, delete 3 / hour / user, `/docs` 10 failed attempts / minute / IP.
2. Ask the user (AskUserQuestion) whether delete and export need a fresh OTP step-up. If yes: a short-lived step-up token from `/verify-otp` with `purpose=step_up`, required by both routes; frontend sheets ask for the code (no autopilot name on these controls).
3. Bound the export queries (no unbounded `SELECT *` of blob columns; stream or page large tables).

## Out of scope
- Moving export to the worker (only if the throttle is not enough).
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Integration tests: 4th export in an hour → 429; 11th wrong `/docs` password → 429; (if step-up chosen) delete without a fresh step-up → 403 with a managed error code, with it → 204.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).


## Decision (user, 2026-11-18)
Step-up OTP required for **both** account delete and full export (plus the 3/hour limits).

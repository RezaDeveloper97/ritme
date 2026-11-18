---
id: T-M12-11b
title: Fix the 10 integration tests red on HEAD with companion_forbidden in internal/http
milestone: M12
type: backend
status: todo
depends_on: []
parallel_group: M12-TEST
touches: [backend-go/internal/http,backend-go/internal/companion,backend-go/internal/family]
skills: []
verify: cd backend-go && make test-int PKG="./internal/http/..."
---

# T-M12-11b — Fix the 10 integration tests red on HEAD with companion_forbidden in internal/http

**Model:** opus. Found while verifying T-M12-01 (2026-11-18); reproduced on a clean HEAD worktree.

## Why
Ten integration tests in `internal/http` fail on `stage` HEAD with `403 companion_forbidden`; CI never runs integration
tests (T-M12-11), so nobody saw it. Either the family/companion authorizer now denies paths it should allow
(a real regression for users who share with family) or the tests encode an outdated contract.

Failing: `TestCompanionSecurity_*` (4), `TestChildrenRoutes_SharedThroughCompanionLink`,
`TestCompanionHome_AccessCombinations`, `TestCompanionRoutes_DelegatedWriteAuditFailClosed`,
`TestLoss_PrivacyOfFollowupsAndNotices`, `TestRecordSharing_FamilyScopeNeverDocuments`,
`TestShopChecklist_ProductionWiring`.

## Scope
1. `git log -S`/bisect to the commit that turned them red; decide per test: code regression or stale test.
2. Code regression → fix the code (authorizer, projections, wiring). Stale test → update it only when the change is
   backed by a decision in `docs/engine-v6/family.md` / DECISIONS (F rows); otherwise ask the user.
3. Check T-M11-08 (in_progress, invites v6) is not the cause or the owner of these; if it is, note it in that task.

## Out of scope
- New family features.

## Acceptance
- `make test-int PKG="./internal/http/..."` green; each test's resolution (regression/stale + commit) in PROGRESS.md.

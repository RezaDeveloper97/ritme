---
id: T-M11-16
title: Owner's Plus for view-level members (optional, per F-18)
milestone: M11
type: backend
status: blocked
depends_on: [T-M11-06, T-M10-29]
parallel_group: M11-C
touches: [backend-go/internal/family/projections, backend-go/internal/family/plus.go, backend-go/internal/healthrecord, backend-go/internal/http/routes_family.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/family/... && make test-int PKG="./internal/family/... ./internal/healthrecord/..."
---

# T-M11-16 — Owner's Plus for view-level members (optional, per F-18)
**Model:** opus. Skip (mark `skipped` in PROGRESS.md) if the user said no to F-18 in T-M11-01.

## Why
SPEC §1.8: when the owner has Plus, Plus features about shared data (for example a PDF report) are also available
to view-level members. Plus is per user today and nothing reads the owner's status in a member request (gap 05
§2.16). DECISIONS.md F-18; D-206.

## Scope
1. Resolve the owner's Plus status once per member request (request memo).
2. Unlock in member projections the Plus analyses of domains the member already sees (the vitals trend when vitals is
   view).
3. `GET /api/v1/family/owners/{ownerId}/report.pdf`: a PDF summary of only the domains the member sees, rendered
   through the health-record export with a domain filter; never the full record and never never-shared data;
   audited.

## Tests
- `TestPlusForMember_OnlySharedDomains`, `TestPlusForMember_OwnerWithoutPlusLocked`,
  `TestMemberPDF_ContainsOnlySharedDomains` (text scan for sentinel values of every other category).

## Out of scope
- Plus purchase by members.

## Acceptance
- Verify passes; PROGRESS.md cites D-206 (or the task is marked skipped per F-18).

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.

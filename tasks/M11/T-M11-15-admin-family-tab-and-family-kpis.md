---
id: T-M11-15
title: Admin family tab and family KPIs
milestone: M11
type: backend
status: blocked
depends_on: [T-M11-08, T-M11-09, T-M11-10, T-M10-30, T-M10-33]
parallel_group: M11-C
touches: [backend-go/internal/admin/users, backend-go/internal/admin/companions, backend-go/internal/metrics, backend-go/internal/okr, backend-go/db/queries/admin, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/admin/... ./internal/metrics/... && make test-int PKG="./internal/admin/users/... ./internal/metrics/..."
---

# T-M11-15 — Admin family tab and family KPIs
**Model:** opus (admin access to family data).

## Why
SPEC §1.8 and §5.6: the admin user detail has a "Family" tab with member counts, roles and invite statuses — never
grant contents — and the KPIs include invite acceptance, average members per circle, proposal approval and doses
logged after a nudge. Today only an active-companion count exists, through raw SQL (gap 05 §2.15, FAM-I1).
DECISIONS.md D-205.

## Scope
1. `GET /api/admin/v1/users/{id}/family` (users view capability, T-M10-33): links as owner and as member (masked names,
   role, status, invite status pending | expired | accepted | declined), guardian counts, open proposal count — no
   domains, levels or content; the raw SQL in `admin/users/v2.go` moves to sqlc.
2. KPIs in the metrics snapshot (T-M10-30): invite acceptance rate (accepted ÷ accepted + declined + expired, invites
   created in the period), average members per circle (owners with ≥ 1 active link), proposal approval rate (approved
   ÷ decided), dose logged within 3 h after a nudge; small-cohort suppression; the four keys join the KR catalogue and
   replace the `null` placeholders.

## Tests
- `TestAdminFamilyTab_NoGrantContents` (JSON scan for domain codes and level names), `TestAdminFamilyTab_RoleGated`
  (analyst → 403), `TestFamilyKPIs_Definitions` (fixtures with known rates), `TestFamilyKPIs_SuppressedBelowCohort`.

## Out of scope
- Admin-web (T-M11-22).

## Acceptance
- Verify passes; PROGRESS.md cites D-205.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.

---
id: B-N9-02b
title: Admin RBAC security fixes — approver independence, user_goal as health, least-privilege defaults, no-store, real clock
milestone: N9
type: fullstack
status: in_progress
depends_on: [B-N9-02]
parallel_group: N9-B2
touches: [backend-go/internal/admin,admin-web/src/screens/access-control]
skills: [security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N9-02b — Admin RBAC security fixes — approver independence, user_goal as health, least-privilege defaults, no-store, real clock

## Why

## Scope
Security audit of 4f56f11d (B-N9-02):
- M1 approver independence: refuse (422) when the approver was created / had password or role changed by the requester
  in the last 30 days (from admin_audit_logs or an `admins.created_by` column — no new migration unless the orchestrator
  gives a number), approver active ≥24 h; alert log on every `health_access.approve`.
- M2 `user_goal` treated as health data: hidden in users list/detail without a grant, removed from the `user.update` audit.
- M3 least privilege: analyst + engineering `users: view`; unmapped `"?"` routes super-only; editor narrowed / editable
  (document the migration path for existing editors).
- L1 `Cache-Control: no-store` on all `/api/admin/v1` responses. L2 real clock (not X-Test-Now) in HealthGrant/decide.
  L4 failed logins audited (no email). L5 pending requests expire after 24 h; approval re-checks requester `health` cap.
  L6 audit view super-only by default (or hide ip for non-super). L7 csvSafe on every string column after TrimLeft.

## Out of scope
- 

## Acceptance
- 

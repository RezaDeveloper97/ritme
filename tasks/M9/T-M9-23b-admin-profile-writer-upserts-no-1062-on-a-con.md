---
id: T-M9-23b
title: Admin profile writer upserts (no 1062 on a concurrent first profile)
milestone: M9
type: backend
status: done
depends_on: [T-M9-23]
parallel_group: M9-A
touches: [backend-go/internal/admin/users,backend-go/db/queries/admin/core.sql,backend-go/internal/admin/store]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/admin/users/... && make test-int PKG=./internal/admin/users/...
---

# T-M9-23b — Admin profile writer upserts (no 1062 on a concurrent first profile)

## Why
T-M9-23 made `user_profiles.user_id` UNIQUE. The admin writer (`internal/admin/users/users.go` + `CreateProfilePlan` in `db/queries/admin/core.sql`) still reads `FirstProfileID` and then runs a plain INSERT. In a race it now fails with 1062 (where it used to create a second row).

## Scope
1. Switch the admin profile create to the same upsert as `profile.InsertProfile`: `INSERT … ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`.
2. Map any remaining duplicate-key error to the existing row, never a 500.

## Acceptance
- An integration test runs concurrent admin first-profile writes for one user: one row, no 500.

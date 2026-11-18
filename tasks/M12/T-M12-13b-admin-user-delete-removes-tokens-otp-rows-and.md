---
id: T-M12-13b
title: Admin user delete removes tokens, OTP rows and Redis keys like self-delete
milestone: M12
type: backend
status: todo
depends_on: [T-M12-13]
parallel_group: M12-TEST
touches: [backend-go/internal/admin/users,backend-go/internal/profile/account_keys.go]
skills: []
verify: cd backend-go && make test-int PKG="./internal/admin/users/... ./internal/profile/..."
---

# T-M12-13b — Admin user delete removes tokens, OTP rows and Redis keys like self-delete

**Model:** opus. Found in T-M12-13 (2026-11-18).

## Why
`internal/admin/users.Destroy` gets the new FK cascades but only *revokes* tokens (oauth_access/refresh rows stay),
keeps the user's OTP rows and leaves Redis keys (engine cache, idempotency responses, message state, family RA).

## Scope
1. Share one delete routine with `profile.Service.DeleteAccount` (tokens deleted, OTP rows for the mobile, files,
   `PurgeUserKeys`), called by both paths.
2. Extend `delete_complete_int_test.go`'s behaviour test (or a twin) to run through the admin delete route too.

## Acceptance
- Admin delete leaves the same zero-row / zero-key state as self-delete; test fails without the shared routine.

---
id: T-M12-13
title: Prove account delete removes every user row and file (schema-driven test) and fix the gaps
milestone: M12
type: backend
status: done
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/profile, backend-go/internal/profile/delete_complete_int_test.go, backend-go/db/migrations, backend-go/internal/files, docs/quality-2026-11/user-tables.md]
skills: []
verify: cd backend-go && make test-int PKG="./internal/profile/..."
---

# T-M12-13 — Prove account delete removes every user row and file (schema-driven test) and fix the gaps
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-03: delete relies on FK cascades plus hand-written deletes; ~139 `user_id` columns vs ~123 FKs, and the code admits tables with no FK. Leftover health rows after "delete my account" is a privacy and legal bug.

## Scope
1. Test 1 (schema): read `information_schema` for every column named `user_id`, `owner_id`, `member_id`, `child_owner_id`, `*_user_id`; each must have an `ON DELETE CASCADE` FK to `users` or be on an explicit allow-list in `docs/quality-2026-11/user-tables.md` with the reason and the code that deletes it. A new table without either fails the test.
2. Test 2 (behaviour): seed one row in every user table + one stored file for a user (and a second user as a control), call `DELETE /api/v1/account`, assert zero rows for the user, control untouched, files gone from storage, Redis keys for the user gone, tokens dead.
3. Fix every gap found: goose migration for missing FKs or hand deletes in the same transaction. Family links: the other side's data stays, the link goes.
4. Then: login with the same phone starts a fresh account (no old data reappears).

## Out of scope
- Legal retention rules for payments receipts — ask the user if a table must be kept (anonymised) rather than deleted.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Both tests green; allow-list documented; PROGRESS.md lists every table that was leaking before.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).

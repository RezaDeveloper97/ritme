---
id: T-M9-23
title: user_profiles.user_id UNIQUE with a safe dedupe
milestone: M9
type: backend
status: todo
depends_on: []
parallel_group: M9-A
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/profile/profile.sql, backend-go/internal/profile]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/profile/... && make test-int PKG="./internal/profile/... ./db/migrations/..."
---

# T-M9-23 — user_profiles.user_id UNIQUE with a safe dedupe
**Model:** opus. Runs alone in a batch (migration).

## Why
`user_profiles` has only a non-unique `KEY(user_id)`; the first save is a plain INSERT and every reader uses
`ORDER BY id LIMIT 1`, while some writers update `WHERE user_id = ?` (every row) and others `WHERE id = ?` (gap 02 X1).
Two concurrent first saves can create two rows; readers and writers then disagree. The v6 loader (T-M9-25) and the
revision row (T-M9-24) assume one profile per user. DECISIONS.md I-09.

## Scope
1. Count duplicates read-only: locally and on stage (`SELECT user_id, COUNT(*) … HAVING COUNT(*) > 1`, credentials
   from env). **Ask the user before running anything against beta.** Record the counts in PROGRESS.md.
2. Goose migration (next free number) + Laravel twin: if duplicates exist, keep the lowest id, fill its NULL columns
   from the newest duplicate, copy every removed row into `user_profiles_dedupe_backup` (same columns + `removed_at`),
   delete the removed rows, then `ALTER TABLE user_profiles ADD UNIQUE KEY user_profiles_user_id_unique (user_id)` and
   drop the old non-unique key if it becomes redundant. `down`: drop the unique key, restore the backup rows, drop the
   backup table.
3. The first-save insert becomes `INSERT … ON DUPLICATE KEY UPDATE` (or 1062 → re-read and update); writers that
   update `WHERE user_id = ?` stay correct.
4. Contract: profile routes unchanged (`make contract ROUTES=profile,onboarding`).

## Tests
- Migration integration test: a fixture with two rows for one user (different NULL columns) → one row with the merged
  values, one backup row; a clean fixture → no backup rows; `down` restores both rows.
- Concurrency: 10 goroutines doing a first profile save for a new user → exactly one row, no error.
- `make schema-diff` green.

## Out of scope
- Other schema changes (T-M9-24).

## Acceptance
- Counts recorded; migration up/down tested; PROGRESS.md cites D-127.

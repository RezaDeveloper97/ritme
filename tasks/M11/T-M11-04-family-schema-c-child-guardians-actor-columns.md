---
id: T-M11-04
title: Family schema C: child guardians, actor columns, child medication log, household to-do scope
milestone: M11
type: backend
status: todo
depends_on: [T-M11-03]
parallel_group: M11-B
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/children, backend-go/db/queries/babylog, backend-go/db/queries/todo, backend-go/contract/fixtures/dump.sql]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && make test-int PKG="./db/migrations/... ./internal/children/... ./internal/babylog/... ./internal/todo/..."
---

# T-M11-04 — Family schema C: child guardians, actor columns, child medication log, household to-do scope
**Model:** opus. Runs alone in a batch.

## Why
SPEC §1.7: a child profile has guardians, every write stores who made it, and the household to-do list is a shared
domain. Today a child has one `owner_id` and the spouse reads through `families/family_children`; no child or
baby-log table stores the actor; to-do is strictly personal; there is no child medication record (gap 05 §1 rows 5,
17, 18, 27, 33; §2.9). DECISIONS.md F-15, F-17, F-24, F-25, F-26; D-192.

## Scope
1. Goose migration + Laravel twin:
   - `child_guardians (id, child_id FK children ON DELETE CASCADE, user_id FK users ON DELETE CASCADE, level
     VARCHAR(8) (write | limited), sub_scopes JSON NULL, status VARCHAR(10) (invited | active | left | removed),
     invited_by BIGINT UNSIGNED NULL, link_id NULL FK companions ON DELETE SET NULL, created_at, updated_at;
     UNIQUE (child_id, user_id); KEY (user_id, status))`.
   - `created_by` and `updated_by` (`BIGINT UNSIGNED NULL`, FK users ON DELETE SET NULL) on `children`,
     `child_measurements`, `child_vaccine_doses`, `child_milestone_checks`, `baby_feeds`, `baby_sleeps`,
     `baby_diapers`, `todo_tasks`, `todo_items`.
   - `baby_med_doses (id, child_id FK children ON DELETE CASCADE, name VARCHAR(80), dose_text VARCHAR(40) NULL,
     given_at DATETIME, note VARCHAR(200) NULL, created_by, updated_by, created_at, updated_at; KEY (child_id,
     given_at))` (F-26).
   - `todo_tasks.scope VARCHAR(10) NOT NULL DEFAULT 'personal'` (personal | household), `assigned_to BIGINT UNSIGNED
     NULL` FK users ON DELETE SET NULL; KEY `(user_id, scope, done_at)`.
   - `children.owner_id` keeps its CASCADE FK: the frozen Laravel production code deletes accounts through it.
     T-M11-11 moves the child to a remaining guardian in Go before the user row is deleted.
2. Backfill in the goose file (idempotent `INSERT IGNORE … SELECT`): every child's owner → active `write` guardian;
   the spouse of every active spouse link with shared children (`families`/`family_children`) → active `write`
   guardian (F-17); `created_by = owner_id` is **not** backfilled (unknown history stays NULL, shown as "—").
3. sqlc queries: guardians by child set, guardians by user, guardian upsert/status change, med-dose CRUD, household
   list reads.

## Tests
- Up/down; backfill on fixtures (owner only; owner + spouse with two shared children; revoked spouse link → no
  guardian); a deleted user's actor columns become NULL; Laravel account deletion still removes a sole owner's child
  (cascade unchanged).

## Out of scope
- Behaviour (T-M11-11, T-M11-14).

## Acceptance
- Verify passes; `make schema-diff` green.

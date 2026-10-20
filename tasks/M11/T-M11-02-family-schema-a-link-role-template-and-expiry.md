---
id: T-M11-02
title: Family schema A: link role, template and expiry, grant mode epoch, owner mode state, activity columns
milestone: M11
type: backend
status: todo
depends_on: [T-M11-01, T-M9-02]
parallel_group: M11-B
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/companion, backend-go/contract/fixtures/dump.sql]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/companion/... && make test-int PKG="./db/migrations/... ./internal/companion/..."
---

# T-M11-02 — Family schema A: link role, template and expiry, grant mode epoch, owner mode state, activity columns
**Model:** opus. Runs alone in a batch (migration + `make sqlc`); T-M11-03 and T-M11-04 follow one after another.

## Why
SPEC §1.7 models a directional link `{owner, member, role, template, status, expires_at}`, one grant per domain and
an activity log with subject, payload reference and undo. `companions` has no role, template or expiry;
`companion_grants` cannot be paused on a mode change; `companion_audit_logs` has no subject, diff or undo columns
(gap 05 §1 rows 1–8, §2.1, §2.12). This task only **adds** columns and tables, so the current code keeps working;
the grant data migration ships with the code that reads it (T-M11-05). DECISIONS.md F-09, F-11, F-12, F-13, F-14,
F-17, F-23; D-190.

## Scope
1. Goose migration + Laravel twin:
   - `companions`: `role VARCHAR(16) NULL` (partner | other_parent | carer | friend | teen_parent), `template
     VARCHAR(24) NULL`, `expires_at TIMESTAMP NULL`, `urgent_sms TINYINT(1) NOT NULL DEFAULT 0`,
     `owner_confirmed_at TIMESTAMP NULL`; `status` admits `awaiting_owner`, `revoked_by` admits `expired | declined |
     system` (widen the columns if needed); KEYs `(companion_user_id, status, owner_id)`, `(owner_id,
     companion_user_id, status)`, `(status, expires_at)`.
   - `companion_grants`: `mode_epoch INT UNSIGNED NULL`; `section` and `level` wide enough for the v6 codes.
   - `family_owner_state (owner_id BIGINT UNSIGNED PRIMARY KEY, FK users ON DELETE CASCADE, mode VARCHAR(16) NOT NULL,
     mode_epoch INT UNSIGNED NOT NULL DEFAULT 1, review_pending TINYINT(1) NOT NULL DEFAULT 0, updated_at)`.
   - `companion_audit_logs` (the family activity log): `subject_type VARCHAR(16) NULL`, `subject_id BIGINT UNSIGNED
     NULL`, `diff_json JSON NULL`, `payload_sealed MEDIUMBLOB NULL`, `payload_key_id TINYINT UNSIGNED NULL`,
     `undo_deadline TIMESTAMP NULL`, `undone_at TIMESTAMP NULL`, `undone_by BIGINT UNSIGNED NULL`, `proposal_id BIGINT
     UNSIGNED NULL`; KEYs `(subject_type, subject_id, created_at)`, `(undo_deadline)`, `(created_at)`.
2. Backfill inside the goose file (idempotent; the Laravel twin is schema only): `role` from `type` (partner →
   partner, spouse → partner, parent → teen_parent; F-17), `template = 'legacy'`.
3. sqlc (`db/queries/companion`): the snapshot statement — links ⋈ grants ⋈ `family_owner_state` ⋈ `users.blocked_at`
   — for (a) one owner and one member and (b) every owner of one member; owner-state upsert with an epoch bump; the
   activity insert with the new columns.
4. `contract/fixtures/dump.sql` gains the new table and columns (D-47 precedent).

## Tests
- Migration up/down; backfill on seeded links of each legacy type; EXPLAIN of both snapshot statements uses the new
  composite indexes; the existing companion unit and integration tests stay green (no behaviour change).

## Out of scope
- Grant code/level migration and every behaviour change (T-M11-05).

## Acceptance
- Verify passes; `make schema-diff` green.

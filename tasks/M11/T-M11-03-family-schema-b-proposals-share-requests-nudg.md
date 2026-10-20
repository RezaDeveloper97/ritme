---
id: T-M11-03
title: Family schema B: proposals, share requests, nudges, member alerts, member notices, attendance
milestone: M11
type: backend
status: todo
depends_on: [T-M11-02]
parallel_group: M11-B
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/companion, backend-go/contract/fixtures/dump.sql]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && make test-int PKG="./db/migrations/... ./internal/companion/..."
---

# T-M11-03 — Family schema B: proposals, share requests, nudges, member alerts, member notices, attendance
**Model:** opus. Runs alone in a batch.

## Why
Proposals, share requests, nudges, member alerts and member notices have no storage (gap 05 §1 rows 9, 13–15, 21,
23; §5.4, §5.5). Member notices today are purged with unindexed JSON scans (FAM-L4). Family consents use the
generic `consent_events` ledger from T-M10-23, so no family consent table is added. DECISIONS.md F-06, F-09, F-12;
D-191.

## Scope
1. Goose migration + Laravel twin:
   - `family_proposals (id, subject_type VARCHAR(8) (owner | child), subject_id BIGINT UNSIGNED, link_id NULL FK
     companions ON DELETE CASCADE, actor_id FK users ON DELETE CASCADE, approver_id FK users ON DELETE CASCADE, kind
     VARCHAR(24) (create_med | create_appointment | update_appointment | child_delete | guardian_remove), domain
     VARCHAR(24), payload_sealed MEDIUMBLOB NULL, payload_key_id TINYINT UNSIGNED NULL, status VARCHAR(12) (pending |
     approved | rejected | expired | cancelled), dedupe_key VARCHAR(80) NULL, pending_key VARCHAR(80) AS (IF(status =
     'pending', dedupe_key, NULL)) STORED, expires_at, decided_by, decided_at, applied_ref VARCHAR(64) NULL,
     created_at, updated_at; UNIQUE (pending_key); KEYs (approver_id, status, created_at), (actor_id, status,
     created_at), (status, expires_at))`.
   - `family_share_requests (id, from_user_id FK users CASCADE, to_user_id FK users CASCADE, link_id NULL FK companions
     CASCADE, domains JSON NOT NULL, status VARCHAR(10) (pending | accepted | declined | expired), expires_at,
     decided_at, created_at; KEYs (to_user_id, status, created_at), (from_user_id, to_user_id, created_at))`.
   - `family_nudges (id, owner_id FK users CASCADE, member_id FK users CASCADE, link_id FK companions CASCADE,
     target_type VARCHAR(12) (dose | appointment), target_id BIGINT UNSIGNED, local_date DATE, slot VARCHAR(5), sent_at,
     UNIQUE (owner_id, target_type, target_id, local_date, slot), KEYs (member_id, owner_id, sent_at), (owner_id,
     sent_at))`.
   - `family_member_alerts (id, member_id FK users CASCADE, owner_id FK users CASCADE, link_id FK companions CASCADE,
     source_alert_id BIGINT UNSIGNED, source_hit INT UNSIGNED, level VARCHAR(12), created_at, seen_at NULL, sms_sent_at
     NULL; UNIQUE (member_id, source_alert_id, source_hit); KEYs (member_id, created_at), (link_id))`.
   - `family_member_notices (id, member_id FK users CASCADE, owner_id FK users CASCADE, link_id FK companions CASCADE,
     kind VARCHAR(24), dedupe_key VARCHAR(80), due_at, sent_at NULL, deferred_reason VARCHAR(24) NULL,
     notification_id BIGINT UNSIGNED NULL, created_at; UNIQUE (member_id, dedupe_key); KEYs (sent_at, due_at),
     (link_id))`.
   - `family_attendance (id, reminder_id FK reminders ON DELETE CASCADE, member_id FK users CASCADE, created_at;
     UNIQUE (reminder_id, member_id))` — "I'm coming" on the owner's appointment.
   - `user_notifications.link_id BIGINT UNSIGNED NULL` + KEY, FK companions ON DELETE CASCADE (member notices are
     purged by link, FAM-L4).
2. sqlc queries for each table (pending lists, lazy-expiry reads, quota counts, idempotent inserts, purge by link).

## Tests
- Up/down; deleting a link removes its proposals, nudges, member alerts, notices and linked notifications; a second
  pending child proposal with the same dedupe key fails while a decided one does not; `local_date`/`slot`
  uniqueness.

## Out of scope
- Behaviour (T-M11-08 … T-M11-13).

## Acceptance
- Verify passes; `make schema-diff` green.

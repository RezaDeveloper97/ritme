---
id: T-M10-17
title: Health schema for vitals targets, lab value confirmation and checkup test kind
milestone: M10
type: backend
status: todo
depends_on: [T-M10-01]
parallel_group: M10-C
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/vitals, backend-go/db/queries/labs, backend-go/db/queries/checkups]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && make test-int PKG=./db/migrations/...
---

# T-M10-17 — Health schema for vitals targets, lab value confirmation and checkup test kind
**Model:** opus. Runs alone in a batch (migration + `make sqlc`).

## Why
Three M10 health tasks need storage: per-user diabetic glucose targets set by a doctor (SPEC §3.2, T-M10-19), the
confirmation of each low-confidence lab value (SPEC §3.3, H-08, T-M10-20) and the kind of the last cervical test plus
informational checkup items (SPEC §3.4, H-10, H-11, T-M10-21). One migration keeps the `make sqlc` churn to one task
so the three logic tasks can run in parallel.

## Scope
1. Goose migration (next free number) + Laravel twin:
   - `vital_targets (id BIGINT UNSIGNED AI PK, user_id FK users ON DELETE CASCADE, context VARCHAR(16) NOT NULL
     (fasting | pre_meal | post_meal | bedtime), low_mgdl SMALLINT UNSIGNED NULL, high_mgdl SMALLINT UNSIGNED NULL,
     set_by VARCHAR(8) NOT NULL (doctor | user), note VARCHAR(160) NULL, created_at, updated_at,
     UNIQUE (user_id, context))`;
   - `lab_markers.confirmed_at TIMESTAMP NULL`; backfill: every marker of a report with `verified_at` gets
     `confirmed_at = verified_at` (today's all-or-nothing confirmation);
   - `checkup_records.test_kind VARCHAR(12) NULL` (cytology | hpv | cotest);
   - `checkup_types.informational TINYINT(1) NOT NULL DEFAULT 0`.
2. sqlc queries: `vital_targets` get/upsert/delete per user; `ConfirmLabMarker(id, user_id)`; record insert/update
   with `test_kind`; type reads with `informational`.
3. The diabetes flag already exists: `user_life_profiles.chronic_illnesses` contains `diabetes` — no new column.

## Tests
- Migration up/down; backfill sets `confirmed_at` only on verified reports; FK cascade on user delete;
  `make schema-diff` green.

## Out of scope
- Every behaviour change (T-M10-19, T-M10-20, T-M10-21); catalog data.

## Acceptance
- Verify passes.

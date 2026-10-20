---
id: T-M10-02
title: Message engine schema v6, versioned approval and message authority
milestone: M10
type: backend
status: todo
depends_on: [T-M10-01, T-M9-13, T-M9-38]
parallel_group: M10-B
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/messages, backend-go/db/queries/challenges, backend-go/internal/messages/rules, backend-go/internal/admin/messages, backend-go/contract/fixtures, backend-go/api/openapi.yaml]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/messages/... ./internal/admin/messages/... && make test-int PKG="./internal/messages/... ./internal/admin/messages/... ./db/migrations/..." && make contract ROUTES=messages
---

# T-M10-02 — Message engine schema v6, versioned approval and message authority
**Model:** opus. Runs alone in a batch (migration + `make sqlc`).

## Why
SPEC §5.3 defines the message model; today's `message_rules` (`db/migrations/00061_messages_v2.sql:15-47`) has one
cap per rule, exactly two variants, date-only schedules with one Tehran hour window, no `content`/`marketing` kinds,
no `approved_version`, and events without version or slot (gap 03 §1 rows 1a–1j, 2a–2g). Two behaviours hurt users
and safety today:
- Fixing a typo in an approved clinical message hides it from every user until re-approval, because `Served()` needs
  the head row approved and nothing keeps the approved version (`rules.go:166-174`, admin `rules.go:243-277`).
- Any admin with the `messages` capability can create, edit, switch off or delete a **safety** message; only approve
  and reject need `CapClinical` (admin `rules.go:37-50, 306-372`). SPEC §5.3/§5.8: only the medical advisor edits and
  publishes safety.
One migration carries every M10 message-engine table so the later tasks run without `make sqlc` churn.
DECISIONS.md M-01, M-02, M-03, M-06, M-07, M-17, M-18; D-160.

## Scope
1. Goose migration (next free number) + Laravel twin (`backend/database/migrations`, same columns, types, indexes):
   - `message_rules` gains: `rule_key VARCHAR(64) NULL` UNIQUE (stable key of seeded and code-referenced rules),
     `approved_version INT UNSIGNED NULL`, `topics JSON NULL`, `caps JSON NULL` (`[{"count":1,"period":"day|week|cycle"}]`),
     `variants JSON NULL` (`[{"key":"a","weight":50}]`), `starts_at DATETIME NULL`, `ends_at DATETIME NULL`,
     `hour_windows JSON NULL` (`[[8,12],[18,22]]`, user-local hours), `goal_event VARCHAR(48) NULL`,
     `locked TINYINT(1) NOT NULL DEFAULT 0`. The old columns (`cap_count/cap_period`, `variants_enabled/ab_split`,
     `starts_on/ends_on`, `hours_from/hours_to`, `goal`) stay and are read when the new column is NULL.
   - `message_rule_versions`: make sure the snapshot holds every served field (content, conditions, exceptions, level,
     kind, topics, variants); add what is missing.
   - `message_events`: `version INT UNSIGNED NULL`, `slot VARCHAR(32) NULL`, `variant` widened to `VARCHAR(16)`; new
     indexes `(user_id, created_at)` and `(created_at, rule_id)`; event value `eligible` allowed.
   - `message_event_daily (day DATE, rule_id BIGINT UNSIGNED, version INT UNSIGNED, variant VARCHAR(16),
     slot VARCHAR(32), event VARCHAR(12), count INT UNSIGNED NOT NULL, PRIMARY KEY (day, rule_id, version, variant,
     slot, event), KEY (rule_id, day))`.
   - `message_inbox (id BIGINT UNSIGNED AI PK, user_id FK users ON DELETE CASCADE, rule_id, version, rule_key
     VARCHAR(64) NULL, slot VARCHAR(32) NOT NULL DEFAULT 'alerts_inbox', kind VARCHAR(16), level VARCHAR(16),
     state VARCHAR(24) NOT NULL, open_flag TINYINT(1) AS (IF(state IN ('active','acknowledged','added_to_visit_note'),
     1, NULL)) STORED, evidence_key VARCHAR(96) NULL, hit_count INT UNSIGNED NOT NULL DEFAULT 1, evidence JSON NULL,
     first_at, last_at, acked_at, noted_at, resolved_at, archived_at (TIMESTAMP NULL), legacy_alert_id BIGINT UNSIGNED
     NULL, created_at, updated_at, UNIQUE (user_id, rule_id, open_flag), KEY (user_id, state, level, last_at),
     KEY (state, last_at))` — at most one open row per user and message (M-16); history rows have `open_flag` NULL.
   - `challenge_programs`: `contraindications JSON NULL`, `start_phase VARCHAR(24) NULL`.
   - Data step (M-18): `daily_advice` + `product` → `content`; `modal` + `product` → `marketing`. List every other
     slot/kind mismatch in `tasks/PROGRESS.md` (they are skipped by selection with `kind_not_allowed`, T-M10-04).
   - Add the new empty tables to the contract fixture dump if the contract harness needs them (precedent D-47).
2. Kinds and levels (`internal/messages/rules`): kinds `safety` (1000), `engine` (800–999), `clinical` and `content`
   (100–799, approval required), `product` and `marketing` (1–99). Level × kind (SPEC §5.3): `info` — every kind but
   safety; `suggestion` — engine/content/product and exactly one CTA; `follow_up` — engine/clinical, evidence required;
   `urgent` — safety/clinical, a call CTA (115 or the doctor) required. Slot × kind table as code constants (used by
   T-M10-04): urgent_modal: safety · status_banner: engine · smart_tip: clinical · daily_advice: content ·
   daily_challenge: product · push: any · modal: marketing · alerts_inbox: any but marketing.
3. Validator (`validate.go`): safety → audience 100, one variant, no caps (M-07); product/marketing → at least one day
   or week cap; variant keys unique, weights > 0, at most 6; `topics` must contain `fertile_window` when conditions or
   text use a fertility fact or variable (M-06; the fact list is a constant here, T-M10-03 moves it to the registry);
   a CTA on safety or urgent never targets `/plus`; `goal_event` from the catalogue (M-17: `period_logged`,
   `log_saved`, `checkup_done`, `plus_purchased`, `challenge_joined`); `locked` rules cannot be switched off or deleted
   (409 `locked`).
4. Versioned approval (M-02, M-03): an edit of content, conditions, exceptions, level, kind, topics or variant content
   bumps the version and leaves it awaiting approval while `approved_version` keeps serving; an edit of priority,
   audience, schedule, hour windows or caps bumps the version and keeps the approval (`approved_version` = new version).
   `Served()` returns the `approved_version` snapshot when the head is not approved. Rejecting a draft keeps the
   approved version live. Seeded rows may be inserted already approved.
5. Authority (SPEC §5.8): create, update, toggle, delete and approve of `safety`, and approve of `clinical`/`content`,
   need `CapClinical`; an admin with only `messages` cannot set a rule's kind to safety/clinical/content or edit one;
   every action keeps writing the existing admin audit row.
6. Admin API (`/api/admin/v1/messages/rules*`, Go-only): accept and return the new fields; accept the spec names as
   aliases on input (`type`→`kind`, `slot`→`placement`, `name`→`title`, M-01) and return them next to the canonical
   keys; 422 codes per field.

## Tests
- Validator table over every level × kind × slot combination (accept/reject expected per SPEC §5.3/§5.4).
- Legacy fallback: a row with only the old columns loads with the same caps, variants, schedule and hours as today.
- Integration: approved clinical v1 is live → edit the body → users still get v1, admin sees v2 pending → approve →
  v2 served; a priority-only edit keeps approval; reject keeps v1.
- Integration: `messages`-only admin → 403 on safety create/update/toggle/delete and on changing a product rule to
  clinical; `CapClinical` admin → 200; locked rule toggle/delete → 409.
- Migration up/down; kind remap on fixture rows; `open_flag` uniqueness (second open row for the same user and rule
  fails, a second history row succeeds); `make schema-diff` green.

## Out of scope
- The fact registry (T-M10-03), selection (T-M10-04), the inbox logic (T-M10-07), admin-web (T-M10-12).

## Acceptance
- Verify passes; `messages` contract unchanged (v1 shapes) or allow-listed under D-160; PROGRESS.md cites D-160.

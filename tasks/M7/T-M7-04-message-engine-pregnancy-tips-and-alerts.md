---
id: T-M7-04
title: Message engine — per-week pregnancy tips and admin-configured alert rules (4 levels)
milestone: M7
type: backend
status: todo
depends_on: [T-M7-03]
parallel_group: M7-C
touches: [backend-go/internal/messages, backend-go/internal/pregnancy/alerts, backend-go/internal/pregnancy/v2/alerts, backend-go/internal/http/routes_pregnancy_v2.go, backend-go/api/openapi.yaml, backend-go/messages/content/defaults.json]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/messages/... && make test-int PKG=./internal/messages/... && make contract ROUTES=messages,pregnancy
---

# T-M7-04 — Message engine — per-week pregnancy tips and admin-configured alert rules

## Why
User requirement: pregnancy messages must live in the main message engine and be editable from admin (docs/pregnancy-v2/README.md §
Message engine integration).

## Scope
1. Pregnancy base layer: `pregnancy_week_tip/{week}` (exact week) → milestone → trimester. Served by
   `GET /messages/daily` and reused by `/pregnancy/v2/today`.
2. Symptom overrides read `pregnancy_symptom_logs` (+ extras) in pregnancy mode. **Contract:** if this changes the
   Laravel-golden output of `/messages/daily` for pregnancy fixtures, record it in `deviations.md` and get the
   user's OK before merging (do not silently break parity).
3. `internal/messages/pregnancyalerts`: rule registry — vomiting_streak, severe_symptom_count, critical_symptom
   (existing), weight_missing_week, week_entered, bp_high, sugar_high, fetal_movement (existing); each rule reads
   enabled/level/params/texts from `message_contents` group `pregnancy_alert` item_key = rule key; emits
   `{level: info|suggestion|follow_up|urgent, what_we_saw, how_sure, advice, actions[]}` persisted to
   `pregnancy_alerts` (dedupe per rule per window). Evaluated after every v1/v2 log save and once daily.
4. `GET /pregnancy/v2/alerts` (7-day window + level legend texts) and
   `POST /pregnancy/v2/alerts/{id}/actions/{ack|add_to_visit_note}` (appends to today's `visit_note`).
5. Tests: each rule fires/doesn't at thresholds; disabling a rule in `message_contents` stops it; texts change
   without a deploy.

## Acceptance
- Rules and texts fully driven by DB rows; v1 alert endpoints unchanged; tests + contract green (or deviation approved).

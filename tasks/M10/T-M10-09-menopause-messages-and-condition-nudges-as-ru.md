---
id: T-M10-09
title: Menopause messages and condition nudges as rules
milestone: M10
type: backend
status: done
depends_on: [T-M10-07]
parallel_group: M10-B
touches: [backend-go/internal/messages/menomessages, backend-go/internal/messages/conditionnudges, backend-go/internal/messages/facts/menopause.go, backend-go/internal/messages/facts/conditions.go, backend-go/db/migrations, backend-go/api/openapi.yaml, docs/engine-v6/clinical-signoff.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... && make test-int PKG="./internal/messages/... ./db/migrations/..." && make contract ROUTES=menopause,conditions
---

# T-M10-09 — Menopause messages and condition nudges as rules
**Model:** opus (health messages).

## Why
Menopause messages (D-43) and condition nudges (D-41) are two more stand-alone engines computed per request with no
inbox, no caps and no admin control (gap 03 §2.6). SPEC §5.4.1 wants one engine for every mode's alerts.
D-166.

## Scope
1. Investigate what each engine detects today (`internal/messages/menomessages`, `internal/messages/conditionnudges`)
   and turn every signal into a fact with evidence: `meno.*` (e.g. bleeding after 12 months without a period, score
   change, hot-flash count over 7 days) and `cond.*` (e.g. heavy-pain days, heavy-bleeding days, PMDD pattern).
2. Data migration seeding one rule per current message: warnings to `alerts_inbox` (kind/level = today's behaviour;
   postmenopausal bleeding keeps today's level and is flagged for the advisor in clinical-signoff.md), tips to
   `smart_tip` (clinical) or `daily_advice` (content). `rule_key = meno.<name>` / `cond.<name>`.
3. The existing GET endpoints become adapters over the inbox and the feed with the same JSON shape.

## Tests
- Parity table: old engine output == new rule matches over the existing fixtures (written before the old logic is
  removed).
- Existing integration tests green through the adapters; contract `menopause` and `conditions` unchanged.
- A repeated menopause bleeding signal merges into one inbox row (count 2).

## Out of scope
- New menopause or condition content; UI.

## Acceptance
- Verify passes; PROGRESS.md cites D-166.

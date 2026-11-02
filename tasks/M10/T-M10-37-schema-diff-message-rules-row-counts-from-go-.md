---
id: T-M10-37
title: schema-diff: message_rules row counts from Go-only data seeds (00112–00115)
milestone: M10
type: backend
status: todo
depends_on: []
parallel_group: M10-B
touches: [backend-go/Makefile,backend-go/scripts,docs/go-migration/migrations.md]
skills: []
verify: cd backend-go && make schema-diff
---

# T-M10-37 — schema-diff: message_rules row counts from Go-only data seeds (00112–00115)

## Why
`make schema-diff` is red: `message_rules 0` (Laravel) vs `29` (goose). goose 00112–00115 seed rules as data-only migrations without Laravel twins (T-M10-06/08/08c/09), on purpose: the v6 message engine is Go-only. Found while verifying T-M10-19b.

## Scope
- Decide and implement one: exclude Go-only seeded tables (message_rules) from the schema-diff row-count check with a documented list, or add Laravel twins. Prefer the exclusion (Laravel never reads message_rules).
- Document the rule in docs/go-migration/migrations.md.

## Out of scope
- Changing the seeded rules.

## Acceptance
- `make schema-diff` green; structural diffs still caught. 

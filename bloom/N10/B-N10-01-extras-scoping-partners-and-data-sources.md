---
id: B-N10-01
title: Extras scoping — partners and data sources
milestone: N10
type: investigate
status: todo
depends_on: [B-N9-13]
parallel_group: N10-A
touches: [docs/night-bloom/extras.md]
skills: []
verify: test -s docs/night-bloom/extras.md
---

# B-N10-01 — Extras scoping — partners and data sources

## Why
Shop, insurance, city services, home lab tests and wearables depend on partners that don't exist yet.

## Scope
- For each extra: what the design shows, minimum viable version, partner/adapter needed, privacy rule (shop separate from health data; never target by health logs), and questions for the user. Update N10 task scopes.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Doc reviewed with the user before N10-02+ start
- `verify` green

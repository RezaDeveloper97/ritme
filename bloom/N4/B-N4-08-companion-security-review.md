---
id: B-N4-08
title: Companion security review
milestone: N4
type: investigate
status: done
depends_on: [B-N4-04,B-N4-05,B-N4-06]
parallel_group: N4-H
touches: [docs/security/bloom-companion.md]
skills: [security-review]
verify: test -s docs/security/bloom-companion.md
---

# B-N4-08 — Companion security review

## Why
Sharing health data with another account is the riskiest feature.

## Scope
- Run the security-auditor agent over N4; fix or file every finding; document threat model.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Doc with findings + resolutions
- `verify` green

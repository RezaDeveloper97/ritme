---
id: T-M10-29b
title: Contract allow-lists for the D-178 Plus changes across all route groups
milestone: M10
type: backend
status: done
depends_on: [T-M10-29]
parallel_group: M10-D
touches: [backend-go/contract/allowlist, backend-go/internal/labs/labs_int_test.go]
skills: []
verify: cd backend-go && make contract ROUTES=all
---

# T-M10-29b — Contract allow-lists for the D-178 Plus changes across all route groups

## Why
T-M10-29's verify ran only `make contract ROUTES=plus`. `ROUTES=all` shows 41 failures (healthrecord-extract 12, labs 10, voicelog 16, teen 2, sharelinks 1) plus the analysis groups (already allow-listed): the 8th entitlement row in trial-start responses, and free-user lab/AI calls now passing the Plus gate (free 1/month) and hitting consent_required 403 instead of plus_required 402.

## Scope
- For each failing case decide: intended D-178 change → narrow allow-list entry citing D-178 (prefer exact paths/cases, not `body`), or a real regression → stop and report.
- No product code changes.

## Out of scope
- Re-recording Laravel goldens.

## Acceptance
- `make contract ROUTES=all`: 0 failed.
- Every new entry cites D-178 with a reason; regressions (if any) reported, not hidden. 

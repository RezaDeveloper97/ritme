---
id: T-M10-01
title: Message, health and analysis decisions, advisor sheet and deviation rows (user OK)
milestone: M10
type: investigate
status: done
depends_on: [T-M9-01]
parallel_group: M10-A
touches: [docs/engine-v6/DECISIONS.md, docs/engine-v6/clinical-signoff.md, docs/go-migration/deviations.md]
skills: []
verify: grep -q '^| D-183 ' docs/go-migration/deviations.md && test -f docs/engine-v6/clinical-signoff.md && ! grep -q 'Status\*\* proposed until T-M10-01' docs/engine-v6/DECISIONS.md
---

# T-M10-01 — Message, health and analysis decisions, advisor sheet and deviation rows (user OK)
**Model:** main session (opus). These are product and clinical decisions; do not delegate.

## Why
M10 rebuilds the message engine, the health-module thresholds and the analysis/KPI/admin layer on SPEC v6 §3–§5.
`docs/engine-v6/DECISIONS.md` sections M (19 rows), H (18 rows) and A (12 rows) hold the open questions with a
recommended default each. Several health rows reduce an alert (EPDS ≥ 13, BP ≥ 140/90 level, random glucose, borderline
labs): those need the medical advisor, and until then the safe default ships (DECISIONS.md H table, column "Safe
default shipped now"). Evidence: `docs/engine-v6/gap/03-message-engine.md` §3, `04-health-modules.md` §4,
`06-analysis-kpi-admin.md` §3.

## Scope
1. Re-read M, H and A and the cited gap sections. Spot-check that nothing in the code changed since 2026-10-21 that
   makes a default wrong (at least `internal/messages/rules/rules.go`, `internal/labs/labs.go:89`,
   `internal/checkups/engine/engine.go`, `internal/metrics/metrics.go`).
2. Ask the user in Persian, one line per decision with the recommended default, in three groups:
   - **Message engine:** M-01 … M-19.
   - **Analysis, KPI, admin:** A-01 … A-12 (A-11 changes what free users see in detail views; say so plainly).
   - **Health items the user owns:** H-03, H-05, H-08, H-12, H-13, H-18.
   Offer "OK to all defaults" and accept per-item changes. Wait for the answer.
3. Advisor items (H-01, H-02, H-04, H-06, H-07, H-09, H-10, H-11, H-14, H-15, H-16, H-17): write
   `docs/engine-v6/clinical-signoff.md` — one row per item: question in Persian, today's value, spec value, source
   (ACOG/USPSTF/IOM/WHO/CDC/ADA citation from SPEC §6.2), the registry key and value that switches it, a blank
   "advisor decision" cell. Add the new registry keys of T-M10-16 that change nothing today but will need a signature
   when changed. Ask the user whether to append these rows to the existing review doc «بازبینی پزشکی محتوای ریتمی»
   (link in `tasks/PROGRESS.md`, T-M4-10 entry) or send the sheet separately. Do not wait for the advisor: the safe
   defaults ship; record each row as `advisor-pending`.
4. Record every answer in DECISIONS.md (`accepted` or `changed — <what>`, date); replace the line "All M, H and A
   rows: Status proposed until T-M10-01 records the answer" with the approval date.
5. Write rows **D-160 … D-183** (table "M10 deviation ids" in DECISIONS.md) in `docs/go-migration/deviations.md`, same
   columns as existing rows, status `accepted — engine v6 M10, not shipped yet`. D-184…D-189 stay reserved (no rows).
6. If an answer changes a task's scope, edit that task file now (Scope/Tests/Acceptance) and list the edits in
   `tasks/PROGRESS.md`.

## Out of scope
- Code. Family decisions (T-M11-01). Getting the advisor's signature (tracked in clinical-signoff.md).

## Acceptance
- No M/H/A row is open; advisor rows are `advisor-pending` or signed; clinical-signoff.md exists.
- D-160…D-183 exist in deviations.md; edited tasks listed in PROGRESS.md.

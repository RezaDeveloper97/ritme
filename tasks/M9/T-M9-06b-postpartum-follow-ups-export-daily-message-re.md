---
id: T-M9-06b
title: Postpartum follow-ups: export, daily message red flags, goldens, translation seed, advisor doc
milestone: M9
type: fullstack
status: done
depends_on: [T-M9-06]
parallel_group: M9-A
touches: [backend-go/internal/messages/manager,backend-go/internal/export,backend-go/internal/account,backend-go/resources/translations,backend-go/internal/i18n/testdata,backend-go/contract/golden/postpartum,backend-go/contract/golden/public,backend-go/api/openapi.yaml, backend-go/contract/fixtures/dump.sql, backend-go/contract/golden, backend-go/contract/allowlist, backend-go/resources/translations, docs/go-migration/deviations.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./... && make contract ROUTES=all
---

# T-M9-06b — Postpartum follow-ups: export, daily message red flags, goldens, translation seed, advisor doc

## Why
T-M9-06 left these outside its touches (see its PROGRESS.md section).

## Scope
1. **Privacy:** add the new `postpartum_recovery_logs` table to the user data-export path (wherever the other health logs are exported; find it first and adjust touches if it lives elsewhere). Deletion already cascades.
2. Daily message: the `messages/manager` postpartum override reads only heavy bleeding and clots. Make it also react to the new urgent red flags (fever ≥ `postpartum.redflags.fever_c`, pad per hour, severe pain, breathless).
3. Re-record the postpartum contract goldens from Go, then mark the new OpenAPI fields `required` (`visits`, the red-flag keys, `thresholds.urgent_total`).
4. Resync `backend-go/resources/translations/{fa,en}/postpartum.json` (and `pregnancy-tools`/others if they drifted) from `frontend/messages/`. Then re-record the `public` contract goldens and `internal/i18n/testdata/messages_*.json`.
5. Add H-20 and H-21 (already in `docs/engine-v6/clinical-signoff.md`) to the shared Claude Docs review doc «بازبینی پزشکی محتوای ریتمی» (main session; doc link in `tasks/PROGRESS.md`, T-M4-10 entry).

6. **Full contract green again.** `make contract ROUTES=all` shows 44 failures, and they fail the same way on HEAD 8b41179b: postpartum 17, pregnancy 9, analysis-ttc 7, analysis 6, pregnancy-tools 3, logs 1, teen 1.
   - Add migrations 00096 (user_profiles unique) and 00097 (`postpartum_recovery_logs`) to `contract/fixtures/dump.sql`, so `postpartum/activate` no longer returns 500.
   - For each remaining failure, decide:
     - an intended Go-side change (for example the M8 Persian copy «سیکل» → «چرخه», or the new T-M9-05/06 fields): add or extend the allow-list entry that cites its existing deviation, or update the golden when the change is a Go-only route;
     - a real regression: fix the code.
   - Never hide a regression in the allow-list. List every decision in the report.

## Acceptance
- An export test includes the recovery logs.
- A daily-message test covers fever and breathlessness.
- `make contract ROUTES=postpartum` is green with the fields required.
- `TestBundle_MatchesLaravelGoldens` and `TestSeedMatchesFrontendMessages` are green; `go test ./...` is fully green.
- `make contract ROUTES=all` → 0 failed.

## Split
- Scope 1 (export) moved to **T-M9-06c**. The export lives in `internal/accountexport`, outside these touches, and needs a user decision: D-96's nine export keys, and whether EPDS answers are exported (D-54).
- Scope 5 (the advisor doc): the main session added H-20/H-21 to the Claude Doc.

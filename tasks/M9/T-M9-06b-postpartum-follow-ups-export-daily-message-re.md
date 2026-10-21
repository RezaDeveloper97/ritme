---
id: T-M9-06b
title: Postpartum follow-ups: export, daily message red flags, goldens, translation seed, advisor doc
milestone: M9
type: fullstack
status: todo
depends_on: [T-M9-06]
parallel_group: M9-A
touches: [backend-go/internal/messages/manager,backend-go/internal/export,backend-go/internal/account,backend-go/resources/translations,backend-go/internal/i18n/testdata,backend-go/contract/golden/postpartum,backend-go/contract/golden/public,backend-go/api/openapi.yaml]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... ./internal/i18n/... ./internal/http/... && make contract ROUTES=postpartum
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

## Acceptance
- An export test includes the recovery logs.
- A daily-message test covers fever and breathlessness.
- `make contract ROUTES=postpartum` is green with the fields required.
- `TestBundle_MatchesLaravelGoldens` is green.

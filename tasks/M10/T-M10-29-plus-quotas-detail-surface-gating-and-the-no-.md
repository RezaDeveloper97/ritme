---
id: T-M10-29
title: Plus quotas, detail-surface gating and the no-paywall guarantee
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-26, T-M10-07, T-M10-08, T-M10-19]
parallel_group: M10-D
touches: [backend-go/internal/analysis/analysis_int_test.go, backend-go/internal/analysis/v6_int_test.go, backend-go/internal/i18n/testdata, backend-go/internal/assistant, backend-go/internal/plus/entitlements.go, backend-go/internal/plus/settings.go, backend-go/internal/plus/service.go, backend-go/internal/plus/gate.go, backend-go/internal/analysis/handlers.go, backend-go/internal/assistant/handlers.go, backend-go/contract/allowlist/plus.yaml, frontend/src/app/message-scopes.ts, backend-go/internal/analysis/range.go, backend-go/internal/assistant/service.go, backend-go/internal/labs/extract.go, backend-go/internal/fertility, backend-go/internal/http/paywall_guard_int_test.go, backend-go/api/openapi.yaml, frontend/src/entities/plus, frontend/src/screens/analysis, frontend/src/screens/phase-details, frontend/src/screens/fertility-bbt, frontend/src/screens/lab-upload, frontend/src/screens/assistant-chat, frontend/messages/fa/plus.json, frontend/messages/en/plus.json, backend-go/resources/translations/fa/plus.json, backend-go/resources/translations/en/plus.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/plus/... ./internal/analysis/... ./internal/assistant/... ./internal/labs/... && make test-int PKG="./internal/plus/... ./internal/http/..." && make contract ROUTES=plus && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run copy:check -- plus
---

# T-M10-29 — Plus quotas, detail-surface gating and the no-paywall guarantee
**Model:** opus for the gates and the guarantee test; sonnet for the paywall UI.

## Why
SPEC §4.3: Lab AI 1/month free and 10 Plus (today free = locked), assistant "unlimited with a fair cap" for Plus (no
cap today), the 1-year analysis range is Plus (free today, `analysis/range.go:23`), and "never a paywall on emergency
paths, medication reminders or EPDS" — true today by inspection but not tested (gap 06 §2.3). §4.3 also lists
"prediction source and confidence, BBT with confirmation, LH-based prediction" as Plus (A-11). DECISIONS.md A-03,
A-04, A-11; D-178.

## Scope
1. `plus/entitlements.go`: LabAI free `{Enabled, Limit 1 per month}` (A-03), admin-tunable through plus settings;
   assistant Plus fair-use cap 300 per month (A-04) from settings, at the cap a polite in-chat message (no error);
   new key `analysis_long_range` used by `analysis/range.go`.
2. A-11 detail surfaces: the engine output stays complete for everyone; Plus gates only (a) the "why this date"
   breakdown (prediction source, basis cycles, confidence reasons) on the prediction card / phase details, (b) the BBT
   chart's coverline and confirmation view, (c) the LH-based prediction explanation. Separate detail endpoints
   check `plus.Require`; fields inside shared payloads are gated in the UI only.
3. Guarantee test (`internal/http/paywall_guard_int_test.go`): a free user calls every emergency, medication-reminder,
   EPDS, postpartum-recovery, pregnancy alert/kick/contraction, vitals-alert and alerts-inbox route with valid
   payloads → never `plus_required`, and a spy shows `plus.Consume/Require` is never called. A static test asserts
   `internal/emergency`, `internal/reminders`, `internal/care`, `internal/postpartum`, `internal/safety` and
   `internal/messages/inbox` never import `internal/plus`.
4. Frontend: paywall card for the 1-year range, Lab AI quota copy (1 per month), the assistant cap message, Plus
   locks on the three detail surfaces; fa/en.

## Tests
- Resolve table per plan; `Consume` at the limit → `plus_quota_exceeded`; LabAI free second use in a month → quota;
  assistant message 301 for Plus → cap message; free user 1-year range → `plus_required`.
- The guarantee test and the import test.
- Frontend vitest for the gates.

## Out of scope
- Prices and plans.

## Acceptance
- Verify passes; PROGRESS.md cites D-178.

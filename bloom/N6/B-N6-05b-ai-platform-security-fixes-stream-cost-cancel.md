---
id: B-N6-05b
title: AI platform security fixes (stream cost, cancellation, thinking tokens, per-user cap, reserve-first, PII, consent)
milestone: N6
type: backend
status: done
depends_on: [B-N6-05]
parallel_group: N6-E2
touches: [backend-go/internal/ai,backend-go/internal/consent,backend-go/internal/profile,backend-go/internal/voicelog,backend-go/internal/http,backend-go/internal/platform/config,backend-go/api]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N6-05b — AI platform security fixes (stream cost, cancellation, thinking tokens, per-user cap, reserve-first, PII, consent)

## Why
Security review of B-N6-05 (59b3acb): H1/H2 must be fixed before any `Client.Chat` feature ships (B-N7-06), M1 before enabling Gemini for real users.

## Scope
- **H1:** streams interrupted mid-way record zero cost → keep the last cumulative `usageMetadata`, record a conservative estimate when usage is missing (never 0), reserve the Plus quota before streaming (refund only on upstream failure before the first delta).
- **H2:** Fiber's `c.Context()` is never cancelled → `Client.Chat` owns a `context.WithTimeout` (e.g. 120 s) + returns `cancel`/`Close()`; `send`/`emit` select on it; document `defer cancel()` and cancel on write error; use Transport idle/header timeouts instead of a 30 s total timeout for streams.
- **M1:** count `thoughtsTokenCount` (+ tool-use prompt tokens) as output; `thinkingConfig.thinkingBudget: 0` and `maxOutputTokens` for parse/extract/transcribe; recommend a pinned model id in prod.
- **M2:** per-user daily cost cap (`AI_USER_DAILY_COST_CAP_USD`, index exists) checked in `access.Guard`; per-feature throttle enforced in `Guard.Chain`.
- **M3:** reserve-first quota pattern (or per-user concurrency semaphore) in `internal/ai/access`; apply to voice logging; silent/empty transcripts still count against cost.
- **M4:** PII filter: normalise digits and Arabic/Persian letters, separator-tolerant 10+ digit runs (cards, national ids, phones, postal codes), Latin↔Persian name matching; reject `patient`/`name` keys in extraction schemas.
- **M5:** `/profile/consents` reports `granted` only for the current version, rejects AI-consent grants without an explicit version, export includes versions.
- **L2** throttle right after auth on voice; **L3** central magic-byte sniffing for images/PDF and limits aligned with the 25 MB body limit; **L4** document server-side chat history only; **L5** null `user_id` in `ai_usage_logs` after 90 days + include in export.

## Out of scope
- 

## Acceptance
- Each finding fixed with tests (stream cancel/cost, caps, PII cases, consent version)
- `verify` green

---
id: B-N6-05
title: AI adapter platform — providers, consent, usage and cost
milestone: N6
type: backend
status: done
depends_on: [B-N5-11]
parallel_group: N6-E
touches: [backend-go/internal/ai,backend-go/internal/consent,backend-go/db,backend-go/config,.env.stage.example]
skills: [security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N6-05 — AI adapter platform — providers, consent, usage and cost

## Why
Voice log, lab analysis and the assistant all need one AI layer.

## Scope
- Interface for text chat (streaming), vision/document extraction and STT; `fake` provider with fixtures (default); one real provider behind config (e.g. Gemini via server-side key from env — never in repo).
- Consent records (versioned consent text, per feature), per-call usage + cost log, quota checks via entitlements, PII minimisation (no name/phone sent).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Fake provider used in all tests
- No key in repo or logs
- `verify` green

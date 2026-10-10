---
id: B-N7-06b
title: Assistant security fixes — encrypt chats at rest, SSE in-flight cap, summary throttles, emergency normalisation, photo deletion
milestone: N7
type: backend
status: todo
depends_on: [B-N7-06]
parallel_group: N7-F2
touches: [backend-go/internal/assistant,backend-go/internal/ai,backend-go/db,backend-go/api]
skills: [security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N7-06b — Assistant security fixes — encrypt chats at rest, SSE in-flight cap, summary throttles, emergency normalisation, photo deletion

## Why

## Scope
Security/privacy audit of B-N7-06 (internal/assistant):
- H1 seal `assistant_messages.body`, `assistant_conversations.title/summary`, `assistant_handoffs.summary` with the
  telemed chat AEAD helper (PRIVATE_NOTE_KEY, AAD = table + row id + user id); decrypt only on read.
- M1 SSE: in-flight release in the stream writer's defer (not at handler return) + hard stream timeout (90 s).
- M2 policy throttles on `/summary`; regenerate only after new paid turns (or charge free users); separate emergency throttle.
- M3 emergency normalisation: fold curly apostrophes, drop apostrophes in compact form, strip Cf/Mn/U+0640, NFKC,
  `unicode.IsSpace`; more floor terms; model `[[triage:emergency]]` → store + emit the 115 card, set `emergency`.
- M4 delete-all removes every `assistant_photo` file of the user; sweep unattached photos > 24 h; forbid photo reuse or
  check use count; delete inside a tx.
- M5 strip URLs from the live stream (buffer partial tokens).
- L1 safety-precedence line after department text; L4 per-request deadline.

## Out of scope
- 

## Acceptance
- 

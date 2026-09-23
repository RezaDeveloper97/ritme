---
id: T-M2-17
title: MessageSystem — daily messages and mode endpoints
milestone: M2
type: backend
status: done
depends_on: [T-M2-14, T-M2-16]
parallel_group: M2-G
touches: [backend-go/internal/messages, backend-go/internal/http/routes_messages.go, backend-go/db/queries/messages, backend-go/contract/allowlist/messages.yaml]
skills: []
verify: cd backend-go && go test ./internal/messages/... && make contract ROUTES=messages
---

# T-M2-17 — MessageSystem

## Why
`/messages/daily` layers engine output, symptom overrides, correlations, patterns and supplements, with content from
`message_contents` and ~2k lines of in-code fallback copy. Read domain-inventory §4.3 and deviation D-05 (dead-field
behaviour is **preserved**).

## Scope
1. `messages/content` = `MessageContentRepository` (all live rows for a locale loaded once per request;
   `resolve(group,item_key,locale,fallback)` with `locale → fa` fallback). Expose it for `BmiService` (T-M2-11 may
   have a minimal reader — replace it with this one if so, and note it).
2. Embedded defaults: export every class's `contentDefaults()` once to JSON via a PHP script (committed with its
   output) and embed with `go:embed`; the Go `MessageContentSeeder` equivalent is not needed until T-M2-27.
3. `messages/manager` + engines (cycle base non-TTC/TTC, override, pregnancy base/trimester/week/override with the
   ±2-week milestone and 0-based-week quirk), CorrelationLayer (premium filtering), PatternLayer (premium only),
   Nutrition/Sleep/Exercise modules; context from the legacy engine + 90 days of logs. Reproduce the dead-field reads
   exactly (a test documents which signals can fire).
4. `GET /messages/daily` (query 422 controller family, 400 incomplete profile), `GET /messages/mode`.

## Out of scope
Changing message logic (future product task after M2).

## Acceptance
- `make contract ROUTES=messages` green for all personas (cycle, TTC, premium, pregnant) × fa/en/ar × several dates.
- A test asserts that the embedded defaults equal the PHP export byte for byte.

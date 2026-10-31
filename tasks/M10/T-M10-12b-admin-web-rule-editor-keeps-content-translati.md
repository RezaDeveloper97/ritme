---
id: T-M10-12b
title: Admin-web rule editor keeps content.translations (en overlay) on save
milestone: M10
type: frontend
status: todo
depends_on: [T-M10-12,T-M10-06]
parallel_group: M10-W
touches: [admin-web/src/screens/message-rules,admin-web/messages/fa.json,admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M10-12b — Admin-web rule editor keeps content.translations (en overlay) on save
**Model:** sonnet.

## Why
T-M10-06 seeds the status-banner rules (goose 00112) with fa copy as the base content and en copy under
`content.translations.en`, which the feed reads. The T-M10-12 editor does not know `content.translations`, so saving
one of these rules (or any rule with an overlay) drops the en copy.

## Scope
1. `lib/rule-form.ts`: round-trip `content.translations` untouched when the admin does not edit it (payload keeps it).
2. Show the overlay per variant as an optional "English copy" block (title, body, CTA label/target), editable; empty
   fields fall back to the base copy. fa/en strings.

## Tests
- Vitest: round trip of a seeded banner rule keeps `translations.en` byte-identical; editing en updates only that
  variant's overlay.

## Acceptance
- Verify passes; no rule save loses `content.translations`.

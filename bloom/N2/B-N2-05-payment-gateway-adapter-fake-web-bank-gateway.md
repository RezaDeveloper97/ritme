---
id: B-N2-05
title: Payment gateway adapter (fake + web bank gateway)
milestone: N2
type: backend
status: done
depends_on: [B-N2-04]
parallel_group: N2-E
touches: [backend-go/internal/payments,backend-go/internal/plus,backend-go/config,backend-go/api,.env.stage.example]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N2-05 — Payment gateway adapter (fake + web bank gateway)

## Why
External services go behind adapters with a test mode (user decision).

## Scope
- `Gateway` interface: create payment → redirect URL, verify callback, refund. `fake` provider (local success/fail page) is the default; one real Iranian web gateway provider behind config, disabled until keys exist.
- Idempotent verify, amount/currency checks, signature checks, payment log. Café Bazaar / Myket in-app billing is Android-only → out of scope (documented).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Replay/double-verify and amount-tamper tests
- No secrets in repo
- `verify` green

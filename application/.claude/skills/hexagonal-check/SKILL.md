---
name: hexagonal-check
description: Verify code respects the hexagonal (Ports & Adapters) architecture of this Android app (Ritme) — correct layer placement and inward-only dependencies. Use when adding/moving classes, before committing, or when reviewing where code belongs (domain vs application vs adapter).
---

# Hexagonal Architecture Check

Enforce the layering from CLAUDE.md §2. Dependencies point **inward only**:
`adapter → application → domain`. Never the reverse.

## Layer placement rules

- `domain/model/` — immutable entities & value objects (`val` only). Pure Kotlin.
- `domain/port/inbound/` — use case interfaces (what the app can DO).
- `domain/port/outbound/` — repository/gateway interfaces (what the app NEEDS:
  HTTP, cache, secure storage, clock, logger).
- `domain/usecase/` — use case implementations, orchestration logic ONLY.
- `application/service/` — thin layer implementing inbound ports, calling outbound ports.
- `adapter/inbound/ui/` — Compose screens, ViewModels, navigation.
- `adapter/outbound/network|persistence/` — real implementations of outbound ports.
- `adapter/outbound/di/` — manual composition root (the only place that instantiates adapters).
- `platform/` — Application class, manifest-bound glue.

## Checklist (run on changed files)

- [ ] `domain/` imports nothing but Kotlin stdlib + `kotlinx.coroutines` core.
      NO `android.*`, NO `androidx.*`, NO `org.json`, NO IO classes.
- [ ] Every outbound capability (HTTP, disk, storage, clock, logger) is an
      interface in `domain/port/outbound`; the implementation lives in `adapter/outbound/*`.
- [ ] Use cases receive ports via constructor injection — never `new` an adapter inside a use case.
- [ ] ViewModels depend on inbound port interfaces (use cases), never on
      network/DB/adapter classes directly.
- [ ] No application/adapter type leaks into a domain signature.
- [ ] Ports are small & role-specific (`QuoteRepository`, `PaymentGateway`),
      not one giant `BackendGateway`.

## How to scan
Grep changed domain files for forbidden imports:
`android.`, `androidx.`, `org.json`, `java.net`, `java.sql`, `android.database`.
Any hit in `domain/` is a violation → move the I/O behind an outbound port.

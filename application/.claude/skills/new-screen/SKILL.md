---
name: new-screen
description: Scaffold a new Compose screen end-to-end following this project's hexagonal layering, state handling, and crash-resilience rules. Use when adding any new screen/flow (plan list, quote, checkout, payment status, profile).
---

# Add a New Screen

Follow the layers top-to-bottom. Wire everything in the composition root, not inline.

## Steps

1. **Domain** — if new data is needed, add immutable model(s) in `domain/model/`
   (`val` only). Define/extend the inbound use case interface in
   `domain/port/inbound/` and outbound port(s) in `domain/port/outbound/`.
2. **Use case** — implement in `domain/usecase/`, orchestration only, ports injected.
   Add short KDoc explaining intent ("why", not "what").
3. **Adapter (outbound)** — implement the outbound port in `adapter/outbound/network`
   or `persistence`. Return sealed `Result<T, AppError>` — never throw across the port.
4. **ViewModel** (`adapter/inbound/ui/...`) — depends on the inbound use case interface
   only. Expose a sealed `UiState` via `StateFlow`. Keep under ~150 lines; if it grows,
   extract a use case. No parsing/networking/SQL here.
5. **Composable screen** — small stable composables, immutable params. Lists are
   `LazyColumn`/`LazyRow` + stable `key`. Heavy/hot drawing via `Canvas` if needed.
   Wrap in `RitmeTheme`; use brand tokens (no inline hex), RTL layout, Persian font +
   digits (see `ritme-brand` skill, §5c).
6. **Last-safe-screen** — in a per-screen `LaunchedEffect`, write
   `{ route, minimalArgs, timestamp }` via `SafeStateRepository` on successful render
   (debounced, not per recomposition). minimalArgs = small (e.g. plan ID).
7. **Breadcrumb** — record "screen entered" into the in-memory ring buffer.
8. **Navigation** — register the route; ensure recovery flow can rebuild it from minimalArgs.
   Wrap the screen in the shared `SwipeBackContainer` so Telegram-style edge
   swipe-to-go-back works (every non-root screen — see `swipe-back` skill, §5b).
9. **DI** — wire all new instances in `adapter/outbound/di` composition root.
10. **Tests** — use-case JUnit test with hand-written fake ports; Compose UI test for
    critical flows (selection, checkout, payment confirmation).

## Done when
All boxes in the `pre-finish-checklist` skill pass.

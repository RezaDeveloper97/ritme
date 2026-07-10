---
name: pre-finish-checklist
description: The mandatory self-verification checklist from CLAUDE.md §11. Run this before finishing ANY task / commit in this project to confirm all architecture, dependency, performance, and crash-resilience rules are met.
---

# Pre-Finish Checklist (CLAUDE.md §11)

Self-verify every item before declaring a task done. Any unchecked box = not done.

- [ ] No new third-party dependency added without explicit user approval. (`no-deps-check`)
- [ ] Domain layer has zero Android/IO imports. (`hexagonal-check`)
- [ ] All outbound I/O accessed through a port interface; real impl in an adapter.
- [ ] No ViewModel doing parsing/networking/SQL directly.
- [ ] Lists are `Lazy*` + stable keys; no unbounded `Column`+`forEach`. (`perf-playbook`)
- [ ] No C++ introduced without a profiled, documented bottleneck.
- [ ] New error paths return sealed `Result`, never throw across a port boundary.
- [ ] Every new screen registers a last-safe-screen write on successful render. (`crash-resilience`)
- [ ] Every non-root screen supports Telegram-style edge swipe-to-go-back via the shared
      `SwipeBackContainer`; system Back has parity. (`swipe-back`, §5b)
- [ ] Any new fatal-prone path is covered by the Crash Guard, not scattered UI try/catch.
- [ ] UI uses Ritme brand tokens via `RitmeTheme` (no inline hex), RTL-first, Persian
      font + digits. (`ritme-brand`, §5c)
- [ ] Naming & file structure match §4 and §8 (one file per class, no `Utils.kt`,
      use cases = verbs, repositories = nouns, ViewModel ≤ ~150 lines).
- [ ] Commit message uses conventional commits (`feat:`/`fix:`/`refactor:`/`test:`/`chore:`)
      and respects layer boundaries.

If any item fails, fix it before finishing — do not report the task complete.

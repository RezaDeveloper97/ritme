---
name: perf-playbook
description: Apply the "feels like Telegram" performance rules — custom drawing, stable Composables, lazy lists, off-main-thread work. Use when building UI, lists, animations, or the network layer, or when a screen feels janky.
---

# Performance Playbook

CLAUDE.md §5. Perceived speed comes from custom-drawn UI avoiding widget/layout overhead.
Priority order for the app: speed > small APK > polish > maintainability.

## UI

- [ ] Hot, frequently-recomposing UI (animated lists, custom progress, premium transitions)
      uses `Canvas`/`drawWithContent` instead of deep nested standard components.
- [ ] Composables small & stable; pass **immutable data classes** so Compose skips recomposition.
- [ ] Avoid lambdas capturing changing state unnecessarily.
- [ ] Every list is `LazyColumn`/`LazyRow` with a stable `key` — never `Column`+`forEach`
      for anything that can grow.
- [ ] Animations via `animate*AsState` / `Animatable` / vector drawables (no Lottie).
- [ ] Telegram-style edge swipe-to-go-back tracks the finger 1:1 without recomposing a
      heavy tree each frame — drive offset via `Modifier.offset`/custom drawing (`swipe-back`).

## Threading

- [ ] JSON parsing, bitmap decoding, SQLite access all on `Dispatchers.IO`.
- [ ] Surfaced to UI as `StateFlow`. No heavy work on the main thread.

## Network

- [ ] Connection reuse (keep-alive), gzip request/response, sane timeouts.
- [ ] Simple in-memory + disk response cache keyed by endpoint+params for GET-heavy
      screens (product catalog, plan details).

## Build & startup

- [ ] R8 full mode + resource shrinking in `release`.
- [ ] WebP for raster images, vector drawables for icons.
- [ ] `Application.onCreate()` minimal; lazy-init anything not needed for the first frame.

## C++/NDK
Do NOT introduce C++ preemptively. Only after a **profiled, documented** bottleneck in
crypto / heavy bulk numeric / codec work (§6). For this CRUD/checkout app, expected: none.

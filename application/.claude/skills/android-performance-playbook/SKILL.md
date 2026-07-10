---
name: android-performance-playbook
description: Use this skill whenever working on UI rendering performance, list/scrolling performance, app startup time, APK size, or any decision about whether to use native C++/NDK code in this Android app (Ritme). Triggers on phrases like "make this faster," "feels laggy," "reduce APK size," "should this be C++," "optimize this screen," or "why is this slow." Provides the Telegram-speed philosophy (custom-drawn UI over heavy widget stacking) translated to Jetpack Compose, plus strict criteria for when native code is actually justified versus premature optimization.
---

# Performance Playbook (Telegram-Speed, in Compose)

This skill captures what actually makes an app feel as fast and fluid as Telegram —
which is mostly **not** about native code — and applies it to this Compose-based,
API-heavy health-tracking app.

## When to apply this

- A screen/list feels janky or slow.
- The user asks about APK size or cold-start time.
- The user asks whether to use C++/NDK for something.
- Reviewing a new screen for performance before it ships.

## The core insight

Telegram's perceived speed comes mostly from **custom-drawn UI avoiding standard
widget/layout overhead**, not from being written in C++. Its core engine (TDLib) is C++
for networking/crypto/state shared across platforms, but the Android UI itself is
mostly about avoiding heavy nested standard views in favor of direct `Canvas` drawing.
Apply the same philosophy in Compose terms — see below.

## Compose-level techniques (apply by default)

- Use `Canvas`/`drawWithContent` custom drawing for hot, frequently-recomposing UI
  (animated lists, custom progress indicators, premium transitions) instead of stacking
  many nested standard Compose components.
- Keep Composables small with stable, immutable parameters (data classes, not raw
  mutable state) so Compose can skip recomposition correctly.
- Always use `LazyColumn`/`LazyRow` with stable `key`s for anything that can grow
  (product lists, quote history). Never `Column` + `forEach`.
- Do heavy work (JSON parsing, bitmap decode, SQLite access) on `Dispatchers.IO`,
  surfaced to the UI through `StateFlow` — never on the main thread.
- Keep `Application.onCreate()` minimal; lazy-init anything not needed for the first
  frame, to protect cold-start time.

## APK size / build

- Enable R8 full mode + resource shrinking in release builds.
- WebP for raster images, vector drawables (`AnimatedVectorDrawable`) for icons and
  simple animations instead of Lottie.
- Avoiding third-party libraries (per the architecture skill) is itself a major size win
  — no transitive dependency bloat.

## Network-level speed

- Connection reuse (`keep-alive`), gzip request/response, sane timeouts.
- Simple in-memory + disk response cache keyed by endpoint+params for GET-heavy screens
  (product catalog, plan details) — built by hand on top of the project's own HTTP
  client, not a library.

## When C++/NDK is actually justified

Default is pure Kotlin everywhere. Only reach for native code if **profiling**
(Android Studio Profiler / Macrobenchmark) shows a measured bottleneck in one of:

- Custom cryptography/protocol work beyond what `javax.crypto`/Android Keystore covers.
- Heavy bulk numeric computation (e.g. complex premium/risk calculations over large
  datasets) where Kotlin is the *proven* bottleneck.
- Custom binary/image/video codec work — unlikely to come up in a forms/checkout
  app at all.

If none of these apply — which is the expected case for this app — don't introduce
C++. JNI boundary overhead and the maintenance cost aren't worth it for ordinary
business logic or UI work. Never introduce native code preemptively or "because
Telegram is fast" — that reasoning is based on a common misconception (see above).

## Quick self-check

- [ ] New lists use `Lazy*` + stable keys, not unbounded `Column`+`forEach`.
- [ ] No heavy work on the main thread.
- [ ] No C++/NDK introduced without a profiled, documented bottleneck.
- [ ] New images are WebP/vector, not unoptimized PNG/JPEG.

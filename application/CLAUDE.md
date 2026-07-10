# CLAUDE.md — Ritme (ریتمی) Android App

This file is the source of truth for how Claude Code must design, structure, and write
this project. Read it fully before generating any code. Every rule below is mandatory
unless the user explicitly overrides it in a follow-up instruction.

---

## 0. Common Commands

Gradle wrapper only (`./gradlew`). In this environment `dl.google.com` is blocked for
**new** dependencies, so build offline against the warm cache — do not add deps casually.

| Task | Command |
|---|---|
| Debug build | `./gradlew --offline assembleDebug` |
| Release build (R8 + shrink) | `./gradlew --offline assembleRelease` |
| Install on running emulator | `./gradlew --offline installDebug` |
| Unit tests (domain/application) | `./gradlew --offline testDebugUnitTest` |
| Instrumented / Compose UI tests | `./gradlew --offline connectedDebugAndroidTest` |
| Lint | `./gradlew --offline lintDebug` |
| Clean | `./gradlew --offline clean` |

- Emulator `Pixel_API_34` boots and installs from the CLI.
- Adding a genuinely new dependency requires network access **and** explicit approval (§3).

---

## 1. Project Summary

- **Type**: Native Android app for **Ritme (ریتمی)** — a Persian women's-health app for
  period/cycle tracking and pregnancy mode (week 1–40 tracker, daily logs, smart
  messages, banners).
- **Priority order**: (1) Speed / responsiveness, (2) small APK size, (3) visual polish,
  (4) maintainability. The app is API-heavy (cycle data, pregnancy content, logs,
  banners, user profile) — not a real-time chat app — so the bar is "feels as snappy and
  fluid as Telegram," not "needs Telegram's TDLib-level real-time engine."
- **Backend**: The existing Ritme Laravel REST API (`backend/`, prod `ritmeapp.ir`,
  local dev on `:8010` with test OTP `1111`). This app is purely a consumer of that
  API (`/api/v1/*`).
- **Target**: Android only, minSdk 26+ (Android 8.0), no iOS/cross-platform concerns.

---

## 2. Architecture: Hexagonal (Ports & Adapters)

Strict hexagonal architecture. Three concentric layers. Dependencies only point **inward**.
The domain must never import an Android class, a UI class, or any I/O class.

```
app/
├── domain/                     # Pure Kotlin. Zero Android/Compose/IO imports.
│   ├── model/                  # Entities & value objects (Cycle, PeriodLog, PregnancyWeek, User...)
│   ├── port/
│   │   ├── inbound/            # Use case interfaces (what the app can DO)
│   │   └── outbound/           # Repository/gateway interfaces (what the app NEEDS)
│   └── usecase/                # Use case implementations, orchestration logic only
│
├── application/                # Thin coordination layer between domain and adapters
│   └── service/                # Implements inbound ports, calls outbound ports
│
├── adapter/
│   ├── inbound/
│   │   └── ui/                 # Compose screens, ViewModels, navigation
│   └── outbound/
│       ├── network/            # Hand-written HTTP client + endpoint adapters
│       ├── persistence/        # Hand-written SQLite layer (cache, offline data)
│       └── di/                 # Manual dependency wiring (composition root)
│
└── platform/                   # App-level glue: Application class, manifest-bound code
```

### Rules
- `domain/` depends on **nothing** except Kotlin stdlib. No `android.*`, no `kotlinx.*`
  besides coroutines-core (justified exception, see §3).
- Every outbound capability (HTTP, disk cache, secure storage, clock, logger) is defined
  as an **interface (port)** in `domain/port/outbound`. The real implementation lives in
  `adapter/outbound/*` and is injected at the composition root — never instantiated
  inside a use case.
- ViewModels belong to the inbound adapter layer. They depend on inbound port interfaces
  (use cases), never directly on adapters/network/DB classes.
- One direction of knowledge only: `adapter → application → domain`. Never the reverse.

---

## 3. Dependency Policy: Write It Yourself

**Default rule: zero third-party libraries.** Use only:
- Kotlin standard library
- Android SDK (`android.*`, `androidx.compose.*` for UI rendering primitives only)
- `kotlinx.coroutines` — the one pre-approved exception, because hand-rolling a
  scheduler/dispatcher correctly is high-risk and coroutines are effectively a Kotlin
  language-level facility, not a "package" in the business-logic sense.

Everything else is written in-house, including the pieces people normally reach for a
library to get:

| Need | Do NOT use | Instead, hand-write |
|---|---|---|
| HTTP networking | OkHttp, Retrofit | A small client over `java.net.HttpURLConnection` (or raw `Socket`/`SSLSocket` if you need more control) with your own request builder, connection pooling, and timeout handling |
| JSON parsing | Gson, Moshi, kotlinx.serialization | A minimal hand-written JSON tokenizer/parser, or at most `org.json` (it ships in the Android SDK itself, so it does not count as a third-party dependency) |
| Local DB / cache | Room | Direct `android.database.sqlite.SQLiteOpenHelper` with hand-written DAOs |
| Dependency injection | Hilt, Koin, Dagger | Manual constructor injection wired in one composition-root file (`adapter/outbound/di`) |
| Image loading/caching | Coil, Glide | A small `LruCache<String, Bitmap>` + manual disk cache + `BitmapFactory` decode on a background dispatcher |
| Lists/collections | any third-party collection lib | Kotlin stdlib `List`/`MutableList`/`ArrayDeque` are fine — **only** write a custom data structure if you have a measured, specific performance reason (e.g., a custom ring buffer for a hot loop). Do not write a custom `ArrayList` "just because." |
| Animations | Lottie | Compose's built-in `animate*AsState`, `Animatable`, and vector drawables/`AnimatedVectorDrawable` |
| Logging | Timber | A 20-line wrapper around `android.util.Log` with build-type gating |

**Why**: full control over binary size, no transitive-dependency bloat, no opaque
behavior to debug around, and it forces every abstraction to match this app's actual
needs instead of a generic library's.

**Exception process**: if Claude Code believes a specific case truly needs a library
(e.g. a cryptographic primitive that must not be hand-rolled for security reasons), it
must stop and explicitly ask before adding it — never add silently.

---

## 4. Code Quality Rules

### SOLID
- **S**: One class, one reason to change. A ViewModel does not parse JSON; a repository
  does not contain UI state.
- **O**: New tracking modes/content types extend behavior via new implementations of an
  existing interface (e.g. a `CycleInsightProvider` port), not via editing existing `when`
  blocks scattered around the codebase. Prefer polymorphism over branching for variant
  behavior, but don't force an interface where a simple `when` over a sealed class is
  clearer — sealed classes + exhaustive `when` are idiomatic Kotlin and acceptable.
- **L**: Any implementation of a port must be substitutable without breaking the
  use case that consumes it (e.g. a `FakeCycleRepository` for tests must behave
  consistently with the real one).
- **I**: Keep port interfaces small and role-specific (`CycleRepository`,
  `PregnancyContentGateway`) instead of one giant `BackendGateway` interface.
- **D**: Use cases and ViewModels depend on abstractions (ports) defined in `domain`,
  never on concrete adapter classes.

### Clean Code
- Functions do one thing. If you need "and" to describe a function, split it.
- No magic numbers/strings — name them as constants in the relevant layer.
- No God objects. No ViewModel over ~150 lines — extract use cases instead of
  growing the ViewModel.
- Immutable data classes for domain models (`val`, not `var`); represent state changes
  by producing new instances, not mutation, unless profiling proves it's a hot path.
- Sealed classes/interfaces for finite states (`UiState`, `Result<T>` equivalents,
  network response states) instead of nullable flags or boolean combinations.
- Naming: descriptive, no abbreviations except extremely common ones (`vm`, `id`).
  Use cases named as verbs (`LogPeriodUseCase`), repositories as nouns
  (`CycleRepository`).
- Every public function in `domain/` and `application/` gets a short KDoc explaining
  intent, not mechanics ("why," not "what the code already shows").

### State Management (MVI)
Every screen follows a small, uniform MVI shape — no ad-hoc `mutableStateOf` flags
scattered across a ViewModel:
- **State**: one immutable `data class XxxUiState` (or a sealed `interface` when the
  screen has genuinely distinct modes: `Loading` / `Content` / `Empty` / `Error`).
  Marked `@Immutable`; exposed as a single `StateFlow<XxxUiState>`.
- **Intent/Event**: a sealed `interface XxxIntent` for everything the user can do
  (`Submit`, `Retry`, `FieldChanged`). The ViewModel exposes one `fun onIntent(i)`;
  the screen never calls bespoke ViewModel methods.
- **Effect**: one-shot side effects (navigation, snackbar, toast) go through a
  `Channel`/`SharedFlow<XxxEffect>` — never modeled as state, so they don't replay on
  recomposition or config change.
- The ViewModel maps use-case `Result` into state transitions only; it does no I/O,
  parsing, or SQL itself (§4 S, §5).

### Error Handling
- No silent failures. Network/parsing/storage adapters return a sealed
  `Result<T, AppError>` type (hand-written, not a library) — never throw across a
  port boundary.
- Domain layer never catches Android-specific exceptions; that's an adapter concern.
- **Stable error codes (`RIT-XXXX`).** Every `AppError` variant carries a stable code so
  logs, crash reports (§7), and user messages line up across app versions. Reserve
  ranges by concern, and keep the catalog in one Kotlin file (`domain/model/AppError.kt`),
  never inline literals:
  - `RIT-1xxx` — network/transport (timeout, no connectivity, TLS, non-2xx).
  - `RIT-2xxx` — parsing/serialization (malformed JSON, missing field).
  - `RIT-3xxx` — persistence/cache (SQLite, disk).
  - `RIT-4xxx` — auth/session (expired token, `accessDenied`/lockout — cf. the `1005`
    backend contract).
  - `RIT-5xxx` — domain/business-rule violations (invalid cycle data, out-of-range week).
  - `RIT-9xxx` — unknown/unexpected (maps to the Crash Guard fatal path).
  - The user-facing Persian message is derived from the code, never hardcoded per call
    site; the raw code is what gets logged/reported.

---

## 5. Performance Playbook (the "feels like Telegram" part)

The key insight from research before this file was written: Telegram's perceived speed
comes mostly from **custom-drawn UI avoiding the standard widget/layout overhead**, not
from being written in C++. Apply the same philosophy here, in Compose terms:

- Favor `Canvas`/`drawWithContent` custom drawing for hot, frequently-recomposing UI
  (e.g. animated lists, custom progress indicators, premium-feeling transitions)
  instead of stacking many nested standard Compose components.
- Keep Composables small and stable; pass immutable data classes as parameters so
  Compose can skip recomposition correctly. Avoid lambdas that capture changing state
  unnecessarily.
- Use `LazyColumn`/`LazyRow` with stable `key`s for every list 
  history) — never `Column` + `forEach` for anything that can grow.
- Defer/avoid heavy work on the main thread: JSON parsing, bitmap decoding, and SQLite
  access all happen on `Dispatchers.IO` via coroutines, surfaced to the UI as
  `StateFlow`.
- Enable R8 full mode + resource shrinking in `release` builds. Use WebP for raster
  images, vector drawables for icons.
- Network layer: connection reuse (`keep-alive`), gzip request/response, sane timeouts,
  and a simple in-memory + disk response cache keyed by endpoint+params for
  GET-heavy screens (pregnancy content, banners).
- Cold start budget: keep `Application.onCreate()` minimal; lazy-init anything not
  needed for the first frame.
- **Recomposition hygiene.** Annotate every state model passed into Composables with
  `@Immutable` (immutable data) or `@Stable` (mutable but with stable-notifying reads).
  Never pass a raw `List<T>` that Compose treats as unstable into a hot Composable — use
  a small hand-written `@Immutable` list wrapper (per §3, no third-party immutable-
  collections lib) or a stable snapshot type. Prefer deferred reads (lambda/`State`
  parameters) for values that change often (scroll offset, drag position) so only the
  drawing layer re-reads them, not the whole subtree.
- **Baseline Profiles.** Ship a Baseline Profile covering the two paths users feel most:
  cold-start-to-first-screen and the main list scroll (home / cycle history list).
  Regenerate it whenever startup wiring or a primary list changes — a stale profile is
  worse than none because it AOT-compiles the wrong code. (Generate with the Macro-
  benchmark module; this uses only AndroidX test tooling, no third-party dependency.)
- **Low-end device budget (non-negotiable §1 target).** Assume ≤2 GB RAM, weak CPU:
  size the image `LruCache` off `ActivityManager.memoryClass` (a fraction of it, not a
  fixed MB), always decode bitmaps downsampled with `BitmapFactory.Options.inSampleSize`
  to the target view size (never full-resolution), and cap in-flight decodes. Avoid
  large allocations on scroll; reuse buffers in hot loops.

---

## 5b. Navigation: Telegram-style Swipe-to-Go-Back

Every screen except the root/start destination must be dismissible with an
**edge swipe gesture**, exactly like Telegram: the user drags from the left edge
toward the right and the current screen slides away to reveal the previous one,
following the finger; releasing past a threshold completes the pop, releasing
before it snaps back.

### Rules
- **Mandatory on every non-root screen.** A new screen is not "done" until its
  swipe-back works (add it to the §11 checklist mentally for every screen).
- **Hand-built, no library.** Per §3, do not pull in a gesture/navigation library.
  Implement with Compose primitives: `pointerInput` + `detectHorizontalDragGestures`
  (or `AnchoredDraggable`), an `Animatable`/`animate*AsState` for the slide offset,
  and a back-stack `pop` triggered when the drag passes the commit threshold.
- **Feels like Telegram (§5 performance):** the screen tracks the finger 1:1 in real
  time (no lag), the underlying previous screen is visible during the drag with a
  subtle parallax/dim, and the spring on release is quick and natural. Drive the
  offset with custom drawing/offset, not by recomposing a heavy tree each frame.
- **Threshold:** commit the pop at roughly >40% width dragged or a fast fling;
  otherwise animate back to 0. Tune for a snappy feel.
- **Edge start:** begin tracking only from a left-edge zone (e.g. first ~20dp) so the
  gesture does not fight horizontal scrolling/`LazyRow` content inside the screen.
- **Consistency:** centralize this in one reusable `SwipeBackContainer` composable that
  wraps screen content and calls the nav `pop` — do not re-implement per screen.
- **Accessibility/back parity:** the system Back button/gesture must do the exact same
  pop; swipe-back is an addition, never the only way back.

---

## 5c. Brand & Design System (Ritme / ریتمی)

This app is **Ritme (ریتمی)** — a Persian women's-health / cycle-tracking app,
package `ir.ritmeapp.ritme`. The brand identity is **pink-based** (warm, feminine,
caring), with a violet accent — the same tokens as the web frontend
(`frontend/src/app/globals.css`).

### Palette (source of truth: `app/src/main/res/values/colors.xml`)
> The hex values mirror the web frontend design tokens. When a value changes, adjust
> the `ritme_*` values in one place (`colors.xml`); never hardcode hex literals in
> Composables.

- Primary brand pink `ritme_pink #E91E63` (deep `#E60076`, light `#FB64B6`,
  soft container `#FFF1F7`).
- Accent violet `ritme_accent #A91EE9` for highlights / secondary CTAs.
- Neutrals: ink `#11202F`, muted `#707983`, surface `#FFFFFF`, background `#EFF2F4`,
  outline `#E6EAF0`. Dark theme variants provided (`*_dark`).
- Semantic: success `#22B07D`, warning `#F5A623`, error `#D64545`, info `#2E7CD6`.

### Rules
- **No magic colors** (ties to §4): every color comes from a named token mapped to a
  Compose `ColorScheme`/theme object — no inline `Color(0xFF...)` in screens.
- **RTL-first.** Ritme is a Persian app: default layout direction is RTL, text is
  right-aligned, and the swipe-back gesture (§5b) still means "from the leading/left
  edge → back" visually. Mirror icons/chevrons where directional.
- **Persian typography.** Bundle a Persian font (e.g. a Vazirmatn/IRANSans-style family)
  as an app asset and expose it as the default `FontFamily` — do NOT pull a font library
  (§3); ship the font file directly. Use Persian (۰۱۲۳) digits in user-facing numbers.
- **Light + dark theme** both defined from the same tokens. Respect system setting.
- One central `RitmeTheme` composable provides colors, typography, and shapes; screens
  read from it, never define their own palette.

---

## 6. When (and only when) to Use C++ / NDK

Default is **pure Kotlin everywhere.** Reach for C++ via NDK/JNI only if profiling
(Android Studio Profiler / Macrobenchmark) shows a *measured* bottleneck in one of these
specific categories — never preemptively:

- Custom cryptography or protocol-level work beyond what `javax.crypto`/Android
  Keystore already provides.
- Heavy bulk numeric computation (e.g. heavy cycle-prediction computation over large
  datasets) where Kotlin is the proven bottleneck.
- Custom binary/image/video codec work — unlikely to apply to this app at all.

If none of these apply (which, for a CRUD/forms health-tracking app, is the expected
case), do not introduce C++. JNI boundary overhead and maintenance cost are not worth it
for ordinary business logic or UI.

---

## 7. Crash Resilience & Error Reporting

**Goal**: the app must never show the user a dead screen or a system "App has stopped"
dialog. On any error — fatal or non-fatal — the user lands on the *last known safe
screen*, and a structured report is queued for upload so the team can see, per app
version, what's actually breaking in the field. No third-party crash SDK (no
Crashlytics, no Sentry) — this is hand-built, same as everything else in this project.

### 7.1 Two classes of error

- **Non-fatal (handled) errors** — an API call fails, a parse fails, a use case throws
  inside a coroutine. These are caught locally at the use case/ViewModel boundary,
  surfaced as a sealed `Result.Error` (see §4 Error Handling), shown as an inline
  message/snackbar, and logged — the app keeps running normally.
- **Fatal (uncaught) errors** — anything that would otherwise crash the process. These
  go through the **Crash Guard** described below.

### 7.2 Last-Safe-Screen tracking

- A `SafeStateRepository` (outbound port + SQLite-backed adapter) persists a tiny record
  after every successful screen render: `{ route, minimalArgs, timestampMillis }`.
  Writes are cheap and debounced (e.g. on `LaunchedEffect` per screen, not on every
  recomposition).
- "Minimal args" means just enough to reconstruct the screen safely (e.g. a log date or week number),
  never large objects — this record must be small and fast to write since it has to
  survive being written right up to the moment of a crash.

### 7.3 Crash Guard (fatal path)

1. In `Application.onCreate()`, install a custom handler via
   `Thread.setDefaultUncaughtExceptionHandler`, wrapping the platform default handler.
2. Inside the handler, **synchronously and fast** (the process is dying, no coroutines,
   no disk-heavy work):
    - Capture: stack trace, app version name + version code, OS version/SDK int, device
      model, free memory, last safe screen record, a short in-memory breadcrumb trail
      (see §7.5).
    - Write this payload as a single small file into app-private storage
      (`filesDir/crash_reports/<timestamp>.json`), hand-written JSON serialization —
      do not pull in a JSON library for this either.
    - Set a `PendingIntent` to relaunch the app's entry `Activity` with an extra
      `recovered_from_crash = true` plus the last safe route, then call
      `Process.killProcess(Process.myPid())` followed by `exitProcess(0)`.
3. On next launch, the entry `Activity`/navigation root checks for that extra (or a
   small "needs recovery" flag file if relaunch didn't carry the intent, e.g. after an
   OS-initiated restart). If present, navigate directly to the last safe screen instead
   of the default start destination, and show a small one-time "something went wrong,
   you're back where you left off" message — calm, not alarming.

### 7.4 Uploading reports

- A simple `CrashReportUploader` runs at app startup (and optionally on a periodic
  in-app timer while the app is foregrounded — no WorkManager, keep it hand-rolled):
  scans `filesDir/crash_reports/`, sends each pending report through the existing
  hand-written HTTP client to a dedicated backend endpoint (e.g.
  `POST /api/diagnostics/crash-reports`), and deletes the local file only after a `2xx`
  response. Failed uploads stay on disk and retry next launch — never lost, never
  duplicated thanks to the delete-after-ack rule.
- Non-fatal errors (§7.1) are queued through the same uploader/endpoint, tagged
  `severity: "non_fatal"` vs `"fatal"`, so both flow into one place on the backend.
- Payload should always include `appVersionCode` and `appVersionName` as first-class
  fields — the entire point is being able to ask "what's breaking in version 4.2.1
  specifically" later.

### 7.5 Breadcrumbs

- A small in-memory ring buffer (max ~20 entries, custom-written, no library) records
  lightweight events as the user navigates: screen entered, button tapped, API call
  started/failed. Each crash/error report includes the last N breadcrumbs so a report
  shows not just *where* it broke but the few steps leading up to it.
- Breadcrumbs are ephemeral (memory only) — never persisted on their own, only embedded
  inside a crash/error payload when one is actually written.

### 7.6 What NOT to do

- Don't try to "catch everything" inside Composables with broad try/catch — Compose
  recomposition errors are not reliably recoverable that way. The defense is at the
  process boundary (§7.3) plus careful error handling inside use cases/ViewModels
  (§7.1), not scattered try/catch in UI code.
- Don't block the uncaught-exception handler on network I/O — by the time it runs, the
  process may have only milliseconds left. Always write locally first, upload later.

---

## 8. Folder/Module Conventions

- Single Gradle module is fine at this stage (`:app`); do not over-engineer into
  multi-module until the domain layer is proven stable and large enough to warrant it.
  Keep the `domain/application/adapter` separation enforced by **package** boundaries
  and code review discipline even within one module.
- One file per class/interface. No "Utils.kt" dumping grounds — extension functions
  live next to the type they extend or in a clearly named file
  (`StringFormatting.kt`, not `Utils.kt`).

---

## 9. Testing

- `domain/` and `application/`: pure JUnit unit tests, no Android dependencies, no
  mocking framework needed beyond hand-written fakes implementing the port interfaces
  (this is *why* ports are small — fakes stay cheap to write).
- `adapter/outbound/network` and `persistence`: integration tests against a local
  test server / in-memory SQLite where feasible.
- `adapter/inbound/ui`: Compose UI tests for critical flows (login/OTP, period logging,
  pregnancy tracker) using `androidx.compose.ui.test` (part of the Android testing
  toolkit, not a third-party add-on).

---

## 10. Git & Workflow

- Conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `chore:`).
- Each PR/commit should respect the layer boundaries above — a commit that adds a
  network call directly inside a Composable is a violation and should be rejected/
  reworked.

---

## 11. Summary Checklist (Claude Code must self-verify against this before finishing any task)

- [ ] No new third-party dependency added without explicit user approval.
- [ ] Domain layer has zero Android/IO imports.
- [ ] All outbound I/O accessed through a port interface, real implementation in an adapter.
- [ ] No ViewModel doing parsing/networking/SQL directly.
- [ ] Lists are `Lazy*` + stable keys; no unbounded `Column`+`forEach`.
- [ ] No C++ introduced without a profiled, documented bottleneck.
- [ ] New error paths return sealed `Result`, never throw across a port boundary, and
  each `AppError` variant has a stable `RIT-XXXX` code (§4 Error Handling).
- [ ] New screens follow the MVI shape: one `@Immutable` state `StateFlow`, sealed
  `Intent` via a single `onIntent`, one-shot effects off a channel (§4 State Management).
- [ ] Hot Composables receive `@Immutable`/`@Stable` params (no raw unstable `List`);
  a changed startup path or primary list updates the Baseline Profile (§5).
- [ ] Every new screen registers a last-safe-screen write on successful render (§7.2).
- [ ] Any new fatal-failure-prone code path is covered by the Crash Guard, not a
  scattered local try/catch around UI code (§7.3, §7.6).
- [ ] Naming and file structure match §4 and §8.

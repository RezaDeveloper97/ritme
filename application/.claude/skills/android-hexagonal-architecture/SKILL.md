---
name: android-hexagonal-architecture
description: Use this skill whenever writing, reviewing, or restructuring any code in this Android women's-health app (Ritme) — new screens, repositories, use cases, network/storage adapters, dependency wiring, or any file under app/. It enforces strict hexagonal (ports & adapters) architecture, a zero-third-party-dependency policy (hand-write your own HTTP client, JSON parsing, local DB layer, DI, image cache — only Kotlin stdlib + Android SDK + coroutines allowed), and SOLID/clean-code rules. Always consult this before adding a Gradle dependency, creating a ViewModel, a repository, or any class that talks to the network/disk. Also use it when the user asks "where should this class go," "should I add this library," or "is this following our architecture."
---

# Android Hexagonal Architecture & Zero-Dependency Policy

This skill governs the structural rules for the Ritme Android app. It is
the single source of truth for where code lives and what it's allowed to depend on.

## When to apply this

- Creating any new class, screen, repository, use case, or adapter.
- Reviewing a diff/PR for architecture violations.
- The user proposes adding a Gradle dependency (Retrofit, Room, Hilt, Glide, etc.) —
  this skill says no by default; see "Dependency decisions" below for the exception path.
- Deciding which package a piece of logic belongs in.

## The three layers (strict inward-only dependency direction)

```
domain/            Pure Kotlin. Zero android.*, zero IO. Models, ports, use cases.
application/        Thin coordination layer implementing inbound ports.
adapter/
  inbound/ui/        Compose screens + ViewModels. Depend on inbound ports only.
  outbound/network/   Hand-written HTTP client + endpoint adapters.
  outbound/persistence/ Hand-written SQLite layer.
  outbound/di/        Manual composition root (constructor injection wiring).
```

**Rule of thumb**: if you're about to write `import android.*` or `import kotlinx.*`
(other than coroutines) inside something in `domain/`, stop — that logic belongs in an
adapter, accessed through a port interface defined in `domain/port/outbound`.

ViewModels never call a network/DB class directly. They depend on use case interfaces
(inbound ports). The real wiring happens once, in the composition root
(`adapter/outbound/di`), nowhere else.

## Dependency decisions

Default answer to "can I add library X?" is **no**. The whole point of this project is
full hand-written control. Concretely:

| Instead of | Write |
|---|---|
| Retrofit/OkHttp | Thin client over `HttpURLConnection`, own request builder/timeouts |
| Room | `SQLiteOpenHelper` + hand-written DAOs |
| Gson/Moshi/kotlinx.serialization | Hand-written JSON parser, or `org.json` (ships in Android SDK, not a 3rd-party add) |
| Hilt/Koin/Dagger | Manual constructor injection in one composition-root file |
| Glide/Coil | `LruCache<String,Bitmap>` + manual disk cache + background `BitmapFactory` decode |
| Lottie | Compose `animate*AsState`/`Animatable` + vector drawables |
| Timber | ~20-line wrapper over `android.util.Log` |

`kotlinx.coroutines` is the one pre-approved exception (treated as a language facility,
not a business-logic dependency).

If there's a genuinely strong case for an exception (e.g. a crypto primitive that
shouldn't be hand-rolled for safety reasons), stop and ask the user explicitly — never
add a dependency silently.

## SOLID / Clean Code checklist (apply on every new class)

- Single responsibility: a ViewModel doesn't parse JSON; a repository doesn't hold UI
  state.
- Depend on abstractions: use cases/ViewModels depend on `domain/port` interfaces, never
  concrete adapter classes.
- Small, role-specific port interfaces (`QuoteRepository`, `PaymentGateway`) over one
  giant `BackendGateway`.
- Immutable domain models (`val`, data classes); represent change as new instances.
- Sealed classes for finite states instead of nullable flags/boolean soup.
- No "Utils.kt" dumping grounds; extension functions live in a named file next to what
  they extend.
- Errors cross port boundaries as a sealed `Result<T, AppError>`, never a thrown
  exception.

## Quick self-check before finishing any task

- [ ] No new Gradle dependency without explicit user approval.
- [ ] `domain/` has zero Android/IO imports.
- [ ] Every outbound I/O call goes through a port interface.
- [ ] No ViewModel doing parsing/networking/SQL directly.
- [ ] New states modeled as sealed classes, not booleans/nullables.

See the project's `CLAUDE.md` for the full rationale and the crash-resilience and
performance companion skills for those specific concerns.

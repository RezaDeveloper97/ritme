---
name: android-crash-resilience
description: Use this skill whenever implementing or touching error handling, crash recovery, navigation state, app startup/entry-point logic, or anything related to reporting failures to the backend in this Android app (Ritme). Triggers on requests like "handle this error," "what happens if this API call fails," "add crash reporting," "the app shouldn't crash," "send errors to the server," "recover to the last screen," or any work on Application.onCreate, uncaught exception handling, or diagnostics endpoints. This skill defines a fully hand-built (no Crashlytics/Sentry) crash guard that prevents dead-screen crashes, restores the user to their last known safe screen, and queues structured error reports — tagged by app version — for later analysis.
---

# Crash Resilience & Self-Built Error Reporting

This skill defines how the app survives errors without showing a broken screen, and how
it reports problems back to the team without using any third-party crash SDK.

## When to apply this

- Writing or editing `Application.onCreate()`, the navigation root, or entry `Activity`.
- Adding error handling inside a ViewModel, use case, or repository.
- Building or modifying anything that talks to a "report this error to the backend"
  endpoint.
- The user mentions crashes, "app stopped working," recovery behavior, or wanting to
  know what's breaking in a specific app version.

## Two classes of error — handle them differently

1. **Non-fatal (handled)** — an API call fails, parsing fails, a use case throws inside
   a coroutine. Catch it at the use case/ViewModel boundary. Surface it as a sealed
   `Result.Error`. Show an inline message. The app keeps running. Still log/queue it
   (tagged `severity: "non_fatal"`) — see "Reporting" below.
2. **Fatal (uncaught)** — anything that would otherwise kill the process. This goes
   through the Crash Guard (below). Never try to "catch everything" with broad
   try/catch scattered through Composables — that's not reliable. The real defense is
   at the process boundary.

## Last-Safe-Screen tracking (do this first — everything else depends on it)

Maintain a `SafeStateRepository` (SQLite-backed, behind a port interface) that, after
every screen successfully renders, persists a small record:

```
{ route: String, minimalArgs: Map<String,String>, timestampMillis: Long }
```

Keep `minimalArgs` tiny (e.g. just a plan ID) — this write has to be cheap because it
needs to be current right up to the moment something fails. Debounce the write (e.g.
once per screen via `LaunchedEffect`), don't write on every recomposition.

## Crash Guard (the fatal path)

1. In `Application.onCreate()`, install:
   `Thread.setDefaultUncaughtExceptionHandler { thread, throwable -> ... }`,
   wrapping (not discarding) the platform default handler.
2. Inside the handler — **synchronous, fast, no coroutines, no network**, because the
   process may have milliseconds left:
   - Build a payload: stack trace, `versionName`/`versionCode`, SDK int, device model,
     last safe screen record, last ~20 breadcrumbs (see below).
   - Hand-serialize it to JSON (no library) and write it to
     `filesDir/crash_reports/<timestamp>.json`.
   - Fire a `PendingIntent` to relaunch the entry `Activity` with
     `recovered_from_crash = true` + the last safe route as an extra.
   - `Process.killProcess(Process.myPid())`, then `exitProcess(0)`.
3. On the next launch, the entry point checks for that extra (or a small on-disk "needs
   recovery" flag, in case the OS relaunched it some other way). If present: navigate
   straight to the last safe screen instead of the normal start destination, and show a
   brief, calm "you're back where you left off" message — never an alarming error
   screen.

## Breadcrumbs

A small custom in-memory ring buffer (cap ~20 entries, no library) logging lightweight
events: screen entered, button tapped, API call started/failed. Embed the last N
breadcrumbs into any crash or non-fatal report so reports show the steps leading up to
the failure, not just the failure point. Breadcrumbs live only in memory — never
persisted standalone.

## Reporting to the backend

- A `CrashReportUploader`, run at app startup (and optionally on a lightweight in-app
  foreground timer — do **not** reach for WorkManager, keep this hand-rolled like
  everything else in this project), scans `filesDir/crash_reports/`, POSTs each report
  through the project's existing hand-written HTTP client to a diagnostics endpoint
  (e.g. `POST /api/diagnostics/crash-reports`), and deletes the local file only after a
  `2xx` response. Anything that fails to upload stays on disk and retries next launch.
- Route non-fatal errors through the same uploader/endpoint, tagged
  `severity: "non_fatal"`.
- Always include `appVersionCode`/`appVersionName` as top-level fields in the payload —
  this is what lets the team later filter "what's breaking specifically in version
  4.2.1."

## What NOT to do

- Don't wrap large swaths of Composables in try/catch hoping to "catch the crash" —
  Compose recomposition failures aren't reliably caught that way.
- Don't do network I/O inside the uncaught-exception handler itself — write locally
  first, upload on next launch.
- Don't use a third-party crash/analytics SDK — this is intentionally hand-built so the
  team controls the data and the backend it lands on.

## Quick self-check

- [ ] New screens write a last-safe-screen record on successful render.
- [ ] New fatal-failure-prone paths are covered by the Crash Guard, not local try/catch
      in UI code.
- [ ] New handled errors return a sealed `Result.Error` and get queued for reporting.
- [ ] Crash/error payloads include app version code/name + breadcrumbs.

See the project's `CLAUDE.md` §7 for the full rationale, and the
`android-hexagonal-architecture` skill for where these classes belong (ports in
`domain/`, real implementation in `adapter/outbound`).

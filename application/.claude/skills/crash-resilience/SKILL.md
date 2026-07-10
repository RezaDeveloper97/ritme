---
name: crash-resilience
description: Apply the hand-built crash-resilience & error-reporting system (Crash Guard, last-safe-screen, breadcrumbs, report upload). Use when touching error handling, the Application class, navigation root, or any fatal-prone code path. No third-party crash SDK allowed.
---

# Crash Resilience & Error Reporting

CLAUDE.md §7. Goal: the user NEVER sees a dead screen or "App has stopped" dialog.
On any error they land on the last known safe screen; a structured report is queued.
No Crashlytics/Sentry — all hand-built.

## Two classes of error

- **Non-fatal (handled)** — API/parse/use-case failure inside a coroutine. Caught at
  use case/ViewModel boundary, returned as sealed `Result.Error`, shown inline/snackbar,
  logged. App keeps running.
- **Fatal (uncaught)** — would crash the process. Goes through the Crash Guard.

## Last-safe-screen

- `SafeStateRepository` (outbound port + SQLite adapter) persists
  `{ route, minimalArgs, timestampMillis }` after every successful render.
- Write is cheap & debounced (per-screen `LaunchedEffect`, NOT per recomposition).
- minimalArgs = just enough to rebuild safely (e.g. a plan ID), never large objects.

## Crash Guard (fatal path)

1. In `Application.onCreate()` install `Thread.setDefaultUncaughtExceptionHandler`,
   wrapping the platform default.
2. In the handler — **synchronous & fast, no coroutines, no heavy disk**:
   - capture stack trace, appVersionName + versionCode, OS/SDK int, device model,
     free memory, last safe screen, last N breadcrumbs.
   - write ONE small file: `filesDir/crash_reports/<timestamp>.json` (hand-written JSON).
   - set `PendingIntent` to relaunch entry Activity with `recovered_from_crash=true`
     + last safe route, then `Process.killProcess(myPid())` + `exitProcess(0)`.
3. On next launch the nav root checks the extra (or a small "needs recovery" flag file
   for OS-initiated restarts) → navigate to last safe screen + show a calm one-time
   "you're back where you left off" message.

## Upload

- `CrashReportUploader` at startup (and optional foreground timer — no WorkManager):
  scan `filesDir/crash_reports/`, POST each via the hand-written HTTP client to
  `POST /api/diagnostics/crash-reports`, **delete only after 2xx**. Failures retry next launch.
- Non-fatal errors flow through the same endpoint tagged `severity: "non_fatal"` vs `"fatal"`.
- Payload always includes `appVersionCode` + `appVersionName` as first-class fields.

## Breadcrumbs
- In-memory ring buffer (~20 entries, custom, no library): screen entered, button tapped,
  API started/failed. Embedded in each report. Never persisted on their own.

## Checklist

- [ ] New fatal-prone path covered by Crash Guard, NOT scattered try/catch in Composables (§7.6).
- [ ] No broad try/catch around Compose recomposition.
- [ ] Uncaught handler does NO network I/O — write locally first, upload later.
- [ ] New screen registers a last-safe-screen write on successful render.
- [ ] Errors crossing a port boundary returned as sealed `Result`, never thrown.

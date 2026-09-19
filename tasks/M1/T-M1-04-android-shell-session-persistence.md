---
id: T-M1-04
title: Android shell — session survives kill, update and low-memory
milestone: M1
type: android
status: done
depends_on: [T-M1-01]
parallel_group: M1-B
touches: [android-shell/app/src/main]
skills: []
verify: cd android-shell && ./gradlew assembleDebug
---

# T-M1-04 — Android shell — session survives kill, update and low-memory

## Why
The shipped Android app is the WebView shell (memory `ritme-android-shell`). Its storage must keep the session
for a year.

## Scope
- Apply whatever T-M1-01 found for the WebView (e.g. flush cookies also on `onStop`, avoid any `clearCache`/
  `clearHistory`/data dir changes, keep the same WebView data directory across app updates).
- Verify `android:allowBackup` / data-extraction rules don't wipe or restore a stale token.
- Build with the JDK noted in memory `android-build-jdk21`.

## Scope change from T-M1-01 (docs/investigations/session-logout.md)
**The shell is not implicated.** In prod logs from 2026-09-07 to 09-19, 21 of 22 post-sign-in cold launches from 24
shell devices kept the session (for up to 11.1 days). The one loss was a full WebView profile wipe (reinstall or clear
data). Reduce the scope to:
- Optional hardening: also `CookieManager.getInstance().flush()` in `onStop`.
- Check that store updates install over the existing app (same signing key, no forced uninstall).
- Run the two manual tests. If both pass, mark done without a code change.

## Out of scope
The web app's own storage logic (T-M1-03).

## Acceptance
- Debug build green. Written manual test: sign in → force-stop → reopen; sign in → install an updated APK over it
  → reopen; both stay signed in. Record the result in the task report.
- If T-M1-01 shows the shell is not involved, mark done with that note (no code change).

## Result
T-M1-01 cleared the shell, so this closes under the last acceptance bullet. Added `CookieManager.flush()` in
`MainActivity.onStop()` (the existing `onPause` flush was kept). `assembleDebug` is green with JBR 21 and JDK 17.
`allowBackup=false` and there are no extraction rules. The WebView data directory stays the same across updates, and
every release is signed with the same key (`ritme-release.jks`, SHA-256 `46d68206…`). Note: `android-shell/` is
gitignored, so the code change exists only in the working tree. The manual force-stop and update tests still need a
human with a device; the script is in PROGRESS.md.

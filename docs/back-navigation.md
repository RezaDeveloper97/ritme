# Back navigation — behaviour and device checklist

Rules for code: `frontend/CLAUDE.md` §4.2. Mechanism: `frontend/src/shared/back-stack/stack.ts`.

## Expected behaviour

Every Back steps back **one layer**:

1. soft keyboard open → only the keyboard closes (Android's IME takes the press);
2. a sheet / modal open → only that closes, the route stays;
3. nothing open → the previous screen; with no previous screen in the app → normal Android (the TWA closes).

Reference flow: **home → `/cycle` → log sheet («ثبت علائم»)**. Back #1 closes the sheet (still on `/cycle`),
Back #2 returns home, Back #3 on home (nothing behind) leaves the app.

## How it is built

| Layer | History entry | Closed by Back | Closed from the UI (×, swipe, backdrop, Escape, save) |
|---|---|---|---|
| Inline sheet (`AppSheet` + own `open`) | one, same URL, tagged `__ritmeLayer` (`useBackLayer`) | `popstate` → `onClose`; **no further `back()`** | `open` → false → the hook pops its own entry; that `popstate` is swallowed |
| URL sheet (`?sheet=`) | one per sheet (`shared/sheet/controller`) | `popstate` → stack shrinks | `closeSheet()` → `history.go(-(1 + inline sheets inside it))` |
| Screen | Next's own | browser | header arrow `router.back('/parent')`: history when the parent is right behind, else `replace(parent)` |

- A sheet closed because a link inside it was followed does **not** pop its entry (it would cancel the navigation);
  the entry is left tagged and stepped over by the next Back.
- Unsaved log edits survive a Back: `features/log-day/model/drafts.ts` (memory only, wiped at session end).
- Sign-out hides the page before its `location.replace`, so a bfcache restore via Back shows nothing of the old
  session; `SessionGuard` then sends it to sign-in.
- The old trap (`shared/back-guard`, a duplicate entry above every screen, `window.__ritmeBack`) is gone: it made
  Back do nothing on screens, which the TWA can't override.

## Device checklist (installed TWA — not a Chrome tab)

Run each row twice: **gesture navigation** and **three-button navigation** (Settings → System → Gestures →
System navigation). Mark ✅/❌ with the Android version and device.

| # | Scenario | Expected |
|---|---|---|
| 1 | Home → «چرخه» → open «ثبت علائم» → tap a text field (keyboard up) → Back | keyboard closes; sheet and screen stay |
| 2 | (continue) Back | sheet closes; still on the cycle screen |
| 3 | (continue) Back | home |
| 4 | (continue) Back on home, app launched fresh | leaves the app (normal Android) |
| 5 | Open the log sheet → close with × → Back | home (not dead, sheet doesn't reopen) |
| 6 | Same with swipe-down on the grip, and with a tap on the scrim | as 5 |
| 7 | Log sheet → open a detail panel (pain / weight) → Back, Back | panel closes, then the log sheet |
| 8 | Inline sheet on a screen (e.g. profile quick-edit) → Back | only the sheet closes |
| 9 | Open the cycle screen from a notification / link (cold start) → Back | leaves the app; nothing forces home |
| 10 | (as 9) tap the header back arrow | goes to the parent screen; then Back leaves the app |
| 11 | Home → cycle → header back arrow → Back | leaves the app / previous screen — never back to cycle |
| 12 | Log sheet: change a few items, don't save → Back → reopen the sheet for the same day | edits still there, «ذخیره» pending |
| 13 | Form field focused in a full sheet | field and the save bar stay above the keyboard and the nav bar |
| 14 | Sheet open: drag on the scrim / over-scroll the sheet body | the screen behind doesn't move |
| 15 | Sign out → Back | no screen of the old session is shown |

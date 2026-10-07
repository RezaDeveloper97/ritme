# PRIV — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png`. Screens: 390 px, fa, headless Chrome over CDP (a copy of
`bloom/bin/shot.mjs --token` with one extra «run JS before navigating» step to plant an app-lock config) against the
worktree Go API :8241 + Next dev :3117. Test user **persona 04 `09900000004`** in `ritme_dev` (onboarding name set to
«سارا رضایی», chronic illness «thyroid», health-record basics blood type O+ / allergy «پنی‌سیلین», emergency card with
`show_on_lock_screen: true`, contact «علی · همسر · 09121112233» and insurance «تکمیلی» last 4 `4821` — all through the
API). The lock screen shots plant a dummy `ritme_app_lock` config in localStorage (no real passcode). Colours are the
app's tokens (`docs/canvas-build/README.md` §4), not the board hex.

## CB-PRIV-01

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Priv_Settings` | `/profile/privacy` | [settings](priv/CB-PRIV-01/fa_profile_privacy.light.png) | [settings](priv/CB-PRIV-01/fa_profile_privacy.dark.png) | ✔ | Title «حریم خصوصی و قفل»; «قفل اپ» group: «قفل هنگام باز کردن اپ» switch, and with the lock on «روش باز کردن» (user icon, «رمز ۴ رقمی [+ اثر انگشت]»), the biometric switch (bloom, only when the device supports it) and «قفل خودکار» (clock icon, «بعد از ۱ دقیقه بیرون رفتن» + bloom's delay tabs). New «پنهان ماندن» group: «اعلان‌های محرمانه» switch (board copy; = B-N1-11 `neutral_copy`, synced with `/profile/notifications`) and «مخفی در صفحه برنامه‌های اخیر» (= bloom's hide-preview blur; sub-line promises only the blur — the web cannot block screenshots). «داده‌های من»: «گرفتن نسخه از همه داده‌ها» / «فایل قابل خواندن (JSON یا PDF)», «حذف کامل حساب و داده‌ها» is now a row (trash icon, danger tone, «غیرقابل بازگشت») instead of the old text button, plus the shield promise card at the end. Dropped: «آیکون و نام اپ» (`nbl_Priv_Icon`, native only — DECISIONS #5). Kept from bloom between the board's groups: consents, «دسترسی دیگران», doctor report links, backup row. Shot with the lock off (with it on, the gate shows the lock screen first — the method / auto-lock rows are covered by typecheck + the same `ListRow` primitive). |
| `nbl_Priv_Lock` | lock overlay (any route) | [lock](priv/CB-PRIV-01/fa_home.lock.light.png) | [lock](priv/CB-PRIV-01/fa_home.lock.dark.png) | ✔ | bloom's lock screen (icon, title, dots, 3×4 keypad, erase, «رمز را فراموش کرده‌ای؟») unchanged; new «کارت اضطراری · بدون نیاز به رمز» pill (danger-soft) under it, only when the card's `show_on_lock_screen` is on. Deliberate: bloom's title «ریتمی قفل است» + subtitle kept over the board's single «رمز را وارد کن». |
| `nbl_Rec_Emergency` | lock → «کارت اضطراری» | [card](priv/CB-PRIV-01/fa_home.card.light.png) | [card](priv/CB-PRIV-01/fa_home.card.dark.png) | ✔ (lock-screen view) | Read-only card inside the lock overlay. The app's screens stay unmounted, and after the security fixes only `GET /health-record/emergency-card/lock` (D-73) is called — the owner view never is. Layout: back header «کارت اضطراری», bordered card with the **first name** + blood-type pill, rows حساسیت / بیماری (chronic only) / داروی دائمی / [وضعیت باردار · N هفته when shared] / تماس اضطراری, then the shield note «فقط همین اطلاعات… بقیه پرونده قفل می‌ماند». The contact row's visible text is «تماس با {name or relation}» (or «تماس»); the number appears only in the `tel:` href. Deliberate: the board's «بیمه» row is not on the lock-screen card — insurance is owner-view only (the `owner` variant of `EmergencyCardView` still renders it for CB-REC-05). Not here (CB-REC-05): the owner's edit screen with the two switches and the QR / public link. |

### Security fixes (audit of 17c93ee6)

- **MEDIUM-1 + LOW-1:** the lock screen no longer reads the owner endpoint. A new owner-scoped
  `GET /health-record/emergency-card/lock` (D-73, `no-store`) answers `{enabled:false}` without reading any health
  data while the flag is off, else `{enabled:true, card}` with the minimal public card (no full name, gyn conditions,
  onboarding medications or insurance). Frontend: `useLockEmergencyCard` + `EmergencyCardView variant="lock"`; the owner
  card is never fetched from the lock path. Covered by an int test, a contract case (`emergency/emergency_lock`,
  Go-recorded) and OpenAPI.
- **MEDIUM-2:** no silent fallback. A discreet owner's invite SMS goes out only with
  `KAVENEGAR_TEMPLATE_COMPANION_INVITE_NEUTRAL`; when that is unset, `Gateway.SendInvite` returns `ErrNotSent`, so the
  answer is `sms_sent: false` (the existing not-sent path) and the owner shares the code herself.
  `notifications.SMSTemplate` now returns `(template, ok)`. Both env lines are now passed through in production
  `docker-compose.yml`, empty by default.
- **LOW-2:** the contact label is the contact's name or relation only; the raw number appears only in `tel:`.
- Re-shot: `fa_home.card.{light,dark}.png`.

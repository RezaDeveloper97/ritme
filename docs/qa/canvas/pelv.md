# PELV — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png`. Screens: 390 px, fa, headless Chrome over CDP against the worktree
Go API :8341 + Next dev :3127 (DB `pelv02_dev`, a copy of the ritme_dev dump — ritme_dev not migrated). Test user
**`09120003402`** (onboarding نیلا / female / cycle; program started 16 days ago → week 3 · level 2; sessions on the
4 days before today → streak 4; today's diary leak = cough, night voids 2). Colours checked against tokens, not board hex.

## CB-PELV-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Pelvic_Plan` | `/programs/pelvic` | [plan](pelv/CB-PELV-02/fa_pelvic.light.png) | [plan](pelv/CB-PELV-02/fa_pelvic.dark.png) | ✔ | Back header «کف لگن و مثانه» (→ `/programs`); plan card: «برنامه ۸ هفته‌ای», Lalezar «هفته ۳ · سطح ۲», streak pill (tone data, only when > 0), WeekDots in the locale's week (Jalali Sat→Fri = the API week; Gregorian Sun→Sat), today outlined, solid CTA «شروع تمرین امروز · ۵ دقیقه» → Kegel trainer («یک دور دیگر» once today is done); «دفترچه مثانه امروز» card: leak chips (single answer, tap again clears; period tint), divider, night-waking stepper; «عفونت ادراری» chips (multi, danger tint) + the `pelvic_alerts/uti_warning` line, promoted to a danger note (`role=alert`) when the API says `uti_alert`; suitability `InfoNote` from `pelvic_alerts/program_suitability`; bottom nav with «خدمات» active. Every diary change is a PUT of that key only. Addition: no program yet → join card «شروع برنامه» (POST `/pelvic/program`). |
| `nbl_Pelvic_Kegel` | `/programs/pelvic/kegel` | pending | pending | — | Not captured: the local Docker daemon (MariaDB for the API) stopped responding after the plan shots. Built per board: full-screen (no nav), header «تمرین کف لگن» + «سطح ۲ · ست ۱ از ۳», glow CountdownRing (hold = brand «منقبض کن», relax = data «رها کن», big seconds left, «از ۵ ثانیه»), rep bar (filled when a squeeze ends), «تکرار ۷ از ۱۰ · بعد ۵ ثانیه رها کن», round «شروع» / «توقف», instruction `InfoNote` (level body from catalog). Finish → POST `/pelvic/sessions` (all sets); stop → saved only after ≥ 1 whole set. Wake Lock held while running (re-taken on tab return, released on stop/unmount); `prefers-reduced-motion`: ring ticks once a second and the hold/relax scale is off. Capture in CB-PELV-03. |
| `services` / `/programs` hub | — | — | — | ✔ (code) | The `pelvic_floor` services tile with no admin href now links `/programs/pelvic` (was «به‌زودی»); the programs hub lists a «کف لگن و مثانه» card after the condition programs. |

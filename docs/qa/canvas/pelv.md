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

## CB-PELV-03 — epic QA

Captured against the worktree Go API :8381 + Next dev :3131 (DB `pelv03_dev`, a fresh copy of the dev dump, auto-migrated; dropped after), 390 px, fa, headless Chrome over CDP, light + dark via the theme attribute. Journey as a brand-new user (test mobile `09120003403`, onboarding done, no program): join → plan → bladder diary (leak «با سرفه یا عطسه», night waking 2) + UTI chips (سوزش + تکرر) → Kegel session (level 1 = 3 s hold / 3 s relax, 3 sets × 10 reps) start → reps → ran to the end → finished → back to plan → «یک دور دیگر» → start → stop before a whole set. Shots in `pelv/CB-PELV-03/`. Board renders: `boards/nbl_Pelvic_Plan.png`, `boards/nbl_Pelvic_Kegel.png` (boards show level 2 / 5 s; ours is level 1 on week 1, same layout).

| Board | State / route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Pelvic_Plan` | no program → join card (`/programs/pelvic`) | [join](pelv/CB-PELV-03/fa_pelvic_join.light.png) | [join](pelv/CB-PELV-03/fa_pelvic_join.dark.png) | ✔ | Addition over the board (board assumes a program): join card + diary/UTI visible below. |
| `nbl_Pelvic_Plan` | after join: week 1 · level 1 | [plan](pelv/CB-PELV-03/fa_pelvic_plan_fresh.light.png) | [plan](pelv/CB-PELV-03/fa_pelvic_plan_fresh.dark.png) | ✔ | Matches board; today outlined, no streak pill (streak 0). |
| `nbl_Pelvic_Plan` | diary + UTI logged | [logged](pelv/CB-PELV-03/fa_pelvic_plan_logged.light.png) | [logged](pelv/CB-PELV-03/fa_pelvic_plan_logged.dark.png) | ✔ | Leak chip rose tint, night stepper 2, UTI chips danger tint with check; warning line. |
| `nbl_Pelvic_Plan` | after a finished session | [after](pelv/CB-PELV-03/fa_pelvic_plan_after.light.png) | [after](pelv/CB-PELV-03/fa_pelvic_plan_after.dark.png) | ✔ | Streak «۱ روز پشت سر هم», «تمرین امروز انجام شد», CTA «یک دور دیگر». |
| `nbl_Pelvic_Kegel` | ready (`/programs/pelvic/kegel`) | [ready](pelv/CB-PELV-03/fa_kegel_ready.light.png) | [ready](pelv/CB-PELV-03/fa_kegel_ready.dark.png) | ✔ | No nav; ring «آماده‌ای؟», «۳ ست · هر ست ۱۰ تکرار», «شروع». |
| `nbl_Pelvic_Kegel` | running · hold | [hold](pelv/CB-PELV-03/fa_kegel_running_hold.light.png) | [hold](pelv/CB-PELV-03/fa_kegel_running_hold.dark.png) | ✔ | Brand ring «منقبض کن», seconds left, «بعد ۳ ثانیه رها کن», «توقف». |
| `nbl_Pelvic_Kegel` | running · relax | [relax](pelv/CB-PELV-03/fa_kegel_running_relax.light.png) | [relax](pelv/CB-PELV-03/fa_kegel_running_relax.dark.png) | ✔ | Turquoise data-accent ring «رها کن», «تکرار ۱ از ۱۰ · استراحت»; rep bar fills from the start side after a squeeze. |
| `nbl_Pelvic_Kegel` | finished | [done](pelv/CB-PELV-03/fa_kegel_finished.light.png) | [done](pelv/CB-PELV-03/fa_kegel_finished.dark.png) | ✔ | Check ring, «۳ ست کامل شد», «آفرین، تمرین امروز تمام شد», «بازگشت به برنامه»; session saved (plan streak/CTA above confirm). |
| `nbl_Pelvic_Kegel` | stopped before a whole set | [stop](pelv/CB-PELV-03/fa_kegel_stopped.light.png) | [stop](pelv/CB-PELV-03/fa_kegel_stopped.dark.png) | ✔ | «تمرین متوقف شد · ۰ ست کامل شد», nothing saved (by design). |

Findings: no open ✘. Small accepted deltas, no code change: the header button on the Kegel screen is a back arrow (board shows an up/close arrow glyph) and routes to the plan; the board is captured at level 2 / 5 s while a new user starts at level 1 / 3 s. Environment note: the shared ritme-go-test MariaDB was recreated by another session mid-run (grants/DB lost) — had to reload the DB copy once.

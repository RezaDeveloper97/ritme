# COND — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png`. Screens: 390 px, fa (plus one en), headless Chrome over CDP against
the worktree Go API :8301 + Next dev :3123 (`ritme_dev`, goose 00047, not migrated). Test user **`09120002302`**
(created for CB-COND-02: onboarding name نیلا / female / goal cycle; enrolled in `endo` via the API; heavy flow logged
on 2026-10-03 and 2026-10-04 so `GET /messages/nudges` returns `heavy_bleeding`). Colours are checked against the token
map (`docs/canvas-build/README.md` §4), not the board hex.

## CB-COND-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Hub` | `/programs` | [hub](cond/CB-COND-02/fa_programs.light.png) · [join sheet](cond/CB-COND-02/fa_programs.sheet-join.light.png) · [en](cond/CB-COND-02/en_programs.light.png) | [hub](cond/CB-COND-02/fa_programs.dark.png) · [active sheet](cond/CB-COND-02/fa_programs.sheet-active.dark.png) | ✔ | Same order: back header «برنامه‌های مراقبت» (→ `/services`), 24/800 «برای شرایطی که داری» + lead, four program cards in API order (endo / PMDD / heavy bleeding / PCOS) with the tone disc (flame·period, brain·brand, drop·period, sprout·data), title + subtitle from catalog `condition_programs`, the «ثبت می‌کنی: …» line from meta `logs`; an enrolled card has the `--period` 1.5px ring and the «فعال» pill (tone data), the rest a chevron; «این برنامه‌ها تشخیص نمی‌دهند…» `InfoNote` from `condition_alerts/not_a_diagnosis`; bottom nav with «خدمات» active (`/programs` added to the nav hubs). Additions (not on the board): this cycle's nudge (`GET /messages/nudges`) as a soft period-tint card above the list with its in-app action link; a card opens a half sheet — not enrolled: «شروع برنامه» (POST `/conditions/enrolments`); enrolled: «فعال از <Jalali date>», open the diary (`/programs/pain`, `/programs/pmdd`, `/programs/bleeding`; PCOS → `/analysis/cycle`) and «ترک برنامه» (DELETE, entries kept). Verified through the UI: join PMDD → card turns active, sheet links `/fa/programs/pmdd`; leave → inactive again. Services: the three condition tiles (`pain_endometriosis`, `pmdd`, `heavy_bleeding`) that had no admin href now link `/programs` ([services](cond/CB-COND-02/fa_services.programs.light.png)); pelvic floor stays «به‌زودی», contraception keeps its href. |

## CB-COND-05

Worktree Go API :8331 + Next dev :3126 against `cond05_dev` (a copy of the `ritme_dev` dump, goose 00047, not migrated —
the test stack had been reset). Test users **`09120002305`** (onboarding نیلا / female / cycle; enrolled in
`heavy_bleeding`; PBAC yesterday light 1 + heavy 2 = 41, today light 1 · medium 2 · heavy 3 · clots none = 71 → period
day 2, score 112 — the board's numbers) and **`09120002306`** (same onboarding, not enrolled).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Bleed` | `/programs/bleeding` | [chart](cond/CB-COND-05/fa_bleeding.light.png) · [en](cond/CB-COND-05/en_bleeding.light.png) · [not enrolled](cond/CB-COND-05/fa_bleeding.join.light.png) | [chart](cond/CB-COND-05/fa_bleeding.dark.png) · [edited](cond/CB-COND-05/fa_bleeding.edited.dark.png) | ✔ | Same order: back header «جدول خونریزی» + «روز ۲ پریود» (`period.day`; «امروز» without a period) → `/programs`; «نوارهای امروز» + hint; one card with three rows — pad pictogram (CSS, `--period` fill at 26 / 55 / 100 %), soak label, «هر عدد ۱ / ۵ / ۲۰ امتیاز», `NumberStepper` 0–50; «لخته» card with نداشتم / کوچک (اندازه سکه) / بزرگ chips (multi style, tone period, one at a time; tapping the active one clears it) and the «نشت از نوار به لباس» switch; «امتیاز امروز» / «امتیاز این پریود» tiles in Lalezar `--period-deep`; the ≥ `alert_threshold` card (`--period-soft` + `--period` border, warning disc) with the admin `condition_alerts/pbac_over_100` copy, own fa/en fallback while editing; sticky «ذخیره» footer, no bottom nav. The tiles and the alert recompute live while editing (period = saved period score − saved day score + live day score; [edited](cond/CB-COND-05/fa_bleeding.edited.dark.png): +1 light pad, large clot, flooding → ۸۲ / ۱۲۳). Save is enabled only when dirty; `clots` is sent only when changed (the API gates it on the mode's clot param). Verified through the UI: + then «ذخیره» → PUT saved (GET shows light 2, score 72) and the screen returns to `/fa/programs` (the board's flow). Addition: not enrolled → a join card («شروع برنامه», POST `/conditions/enrolments`) since PUT needs the enrolment. The board shows «کوچک» selected with a total of 71, which doesn't add up (small clot = +1); the seed uses «نداشتم». |

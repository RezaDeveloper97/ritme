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

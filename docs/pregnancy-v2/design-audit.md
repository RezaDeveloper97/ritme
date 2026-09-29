# Pregnancy v2 (M7): design-fidelity audit

Date: 2026-09-29. Branch `stage`. The audit changed no app code; the fixes came in T-M7-18 and T-M7-20 (see Resolution).

**Design:** `docs/design/pregnancy-v2/` has 6 `.dc.html` artboards (Setup, Main, Week, Log, Calendar, Alerts), light only
(there is no dark artboard for this module, see README). The artboards need the design-canvas runtime (`support.js`,
not in the repo), so a small local stand-in (DCLogic + `sc-for`/`sc-if` + `{{…}}` bindings, initial state from the query
string) rendered them in headless Chrome (CDP, port 9274, private profile) at 390 px wide, full artboard height, @2x.
Interactive states were rendered from each artboard's own `renderVals()` script: Setup steps 0–3 (plus the ultrasound
source), Week tabs 0–2, Log default and `saved`.

**Implementation:** the current code on `stage` ran locally. backend-go on `:8024` (`:8020` and `:8022` were busy), scratch
DB `ritme_m7audit` (goose v6) on the docker test stack, `REDIS_PREFIX=ritme-m7audit:`, `SMS_PROVIDER=log`. Next dev ran
from a scratch copy of `frontend/` on `:3024` with `NEXT_PUBLIC_API_BASE_URL` as an env var. `.env.local` was not touched.
No staging or production host was contacted.

The data was set up to match the artboards, shifted to the real today (2026-09-29): LMP 2026-08-01, so today is
8 w + 3 d, the same age as the artboard. Last weight 62.1 kg a week ago; vomiting on the 4 previous days (2 severe);
today mood «معمولی», تهوع متوسط + خستگی شدید, 5 glasses; week tasks 2 of 3 done; first visit done, NT scan booked
18 days ahead at 10:30. Every screen was captured full page (inner scroll unrolled) at 390 px @2x, locale fa, light
and dark.

**Colour rule:** the design wins on layout and content. `frontend/CLAUDE.md` §10.2 wins on colour. Differences caused by
that rule are marked *palette* and are not counted. Dark mode has no artboard, so the dark captures were checked
against the light artboard for layout and against §10.3 for contrast.

## Summary

| Severity | Count |
|---|---|
| High | 4 |
| Med | 12 |
| Low | 21 |
| **Total** | **37** |

Not counted: *palette* items (P1–P3) and the notes at the end.

Top issues:
1. **B1** Today carousel shows the trimester line twice; the age headline «۸ هفته و ۳ روز» is missing.
2. **D1** Log «ذخیره شد» in dark mode is white text on a light mint fill (about 1.5:1 contrast).
3. **F1** Alerts puts a gradient CTA on every card (6 on one screen). The design uses one solid primary on the
   follow-up card, a text link on the suggestion and no action on the info card.
4. **A1** Setup welcome has no illustration and no benefits list. The seeded admin copy `pregnancy_setup/welcome` is
   not used at all.

The paired screenshots are in `screenshots/audit/`, 585 px wide and compressed with pngquant. For each screen there is
`<screen>-design.png`, `<screen>-light-impl.png` and `<screen>-dark-impl.png`. The screens are `setup-welcome`,
`setup-dating`, `setup-dating-us`, `setup-history`, `setup-result`, `today`, `week-baby`, `week-body`, `week-tasks`,
`log`, `log-saved`, `calendar` and `alerts`.

## Resolution (T-M7-18, 2026-09-29)

Every row below has a Resolution. 34 fixed, 1 partly fixed, 2 won't fix (T-M7-20 finished A1, E2, F4 and the
`lint:dark` part of D1):

| Severity | Fixed | Partly | Won't fix |
|---|---|---|---|
| High (4) | 4 (A1, B1, D1, F1) | 0 | 0 |
| Med (12) | 11 | 0 | 1 (B2) |
| Low (21) | 19 | 1 (A9) | 1 (C3) |

The "partly" rows are blocked on files outside T-M7-18's paths: a read route for the Setup copy
(`internal/http/routes_pregnancy_v2.go`), the admin message registry plus a seed migration (calendar note, the
`week_entered` wording), and `frontend/scripts/check-dark-mode.mjs` (D1's lint pair).

API changes (Go only, OpenAPI updated): `carousel[].title` is the age; `dating-preview` adds `copy` (the
`pregnancy_setup/result` texts) and its `range_label` drops the year; calendar visits add `age_label` and
`source_note` leads with the care-plan caveat; v2 alerts add `fact_date`, and `date_label` follows it.

Known items: **9d** is F2 (fixed). **9i** was already fixed in T-M7-17 (fa «،» beside digits in the PDF). **9b** (return
to the calendar after saving a care-plan booking) is not fixed: the redirect lives in
`screens/reminder-appointment-form`, outside this task's paths. **T-M3-10 L-6:** «یادآورهای امروز» now sits right under
the Today hero, and its appointment row is hidden when the «ویزیت بعدی» card shows the same appointment.

The `*-impl` screenshots were re-taken after the fixes: backend-go on `:8032`, scratch DB `ritme_m718` (goose v7),
`REDIS_DB=7` / `REDIS_PREFIX=ritme-m718:`, Next dev from a scratch copy of `frontend/` on `:3032`, headless Chrome on
`:9282`, the same data as the audit (LMP 2026-08-01). The data raised no `weight_missing_week` alert (the weight was
logged in week 8 and the rule waits for day 6 of week 9), so the suggestion card is not in the capture.

## Deviations

### A. Setup: `Setup.dc.html` (`setup-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix | Resolution |
|---|---|---|---|---|---|
| A1 | **high** | Welcome: an illustration (fetus in a dashed ring), then title and body, then a white card with 3 green-check benefits («ابزارهای بارداری همیشه رایگان می‌مونن», «داده‌های چرخه‌ات حفظ می‌شه», «هر وقت خواستی می‌تونی به حالت چرخه برگردی») | `frontend/src/screens/pregnancy-onboarding/ui/PregnancyOnboardingPage.tsx:79-98` | Only a 48 px outline heart, title and body. There is no benefits card. The title and body come from the i18n bundle, and the admin-defined `pregnancy_setup/welcome` row (`benefits[]`, `primary`, `secondary`) is never read, so an admin edit changes nothing. | Render the illustration component, and read the welcome, dating, history and result copy from `pregnancy_setup` (README «What the admin defines»), with the bundle as fallback. Show the benefits as a check list card. | **partly fixed** — welcome now has the `WelcomePregnancy` illustration and a white card with the 3 green-check benefits (bundle `setup.benefits.*`, same text as the seed); «فعلاً نه» is a text link. The result step reads the admin `pregnancy_setup/result` copy (`lead`, `suffix`, `due_label`, `confidence`, `primary`, `secondary`), which `dating-preview` now returns as `copy`. **Fixed (T-M7-20):** `GET /pregnancy/v2/setup-copy` returns the admin `pregnancy_setup` welcome / dating / source / history / result texts in the request locale (default-language row as fallback); the welcome (title, body, benefits, buttons), dating and history intros, source labels + hints, history disclaimer and skip read it, the bundle only as fallback. OpenAPI, contract group `pregnancy-v2`, int test `TestSetupCopy`. |
| A2 | med | All steps sit on the lavender canvas `#F2ECFF` | `frontend/src/app/globals.css:1851` (`.onb-page { background: var(--surface) }`) | Setup (and the rest of onboarding) is on white. §10.2 also names the canvas for "onboarding & long-read screens". | Use `var(--page)` for `.onb-page`, and keep the cards on `--surface`. Check the signup onboarding screens too, because they share the class. | **fixed** — Setup uses `.pon-page` (`--page`), cards stay on `--surface`. The signup onboarding screens (shared `.onb-page`) are outside this task's screens and stay white. |
| A3 | med | Result: the illustration, then a centred hero «بر اساس داده‌های فعلی، احتمالاً» / **«۸ هفته و ۳ روز»** in brand / «باردار هستی» at 28 px, then a due-date card with a teal «دقت تخمین: متوسط» pill and the range plus basis in one paragraph | `frontend/src/screens/pregnancy-onboarding/ui/SetupSteps.tsx:205-241` | The result is two plain cards with the age at card-title size. Confidence is a text line («دقت تخمین: متوسط · حدود ±۳ روز»). There is no illustration, and the step is visually weaker than the welcome. | Build the hero (illustration + 3-line headline, age in `--brand`). Show confidence as a `--data-soft`/`--data-deep` chip (it is algorithmic data, so turquoise fits §10.2). | **fixed** — `ResultBaby` illustration, 3-line hero with the age in `--brand` (28 px), due-date card with a turquoise «دقت تخمین: …» chip (`--preg-teal-ink` on `--data-soft`: `--data-deep` on that tint is only 2.7:1) and range + basis in one paragraph. |
| A4 | med | Ultrasound source: a compact date field («۲۵ شهریور ۱۴۰۵») and two number fields «هفته» / «روز» side by side | `SetupSteps.tsx:72-100` | A full inline calendar, an empty full-width week input and seven day chips. The step is 1002 px tall instead of fitting on one screen. | Use a date field that opens the calendar in a sheet, and put week and day in two inputs on one row. | **fixed** — the ultrasound date is a field that opens the calendar in a half sheet; week and day are two inputs on one row, so the step fits one screen. The inputs are the shared `NumberField` (`type=number`), which shows Latin digits. |
| A5 | low | Dating (LMP): one calendar card with the month title and a «۳ مرداد» chip for the selected date | `SetupSteps.tsx:66-69`, `frontend/src/shared/ui/CalendarPicker.tsx:70` | A `PgCard` with a title wraps the `.card.jcal`, so there are two nested borders. The selected date is shown only as the filled cell. | Drop the outer `PgCard` (or the inner card border) and add the selected-date chip to the calendar header. | **fixed** — the outer `PgCard` is gone; the selected date is a chip («۱۰ مرداد») on a label row right above the calendar (`CalendarPicker` is shared UI, so the chip is not inside its header). |
| A6 | low | Source segmented control without a label; two-line options fit inside the pill | `SetupSteps.tsx:46-62` | An extra label «مبنای محاسبهٔ سن بارداری» is shown. The two-line options («اولین روز آخرین قاعدگی», «خودم هفته رو وارد می‌کنم») touch the bottom edge of the active pill. | Remove the label (the title already says it) and give `.seg` buttons a `min-height` with vertical padding. | **fixed** — visible label removed (kept as the radiogroup's `aria-label`); `.seg.pon-seg` gives the options a 44 px min-height with vertical padding. |
| A7 | low | History: miscarriage and high-risk as rows in one card; conditions with «هیچ‌کدام» **first**; blood group and Rh as **two selects side by side**; the disclaimer as plain text | `SetupSteps.tsx:137-190` | Toggles instead of checkboxes (fine). «هیچ‌کدام» is last. Blood group and Rh are chip rows inside a third card, so the step is 1.3 screens long. | Put «هیچ‌کدام» first, and show blood group and Rh as two compact selects (or keep the chips without the card). The disclaimer copy change is deliberate (open point 1). | **fixed** — «هیچ‌کدام» first (`noneFirst`), blood group and Rh are two selects side by side, no third card. Toggles kept; the disclaimer copy change is deliberate (open point 1). |
| A8 | low | Header back is a round white button; the secondary actions («فعلاً نه», «فعلاً رد می‌شم», «مبنای محاسبه رو عوض می‌کنم») are plain text links | `PregnancyOnboardingPage.tsx:93,107,160,170` | Back is a bare chevron. The secondary actions are full-width outlined `btn-ghost` buttons, so they compete with the primary CTA. | Use the round `iconbtn` style on a surface, and a text-link button for the secondary actions. | **fixed** — back is the round `.pg2-roundbtn`; «فعلاً نه», «فعلاً رد می‌شم» and «مبنای محاسبه رو عوض می‌کنم» are `.pon-textbtn` text links. |
| A9 | low | Result and Today: «بازهٔ معمول تولد: ۲۸ فروردین تا ۲۵ اردیبهشت», which is due date ±14 days | `backend-go/internal/pregnancy/v2/dating.go:89-91` | The range is ±(14 + uncertainty), so ±17 days for LMP. The labels also carry the year («۱ اردیبهشت ۱۴۰۶ تا ۴ خرداد ۱۴۰۶»). | Confirm the rule with the clinician (goes with 9j). Leave out the year when it is the current or next year and there is no ambiguity. | **partly fixed** — the range labels drop the year (server `range_label` uses day + month; the due date keeps it). The ±(14 + uncertainty) rule is clinical and stays for the sign-off (T-M7-15, 9j). |

### B. Today: `Main.dc.html` (`today-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix | Resolution |
|---|---|---|---|---|---|
| B1 | **high** | Carousel: a small line «سه‌ماههٔ اول · هفتهٔ ۸ از ۴۰», then a 30 px headline **«۸ هفته و ۳ روز»** (the previous and next slides use «۷ هفته» / «۹ هفته») | `frontend/src/widgets/pregnancy-week-carousel/ui/PregnancyWeekCarousel.tsx:76-85`, `backend-go/internal/pregnancy/v2/service.go:130` | The API's `carousel[].title` is `WeekLabel(w)`, which is the same trimester·week string as the eyebrow. So the card prints «سه‌ماههٔ اول · هفتهٔ ۹ از ۴۰» twice, and the second one wraps to 2 lines at 22 px. The user's age in weeks and days, which is the main number on the screen, is not shown anywhere on Today. | Make `title` the age: «N هفته و M روز» for the current slide and «N هفته» for the others (server side, localised digits), or build it on the client from `progress`. | **fixed** — the API's `carousel[].title` is now the age: «۸ هفته و ۳ روز» on the current slide, «۷ هفته» / «۹ هفته» on the others (completed weeks at the start of that week, localized digits). The headline is 30 px and never wraps. |
| B2 | med | 8 w + 3 d is «هفتهٔ ۸» everywhere: carousel, due card «هفتهٔ ۸ از ۴۰», Week header, «درباره هفتهٔ ۸ بیشتر بخون», alert «وارد هفتهٔ ۸ شدی» | `backend-go/internal/pregnancy/v2/service.go` (`progress.week`, `WeekLabel`) | The app counts the current week as completed weeks + 1, so the same age is «هفتهٔ ۹», and the week page, tip and tasks follow week 9. | Decide the convention with product and the clinician and apply it in one place. Everything already derives from `progress.week`. The artboard itself is not fully consistent (Calendar puts NT at «هفتهٔ ۱۱ و ۱ روز», 18 days after 8 w + 3 d). | **won't fix** — the rest of the app counts the current week as completed weeks + 1: `calc.CurrentWeek` (Laravel-compatible v1 contract), the fetal-movement thresholds (weeks 18 / 24), the message-engine milestones and every week-keyed content row (details, tips, tasks). The design's completed-weeks numbering does not match that convention, and the artboard is not consistent either (NT at «هفتهٔ ۱۱ و ۱ روز»). With B1 the screen now shows both: «۸ هفته و ۳ روز» (completed age) under «هفتهٔ ۹» (the week you are in). |
| B3 | med | A full-bleed gradient header (rounded bottom corners) holds the date pill «سه‌شنبه، ۳۱ شهریور», an alerts bell with a turquoise new-dot, the 7-day strip (past days tinted, today white, future days dashed) and the carousel | `frontend/src/screens/pregnancy/ui/PregnancyPage.tsx:60-80`, `PregnancyWeekCarousel.tsx:61` | The date is a plain heading, the strip is white tiles on the canvas with no dashed future days, and the carousel is an inset card. There is no bell in the header; alerts are reachable only through the quick-action tile. | Move the strip into the hero, add the bell (`--data` dot when `unread_alerts > 0`, §10.2 "new-notification dots") and use dashed tiles for future days. | **fixed** — full-bleed `.pg2-hero` (rounded bottom) with the date pill, the alerts bell (turquoise `--data` dot when `unread_alerts > 0`), the 7-day strip (past tinted, today white, future dashed) and the carousel. |
| B4 | med | Due date and 40-week progress are **one** card: date, days left + range, a thin bar, «هفتهٔ ۸ از ۴۰ / ۲۱٪ از مسیر رو اومدی», trimester segments. A pencil button at the start edits the basis | `PregnancyPage.tsx:97-150` | There are two cards (`DueDateCard`, `ProgressCard`). The calendar icon is decorative, so there is no way to change the basis from Today. | Merge the cards and make the start button link to `/pregnancy/setup` (the «مبنای محاسبه رو عوض می‌کنم» path). | **fixed** — one card: date, days left + range, 40-week bar, «هفتهٔ ۹ از ۴۰ / ۲۱٪», trimester segments; the pen button links to `/pregnancy/setup`. |
| B5 | med | Quick actions: a 2×2 grid, with the icon tile at the start and the label beside it. Tones: pink edit («ثبت علائم روزانه»), violet stethoscope, teal calendar («مرور هفته‌ها»), amber warning («هشدارها») | `PregnancyPage.tsx:159-192` | A 4-column row with the icon above the label, all tiles in brand tone, icons `plus`/`stetho`/`bookOpen`/`bell`. «ثبت علائم روزانه» wraps to two lines. | Use `grid-cols-2` with horizontal tiles and the design icons. Map the tones to the existing `pg2-tone-*` classes (teal only if it is kept as data, otherwise neutral). | **fixed** — 2 × 2 grid of horizontal tiles, icons pen / stethoscope / calendar / warning, tones pink / brand / neutral / warning (teal not used: browsing weeks is not data, §10.2). |
| B6 | low | Confidence chip «دقت تخمین: متوسط · حدود ±۵ روز» is a teal data chip with an info icon | `PregnancyWeekCarousel.tsx:101` | The chip is translucent white on the gradient. | Use `bg-(--data-soft) text-(--data-deep)` with the info icon. It is algorithmic data, so turquoise is correct under §10.2. The ±5 vs ±3 copy is 9j. | **fixed** — turquoise data chip with the info icon. The ±5 vs ±3 copy is 9j (clinician). |
| B7 | low | Tip card: the eyebrow «نکتهٔ هوشمند · این هفته» is in brand colour | `PregnancyPage.tsx:241` | The eyebrow is muted with a sparkle icon. The week-9 tip in the seed is formal register («را دنبال کنید»), while the app and the artboard are colloquial. | Use `text-(--brand)` for the eyebrow. Add the tone of the tip copy to the T-M7-15 content review (weeks other than 8 are generic). | **fixed** — eyebrow in `--brand` without the sparkle. The tip's register is content for the T-M7-15 review. |
| B8 | low | «مراقبت‌های این هفته»: square violet checkboxes | `frontend/src/widgets/pregnancy-care-checklist/ui/PregnancyCareChecklist.tsx:44-60` | Round checks filled with `--success` green. This is the reverse of known item 9e: the artboard uses square boxes on both screens. | Pick one style for Today and Week (the design uses a square brand box) and share it. | **fixed** — one square brand box (`PregnancyCheckBox`, `.pg2-check`) on Today and in the Week tasks tab. |
| B9 | low | Due-date disclaimer: a calm lavender box with a brand info icon | `PregnancyPage.tsx:328` (`pg2-warn`) | An amber warning box, which makes a routine note look like a warning. The same applies on Alerts (F5). | Use a `--surface-2`/lavender note style and keep `pg2-warn` for real warnings (the Week warning box). | **fixed** — calm `.pg2-note` (lavender tint, brand info icon); `pg2-warn` stays for real warnings. |
| B10 | low | Next-visit card has an end chevron, and the meta reads «شنبه · ۱۰:۳۰ · هفتهٔ ۱۱» | `PregnancyPage.tsx:196-222` | There is no chevron. The weekday comes from `new Intl.DateTimeFormat(…)` inside the screen (line ~200), which is date formatting outside `shared/lib/date` (§7). | Add the chevron, and move the weekday formatting to `shared/lib/date`. | **fixed** — the card links to the calendar with an end chevron; the weekday comes from the new `formatWeekday` in `shared/lib/date` (no `Intl` in the screen). |

### C. Week: `Week.dc.html` (`week-baby-*`, `week-body-*`, `week-tasks-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix | Resolution |
|---|---|---|---|---|---|
| C1 | med | Warning box starts with the bold lead «این موارد رو با پزشکت در میون بذار:» | `frontend/src/screens/pregnancy-week/ui/PregnancyWeekPage.tsx:184-188` | Only the stored `warning` text is shown. It starts with «خونریزی، درد شدید…», so the box has no instruction. The README says the lead line is UI copy. | Add a bundled `warningLead` string, bold, before `d.warning`. | **fixed** — bold bundled `week.warningLead` before the stored warning. |
| C2 | low | Stats «~۱.۶ سانتی‌متر», «~۱ گرم», «۱۵۰ تا ۱۷۰ ضربان در دقیقه» | `PregnancyWeekPage.tsx:163-172` | «۲.۳» uses an ASCII dot instead of «٫», the range is «۱۴۰-۱۷۰» with a hyphen in an LTR box, and there is no «~». The README says the UI adds «~». | Format with a helper that localises the decimal separator, turns `a-b` into «a تا b», and adds «~» for averages. | **fixed** — `parseWeekStat` + `formatDecimal`: «~۱٫۶», «۱۵۰ تا ۱۷۰», «کمتر از ۱»; no LTR box. |
| C3 | low | Baby tab: 3 highlights (hand/violet, heart/pink, face/teal), each with a short colloquial title and body | seed `backend-go/db/migrations/00005_pregnancy_v2.sql` | Only week 8 (the artboard week) has 3 highlights. Week 9 has one highlight («رشد جنین») with a long formal paragraph. This is content, not UI. | Add it to the T-M7-15 content review, and fill 2–3 highlights per week in admin. | **won't fix** — content, not UI; for the T-M7-15 review (2–3 highlights per week in admin). |
| C4 | low | Tasks tab: done tasks keep their normal ink (check only) | `frontend/src/screens/pregnancy-week/ui/WeekTabs.tsx:156` | Done tasks are struck through and muted. | Drop `line-through` here, or confirm with design. | **fixed** — Week tasks keep their normal ink; the check carries the state (Today keeps the strike-through, as its artboard does). |

### D. Log: `Log.dc.html` (`log-*`, `log-saved-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix | Resolution |
|---|---|---|---|---|---|
| D1 | **high** | «ذخیره شد»: white text on a saturated success green | `frontend/src/screens/pregnancy-log/ui/DayLogPage.tsx:421` (`bg-(--success)` + `text-(--on-accent)`) | In dark mode `--success` is lifted to `#7FE0A8` so it reads as *text* (`globals.css:324`). As a fill under white text that gives about 1.5:1, so the confirmation is almost unreadable (`log-saved-dark-impl.png`). This is the `--brand` / `--brand-fill` trap from §10.3. | Add a theme-stable `--success-fill` (both blocks, like `--brand-fill`) and use it here. Add the pair to `lint:dark`. | **fixed** — new theme-stable `--success-fill` (same value in both blocks, like `--brand-fill`) under «ذخیره شد». **Fixed (T-M7-20):** `lint:dark` checks `--on-accent` on `--success-fill` (floor 4.5:1). |
| D2 | low | Symptoms: a 3-column grid of equal buttons. Mood, symptoms and water are sections on the canvas; severities are grouped in a tinted panel | `DayLogPage.tsx` (mood/symptom sections) | Flow-wrapped chips of different widths, and each section sits in a white card. It is functionally the same. | Use `grid grid-cols-3` for the 9 toggles. The card wrapping can stay if it is the app convention. | **fixed** — the 9 toggles are a `grid-cols-3` of equal buttons; the white section cards stay (app convention). |
| D3 | low | Weight: a centred number field with the unit «کیلوگرم» outside it | `DayLogPage.tsx:335-352` | The field shows «کیلوگرم» as placeholder-style text inside the box, so the unit is not visible next to the typed value. | Put the unit label beside the input. | **fixed** — «کیلوگرم» sits beside the input, outside the field. |

### E. Calendar: `Calendar.dc.html` (`calendar-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix | Resolution |
|---|---|---|---|---|---|
| E1 | med | Care plan rows «هفتهٔ ۱۸ تا ۲۲ · حدود ۲۴ آذر», «هفتهٔ ۲۴ تا ۲۸ · حدود ۶ بهمن» | `frontend/src/screens/pregnancy-calendar/ui/PregnancyCalendarPage.tsx:476` | `CareRow` reads `item.date` only. The API sends `suggested_date` for `to_book` items (anomaly 2026-11-28, GTT 2027-01-09, Tdap 2027-01-30), so «حدود …» is never shown and the rows show only the week window. | Use `item.date ?? item.suggestedDate`. The `aroundDate` branch already exists. | **fixed** — rows use `date ?? suggested_date`, so unbooked items read «حدود ۷ آذر». |
| E2 | med | Source note: «زمان‌ها بر اساس برنامهٔ رایج مراقبت‌های بارداری‌اند و ممکنه پزشکت برنامهٔ متفاوتی بده. [منبع و تاریخ بازبینی]» | `backend-go/internal/pregnancy/v2/calendar` (`source_note` is null), `PregnancyCalendarPage.tsx:250` | The API returns `source_note: null`, and the screen shows «تاریخ‌ها بر اساس اولین روز آخرین قاعدگی محاسبه شده‌اند.» The caveat that the doctor's plan may differ, and the source, are lost (§11 "not medical advice", attributed sources). | Seed and return the design note (admin-editable, `pregnancy_setup` or the care-plan config), and keep the basis sentence as a second line if it is wanted. | **partly fixed** — `source_note` now leads with «زمان‌ها بر اساس برنامهٔ رایج مراقبت‌های بارداری‌اند و ممکنه پزشکت برنامهٔ متفاوتی بده.» and keeps the basis sentence after it (server copy, `pregnancy_calendar` lang file). **Fixed (T-M7-20):** admin-editable as `pregnancy_setup/calendar_note` (`plan_note`, `basis_lmp|ultrasound|manual`; registry item + goose `00008_pregnancy_v2_copy.sql` seed, fa + en); each part falls back to the lang text. Int test `TestCalendar_SourceNoteIsAdminEditable`. The source / review date still waits for the T-M7-15 sign-off. |
| E3 | low | On open, the selected day is the next visit (18 مهر) with «سونوگرافی NT · ساعت ۱۰:۳۰» | `PregnancyCalendarPage.tsx:57` | Today is selected, so the panel reads «برای این روز چیزی ثبت نشده…». | Default the selection to `next_visit.date` when it is in the shown month. | **fixed** — until the user picks a day, the selection is the next visit's day when it is in the shown month. |
| E4 | low | Next visit meta «شنبه · ساعت ۱۰:۳۰ · هفتهٔ ۱۱ و ۱ روز» and a second line «[نام پزشک] · [نام مرکز]» | `PregnancyCalendarPage.tsx:340-347` | «شنبه، ۲۵ مهر، ۱۰:۳۰، سه‌ماههٔ اول · هفتهٔ ۱۲ از ۴۰»: the date repeats the tile, and «،» and «·» are mixed in one line. | Show weekday · time · age at the visit (weeks + days), using one separator. | **fixed** — «شنبه، ساعت ۱۰:۳۰، ۱۱ هفته»: weekday, time and the age on the visit day (new `age_label`), one separator; the date stays on the tile. Doctor · place stays the second line. |
| E5 | low | Care-plan dates without the year («۲۵ شهریور · انجام شد») | `PregnancyCalendarPage.tsx:476-480` (`dateLabel` from the API) | «۲۵ شهریور ۱۴۰۵، انجام شد», so the rows get longer. | Use day + month for dates within ±12 months. | **fixed** — care-plan dates are day + month within a year of today. |
| E6 | low | «مسیریابی» is a dark ink button | `PregnancyCalendarPage.tsx` (next-visit actions) | In dark mode the `--ink` fill flips to a light-grey slab, which is the brightest element on the screen. | Use a neutral surface button (`btn-ghost`) or `--brand-fill` for the one primary action. | **fixed** — «مسیریابی» is a neutral surface button like its neighbour. |

### F. Alerts: `Alerts.dc.html` (`alerts-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix | Resolution |
|---|---|---|---|---|---|
| F1 | **high** | Actions per level: follow-up = **solid** «افزودن به یادداشت ویزیت» + outlined «دیدم، ممنون»; suggestion = text link «ثبت وزن ›»; info = no actions | `frontend/src/screens/pregnancy-alerts/ui/PregnancyAlertsPage.tsx:112` | The first action of every card is `btn-primary` (gradient) and every card also has a full-width «دیدم، ممنون». With 4 alerts that is 4 gradient buttons plus the bottom-nav FAB, which breaks the §10.2 "gradient is scarce" rule, and every level looks equally urgent. | Style by level: only `follow_up`/`urgent` get a primary button, `suggestion` gets a text link, and `info` gets no button (or ack only). Use a solid `--brand-fill` rather than the gradient. | **fixed** — actions by level: `follow_up` / `urgent` get one solid `--brand-fill` button (`.pg2-btn-solid`, not the gradient) plus outlined «دیدم، ممنون»; `suggestion` gets text links (no ack); `info` gets none. |
| F2 | med | Most important first: «ارزش پیگیری داره» (follow-up), then the suggestion, then info | `frontend/src/screens/pregnancy-alerts/model/alerts.ts:11-20` | The order is info → suggestion → follow-up → follow-up (ascending). This is known 9d, and it holds for every level, not just urgent. | Sort within the day by level (`urgent > follow_up > suggestion > info`), then by time. | **fixed** (9d) — within a day: urgent › follow_up › suggestion › info, then newest first. |
| F3 | med | The follow-up card has an amber border and a warning icon in the level pill. «چی دیدیم» and «چقدر مطمئنیم» are two tinted tiles side by side | `PregnancyAlertsPage.tsx:81-95` | Every card is the same flat card, and the two facts are inline `dt: dd` lines. | Add a level modifier (amber border for `follow_up`, danger for `urgent`) and render the facts as a 2-column tile grid. | **fixed** — amber border on `follow_up`, rose on `urgent`, warning icon in their level chip; «چی دیدیم» / «چقدر مطمئنیم» are two tinted tiles side by side (on those two levels, like the artboard). |
| F4 | low | Card time is the event date («دیروز», «۲۸ شهریور»); the info card text is «مبنای محاسبه هنوز آخرین قاعدگیه…» | `backend-go/internal/messages/pregnancyalerts/rules.go:121` + seed `pregnancy_alert/week_entered` | All cards say «امروز» (the `created_at` of the evaluation). `week_entered` says «هفتهٔ ۹ بارداری از امروز شروع شده» although week 9 started 3 days earlier (it fires on the first evaluation of the week). | Store the fact date (week start, streak end) and show it. Change the copy to «از {date}» or drop «امروز». | **partly fixed** — alerts carry a `fact_date` (week_entered: the week's first day; vomiting_streak: the streak's last day) and are dated and grouped by it («۴ مهر», «دیروز»). **Fixed (T-M7-20):** goose `00008_pregnancy_v2_copy.sql` rewrites the seeded `what_we_saw` to «هفتهٔ {week} بارداری شروع شده» / "Pregnancy week {week} has started", only where the row still holds the seed (admin edits kept; Down restores it with the same guard). Int test `TestPregnancyV2CopyMigration`. |
| F5 | low | Header has a settings gear at the start. The legend and the disclaimer are one soft-tint panel, and the disclaimer is plain text | `PregnancyAlertsPage.tsx:33,255` | There is no gear. The legend is a white card and the disclaimer is a separate amber box (same issue as B9). | Add the gear (link to reminder settings, the same target as «تنظیم یادآورها»), and merge the legend and disclaimer into one tint panel. | **fixed** — the header has a gear (opens the notifications sheet, the old «تنظیم یادآورها» target); legend and disclaimer are one `.pg2-note` panel. |

## Palette (not counted)

- P1. Setup steps 1–2 «ادامه» is a solid brand button in the design and the gradient in the app. The gradient CTA is the
  §10.2 primary role, so keep it.
- P2. The success green is `#22B07D` in the design and `--success` (`#0F7B6C` light) in the app. Token-driven; only D1's
  dark fill is a defect.
- P3. The bottom nav is a flat bar with a raised FAB in the design and the app-wide floating pill nav in the app. It is
  shared app chrome, not a module deviation.

## Notes

- **Extra in the app, not in this artboard:** «یادآورهای امروز» on Today (from the M3 `v13_Preg_Home` design),
  «ثبت‌های دیگر» (weekly checkup, fetal movement) on Log, and pager dots on the carousel. None of these is counted.
- **Reviewer line** on Week reads «در انتظار بازبینی متخصص» because every seeded `reviewed_at` is NULL. That is the
  intended state until the T-M7-15 sign-off.
- **Known items:** 9d is F2; 9e is the reverse of B8 (the design uses square boxes on both screens); 9j covers the ±5 vs
  ±3 copy in B6 and A9.
- **Data difference:** the app raised a 4th alert, `severe_symptom_count` («۳ علامت شدید در یک هفته»), because the
  artboard's own Log state (خستگی شدید) plus the 2 severe vomiting days reach the threshold. The artboard shows 3 alerts.
- The design renders were made with a local runtime stand-in, because the canvas `support.js` is not in the repo.
  Artboard fonts (Vazirmatn) loaded from Google Fonts.
- The Next.js dev indicator was removed from the captures.

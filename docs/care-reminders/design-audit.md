# Care reminders (M3) — design-fidelity audit vs `reminders-v13`

Date: 2026-09-29 · branch `stage` @ 0a1cf54 · auditor: Claude (headless Chrome, read-only — no app code changed).

## Method

- **Design:** the 12 artboards in [`docs/design/reminders-v13/`](../design/reminders-v13/) rendered in headless Chrome
  (390 px wide, DPR 1.5 → 585 px PNG, full artboard height).
- **Implementation:** local stack — scratch MariaDB `ritme_audit_m3` loaded from `contract/fixtures/dump.sql`,
  backend-go on :8020 (`RUN_MIGRATIONS=true` stamped goose 2–6), Next dev on :3000 with
  `NEXT_PUBLIC_API_BASE_URL` passed as an env var (`.env.local` untouched). Same viewport/DPR; tall screens were
  shot with the viewport grown to the scroll height (capped at 3244 px, so the long cycle home is cut off at the bottom).
- **Data matching the artboards:** personas `09900000015` (pregnant, week 27) and `09900000004` (cycle) got
  فولیک اسید ۴۰۰ mcg tablet 08:00 (taken today), ویتامین D capsule 21:00, آهن (فروس سولفات) tablet 13:00 on
  ش/د/چ/ج (switched off), an in-person NT ultrasound on 8 مهر 10:30 with دکتر احمدی · زنان و زایمان, location,
  remind 1 day before and 3 prep items (1 done), plus an online nutrition consult with خانم مهدوی on 20 مهر 18:00.
  `09900000012` has no reminders (empty states). Today = 7 مهر, so the NT visit is «فردا» (the artboard says
  «۸ روز دیگر»); the pregnancy week also differs (27 vs 11). Those are data differences, not deviations.
- Colour differences that come from the app's own tokens are **not** counted (frontend/CLAUDE.md §10: design wins
  on layout/content, palette rules win on colour) — e.g. the dark canvas/card being the app's `--page`/`--surface`
  rather than `#17112B`/`#221A3D`, or `--brand-fill` staying saturated in dark (§10.3), where the artboards use
  lavender `#B9A6FF` fills with dark text. Status bar, home indicator and the Next dev badge are ignored.

Screenshots: [`screenshots/audit/`](screenshots/audit/) — each pair is `<screen>-<theme>-design.png` next to
`<screen>-<theme>-impl*.png` (585 px wide, pngquant). The two `home-card-*-crop-impl-vs-design.png` files are
side-by-side crops (implementation left, design right).

| Screen | Design | Implementation |
|---|---|---|
| Home card «یادآورهای امروز» | `home-card-{light,dark}-design.png` | `…-impl-pregnancy.png`, `…-impl-cycle.png`, `…-impl-empty.png` |
| AddChooser sheet | `add-chooser-{light,dark}-design.png` | `add-chooser-{light,dark}-impl.png` |
| /reminders | `reminders-{light,dark}-design.png` | `reminders-…-impl.png`, `reminders-meds-…`, `reminders-appts-…`, `reminders-empty-…` |
| Add/edit medication | `medication-form-{light,dark}-design.png` | `…-impl-new.png`, `…-impl-edit.png` |
| Add/edit appointment | `appointment-form-{light,dark}-design.png` | `…-impl-new.png`, `…-impl-new-phone.png`, `…-impl-edit.png` |
| Appointment detail | `appointment-detail-{light,dark}-design.png` | `appointment-detail-{light,dark}-impl.png` |

## Summary

**27 deviations — 4 high, 12 medium, 11 low.** The home card, the AddChooser sheet and the /reminders list are
close to the artboards. Most of the drift is in the two forms and the appointment detail hero, which use the app's
generic form primitives (`.field`, `.seg`, outlined `.chip`, card headers) instead of the artboards' soft filled
fields, separate kind cards, time buttons and the large-number hero.

Top issues:
1. **Appointment form kind selector is broken** — the icon wraps above the label inside a 40 px pill, so it is
   clipped and sits in the top corner; the artboard has three separate stacked cards (H-1).
2. **Appointment detail hero lost its hierarchy** — no large time, small amber tile, plain white card; the design's
   hero is a tinted card with a big date tile and a big «۱۰:۳۰» (H-2).
3. **Medication form: wrong card title and missing label** — the first card is titled «شکل دارو» although it holds
   the name and dose; the form chips lose their label; the third card title repeats a row label (H-3).
4. **Medication form time slots** — no «ساعت نوبت‌ها» label and the time is a small outlined chip «۰۸:۰۰» instead
   of the design's slot row with a big «۸:۰۰» button (H-4).
5. The appointment's «با چه کسی» moves from the title into the meta line and the place disappears from list rows;
   the detail's «در خصوص» shows only the topic label, and the user's note is shown nowhere (M-5, M-10).

## Findings

Severity: **high** = wrong or broken structure a user notices at once · **med** = clearly different
element/hierarchy/copy · **low** = polish. «Impl» paths are relative to `frontend/src/`.

| # | Sev | Screen · design element (file) | Deviation | Implementation | Suggested fix |
|---|---|---|---|---|---|
| H-1 | high | AddAppointment · kind selector (`nbl/nbd_v13_AddAppointment`) | Design: three separate cards (≈112×78, radius 20, icon stacked above label, selected = soft brand fill + brand border). Impl reuses the /reminders tab track (`.rmd-tabs`, 40 px pills, selected = solid brand fill); the block-level `Icon` wraps above the label and is clipped at the top corner. | `screens/reminder-appointment-form/ui/AppointmentFormPage.tsx:209-221`; `app/globals.css:3251-3261` | Give the form its own kind-card class (flex column, centred, icon over label, `--brand-soft` + `--brand` border when on); keep `.rmd-tab` for the list tabs only. |
| H-2 | high | AppointmentDetail · hero (`*_v13_AppointmentDetail`) | Design: tinted hero card, kind pill (start) and week pill (end), large bordered date tile, eyebrow «سه‌شنبه · ساعت», **large Lalezar time «۱۰:۳۰»**, title under it. Impl: plain white card, small amber `rmd-date` tile, title first, time buried in an 11.5 px meta line, plus an extra topic line («سونوگرافی») repeating the title; week pill is an outlined brand chip. | `screens/reminder-appointment-detail/ui/AppointmentDetailPage.tsx:160-190` | Build a dedicated hero: soft brand/`--surface-2` gradient surface, big date tile, weekday+«ساعت» eyebrow, time in the display face at ~34 px, title below; drop the duplicate topic line; soft (not outlined) pills. |
| H-3 | high | AddMedication · card 1 (`*_v13_AddMedication`) | Design card 1 has no header: name → dose/unit → label «شکل دارو» → form chips. Impl gives the card a header «شکل دارو» (with pill icon) above the *name* field, and the form chips have no label of their own. Card 3 header «مدت مصرف» repeats the row label «مدت مصرف» right below it. | `screens/reminder-medication-form/ui/MedicationFormPage.tsx:67-79` (`Card`), `:254`, `:305`, `:419`, `:430` | Drop the card headers (the artboard has none) or at least retitle; put the «شکل دارو» label above the chips at 305. |
| H-4 | high | AddMedication · «ساعت نوبت‌ها» | Design: section label «ساعت نوبت‌ها», then one soft row per slot («نوبت ۱» · «صبح» · white button with a large brand «۸:۰۰»). Impl: no section label; each slot is a plain line «نوبت ۱، صبح» with a small outlined `chip on` showing «۰۸:۰۰» (leading zero). | `MedicationFormPage.tsx:338-356` | Add the label; render the slot as a `--surface-2` row with a `--surface` time button (display face, ~20 px, `--brand`); format the time without a leading zero (reuse `slotClock`). |
| M-1 | med | AddMedication + AddAppointment · text fields | Design fields are filled soft surfaces (`#F1EDFB`/dark `rgba(255,255,255,.06)`), no border, radius 16, with a leading icon (pill, person, note, pin). Impl uses outlined white `.field`, no icons except the name field. | `app/globals.css:478` (`.field`); `AppointmentFormPage.tsx:224-277, 326-337`; `MedicationFormPage.tsx:255-302, 447-457` | Add a filled `field` variant (`--surface-2`, no border) for the care forms and pass the design's icons. |
| M-2 | med | AddMedication · «چند بار در روز؟» | Design: four separate outlined chips, selected = soft brand fill + brand border. Impl: grey `.seg` segmented track with a white selected segment. | `MedicationFormPage.tsx:323-336`; `globals.css:507` | Use the `chip` / `chip on` pattern used for the form chips. |
| M-3 | med | AddMedication · «مقدار در هر نوبت» | Design: label above a soft field with a large «۱» + unit and two white square − / + buttons at the end. Impl: one inline row, small 32 px round buttons around «۱ قرص». | `MedicationFormPage.tsx:389-416` | Match the stepper field: soft container, number in the display face, 44 px square buttons (also fixes the touch-target size). |
| M-4 | med | AddMedication · «شروع از» / «مدت مصرف» rows | Design: value as a subtitle under the label («امروز · ۳۰ شهریور», «تا پایان بارداری»), a round calendar icon tile for the start date, and a small soft-brand «تغییر» pill. Impl: the start date is an outlined pill on the far side, «تغییر» is a large neutral outlined chip. | `MedicationFormPage.tsx:419-437` | Move the value under the label, add the icon tile, style «تغییر» as a soft-brand small pill. |
| M-5 | med | Reminders + home card · appointment row copy | Design title «{title} · {with}», meta «{time} · {kind} · {place}». Impl list row: title only, meta shows *with* instead of place (`detail = with ‖ location`), so the place never appears. Home card: the full location wraps onto an extra line. | `screens/reminders/model/view.ts:189-201`; `screens/reminders/ui/AppointmentSection.tsx:15-22, 79-80`; `widgets/today-reminders/ui/TodayRemindersCard.tsx:104-113` | List: use `home.appointmentTitle` for the title and put the location in the meta (fall back to *with* only when there is no place). Consider a short place in the home meta (clamp to one line). |
| M-6 | med | Reminders · «امروز» dose strip | Design cards are ≈174 px wide with the name on one line, and the strip scrolls. Impl cards are 150 px, so «فولیک اسید ۴۰۰ میکروگرم» wraps onto two lines and both cards fit without scrolling. | `app/globals.css:3286-3291` | `width: 174px` (or `min-width`), `white-space: nowrap; text-overflow: ellipsis` on `.rmd-dose-name`. |
| M-7 | med | AddAppointment · date & time | Design: two soft fields side by side, labels «تاریخ» / «ساعت» above, value in the display face («۸ مهر», «۱۰:۳۰») with a small icon. Impl: two list-style rows with coloured 56 px tiles (amber date tile, teal clock tile). In edit mode the tiles themselves show «۸ مهر» and «۱۰:۳۰», so each value appears twice («۸ مهر» + «۸ مهر ۱۴۰۵», «۱۰:۳۰» + «۱۰:۳۰»); a stray bottom border shows under the date row. | `AppointmentFormPage.tsx:279-323` | Replace with the two-field grid; show each value once. |
| M-8 | med | AddAppointment · card grouping | Design: 3 cards — (who, specialty, topic, note) · (date, time, place) · (remind-before, calendar, prep). Impl: 5 cards (who/specialty and topic/note split; prep in its own card). | `AppointmentFormPage.tsx:224-386` | Merge into the design's three cards. |
| M-9 | med | AppointmentDetail · actions | Design: one row — wide outlined neutral «افزودن به تقویم» (calendar icon) + narrow «لغو نوبت» with rose text and rose border. Impl: full-width solid brand «افزودن به تقویم», then full-width ghost «لغو نوبت» in brand purple. The cancel confirm sheet uses `btn-primary` (gradient) for the destructive «بله، لغو شود». | `AppointmentDetailPage.tsx:270-280, 290-296` | Secondary outlined calendar button; cancel in the danger role (`--danger-deep` text, `--danger-soft` border) in both the button and the confirm sheet. |
| M-10 | med | AppointmentDetail · «در خصوص» row | Design: the appointment's own text («سونوگرافی NT هفته ۱۱ · غربالگری سه‌ماهه اول» = title · note). Impl: only the topic label («سونوگرافی»); the note (`appt.notes`) is not shown anywhere on the detail. | `AppointmentDetailPage.tsx:194` | Value = `[appt.title, appt.notes].filter(Boolean).join(' · ')`, falling back to the topic label. |
| M-11 | med | AppointmentDetail · prep checklist | Design: the whole checklist sits in one card with the title «قبل از نوبت» inside; items use square brand checkboxes, no strikethrough. Impl: the title is outside the card and items use round success-green check dots with strikethrough (the home-dose visual). | `AppointmentDetailPage.tsx:244-266` | Put the heading inside the card; square `--brand-fill` checkbox; keep strikethrough off (or muted text only). |
| M-12 | med | Medication edit · dose value | Stored dose shows Latin digits «400» in the input; design (and the rest of the screen) uses Persian digits «۴۰۰». | `MedicationFormPage.tsx:275-283` | Display `formatNumber(state.dose, locale)` and convert Persian digits back to ASCII on change before the value is saved. |
| L-1 | low | Reminders · medication list order | Design order is by first slot (فولیک 08:00, ویتامین D 21:00, آهن 13:00 appears last as inactive). Impl lists newest first (آهن on top). | `screens/reminders/ui/MedicationSection.tsx:77` | Sort active first, then by first time. |
| L-2 | low | Reminders + home · separators | Design joins meta parts with « · »; impl uses «، » everywhere (`rowMeta`, `nextAppointmentMeta`, `reminderAt`). | `frontend/messages/fa/care.json:21-22, 47, 61-62, 240` (+ `backend-go/resources/translations/fa/care.json`) | Switch the fa strings to « · ». |
| L-3 | low | Reminders · iron row schedule | Design «یک روز در میان»; impl «۴ روز در هفته». `medications.everyOtherDay` exists in the messages but is never used. | `screens/reminders/model/view.ts:151-160` | Use `everyOtherDay` for alternating patterns, or drop the unused string. |
| L-4 | low | Reminders · time format | Design shows 13:00 as «۱۳:۰۰»; impl «۱:۰۰ ظهر». (The artboard itself mixes «۸:۰۰ صبح» and «۱۳:۰۰», so treat as a copy decision.) | `screens/reminders/model/view.ts:85` | Decide on one format; if 24 h for afternoon slots, follow the artboard. |
| L-5 | low | Reminders · medication tile colour | Design gives iron a rose tile; impl colours by form (tablet → brand). The home card shows vitamin D in a teal capsule tile where the home artboard used the brand pill for every dose. | `widgets/today-reminders/model/dose-row.ts:72-73` | Keep form-based (consistent), or add a rose tone for iron if the design intent is per-substance; document the choice. |
| L-6 | low | Home card · placement | Pregnancy artboard puts the card right under the baby-size card; impl puts it after the quick actions and the «ویزیت بعدی» card, which shows the same NT visit directly above the card's own appointment row. On the cycle home it sits far down (after the middle banner). | `screens/pregnancy/ui/PregnancyPage.tsx:324-325`; `screens/home/ui/HomePage.tsx:989` | Move the card up; hide the appointment row (or the NextVisit card) when both would show the same visit. |
| L-7 | low | Home card · taken tick | Design tick is dark ink on the success fill; impl uses a white tick. | `app/globals.css:3125` | Use `--ink`-on-success (light) as in the artboard, if contrast allows. |
| L-8 | low | AddChooser · sheet chrome | Design title ≈20 px/800 with a 44 px soft circular close button; impl uses the shared `AppSheet` header (smaller title, bare ×). | `app/sheets/registry.tsx` (sheet `reminders-add`) / `shared/sheet` | Shared primitive, so accept or change it app-wide; do not fork it for this sheet. |
| L-9 | low | AddAppointment · defaults & CTA | «افزودن به تقویم گوشی» defaults off (design on); default topic «ویزیت دوره‌ای» (design shows «سونوگرافی» selected); save button carries a ✓ icon the design lacks. Medication save is a gradient `btn-primary` (radius 14, 44 px), appointment save a solid pill `rmd-cta` (54 px) — the two forms are inconsistent (design: solid pill for both). | `screens/reminder-appointment-form/model/form.ts:90`; `AppointmentFormPage.tsx:390-393`; `MedicationFormPage.tsx:468` | Default `addToCalendar: true`; drop the icon; use `rmd-cta` on both forms (the gradient is still allowed on main CTAs by §10.2, but pick one). |
| L-10 | low | AddMedication · weekdays summary | Design shows «هر روز» as a caption under the day circles; impl puts it at the end of the label row in brand colour. | `MedicationFormPage.tsx:358-363` | Move it under the chips, muted colour. |
| L-11 | low | All · Latin glyphs | «NT», «D» render in the system fallback at regular weight next to bold Persian (the app's Vazirmatn subset has no Latin); the design renders them bold. App-wide, not M3-specific. | `app/[locale]/layout.tsx:28-42` | Optional: add Basic Latin to the Vazirmatn subset (~+8 KB). |

## States

- **Empty:** /reminders (`reminders-empty-*`) shows «امروز دارویی برای مصرف نیست.», «هنوز دارو یا مکملی اضافه نکردی.»,
  «نوبتی ثبت نکردی.»; the home card shows «امروز یادآوری نداری.» + «افزودن یادآور». There is no empty-state artboard, and
  these fit the design's card language.
- **Loading / error:** implemented in code (skeleton lines and a `loadError` + retry block in `TodayCard.tsx:50-61`,
  `MedicationSection.tsx:61-73`, `AppointmentSection.tsx:60-72`, `AppointmentDetailPage.tsx:60-70`; the home card
  shows a skeleton while loading and hides on error). They were not screenshotted. There is no artboard for them.
- **Tabs:** «داروها» / «نوبت‌ها» filter correctly and keep the dose strip on «داروها» (`reminders-meds-*`, `reminders-appts-*`).
- **Chooser → phone:** «مشاوره پزشکی» preselects «مشاوره تلفنی» and relabels the place field «شماره تماس» (`appointment-form-*-impl-new-phone.png`).
- **RTL:** no mirroring bugs found. Chevrons, back arrows, switches (on = knob at the start) and date tiles are on the correct side.

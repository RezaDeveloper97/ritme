# Checkups (M4): design-fidelity audit against v14

Date: 2026-09-29. Branch `stage`. This is an audit only: no app code was changed.

**Design:** `docs/design/checkups-v14/` has 12 `.dc.html` artboards (`nbl_` light, `nbd_` dark). Each was rendered in headless
Chrome (CDP, port 9272, private profile) at 390 px wide and full artboard height, at @2x.

**Implementation:** the current code on `stage` ran locally. That means backend-go on `:8023` (`:8020`, `:8021` and `:8022` were
busy), scratch DB `ritme_m4audit` goose v6 on the docker test stack, and `SMS_PROVIDER=log`. Next dev ran from a scratch copy
of `frontend/` on `:3017`, with `NEXT_PUBLIC_API_BASE_URL` passed as an env var. `.env.local` was not touched.

The data was set up to match the artboards. The user is 34 years old with a 28-day cycle, 4 periods and today at cycle day 17.
The records are: Pap 2022-04-10, which makes it overdue; clinical breast exam 2025-10-05; blood test 2026-02-05 (follow-up, with
a note and an attachment flag); and dentist 2026-06-01. The self-exam has never been done. Every screen was captured full page
at 390×844 @2x in fa, in light and in dark.

The stage screenshots in `screenshots/stage-*.png` only show the viewport, so they were used to cross-check, not as the source
for the comparison.

**Colour rule:** the design wins on layout and content. `frontend/CLAUDE.md` §10.2 wins on colour. Some differences come from
that rule, for example the gradient on the primary CTA. Those are marked *palette* and are not counted as defects.

**Admin:** no admin design exists. There is nothing for checkup types in `docs/design/**`, so admin-web was not compared. Only
the known item 5c applies there (Latin and Persian digits mixed in the fa UI).

## Summary

| Severity | Count |
|---|---|
| High | 4 |
| Med | 16 |
| Low | 20 |
| **Total** | **40** |

Two more items are listed but not counted: A4 and B10 are *palette* or already-decided choices.

The paired screenshots are in `screenshots/audit/`. For each screen there is a `<screen>-<light|dark>-design.png` and an
`-impl.png`, both 585 px wide and compressed with pngquant. The screens are `home-card`, `list`, `detail`, `markdone`,
`history` and `self-exam`. The `detail` impl shot is Pap smear in the overdue state, which is the same item the artboard shows.

## Deviations

### A. Home card: `v14_Main` → «چکاپ‌های دوره‌ای» (`home-card-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| A1 | **high** | Main: Pap overdue row on an amber tint, with the sub-line «عقب‌افتاده از فروردین» and the action «ثبت نوبت» | `frontend/src/widgets/checkups-card/model/counts.ts:29-31`, `ui/CheckupsCard.tsx:85,99-102` | An overdue Pap smear is shown as a rose «راهنما» row that links to the detail page, not as the amber «ثبت نوبت» booking row. `highlightKind` returns `book` only when there is no `timingLabel`, and Pap is cycle-timed (days 10–20), so it never gets `book`. | Make `book` depend on status alone (`overdue` and not self-performed → `book`), whatever the cycle timing. Keep `guide` for self-exam. |
| A2 | med | Main: row sub-line «۳ روز دیگر · روز ۷ تا ۱۰ سیکل» and «عقب‌افتاده از فروردین» | `CheckupsCard.tsx:80` | The sub-line shows only `timingLabel` (for example «روز ۱۰ تا ۲۰ سیکل»). The *when* is missing: the days left, or «overdue since …». | Build the meta from `nextDueLabel` and `timingLabel` joined with `separator`. For an overdue row, show the overdue-since label. |
| A3 | low | Main: self-exam (due) comes first, then Pap (overdue) | `backend-go/internal/checkups/handlers.go:148-158` | The ranking puts a cycle-timed overdue item first, so Pap comes before the self-exam. | Confirm the order with design. The artboard order is: the self-exam guide first, then the other overdue items. |
| A4 | *palette* | Main: the progress-ring arc is violet (brand) | `CheckupsCard.tsx:61` | The arc is `--success` green. §10.2 lists *active progress* as a brand-gradient role. | Use `--brand`, or the gradient via an SVG `linearGradient` built from tokens. This is also closer to the design. |
| A5 | low | Main: the 2 rows sit about 10 px apart | `CheckupsCard.tsx:84` (`mt-2.5`) with the row's `py-2.5` | The rows look about twice as far apart as in the design (see `home-card-light-*`). | Wrap the rows in `flex flex-col gap-2` and drop the per-row `mt-2.5`. |

### B. Checkups list: `v14_Checkups` (`list-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| B1 | **high** | Checkups: section «عقب‌افتاده» holds Pap smear | `backend-go/internal/checkups/engine/engine.go:282-292` | A cycle-timed *overdue* item goes into «این ماه», so Pap (days 10–20, overdue since 1404) sits under «این ماه» and the «عقب‌افتاده» section is not shown. The fix for local bug 4 only reordered the sections. | Send cycle-timed items to `this_month` only while `due` (with the window inside the month). Send `overdue` to `overdue`. The self-exam is the exception, because missing it is a monthly miss. |
| B2 | med | Checkups: items in the same section share **one card** with dividers (سالانه: breast exam + blood test) | `frontend/src/screens/checkups/ui/CheckupsPage.tsx:98-103,231-235` | Each item is its own `card`, so the list is longer and the grouping is weaker. | Render one `card` per section with `divide-y divide-(--line)`, and give each row `p-4` and no border. |
| B3 | med | Checkups: line 2 of a row «هر سال · توسط پزشک», «هر سال · CBC، تیروئید، ویتامین D، آهن», «از ۴۰ سالگی · هر ۱ تا ۲ سال» | `CheckupsPage.tsx:94,110` | The meta joins only `intervalLabel` and `timingLabel`. The subtitle or performer and the age-start («از ۴۰ سالگی») are missing. | Add `item.subtitle`, or the performer label, and the age-start label to `meta`. |
| B4 | med | Checkups: lines wrap to two lines («… بعدی: ۳ روز دیگر») | `CheckupsPage.tsx:109-111` (`truncate`) | The meta and last/next lines are cut with «…», so key data is lost («بعدی: ع…», «بعدی: ۶…», «از ۱۴۱۱ (۴۰ …»). | Drop `truncate` from the two meta lines, or use `line-clamp-2`. |
| B5 | med | Checkups: status pills are **outlined with an icon**: bell «موعدش رسیده», info «عقب‌افتاده», clock «به‌زودی», check «به‌روز» | `CheckupsPage.tsx:31-42` | The pills are filled tints with no icon. The same pill is used in the summary card, where the design also uses the outlined bell pill. | Use `border-[1.5px] border-(--ck-ink) bg-transparent` or a soft fill, plus a status icon map (`bellPlain`, `info`, `clock`, `check`). |
| B6 | med | Checkups: tiles are clinical exam = amber stethoscope, Pap = violet shield, blood test = teal flask, dentist = green tooth, mammography = neutral ribbon | `backend-go/db/migrations/00003_checkups.sql:87-91` (seed `icon`/`tone`), `frontend/src/entities/checkup/model/icon.ts:11-29` | The seed has clinical = `breast`→ribbon violet, Pap = flask violet, blood = amber, dentist = teal, mammography = shieldCheck rose. The icons and tones do not match the artboard. Admin-web shows the same seed. | Change the seed or admin data: clinical `stethoscope`/amber, Pap `shield`/violet, blood `flask` with tone amber (teal is reserved for data, §10.2), dentist `tooth`/green, mammography `ribbon`/neutral. It is data only, and can be set in admin. |
| B7 | low | Checkups: rows have no chevron | `CheckupsPage.tsx:116` | There is an extra chevron on every row. | Remove it, or keep it for affordance after confirming with design. |
| B8 | low | Checkups: header start button is a filter/sliders icon | `CheckupsPage.tsx:177` | The button shows `cog`. | Use the filter/sliders icon if the set has one. The action (plan settings) stays the same. |
| B9 | low | Checkups: section headings are 15 px ink, extrabold | `CheckupsPage.tsx:227` | The headings are 13 px `--ink-3`, so the hierarchy is weaker. | Use `text-[15px] text-(--ink)`. |
| B10 | *palette/known* | Checkups/Main: the separator is « · » | `frontend/messages/fa/checkups.json` `separator` | fa uses «، ». This is the deliberate fix for stage bug S1 (T-M7-17). | Keep it. |
| B11 | low | Checkups: last and next dates are month + year («آخرین: مهر ۱۴۰۴») | `CheckupsPage.tsx:92` (`formatLongDate`) | Full dates («۱۴ مهر ۱۴۰۴») make the lines longer, which feeds into B4. | Use a month-year formatter for list rows. Keep the full date on the detail page. |
| B12 | low | Checkups: legend without counts; info note is a white card with a round violet icon tile; «افزودن چکاپ سفارشی» is neutral ink with a line border | `CheckupsPage.tsx:80,241-249` | The legend adds counts, which is harmless. The note is a `--surface-2` box with a bare icon. The add button has brand text and a brand border. | Match the note card and the neutral add button (`btn-ghost` with ink text and a `--line` border). |
| B13 | low | Checkups: mammography under the age shows «به‌روز» | server status `not_yet` → «هنوز زود است» | The status differs. The implementation is arguably clearer. | Confirm with design. The engine already counts `not_yet` as up to date in the summary. |

### C. Checkup detail: `v14_CheckupDetail` (`detail-*`, Pap smear overdue)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| C1 | **high** | Detail hero: «حدود ۶ ماه گذشته · آخرین بار فروردین ۱۴۰۱», meaning time since the **due date** | `frontend/src/screens/checkup-detail/ui/CheckupDetailPage.tsx:59-63` | `monthsSince(lastDoneOn)` gives «حدود ۵۴ ماه گذشته», the time since the last visit, printed right under «موعد بعدی». Read that way it misstates how overdue the item is. | When the item is overdue, count months since `nextDueOn` («… از موعد گذشته»). Otherwise leave it out. |
| C2 | med | Detail hero: a soft lavender tint card with ink text; «انجام دادم» on a white/surface button; «ثبت نوبت» on a lavender tint outlined button | `CheckupDetailPage.tsx:71,92,99,108` | The hero uses the full brand gradient with white text. In dark mode it is a bright gradient slab, while the design is a deep violet tint. | Use a tint surface (`--pink-bg` → `--surface` or the design's lavender token) with ink text. The gradient then stays scarce (§10.2). |
| C3 | med | Detail hero: the category chip is the **subtitle** with the type icon («غربالگری دهانه رحم») | `CheckupDetailPage.tsx:74-76` | The chip shows the category («هر چند سال»). | Show `detail.subtitle` and the type icon. The category is already in the header sub-line. |
| C4 | med | Detail «سابقه»: a status dot, month + year, «گزارش پیوست شده»/«بدون پیوست», and an outlined result/status pill at the end | `CheckupDetailPage.tsx:118-143` | The result is plain text after the date («…، نرمال»), there is an `x` icon on «بدون پیوست», a pencil icon instead of the pill, and no dot. | Use a result chip (the same `RESULT_CHIP` as History), a leading tone dot, and a note icon only when there is an attachment. Keep tap-to-edit on the row without the pencil. |
| C5 | low | Detail hero: a big type icon in a circle, vertically centred at the start of the hero, and the status pill in the top corner | `CheckupDetailPage.tsx:81-83` | The icon is small, at the top, and always `shield` (known 5b). | Use `checkupIcon(detail.icon)` in a 56 px circle beside the date block. |
| C6 | low | Detail hero: next due is month + year («فروردین ۱۴۰۴») | `CheckupDetailPage.tsx:60` | The full date «۲۲ فروردین ۱۴۰۴» is shown. | Use month + year, as in B11. |
| C7 | low | Detail prep: the step text is semibold ink, and the cycle hint is plain small text under the list | `CheckupDetailPage.tsx:193,201-209` | The steps are regular `--ink-2`, and the hint is in a tinted box with a drop icon. | Use `font-semibold text-(--ink)` and a plain hint line. |
| C8 | low | Detail: the disclaimer is plain centred text | `CheckupDetailPage.tsx:240-243` | It is a tinted box with an info icon. The same applies to self-exam (E-list). | Use plain centred `text-[11.5px] text-(--ink-3)`. |

### D. MarkDone sheet: `v14_MarkDone` (`markdone-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| D1 | med | MarkDone: the result options have icons (check, info, clock). The selected «نرمال» is **green** (success tint and border) | `frontend/src/screens/checkup-mark-done/ui/MarkDoneSheet.tsx:228-243` | There are no icons, and the selected option is brand violet for every result. | Add icons and colour by result: normal → `--success`/`--success-soft`, follow-up → amber, pending → neutral. These are the status roles §10.2 allows. |
| D2 | med | MarkDone: a green next-due banner (success-soft) with a bell icon, green title, and «تغییر» as a text link | `MarkDoneSheet.tsx:318-329` | The banner is on `--pink-bg`, with a calendar icon, ink title, and «تغییر» as a chip button. | Use a `--success-soft` surface, `bellPlain`, a `--success` title, and a text-link style for «تغییر». |
| D3 | med | MarkDone: the sheet hugs its content, about 60 % of the screen | `frontend/src/app/sheets/registry.tsx:174-175` (`size: 'full'`) | The sheet opens full height, leaving a large empty area under «ثبت» in both themes. | Use `size: 'auto'`/`'half'` (content height with a max). |
| D4 | low | MarkDone: the date is a full-width filled field under its label | `MarkDoneSheet.tsx:216-222` | The date is an inline chip beside the label. | Use a label-above field (`fld-card`-style) with the calendar icon at the end. |
| D5 | low | MarkDone: the attachment buttons have **dashed** borders, with the icon stacked above the label | `MarkDoneSheet.tsx:280-287` | They use solid `btn-ghost` with an inline icon. | Use `border-dashed` and `flex-col`. |
| D6 | low | MarkDone: the note is a single-line filled field with a note icon; the label is «یادداشت» (without «اختیاری»); placeholder «مثلاً پزشک گفت سال بعد تکرار شود» | `MarkDoneSheet.tsx:307-312`, `messages/fa/checkups.json` (`notePlaceholder`) | The note is an outlined multi-line textarea with a different placeholder. | Align the copy, and style it as a filled field that grows. |
| D7 | low | MarkDone: the close button is a tinted circle | sheet title/close (shared `AppSheet`) | It is a plain «×». | Use the tinted round close button if other sheets allow it (shared component). |

### E. Self-exam: `v14_SelfExam` (`self-exam-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| E1 | **high** | SelfExam: «سه مرحله، حدود **۵** دقیقه» | `frontend/src/screens/checkup-self-exam/ui/SelfExamPage.tsx:123-126` | The page says «۳ مرحله، حدود **۳** دقیقه», because `minutes = max(3, steps.length)`. That contradicts the seeded prep step «حدود ۵ دقیقه» and the artboard. | Take the minutes from the catalog (a new field) or use the fixed 5 from the design. Don't derive it from the step count. |
| E2 | med | SelfExam hero: a soft tint card. Chips «این ماه» and «روز ۷ سیکل · ۳ روز دیگر» (the window start and how far away it is). Headline «چند روز بعد از پریود» in large display type, the body under it, and the day ring at the end | `SelfExamPage.tsx:71-91` | The hero is a full gradient with the **title** «خودآزمایی سینه» as the headline. «بهترین زمان» is pushed into a nested box. The chip reads «روز ۱۷ سیکل، ۱۸ روز دیگر», the *current* day, which is ambiguous next to «۱۸ روز دیگر». | Restructure the hero to follow the artboard: best-time headline, window chip «روز {from} سیکل · {n} روز دیگر», tint surface. |
| E3 | med | SelfExam header: subtitle «هر ماه · روز ۷ تا ۱۰ سیکل», and the end button is a **bell** (remind toggle) | `SelfExamPage.tsx:248-258` | There is no subtitle, and the end button is a `history` link. The remind toggle is not on this screen. | Add the interval/timing subtitle and the bell toggle (the same as `CheckupDetailPage.tsx:36-47`). Move history elsewhere, for example to the adherence line. |
| E4 | low | SelfExam: the finding chip «چیزی متفاوت نبود» is preselected (violet outline) | `SelfExamPage.tsx:150-160` | No chip is preselected. | Preselect the `exclusive` "none" option, or confirm with design. |
| E5 | low | SelfExam: the adherence line has a history icon, and there is no disclaimer | `SelfExamPage.tsx:193-202` | There is no icon, and there is an extra disclaimer box (see C8). | Add the icon, and make the line link to History. |

### F. History: `v14_History` (`history-*`)

| # | Sev | Design (element) | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| F1 | med | History: **one card** holding a timeline. Each entry has the checkup's **icon tile (tone)** with a connector, the month + year, the title, the result plus the **note** («ویتامین D پایین · پیگیری شد»), and the «پیوست» chip **inline** | `frontend/src/screens/checkup-history/ui/CheckupHistoryPage.tsx:110-156` | Each record is its own card beside a brand dot. There is no type icon, the note is never shown, the full date is repeated, there is a pencil, and the «پیوست» chip sits outside the card (known 5a). | Rebuild the row: tone icon tile and connector, month label, title, «result · note», and the inline attachment chip, all inside one container card. |
| F2 | med | History: the «خلاصه برای پزشک» card is at the **bottom**, with a chevron | `CheckupHistoryPage.tsx:217-233` | It sits at the top, above the tabs, with a «خروجی PDF» text action. | Move it below the timeline and use the chevron affordance. |
| F3 | low | History: the header end button is share/upload | `CheckupHistoryPage.tsx:212` | The button shows `download`. | Use the share icon (and `navigator.share` where it is available). |
| F4 | low | History: the result is plain text inside the line | `CheckupHistoryPage.tsx:133-137` | The result is a coloured chip. It reads well, and it conflicts with F1 only in hierarchy. | Confirm with design. The chip may be kept for follow-up. |

## Not compared / notes

- **Custom checkup form** (`/checkups/custom/new`) and the plan-settings sheet: no artboard in v14.
- **Admin-web checkup types**: no admin design. Only the known item 5c applies (Latin digits «هر 12 ماه» and «21 تا 65» next to
  Persian stats). Fixing B6 in the seed also changes the admin list tiles.
- Known items from the README: 5a is F1, 5b is C5, 5c is under admin, and 5d («موعدش رسیده» with «بعدی: ۱۸ روز دیگر» for the
  self-exam) is still visible on the list (`list-*-impl.png`). E2 makes it worse.
- The Next.js dev indicator (the round «N» at the bottom start) appears in the impl captures. It is a dev-server artifact.

# Fertility / TTC (M5) — design-fidelity audit

Date 2026-09-29, branch `stage`. Design: `docs/design/ttc-v19/` (`v19_*` light, `nb2_*` dark), rendered with headless
Chrome at 390 px wide, full artboard height. Implementation: `screenshots/stage-*.png` (staging) plus a local run for
the full-height pages and the parts staging did not cover (Log symptoms + note, dark full pages): backend-go on :8022
against a scratch DB, Next dev on :3022, TTC user with 4 × 28-day cycles, cycle day 16, BBT cd 4–15 with a shift at
cd 13, LH faint cd 11 / positive cd 12, today = LH positive, 36.52, «شفاف و کشسان», «بدون محافظت», «درد تخمدان».
Stack stopped and scratch DB dropped afterwards. No app code changed.

Rules for judging: the **design wins on layout, hierarchy and copy**. **frontend/CLAUDE.md §10.2 wins on colour.** So
the gradient «ذخیره» and «ثبت علائم امروز» buttons (the design has solid `#7B61FF`) are correct and are not listed.
Amber = fertile window, turquoise = data/ovulation, red = period only.

Paired screenshots (design on the left, implementation on the right, 585 px, pngquant), `screenshots/audit/`:
`home-{light,dark}.png`, `log-{light,dark}.png`, `bbt-{light,dark}.png`, `insights-{light,dark}.png`.

**Count: 27 deviations: 5 high, 12 medium, 10 low.** Bugs already fixed in T-M5-10 (tile colours, gutters, card
padding, fa digits, chart «٫», two-month calendar) are verified fixed and not repeated here.

## Deviations

Paths are relative to `frontend/src/` unless they start with `backend-go/`.

| # | Sev | Design file → element | Implementation | Deviation | Suggested fix |
|---|---|---|---|---|---|
| 1 | **high** | `v19_Main` → ring caption «روز ۱۲ سیکل · شانس بارداری زیاد» + tiles vs home phase card | `screens/home/ui/HomePage.tsx:754` (`dailyCard.fertilityLabel`) vs `backend-go/internal/fertility/day.go:131` (top-level `fertility_level`) | Same day, two different chance values. Staging showed home «متوسط» and Log «کم». Locally both are low, but the words differ: home «پایین», Log «کم». | Use one field everywhere; see the fertility_level section below. |
| 2 | **high** | `v19_TTC_Insights` → «تخمک‌گذاری در سیکل‌های قبل»: one dot strip per month (period red, fertile amber, ovulation turquoise with a glow) and «روز N» in turquoise | `screens/fertility-insights/ui/FertilityInsightsPage.tsx:98-117` | The history is a plain two-column text list (month … «روز ۱۵»), with no cycle strip at all. This is the main visual on the design's insights screen. | Render a strip per `history[]` row. The API needs `cycle_length` + `period_length` per row (or the days), with the ovulation dot from `ovulation_day`. Use the tokens `--period`, `--fert-amber`, `--fert-teal`. |
| 3 | **high** | `v19_TTC_Insights` → window card: small amber overline «پنجره باروری», then the estimate as a 19 px/800 headline | `FertilityInsightsPage.tsx:185-197` | The hierarchy is flipped. «پنجره باروری» is the 15 px bold ink heading, and the estimate sentence (the actual answer) is 13 px muted body text. | Make the title an 11 px overline in `--fert-amber` (tracking .12em) and the range line `text-[19px] font-extrabold text-(--ink) leading-[1.7]`. |
| 4 | **high** | `v19_TTC_Insights` → calendar: window days are 36 px circles with a 1.5 px **dashed amber** border and amber text; **today** is a filled ink disc | `FertilityInsightsPage.tsx:216-225, 235-243` | Window days are filled turquoise-soft with turquoise text, which breaks §10.2 (amber owns the fertile window) and was open item #2 in the README. **Today is not marked at all** (7 مهر has no state). | Window → `border-[1.5px] border-dashed border-(--fert-amber) text-(--fert-amber)`. Keep ovulation turquoise (data). Add a `today` mark (`bg-(--ink) text-(--surface)`). Update the legend swatches to match. |
| 5 | **high** | `v19_Main` → TTC home: ring centre «تخمک‌گذاری تا ۲ روز», phase legend pills (پریود / فولیکولار / پنجره باروری / لوتئال), «شانس بارداری امروز» card (donut + «زیاد» + window start / ovulation / next period), «امروز تست LH بزن» tip card | `HomePage.tsx:922` mounts only `<FertilityTiles />`; `messages/fa/fertility.json:37-60` (`home.*`) is unused | The design's TTC home is not built. Only the three tiles ship, and they sit under the generic phase card. The copy for the rest already exists in i18n but nothing reads it, so those strings are dead. README scoped M5 to the tiles only, so this is a **scope decision**, but the gap is the largest one on the screen. | Decide: either queue a «TTC home» task (ring caption, chance card, LH tip, from `/fertility/today` + the cycle view) or delete the unused `fertility.home.*` keys. Tiles should then go **below the chance card**, as in the design. |
| 6 | med | `v19_TTC_Log` → all inputs (LH, BBT, mucus, intercourse, symptoms) sit in **one** card with 20 px gaps. The note is outside it with its own label, above the button. | `screens/fertility-log/ui/FertilityLogPage.tsx:166-181, 316-431` | Every section is its own `card`: 6 stacked cards, and the note is in a card too. The page is noticeably longer and busier than the design. | Wrap LH → symptoms in one `card p-4 flex-col gap-5` and make `Section` a plain `<section>` (keep the focus ring on the section). Put the note label + textarea outside the card. |
| 7 | med | `v19_TTC_Log` → BBT field: one lavender bordered field, value right-aligned (Lalezar ~40 px), unit, and **both − / + buttons on the left inside the field** | `FertilityLogPage.tsx:327-358` | The − and + buttons flank a centred value box (− right, + left). The field has no border. There is also an extra «پاک کردن دما» link (`:359-367`), which the design does not show. | Put the steppers together at the inline-end inside a bordered `--fert-field` box, with the value at the start. Keep "clear" only if product wants it (the design implies empty = not logged). |
| 8 | med | `v19_TTC_Log` → chance card: amber overline «شانس بارداری امروز», value «زیاد» 22 px/800 **ink**; 5 bars with graduated heights and filled width | `FertilityLogPage.tsx:137-147` | The colours are inverted: label muted, value amber. The bars are thin (`w-1.5`) and read as a faint glyph. | Label → `text-(--fert-amber) font-bold text-[12px]`, value → `text-(--ink) text-[22px] font-extrabold`, bars `w-2.5` with rounded tops. |
| 9 | med | `v19_TTC_BBT` → range tabs: bordered pill container, active tab a **solid violet fill with white text** | `screens/fertility-bbt/ui/FertilityBbtPage.tsx:212-237` | The active tab is a lavender tint with violet text, and the container has no border. It reads as a weak state. Solid violet is `--brand-fill`, not the gradient, so it fits the palette. | Container `border border-(--line) bg-(--surface) p-1`, active `bg-(--brand-fill) text-(--on-accent)`. |
| 10 | med | `v19_TTC_BBT` → stat cards: icon disc on top (thermometer teal / check green); sub-lines coloured «۶ روز اول» (teal) and «بدون جاافتادگی» (green `#0F7B6C`) | `FertilityBbtPage.tsx:123-147, 259-280` | No icon discs. Hints are muted grey. The avg hint says «میانگین دماهای قبل از جهش» and not «۶ روز اول». The value is «۳۶٫۳۵°C» where the design has «۳۶٫۳۲°». | Add `fert-disc fert-tone-teal` / green `thermo` / `check` discs. Colour the hint `--fert-teal` / a new green `--fert-ok` token (README pair `#0F7B6C` / `#7FE0A8`; amber when gaps > 0). The hint copy is a product choice: the unused key `bbt.stats.firstSixDays` matches the design, while the current copy matches the engine. Use «°» without «C». |
| 11 | med | `v19_TTC_BBT` → tip: sparkle icon on a teal disc; the body ends «در دو سیکل قبل این جهش روز ۱۵ و ۱۶ بود.» | `FertilityBbtPage.tsx:149-162`; `backend-go/internal/fertility/chart.go:169-180` | The icon is `info`, not sparkle. The past-shift sentence only appears when `past_shift_days` is non-empty, which is correct. The frontend key `bbt.tip.pastShifts` (`messages/fa/fertility.json:162`) is unused. | Use `sparkle`. Drop the unused key (the API localises the sentence). |
| 12 | med | `v19_TTC_BBT` → «یادآوری ثبت دمای فردا»: neutral outline button (line border, ink text, **moon** icon) | `FertilityBbtPage.tsx:306-313` | Violet border, violet text and a bell icon. It reads like a secondary CTA competing with the tabs. | `border-(--line) text-(--ink)`, `moon` icon (keep `bellRing` for the "on" state). |
| 13 | med | `v19_TTC_BBT` / `v19_TTC_Insights` / `v19_TTC_Log` → header: back and info are **44 px round buttons** (surface fill + line border), title block centred | `FertilityLogPage.tsx:99-116`, `FertilityBbtPage.tsx:173-209`, `FertilityInsightsPage.tsx:129-155` | Bare chevron / info glyphs (`iconbtn`), and the title is start-aligned next to the chevron. | Use the round header buttons the other v13/v14 screens already use, with the title centred. |
| 14 | med | `v19_TTC_BBT` → header has only back + info | `FertilityBbtPage.tsx:195-200` | Extra «پیش‌بینی‌ها» text link in the header. It crowds the title, and in RTL it sits between the title and the info button. | Move the insights entry to the tip card or the stat cards (README allows both), or make it an icon button. |
| 15 | med | `v19_TTC_Insights` → evidence rows: icons check (green) / thermo (teal) / flask (amber); strength pills are **outlined** (`#F4F0FF` bg, 1 px border): قوی green, متوسط **turquoise**, ندارد amber; ~14 px row gap | `FertilityInsightsPage.tsx:263-289` (+ `STRENGTH_CLASS`, `evidenceIcon`) | All three discs are turquoise, the first icon is a calendar, pills are filled, «متوسط» is amber and rows are tight (`gap-2`). | Per-key tone (cycles → green check, bbt → teal, lh → amber flask). Outlined pills with medium = `--fert-teal`, none = `--fert-amber`. Row gap 14 px. |
| 16 | med | `v19_TTC_Insights` → confidence pill: outlined, turquoise for «اطمینان متوسط» | `FertilityInsightsPage.tsx:190` | A filled violet pill for every level. Confidence comes from the algorithm, so it is data and should be turquoise. | `border border-(--fert-teal) text-(--fert-teal) bg-(--fert-field)`; low = muted, high = teal. |
| 17 | med | `v19_TTC_Insights` → bottom: a single soft-lavender pill link «✦ چطور تخمین را دقیق‌تر کنم؟» | `FertilityInsightsPage.tsx:296-313` | Rendered as a full card (heading, bullet tips, gradient «ثبت علائم امروز» CTA). It adds a second gradient to the flow and a lot of height. | Collapse to the pill link that opens the tips in a sheet, or keep the card and drop the gradient CTA (`btn-secondary`). |
| 18 | low | `v19_TTC_Insights` → evidence copy «۶ سیکل کامل ثبت شده / طول معمول ۲۹ روز، نوسان ±۲», «جهش دما در ۲ سیکل قبل / روز ۱۵ و ۱۶ سیکل», «تست LH / هنوز در این سیکل ثبت نشده» | `backend-go/internal/fertility` insights evidence titles | Titles are generic («سیکل‌های کامل», «جهش دمای پایه», «تست LH این سیکل») with the facts in the detail line. «(±۰)» has parentheses where the design says «نوسان ±۲». | Put the count/fact into the title as the design does. Detail = «طول معمول N روز، نوسان ±M». |
| 19 | low | `v19_TTC_Insights` → calendar has no month title and no legend (one month) | `FertilityInsightsPage.tsx:199-244` | Two full months, each with a title, plus a legend. That is needed for a window that spans two months, but the card is now about 900 px tall. | Show only the weeks that contain the window (± 1 week), or a single month when it fits. |
| 20 | low | `v19_TTC_Insights` → disclaimer: amber text on the amber-soft box, no icon | `FertilityInsightsPage.tsx:247-250` | Ink-3 text with an info icon. | a text-safe amber (a new `--fert-amber-deep` token; `--fert-amber` fails contrast on the soft box) and drop the icon, or keep the icon for a11y and colour it amber. |
| 21 | low | `v19_TTC_BBT` → legend on one row; coverline label «خط مبنا · میانگین ۶ روز اول» | `widgets/bbt-chart/ui/BbtChart.tsx:165-182` | Wraps to two rows, and the label says «بیشترین ۶ دمای قبل از جهش». The implementation is **correct** for the engine (coverline = max of 6), so the design copy is wrong. | Keep the copy. Shorten it («خط مبنا») so the legend fits on one row, and delete the unused `bbt.legend.coverline` key. |
| 22 | low | `v19_TTC_BBT` → x-axis 1…29 in steps of 5, today's point with a soft halo | `BbtChart.tsx:100-110, 150-155` | Ticks are 1/7/14/21/28, and today is a solid dot with no halo. | Step-5 ticks to the cycle length. Add a 6 px halo circle (`--fert-teal`, opacity .25) behind today. |
| 23 | low | `v19_TTC_BBT` → hero value «۳۶٫۴۲» set tight | `FertilityBbtPage.tsx:103-111` | Lalezar renders «٫» as a spaced comma («۳۶ , ۵۲»). Still open (README open item #3). The Log field shows the same glyph as «۳۶/۵۲». | Render the separator in Vazirmatn: split int/frac and wrap «٫» in a `font-sans` span. |
| 24 | low | `v19_TTC_Log` → header subtitle «۲۹ شهریور · روز ۱۲» | `messages/fa/fertility.json:63` (`log.subtitle`) | «۷ مهر، روز ۱۶» uses a Persian comma. BBT uses «·». | `"{date} · روز {day}"` |
| 25 | low | `v19_TTC_Log` → mucus chips: خشک / چسبنده / کرمی / شفاف و کشسان (no «ثبت نشه») | `FertilityLogPage.tsx:379-387` | Adds a «ثبت نشه» chip. It is useful for clearing, but it adds a fifth chip and wraps differently. | Allow tapping the selected chip again to clear, and drop the extra chip (the same for LH/intercourse would diverge from the design, so keep those). |
| 26 | low | `v19_Main` → tiles: 1 px `#E7E1F4` border, 24 px radius, 16 px/8 px padding, value 10.5 px/600 `#5A5A64` | `widgets/fertility-tiles/ui/FertilityTiles.tsx:37, 46` | No border, 16 px radius (`rounded-2xl`). | `border border-(--line) rounded-3xl`. |
| 27 | low | `v19_Main` → tiles appear with the rest of the home | `HomePage.tsx:922` | Tiles wait for `GET /profile`, which caused a late layout jump on staging (README open item #4). | Mount on `/messages/mode` `is_ttc` (already fetched), or reserve the skeleton height while `/profile` is pending. |

Not deviations (checked): chip selected state (violet border + lavender fill) matches; the chance card gradient and
amber border match in both themes; the dark palette (`nb2_`) maps correctly onto the tokens on every screen. There
is no horizontal overflow and no console errors. RTL order of tiles (LH at the start), chips and stat cards matches
the design. The chart is LTR as specified. Design quirk, not to copy: in `v19_TTC_Insights` the history label
«روز ۱۶» wraps onto two lines.

## fertility_level: which field both screens should use

**Recommendation: the cycle view's top-level `fertility_level` (v1.1, task.md §26) for both the home and the
fertility screens. Retire `daily_card.fertility_level` / `fertility_label` as a source of chance in the UI.**

Evidence:

- `task.md` §26 is the reference mapping: `post_ovulation = low`, `ovulation_likely = peak`,
  `late_follicular_transition = medium`, `period_expected / unknown = unknown`. §35 puts `fertility_level` top-level
  in the API output.
- `backend-go/internal/enums/cycle_logic.go:10-23` `FertilityLevelV11()` implements exactly §26. The resolver uses it
  (`backend-go/internal/cycle/resolver/resolver.go:337-340`, and it also forces `unknown` when the main phase is
  unknown). That value is what `/fertility/*` reads (`backend-go/internal/fertility/day.go:85-131`).
- `backend-go/internal/enums/cycle_logic.go:25-38` `FertilityLevel()` is the **legacy** calendar mapping (old §17):
  `post_ovulation → medium`, `ovulation_likely → very_high`, and never `unknown`. `daily_card` is built with it
  (`backend-go/internal/cycle/view/daily_card.go:262`, a hand port of `DailyCardBuilder.php`). That explains staging
  at cycle day 16 (`subphase post_ovulation`): top-level «low», daily_card «medium».
- The frontend already parses the top-level value: `entities/cycle/api/schema.ts:283,311` → `CycleView.fertilityLevel`.
  Only `screens/home/ui/HomePage.tsx:754` reads `dailyCard.fertilityLabel`.

Fix, frontend-only and without breaking the Laravel-parity contract of `daily_card` while prod still runs Laravel:
the home phase-card pill should read `infoView.fertilityLevel` and label it with `fertility.chance.levels.*`, the same
labels as the Log chance card. That also removes the «پایین» vs «کم» wording split. Hide the pill for `unknown`.
After T-M2-26 (prod on Go), optionally switch `daily_card.go:262` to `FertilityLevelV11()` so the payload is
consistent too. That is a contract change, so update the contract case.

Related, same root: the home timeline's «پنجره باروری» uses the display window from the cycle view (ends on the
estimated ovulation day, §19), and `/fertility/bbt` draws `fertile_window` 10–15. They can differ by a day. Both
should come from the same `anchors` (`estimated_ovulation_date`, §17–19).

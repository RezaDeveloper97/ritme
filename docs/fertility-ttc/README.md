# Fertility tracking for TTC users (M5) — quick tiles, day log, BBT chart, predictions

Source design: the Design canvas (claude.ai artifact `LQgpWxuwxq5oEuwmhCYsvf`). Local copies in
[`docs/design/ttc-v19/`](../design/ttc-v19/): **`v19_*` = light, `nb2_*` = dark** (same markup, dark palette).
The inline styles are the spec.

| Artboard (canvas title) | What | Frontend route / slot |
|---|---|---|
| `v19_Main` → the 3 tiles | grid of 3 cards: تست LH (amber), دمای پایه (teal), رابطه (rose); icon disc (color at 13 % fill, 33 % border), label, today's value or «ثبت نشده» | widget `fertility-tiles` on the cycle home |
| `v19_TTC_Log` «TTC · ثبت روز» | header «ثبت روز» + «date · روز N»; chance card (level + 5 bars); LH chips; BBT input with − / + (0.01 °C steps); cervical mucus chips; intercourse chips; symptom chips; note; «ذخیره» | `/[locale]/fertility/log?date=&focus=lh|bbt|intercourse` |
| `v19_TTC_BBT` «TTC · دمای پایه» | tabs این سیکل / ۳ سیکل / ۶ سیکل; today's value + phase pill (قبل از جهش / بعد از جهش); line chart with coverline (dashed amber), fertile-window band, cycle-day axis; 2 stat cards (pre-ovulation average, logged days / gaps); tip card; «یادآوری ثبت دمای فردا» | `/[locale]/fertility/bbt` |
| `v19_TTC_Insights` «TTC · پیش‌بینی‌ها» | «بر اساس N سیکل ثبت‌شده»; fertile-window card with confidence pill + month calendar (window + ovulation highlighted) + contraception disclaimer; «این تخمین از کجا آمده؟» evidence rows with strength (قوی / متوسط / ندارد); past ovulation per month; «چطور تخمین را دقیق‌تر کنم؟» | `/[locale]/fertility/insights` |

Tile → screen: **تست LH → Log (focus LH)**, **دمای پایه → BBT screen**, **رابطه → Log (focus intercourse)**.
Insights is reached from the Log chance card, the BBT screen (header/tip link) and the BBT stat cards. Log back →
previous screen. Saving the Log invalidates tiles, BBT and insights.

Visibility (assumption, confirm with the user): tiles show on the cycle home when the profile's
`pregnancy_intention = trying`; hidden in pregnancy mode. The screens are reachable for everyone via deep link.

## Data — what exists and what is new

Already in `daily_health_logs` (keep using, so the legacy log page and the cycle engines see the same data):
`basal_body_temperature decimal(4,2)`, `intercourse_type` (`protected|unprotected`, null = «ثبت نشه»),
`ovarian_pain_intensity`, `bloating_intensity`, `breast_sensitivity_intensity`, `spotting`, `notes`.

New table **`fertility_logs`** (goose + schema-only Laravel twin, docs/go-migration/migrations.md) — kept separate so
the Laravel-parity contract of `/health-logs` does not change:

```
id, user_id FK cascade, log_date date, lh_test varchar null (negative|faint|positive),
cervical_mucus varchar null (dry|sticky|creamy|egg_white), bbt_time time null, created_at, updated_at,
UNIQUE(user_id, log_date)
```

Symptom chips map to the existing columns: درد تخمدان → `ovarian_pain_intensity = mild` (unset → null),
نفخ → `bloating_intensity`, حساسیت سینه → `breast_sensitivity_intensity`, لکه‌بینی → `spotting = true`.
Writes to `daily_health_logs` go through the Go healthlog service so its side effects (period-start check,
recalculation mark) run exactly like `POST /health-logs`.

## API (Go only, `/api/v1/fertility`, auth:api, Accept-Language, standard envelope)

| Method | Path | Returns |
|---|---|---|
| GET | `/fertility/today` | `{date, cycle_day, chance:{level,label,bars:0-5}, lh:{value,label}, bbt:{value}, intercourse:{value,label}}` |
| GET | `/fertility/days/{date}` | the merged day: lh, mucus, bbt, intercourse, symptoms[], note, chance, cycle_day |
| PUT | `/fertility/days/{date}` | same body (all optional; explicit null clears) → the merged day; 422 fa/en (bbt 35.00–38.50, not future) |
| GET | `/fertility/bbt?range=1|3|6` | per cycle: `points[{cycle_day,date,value}]`, `coverline`, `fertile_window{from_day,to_day}`, `shift_day|null`, `phase` (pre_shift/post_shift), stats `{pre_ovulation_avg, logged_days, cycle_days_so_far, gaps}`, `past_shift_days[]`, localized tip |
| GET | `/fertility/insights` | `{cycles_used, window{start,end,ovulation}, confidence(low|medium|high), evidence[{key,title,detail,strength}], history[{month_label,ovulation_day}], tips[]}` |

BBT shift rule (engine, pure + tested): coverline = max of the 6 readings before the first reading that is ≥ 0.2 °C
above them; shift confirmed when 3 consecutive readings are above the coverline (3-over-6). Ovulation ≈ the day
before the first high reading. Confidence combines cycle count/variability (cycle engine), confirmed BBT shifts
and positive LH tests. The fertile window and chance level come from the existing cycle view (`fertility_level`,
`estimated_ovulation_date`) — no second prediction model.

Insights confidence (T-M5-03): evidence strengths score strong = 2, medium = 1, none = 0 across `cycles` (strong =
relatively regular over the last 3 cycles, medium = ≥ 1 valid cycle), `bbt_shift` (confirmed shifts in the current +
last 5 cycles: ≥ 2 strong, 1 medium) and `lh` (this cycle: positive strong, faint medium). high ≥ 4 with `cycles` ≠
none; medium ≥ 2; otherwise low (always low without a window). History ovulation = BBT shift − 1, else first positive
LH + 1, else the engine estimate.

## Colors — light (`v19_`) → dark (`nb2_`)

Tokens only (frontend/CLAUDE.md §10), both themes, `lint:dark` green. New/specific pairs:

| role | light | dark |
|---|---|---|
| LH / chance / coverline / window (amber) | `#F5A623`, window band `#FEF3C6` | `#FFB86B`, band `#5A4A2E` |
| BBT line / BBT tile (teal) | `#0FA3C9`, tip disc `#E3F9FE` | `#4CE0C3`, disc `rgba(76,224,195,.14)` |
| intercourse tile (rose) | `#E8436F` | `#FF6B8B` |
| tile disc fill / border | color + `22` / `55` alpha | same alpha on the dark color |
| logged-days ok (green) | `#0F7B6C` | `#7FE0A8` |
| field bg | `#F4F0FF` | `rgba(255,255,255,.05)` |
| selected chip | border `#7B61FF`, bg `#ECE6FF` | border `#B9A6FF`, bg `rgba(185,166,255,.16)` |
| chance card gradient | `rgba(245,166,35,.22)` → `#FFFFFF` | `rgba(255,184,107,.28)` → `#221A3D` |
| chart grid / labels / point fill | `#E7E1F4` / `#6F6F78` / `#F7F3FF` | `#34295A` / `#8C82AD` / `#17112B` |
| hero chip line | `#DDD6FA` | `#3B2F63` |

Base surfaces/text as in docs/care-reminders/README.md. Chart is inline SVG, LTR, with an accessible label and a
visually hidden data table; numbers in Lalezar; fa digits and decimal separator «٫».

## Local e2e (T-M5-09) — 2026-09-27

Everything below ran locally on branch `stage`: backend-go on `:8020` against the docker test-stack MariaDB
(`ritme_dev`, goose up to v5 incl. `00004_fertility_logs`) with `SMS_PROVIDER=log`, and the Next dev server on
`:3000` with `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1`. The test user was `09120000001` (OTP read from
the DB), profile `pregnancy_intention=trying`, `user_goal=ttc`, 28-day cycle, with 4 confirmed periods logged
through `POST /cycle/period` (06-20, 07-18, 08-15, 09-12, 5 days each). Today = 2026-09-27 = cycle day 16.

### verify-all

| Check | Result |
|---|---|
| `go vet ./...` | ✔ pass |
| `go test ./...` | ✔ pass (63 packages ok) |
| `golangci-lint run` | ✔ 0 issues |
| `make test-int PKG=./internal/fertility/...` | ✔ pass (`fertility` 5.0 s, `fertility/bbt` 0.4 s) |
| `npm run typecheck` | ✔ pass |
| `npm run lint` | ✔ pass |
| `npm run fsd:lint` | ✔ No problems found |
| `npm run lint:styles` | ✔ style gate passed (457 files) |
| `npm run lint:dark` | ✔ passed (57 contrast pairs, 123 tokens; only the pre-existing light-mode ⚠ pairs) |
| `npm run test` | ✔ 551 passed |
| Laravel (`pint`, `artisan test`) | skipped — `backend/` unchanged |

### API e2e (`Accept-Language: fa`, bearer token)

| # | Call | Result |
|---|---|---|
| 1 | `GET /fertility/today` | 200 — `cycle_day 16`, chance `low`/«کم»/1 bar, lh/bbt/intercourse all `null` |
| 2 | `PUT /fertility/days/2026-09-27` `{"lh":"positive","bbt":36.74,"intercourse":"unprotected"}` | 200 «ثبت روز ذخیره شد», merged day returned |
| 3 | `GET /fertility/today` | 200 — lh `positive`/«مثبت», bbt `"36.74"`, intercourse `unprotected`/«بدون محافظت» ✔ updated |
| 4 | `PUT …/2026-09-27 {"bbt":39.2}` / `PUT …/2026-09-30 {"lh":"negative"}` | 422 «دما باید بین ۳۵٫۰۰ و ۳۸٫۵۰ درجه باشد.» / 422 «ثبت برای روزهای آینده ممکن نیست.» ✔ |
| 5 | `PUT /fertility/days/{09-15…09-26}` BBT | 12 × 200: low phase 36.30–36.42 (days 4–12), high 36.66/36.70/36.72 (days 13–15); LH faint day 11, positive day 12 |
| 6 | `GET /fertility/bbt?range=1` | 13 points, `coverline 36.40` (= max of days 7–12), `shift_day 13`, `phase post_shift`, `fertile_window 10–15`, stats `pre_ovulation_avg 36.38, logged_days 13, cycle_days_so_far 16, gaps 3`, fa tip ✔ 3-over-6 detected |
| 7 | `GET /fertility/bbt?range=3` | 200, same shape (`cycles, past_shift_days, range, stats, tip`) |
| 8 | `GET /fertility/insights` | `cycles_used 3`, window 2026-10-19 → 10-24 (ovulation 10-24, next cycle), confidence `high`; evidence cycles strong (۳ × ۲۸ روز ±۰), bbt_shift medium (روز ۱۳), lh strong (مثبت روز ۱۲); history شهریور/مرداد/تیر day 15 (`estimate`); 1 tip |
| 9 | `GET /fertility/days/2026-09-27` | merged day (lh, bbt, intercourse, chance, cycle_day 16) ✔ |

Note on seeding: back-dated bleeding via `POST /health-logs` creates *unconfirmed* `cycle_histories` rows (negative
`cycle_length` when out of order), which the resolver's anchor (§10.1, confirmed starts only) ignores — the first try
gave `cycle_day 100`. That is existing Laravel-parity behaviour, not a fertility bug; seed periods with
`POST /cycle/period` instead.

### UI click-through (headless Chrome over CDP, 390×844 @2x, fa)

Flow per theme: home tiles (today cleared) → tap «تست LH» → Log opens with `focus=lh` → select LH «مثبت», BBT + ×3
(→ ۳۶٫۵۲), mucus «شفاف و کشسان», intercourse «بدون محافظت» → «ذخیره» (returns to home) → tiles now read
«مثبت · ۳۶٫۵۲° · بدون محافظت» → tap «دمای پایه» → BBT chart → insights. No raw i18n keys, no horizontal overflow,
no console errors/exceptions in either theme; the chart SVG has its accessible label «نمودار دمای پایه بدن».

Screenshots (`screenshots/`, each in `-light` and `-dark`):

| File | Shows |
|---|---|
| `home-tiles-empty-*.png` | cycle home, TTC tiles all «ثبت نشده» |
| `log-focus-lh-*.png` | «ثبت روز» opened from the LH tile, chance card «کم» |
| `log-filled-*.png` | LH / BBT / mucus / intercourse selected before save |
| `home-tiles-updated-*.png` | tiles after save |
| `bbt-*.png` | BBT chart: coverline, fertile-window band, shift, stats, tip, reminder button |
| `insights-*.png`, `insights-middle-*.png`, `insights-bottom-*.png` | window card + calendar, evidence, history, tips |

### Bugs found (not fixed — report only)

1. **Tile colours swapped vs spec.** LH tile is turquoise and BBT tile is amber; the design (`v19_Main`: LH `#F5A623`,
   BBT `#0FA3C9`) and the tokens' own comments (`--fert-amber` "LH tile", `--fert-teal` "BBT tile") say the
   opposite. `frontend/src/widgets/fertility-tiles/ui/FertilityTiles.tsx:13-17` (`DISC` uses `--data` for `lh`,
   `--amber` for `bbt`, and not the `--fert-*` tokens). Both themes. Repro: `home-tiles-*.png`.
2. **Tiles have no page gutter.** The 3-tile grid runs edge-to-edge (0–390 px) while every other home card sits
   16 px in. `FertilityTiles.tsx:72` (`<section>` without horizontal margin) mounted at
   `frontend/src/screens/home/ui/HomePage.tsx:877` outside the padded wrappers. Repro: `home-tiles-*.png`.
3. **Fertility cards have no inner padding.** `.card` (`frontend/src/app/globals.css:477`) only sets
   background/border/radius; the fertility screens add no padding, so titles, pills and chips touch the card border
   (log sections, BBT hero + stat cards, insights window/evidence/history/tips cards). Sites:
   `frontend/src/screens/fertility-log/ui/FertilityLogPage.tsx:171`, `frontend/src/screens/fertility-bbt/ui/FertilityBbtPage.tsx:96,267`,
   `frontend/src/screens/fertility-insights/ui/FertilityInsightsPage.tsx:99,186,257,297`. Both themes.
4. **Latin digits in fa.** «بر اساس 3 سیکل ثبت‌شده» (insights header) and «3 روز جاافتاده» (BBT stat): the raw count
   is passed to ICU. `FertilityInsightsPage.tsx:148-151` (`count: data.cyclesUsed`) and
   `FertilityBbtPage.tsx:142` (`count: stats.gaps`) — the neighbouring values use `formatNumber(…, locale)`.
5. **Chart Y-axis uses "." not «٫».** Labels render «۳۶.۸»; spec: fa digits and decimal separator «٫».
   `frontend/src/widgets/bbt-chart/ui/BbtChart.tsx:95` (`formatNumber(v.toFixed(1), locale)` — transliterates
   digits only; `formatBbt` would be consistent with the rest of the screen).
6. **Insights calendar hides part of the window.** Window 27 مهر → 2 آبان, but only آبان is drawn (month of the
   ovulation date), so 27–30 مهر are not highlighted. `FertilityInsightsPage.tsx:182-183`. Repro: `insights-*.png`.
7. **Chance disagrees on the same screen flow.** Home phase card shows «شانس بارداری: متوسط» while the tiles'
   source and the Log chance card show «کم» for the same day. The cycle view returns top-level
   `fertility_level: "low"` but `daily_card.fertility_level: "medium"` (GET `/cycle/today`, cycle day 16,
   `subphase post_ovulation`); fertility uses the former (`backend-go/internal/fertility/day.go:131`), the home card the
   latter (`HomePage.tsx:735`). Pre-existing cycle-engine inconsistency, made visible by the TTC tiles. The home
   timeline also shows the fertile window as «۳۰ شهریور تا ۵ مهر — در جریانه» while `/fertility/bbt` gives days
   10–15 (ends 4 مهر).
8. Minor / to confirm with design: the insights calendar paints the window turquoise (`--data`) where the spec table
   lists amber for the window; the BBT hero value's «٫» renders like a spaced comma in Lalezar («۳۶ , ۵۲»).

### Still open (human)

- **Staging e2e** — the agent has no server access (ssh/curl to the staging host are denied), so
  `/api/v1/fertility` on stage and the stage screenshots still need the user.
- **Production** — only when the user asks.

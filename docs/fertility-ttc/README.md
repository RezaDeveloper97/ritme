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

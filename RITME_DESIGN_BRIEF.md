# Ritme — Design Brief for Google Stitch

> Written in English on purpose (Stitch understands English prompts best), but the
> product ships in **Persian (fa, RTL)** first and English (en, LTR) second.
> Every screen must be designed **RTL-first**.

---

## 1. What Ritme is

**Ritme (ریتمی)** is a mobile-first **women's health companion**: a menstrual-cycle
tracker with a full **pregnancy mode**, daily symptom logging, personalized
insights and educational articles.

- **Platform:** installable PWA (Next.js) wrapped in an Android WebView shell.
  So: **mobile app look and feel, inside a phone viewport** — not a website.
- **Canvas size:** design at **390 × 844** (iPhone-class). Max content width is
  ~430px; on wider screens the app renders as a centered phone-shaped shell.
- **Audience:** Persian-speaking women, 18–40. Tone is warm, calm, reassuring,
  medically trustworthy — never clinical-cold, never childish-pink-cute.
- **Chrome:** no browser UI. Custom in-app header per screen + a persistent
  bottom navigation bar. Safe-area insets at top and bottom.

---

## 2. Design language

**Soft, rounded, airy, card-based.** A lavender canvas with pure-white floating
cards. Generous whitespace, one clear focal element per screen, gradient used
sparingly as a *reward*, not as decoration.

Reference feel: Apple Health's calm + Flo/Clue's data clarity, with a
purple→rose brand signature.

Rules:
- Cards float on the canvas: white surface, 1px hairline border, 16px radius.
- Never more than **one** gradient element visible per screen.
- Numbers are hero. Big, bold, tabular. Labels are small and gray.
- Icons are line icons (~1.75px stroke, rounded caps), 20–26px.
- Elevation is soft and colored (purple-tinted shadows), never gray-black harsh.

---

## 3. Color system

### 3.1 Brand — purple → rose gradient (use sparingly)
| Token | Hex | Use |
|---|---|---|
| Brand purple | `#7B61FF` | primary accent, active states |
| Brand deep | `#6A4FE8` | pressed/darker brand |
| Brand rose | `#FF6FAE` | far end of the gradient |
| **Brand gradient** | `linear-gradient(135deg, #7B61FF 0%, #FF6FAE 100%)` | main CTA, active tab, hero card, active progress **only** |

### 3.2 Data accent — turquoise
Used **only** for data, active/current cycle state, insights, prediction curves,
"new" badges. Never for decoration.
| Token | Hex |
|---|---|
| Data | `#3DD6F3` |
| Data deep (text-safe) | `#0FA3C9` |
| Data soft (surface) | `#E3F9FE` |

### 3.3 Period red — the ONE red in the system
Bleeding days read as red by universal convention. Tuned toward the rose end.
Calendar period markers only.
| Token | Hex |
|---|---|
| Period | `#E8436F` |
| Period deep (text) | `#C92C57` |
| Period soft (day cell) | `#FFE4EC` |

### 3.4 Canvas & surfaces
| Token | Hex |
|---|---|
| Page canvas (lavender) | `#F2ECFF` |
| Surface (cards) | `#FFFFFF` |
| Surface 2 | `#F4F0FF` |
| Surface 3 (faint inner panel) | `#FAF8FF` |
| Hairline / divider | `#E7E1F4` |
| Track (progress rail, switch off) | `#DBD4EC` |
| Home backdrop gradient | `#F2ECFF` → `#E3F9FE` (top → bottom) |

### 3.5 Text ramp (strongest → faintest)
`#2F2F35` body strong · `#5A5A66` medium · `#6F6F78` secondary labels ·
`#8B8B96` tertiary · `#ACACB8` placeholder/disabled.
Text on a saturated fill is always `#FFFFFF`.

### 3.6 Status & category accents
- Success `#22B07D` / soft `#DCFCE7` · Warning `#F5A623` / soft `#FEF3C6`
- Danger `#E5484D` / soft `#FCE7E7`
- Violet soft `#ECE6FF` · Indigo `#8D7BF5` / soft `#F0EDFF`
- Teal `#0FA3C9` / soft `#E9FAFE` · Orange `#E9662E` / soft `#FDEBE2`

### 3.7 Cycle phase marker colors (calendar + week strip)
| Phase | Dot color | Cell tint |
|---|---|---|
| Period (menstruation) | `#E8436F` | `#FFE4EC` |
| Fertile window | `#22B07D` | `#E7F8EF` |
| Ovulation day | `#3DD6F3` | `#E3F9FE` |
| PMS / luteal | `#7B61FF` | `#ECE6FF` |
Each tint has a **faint / medium / strong** step graded by conception
probability — so the calendar reads as a soft heat-map, not flat blocks.
A **predicted** period day = hollow ring outline; a **logged** period day = solid fill.

### 3.8 Dark mode (design both)
Canvas `#131022` · surface `#1B172B` · surface-2 `#251E3E` · hairline `#2A2440` ·
text `#EDECF2` / `#C7C5D6` / `#9C9AAF`. Saturated accents lighten
(period `#FF6D91`, data `#5FDCF5`, brand-strong `#B3A5FF`); soft tints become
dark tints of the same hue. White-on-accent text stays white.

---

## 4. Typography

- **Font:** `Vazirmatn` (Persian + Latin, variable). Fallback Tahoma / system-ui.
- Scale: page title **20px / 800** · card title 16px / 700 · body 15px / 500 ·
  label 13px / 500 · caption 11–12px · hero number **34–48px / 800**.
- Line-height 1.7 for Persian body text (Persian needs more leading).
- Numbers: tabular figures. **Persian locale renders Persian digits** (۰۱۲۳۴۵۶۷۸۹).
- Text alignment is always logical (`start`), never hardcoded left/right.

---

## 5. Core components

| Component | Spec |
|---|---|
| **Primary button** | h 44, radius 14, brand gradient fill, white 14/700 text, soft purple glow shadow `0 10px 22px -10px rgba(123,97,255,.7)`. Disabled = flat `#D6CEF4`. |
| **Ghost button** | h 44, radius 24, transparent, 1.5px `#DDD6FA` border, brand text |
| **Small pill button** | h 40, radius 24, 13px |
| **Card** | white, radius 16, 1px `#E7E1F4` border, padding 16 |
| **Input field** | h 52, radius 14, 1.5px `#E3DDF2` border; focus = brand border + 4px `rgba(123,97,255,.12)` ring |
| **Segmented control** | radius 14, `#E7E1F4` track, white active pill w/ soft shadow, brand active text |
| **Icon button** | 36×36 circle, ghost |
| **OTP cell** | 60×56, radius 16, 24px/800 centered |
| **Bottom sheet** | radius 36 top corners, `0 8px 30px rgba(17,32,47,.12)`, scrim `rgba(17,32,47,.38)`, drag handle |
| **Chip / tag** | radius 22, 11px text, soft tinted surface + matching deep text |
| **Frosted pill** | translucent white-on-gradient pill w/ blur — used on colored hero cards |

**Bottom navigation (persistent, 5 slots):**
`Today (home)` · `Calendar` · **`+ Log` (center FAB, gradient circle, raised, purple glow)** · `Cycle (chart)` · `Profile (user)`.
Active item = brand color + filled feel; inactive = `#8B8B96` line icon.
Supports horizontal swipe between tabs.

---

## 6. Screens to design

### A. Onboarding & auth (13 screens)
Single-purpose, one-question-per-screen wizard. Each has: back arrow, step
counter (`۳ از ۱۲`), a thin brand-gradient progress rail, large title, short
gray subtitle, the input, and a bottom primary button.

1. **Splash** — brand mark "ریـــــتمی" on lavender, subtle gradient bloom.
2. **Welcome / intro carousel** — 3 swipeable illustrated slides + dots, "Start".
3. **Signup** — phone number entry (Iranian format), terms checkbox linking to a terms bottom sheet.
4. **OTP** — 4-cell code input, resend countdown timer.
5. **Name** — text field.
6. **Birthday** — Jalali (Persian) date picker wheel.
7. **Intention** — big choice cards: *track my cycle* / *trying to conceive* / *pregnant*.
8. **Cycle length** — number stepper / wheel, default 28.
9. **Period length** — wheel, default 5.
10. **Cycle regularity (duration)** — regular / irregular / not sure.
11. **Height** — wheel, cm.
12. **Weight** — wheel, kg.
13. **Health conditions** — multi-select chip grid (PCOS, thyroid, endometriosis, …).
14. **Setting up** — animated loading/"building your cycle" state with an orbiting loader.
15. **Pregnancy basis** — (if pregnant) last period date vs. due date choice.

### B. Home / "Today" — the flagship screen
Vertical scroll on the lavender→turquoise backdrop:
1. **Header** — recalculate icon (start side) · centered logo "ریـــــتمی" + tagline · bell icon (end side).
2. **Banner slideshow** — full-width rounded promo carousel with swipe + dots.
3. **Connected calendar↔info unit** (single rounded card, two halves, no gap):
   - *Top half (white):* month label + "full month / close" toggle, weekday row,
     the **current week strip** of day circles. Today = filled circle in its
     phase color with a glow. Marked days carry their phase tint; predicted
     period days are hollow rings. Expanded = full month grid + a color legend.
   - *Bottom half (brand/period colored hero):* the selected day's headline —
     a **progress ring** (day X of N), a big "**۴ روز تا پریود**" number, the
     phase name + one-line description, a fertility read-out, and a row of
     **frosted translucent status pills** (period / fertile / ovulation / PMS /
     period tomorrow) with small icons.
4. **Smart tip card** — personalized daily message, turquoise-accented.
5. **Today's challenge card** — a small wellbeing task with a complete toggle.
6. **Articles rail** — horizontally scrolling article cards matched to the current phase (cover image, title, read time).

### C. Calendar
Full month grid (Jalali), month nav arrows + a year/month picker sheet, phase
heat-map cells, legend, and — under the grid — a **selected-day card**: date
title, phase chip, quick stats row, that day's logged entries list, note box,
and actions ("period started here" / "edit period" / "log for this day"), plus a
"today" jump button and a toast after saving.

### D. Log (the + FAB)
Day navigator strip at the top (prev/next day, "today" pill). Then a list of
**category cards**, each a colored dot + title + summary of what's filled +
chevron. Tapping opens a **full-height bottom sheet** for that category.
Categories: **Bleeding · Pain · Digestion · Mood · Sleep · Exercise · Body ·
Discharge · Weight**.
Inside a sheet, fields render as: choice chips, multi-select chips, 0–3 pain
degree selectors, boolean toggles, and number steppers.

### E. Cycle (insights/stats)
Header with title + tagline, then stacked cards:
1. **My cycles** — a list/bar chart of recent cycles with length and regularity verdict.
2. **Smart tip**.
3. **Cycle summary** — average cycle length, average period length, next predicted date, variability.
4. **Week summary** — small multiples of what was logged this week (mood, sleep, pain).
5. **BMI card** — a gauge with the user's BMI band.

### F. Pregnancy mode (replaces cycle mode when active)
1. **Gate / promo** — if not enabled: illustrated card + "start pregnancy mode" CTA.
2. **Pregnancy home** — gradient hero with the **week number** (e.g. «هفته ۲۴»),
   gestational age, a fruit/size badge, and a 40-week progress bar; a week
   navigator card (prev/next week, "current week"); stat tiles (days remaining,
   trimester, due date); **alerts card** (warning-signs to watch); rich weekly
   content sections (baby development, mother's body, tips); quick actions.
3. **Pregnancy onboarding** — LMP or due-date entry.
4. **Pregnancy log** — pregnancy-specific symptom logging.

### G. Articles
- **Articles list** — search + category chips + article cards.
- **Article reader** — presented as a bottom sheet / full-screen reader: cover
  image, title, meta, long-form Persian body, related articles.

### H. Profile & settings
- **Profile home** — identity card (avatar, name, phone, edit icon), a stats row,
  then grouped setting rows with chevrons and dividers, plus an app version line.
- Sub-screens: **Personal info · Health data · Language · Notifications ·
  Reminders · Account management (logout / delete account)**.
- **Phase details** — an explainer screen for each cycle phase.

### I. System states (design these too)
Loading skeletons, an orbiting brand loader, empty states with a small
illustration, error/retry card, toast, PWA **install prompt** sheet, and a
**forced update** blocking screen.

---

## 7. Non-negotiable constraints

1. **RTL first.** Persian layout is the primary design. Back arrows point right,
   chevrons point left, progress fills from the right. Every layout must mirror
   cleanly for English.
2. **Persian numerals** in the fa design; Latin in en.
3. **Jalali (Shamsi) calendar** — month names فروردین…اسفند, week starts on
   **Saturday** (شنبه). Never show a Gregorian month grid in the fa design.
4. Mobile only. No desktop layout, no sidebars, no hover-dependent UI.
5. Minimum tap target 44×44. Body text never below 12px.
6. WCAG AA contrast on both light and dark themes.
7. One gradient per screen. Turquoise only for data. Red only for bleeding.

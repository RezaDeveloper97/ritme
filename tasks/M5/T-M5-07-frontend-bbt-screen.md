---
id: T-M5-07
title: Frontend — «دمای پایه» BBT chart screen
milestone: M5
type: frontend
status: done
depends_on: [T-M5-04]
parallel_group: M5-B
touches: [frontend/src/screens/fertility-bbt, frontend/src/widgets/bbt-chart, frontend/src/app/[locale]/fertility/bbt]
skills: [new-fsd-slice, dataviz]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M5-07 — Frontend — «دمای پایه» BBT chart screen

## Why
Artboard `v19_TTC_BBT` / `nb2_TTC_BBT`.

## Scope
1. Header «دمای پایه بدن» + «سیکل فعلی · روز N», info button (explainer sheet), link to Insights.
2. Tabs این سیکل / ۳ سیکل / ۶ سیکل (URL-synced) → `range`.
3. Card: today's value (Lalezar 44 px, LTR) + phase pill; widget `bbt-chart` — inline SVG, token colors, y grid
   36.2–36.8 auto-scaled, fertile band, dashed coverline, area gradient, points, today highlighted, cycle-day x-axis;
   multi-cycle ranges overlay previous cycles in muted strokes; legend; visually hidden data table.
4. Two stat cards; tip card; outline button «یادآوری ثبت دمای فردا» → creates a daily custom reminder at 07:00 via
   the existing reminders API (or toggles it off if it exists).
5. Empty state (< 3 readings) with a CTA to the Log (focus bbt).

## Acceptance
- Light/dark × fa/en screenshots; chart unit tests (scales, coverline y, band x); build green.

## Note from T-M5-02
`cycles[0]` is the current cycle; «امروز صبح» = last point if `date == today`; «روز N» = `stats.cycle_days_so_far`;
coverline is provisional while `phase` is pre-shift and means **max of the 6 readings before the shift** (not the
artboard's «میانگین ۶ روز اول» — word the legend to match, e.g. «خط مبنا»); `fertile_window` may be `null`;
`tip.body` already contains the past-shift sentence.

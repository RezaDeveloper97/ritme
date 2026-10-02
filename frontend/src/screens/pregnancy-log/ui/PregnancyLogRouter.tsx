"use client";

import { DayLogPage } from "./DayLogPage";
import { PregnancyLogPage } from "./PregnancyLogPage";
import { PregnancySheetPage } from "./PregnancySheetPage";

const V1_TABS = ["symptoms", "weekly", "movement"];

/**
 * `/pregnancy/log` routing (B-N3-06):
 * - default (`[?date=]`) → the log sheet v2 on its pregnancy preset (`/logs/days`), what the pregnancy
 *   «+» and home tiles open;
 * - `?tab=day[&date=]` or `?focus=` (pregnancy alerts «ثبت وزن») → the v2 pregnancy day log (T-M7-12:
 *   `/pregnancy/v2/days`, raises pregnancy alerts);
 * - `?tab=weekly|movement|symptoms` → the v1 tabs (weekly checkup, kick log, symptoms).
 */
export function PregnancyLogRoute({
  tab,
  date,
  focus,
}: {
  tab?: string;
  date?: string;
  focus?: string;
}) {
  if (tab && V1_TABS.includes(tab)) return <PregnancyLogPage initialTab={tab} />;
  if (tab === "day" || focus) return <DayLogPage date={date} />;
  return <PregnancySheetPage date={date} />;
}

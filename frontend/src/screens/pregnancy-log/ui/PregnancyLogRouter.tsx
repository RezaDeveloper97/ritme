"use client";

import { DayLogPage } from "./DayLogPage";
import { PregnancyLogPage } from "./PregnancyLogPage";

const V1_TABS = ["symptoms", "weekly", "movement"];

/**
 * `/pregnancy/log` — the v2 day log (T-M7-12) by default; `?tab=weekly|movement|symptoms`
 * keeps the v1 tabs reachable (the weekly checkup is linked from Today).
 */
export function PregnancyLogRoute({
  tab,
  date,
}: {
  tab?: string;
  date?: string;
}) {
  return tab && V1_TABS.includes(tab) ? (
    <PregnancyLogPage initialTab={tab} />
  ) : (
    <DayLogPage date={date} />
  );
}

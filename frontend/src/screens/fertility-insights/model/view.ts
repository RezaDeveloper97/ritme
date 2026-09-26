import type { FertilityInsights } from "@/entities/fertility";

/** Fewer than two logged cycles → the low-data state (Scope §5). */
export const MIN_CYCLES = 2;

export function isLowData(data: FertilityInsights): boolean {
  return data.cyclesUsed < MIN_CYCLES || data.window === null;
}

export type DayMark = "ovulation" | "window" | null;

/** How a `Y-m-d` day sits relative to the estimated window. */
export function markFor(
  day: string,
  window: FertilityInsights["window"],
): DayMark {
  if (!window) return null;
  if (window.ovulation === day) return "ovulation";
  if (day >= window.start && day <= window.end) return "window";
  return null;
}

/** Evidence row icon, picked by the server key. */
export function evidenceIcon(
  key: string,
): "chart" | "thermo" | "flaskLh" | "drop" | "calendar" | "info" {
  switch (key) {
    case "cycles":
      return "calendar";
    case "bbt_shift":
    case "bbt":
      return "thermo";
    case "lh":
      return "flaskLh";
    case "mucus":
      return "drop";
    case "history":
      return "chart";
    default:
      return "info";
  }
}

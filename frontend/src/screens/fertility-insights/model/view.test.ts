import { createTranslator } from "next-intl";
import { describe, expect, it } from "vitest";

import type { FertilityInsights } from "@/entities/fertility";
import { formatNumber, toApiDate } from "@/shared/lib/date";

import {
  CONFIDENCE_TONE,
  STRENGTH_TONE,
  calendarMark,
  evidenceIcon,
  evidenceTone,
  historyStrip,
  isLowData,
  markFor,
  nextStarts,
  windowMonths,
  windowWeeks,
} from "./view";
import en from "../../../../messages/en/fertility.json";
import fa from "../../../../messages/fa/fertility.json";

const base: FertilityInsights = {
  cyclesUsed: 3,
  window: { start: "2026-09-10", end: "2026-09-15", ovulation: "2026-09-14" },
  confidence: "medium",
  evidence: [],
  history: [],
  tips: [],
};

describe("fertility insights view", () => {
  it("detects low data", () => {
    expect(isLowData(base)).toBe(false);
    expect(isLowData({ ...base, cyclesUsed: 1 })).toBe(true);
    expect(isLowData({ ...base, window: null })).toBe(true);
  });

  it("marks window and ovulation days", () => {
    expect(markFor("2026-09-14", base.window)).toBe("ovulation");
    expect(markFor("2026-09-10", base.window)).toBe("window");
    expect(markFor("2026-09-15", base.window)).toBe("window");
    expect(markFor("2026-09-16", base.window)).toBeNull();
    expect(markFor("2026-09-14", null)).toBeNull();
  });

  it("falls back to info icon", () => {
    expect(evidenceIcon("lh")).toBe("flaskLh");
    expect(evidenceIcon("zzz")).toBe("info");
  });

  it("lists every month the window touches", () => {
    // 2026-10-19 = 27 Mehr 1405, 2026-10-24 = 2 Aban 1405.
    const cross = { start: "2026-10-19", end: "2026-10-24", ovulation: "2026-10-24" };
    expect(windowMonths(cross, "fa")).toEqual([
      { year: 1405, month: 7 },
      { year: 1405, month: 8 },
    ]);
    // Same dates sit inside one Gregorian month.
    expect(windowMonths(cross, "en")).toEqual([{ year: 2026, month: 10 }]);
    // Year boundary in Gregorian.
    expect(
      windowMonths({ start: "2026-12-29", end: "2027-01-03", ovulation: null }, "en"),
    ).toEqual([
      { year: 2026, month: 12 },
      { year: 2027, month: 1 },
    ]);
    expect(windowMonths(base.window, "en")).toEqual([{ year: 2026, month: 9 }]);
    expect(windowMonths(null, "fa")).toEqual([]);
  });
});

describe("insights.basedOn count", () => {
  // The count goes through formatNumber (Persian digits in fa) and must still
  // drive the English plural.
  const render = (locale: "fa" | "en", count: number) =>
    createTranslator({
      locale,
      // Only the fertility namespace is loaded; the cast satisfies the global type.
      messages: { fertility: locale === "fa" ? fa : en } as unknown as IntlMessages,
      namespace: "fertility",
    })(
      "insights.basedOn",
      { count: formatNumber(count, locale) },
    );

  it("renders Persian digits in fa and keeps en plurals", () => {
    expect(render("fa", 3)).toBe("بر اساس ۳ سیکل ثبت‌شده");
    expect(render("en", 3)).toBe("Based on 3 logged cycles");
    expect(render("en", 1)).toBe("Based on 1 logged cycle");
  });
});

describe("insights calendar + history strip", () => {
  const w = { start: "2026-10-19", end: "2026-10-24", ovulation: "2026-10-24" };

  it("covers the window ± 1 week in whole locale weeks", () => {
    const weeks = windowWeeks(w, "fa");
    // Saturday-first: 10-10 (Sat) … 10-31 (Sat) → 4 rows ending on Friday 11-06.
    expect(weeks.length).toBe(4);
    expect(weeks.every((wk) => wk.length === 7)).toBe(true);
    expect(toApiDate(weeks[0][0])).toBe("2026-10-10");
    expect(toApiDate(weeks[3][6])).toBe("2026-11-06");
    expect(windowWeeks(null, "fa")).toEqual([]);
  });

  it("marks today above the window", () => {
    expect(calendarMark("2026-10-20", w, "2026-10-20")).toBe("today");
    expect(calendarMark("2026-10-20", w, "2026-09-29")).toBe("window");
    expect(calendarMark("2026-10-24", w, "2026-09-29")).toBe("ovulation");
  });

  it("tones evidence rows, strengths and confidence", () => {
    expect(evidenceTone("cycles")).toBe("green");
    expect(evidenceTone("bbt_shift")).toBe("teal");
    expect(evidenceTone("lh")).toBe("amber");
    expect(evidenceIcon("cycles")).toBe("check");
    expect(STRENGTH_TONE.medium).toBe("teal");
    expect(STRENGTH_TONE.none).toBe("amber");
    expect(CONFIDENCE_TONE.low).toBe("muted");
    expect(CONFIDENCE_TONE.high).toBe("teal");
  });

  it("draws a finished cycle as a strip", () => {
    const row = { monthLabel: "شهریور", ovulationDay: 15, cycleStart: "2026-08-17" };
    const strip = historyStrip(row, "2026-09-14", 5)!;
    expect(strip).toHaveLength(28);
    expect(strip.slice(0, 5)).toEqual(Array(5).fill("period"));
    expect(strip[5]).toBe("none");
    expect(strip.slice(9, 14)).toEqual(Array(5).fill("fertile"));
    expect(strip[14]).toBe("ovulation");
    expect(strip.slice(24)).toEqual(Array(4).fill("pms"));
    // Unknown ovulation → no window, still a strip.
    expect(historyStrip({ ...row, ovulationDay: null }, "2026-09-14", 5)).not.toContain("ovulation");
    expect(historyStrip({ ...row, cycleStart: null }, "2026-09-14", 5)).toBeNull();
    expect(historyStrip(row, null, 5)).toBeNull();
  });

  it("pairs each row with the next cycle's start", () => {
    const rows = [
      { monthLabel: "a", ovulationDay: 14, cycleStart: "2026-08-17" },
      { monthLabel: "b", ovulationDay: 14, cycleStart: "2026-07-20" },
    ];
    expect(nextStarts(rows, "2026-09-14")).toEqual(["2026-09-14", "2026-08-17"]);
  });
});

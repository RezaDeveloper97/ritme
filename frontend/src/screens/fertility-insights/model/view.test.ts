import { createTranslator } from "next-intl";
import { describe, expect, it } from "vitest";

import type { FertilityInsights } from "@/entities/fertility";
import { formatNumber } from "@/shared/lib/date";

import { evidenceIcon, isLowData, markFor, windowMonths } from "./view";
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

import { createTranslator } from "next-intl";
import { describe, expect, it } from "vitest";

import type { BbtCycle } from "@/entities/fertility";
import type { Reminder } from "@/entities/reminder";
import { formatNumber } from "@/shared/lib/date";

import {
  findBbtReminder,
  hasEnoughReadings,
  parseRange,
  todayReading,
} from "./view";
import en from "../../../../messages/en/fertility.json";
import fa from "../../../../messages/fa/fertility.json";

const cycle: BbtCycle = {
  startDate: "2026-09-01",
  points: [
    { cycleDay: 2, date: "2026-09-02", value: 36.4 },
    { cycleDay: 5, date: "2026-09-05", value: 36.5 },
  ],
  coverline: null,
  fertileWindow: null,
  shiftDay: null,
  phase: null,
};

describe("fertility-bbt view model", () => {
  it("parses the range param", () => {
    expect(parseRange("3")).toBe(3);
    expect(parseRange("6")).toBe(6);
    expect(parseRange("2")).toBe(1);
    expect(parseRange(undefined)).toBe(1);
  });
  it("reads today only when the last point is today", () => {
    expect(todayReading(cycle, "2026-09-05")?.value).toBe(36.5);
    expect(todayReading(cycle, "2026-09-06")).toBeNull();
    expect(todayReading(undefined, "2026-09-05")).toBeNull();
  });
  it("needs 3 readings", () => {
    expect(hasEnoughReadings(cycle)).toBe(false);
  });
  it("finds the active daily 07:00 reminder by title", () => {
    const r: Reminder = {
      id: "1",
      type: "custom",
      title: "BBT",
      subtitle: null,
      notes: null,
      scheduledAt: null,
      recurrence: "daily",
      recurrenceTime: "07:00",
      startsOn: null,
      endsOn: null,
      isActive: true,
    };
    expect(findBbtReminder([r], "BBT")?.id).toBe("1");
    expect(findBbtReminder([{ ...r, isActive: false }], "BBT")).toBeNull();
    expect(findBbtReminder([r], "Other")).toBeNull();
  });
});

describe("bbt.stats.gaps count", () => {
  // The count goes through formatNumber (Persian digits in fa) and must still
  // drive the English plural.
  const render = (locale: "fa" | "en", count: number) =>
    createTranslator({
      locale,
      // Only the fertility namespace is loaded; the cast satisfies the global type.
      messages: { fertility: locale === "fa" ? fa : en } as unknown as IntlMessages,
      namespace: "fertility",
    })(
      "bbt.stats.gaps",
      { count: formatNumber(count, locale) },
    );

  it("renders Persian digits in fa and keeps en plurals", () => {
    expect(render("fa", 3)).toBe("۳ روز جاافتاده");
    expect(render("en", 3)).toBe("3 days missed");
    expect(render("en", 1)).toBe("1 day missed");
  });
});

import { describe, expect, it } from "vitest";

import type { BbtCycle } from "@/entities/fertility";
import type { Reminder } from "@/entities/reminder";

import {
  findBbtReminder,
  hasEnoughReadings,
  parseRange,
  todayReading,
} from "./view";

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

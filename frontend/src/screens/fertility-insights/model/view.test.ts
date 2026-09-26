import { describe, expect, it } from "vitest";

import type { FertilityInsights } from "@/entities/fertility";

import { evidenceIcon, isLowData, markFor } from "./view";

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
});

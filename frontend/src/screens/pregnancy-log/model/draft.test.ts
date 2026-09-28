import { describe, expect, it } from "vitest";

import {
  displayWeight,
  draftFromDay,
  inputFromDraft,
  parseWeight,
  selectedSymptoms,
  setSeverity,
  stepWater,
  toggleSymptom,
} from "./draft";

describe("pregnancy log draft", () => {
  it("toggles symptoms at mild and keeps canonical order", () => {
    let s = toggleSymptom({}, "spotting");
    s = toggleSymptom(s, "nausea");
    expect(s).toEqual({ spotting: "mild", nausea: "mild" });
    expect(selectedSymptoms(s)).toEqual(["nausea", "spotting"]);
    s = setSeverity(s, "nausea", "severe");
    expect(s.nausea).toBe("severe");
    expect(setSeverity(s, "fatigue", "severe").fatigue).toBeUndefined();
    expect(toggleSymptom(s, "nausea")).toEqual({ spotting: "mild" });
  });

  it("clamps water to 0–15", () => {
    expect(stepWater(0, -1)).toBe(0);
    expect(stepWater(15, 1)).toBe(15);
    expect(stepWater(5, 1)).toBe(6);
  });

  it("parses Persian and comma weights", () => {
    expect(parseWeight("۶۲٫۱")).toBe(62.1);
    expect(parseWeight("62,5")).toBe(62.5);
    expect(parseWeight("")).toBeNull();
    expect(parseWeight("abc")).toBeNull();
    expect(parseWeight("5")).toBeNull();
  });

  it("round-trips a day into a full input", () => {
    const draft = draftFromDay({
      date: "2026-09-20",
      week: 8,
      mood: 4,
      symptoms: { fatigue: "moderate" },
      waterGlasses: null,
      weight: 62.1,
      visitNote: null,
      lastWeight: null,
      alerts: [],
    });
    expect(inputFromDraft(draft)).toEqual({
      mood: 4,
      symptoms: { fatigue: "moderate" },
      waterGlasses: 0,
      weight: 62.1,
      visitNote: "",
    });
  });

  it("shows the weight with the locale's digits and decimal separator", () => {
    expect(displayWeight("62.5", "fa")).toBe("۶۲٫۵");
    expect(displayWeight("۶۲٫۵", "fa")).toBe("۶۲٫۵");
    expect(displayWeight("62.5", "en")).toBe("62.5");
    expect(displayWeight("", "fa")).toBe("");
    // What the field shows parses back to the same kilograms.
    expect(parseWeight(displayWeight("62.5", "fa"))).toBe(62.5);
  });
});

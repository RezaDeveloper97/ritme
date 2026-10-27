import { describe, expect, it } from "vitest";

import {
  calcInFertileWindow,
  calcMainPhase,
  cycleDayMarker,
  deriveCyclePredictions,
  markerPhase,
  normalizePhase,
} from "./predictions";
import type { CycleCalculation } from "./types";

function makeCalc(overrides: Partial<CycleCalculation> = {}): CycleCalculation {
  return {
    calculationDate: "2026-01-08",
    cycleDay: 8,
    phase: "follicular",
    subphase: null,
    estimatedOvulationDay: 14,
    cycleLength: 28,
    isFertileWindow: false,
    isPmsWindow: false,
    isPeriodTomorrow: false,
    fertilityPercent: 19.89,
    cycleVariability: "regular",
    dailyTips: [],
    ...overrides,
  };
}

describe("normalizePhase", () => {
  it("passes through known phases", () => {
    expect(normalizePhase("ovulation")).toBe("ovulation");
    expect(normalizePhase("period")).toBe("period");
  });

  it('maps the backend "menstruation" phase to "period"', () => {
    expect(normalizePhase("menstruation")).toBe("period");
  });

  it("falls back to luteal for unknown backend strings", () => {
    expect(normalizePhase("some_future_phase")).toBe("luteal");
    expect(normalizePhase("")).toBe("luteal");
  });

  it("reads the engine `unknown` as no phase, never luteal", () => {
    expect(normalizePhase("unknown")).toBeNull();
    const calc = makeCalc({ phase: "unknown", isFertileWindow: true });
    expect(calcMainPhase(calc)).toBeNull();
    expect(calcInFertileWindow(calc)).toBe(false);
    expect(cycleDayMarker(calc)).toBeNull();
    expect(deriveCyclePredictions(calc).phase).not.toBe("luteal");
  });
});

describe("cycleDayMarker", () => {
  it("marks period and ovulation from the phase", () => {
    expect(cycleDayMarker(makeCalc({ phase: "menstruation" }))).toBe("period");
    expect(cycleDayMarker(makeCalc({ phase: "ovulation" }))).toBe("ovulation");
  });

  it("marks the fertile and PMS windows", () => {
    expect(
      cycleDayMarker(makeCalc({ phase: "follicular", isFertileWindow: true })),
    ).toBe("fertile");
    expect(
      cycleDayMarker(
        makeCalc({ phase: "luteal", cycleDay: 27, isPmsWindow: true }),
      ),
    ).toBe("pms");
    // The legacy flag's first days (7-day run) are plain luteal (§25.2: 3 days).
    expect(
      cycleDayMarker(
        makeCalc({ phase: "luteal", cycleDay: 23, isPmsWindow: true }),
      ),
    ).toBeNull();
  });

  it("prioritizes period/ovulation over the window flags", () => {
    expect(
      cycleDayMarker(makeCalc({ phase: "ovulation", isFertileWindow: true })),
    ).toBe("ovulation");
    expect(
      cycleDayMarker(makeCalc({ phase: "menstruation", isPmsWindow: true })),
    ).toBe("period");
  });

  it("returns null for a plain follicular/luteal day", () => {
    expect(cycleDayMarker(makeCalc({ phase: "follicular" }))).toBeNull();
    expect(cycleDayMarker(makeCalc({ phase: "luteal" }))).toBeNull();
  });
});

describe("deriveCyclePredictions", () => {
  it("derives day offsets from the calculation", () => {
    const p = deriveCyclePredictions(makeCalc());
    expect(p.daysUntilNextPeriod).toBe(21); // 28 - 8 + 1 (next period is cycle day 29)
    expect(p.daysUntilOvulation).toBe(6); // 14 - 8
    expect(p.daysUntilFertileWindow).toBe(1); // 6 - 5 lead days
    expect(p.fertilityPercent).toBe(20); // rounded
    expect(p.cycleDay).toBe(8);
    expect(p.cycleLength).toBe(28);
  });

  it("derives the PMS window as the days before the next period", () => {
    const p = deriveCyclePredictions(makeCalc()); // next period in 20 days
    expect(p.daysUntilPmsEnd).toBe(20); // day before next period
    expect(p.daysUntilPmsStart).toBe(18); // 3-day window (task.md §25.2)
  });

  it("clamps the PMS window to today when a period is imminent", () => {
    const p = deriveCyclePredictions(
      makeCalc({ cycleDay: 28, cycleLength: 28 }),
    );
    expect(p.daysUntilPmsEnd).toBe(0);
    expect(p.daysUntilPmsStart).toBe(0);
  });

  it("never reports a negative days-until-next-period", () => {
    const p = deriveCyclePredictions(
      makeCalc({ cycleDay: 30, cycleLength: 28 }),
    );
    expect(p.daysUntilNextPeriod).toBe(0);
  });

  it("reports a negative ovulation offset once ovulation has passed", () => {
    const p = deriveCyclePredictions(
      makeCalc({ cycleDay: 20, estimatedOvulationDay: 14 }),
    );
    expect(p.daysUntilOvulation).toBe(-6);
  });

  it("clamps the fertility probability to 0–100", () => {
    expect(
      deriveCyclePredictions(makeCalc({ fertilityPercent: 140 }))
        .fertilityPercent,
    ).toBe(100);
    expect(
      deriveCyclePredictions(makeCalc({ fertilityPercent: -5 }))
        .fertilityPercent,
    ).toBe(0);
  });
});

// Legacy calcs draw the biological O−5 … O+1 window (phase `ovulation` on O and
// O+1). Every reader goes through the §19 display window, which ends on O.
describe("§19 reading of a legacy calculation", () => {
  const day = (cycleDay: number, over: Partial<CycleCalculation> = {}) =>
    makeCalc({ cycleDay, estimatedOvulationDay: 15, cycleLength: 28, ...over });

  it("reads the legacy O+1 `ovulation` day as luteal, outside the window", () => {
    const o1 = day(16, { phase: "ovulation", isFertileWindow: true });
    expect(calcMainPhase(o1)).toBe("luteal");
    expect(calcInFertileWindow(o1)).toBe(false);
    expect(cycleDayMarker(o1)).toBeNull();
    expect(deriveCyclePredictions(o1).phase).toBe("luteal");
    expect(deriveCyclePredictions(o1).isFertileWindow).toBe(false);
  });

  it("keeps O−5 … O−1 fertile and O as ovulation", () => {
    expect(cycleDayMarker(day(10, { isFertileWindow: true }))).toBe("fertile");
    expect(cycleDayMarker(day(14, { isFertileWindow: true }))).toBe("fertile");
    const o = day(15, { phase: "ovulation", isFertileWindow: true });
    expect(cycleDayMarker(o)).toBe("ovulation");
    expect(calcInFertileWindow(o)).toBe(true);
  });

  it("never marks a bleeding day fertile (menstrual overrides the window)", () => {
    const bleeding = day(10, { phase: "menstruation", isFertileWindow: true });
    expect(calcInFertileWindow(bleeding)).toBe(false);
    expect(cycleDayMarker(bleeding)).toBe("period");
  });
});

describe("markerPhase", () => {
  it("follows a window marker and splits an unmarked day around ovulation", () => {
    const c = makeCalc({
      cycleDay: 12,
      estimatedOvulationDay: 15,
      isFertileWindow: true,
    });
    expect(markerPhase(c, "fertile")).toBe("fertile");
    expect(markerPhase(c, null)).toBe("follicular");
    expect(
      markerPhase(
        makeCalc({
          cycleDay: 16,
          estimatedOvulationDay: 15,
          phase: "ovulation",
        }),
        null,
      ),
    ).toBe("luteal");
    expect(
      markerPhase(makeCalc({ phase: "luteal", cycleDay: 20 }), "pms"),
    ).toBe("luteal");
  });
});

// SPEC §2.36 (21 / 28 / 35): fertile window = O−5 … O (O = L−14), PMS = the last 3 days.
describe("§2.36 table on the legacy calculation", () => {
  const table = [
    { L: 21, ovulation: 7, fertile: [2, 7], pms: [19, 21] },
    { L: 28, ovulation: 14, fertile: [9, 14], pms: [26, 28] },
    { L: 35, ovulation: 21, fertile: [16, 21], pms: [33, 35] },
  ] as const;

  for (const row of table) {
    it(`${row.L}-day cycle`, () => {
      const at = (cycleDay: number, over: Partial<CycleCalculation> = {}) =>
        makeCalc({
          cycleLength: row.L,
          cycleDay,
          estimatedOvulationDay: row.ovulation,
          phase: "follicular",
          ...over,
        });
      // Predictions: the fertile window opens 5 days before O and PMS is 3 days.
      const p = deriveCyclePredictions(at(1));
      expect(p.daysUntilOvulation).toBe(row.ovulation - 1);
      expect(p.daysUntilFertileWindow).toBe(row.fertile[0] - 1);
      expect(p.daysUntilPmsStart).toBe(row.pms[0] - 1);
      // PMS markers cover exactly the last three days.
      for (let d = 1; d <= row.L; d++) {
        const marker = cycleDayMarker(
          at(d, { phase: "luteal", isPmsWindow: true, isFertileWindow: false }),
        );
        expect(marker === "pms").toBe(d >= row.pms[0] && d <= row.pms[1]);
      }
    });
  }
});

describe("unknown phase / missing fertility number (T-M9-40c)", () => {
  const base: CycleCalculation = {
    calculationDate: "2026-10-27",
    cycleDay: 40,
    phase: null,
    subphase: null,
    estimatedOvulationDay: null,
    cycleLength: 28,
    isFertileWindow: false,
    isPmsWindow: false,
    isPeriodTomorrow: false,
    fertilityPercent: null,
    cycleVariability: null,
    dailyTips: [],
  } as const;

  it("passes null through without a placeholder phase or a number", () => {
    const p = deriveCyclePredictions(base);
    expect(p.phase).toBeNull();
    expect(p.fertilityPercent).toBeNull();
    expect(markerPhase(base, cycleDayMarker(base))).toBeNull();
  });
});

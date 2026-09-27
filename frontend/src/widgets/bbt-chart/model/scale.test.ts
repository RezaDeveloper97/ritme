import { describe, expect, it } from "vitest";

import type { BbtCycle } from "@/entities/fertility";

import {
  buildScales,
  coverlineY,
  DEFAULT_FRAME,
  fertileBandX,
  maxCycleDay,
  yDomain,
  yTickLabel,
} from "./scale";

const cycle = (over: Partial<BbtCycle> = {}): BbtCycle => ({
  startDate: "2026-09-01",
  points: [
    { cycleDay: 1, date: "2026-09-01", value: 36.3 },
    { cycleDay: 2, date: "2026-09-02", value: 36.4 },
  ],
  coverline: 36.45,
  fertileWindow: { fromDay: 10, toDay: 15 },
  shiftDay: null,
  phase: "pre_shift",
  ...over,
});

describe("yDomain", () => {
  it("keeps the 36.2–36.8 grid when values fit", () => {
    const d = yDomain([36.3, 36.7]);
    expect(d.min).toBe(36.2);
    expect(d.max).toBe(36.8);
    expect(d.ticks).toEqual([36.2, 36.3, 36.4, 36.5, 36.6, 36.7, 36.8]);
  });
  it("widens to the nearest 0.1 outside the grid", () => {
    const d = yDomain([35.95, 37.02]);
    expect(d.min).toBe(35.9);
    expect(d.max).toBe(37.1);
    expect(d.ticks[0]).toBe(35.9);
    expect(d.ticks.at(-1)).toBe(37.1);
  });
});

describe("scales", () => {
  it("spans at least 28 days and reaches the furthest point/window", () => {
    expect(maxCycleDay([cycle()])).toBe(28);
    expect(
      maxCycleDay([cycle({ fertileWindow: { fromDay: 20, toDay: 33 } })]),
    ).toBe(33);
  });
  it("maps domain ends to plot edges", () => {
    const s = buildScales([cycle()]);
    const f = DEFAULT_FRAME;
    expect(s.x(1)).toBe(f.left);
    expect(s.x(28)).toBeCloseTo(f.width - f.right);
    expect(s.y(36.8)).toBeCloseTo(f.top);
    expect(s.y(36.2)).toBeCloseTo(f.height - f.bottom);
  });
  it("places the coverline proportionally", () => {
    const s = buildScales([cycle()]);
    const f = DEFAULT_FRAME;
    const plotH = f.height - f.top - f.bottom;
    expect(coverlineY(s, 36.5)).toBeCloseTo(f.top + plotH / 2);
    expect(coverlineY(s, null)).toBeNull();
  });
  it("centres the fertile band on its days with half-day padding", () => {
    const s = buildScales([cycle()]);
    const half = (s.x(2) - s.x(1)) / 2;
    const band = fertileBandX(s, { fromDay: 10, toDay: 15 })!;
    expect(band.x).toBeCloseTo(s.x(10) - half);
    expect(band.width).toBeCloseTo(s.x(15) - s.x(10) + 2 * half);
    expect(fertileBandX(s, null)).toBeNull();
  });
  it("clamps the band to the plot", () => {
    const s = buildScales([cycle()]);
    expect(fertileBandX(s, { fromDay: 1, toDay: 2 })!.x).toBe(
      DEFAULT_FRAME.left,
    );
  });
});

describe("yTickLabel", () => {
  it("uses the locale digits and decimal separator", () => {
    expect(yTickLabel(36.8, "fa")).toBe("۳۶٫۸");
    expect(yTickLabel(36.8, "en")).toBe("36.8");
    expect(yTickLabel(36.75, "en")).toBe("36.8");
  });
});

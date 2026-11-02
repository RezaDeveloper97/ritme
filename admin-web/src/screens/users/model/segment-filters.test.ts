import { describe, expect, it } from "vitest";

import {
  filtersToSegment,
  isFiltered,
  LIST_DEFAULTS,
  segmentToSearch,
} from "./segment-filters";

describe("segment filters", () => {
  it("drops empty values, default status and the segment id when saving", () => {
    expect(
      filtersToSegment({
        ...LIST_DEFAULTS,
        plan: "plus",
        churn: "new",
        mode: "pregnancy",
        segment: "4",
      }),
    ).toEqual({ plan: "plus", churn: "new", mode: "pregnancy" });
  });

  it("keeps a non-default status", () => {
    expect(filtersToSegment({ ...LIST_DEFAULTS, status: "blocked" })).toEqual({
      status: "blocked",
    });
  });

  it("round-trips filters to a query string", () => {
    const seg = { plan: "free", data_quality: "poor", main_warning: "x" };
    const sp = new URLSearchParams(segmentToSearch(seg));
    expect(Object.fromEntries(sp)).toEqual(seg);
  });

  it("detects a narrowed list", () => {
    expect(isFiltered("", LIST_DEFAULTS)).toBe(false);
    expect(isFiltered("RT-", LIST_DEFAULTS)).toBe(true);
    expect(isFiltered("", { ...LIST_DEFAULTS, segment: "2" })).toBe(true);
    expect(isFiltered("", { ...LIST_DEFAULTS, status: "active" })).toBe(true);
  });
});

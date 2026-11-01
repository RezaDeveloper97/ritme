import { describe, expect, it } from "vitest";

import { failureLines } from "./runs";

describe("failureLines", () => {
  it("lists the wrong probes then the problems", () => {
    expect(
      failureLines({
        detail: {
          threshold: "> 180",
          probes: [
            { input: "180/119", want: false, got: false },
            { input: "181/119", want: true, got: false },
          ],
          problems: ["preg.bp_severe: inactive"],
        },
      }),
    ).toEqual(["181/119: true → false", "preg.bp_severe: inactive"]);
    expect(failureLines({ detail: null })).toEqual([]);
  });
});

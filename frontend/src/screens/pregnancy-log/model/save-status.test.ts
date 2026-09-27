import { describe, expect, it } from "vitest";

import { isConfirmedSave, statusAfterSync } from "./save-status";

const key = "pregnancy-day:2026-09-27";

describe("pregnancy log save status", () => {
  it("moves a queued day to saved once its outbox entry is sent", () => {
    expect(statusAfterSync("queued", key, [key])).toBe("saved");
    expect(statusAfterSync("queued", key, ["pregnancy-day:2026-09-26", key])).toBe("saved");
  });

  it("stays queued while another day's entry is what got sent", () => {
    expect(statusAfterSync("queued", key, ["pregnancy-day:2026-09-26"])).toBe("queued");
    expect(statusAfterSync("queued", null, [key])).toBe("queued");
  });

  it("leaves non-queued statuses alone", () => {
    for (const s of ["idle", "saving", "saved", "error"] as const) {
      expect(statusAfterSync(s, key, [key])).toBe(s);
    }
  });

  it("shows the saved button only for a server-confirmed save", () => {
    expect(isConfirmedSave("saved")).toBe(true);
    expect(isConfirmedSave("queued")).toBe(false);
    expect(isConfirmedSave("saving")).toBe(false);
  });
});

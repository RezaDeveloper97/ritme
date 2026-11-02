import { describe, expect, it } from "vitest";

import { can } from "@/entities/admin";

import en from "../../../../messages/en.json";
import fa from "../../../../messages/fa.json";

describe("segment write gating", () => {
  it("marketing manages segments, view-only roles only read, others see nothing", () => {
    expect(can("marketing", { segments: "manage" }, "segments", "manage")).toBe(
      true,
    );
    expect(can("analyst", { segments: "view" }, "segments", "manage")).toBe(
      false,
    );
    expect(can("analyst", { segments: "view" }, "segments")).toBe(true);
    expect(can("support", {}, "segments")).toBe(false);
    expect(can("super", {}, "segments", "manage")).toBe(true);
  });
});

describe("locale parity for the new keys", () => {
  const flat = (o: Record<string, unknown>, p = ""): string[] =>
    Object.entries(o).flatMap(([k, v]) =>
      v && typeof v === "object"
        ? flat(v as Record<string, unknown>, `${p}${k}.`)
        : [`${p}${k}`],
    );
  it("users and access have the same keys in fa and en", () => {
    expect(flat(fa.users).sort()).toEqual(flat(en.users).sort());
    expect(flat(fa.access).sort()).toEqual(flat(en.access).sort());
  });
});

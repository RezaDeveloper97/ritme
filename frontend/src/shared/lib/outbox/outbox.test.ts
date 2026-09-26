import { describe, expect, it } from "vitest";

import { createOutbox } from "./outbox";
import { createMemoryStore } from "./store";

function setup() {
  let t = 0;
  return createOutbox(createMemoryStore(), () => ++t);
}

const entry = (key: string, body: unknown) => ({
  key,
  method: "put" as const,
  url: `/x/${key}`,
  body,
});

describe("outbox", () => {
  it("keeps only the latest write per key", async () => {
    const ob = setup();
    await ob.enqueue(entry("a", 1));
    await ob.enqueue(entry("a", 2));
    const list = await ob.pending();
    expect(list).toHaveLength(1);
    expect(list[0]?.body).toBe(2);
  });

  it("replays oldest first and removes sent and rejected entries", async () => {
    const ob = setup();
    await ob.enqueue(entry("a", 1));
    await ob.enqueue(entry("b", 2));
    const order: string[] = [];
    const { sent } = await ob.replay(async (e) => {
      order.push(e.key);
      return e.key === "a" ? "sent" : "rejected";
    });
    expect(order).toEqual(["a", "b"]);
    expect(sent.map((e) => e.key)).toEqual(["a"]);
    expect(await ob.pending()).toHaveLength(0);
  });

  it("stops on offline and keeps the rest", async () => {
    const ob = setup();
    await ob.enqueue(entry("a", 1));
    await ob.enqueue(entry("b", 2));
    await ob.replay(async () => "offline");
    expect(await ob.pending()).toHaveLength(2);
    await ob.replay(async () => {
      throw new Error("network");
    });
    expect(await ob.pending()).toHaveLength(2);
  });

  it("keeps a newer write queued while the older one was in flight", async () => {
    const ob = setup();
    await ob.enqueue(entry("a", 1));
    await ob.replay(async () => {
      await ob.enqueue(entry("a", 2));
      return "sent";
    });
    const list = await ob.pending();
    expect(list.map((e) => e.body)).toEqual([2]);
  });

  it("notifies subscribers with the pending count", async () => {
    const ob = setup();
    const counts: number[] = [];
    ob.subscribe((n) => counts.push(n));
    await ob.enqueue(entry("a", 1));
    await ob.replay(async () => "sent");
    expect(counts.at(-1)).toBe(0);
    expect(counts).toContain(1);
  });
});

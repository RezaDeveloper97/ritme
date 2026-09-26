import {
  createIdbStore,
  createMemoryStore,
  type OutboxEntry,
  type OutboxStore,
} from "./store";

/**
 * Offline outbox: writes that failed for lack of a network are queued and
 * replayed on reconnect. Domain-free — the caller decides the key, the URL
 * and how an entry is sent (§3: `shared` knows no domain).
 */

/** What `send` reports so replay can decide what to do with the entry. */
export type SendResult = "sent" | "offline" | "rejected";

export interface Outbox {
  enqueue(entry: Omit<OutboxEntry, "queuedAt">): Promise<void>;
  pending(): Promise<OutboxEntry[]>;
  /**
   * Send every queued entry, oldest first. `sent` and `rejected` (the server
   * refused it — retrying can never succeed) remove the entry; `offline` stops
   * the run and keeps the rest for the next reconnect. Concurrent calls share
   * one run, so an entry is never sent twice at once.
   */
  replay(
    send: (entry: OutboxEntry) => Promise<SendResult>,
  ): Promise<{ sent: OutboxEntry[] }>;
  subscribe(listener: (count: number) => void): () => void;
}

export function createOutbox(
  store: OutboxStore,
  now: () => number = Date.now,
): Outbox {
  const listeners = new Set<(count: number) => void>();
  let running: Promise<{ sent: OutboxEntry[] }> | null = null;

  const notify = async () => {
    if (listeners.size === 0) return;
    const count = (await store.all()).length;
    listeners.forEach((l) => l(count));
  };

  const pending = async () =>
    (await store.all()).sort((a, b) => a.queuedAt - b.queuedAt);

  const run = async (send: (entry: OutboxEntry) => Promise<SendResult>) => {
    const sent: OutboxEntry[] = [];
    for (const entry of await pending()) {
      let result: SendResult;
      try {
        result = await send(entry);
      } catch {
        result = "offline";
      }
      if (result === "offline") break;
      // A newer write to the same key may have been queued while this one was
      // in flight — only drop the entry if it is still the one we sent.
      const current = (await store.all()).find((e) => e.key === entry.key);
      if (current && current.queuedAt === entry.queuedAt)
        await store.delete(entry.key);
      if (result === "sent") sent.push(entry);
    }
    await notify();
    return { sent };
  };

  return {
    enqueue: async (entry) => {
      await store.put({ ...entry, queuedAt: now() });
      await notify();
    },
    pending,
    replay: (send) => {
      running ??= run(send).finally(() => {
        running = null;
      });
      return running;
    },
    subscribe: (listener) => {
      listeners.add(listener);
      void store.all().then((all) => listener(all.length));
      return () => listeners.delete(listener);
    },
  };
}

let shared: Outbox | null = null;

/** The app-wide outbox (IndexedDB, or memory where IndexedDB is missing). */
export function getOutbox(): Outbox {
  shared ??= createOutbox(
    typeof indexedDB !== "undefined"
      ? createIdbStore(indexedDB)
      : createMemoryStore(),
  );
  return shared;
}

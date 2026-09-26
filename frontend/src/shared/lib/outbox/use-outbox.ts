"use client";

import { useEffect, useRef, useState } from "react";

import { getOutbox, type SendResult } from "./outbox";
import type { OutboxEntry } from "./store";

/** Number of writes still waiting in the outbox (0 until it is read). */
export function useOutboxPending(): number {
  const [count, setCount] = useState(0);
  useEffect(() => getOutbox().subscribe(setCount), []);
  return count;
}

/**
 * Replays the outbox on mount and on every `online` event. `onSent` gets the
 * entries the server accepted, so the caller can refresh its caches.
 */
export function useOutboxReplay(
  send: (entry: OutboxEntry) => Promise<SendResult>,
  onSent?: (sent: OutboxEntry[]) => void,
): void {
  const ref = useRef({ send, onSent });
  ref.current = { send, onSent };
  useEffect(() => {
    const replay = () => {
      if (typeof navigator !== "undefined" && navigator.onLine === false)
        return;
      void getOutbox()
        .replay((e) => ref.current.send(e))
        .then(({ sent }) => {
          if (sent.length > 0) ref.current.onSent?.(sent);
        })
        .catch(() => undefined);
    };
    replay();
    window.addEventListener("online", replay);
    return () => window.removeEventListener("online", replay);
  }, []);
}

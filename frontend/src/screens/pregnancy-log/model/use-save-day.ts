"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";

import {
  type PregnancyAlertV2,
  type PregnancyDayInput,
  pregnancyKeys,
} from "@/entities/pregnancy";
import {
  toPregnancyDayBody,
  useSavePregnancyDay,
} from "@/features/track-pregnancy";
import { apiClient, getApiErrorStatus } from "@/shared/api";
import {
  getOutbox,
  type OutboxEntry,
  type SendResult,
  useOutboxPending,
  useOutboxReplay,
} from "@/shared/lib/outbox";

/** Outbox key of one day's log — a newer save of the same day replaces it. */
const dayKey = (date: string) => `pregnancy-day:${date}`;

export type SaveStatus = "idle" | "saving" | "saved" | "queued" | "error";

/** No HTTP response at all = the network, not the server, failed. */
const isOffline = (error: unknown) => getApiErrorStatus(error) === undefined;

async function sendEntry(entry: OutboxEntry): Promise<SendResult> {
  try {
    if (entry.method === "delete") await apiClient.delete(entry.url);
    else await apiClient[entry.method](entry.url, entry.body);
    return "sent";
  } catch (error) {
    // 5xx may pass on retry; 4xx (validation, not-in-pregnancy) never will.
    const status = getApiErrorStatus(error);
    return status === undefined || status >= 500 ? "offline" : "rejected";
  }
}

/**
 * Save a day online, or queue it in the offline outbox when there is no
 * network (T-M7-12). Queued days replay on reconnect; the PUT replaces the
 * whole day, so replaying it is idempotent.
 */
export function useSaveDay() {
  const queryClient = useQueryClient();
  const save = useSavePregnancyDay();
  const pending = useOutboxPending();
  const [status, setStatus] = useState<SaveStatus>("idle");
  const [alerts, setAlerts] = useState<PregnancyAlertV2[]>([]);

  useOutboxReplay(sendEntry, () => {
    void queryClient.invalidateQueries({ queryKey: pregnancyKeys.all });
  });

  const queue = useCallback(async (date: string, input: PregnancyDayInput) => {
    try {
      await getOutbox().enqueue({
        key: dayKey(date),
        method: "put",
        url: `/pregnancy/v2/days/${date}`,
        body: toPregnancyDayBody(input),
      });
      setStatus("queued");
    } catch {
      setStatus("error");
    }
  }, []);

  const submit = useCallback(
    async (date: string, input: PregnancyDayInput) => {
      setAlerts([]);
      if (typeof navigator !== "undefined" && navigator.onLine === false) {
        await queue(date, input);
        return;
      }
      setStatus("saving");
      try {
        const day = await save.mutateAsync({ date, input });
        setAlerts(day.alerts);
        setStatus("saved");
      } catch (error) {
        if (isOffline(error)) await queue(date, input);
        else setStatus("error");
      }
    },
    [queue, save],
  );

  const reset = useCallback(() => {
    setStatus((s) => (s === "saving" ? s : "idle"));
  }, []);

  return { submit, reset, status, alerts, pending };
}

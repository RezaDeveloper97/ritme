'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';

import { isConflict, kickApi, pregnancyToolKeys, putKickSession } from '../api/queries';
import type { KickSession } from './types';

/**
 * Taps on the big button, sent one request at a time so the count never jumps
 * back when answers arrive out of order. `pending` (+ taps, − undos) is shown on
 * top of the server count straight away. The first tap with no running session
 * starts one and counts as its first movement.
 */
export function useKickTaps(active: KickSession | null) {
  const queryClient = useQueryClient();
  const [pending, setPending] = useState(0);
  const [error, setError] = useState<unknown>(null);
  const pendingRef = useRef(0);
  const drainingRef = useRef(false);
  const startingRef = useRef(false);
  const sessionRef = useRef<number | null>(active?.id ?? null);

  useEffect(() => {
    sessionRef.current = active?.id ?? null;
  }, [active?.id]);

  const fail = useCallback(
    (e: unknown) => {
      pendingRef.current = 0;
      setPending(0);
      setError(e);
      if (isConflict(e)) void queryClient.invalidateQueries({ queryKey: pregnancyToolKeys.kicks() });
    },
    [queryClient],
  );

  const drain = useCallback(
    async (id: number) => {
      if (drainingRef.current) return;
      drainingRef.current = true;
      try {
        while (pendingRef.current !== 0) {
          const up = pendingRef.current > 0;
          const s = up ? await kickApi.kick(id) : await kickApi.undo(id);
          pendingRef.current += up ? -1 : 1;
          setPending(pendingRef.current);
          putKickSession(queryClient, s);
        }
        // Today's total and the history header follow the last answer.
        void queryClient.invalidateQueries({ queryKey: pregnancyToolKeys.kicks() });
      } catch (e) {
        fail(e);
      } finally {
        drainingRef.current = false;
      }
    },
    [fail, queryClient],
  );

  const tap = useCallback(async () => {
    setError(null);
    pendingRef.current += 1;
    setPending(pendingRef.current);
    const id = sessionRef.current;
    if (id !== null) return drain(id);
    if (startingRef.current) return; // the start in flight will send this tap too
    startingRef.current = true;
    try {
      const s = await kickApi.start();
      sessionRef.current = s.id;
      putKickSession(queryClient, s);
      await drain(s.id);
    } catch (e) {
      fail(e);
    } finally {
      startingRef.current = false;
    }
  }, [drain, fail, queryClient]);

  const undo = useCallback(
    (shown: number) => {
      const id = sessionRef.current;
      if (id === null || shown <= 0) return;
      setError(null);
      pendingRef.current -= 1;
      setPending(pendingRef.current);
      void drain(id);
    },
    [drain],
  );

  return { pending, error, busy: pending !== 0, tap, undo, clearError: () => setError(null) };
}

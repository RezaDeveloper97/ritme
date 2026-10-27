import { useQueries } from '@tanstack/react-query';
import { useMemo } from 'react';

import {
  cycleKeys,
  fetchCycleMonth,
  type CycleCalculation,
  type CycleFutureCycle,
  type CycleLimits,
  type CycleMonthDay,
} from '@/entities/cycle';
import { isAuthenticated } from '@/shared/session';
import { toApiDate } from '@/shared/lib/date';

import { gregorianMonthsBetween } from './months';

export interface CalcMap {
  map: Map<string, CycleCalculation>;
  /** v6 per-day paint data keyed by ISO date; empty for a Laravel / v1 backend. */
  days: Map<string, CycleMonthDay>;
  /** Whether the month payloads are v6 (they carry `period_kind` / `main_phase` per day). */
  isV6: boolean;
  /** v6 `future_cycles` (k ≤ 6) of the freshest month; empty for Laravel. */
  futureCycles: CycleFutureCycle[];
  /** v6 `limits` of the freshest month; null for Laravel. */
  limits: CycleLimits | null;
  /** First load of any month in range. */
  pending: boolean;
  fetching: boolean;
  error: boolean;
  refetch: () => void;
  /** Freshest fetch time across the months (for the optimistic overlay). */
  freshness: number;
}

/**
 * Every engine calculation between `first` and `last`, keyed by ISO date — a
 * displayed Jalali month spans two Gregorian ones, the year view thirteen.
 * Same cache entries as `useCycleMonth` (home reads them too).
 */
export function useCalcMap(first: Date, last: Date, enabled = true): CalcMap {
  const firstIso = toApiDate(first);
  const lastIso = toApiDate(last);
  const months = useMemo(
    () => gregorianMonthsBetween(first, last),
    // Dates are recreated every render; the ISO strings are the identity.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [firstIso, lastIso],
  );
  const results = useQueries({
    queries: months.map(({ year, month }) => ({
      queryKey: cycleKeys.month(year, month),
      queryFn: () => fetchCycleMonth(year, month),
      enabled: enabled && isAuthenticated(),
      staleTime: 5 * 60_000,
      retry: false,
    })),
  });

  const stamp = results.map((r) => r.dataUpdatedAt).join(',');
  const map = useMemo(() => {
    const m = new Map<string, CycleCalculation>();
    for (const r of results) for (const c of r.data?.calculations ?? []) m.set(c.calculationDate, c);
    return m;
    // `results` is a new array each render; the fetch stamps say when data changed.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stamp]);

  const v6 = useMemo(() => {
    const days = new Map<string, CycleMonthDay>();
    let futureCycles: CycleFutureCycle[] = [];
    let limits: CycleLimits | null = null;
    let freshest = -1;
    for (const r of results) {
      if (!r.data) continue;
      for (const d of r.data.days) days.set(d.date, d);
      // Every month answers with today's future cycles; the newest answer wins.
      if (r.dataUpdatedAt > freshest && (r.data.days.length > 0 || r.data.limits)) {
        freshest = r.dataUpdatedAt;
        futureCycles = r.data.futureCycles;
        limits = r.data.limits;
      }
    }
    return { days, futureCycles, limits };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stamp]);

  return {
    map,
    days: v6.days,
    isV6: v6.days.size > 0,
    futureCycles: v6.futureCycles,
    limits: v6.limits,
    pending: results.some((r) => r.isPending && r.fetchStatus !== 'idle'),
    fetching: results.some((r) => r.isFetching),
    error: results.some((r) => r.isError),
    refetch: () => results.forEach((r) => void r.refetch()),
    freshness: Math.max(0, ...results.map((r) => r.dataUpdatedAt)),
  };
}

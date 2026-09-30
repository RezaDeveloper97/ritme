import { useQueries } from '@tanstack/react-query';
import { useMemo } from 'react';

import { cycleKeys, fetchCycleMonth, type CycleCalculation } from '@/entities/cycle';
import { isAuthenticated } from '@/shared/session';
import { toApiDate } from '@/shared/lib/date';

import { gregorianMonthsBetween } from './months';

export interface CalcMap {
  map: Map<string, CycleCalculation>;
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

  return {
    map,
    pending: results.some((r) => r.isPending && r.fetchStatus !== 'idle'),
    fetching: results.some((r) => r.isFetching),
    error: results.some((r) => r.isError),
    refetch: () => results.forEach((r) => void r.refetch()),
    freshness: Math.max(0, ...results.map((r) => r.dataUpdatedAt)),
  };
}

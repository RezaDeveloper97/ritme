import type { QueryClient } from '@tanstack/react-query';

/**
 * Every query root that reads the cycle engine's output. After any write that
 * can move a prediction (period, day log, fertility log, profile lengths, cycle
 * settings, contraception) all of them go stale together: the fertility window
 * and chance, the regularity analysis, the daily message and the before-period
 * care reminders all derive from the same cycle view.
 *
 * `shared` cannot import the entity key factories, so these are literals;
 * `invalidation.test.ts` asserts they equal the factories' `all` keys.
 */
export const CYCLE_DEPENDENT_ROOTS: readonly (readonly string[])[] = [
  ['cycle'],
  ['fertility'],
  ['analysis'],
  ['message'],
  ['care'],
  ['period-status'],
  ['period-history'],
];

/** Invalidate every cycle-dependent query root (see {@link CYCLE_DEPENDENT_ROOTS}). */
export function invalidateCycleDependents(queryClient: QueryClient): void {
  for (const root of CYCLE_DEPENDENT_ROOTS) {
    void queryClient.invalidateQueries({ queryKey: [...root] });
  }
}

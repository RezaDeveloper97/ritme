import type { BbtRange } from '../model/types';

/**
 * Query-key factory for `/fertility/*` (CLAUDE.md §8). `features/log-fertility-day`
 * invalidates through these — never hand-written arrays.
 *
 * Saving a day moves the tiles, the chance level, the BBT chart and the
 * insights at once, so the mutation simply invalidates {@link fertilityKeys.all}.
 * The `*All()` keys are prefixes for finer-grained refreshes.
 */
export const fertilityKeys = {
  all: ['fertility'] as const,
  /** `date` omitted = "today" as the server resolves it (Tehran). */
  today: () => [...fertilityKeys.all, 'today'] as const,
  daysAll: () => [...fertilityKeys.all, 'day'] as const,
  day: (date: string) => [...fertilityKeys.daysAll(), date] as const,
  bbtAll: () => [...fertilityKeys.all, 'bbt'] as const,
  bbt: (range: BbtRange = 1) => [...fertilityKeys.bbtAll(), range] as const,
  insights: () => [...fertilityKeys.all, 'insights'] as const,
};

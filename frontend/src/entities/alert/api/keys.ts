import type { AlertListState } from '../model/types';

/**
 * Query keys for `/alerts` (CLAUDE.md §8). The prefix `['alerts']` is shared with `entities/message`'s
 * `useAckAlert`, so acknowledging from the home invalidates this list too.
 */
export const alertKeys = {
  all: ['alerts'] as const,
  list: (state: AlertListState) => [...alertKeys.all, 'list', state] as const,
};

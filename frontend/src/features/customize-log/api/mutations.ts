'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  addLogCustomItem,
  deleteLogCustomItem,
  type LogCustomItem,
  type LogPreferences,
  type LogPreferencesChanges,
  logKeys,
  renameLogCustomItem,
  resetLogPreferences,
  saveLogPreferences,
} from '@/entities/health-log';

/**
 * Writes of the customisation screen (B-N3-04). Preferences and custom items are optimistic on the
 * current-mode cache entry (the one the log sheet reads) and roll back on failure; every write ends by
 * invalidating all preferences entries so the log sheet re-reads them.
 */

type Ctx = { previous?: LogPreferences };

function usePrefsCache(mode?: string) {
  const queryClient = useQueryClient();
  const key = logKeys.preferences(mode);
  return {
    async snapshot(next?: (p: LogPreferences) => LogPreferences): Promise<Ctx> {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<LogPreferences>(key);
      if (previous && next) queryClient.setQueryData(key, next(previous));
      return { previous };
    },
    rollback(ctx?: Ctx) {
      if (ctx?.previous) queryClient.setQueryData(key, ctx.previous);
    },
    set(prefs: LogPreferences) {
      queryClient.setQueryData(key, prefs);
    },
    settle() {
      void queryClient.invalidateQueries({ queryKey: logKeys.preferencesAll() });
    },
  };
}

export interface SavePrefsVars {
  changes: LogPreferencesChanges;
  /** The preferences as they will read after the save (optimistic cache entry). */
  optimistic: LogPreferences;
}

/** One PUT with only the lists that changed. */
export function useSaveLogPreferences(mode?: string) {
  const cache = usePrefsCache(mode);
  return useMutation<LogPreferences, unknown, SavePrefsVars, Ctx>({
    mutationFn: ({ changes }) => saveLogPreferences(changes, mode),
    onMutate: ({ optimistic }) => cache.snapshot(() => optimistic),
    onError: (_e, _v, ctx) => cache.rollback(ctx),
    onSuccess: (prefs) => cache.set(prefs),
    onSettled: () => cache.settle(),
  });
}

/** DELETE — back to the mode's defaults (custom items stay). */
export function useResetLogPreferences(mode?: string) {
  const cache = usePrefsCache(mode);
  return useMutation<LogPreferences, unknown, void, Ctx>({
    mutationFn: () => resetLogPreferences(mode),
    onMutate: () => cache.snapshot(),
    onError: (_e, _v, ctx) => cache.rollback(ctx),
    onSuccess: (prefs) => cache.set(prefs),
    onSettled: () => cache.settle(),
  });
}

export function useAddLogCustomItem(mode?: string) {
  const cache = usePrefsCache(mode);
  return useMutation<LogCustomItem, unknown, { category: string; label: string }, Ctx>({
    mutationFn: ({ category, label }) => addLogCustomItem(category, label),
    onSuccess: async (item) => {
      // Shown at once; the refetch below confirms it.
      await cache.snapshot((p) => ({ ...p, customItems: [...p.customItems, item] }));
    },
    onSettled: () => cache.settle(),
  });
}

export function useRenameLogCustomItem(mode?: string) {
  const cache = usePrefsCache(mode);
  return useMutation<LogCustomItem, unknown, { id: number; label: string }, Ctx>({
    mutationFn: ({ id, label }) => renameLogCustomItem(id, label),
    onMutate: ({ id, label }) =>
      cache.snapshot((p) => ({ ...p, customItems: p.customItems.map((i) => (i.id === id ? { ...i, label } : i)) })),
    onError: (_e, _v, ctx) => cache.rollback(ctx),
    onSettled: () => cache.settle(),
  });
}

export function useDeleteLogCustomItem(mode?: string) {
  const cache = usePrefsCache(mode);
  return useMutation<void, unknown, number, Ctx>({
    mutationFn: (id) => deleteLogCustomItem(id),
    onMutate: (id) => cache.snapshot((p) => ({ ...p, customItems: p.customItems.filter((i) => i.id !== id) })),
    onError: (_e, _v, ctx) => cache.rollback(ctx),
    onSettled: () => cache.settle(),
  });
}

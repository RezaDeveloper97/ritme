import { create } from 'zustand';

import type { LogDayValues } from '@/entities/health-log';
import { onSessionEnd } from '@/shared/session';

/**
 * Unsaved log edits, per day, kept outside the sheet.
 *
 * The system Back closes the log sheet (or leaves `/log`) like × does, and a
 * Back gesture is easy to make by accident mid-edit. So the draft outlives the
 * component: reopening the same day shows the edits again, with «ذخیره» still
 * pending. Memory only — health data is never written to storage for this —
 * and wiped when the session ends, so it can't reach the next account.
 */
interface LogDraftState {
  values: Record<string, LogDayValues>;
  /** `category.param` keys taken from a confirmed voice suggestion, per draft. */
  voice: Record<string, string[]>;
  setValues: (key: string, next: (current: LogDayValues | undefined) => LogDayValues) => void;
  addVoice: (key: string, keys: readonly string[]) => void;
  clear: (key: string) => void;
  reset: () => void;
}

function without<T>(all: Record<string, T>, key: string): Record<string, T> {
  const next = { ...all };
  delete next[key];
  return next;
}

export const useLogDraftStore = create<LogDraftState>((set) => ({
  values: {},
  voice: {},
  setValues: (key, next) => set((s) => ({ values: { ...s.values, [key]: next(s.values[key]) } })),
  addVoice: (key, keys) =>
    set((s) => ({ voice: { ...s.voice, [key]: [...new Set([...(s.voice[key] ?? []), ...keys])] } })),
  clear: (key) => set((s) => ({ values: without(s.values, key), voice: without(s.voice, key) })),
  reset: () => set({ values: {}, voice: {} }),
}));

/** A draft belongs to a day as seen by one log preset (the menopause page has its own taxonomy). */
export const draftKey = (date: string, modeOverride?: string): string => `${modeOverride ?? '*'}|${date}`;

if (typeof window !== 'undefined') onSessionEnd(() => useLogDraftStore.getState().reset());

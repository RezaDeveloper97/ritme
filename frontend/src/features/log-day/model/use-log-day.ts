'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useCallback, useMemo, useState } from 'react';

import { useMedications } from '@/entities/care-reminder';
import {
  useLogDay,
  useLogPreferences,
  useLogTaxonomy,
  type LogCategory,
  type LogDayValues,
  type LogOption,
  type LogParamValue,
} from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';

import { useSaveLogDay } from '../api/save';
import { dayEntries, diffDay, hasChanges, setValue, type LabelContext, type LogEntry } from './draft';

const EMPTY: LogDayValues = {};

export interface LogDayController {
  mode: string | null;
  /** Visible categories in the user's order (hidden ones dropped). */
  categories: LogCategory[];
  /** Quick tile keys (≤ 8) that exist in the visible categories. */
  tiles: string[];
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  /** Marks `category.param` keys as taken from a confirmed voice-log suggestion (saved as `source: voice`). */
  markVoice: (keys: readonly string[]) => void;
  /** Extra pickable items of a param: the user's custom items, or her medications for `meds.taken`. */
  extraOptions: (category: string, param: string) => LogOption[];
  entries: LogEntry[];
  labels: LabelContext;
  dirty: boolean;
  /** One PUT with what changed; `onDone` runs after the server accepted it. */
  save: (onDone?: () => void) => void;
  saving: boolean;
  saveError: boolean;
  justSaved: boolean;
  loading: boolean;
  error: boolean;
  retry: () => void;
}

/**
 * State of the log sheet for one day: taxonomy + preferences + the saved day (server state, TanStack
 * Query) and the local draft per date. Edits stay local until «ذخیره», which sends one partial PUT.
 */
export function useLogDayController(date: string, modeOverride?: string): LogDayController {
  const t = useTranslations('logSheet');
  const locale = useLocale() as Locale;
  const taxonomy = useLogTaxonomy(modeOverride);
  const prefs = useLogPreferences(modeOverride);
  const day = useLogDay(date);
  const meds = useMedications();
  const save = useSaveLogDay();
  const [drafts, setDrafts] = useState<Record<string, LogDayValues>>({});
  const [savedAt, setSavedAt] = useState<string | null>(null);
  const [voiceKeys, setVoiceKeys] = useState<Record<string, string[]>>({});

  const saved = day.data?.categories ?? EMPTY;
  const values = drafts[date] ?? saved;

  const mode = taxonomy.data?.mode ?? prefs.data?.mode ?? modeOverride ?? null;

  const categories = useMemo(() => {
    const custom = prefs.data?.customItems ?? [];
    // A category whose only params host the user's custom items shows once she has some (B-N3-04 adds them).
    const all = (taxonomy.data?.categories ?? []).filter((c) =>
      c.params.some(
        (p) =>
          (p.type !== 'items' && p.type !== 'multi') ||
          p.options.length > 0 ||
          (c.code === 'meds' && p.code === 'taken') ||
          custom.some((i) => i.category === c.code && i.param === p.code),
      ),
    );
    const order = prefs.data?.categories;
    if (!order?.length) return all;
    const byCode = new Map(all.map((c) => [c.code, c]));
    const out: LogCategory[] = [];
    for (const p of order) {
      const c = byCode.get(p.code);
      if (c && !p.hidden) out.push(c);
      byCode.delete(p.code);
    }
    // A category the preferences don't know yet (a newer taxonomy) follows in registry order.
    for (const c of all) if (byCode.has(c.code) && !order.some((p) => p.code === c.code)) out.push(c);
    return out;
  }, [taxonomy.data, prefs.data]);

  const tiles = useMemo(() => {
    const visible = new Set(categories.map((c) => c.code));
    return (prefs.data?.pinned ?? []).filter((k) => visible.has(k.split('.')[0]));
  }, [prefs.data, categories]);

  const extraLabels = useMemo(() => {
    const out: Record<string, string> = {};
    for (const c of prefs.data?.customItems ?? []) out[c.code] = c.label;
    for (const m of meds.data ?? []) out[String(m.id)] = m.title;
    return out;
  }, [prefs.data, meds.data]);

  const labels = useMemo<LabelContext>(
    () => ({
      extraLabels,
      unknown: t('unknownValue'),
      formatNumber: (n, scale) => formatDecimal(n.toFixed(scale), locale),
    }),
    [extraLabels, t, locale],
  );

  const extraOptions = useCallback(
    (category: string, param: string): LogOption[] => {
      if (category === 'meds' && param === 'taken') {
        const active = (meds.data ?? []).filter((m) => m.isActive);
        const stored = Object.keys((values.meds?.taken as Record<string, unknown> | undefined) ?? {});
        const list = active.map((m) => ({ value: String(m.id), label: m.title, modes: null, legacyOnly: false }));
        // A taken dose of a since-deleted reminder stays visible (and can be unticked).
        for (const code of stored) {
          if (!list.some((o) => o.value === code)) list.push({ value: code, label: t('meds.removed'), modes: null, legacyOnly: false });
        }
        return list;
      }
      return (prefs.data?.customItems ?? [])
        .filter((c) => c.category === category && c.param === param)
        .map((c) => ({ value: c.code, label: c.label, modes: null, legacyOnly: false }));
    },
    [meds.data, prefs.data, values, t],
  );

  const setParam = useCallback(
    (category: string, param: string, value: LogParamValue | null) => {
      setSavedAt(null);
      setDrafts((all) => ({ ...all, [date]: setValue(all[date] ?? saved, category, param, value) }));
    },
    [date, saved],
  );

  const markVoice = useCallback(
    (keys: readonly string[]) =>
      setVoiceKeys((all) => ({ ...all, [date]: [...new Set([...(all[date] ?? []), ...keys])] })),
    [date],
  );

  const changes = useMemo(() => diffDay(saved, values), [saved, values]);
  const dirty = hasChanges(changes);
  const entries = useMemo(() => dayEntries(categories, values, labels), [categories, values, labels]);

  const doSave = useCallback((onDone?: () => void) => {
    if (!dirty || save.isPending) return;
    const draft = values;
    // Only voice params that this save actually sets (a cleared one is a manual edit).
    const voiceParams = (voiceKeys[date] ?? []).filter((key) => {
      const [cat, param] = key.split('.');
      const v = changes[cat]?.[param];
      return v !== undefined && v !== null;
    });
    save.mutate(
      { date, changes, draft, voiceParams },
      {
        onSuccess: () => {
          setDrafts((all) => {
            const next = { ...all };
            delete next[date];
            return next;
          });
          setVoiceKeys((all) => {
            const next = { ...all };
            delete next[date];
            return next;
          });
          setSavedAt(date);
          onDone?.();
        },
      },
    );
  }, [dirty, save, values, date, changes, voiceKeys]);

  return {
    mode,
    categories,
    tiles,
    values,
    setParam,
    markVoice,
    extraOptions,
    entries,
    labels,
    dirty,
    save: doSave,
    saving: save.isPending,
    saveError: save.isError,
    justSaved: savedAt === date && !dirty,
    loading: taxonomy.isPending || prefs.isPending || day.isPending,
    error: taxonomy.isError || day.isError,
    retry: () => {
      void taxonomy.refetch();
      void prefs.refetch();
      void day.refetch();
    },
  };
}

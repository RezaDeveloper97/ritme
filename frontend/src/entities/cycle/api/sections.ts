'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { cycleKeys } from './queries';

/**
 * The cycle-history cards on the home page, served by the backend's
 * server-driven sections (`GET /home/sections/{key}`).
 *
 * Most of the page renders from `/cycle/today`, but these two need facts only
 * the backend can derive — how long each past cycle actually ran, and how those
 * numbers read against the usual range. The sections own that logic, so the
 * client just renders what they return (§8: server state lives in TanStack
 * Query, never copied into local state).
 *
 * Their cache keys hang off {@link cycleKeys} so logging or editing a period —
 * which already invalidates `cycleKeys.all` — refreshes these cards too.
 */

/** Query key for one home section, nested under the cycle entity (§8). */
const homeSectionKey = (key: string) => [...cycleKeys.all, 'home-section', key] as const;

const sectionActionSchema = z.object({ key: z.string(), label: z.string() });

/** One recorded cycle: its bleed, and the length it ran until the next period. */
const cycleRecordSchema = z.object({
  id: z.number(),
  period_start_date: z.string(),
  period_end_date: z.string().nullable(),
  /** Days from this period's start to the next one — null while it's the current cycle. */
  cycle_length: z.number().nullable(),
  /** Days of bleeding — null while the period is still open. */
  period_length: z.number().nullable(),
  is_ongoing: z.boolean(),
  is_current: z.boolean(),
  is_confirmed: z.boolean(),
  is_estimated: z.boolean(),
  source: z.string().nullable(),
});

export type CycleRecord = z.infer<typeof cycleRecordSchema>;

const myCyclesDataSchema = z.object({
  current: z.object({
    id: z.number().nullable(),
    cycle_day: z.number(),
    started_at: z.string(),
    period_end_date: z.string().nullable(),
    period_length: z.number().nullable(),
    is_ongoing: z.boolean(),
    cycle_length: z.number().nullable(),
    cycle_length_source: z.string(),
  }),
  previous_count: z.number(),
  previous: z.array(cycleRecordSchema),
  averages: z.object({
    cycle_length: z.number().nullable(),
    period_length: z.number().nullable(),
    based_on_cycles: z.number(),
  }),
});

export type MyCyclesData = z.infer<typeof myCyclesDataSchema>;

/** How one summary row reads against the range that is typical for most people. */
export const summaryStatuses = ['normal', 'outside_range', 'unknown'] as const;

const summaryItemSchema = z.object({
  key: z.string(),
  label: z.string(),
  /** Raw number so the client formats digits per locale (§7); null when unknown. */
  value: z.number().nullable(),
  value_min: z.number().nullable(),
  value_max: z.number().nullable(),
  /** Pre-worded value for rows that aren't a number (e.g. regularity). */
  text: z.string().nullable(),
  unit: z.enum(['days']).nullable(),
  unit_label: z.string().nullable(),
  status: z.enum(summaryStatuses),
  status_label: z.string(),
  hint: z.string().nullable(),
  normal_range: z.object({ min: z.number(), max: z.number() }).nullable(),
});

export type CycleSummaryItem = z.infer<typeof summaryItemSchema>;

const cycleSummaryDataSchema = z.object({
  items: z.array(summaryItemSchema),
  based_on_cycles: z.number(),
  has_history: z.boolean(),
  normal_ranges: z.object({
    cycle_length: z.object({ min: z.number(), max: z.number() }),
    period_duration: z.object({ min: z.number(), max: z.number() }),
  }),
});

export type CycleSummaryData = z.infer<typeof cycleSummaryDataSchema>;

/** The envelope every section shares; only `data` differs per section. */
const sectionEnvelope = {
  key: z.string(),
  type: z.string(),
  title: z.string().nullable(),
  subtitle: z.string().nullable(),
  order: z.number(),
  action: sectionActionSchema.nullable(),
};

const myCyclesSectionSchema = z.object({ ...sectionEnvelope, data: myCyclesDataSchema });
const cycleSummarySectionSchema = z.object({ ...sectionEnvelope, data: cycleSummaryDataSchema });

export type HomeSection<T> = Omit<z.infer<typeof myCyclesSectionSchema>, 'data'> & { data: T };

/**
 * Read one section's raw payload. The backend returns `section: null` when the
 * user has no data for it yet (or the section doesn't apply to their mode) —
 * that's an empty state, not an error, so it resolves to `null` rather than
 * throwing.
 */
async function fetchSection(key: string): Promise<unknown> {
  const { data } = await apiClient.get<ApiEnvelope<{ section: unknown }>>(`/home/sections/${key}`);
  return data.data?.section ?? null;
}

/** GET /home/sections/my_cycles — current cycle + the recorded ones behind it. */
export function useMyCyclesSection() {
  return useQuery({
    queryKey: homeSectionKey('my_cycles'),
    queryFn: async () => {
      const section = await fetchSection('my_cycles');
      return section === null ? null : myCyclesSectionSchema.parse(section);
    },
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /home/sections/cycle_summary — the user's numbers vs. the usual range. */
export function useCycleSummarySection() {
  return useQuery({
    queryKey: homeSectionKey('cycle_summary'),
    queryFn: async () => {
      const section = await fetchSection('cycle_summary');
      return section === null ? null : cycleSummarySectionSchema.parse(section);
    },
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  MenopausePatterns,
  MenopauseScoreEntry,
  MenopauseScoreHistory,
  MenopauseScoreInput,
  MenopauseScoreQuestion,
} from '../model/score';
import { menopauseKeys } from './keys';

/*
 * `GET|POST /menopause/scores`, `GET /menopause/patterns` (CB-MENO-02) and the
 * questionnaire's catalog group (CB-CORE-03). Lenient on display-only fields.
 * Health data (§11): never log a payload or a response.
 */

const nullableText = z.string().nullable().catch(null);
const nullableInt = z.number().int().nullable().catch(null);

const bandSchema = z.object({
  code: z.string(),
  title: nullableText,
  min: z.number().int().catch(0),
  max: z.number().int().catch(0),
});

export const menopauseScoreEntrySchema = z
  .object({
    month: z.string(),
    total: z.number().int(),
    max: z.number().int(),
    band: bandSchema.nullable().catch(null),
    domains: z
      .array(z.object({ code: z.string(), score: z.number().int(), max: z.number().int() }))
      .catch([]),
    answers: z.record(z.string(), z.number().int()).catch({}),
    delta: nullableInt,
  })
  .transform((d): MenopauseScoreEntry => d);

export const menopauseScoreHistorySchema = z
  .object({
    months: z.number().int().catch(6),
    max: z.number().int().catch(44),
    bands: z.array(bandSchema).catch([]),
    latest: menopauseScoreEntrySchema.nullable().catch(null),
    trend: z
      .array(z.object({ month: z.string(), total: nullableInt, band: nullableText }))
      .catch([]),
    items: z.array(menopauseScoreEntrySchema).catch([]),
    hrt: z
      .object({
        started_on: z.string(),
        baseline_total: nullableInt,
        latest_total: nullableInt,
        change: nullableInt,
      })
      .nullable()
      .catch(null),
  })
  .transform(
    (d): MenopauseScoreHistory => ({
      months: d.months,
      max: d.max,
      bands: d.bands,
      latest: d.latest,
      trend: d.trend,
      items: d.items,
      hrt: d.hrt
        ? {
            startedOn: d.hrt.started_on,
            baselineTotal: d.hrt.baseline_total,
            latestTotal: d.hrt.latest_total,
            change: d.hrt.change,
          }
        : null,
    }),
  );

export const menopausePatternsSchema = z
  .object({
    days_logged: z.number().int().catch(0),
    min_days: z.number().int().catch(20),
    found: z.number().int().catch(0),
    disclaimer: z.object({ title: nullableText, body: nullableText }).nullable().catch(null),
    items: z
      .array(
        z.object({
          key: z.string(),
          trigger: nullableText,
          found: z.boolean().catch(false),
          text: nullableText,
        }),
      )
      .catch([]),
  })
  .transform(
    (d): MenopausePatterns => ({
      daysLogged: d.days_logged,
      minDays: d.min_days,
      found: d.found,
      disclaimer: d.disclaimer,
      items: d.items,
    }),
  );

/** `meno_score_items` → questions in catalog order; items without a title are dropped. */
export const menopauseScoreQuestionsSchema = z
  .object({
    items: z
      .array(
        z.object({
          code: z.string(),
          title: nullableText,
          body: nullableText,
          meta: z
            .object({ domain: z.string().catch('somatic'), max: z.number().int().catch(4) })
            .nullable()
            .catch(null),
        }),
      )
      .catch([]),
  })
  .transform((d): MenopauseScoreQuestion[] =>
    d.items.flatMap((it) =>
      it.title
        ? [{ code: it.code, title: it.title, body: it.body, domain: it.meta?.domain ?? 'somatic', max: it.meta?.max ?? 4 }]
        : [],
    ),
  );

/** GET /menopause/scores?months= — the screen's header, domains, chart and HRT note. */
export function useMenopauseScores(months = 6) {
  return useQuery({
    queryKey: menopauseKeys.scores(months),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/scores', { params: { months } });
      return menopauseScoreHistorySchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /menopause/patterns — 90-day associations in her own logs (never a diagnosis). */
export function useMenopausePatterns() {
  return useQuery({
    queryKey: menopauseKeys.patterns(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/patterns');
      return menopausePatternsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /catalog/meno_score_items — the 11 questions (admin-editable, request locale). */
export function useMenopauseScoreQuestions() {
  return useQuery({
    queryKey: menopauseKeys.scoreQuestions(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/catalog/meno_score_items', {
        params: { audience: 'menopause' },
      });
      return menopauseScoreQuestionsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** POST /menopause/scores — saves (or replaces) this month's questionnaire; the score + home refetch. */
export function useSaveMenopauseScore() {
  const queryClient = useQueryClient();
  return useMutation<MenopauseScoreEntry, unknown, MenopauseScoreInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/menopause/scores', { answers: input.answers });
      return menopauseScoreEntrySchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [...menopauseKeys.all, 'scores'] });
      void queryClient.invalidateQueries({ queryKey: menopauseKeys.today() });
      void queryClient.invalidateQueries({ queryKey: menopauseKeys.messages() });
    },
  });
}

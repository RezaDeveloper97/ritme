'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  SIDE_EFFECT_CODES,
  type SideEffectCode,
  TREATMENT_GOAL_UNITS,
  TREATMENT_SCHEDULES,
  type TreatmentItem,
  type TreatmentItemInput,
  type TreatmentScreen,
  type TreatmentTip,
  toTreatmentItemBody,
} from '../model/treatment';
import { menopauseKeys } from './keys';

/*
 * `/menopause/treatment*` (CB-MENO-03), Go only. Every write answers with the
 * screen of the week it touched, so the cache is replaced from one body.
 * Health data (§11): never log a payload or a response.
 */

const nullableText = z.string().nullable().catch(null);
const nullableInt = z.number().int().nullable().catch(null);
const codeList = z
  .array(z.string())
  .catch([])
  .transform((codes) => codes.filter((c): c is SideEffectCode => (SIDE_EFFECT_CODES as readonly string[]).includes(c)));

const itemSchema = z
  .object({
    id: z.number().int(),
    kind: z.enum(['hrt', 'supplement', 'lifestyle']),
    name: z.string(),
    dose: nullableText,
    schedule: z.enum(TREATMENT_SCHEDULES).nullable().catch(null),
    form: nullableText,
    started_on: nullableText,
    review_on: nullableText,
    stopped_on: nullableText,
    active: z.boolean().catch(true),
    taken_today: z.boolean().catch(false),
    week: z
      .array(z.object({ date: z.string(), taken: z.boolean().catch(false), amount: nullableInt }))
      .catch([]),
    days_taken: z.number().int().catch(0),
    days: z.number().int().catch(0),
    adherence_pct: nullableInt,
    goal: z
      .object({
        target: z.number().int(),
        unit: z.enum(TREATMENT_GOAL_UNITS),
        amount: z.number().int().catch(0),
        done: z.boolean().catch(false),
      })
      .nullable()
      .catch(null),
    reminder: z
      .object({ id: z.number().int(), time: z.string(), notify: z.boolean().catch(true), active: z.boolean().catch(true) })
      .nullable()
      .catch(null),
  })
  .transform(
    (d): TreatmentItem => ({
      id: d.id,
      kind: d.kind,
      name: d.name,
      dose: d.dose,
      schedule: d.schedule,
      form: d.form,
      startedOn: d.started_on,
      reviewOn: d.review_on,
      stoppedOn: d.stopped_on,
      active: d.active,
      takenToday: d.taken_today,
      week: d.week,
      daysTaken: d.days_taken,
      days: d.days,
      adherencePct: d.adherence_pct,
      goal: d.goal,
      reminder: d.reminder,
    }),
  );

/** Unknown kinds or broken rows are dropped rather than failing the screen. */
const itemList = z
  .array(z.unknown())
  .catch([])
  .transform((rows) =>
    rows.flatMap((row) => {
      const parsed = itemSchema.safeParse(row);
      return parsed.success ? [parsed.data] : [];
    }),
  );

const tipSchema = z
  .object({
    code: z.string(),
    title: nullableText,
    body: nullableText,
    meta: z
      .object({
        placement: nullableText,
        weekly_goal: nullableInt,
        goal_unit: z.enum(TREATMENT_GOAL_UNITS).nullable().catch(null),
      })
      .partial()
      .nullable()
      .catch(null),
  })
  .transform(
    (d): TreatmentTip => ({
      code: d.code,
      title: d.title,
      body: d.body,
      placement: d.meta?.placement ?? null,
      weeklyGoal: d.meta?.weekly_goal ?? null,
      goalUnit: d.meta?.goal_unit ?? null,
    }),
  );

export const treatmentScreenSchema = z
  .object({
    date: z.string(),
    week: z.object({ from: z.string(), to: z.string() }),
    items: z
      .object({ hrt: itemList, supplement: itemList, lifestyle: itemList })
      .partial()
      .catch({}),
    stopped: itemList,
    review: z
      .object({ on: z.string(), item_id: nullableInt, suggested: z.boolean().catch(false) })
      .nullable()
      .catch(null),
    side_effects: z
      .object({
        codes: codeList,
        today: codeList,
        week: z.array(z.object({ date: z.string(), codes: codeList })).catch([]),
      })
      .catch({ codes: [...SIDE_EFFECT_CODES], today: [], week: [] }),
    tips: z
      .array(z.unknown())
      .catch([])
      .transform((rows) =>
        rows.flatMap((row) => {
          const parsed = tipSchema.safeParse(row);
          return parsed.success ? [parsed.data] : [];
        }),
      ),
  })
  .transform(
    (d): TreatmentScreen => ({
      date: d.date,
      week: d.week,
      items: { hrt: d.items.hrt ?? [], supplement: d.items.supplement ?? [], lifestyle: d.items.lifestyle ?? [] },
      stopped: d.stopped,
      review: d.review ? { on: d.review.on, itemId: d.review.item_id, suggested: d.review.suggested } : null,
      sideEffects: {
        codes: d.side_effects.codes.length ? d.side_effects.codes : [...SIDE_EFFECT_CODES],
        today: d.side_effects.today,
        week: d.side_effects.week,
      },
      tips: d.tips,
    }),
  );

const parseScreen = (body: ApiEnvelope<unknown>): TreatmentScreen => treatmentScreenSchema.parse(body.data);

export async function fetchTreatment(date: string | null): Promise<TreatmentScreen> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/treatment', {
    params: date ? { date } : undefined,
  });
  return parseScreen(data);
}

/** GET /menopause/treatment — the week of `date` (null = this week). */
export function useTreatment(date: string | null = null) {
  return useQuery({
    queryKey: menopauseKeys.treatment(date),
    queryFn: () => fetchTreatment(date),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** A write's screen replaces this week's cache when it is this week; the home's card refetches. */
function useApplyScreen() {
  const queryClient = useQueryClient();
  return (screen: TreatmentScreen) => {
    const current = queryClient.getQueryData<TreatmentScreen>(menopauseKeys.treatment(null));
    if (!current || current.week.from === screen.week.from) {
      queryClient.setQueryData(menopauseKeys.treatment(null), screen);
    }
    void queryClient.invalidateQueries({ queryKey: menopauseKeys.treatments(), predicate: (q) => q.queryKey[3] !== 'today' });
    void queryClient.invalidateQueries({ queryKey: menopauseKeys.today() });
    void queryClient.invalidateQueries({ queryKey: menopauseKeys.messages() });
  };
}

/** POST (no id) or PUT (id: full replace, `stoppedOn` stops it) /menopause/treatment/items. */
export function useSaveTreatmentItem() {
  const apply = useApplyScreen();
  return useMutation<TreatmentScreen, unknown, { id: number | null; input: TreatmentItemInput }>({
    mutationFn: async ({ id, input }) => {
      const body = toTreatmentItemBody(input, id === null);
      const { data } =
        id === null
          ? await apiClient.post<ApiEnvelope<unknown>>('/menopause/treatment/items', body)
          : await apiClient.put<ApiEnvelope<unknown>>(`/menopause/treatment/items/${id}`, body);
      return parseScreen(data);
    },
    onSuccess: apply,
  });
}

/** DELETE /menopause/treatment/items/{id} — the item, its intakes and its care reminder. */
export function useDeleteTreatmentItem() {
  const apply = useApplyScreen();
  return useMutation<TreatmentScreen, unknown, number>({
    mutationFn: async (id) => {
      const { data } = await apiClient.delete<ApiEnvelope<unknown>>(`/menopause/treatment/items/${id}`);
      return parseScreen(data);
    },
    onSuccess: apply,
  });
}

/**
 * PUT / DELETE /menopause/treatment/items/{id}/intakes/{date} — taken or not
 * that day (≤ 30 days back). Lifestyle minutes need `amount`; sessions may send a count.
 */
export function useTreatmentIntake() {
  const apply = useApplyScreen();
  return useMutation<TreatmentScreen, unknown, { id: number; date: string; taken: boolean; amount?: number | null }>({
    mutationFn: async ({ id, date, taken, amount }) => {
      const url = `/menopause/treatment/items/${id}/intakes/${date}`;
      const { data } = taken
        ? await apiClient.put<ApiEnvelope<unknown>>(url, amount ? { amount } : {})
        : await apiClient.delete<ApiEnvelope<unknown>>(url);
      return parseScreen(data);
    },
    onSuccess: apply,
  });
}

/** PUT /menopause/treatment/side-effects/{date} — the day's codes as a set (empty clears). */
export function useSaveSideEffects() {
  const apply = useApplyScreen();
  return useMutation<TreatmentScreen, unknown, { date: string; codes: SideEffectCode[] }>({
    mutationFn: async ({ date, codes }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/menopause/treatment/side-effects/${date}`, { codes });
      return parseScreen(data);
    },
    onSuccess: apply,
  });
}

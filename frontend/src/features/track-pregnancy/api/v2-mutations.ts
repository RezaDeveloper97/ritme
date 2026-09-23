'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { messageKeys } from '@/entities/message';
import {
  type AlertServerAction,
  applyDoneKeys,
  type PregnancyDay,
  type PregnancyDayInput,
  type PregnancyToday,
  type PregnancyWeek,
  pregnancyDaySchema,
  pregnancyKeys,
  type WeekState,
  type WeekStateInput,
  weekStateSchema,
} from '@/entities/pregnancy';

import { toPregnancyDayBody } from '../model/v2-body';

/*
 * Pregnancy v2 writes (M7, `/pregnancy/v2/*`). Reads live in
 * `entities/pregnancy`; every mutation here invalidates through
 * `pregnancyKeys` — never hand-written arrays (§8).
 *
 * Privacy (§11): the payloads are pregnancy health data — API only, never
 * logged, never in a URL (the date and week in the path are not health data).
 */

// ── Day log: PUT /pregnancy/v2/days/{date} ─────────────────────

export interface SavePregnancyDayVars {
  /** `YYYY-MM-DD` of the day being logged (not in the future — the API 422s). */
  date: string;
  input: PregnancyDayInput;
}

/**
 * «ذخیرهٔ ثبت امروز». The server writes three stores (v1 symptom log, daily
 * extras, the week's weekly log for weight) and re-runs the alert rules, so a
 * save refreshes: the day itself (adopted from the response), Today (tasks,
 * alert badge), the v2 alert list, report data, the v1 caches that read the
 * same rows, and the daily message (symptom overrides read pregnancy logs in
 * pregnancy mode, T-M7-04). The response's `alerts` are what this save raised
 * — the Log screen shows them inline (T-M7-12).
 */
export function useSavePregnancyDay() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyDay, unknown, SavePregnancyDayVars>({
    mutationFn: async ({ date, input }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/pregnancy/v2/days/${date}`,
        toPregnancyDayBody(input),
      );
      return pregnancyDaySchema.parse(data.data);
    },
    onSuccess: (day, { date }) => {
      queryClient.setQueryData(pregnancyKeys.v2.day(date), day);
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.today() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.reportAll() });
      // Other days show «آخرین ثبت» weight — refetch them when next opened.
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.dayAll(), refetchType: 'none' });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.symptom(date) });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.weeklyLogAll() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
      void queryClient.invalidateQueries({ queryKey: messageKeys.all });
    },
  });
}

// ── Week state: PUT /pregnancy/v2/weeks/{n}/state ──────────────

export interface UpdateWeekStateVars {
  week: number;
  state: WeekStateInput;
}

interface WeekStateContext {
  previousWeek: PregnancyWeek | null | undefined;
  previousToday: PregnancyToday | null | undefined;
}

/**
 * Bookmark toggle and «مراقبت‌های این هفته» ticks (Week + Today share the
 * task list). Optimistic: the tick shows at once on both screens and rolls
 * back if the PUT fails. Send the *whole* `done_task_keys` list — build it
 * with `toggleDoneKey` from `entities/pregnancy`.
 */
export function useUpdateWeekState() {
  const queryClient = useQueryClient();
  return useMutation<WeekState, unknown, UpdateWeekStateVars, WeekStateContext>({
    mutationFn: async ({ week, state }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/pregnancy/v2/weeks/${week}/state`, state);
      return weekStateSchema.parse(data.data ?? { week, ...state });
    },
    onMutate: async ({ week, state }) => {
      const weekKey = pregnancyKeys.v2.week(week);
      const todayKey = pregnancyKeys.v2.today();
      await Promise.all([
        queryClient.cancelQueries({ queryKey: weekKey }),
        queryClient.cancelQueries({ queryKey: todayKey }),
      ]);
      const previousWeek = queryClient.getQueryData<PregnancyWeek | null>(weekKey);
      const previousToday = queryClient.getQueryData<PregnancyToday | null>(todayKey);

      if (previousWeek) {
        queryClient.setQueryData<PregnancyWeek>(weekKey, {
          ...previousWeek,
          bookmarked: state.bookmarked ?? previousWeek.bookmarked,
          tasks: state.done_task_keys ? applyDoneKeys(previousWeek.tasks, state.done_task_keys) : previousWeek.tasks,
        });
      }
      // Today's checklist is the current week's task list.
      if (previousToday && state.done_task_keys && previousToday.progress.week === week) {
        queryClient.setQueryData<PregnancyToday>(todayKey, {
          ...previousToday,
          tasks: applyDoneKeys(previousToday.tasks, state.done_task_keys),
        });
      }
      return { previousWeek, previousToday };
    },
    onError: (_error, { week }, context) => {
      if (context?.previousWeek !== undefined) {
        queryClient.setQueryData(pregnancyKeys.v2.week(week), context.previousWeek);
      }
      if (context?.previousToday !== undefined) {
        queryClient.setQueryData(pregnancyKeys.v2.today(), context.previousToday);
      }
    },
    onSettled: (_data, _error, { week }) => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.week(week) });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.today() });
    },
  });
}

// ── Alerts: POST /pregnancy/v2/alerts/{id}/actions/{action} ────

export interface PregnancyAlertActionVars {
  id: number;
  action: AlertServerAction;
}

/**
 * «دیدم، ممنون» (`ack`) and «افزودن به یادداشت ویزیت» (`add_to_visit_note`,
 * which appends to today's `visit_note` server-side). Deep-link actions such
 * as `log_weight` never reach this hook — the Alerts screen routes them.
 */
export function usePregnancyAlertAction() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, PregnancyAlertActionVars>({
    mutationFn: async ({ id, action }) => {
      await apiClient.post(`/pregnancy/v2/alerts/${id}/actions/${action}`);
    },
    onSuccess: (_data, { action }) => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.today() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
      if (action === 'add_to_visit_note') {
        void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.dayAll() });
        void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.reportAll() });
      }
    },
  });
}

'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** GET/PUT /dashboard/alert-rules (admin-api.md §6b, T-M10-31): the comparator, window and min volume are read-only. */
const ruleSchema = z.object({
  key: z.string(),
  comparator: z.string(),
  threshold: z.number(),
  window_days: z.number(),
  min_volume: z.number().nullable(),
  enabled: z.boolean(),
});
export type AlertRule = z.infer<typeof ruleSchema>;

export const alertRuleKeys = { all: ['dashboard', 'alert-rules'] as const };

export function useAlertRules(enabled: boolean) {
  return useQuery({
    queryKey: alertRuleKeys.all,
    queryFn: ({ signal }) => api.get('/dashboard/alert-rules', { schema: z.array(ruleSchema), signal }),
    enabled,
  });
}

export function useSaveAlertRule() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ key, threshold, enabled }: { key: string; threshold: number; enabled: boolean }) =>
      api.put(`/dashboard/alert-rules/${key}`, { threshold, enabled }),
    onSuccess: () => client.invalidateQueries({ queryKey: alertRuleKeys.all }),
  });
}

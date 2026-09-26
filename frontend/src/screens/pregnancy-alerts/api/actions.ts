'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type AlertServerAction, pregnancyKeys } from '@/entities/pregnancy';
import { apiClient } from '@/shared/api';

interface Vars {
  id: number;
  action: AlertServerAction;
}

/** POST /pregnancy/v2/alerts/{id}/actions/{ack|add_to_visit_note}. */
export function useAlertV2Action() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, Vars>({
    mutationFn: async ({ id, action }) => {
      await apiClient.post(`/pregnancy/v2/alerts/${id}/actions/${action}`);
    },
    onSuccess: () => {
      // ack/visit note change the list, the Today badge and the day log.
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.all() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

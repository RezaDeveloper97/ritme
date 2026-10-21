import { useMutation, useQueryClient } from '@tanstack/react-query';

import { pregnancyKeys, type PregnancyAlertV2 } from '@/entities/pregnancy';
import { apiClient } from '@/shared/api';

/** Rule of the urgent alert a short kick count raises (server `fetal_movement`, T-M9-05). */
export const KICK_ALERT_RULE = 'fetal_movement';

/**
 * The fetal-movement alerts the kick screen shows right away: urgent and not
 * acknowledged yet. The server raises them when a count ends (or runs past its
 * window) short of the target; acknowledging one hides it here and in the
 * alert centre.
 */
export function openKickAlerts(alerts: readonly PregnancyAlertV2[] | null | undefined): PregnancyAlertV2[] {
  return (alerts ?? []).filter((a) => a.ruleKey === KICK_ALERT_RULE && a.level === 'urgent' && !a.isAcked);
}

/** POST /pregnancy/v2/alerts/{id}/actions/ack («دیدم، ممنون»). */
export function useAckKickAlert() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.post(`/pregnancy/v2/alerts/${id}/actions/ack`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.all() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

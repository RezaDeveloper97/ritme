import { apiClient, type ApiEnvelope } from '@/shared/api';
import { isAuthenticated, onSessionEnd } from '@/shared/session';

import { createTracker, type AnalyticsEvent } from './track';

export type { AnalyticsEvent } from './track';

const tracker = createTracker({
  isAuthenticated,
  storage: typeof window === 'undefined' ? null : window.localStorage,
  send: async (events) => {
    const { data } = await apiClient.post<ApiEnvelope<{ accepted: number; tracking: boolean }>>('/analytics/events', {
      events,
    });
    return { tracking: data.data?.tracking ?? false };
  },
});

if (typeof window !== 'undefined') {
  onSessionEnd(() => tracker.reset());
  // Send what is queued when the tab is hidden (best effort).
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') void tracker.flush();
  });
}

/** Records a product analytics event (batched; a no-op without the anonymous-statistics consent). */
export const track = (event: AnalyticsEvent): void => tracker.track(event);
/** Call when the anonymous-statistics consent switch changes. */
export const analyticsConsentChanged = (): void => tracker.consentChanged();
